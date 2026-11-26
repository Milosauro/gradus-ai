package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Canonical normalized action classes.
const (
	ActionCommand = "command"
	ActionWrite   = "write"
	ActionOther   = "other"
)

// NormalizedAction represents a tool invocation normalized across Claude Code and Antigravity.
type NormalizedAction struct {
	ToolName            string
	Action              string
	Target              string
	WorkspaceDir        string
	IsAntigravityNative bool
	IsClaudeCodeNative  bool
}

type antigravityToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type rawHookPayload struct {
	// Claude Code / generic format
	HookEventName string         `json:"hook_event_name"`
	SessionID     string         `json:"session_id"`
	ToolName      string         `json:"tool_name"`
	Tool          string         `json:"tool"`
	ToolInput     map[string]any `json:"tool_input"`
	Input         map[string]any `json:"input"`

	// Antigravity native camelCase protojson format
	ToolCall       *antigravityToolCall `json:"toolCall"`
	Cwd            string               `json:"cwd"`
	WorkspacePaths []string             `json:"workspacePaths"`
}

// ParsePayload parses a PreToolUse JSON payload from Claude Code or Antigravity
// (both legacy tool_name/tool_input and native Antigravity toolCall/args + cwd)
// and normalizes it into a canonical action class and target.
// Empty, malformed, or tool-less payloads fail closed with an error.
func ParsePayload(rootDir string, data []byte) (NormalizedAction, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return NormalizedAction{}, fmt.Errorf("empty guard payload on stdin")
	}

	var raw rawHookPayload
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return NormalizedAction{}, fmt.Errorf("invalid guard JSON payload: %w", err)
	}

	toolName := strings.TrimSpace(raw.ToolName)
	input := raw.ToolInput
	isAntigravityNative := false

	if toolName == "" && raw.ToolCall != nil {
		isAntigravityNative = true
		toolName = strings.TrimSpace(raw.ToolCall.Name)
		if input == nil {
			input = raw.ToolCall.Args
		}
	}
	if toolName == "" {
		toolName = strings.TrimSpace(raw.Tool)
	}
	if toolName == "" {
		return NormalizedAction{}, fmt.Errorf("guard payload missing tool_name")
	}
	if input == nil {
		input = raw.Input
	}

	effectiveRoot := resolveWorkspaceRoot(rootDir, raw.Cwd, raw.WorkspacePaths)
	isClaudeCodeNative := !isAntigravityNative && (strings.TrimSpace(raw.HookEventName) != "" || strings.TrimSpace(raw.SessionID) != "")

	switch toolName {
	case "Bash", "run_command":
		cmd := firstString(input, "command", "CommandLine", "cmd")
		return NormalizedAction{
			ToolName:            toolName,
			Action:              ActionCommand,
			Target:              cmd,
			WorkspaceDir:        effectiveRoot,
			IsAntigravityNative: isAntigravityNative,
			IsClaudeCodeNative:  isClaudeCodeNative,
		}, nil

	case "Edit", "Write", "MultiEdit",
		"write_to_file", "replace_file_content", "multi_replace_file_content",
		"write_file", "edit_file":
		p := firstString(input, "file_path", "TargetFile", "Path", "path")
		return NormalizedAction{
			ToolName:            toolName,
			Action:              ActionWrite,
			Target:              NormalizeTargetPath(effectiveRoot, p),
			WorkspaceDir:        effectiveRoot,
			IsAntigravityNative: isAntigravityNative,
			IsClaudeCodeNative:  isClaudeCodeNative,
		}, nil

	default:
		return NormalizedAction{
			ToolName:            toolName,
			Action:              ActionOther,
			Target:              "",
			WorkspaceDir:        effectiveRoot,
			IsAntigravityNative: isAntigravityNative,
			IsClaudeCodeNative:  isClaudeCodeNative,
		}, nil
	}
}

func resolveWorkspaceRoot(fallback, cwd string, workspacePaths []string) string {
	candidates := make([]string, 0, 2+len(workspacePaths))
	if strings.TrimSpace(cwd) != "" {
		candidates = append(candidates, strings.TrimSpace(cwd))
	}
	for _, wp := range workspacePaths {
		if strings.TrimSpace(wp) != "" {
			candidates = append(candidates, strings.TrimSpace(wp))
		}
	}
	if strings.TrimSpace(fallback) != "" {
		candidates = append(candidates, strings.TrimSpace(fallback))
	}

	// Prefer the first candidate (or its immediate parent when no .git boundary separates them,
	// e.g. when hook CWD is .agents/) that contains gradus.yaml.
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "gradus.yaml")); err == nil {
			return c
		}
		if _, err := os.Stat(filepath.Join(c, ".git")); err == nil {
			continue
		}
		parent := filepath.Dir(c)
		if parent != c {
			if _, err := os.Stat(filepath.Join(parent, "gradus.yaml")); err == nil {
				return parent
			}
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return fallback
}

// NormalizeTargetPath converts backslashes to forward slashes, strips a matching rootDir prefix,
// and cleans relative prefixes (such as "./") for consistent cross-platform matching.
func NormalizeTargetPath(rootDir, rawPath string) string {
	p := strings.TrimSpace(rawPath)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", "/")

	if rootDir != "" {
		absRoot, err := filepath.Abs(rootDir)
		if err == nil {
			rootSlash := strings.TrimSuffix(filepath.ToSlash(absRoot), "/")
			if p == rootSlash {
				return "."
			}
			if strings.HasPrefix(p, rootSlash+"/") {
				p = strings.TrimPrefix(p, rootSlash+"/")
			} else if strings.EqualFold(p, rootSlash) {
				return "."
			} else if len(p) > len(rootSlash)+1 && strings.EqualFold(p[:len(rootSlash)+1], rootSlash+"/") {
				p = p[len(rootSlash)+1:]
			}
		}
		rawRootSlash := strings.TrimSuffix(strings.ReplaceAll(rootDir, "\\", "/"), "/")
		if rawRootSlash != "" && strings.HasPrefix(p, rawRootSlash+"/") {
			p = strings.TrimPrefix(p, rawRootSlash+"/")
		}
	}

	p = path.Clean(p)
	p = strings.TrimPrefix(p, "./")
	if p == "." {
		return "."
	}
	return p
}

func firstString(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

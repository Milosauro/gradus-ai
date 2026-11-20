package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	ToolName string
	Action   string
	Target   string
}

type rawHookPayload struct {
	ToolName  string         `json:"tool_name"`
	Tool      string         `json:"tool"`
	ToolInput map[string]any `json:"tool_input"`
	Input     map[string]any `json:"input"`
}

// ParsePayload parses a PreToolUse JSON payload from Claude Code or Antigravity
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
	if toolName == "" {
		toolName = strings.TrimSpace(raw.Tool)
	}
	if toolName == "" {
		return NormalizedAction{}, fmt.Errorf("guard payload missing tool_name")
	}

	input := raw.ToolInput
	if input == nil {
		input = raw.Input
	}

	switch toolName {
	case "Bash", "run_command":
		cmd := firstString(input, "command", "CommandLine", "cmd")
		return NormalizedAction{
			ToolName: toolName,
			Action:   ActionCommand,
			Target:   cmd,
		}, nil

	case "Edit", "Write", "MultiEdit",
		"write_to_file", "replace_file_content", "multi_replace_file_content",
		"write_file", "edit_file":
		p := firstString(input, "file_path", "TargetFile", "Path", "path")
		return NormalizedAction{
			ToolName: toolName,
			Action:   ActionWrite,
			Target:   NormalizeTargetPath(rootDir, p),
		}, nil

	default:
		return NormalizedAction{
			ToolName: toolName,
			Action:   ActionOther,
			Target:   "",
		}, nil
	}
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

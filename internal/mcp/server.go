package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/toolchain"
	"github.com/Milosauro/gradus-ai/internal/version"
)

const (
	protocolVersion  = "2024-11-05"
	maxLineBytes     = 4 * 1024 * 1024
	errCodeParse     = -32700
	errCodeInvalid   = -32600
	errCodeNotFound  = -32601
	errCodeBadParams = -32602
)

// Server implements a zero-dependency MCP JSON-RPC 2.0 stdio server over a Gradus pyramid (§20).
type Server struct {
	RootDir string
	Offline bool
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolAnnotations struct {
	ReadOnlyHint    bool `json:"readOnlyHint"`
	DestructiveHint bool `json:"destructiveHint"`
	IdempotentHint  bool `json:"idempotentHint"`
}

type toolDef struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema map[string]any   `json:"inputSchema"`
	Annotations *toolAnnotations `json:"annotations,omitempty"`
}

var errLineTooLong = errors.New("request exceeds maximum line size")

func readBoundedLine(r *bufio.Reader, maxBytes int) (string, error) {
	var buf bytes.Buffer
	for {
		frag, isPrefix, err := r.ReadLine()
		if len(frag) > 0 {
			if buf.Len()+len(frag) > maxBytes {
				// Drain remainder of oversized line so the stream stays synchronized.
				for isPrefix && err == nil {
					_, isPrefix, err = r.ReadLine()
				}
				return "", errLineTooLong
			}
			buf.Write(frag)
		}
		if err != nil {
			if errors.Is(err, io.EOF) && buf.Len() > 0 {
				return buf.String(), nil
			}
			return "", err
		}
		if !isPrefix {
			return buf.String(), nil
		}
	}
}

// ServeStdio reads newline-delimited JSON-RPC 2.0 messages from in and writes responses to out until EOF.
func (s *Server) ServeStdio(in io.Reader, out io.Writer) error {
	reader := bufio.NewReaderSize(in, 64*1024)
	enc := json.NewEncoder(out)

	for {
		rawLine, err := readBoundedLine(reader, maxLineBytes)
		if errors.Is(err, errLineTooLong) {
			_ = enc.Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &rpcError{Code: errCodeInvalid, Message: fmt.Sprintf("request exceeds maximum size of %d bytes", maxLineBytes)},
			})
			continue
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = enc.Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &rpcError{Code: errCodeParse, Message: fmt.Sprintf("parse error: %v", err)},
			})
			continue
		}
		if len(req.ID) == 0 || string(req.ID) == "null" {
			// JSON-RPC notifications (e.g. notifications/initialized) do not receive responses.
			continue
		}
		if req.JSONRPC != "" && req.JSONRPC != "2.0" {
			_ = enc.Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: errCodeInvalid, Message: "unsupported jsonrpc version"},
			})
			continue
		}
		resp := s.handleRequest(req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		pv := protocolVersion
		if len(req.Params) > 0 {
			var initParams struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			if err := json.Unmarshal(req.Params, &initParams); err == nil && strings.TrimSpace(initParams.ProtocolVersion) != "" {
				pv = strings.TrimSpace(initParams.ProtocolVersion)
			}
		}
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": pv,
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "gradusctl",
					"version": version.Get().Version,
				},
			},
		}
	case "ping":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": toolDefinitions()},
		}
	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil || strings.TrimSpace(params.Name) == "" {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: errCodeBadParams, Message: "invalid tools/call params"},
			}
		}
		res := s.callTool(params.Name, params.Arguments)
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: res}
	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: errCodeNotFound, Message: fmt.Sprintf("method not found: %s", req.Method)},
		}
	}
}

func (s *Server) callTool(name string, args map[string]any) toolResult {
	switch name {
	case "resolve_rules":
		return s.toolResolveRules(args)
	case "explain_rule":
		return s.toolExplainRule(args)
	case "check_text":
		return s.toolCheckText(args)
	case "propose_rule":
		return s.toolProposeRule(args)
	case "get_toolchain":
		return s.toolGetToolchain(args)
	default:
		return errorToolResult(fmt.Sprintf("unknown tool %q", name))
	}
}

func (s *Server) toolResolveRules(args map[string]any) toolResult {
	res, err := resolve.Resolve(s.RootDir, resolve.Options{OfflineOnly: s.Offline})
	if err != nil {
		return errorToolResult(err.Error())
	}
	levelFilter := strings.TrimSpace(stringArg(args, "level"))
	enfFilter := strings.TrimSpace(stringArg(args, "enforcement"))

	var filtered []resolve.ResolvedRule
	for _, r := range res.View.Rules {
		if levelFilter != "" && !strings.EqualFold(r.Level, levelFilter) {
			continue
		}
		if enfFilter != "" && !strings.EqualFold(r.Enforcement, enfFilter) {
			continue
		}
		filtered = append(filtered, r)
	}
	if filtered == nil {
		filtered = []resolve.ResolvedRule{}
	}

	return jsonToolResult(map[string]any{
		"view_hash": res.View.ViewHash,
		"levels":    res.View.Levels,
		"rules":     filtered,
		"conflicts": res.View.Conflicts,
	})
}

func (s *Server) toolExplainRule(args map[string]any) toolResult {
	id := strings.TrimSpace(stringArg(args, "id"))
	if id == "" {
		return errorToolResult("argument 'id' is required")
	}
	res, err := resolve.Resolve(s.RootDir, resolve.Options{OfflineOnly: s.Offline})
	if err != nil {
		return errorToolResult(err.Error())
	}
	for _, r := range res.View.Rules {
		if r.ID != id {
			continue
		}
		var charter *schema.Charter
		for _, lvl := range res.View.Levels {
			if lvl.ID == r.Level {
				charter = lvl.Charter
				break
			}
		}
		return jsonToolResult(map[string]any{
			"rule":    r,
			"charter": charter,
		})
	}
	return errorToolResult(fmt.Sprintf("rule %q not found in resolved view", id))
}

func (s *Server) toolCheckText(args map[string]any) toolResult {
	content := stringArg(args, "content")
	if content == "" {
		return errorToolResult("argument 'content' is required")
	}
	relPath := filepath.ToSlash(strings.TrimSpace(stringArg(args, "path")))
	if relPath == "" {
		relPath = "snippet.txt"
	}

	res, err := resolve.Resolve(s.RootDir, resolve.Options{OfflineOnly: s.Offline})
	if err != nil {
		return errorToolResult(err.Error())
	}

	findings := []evidence.Finding{}
	var notes []string
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	for _, rule := range res.View.Rules {
		if rule.Waived || rule.Enforcement != "check" || rule.Checker == nil {
			continue
		}
		if !scopeMatchesPath(rule.Scope, relPath) {
			continue
		}
		cname := strings.ToLower(strings.TrimSpace(rule.Checker.Name))
		if cname != "pattern" && cname != "regex" {
			notes = append(notes, fmt.Sprintf("rule %s uses external checker %q and was skipped for in-memory check_text", rule.ID, rule.Checker.Name))
			continue
		}
		rawPattern := rule.Checker.Pattern
		if rawPattern == "" {
			rawPattern = rule.Checker.Config
		}
		re, compErr := regexp.Compile(rawPattern)
		if compErr != nil {
			return errorToolResult(fmt.Sprintf("rule %s invalid regex %q: %v", rule.ID, rawPattern, compErr))
		}
		for idx, line := range lines {
			if re.MatchString(line) {
				findings = append(findings, evidence.Finding{
					RuleID:     rule.ID,
					Severity:   rule.Severity,
					Level:      rule.Level,
					Source:     rule.Provenance.Source,
					Path:       relPath,
					Line:       idx + 1,
					Message:    fmt.Sprintf("%s (matched pattern %q)", rule.Title, rawPattern),
					Confidence: 1.0,
					Status:     "open",
				})
			}
		}
	}

	verdict := "pass"
	if len(findings) > 0 {
		verdict = "fail"
	}
	return jsonToolResult(map[string]any{
		"verdict":  verdict,
		"path":     relPath,
		"findings": findings,
		"notes":    notes,
	})
}

func (s *Server) toolProposeRule(args map[string]any) toolResult {
	id := strings.TrimSpace(stringArg(args, "id"))
	title := strings.TrimSpace(stringArg(args, "title"))
	level := strings.TrimSpace(stringArg(args, "level"))
	severity := strings.TrimSpace(stringArg(args, "severity"))
	enforcement := strings.TrimSpace(stringArg(args, "enforcement"))
	statement := strings.TrimSpace(stringArg(args, "statement"))
	if enforcement == "" {
		enforcement = "prompt"
	}
	if id == "" || title == "" || level == "" || severity == "" || statement == "" {
		return errorToolResult("arguments 'id', 'title', 'level', 'severity', and 'statement' are required")
	}
	if strings.ContainsAny(level, `/\`) || strings.Contains(level, "..") {
		return errorToolResult(fmt.Sprintf("invalid level %q: path traversal is not allowed", level))
	}

	res, err := resolve.Resolve(s.RootDir, resolve.Options{OfflineOnly: s.Offline})
	if err != nil {
		return errorToolResult(err.Error())
	}

	levelExists := false
	var charterWarnings []string
	for _, lvl := range res.View.Levels {
		if lvl.ID == level {
			levelExists = true
			if lvl.Charter != nil && len(lvl.Charter.AllowedSeverities) > 0 {
				if !slices.Contains(lvl.Charter.AllowedSeverities, severity) {
					charterWarnings = append(charterWarnings, fmt.Sprintf("severity %s is not in level %q charter allowed_severities (%v)", severity, level, lvl.Charter.AllowedSeverities))
				}
			}
			break
		}
	}
	if !levelExists {
		return errorToolResult(fmt.Sprintf("level %q does not exist in the resolved pyramid", level))
	}

	md := fmt.Sprintf("---\nid: %s\ntitle: %q\nseverity: %s\nenforcement: %s\n---\n\n%s\n",
		id, title, severity, enforcement, statement)

	proposedPath := fmt.Sprintf(".gradus/levels/%s/rules/%s.md", level, id)
	if _, parseErr := schema.ParseRuleBytes([]byte(md), proposedPath, level, level); parseErr != nil {
		return errorToolResult(fmt.Sprintf("invalid rule proposal: %v", parseErr))
	}
	lockedTarget := false
	if cfgBytes, readErr := os.ReadFile(filepath.Join(s.RootDir, "gradus.yaml")); readErr == nil {
		if cfgRes, parseErr := schema.ParsePyramidConfig(cfgBytes); parseErr == nil {
			for _, lvl := range cfgRes.Value.Levels {
				if lvl.ID == level && lvl.Locked {
					lockedTarget = true
					break
				}
			}
		}
	}

	return jsonToolResult(map[string]any{
		"valid":            len(charterWarnings) == 0,
		"locked_target":    lockedTarget,
		"charter_warnings": charterWarnings,
		"proposed_path":    proposedPath,
		"markdown":         md,
	})
}

func (s *Server) toolGetToolchain(args map[string]any) toolResult {
	res, err := resolve.Resolve(s.RootDir, resolve.Options{OfflineOnly: s.Offline})
	if err != nil {
		return errorToolResult(err.Error())
	}
	out := map[string]any{
		"view_hash": res.View.ViewHash,
		"toolchain": res.View.Toolchain,
	}
	if inc, ok := args["include_check"].(bool); ok && inc {
		out["check"] = toolchain.Check(s.RootDir, res.View.Toolchain)
	}
	return jsonToolResult(out)
}

func scopeMatchesPath(scopes []string, path string) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, sc := range scopes {
		if sc == "**/*" || sc == "**" {
			return true
		}
		if ok, err := doublestar.Match(sc, path); err == nil && ok {
			return true
		}
	}
	return false
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, _ := args[key].(string)
	return v
}

func jsonToolResult(v any) toolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return toolResult{
		Content: []toolContent{{Type: "text", Text: string(b)}},
	}
}

func errorToolResult(msg string) toolResult {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return toolResult{
		Content: []toolContent{{Type: "text", Text: string(b)}},
		IsError: true,
	}
}

func toolDefinitions() []toolDef {
	ro := &toolAnnotations{
		ReadOnlyHint:    true,
		DestructiveHint: false,
		IdempotentHint:  true,
	}
	return []toolDef{
		{
			Name:        "resolve_rules",
			Description: "Resolve the Gradus pyramid offline and return active rules, levels, and conflicts.",
			Annotations: ro,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"level":       map[string]any{"type": "string", "description": "Optional level filter (e.g. org, team, project)"},
					"enforcement": map[string]any{"type": "string", "description": "Optional enforcement filter (prompt, review, check, hook, permission)"},
				},
			},
		},
		{
			Name:        "explain_rule",
			Description: "Explain a resolved rule by ID including provenance, rationalizations, and owning level charter.",
			Annotations: ro,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "Rule ID to explain"},
				},
			},
		},
		{
			Name:        "check_text",
			Description: "Evaluate an in-memory text snippet against active built-in regex/pattern check rules without modifying disk.",
			Annotations: ro,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"content"},
				"properties": map[string]any{
					"content": map[string]any{"type": "string", "description": "Text content to check"},
					"path":    map[string]any{"type": "string", "description": "Optional virtual relative path for scope matching"},
				},
			},
		},
		{
			Name:        "propose_rule",
			Description: "Draft and validate a gradus/v1 rule Markdown proposal against schema and level charter without writing files.",
			Annotations: ro,
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"id", "title", "level", "severity", "statement"},
				"properties": map[string]any{
					"id":          map[string]any{"type": "string"},
					"title":       map[string]any{"type": "string"},
					"level":       map[string]any{"type": "string"},
					"severity":    map[string]any{"type": "string", "enum": []string{"MUST", "SHOULD", "MAY"}},
					"enforcement": map[string]any{"type": "string", "enum": []string{"prompt", "review", "check", "hook", "permission"}},
					"statement":   map[string]any{"type": "string"},
				},
			},
		},
		{
			Name:        "get_toolchain",
			Description: "Return the resolved monotonic toolchain (required, recommended, forbidden) and optional local check results.",
			Annotations: ro,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"include_check": map[string]any{"type": "boolean", "description": "If true, also run offline toolchain checks"},
				},
			},
		},
	}
}

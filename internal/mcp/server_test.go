package mcp_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/mcp"
)

func TestMCPServerProtocolAndTools(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "team", "rules"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)

	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "team", "level.yaml"), []byte(`schema: gradus/v1
id: team
charter:
  purpose: Team conventions
  allowed_severities: [SHOULD, MAY]
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "team", "rules", "TEAM-01.md"), []byte(`---
id: TEAM-01
title: Avoid hardcoded tokens
severity: SHOULD
enforcement: check
scope: ["**/*.go"]
checker:
  name: regex
  pattern: 'SECRET_TOKEN'
---

Do not hardcode SECRET_TOKEN in Go files.
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
  - id: project
    source: { path: .gradus/levels/project }
toolchain:
  required:
    - { type: tool, id: git }
  forbidden:
    - { type: mcp, id: "*-pastebin" }
`), 0o644)

	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"resolve_rules","arguments":{"level":"team"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"explain_rule","arguments":{"id":"TEAM-01"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"check_text","arguments":{"path":"main.go","content":"const tok = \"SECRET_TOKEN\""}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"check_text","arguments":{"path":"main.go","content":"const ok = 1"}}}`,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"propose_rule","arguments":{"id":"TEAM-02","title":"Charter violation test","level":"team","severity":"MUST","statement":"Must fail charter."}}}`,
		`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"get_toolchain","arguments":{"include_check":true}}}`,
		`{"jsonrpc":"2.0","id":10,"method":"unknown/method"}`,
		`{not-valid-json`,
	}, "\n") + "\n"

	var out bytes.Buffer
	srv := &mcp.Server{RootDir: tmp, Offline: true}
	if err := srv.ServeStdio(strings.NewReader(requests), &out); err != nil {
		t.Fatalf("ServeStdio failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// 11 responses (notifications/initialized produces no response)
	if len(lines) != 11 {
		t.Fatalf("expected 11 JSON-RPC responses, got %d:\n%s", len(lines), out.String())
	}

	var respList struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[2]), &respList); err != nil {
		t.Fatal(err)
	}
	if len(respList.Result.Tools) != 5 {
		t.Fatalf("expected 5 MCP tools, got %+v", respList.Result.Tools)
	}

	// Verify check_text fail (id=6 -> lines[5]) and pass (id=7 -> lines[6])
	if !strings.Contains(lines[5], `\"verdict\": \"fail\"`) || !strings.Contains(lines[5], "TEAM-01") {
		t.Fatalf("expected check_text fail with TEAM-01, got: %s", lines[5])
	}
	if !strings.Contains(lines[6], `\"verdict\": \"pass\"`) {
		t.Fatalf("expected check_text pass on clean snippet, got: %s", lines[6])
	}

	// Verify propose_rule charter warning and non-mutation (id=8 -> lines[7])
	if !strings.Contains(lines[7], "allowed_severities") {
		t.Fatalf("expected charter_warnings for MUST on team level, got: %s", lines[7])
	}
	if _, err := os.Stat(filepath.Join(tmp, ".gradus", "levels", "team", "rules", "TEAM-02.md")); !os.IsNotExist(err) {
		t.Fatalf("propose_rule must never write files to disk")
	}

	// Verify get_toolchain (id=9 -> lines[8])
	if !strings.Contains(lines[8], "*-pastebin") {
		t.Fatalf("expected forbidden pastebin in get_toolchain output, got: %s", lines[8])
	}

	// Verify unknown method (-32601 -> lines[9]) and parse error (-32700 -> lines[10])
	if !strings.Contains(lines[9], "-32601") || !strings.Contains(lines[10], "-32700") {
		t.Fatalf("expected -32601 and -32700 errors, got:\n%s\n%s", lines[9], lines[10])
	}
}

func TestMCPHardeningV041(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644)

	oversized := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"check_text","arguments":{"content":"` + strings.Repeat("A", 5*1024*1024) + `"}}}`
	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`,
		oversized,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"propose_rule","arguments":{"id":"PRJ-01","title":"Traversal","level":"../../etc","severity":"MUST","statement":"Bad"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"propose_rule","arguments":{"id":"PRJ-01","title":"Missing","level":"nonexistent","severity":"MUST","statement":"Bad"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"ping"}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	srv := &mcp.Server{RootDir: tmp, Offline: true}
	if err := srv.ServeStdio(strings.NewReader(requests), &out); err != nil {
		t.Fatalf("ServeStdio failed after oversized request: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 responses (server must stay alive after oversized request), got %d:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], `"protocolVersion":"2025-03-26"`) {
		t.Fatalf("expected negotiated protocolVersion 2025-03-26, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "-32600") {
		t.Fatalf("expected -32600 error for oversized 5MB line, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], `"readOnlyHint":true`) {
		t.Fatalf("expected readOnlyHint annotation in tools/list, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], `"isError":true`) || !strings.Contains(lines[4], `"isError":true`) {
		t.Fatalf("expected isError:true for invalid/traversal propose_rule levels, got:\n%s\n%s", lines[3], lines[4])
	}
	if !strings.Contains(lines[5], `"id":6`) {
		t.Fatalf("expected server to answer ping after oversized/error requests, got: %s", lines[5])
	}
}

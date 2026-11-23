package toolchain_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/toolchain"
)

func TestToolchainCheck(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, ".claude", "skills", "gradus-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: gradus-review\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tc := &resolve.ResolvedToolchain{
		Required: []resolve.ResolvedToolchainItem{
			{Type: "skill", ID: "gradus-review", Level: "team"},
			{Type: "tool", ID: "go", Level: "team"},
		},
		Recommended: []resolve.ResolvedToolchainItem{
			{Type: "tool", ID: "definitely-missing-recommended-tool-xyz", Level: "project"},
		},
		Forbidden: []resolve.ResolvedToolchainItem{
			{Type: "mcp", ID: "*-public-pastebin", Level: "team"},
		},
	}

	// 1. Clean pass with non-blocking warning for missing recommended tool
	rep := toolchain.Check(tmp, tc)
	if rep.Verdict != "pass" || len(rep.Findings) != 0 || len(rep.Warnings) != 1 {
		t.Fatalf("expected pass with 1 warning, got: %+v", rep)
	}

	// 2. Forbidden MCP configured in .mcp.json -> fail
	if err := os.WriteFile(filepath.Join(tmp, ".mcp.json"), []byte(`{"mcpServers":{"acme-public-pastebin":{"command":"node"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep2 := toolchain.Check(tmp, tc)
	if rep2.Verdict != "fail" || len(rep2.Findings) != 1 || rep2.Findings[0].Kind != "forbidden_present" {
		t.Fatalf("expected forbidden_present failure, got: %+v", rep2)
	}

	// 3. Missing required skill -> fail
	_ = os.Remove(filepath.Join(skillDir, "SKILL.md"))
	rep3 := toolchain.Check(tmp, tc)
	if rep3.Verdict != "fail" || len(rep3.Findings) < 2 || !strings.Contains(rep3.Findings[0].Kind, "missing_required") {
		t.Fatalf("expected missing_required failure, got: %+v", rep3)
	}
}

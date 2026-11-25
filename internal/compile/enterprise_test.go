package compile_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestManagedSettingsGeneration(t *testing.T) {
	view := resolve.ResolvedView{
		Schema:   "gradus/v1",
		ViewHash: "sha256:ms1",
		Rules: []resolve.ResolvedRule{
			{
				ID:          "ORG-PERM-01",
				Level:       "org",
				Enforcement: "permission",
				Guard:       &schema.GuardConfig{Action: "command", Pattern: "rm -rf *", Decision: "deny"},
			},
			{
				ID:          "PRJ-PERM-02",
				Level:       "project",
				Enforcement: "permission",
				Guard:       &schema.GuardConfig{Action: "command", Pattern: "git push *", Decision: "ask"},
			},
			{
				ID:          "USR-PERM-03",
				Level:       "user",
				Enforcement: "permission",
				Guard:       &schema.GuardConfig{Action: "command", Pattern: "echo *", Decision: "deny"},
			},
		},
		Toolchain: &resolve.ResolvedToolchain{
			Forbidden: []resolve.ResolvedToolchainItem{
				{Type: "mcp", ID: "acme-public-pastebin", Level: "org"},
				{Type: "skill", ID: "bad-skill", Level: "org"},
			},
		},
	}

	raw, err := compile.RenderClaudeManagedSettings(view, "deny")
	if err != nil {
		t.Fatalf("RenderClaudeManagedSettings failed: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["disableBypassPermissionsMode"] != "disable" {
		t.Fatalf("expected disableBypassPermissionsMode=disable, got %v", parsed["disableBypassPermissionsMode"])
	}
	text := string(raw)
	for _, want := range []string{"gradusctl guard", "gradusctl status --brief --hook", "Bash(rm -rf *)", "Bash(git push *)", "acme-public-pastebin"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in managed-settings.json:\n%s", want, text)
		}
	}
	if strings.Contains(text, "echo *") || strings.Contains(text, "bad-skill") {
		t.Fatalf("did not expect user-level permission or non-mcp toolchain item in managed-settings.json:\n%s", text)
	}
}

func TestChatBundleZIPReproducibility(t *testing.T) {
	tmp := t.TempDir()
	memFS := fstest.MapFS{
		"gradus-review/SKILL.md": &fstest.MapFile{Data: []byte("---\nname: gradus-review\ndescription: Review\n---\n\nBody.\n")},
		"gradus-author/SKILL.md": &fstest.MapFile{Data: []byte("---\nname: gradus-author\ndescription: Author\n---\n\nBody.\n")},
	}
	view := resolve.ResolvedView{
		Schema:   "gradus/v1",
		ViewHash: "sha256:cb1",
		Levels: []resolve.LevelSummary{
			{ID: "org", Hash: "sha256:o"},
			{ID: "user", Hash: "sha256:u"},
		},
		Rules: []resolve.ResolvedRule{
			{ID: "ORG-01", Title: "Org rule", Level: "org", Severity: "MUST", Enforcement: "prompt", Scope: []string{"**/*"}, Text: "Shared org rule.", Provenance: resolve.RuleProvenance{Source: "org"}},
			{ID: "USR-01", Title: "User rule", Level: "user", Severity: "MAY", Enforcement: "prompt", Scope: []string{"**/*"}, Text: "Personal user secret.", Provenance: resolve.RuleProvenance{Source: "user"}},
		},
	}

	zip1 := filepath.Join(tmp, "bundle1.zip")
	zip2 := filepath.Join(tmp, "bundle2.zip")

	res1, err := compile.WriteChatBundleZIP(zip1, view, memFS)
	if err != nil {
		t.Fatalf("WriteChatBundleZIP 1 failed: %v", err)
	}
	if _, err := compile.WriteChatBundleZIP(zip2, view, memFS); err != nil {
		t.Fatalf("WriteChatBundleZIP 2 failed: %v", err)
	}
	if res1.RuleCount != 1 || len(res1.Skills) != 2 {
		t.Fatalf("unexpected bundle result: %+v", res1)
	}

	b1, _ := os.ReadFile(zip1)
	b2, _ := os.ReadFile(zip2)
	if sha256.Sum256(b1) != sha256.Sum256(b2) {
		t.Fatalf("expected byte-reproducible ZIP bundles")
	}

	zr, err := zip.NewReader(bytes.NewReader(b1), int64(len(b1)))
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()
		contents[f.Name] = string(data)
	}
	if !strings.Contains(contents["RULES.md"], "ORG-01") || strings.Contains(contents["RULES.md"], "USR-01") {
		t.Fatalf("expected RULES.md to include ORG-01 and exclude USR-01, got:\n%s", contents["RULES.md"])
	}
	if contents["manifest.json"] == "" || contents["skills/gradus-review/SKILL.md"] == "" {
		t.Fatalf("missing manifest.json or skills in ZIP: %v", contents)
	}
}

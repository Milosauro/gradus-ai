package adapter_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/adapter"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" && i+1 < len(args) {
			switch args[i+1] {
			case "version-ok":
				fmt.Println("tool v1.5.2")
				os.Exit(0)
			case "evidence-ok":
				fmt.Println(`{"ok":true}`)
				os.Exit(0)
			case "evidence-fail":
				fmt.Fprintln(os.Stderr, "release check failed")
				os.Exit(1)
			}
		}
	}
	os.Exit(2)
}

func TestAdapterSyncAndVerify(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	tmp := t.TempDir()
	constDir := filepath.Join(tmp, ".walden")
	if err := os.MkdirAll(constDir, 0o755); err != nil {
		t.Fatal(err)
	}
	humanHeader := "# Project Constitution\n\nCustom team preamble.\n"
	constPath := filepath.Join(constDir, "constitution.md")
	if err := os.WriteFile(constPath, []byte(humanHeader), 0o644); err != nil {
		t.Fatal(err)
	}

	view := resolve.ResolvedView{
		ViewHash: "sha256:adapterview1",
		Levels:   []resolve.LevelSummary{{ID: "project"}},
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-01",
				Title:       "Require tests",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "review",
				Scope:       []string{"**/*"},
				Text:        "All changes must include tests.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
		},
	}

	m := adapter.Manifest{
		Schema:          "gradus/v1",
		ID:              "walden",
		TargetFile:      ".walden/constitution.md",
		VersionCommand:  []string{exe, "-test.run=^TestHelperProcess$", "--", "version-ok"},
		EvidenceCommand: []string{exe, "-test.run=^TestHelperProcess$", "--", "evidence-ok"},
	}

	// 1. First sync injects managed block while preserving human prose
	syncRes, err := adapter.Sync(tmp, m, view)
	if err != nil || !syncRes.Changed {
		t.Fatalf("expected first sync changed=true, got res=%+v err=%v", syncRes, err)
	}
	raw, err := os.ReadFile(constPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Custom team preamble.") || !strings.Contains(string(raw), "PRJ-01") {
		t.Fatalf("unexpected constitution content:\n%s", string(raw))
	}

	// 2. Second sync is idempotent (NFR3)
	syncRes2, err := adapter.Sync(tmp, m, view)
	if err != nil || syncRes2.Changed {
		t.Fatalf("expected second sync changed=false, got res=%+v err=%v", syncRes2, err)
	}

	// 3. Verify passes when clean, CLI >= 1.4.0 satisfied, and evidence passes
	verRes := adapter.Verify(tmp, m, view, map[string]string{"cli": ">=1.4.0"}, false)
	if verRes.Status != "pass" || verRes.CLIVersion != "1.5.2" {
		t.Fatalf("expected pass with version 1.5.2, got: %+v", verRes)
	}

	// 4. Unsatisfied CLI version fails
	verBadVer := adapter.Verify(tmp, m, view, map[string]string{"cli": ">=2.0.0"}, true)
	if verBadVer.Status != "fail" || len(verBadVer.Issues) == 0 {
		t.Fatalf("expected failure on unsatisfied CLI version, got: %+v", verBadVer)
	}

	// 5. Failing evidence command fails unless skipEvidence=true
	mFailEv := m
	mFailEv.EvidenceCommand = []string{exe, "-test.run=^TestHelperProcess$", "--", "evidence-fail"}
	verFailEv := adapter.Verify(tmp, mFailEv, view, nil, false)
	if verFailEv.Status != "fail" {
		t.Fatalf("expected evidence failure, got: %+v", verFailEv)
	}
	if verSkip := adapter.Verify(tmp, mFailEv, view, nil, true); verSkip.Status != "pass" {
		t.Fatalf("expected pass with skipEvidence=true, got: %+v", verSkip)
	}

	// 6. Managed-section drift detected
	if err := os.WriteFile(constPath, []byte(strings.ReplaceAll(string(raw), "Require tests", "Tampered")), 0o644); err != nil {
		t.Fatal(err)
	}
	verDrift := adapter.Verify(tmp, m, view, nil, true)
	if verDrift.Status != "fail" || verDrift.Drift != "modified" {
		t.Fatalf("expected drift=modified failure, got: %+v", verDrift)
	}
}

package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestHandbookCommand(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, ".gradus", "levels", "project")
	projRules := filepath.Join(projDir, "rules")
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
targets: [agents-md]
`), 0o644)
	_ = os.WriteFile(filepath.Join(projDir, "level.yaml"), []byte(`schema: gradus/v1
id: project
version: "0.2.0"
charter:
  purpose: Repository engineering conventions
  includes: ["testing", "style"]
  excludes: ["company compliance"]
  allowed_severities: [MUST, SHOULD, MAY]
`), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-01.md"), []byte(`---
id: PRJ-01
title: Use canonical English
severity: MUST
enforcement: prompt
scope: ["**/*"]
---
All repository files MUST be written in canonical English.
`), 0o644)

	var stdout1, stdout2, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"handbook", "--dir", tmp}, &stdout1, &stderr); code != cli.ExitPass {
		t.Fatalf("handbook failed (%d): %s\n%s", code, stdout1.String(), stderr.String())
	}
	if code := cli.ExecuteCode([]string{"handbook", "--dir", tmp}, &stdout2, &stderr); code != cli.ExitPass {
		t.Fatalf("second handbook run failed: %d", code)
	}
	if stdout1.String() != stdout2.String() {
		t.Fatalf("expected byte-identical deterministic handbook output")
	}

	md := stdout1.String()
	for _, needle := range []string{"project", "Repository engineering conventions", "PRJ-01", "Use canonical English", "gradusctl exception"} {
		if !strings.Contains(md, needle) {
			t.Fatalf("handbook missing %q:\n%s", needle, md)
		}
	}

	outPath := filepath.Join(tmp, "docs", "HANDBOOK.md")
	stdout1.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"handbook", "--dir", tmp, "--output", outPath, "--json"}, &stdout1, &stderr); code != cli.ExitPass {
		t.Fatalf("handbook --output --json failed (%d): %s\n%s", code, stdout1.String(), stderr.String())
	}
	var rep struct {
		ViewHash   string `json:"view_hash"`
		Markdown   string `json:"markdown"`
		OutputPath string `json:"output_path"`
		RulesCount int    `json:"rules_count"`
	}
	if err := json.Unmarshal(stdout1.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.ViewHash == "" || rep.Markdown == "" || rep.RulesCount != 1 || rep.OutputPath != outPath {
		t.Fatalf("unexpected handbook json output: %+v", rep)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected handbook output file %s: %v", outPath, err)
	}
}

func TestExampleTemplates(t *testing.T) {
	for _, name := range []string{"personal", "team", "regulated-fintech"} {
		t.Run(name, func(t *testing.T) {
			exDir := filepath.Join("..", "..", "examples", name)
			readmeBytes, err := os.ReadFile(filepath.Join(exDir, "README.md"))
			if err != nil {
				t.Fatalf("missing README.md in %s: %v", exDir, err)
			}
			if !strings.Contains(strings.ToLower(string(readmeBytes)), "non-prescriptive") {
				t.Fatalf("expected non-prescriptive disclaimer in %s/README.md", exDir)
			}

			var stdout, stderr bytes.Buffer
			if code := cli.ExecuteCode([]string{"resolve", "--dir", exDir, "--json"}, &stdout, &stderr); code != cli.ExitPass {
				t.Fatalf("resolve failed on %s (%d): %s\n%s", name, code, stdout.String(), stderr.String())
			}

			stdout.Reset()
			stderr.Reset()
			if code := cli.ExecuteCode([]string{"test", "--dir", exDir, "--json"}, &stdout, &stderr); code != cli.ExitPass {
				t.Fatalf("gradusctl test failed on %s (%d): %s\n%s", name, code, stdout.String(), stderr.String())
			}
			var testRep struct {
				Passed int `json:"passed"`
				Failed int `json:"failed"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &testRep); err != nil || testRep.Passed < 2 || testRep.Failed != 0 {
				t.Fatalf("expected >=2 passing rule test cases and 0 failures for %s, got err=%v rep=%+v", name, err, testRep)
			}
		})
	}
}

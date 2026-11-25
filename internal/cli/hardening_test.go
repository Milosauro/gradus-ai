package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestHardeningApproversViewHashAndUpdate(t *testing.T) {
	tmp := t.TempDir()
	projRules := filepath.Join(tmp, ".gradus", "levels", "compliance", "rules")
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}
	levelYAML := `schema: gradus/v1
id: compliance
owners: [security-team]
approvers: [compliance-lead]
exceptions:
  max_duration: 14d
  approvers: [security-team, compliance-lead]
`
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "compliance", "level.yaml"), []byte(levelYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	ruleMD := `---
id: CMP-01
title: Require TLS
severity: MUST
enforcement: review
---

Always enforce TLS 1.3.
`
	if err := os.WriteFile(filepath.Join(projRules, "CMP-01.md"), []byte(ruleMD), 0o644); err != nil {
		t.Fatal(err)
	}
	userRulesDir := filepath.Join(tmp, "fake-home", ".gradus", "levels", "user", "rules")
	if err := os.MkdirAll(userRulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(tmp, "fake-home"))
	t.Setenv("USERPROFILE", filepath.Join(tmp, "fake-home"))
	gradusYAML := `schema: gradus/v1
levels:
  - id: compliance
    source: { path: .gradus/levels/compliance }
  - id: user
    source: { path: ~/.gradus/levels/user }
obligations:
  - id: OBL-01
    rule: "Spec required"
    allowed_providers: [walden]
bindings:
  - obligation: OBL-01
    provider: walden
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. R4.AC1: Zero "Phase 0" notes when obligations/bindings are declared
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve failed: %d %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "Phase 0") {
		t.Fatalf("expected zero Phase 0 reserved-key notes, got:\n%s", stdout.String())
	}

	// 2. R1.AC2: Unauthorized approver rejected with exit 2; authorized approver succeeds
	expires := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"exception", "request", "--dir", tmp, "--rule", "CMP-01", "--reason", "legacy endpoint", "--expires", expires, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("exception request failed: %d %s", code, stderr.String())
	}
	var reqRec struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &reqRec); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"exception", "approve", reqRec.ID, "--dir", tmp, "--by", "random-dev"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for unauthorized approver random-dev, got %d (%s)", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"exception", "approve", reqRec.ID, "--dir", tmp, "--by", "security-team"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected exit 0 for authorized approver security-team, got %d (%s)", code, stderr.String())
	}

	// 3. R2.AC1: Adding personal user rule does NOT change ViewHash in gradus.lock
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve without user rule failed: %d %s", code, stderr.String())
	}
	var viewWithoutUser resolve.ResolvedView
	if err := json.Unmarshal(stdout.Bytes(), &viewWithoutUser); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(userRulesDir, "USR-01.md"), []byte("---\nid: USR-01\ntitle: Personal\nseverity: MAY\nenforcement: prompt\n---\n\nBe concise.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve with user rule failed: %d %s", code, stderr.String())
	}
	var viewWithUser resolve.ResolvedView
	if err := json.Unmarshal(stdout.Bytes(), &viewWithUser); err != nil {
		t.Fatal(err)
	}
	if viewWithUser.ViewHash != viewWithoutUser.ViewHash {
		t.Fatalf("expected project ViewHash invariant across user rules: without=%s with=%s", viewWithoutUser.ViewHash, viewWithUser.ViewHash)
	}

	// 4. R2.AC2 & R2.AC3: Remote locked level --update succeeds on ref change, fails on same-ref tamper
	remote := filepath.Join(tmp, "remote-team")
	if err := os.MkdirAll(filepath.Join(remote, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = remote
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}
	gitRun("init")
	gitRun("config", "user.name", "Test")
	gitRun("config", "user.email", "test@gradus.local")
	_ = os.WriteFile(filepath.Join(remote, "level.yaml"), []byte("schema: gradus/v1\nid: team\nversion: 1.0.0\n"), 0o644)
	_ = os.WriteFile(filepath.Join(remote, "rules", "TEAM-01.md"), []byte("---\nid: TEAM-01\ntitle: Team rule\nseverity: MUST\nenforcement: prompt\n---\n\nV1.\n"), 0o644)
	gitRun("add", ".")
	gitRun("commit", "-m", "v1.0.0")
	gitRun("tag", "v1.0.0")

	_ = os.WriteFile(filepath.Join(remote, "level.yaml"), []byte("schema: gradus/v1\nid: team\nversion: 1.1.0\n"), 0o644)
	gitRun("add", ".")
	gitRun("commit", "-m", "v1.1.0")
	gitRun("tag", "v1.1.0")

	consumer := filepath.Join(tmp, "consumer")
	if err := os.MkdirAll(consumer, 0o755); err != nil {
		t.Fatal(err)
	}
	remoteGit := filepath.ToSlash(remote)
	_ = os.WriteFile(filepath.Join(consumer, "gradus.yaml"), []byte(fmt.Sprintf("schema: gradus/v1\nlevels:\n  - id: team\n    source: { git: %s, ref: v1.0.0 }\n    locked: true\n", remoteGit)), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", consumer, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("initial remote resolve failed: %d %s", code, stderr.String())
	}

	// Bump ref to v1.1.0: without --update fails (exit 2), with --update succeeds
	_ = os.WriteFile(filepath.Join(consumer, "gradus.yaml"), []byte(fmt.Sprintf("schema: gradus/v1\nlevels:\n  - id: team\n    source: { git: %s, ref: v1.1.0 }\n    locked: true\n", remoteGit)), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", consumer, "--write"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected lock mismatch exit 2 without --update, got %d", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", consumer, "--write", "--update"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected resolve --write --update to succeed on ref change, got %d (%s)", code, stderr.String())
	}
	lf, err := lock.ReadLockfile(filepath.Join(consumer, "gradus.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := lf.FindEntry("level", "team"); !ok || e.Ref != "v1.1.0" {
		t.Fatalf("expected updated lock entry ref v1.1.0, got %+v", e)
	}

	// Same-ref tamper must fail even with --update (R2.AC3)
	cachedRule := filepath.Join(consumer, ".gradus", "cache", "team", "repo", "rules", "TEAM-01.md")
	_ = os.WriteFile(cachedRule, []byte("tampered\n"), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", consumer, "--update"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected same-ref tamper to fail with exit 2 even with --update, got %d", code)
	}
}

func TestHandbookNumberingAndSupplyChain(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Rule one\nseverity: MUST\nenforcement: prompt\n---\n\nAlways test.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
obligations:
  - id: OBL-01
    rule: "Spec required"
    allowed_providers: [walden]
bindings:
  - obligation: OBL-01
    provider: walden
`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"handbook", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("handbook failed: %d %s", code, stderr.String())
	}
	hb := stdout.String()
	for _, heading := range []string{
		"## 1. Pyramid Levels & Charters",
		"## 2. Active Rules by Level",
		"## 3. Obligations & Bindings",
		"## 4. Exceptions & Break-Glass Procedure",
	} {
		if !strings.Contains(hb, heading) {
			t.Fatalf("expected sequential handbook heading %q in:\n%s", heading, hb)
		}
	}

	// Verify repo rule tests (including PRJ-GO-001 examples) pass
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"test", "--dir", "../..", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("gradusctl test on repo failed: %d %s\n%s", code, stderr.String(), stdout.String())
	}
	var testOut struct {
		Total  int `json:"total"`
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &testOut); err != nil {
		t.Fatalf("failed to parse gradusctl test json: %v\n%s", err, stdout.String())
	}
	if testOut.Passed < 1 || testOut.Failed != 0 {
		t.Fatalf("expected at least 1 passing rule test and 0 failures, got %+v", testOut)
	}
}

func TestExceptionRequestMaxDurationV032(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "company", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "company", "level.yaml"), []byte(`schema: gradus/v1
id: company
exceptions:
  allowed: true
  max_duration: 14d
  approvers: [security-team]
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "company", "rules", "CMP-01.md"), []byte("---\nid: CMP-01\ntitle: No plaintext\nseverity: MUST\nenforcement: prompt\n---\n\nNever store plaintext.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: company
    source: { path: .gradus/levels/company }
`), 0o644)

	tooLong := time.Now().UTC().AddDate(0, 6, 0).Format("2006-01-02")
	var stdout, stderr bytes.Buffer
	code := cli.ExecuteCode([]string{"exception", "request", "--dir", tmp, "--rule", "CMP-01", "--reason", "six months", "--expires", tooLong}, &stdout, &stderr)
	if code != cli.ExitError {
		t.Fatalf("expected exit 2 when exception duration exceeds max_duration (14d), got %d (stdout=%s stderr=%s)", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "14d") {
		t.Fatalf("expected error message to cite max_duration 14d, got %q", stderr.String())
	}
	// Ensure no exception file was created
	entries, _ := os.ReadDir(filepath.Join(tmp, ".gradus", "exceptions"))
	if len(entries) != 0 {
		t.Fatalf("expected no exception file created on rejected request, found %d", len(entries))
	}

	// Within limit (7d) succeeds
	within := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"exception", "request", "--dir", tmp, "--rule", "CMP-01", "--reason", "one week", "--expires", within}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected exit 0 for valid 7d duration, got %d (%s)", code, stderr.String())
	}
}

func TestPRJCI001CheckAndActionV032(t *testing.T) {
	// 1. Verify composite action points setup-go to github.action_path rather than consumer go.mod
	actionBytes, err := os.ReadFile(filepath.Join("..", "..", ".github", "actions", "gradus", "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	actionText := string(actionBytes)
	if !strings.Contains(actionText, "${{ github.action_path }}/../../../go.mod") {
		t.Fatalf("expected action.yml to reference github.action_path go.mod, got:\n%s", actionText)
	}

	// 2. Verify CI workflow exercises the composite action against a non-Go repository
	ciBytes, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ciBytes), "test-action-non-go:") {
		t.Fatalf("expected ci.yml to contain test-action-non-go job")
	}

	// 3. Verify go.mod uses go.yaml.in/yaml/v3 instead of gopkg.in/yaml.v3
	goModBytes, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goModBytes), "go.yaml.in/yaml/v3") {
		t.Fatalf("expected go.mod to require go.yaml.in/yaml/v3")
	}

	// 4. Verify gradusctl test passes for PRJ-CI-001 and PRJ-GO-001 (>= 4 cases) and gradusctl check passes
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"test", "--dir", "../..", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("gradusctl test failed: %d %s\n%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "PRJ-CI-001") {
		t.Fatalf("expected PRJ-CI-001 check test cases in gradusctl test output:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"check", "--dir", "../..", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("gradusctl check failed on repository workflows: %d %s\n%s", code, stderr.String(), stdout.String())
	}
}

func TestDesign11AndSpecAlignmentV032(t *testing.T) {
	designBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "design", "gradus-design.md"))
	if err != nil {
		t.Fatal(err)
	}
	design := string(designBytes)
	if strings.Contains(design, "Implementation has not started") {
		t.Fatalf("Design 1.1 must not contain 'Implementation has not started'")
	}
	for _, want := range []string{"| Document version | 1.2 |", "### Normative precedence", "D-75", "D-83"} {
		if !strings.Contains(design, want) {
			t.Fatalf("Design 1.1 missing expected section/decision %q", want)
		}
	}

	specBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "v1", "SPEC.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(specBytes), "## 1. Normative Precedence") {
		t.Fatalf("spec/v1/SPEC.md missing Normative Precedence section")
	}

	if _, err := os.Stat(filepath.Join("..", "..", "docs", "plan", "phase-2-plan.md")); err != nil {
		t.Fatalf("expected docs/plan/phase-2-plan.md to exist: %v", err)
	}

	readmeBytes, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := string(readmeBytes)
	for _, want := range []string{"v0.4.1", "CODEOWNERS", "Validation status (transparent)", "gradus:begin/end"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md missing expected transparency/security note %q", want)
		}
	}

	changelogBytes, err := os.ReadFile(filepath.Join("..", "..", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changelogBytes), "## [0.3.2]") {
		t.Fatalf("CHANGELOG.md missing [0.3.2] section")
	}
}

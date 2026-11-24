package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestExceptionCommandWorkflow(t *testing.T) {
	tmp := t.TempDir()
	orgDir := filepath.Join(tmp, "levels", "org")
	teamDir := filepath.Join(tmp, "levels", "team")
	for _, d := range []string{filepath.Join(orgDir, "rules"), filepath.Join(teamDir, "rules")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(filepath.Join(tmp, "gradus.yaml"), `schema: gradus/v1
levels:
  - id: org
    source: { path: levels/org }
    offline: { mode: fail-closed }
  - id: team
    source: { path: levels/team }
targets: [agents-md]
`)
	write(filepath.Join(orgDir, "level.yaml"), `schema: gradus/v1
id: org
exceptions:
  break_glass: false
`)
	write(filepath.Join(teamDir, "level.yaml"), `schema: gradus/v1
id: team
exceptions:
  max_duration: 30d
  break_glass: true
`)
	write(filepath.Join(orgDir, "rules", "ORG-01.md"), `---
id: ORG-01
title: Org invariant
severity: MUST
---
Org invariant.
`)
	write(filepath.Join(teamDir, "rules", "TEAM-01.md"), `---
id: TEAM-01
title: Team rule
severity: SHOULD
---
Team rule.
`)

	var stdout, stderr bytes.Buffer

	// 1. Request pending exception.
	reqExpires := time.Now().UTC().AddDate(0, 0, 14).Format("2006-01-02")
	if code := cli.ExecuteCode([]string{
		"exception", "request",
		"--rule", "TEAM-01",
		"--reason", "Legacy migration",
		"--expires", reqExpires,
		"--dir", tmp,
		"--json",
	}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("exception request failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	var reqOut struct {
		ID         string   `json:"id"`
		ApprovedBy []string `json:"approved_by"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &reqOut); err != nil {
		t.Fatalf("invalid request json: %v\n%s", err, stdout.String())
	}
	if reqOut.ID == "" || len(reqOut.ApprovedBy) != 0 {
		t.Fatalf("expected non-empty ID and empty approved_by, got %+v", reqOut)
	}

	// 2. List shows pending.
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"exception", "list", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("exception list failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	var listOut struct {
		Exceptions []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"exceptions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &listOut); err != nil {
		t.Fatal(err)
	}
	if len(listOut.Exceptions) != 1 || listOut.Exceptions[0].Status != "pending" {
		t.Fatalf("expected 1 pending exception, got %+v", listOut.Exceptions)
	}

	// 3. Approve exception -> list shows active.
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"exception", "approve", reqOut.ID, "--by", "lead@example.com", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("exception approve failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	_ = cli.ExecuteCode([]string{"exception", "list", "--dir", tmp, "--json"}, &stdout, &stderr)
	listOut.Exceptions = nil
	if err := json.Unmarshal(stdout.Bytes(), &listOut); err != nil {
		t.Fatal(err)
	}
	if len(listOut.Exceptions) != 1 || listOut.Exceptions[0].Status != "active" {
		t.Fatalf("expected 1 active exception after approve, got %+v", listOut.Exceptions)
	}

	// 4. Expire exception -> list shows expired.
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"exception", "expire", reqOut.ID, "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("exception expire failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	_ = cli.ExecuteCode([]string{"exception", "list", "--dir", tmp, "--json"}, &stdout, &stderr)
	listOut.Exceptions = nil
	if err := json.Unmarshal(stdout.Bytes(), &listOut); err != nil {
		t.Fatal(err)
	}
	if len(listOut.Exceptions) != 1 || listOut.Exceptions[0].Status != "expired" {
		t.Fatalf("expected 1 expired exception after expire, got %+v", listOut.Exceptions)
	}

	// 5. Break-glass allowed on team, disallowed on org.
	bgExpires := time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02")
	bgPostReview := time.Now().UTC().AddDate(0, 0, 10).Format("2006-01-02")
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{
		"exception", "break-glass",
		"--rule", "TEAM-01",
		"--reason", "Hotfix incident 42",
		"--expires", bgExpires,
		"--post-review-due", bgPostReview,
		"--by", "oncall@example.com",
		"--dir", tmp,
		"--json",
	}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected break-glass on team to succeed, got %d: %s\n%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{
		"exception", "break-glass",
		"--rule", "ORG-01",
		"--reason", "Hotfix incident 43",
		"--expires", bgExpires,
		"--post-review-due", bgPostReview,
		"--by", "oncall@example.com",
		"--dir", tmp,
		"--json",
	}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected break-glass on org (break_glass: false) to exit 1, got %d (%s)", code, stdout.String())
	}
}

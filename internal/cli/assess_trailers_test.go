package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/assess"
	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/trailers"
)

func TestAssessAndTrailersCLI(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Test\nseverity: MUST\nenforcement: prompt\n---\n\nTest.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --write failed: %d %s", code, stderr.String())
	}

	// 1. gradusctl assess --json
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"assess", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("assess failed: %d %s", code, stderr.String())
	}
	var rep assess.Report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.MaxScore != 100 || len(rep.Dimensions) != 5 {
		t.Fatalf("unexpected assess report: %+v", rep)
	}

	// 2. trailers generate invalid --model-tier exits 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"trailers", "generate", "--dir", tmp, "--model-tier", "tier-bogus"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for invalid --model-tier, got %d", code)
	}

	// 3. trailers generate valid lines
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"trailers", "generate", "--dir", tmp, "--agent", "claude-code", "--model-tier", "tier-high", "--review-id", "ev-123"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("trailers generate failed: %d %s", code, stderr.String())
	}
	trailerBlock := stdout.String()
	for _, want := range []string{"Gradus-Lock-Hash:", "Gradus-View-Hash:", "Gradus-Agent: claude-code", "Gradus-Model-Tier: tier-high", "Gradus-Review-ID: ev-123"} {
		if !strings.Contains(trailerBlock, want) {
			t.Fatalf("expected %q in generated trailers:\n%s", want, trailerBlock)
		}
	}

	// 4. Git commit with generated trailers passes trailers verify --require-lock-hash
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}
	runGit("init")
	runGit("config", "user.name", "Gradus Test")
	runGit("config", "user.email", "test@gradus.local")
	runGit("add", ".")
	runGit("commit", "-m", "feat: initial commit\n\n"+trailerBlock)

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"trailers", "verify", "--dir", tmp, "--ref", "HEAD", "--require-lock-hash", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("trailers verify failed: %d %s\n%s", code, stderr.String(), stdout.String())
	}
	var vr trailers.VerifyReport
	if err := json.Unmarshal(stdout.Bytes(), &vr); err != nil {
		t.Fatal(err)
	}
	if vr.Verdict != "pass" || vr.Trailers["Gradus-Agent"] != "claude-code" {
		t.Fatalf("unexpected verify report: %+v", vr)
	}

	// 5. Commit without trailers fails --require-lock-hash with exit 1
	_ = os.WriteFile(filepath.Join(tmp, "note.txt"), []byte("hi\n"), 0o644)
	runGit("add", "note.txt")
	runGit("commit", "-m", "chore: no trailers")
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"trailers", "verify", "--dir", tmp, "--ref", "HEAD", "--require-lock-hash", "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on commit missing Gradus-Lock-Hash, got %d", code)
	}
}

package cli_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/state"
)

func TestLessonsCommand(t *testing.T) {
	tmp := t.TempDir()
	lessonsDir := t.TempDir()

	// 1. Add lesson with secret and email
	code, out, _ := runCLI(t, []string{
		"lessons", "add",
		"--dir", tmp,
		"--lessons-dir", lessonsDir,
		"--kind", "correction",
		"--statement", "Do not leak ghp_1234567890abcdef123456 or dev@company.com",
		"--level", "project",
		"--enforcement", "check",
		"--json",
	})
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for lessons add, got %d (%s)", code, out)
	}
	var addResp struct {
		Lesson state.LessonRecord `json:"lesson"`
		Path   string             `json:"path"`
	}
	if err := json.Unmarshal([]byte(out), &addResp); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(addResp.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "ghp_") || strings.Contains(string(raw), "dev@company.com") {
		t.Fatalf("expected secrets/PII redacted on disk, got:\n%s", string(raw))
	}

	// 2. List lessons
	code, out, _ = runCLI(t, []string{"lessons", "list", "--dir", tmp, "--lessons-dir", lessonsDir, "--json"})
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for lessons list, got %d", code)
	}
	var listResp struct {
		Lessons []state.LessonRecord `json:"lessons"`
	}
	if err := json.Unmarshal([]byte(out), &listResp); err != nil || len(listResp.Lessons) != 1 {
		t.Fatalf("expected 1 lesson listed, got %+v (err=%v)", listResp, err)
	}

	// 3. Update status to promoted and increment occurrences
	code, out, _ = runCLI(t, []string{
		"lessons", "update",
		"--dir", tmp,
		"--lessons-dir", lessonsDir,
		"--id", addResp.Lesson.ID,
		"--status", "promoted",
		"--increment",
		"--json",
	})
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for lessons update, got %d (%s)", code, out)
	}
	var updResp struct {
		Lesson state.LessonRecord `json:"lesson"`
	}
	if err := json.Unmarshal([]byte(out), &updResp); err != nil {
		t.Fatal(err)
	}
	if updResp.Lesson.Status != "promoted" || updResp.Lesson.Occurrences != 2 {
		t.Fatalf("unexpected updated lesson: %+v", updResp.Lesson)
	}
}

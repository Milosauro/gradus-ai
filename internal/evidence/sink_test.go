package evidence_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestEvidenceSinks(t *testing.T) {
	tmp := t.TempDir()
	now := time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC)

	// 1. Seed an expired local record (> 30d old)
	localDir := filepath.Join(tmp, ".gradus", "evidence")
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	expiredRec := `{"schema":"gradus/v1","id":"ev-old","kind":"check","timestamp":"2026-09-01T00:00:00Z","lock_hash":"sha256:a","view_hash":"sha256:b","producer":{"tool":"gradusctl","version":"0.1.0"},"findings":[],"verdict":"pass"}`
	if err := os.WriteFile(filepath.Join(localDir, "ev-old.json"), []byte(expiredRec), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Test all 5 sinks: none, local, ci-artifact, repo, command
	cmdOutFile := filepath.Join(tmp, "cmd-sink.json")
	sinkCmd := "cat > cmd-sink.json"
	if runtime.GOOS == "windows" {
		sinkCmd = `findstr /R ".*" > cmd-sink.json`
	}
	cfg := &schema.EvidenceConfig{
		Sinks: []schema.SinkConfig{
			{Type: "none"},
			{Type: "local", Retention: "30d"},
			{Type: "ci-artifact", Path: "artifacts"},
			{Type: "repo", Path: "audit/evidence"},
			{Type: "command", Command: sinkCmd},
		},
	}

	rec := evidence.NewRecord(tmp, "check", "pass", "origin/main", "sha256:lock1", "sha256:view1", nil, now)
	written, err := evidence.WriteRecord(tmp, cfg, rec, now)
	if err != nil {
		t.Fatalf("WriteRecord failed: %v", err)
	}
	if len(written) != 4 { // none produces no written path
		t.Fatalf("expected 4 written targets (local, ci-artifact, repo, command), got %v", written)
	}

	// Verify expired record was pruned and new record is listed
	listed, err := evidence.ListLocalRecords(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != rec.ID {
		t.Fatalf("expected only new record after retention pruning, got: %+v", listed)
	}

	// Verify ci-artifact, repo, and command sink files exist and contain view_hash
	for _, p := range []string{
		filepath.Join(tmp, "artifacts", rec.ID+".json"),
		filepath.Join(tmp, "audit", "evidence", rec.ID+".json"),
		cmdOutFile,
	} {
		data, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(data), "sha256:view1") {
			t.Fatalf("expected evidence written to %s, err=%v", p, err)
		}
	}

	// 3. Test CI sink override
	t.Setenv("CI", "true")
	cfgWithCI := &schema.EvidenceConfig{
		Sinks: []schema.SinkConfig{{Type: "local"}},
		CI:    &schema.EvidenceSinksOnly{Sinks: []schema.SinkConfig{{Type: "ci-artifact", Path: "ci-only"}}},
	}
	active := evidence.ActiveSinks(cfgWithCI, true)
	if len(active) != 1 || active[0].Type != "ci-artifact" {
		t.Fatalf("expected ci-artifact sink in CI, got: %+v", active)
	}
}

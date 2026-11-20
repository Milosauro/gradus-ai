package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/evidence"
)

func TestEvidenceCommand(t *testing.T) {
	tmp := t.TempDir()

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"init", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("init failed: %d %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"evidence", "write", "--dir", tmp, "--kind", "check", "--verdict", "pass", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("evidence write failed: %d %s", code, stderr.String())
	}

	var writeOut struct {
		Record  evidence.Record `json:"record"`
		Written []string        `json:"written"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &writeOut); err != nil {
		t.Fatalf("invalid evidence write JSON: %v", err)
	}
	if writeOut.Record.Kind != "check" || writeOut.Record.Verdict != "pass" || len(writeOut.Written) != 1 {
		t.Fatalf("unexpected evidence write output: %+v", writeOut)
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"evidence", "list", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("evidence list failed: %d %s", code, stderr.String())
	}
	var listed []evidence.Record
	if err := json.Unmarshal(stdout.Bytes(), &listed); err != nil {
		t.Fatalf("invalid evidence list JSON: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != writeOut.Record.ID {
		t.Fatalf("expected listed record %s, got: %+v", writeOut.Record.ID, listed)
	}
}

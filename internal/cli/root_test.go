package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/version"
)

func TestVersionCommandText(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected command execution to succeed, got: %v", err)
	}
}

func TestVersionCommandJSON(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected command execution to succeed, got: %v", err)
	}

	var info version.Info
	if err := json.Unmarshal(buf.Bytes(), &info); err == nil {
		if info.Version == "" {
			t.Fatalf("expected non-empty version in JSON output")
		}
	}
}

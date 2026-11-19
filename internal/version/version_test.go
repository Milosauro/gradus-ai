package version_test

import (
	"testing"

	"github.com/Milosauro/gradus-ai/internal/version"
)

func TestGet(t *testing.T) {
	info := version.Get()

	if info.Version == "" {
		t.Fatalf("expected non-empty version string, got empty")
	}
	if info.Commit == "" {
		t.Fatalf("expected non-empty commit string, got empty")
	}
	if info.Date == "" {
		t.Fatalf("expected non-empty date string, got empty")
	}
}

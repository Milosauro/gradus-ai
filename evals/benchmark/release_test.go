package benchmark_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestReleasePipelineConfig(t *testing.T) {
	root := filepath.Join("..", "..")

	// 1. Validate .goreleaser.yaml
	grBytes, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("missing .goreleaser.yaml: %v", err)
	}
	gr := string(grBytes)
	for _, req := range []string{"CGO_ENABLED=0", "checksum:", "sboms:", "signs:", "cosign"} {
		if !strings.Contains(gr, req) {
			t.Fatalf(".goreleaser.yaml missing %q", req)
		}
	}

	// 2. Validate .github/workflows/release.yml and strict 40-char SHA pinning (PRJ-CI-001 / T3)
	relBytes, err := os.ReadFile(filepath.Join(root, ".github/workflows/release.yml"))
	if err != nil {
		t.Fatalf("missing .github/workflows/release.yml: %v", err)
	}
	rel := string(relBytes)
	if !strings.Contains(rel, "'v*'") || !strings.Contains(rel, "attest-build-provenance") {
		t.Fatalf(".github/workflows/release.yml missing v* trigger or provenance attestation")
	}

	usesRe := regexp.MustCompile(`uses:\s*([^\s#]+)`)
	pinnedRe := regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}$`)
	matches := usesRe.FindAllStringSubmatch(rel, -1)
	if len(matches) < 4 {
		t.Fatalf("expected >= 4 GitHub Actions uses: entries in release.yml, got %d", len(matches))
	}
	for _, m := range matches {
		if !pinnedRe.MatchString(m[1]) {
			t.Fatalf("release.yml action %q is not pinned to a 40-char commit SHA", m[1])
		}
	}

	// 3. Validate CHANGELOG.md has 0.1.0 section
	clBytes, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("missing CHANGELOG.md: %v", err)
	}
	if !strings.Contains(string(clBytes), "## [0.1.0]") {
		t.Fatalf("CHANGELOG.md missing [0.1.0] section")
	}
}

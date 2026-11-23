package schema_test

import (
	"testing"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestSemVerAndRemoteLevelValidation(t *testing.T) {
	v1, err := schema.ParseSemVer("v1.2.3")
	if err != nil {
		t.Fatalf("ParseSemVer(v1.2.3) failed: %v", err)
	}
	v2, err := schema.ParseSemVer("1.2.4")
	if err != nil {
		t.Fatalf("ParseSemVer(1.2.4) failed: %v", err)
	}
	if schema.CompareSemVer(v1, v2) >= 0 {
		t.Fatalf("expected v1.2.3 < 1.2.4")
	}

	pre, err := schema.ParseSemVer("1.2.3-rc.1")
	if err != nil {
		t.Fatalf("ParseSemVer(1.2.3-rc.1) failed: %v", err)
	}
	if schema.CompareSemVer(pre, v1) >= 0 {
		t.Fatalf("expected 1.2.3-rc.1 < v1.2.3")
	}

	if _, err := schema.ParseSemVer("not-a-version"); err == nil {
		t.Fatalf("expected invalid semver to fail")
	}
	if _, err := schema.ParseSemVer("01.2.3"); err == nil {
		t.Fatalf("expected semver with leading zero to fail")
	}

	// Valid signature & min_version in gradus.yaml
	validPyramid := `schema: gradus/v1
levels:
  - id: team
    source: { git: https://example.com/team.git, ref: v1.3.0 }
    locked: true
    signature: required
    min_version: 1.2.0
`
	if _, err := schema.ParsePyramidConfig([]byte(validPyramid)); err != nil {
		t.Fatalf("expected valid pyramid config, got: %v", err)
	}

	// Invalid signature in gradus.yaml
	invalidSigPyramid := `schema: gradus/v1
levels:
  - id: team
    source: { git: https://example.com/team.git, ref: v1.3.0 }
    signature: strict
`
	if _, err := schema.ParsePyramidConfig([]byte(invalidSigPyramid)); err == nil {
		t.Fatalf("expected invalid signature in gradus.yaml to fail")
	}

	// Invalid min_version in gradus.yaml
	invalidMinVerPyramid := `schema: gradus/v1
levels:
  - id: team
    source: { git: https://example.com/team.git, ref: v1.3.0 }
    min_version: not-a-version
`
	if _, err := schema.ParsePyramidConfig([]byte(invalidMinVerPyramid)); err == nil {
		t.Fatalf("expected invalid min_version in gradus.yaml to fail")
	}

	// Valid and invalid level.yaml
	validLevel := `schema: gradus/v1
id: team
version: 1.3.0
min_version: v1.2.0
signature: optional
`
	if _, err := schema.ParseLevelConfig([]byte(validLevel)); err != nil {
		t.Fatalf("expected valid level config, got: %v", err)
	}

	invalidLevel := `schema: gradus/v1
id: team
min_version: bad
`
	if _, err := schema.ParseLevelConfig([]byte(invalidLevel)); err == nil {
		t.Fatalf("expected invalid min_version in level.yaml to fail")
	}
}

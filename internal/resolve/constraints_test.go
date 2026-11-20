package resolve_test

import (
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestConstraintsAndOverlayOrdering(t *testing.T) {
	levels := []resolve.PyramidStep{
		{Kind: "level", ID: "org"},
		{Kind: "level", ID: "team"},
		{Kind: "level", ID: "project"},
	}
	overlays := []resolve.PyramidStep{
		{Kind: "overlay", ID: "gdpr", Strength: "org"},
		{Kind: "overlay", ID: "stack-go", Strength: "team"},
	}

	ordered, ovConflicts := resolve.OrderPyramidSteps(levels, overlays)
	if len(ovConflicts) != 0 {
		t.Fatalf("expected no ordering conflicts, got: %+v", ovConflicts)
	}

	var gotOrder []string
	for _, s := range ordered {
		gotOrder = append(gotOrder, s.ID)
	}
	want := "org,gdpr,team,stack-go,project"
	if strings.Join(gotOrder, ",") != want {
		t.Fatalf("expected order %s, got %s", want, strings.Join(gotOrder, ","))
	}

	// Inherited constraints violation checks
	maxExtra := 0
	cfg := schema.PyramidConfig{
		Schema: schema.SchemaVersion,
		Levels: []schema.LevelRef{
			{ID: "team"},
			{ID: "extra-mid"},
			{ID: "project"},
		},
		Overlays: []schema.OverlayRef{
			{ID: "unapproved-overlay", Strength: "team"},
		},
	}
	levelConfigs := map[string]*schema.LevelConfig{
		"team": {
			Schema: schema.SchemaVersion,
			ID:     "team",
			Constraints: &schema.Constraints{
				RequireLevels:   []string{"project", "security"},
				MaxExtraLevels:  &maxExtra,
				AllowedOverlays: []string{"stack-*", "gdpr"},
			},
		},
	}

	conflicts := resolve.CheckInheritedConstraints(cfg, levelConfigs)
	if len(conflicts) != 3 {
		t.Fatalf("expected 3 constraint conflicts (missing security, max_extra_levels, disallowed overlay), got %d: %+v", len(conflicts), conflicts)
	}
}

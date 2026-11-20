package resolve

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

// PyramidStep represents an ordered level or overlay in the resolution pipeline.
type PyramidStep struct {
	Kind          string // "level" or "overlay"
	ID            string
	Strength      string
	Locked        bool
	EditableRules []string
	Dir           string
	SourceLabel   string
	LevelConfig   *schema.LevelConfig
}

// CheckInheritedConstraints validates lower levels and overlays against upper-level constraints (§7.2).
func CheckInheritedConstraints(cfg schema.PyramidConfig, levelConfigs map[string]*schema.LevelConfig) []Conflict {
	var conflicts []Conflict

	presentLevels := make(map[string]int, len(cfg.Levels))
	for i, l := range cfg.Levels {
		presentLevels[l.ID] = i
	}

	for idx, lvl := range cfg.Levels {
		lc, ok := levelConfigs[lvl.ID]
		if !ok || lc == nil || lc.Constraints == nil {
			continue
		}
		c := lc.Constraints

		for _, req := range c.RequireLevels {
			if _, exists := presentLevels[req]; !exists {
				conflicts = append(conflicts, Conflict{
					Kind:    "constraint",
					Level:   lvl.ID,
					Message: fmt.Sprintf("level %q requires level %q to be present in gradus.yaml", lvl.ID, req),
				})
			}
		}

		if c.MaxExtraLevels != nil {
			// Count levels below this level excluding required levels and "user".
			reqSet := make(map[string]bool, len(c.RequireLevels))
			for _, r := range c.RequireLevels {
				reqSet[r] = true
			}
			extra := 0
			for j := idx + 1; j < len(cfg.Levels); j++ {
				belowID := cfg.Levels[j].ID
				if belowID == "user" || reqSet[belowID] {
					continue
				}
				extra++
			}
			if extra > *c.MaxExtraLevels {
				conflicts = append(conflicts, Conflict{
					Kind:    "constraint",
					Level:   lvl.ID,
					Message: fmt.Sprintf("level %q allows at most %d extra level(s) below it, found %d", lvl.ID, *c.MaxExtraLevels, extra),
				})
			}
		}

		if len(c.AllowedOverlays) > 0 {
			for _, ov := range cfg.Overlays {
				if !matchesAnyPattern(ov.ID, c.AllowedOverlays) {
					conflicts = append(conflicts, Conflict{
						Kind:    "constraint",
						Level:   lvl.ID,
						Target:  ov.ID,
						Message: fmt.Sprintf("overlay %q is not permitted by level %q allowed_overlays %v", ov.ID, lvl.ID, c.AllowedOverlays),
					})
				}
			}
		}
	}

	return conflicts
}

// OrderPyramidSteps builds the ordered sequence of levels with each active overlay
// inserted immediately after the level matching its strength (§10 step 5).
func OrderPyramidSteps(levels []PyramidStep, overlays []PyramidStep) ([]PyramidStep, []Conflict) {
	var ordered []PyramidStep
	var conflicts []Conflict

	levelSeen := make(map[string]bool, len(levels))
	for _, l := range levels {
		levelSeen[l.ID] = true
	}

	overlaysByStrength := make(map[string][]PyramidStep)
	for _, ov := range overlays {
		if !levelSeen[ov.Strength] {
			conflicts = append(conflicts, Conflict{
				Kind:    "constraint",
				Target:  ov.ID,
				Message: fmt.Sprintf("overlay %q declares unknown strength level %q", ov.ID, ov.Strength),
			})
			continue
		}
		overlaysByStrength[ov.Strength] = append(overlaysByStrength[ov.Strength], ov)
	}

	for _, l := range levels {
		ordered = append(ordered, l)
		if attached, ok := overlaysByStrength[l.ID]; ok {
			ordered = append(ordered, attached...)
		}
	}

	return ordered, conflicts
}

func matchesAnyPattern(value string, patterns []string) bool {
	for _, p := range patterns {
		if ok, err := doublestar.Match(p, value); err == nil && ok {
			return true
		}
	}
	return false
}

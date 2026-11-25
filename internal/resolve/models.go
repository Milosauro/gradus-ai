package resolve

import (
	"fmt"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

// ResolvedModelPolicy holds the monotonically merged capability tiers and operation requirements (§21).
type ResolvedModelPolicy struct {
	Tiers      map[string]map[string]string `json:"tiers,omitempty"`
	Operations map[string]string            `json:"operations,omitempty"`
}

// TierRank returns the numeric rank of a capability tier (tier-light=1 < tier-standard=2 < tier-high=3).
func TierRank(tier string) int {
	switch strings.TrimSpace(tier) {
	case "tier-light":
		return 1
	case "tier-standard":
		return 2
	case "tier-high":
		return 3
	default:
		return 0
	}
}

// ModelForSkill returns the concrete model name configured for target (e.g. "claude-code")
// and a Gradus skill name (e.g. "gradus-review" -> operation "review"), or "" if unconfigured.
func (m *ResolvedModelPolicy) ModelForSkill(target, skillName string) string {
	if m == nil {
		return ""
	}
	op := strings.TrimPrefix(strings.TrimSpace(skillName), "gradus-")
	tier, ok := m.Operations[op]
	if !ok || tier == "" {
		return ""
	}
	agents, ok := m.Tiers[tier]
	if !ok {
		return ""
	}
	return strings.TrimSpace(agents[target])
}

// ResolveModelPolicy merges model policies across levels in pyramid order and project config.
// Lower levels may tighten an operation's tier (e.g. tier-standard -> tier-high), but relaxing
// an upper level's tier emits a blocking model_policy conflict (§21).
func ResolveModelPolicy(steps []PyramidStep, projectModels *schema.ModelPolicy) (*ResolvedModelPolicy, []Conflict) {
	tiers := make(map[string]map[string]string)
	ops := make(map[string]string)
	opOwner := make(map[string]string)
	var conflicts []Conflict
	hasAny := false

	mergeOne := func(levelID string, mp *schema.ModelPolicy) {
		if mp == nil {
			return
		}
		hasAny = true
		for tier, agents := range mp.Tiers {
			if tiers[tier] == nil {
				tiers[tier] = make(map[string]string)
			}
			for agent, model := range agents {
				tiers[tier][agent] = model
			}
		}
		for op, tier := range mp.Operations {
			prevTier, exists := ops[op]
			if !exists {
				ops[op] = tier
				opOwner[op] = levelID
				continue
			}
			if TierRank(tier) < TierRank(prevTier) {
				conflicts = append(conflicts, Conflict{
					Kind:    "model_policy",
					Target:  op,
					Level:   levelID,
					Message: fmt.Sprintf("level %q cannot relax operation %q model tier from %s (set by %q) to %s", levelID, op, prevTier, opOwner[op], tier),
				})
				continue
			}
			if TierRank(tier) > TierRank(prevTier) {
				ops[op] = tier
				opOwner[op] = levelID
			}
		}
	}

	for _, step := range steps {
		if step.LevelConfig != nil {
			mergeOne(step.ID, step.LevelConfig.Models)
		}
	}
	if projectModels != nil {
		mergeOne("gradus.yaml", projectModels)
	}

	if !hasAny {
		return nil, nil
	}
	return &ResolvedModelPolicy{
		Tiers:      tiers,
		Operations: ops,
	}, conflicts
}

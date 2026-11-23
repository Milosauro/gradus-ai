package resolve

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func resolveObligationsAndBindings(cfg schema.PyramidConfig, steps []PyramidStep) ([]ResolvedObligation, []ResolvedBinding, []Conflict) {
	obMap := make(map[string]*ResolvedObligation)
	var conflicts []Conflict

	applyObligation := func(levelID string, o schema.Obligation) {
		existing, ok := obMap[o.ID]
		if !ok {
			obMap[o.ID] = &ResolvedObligation{
				ID:               o.ID,
				Rule:             o.Rule,
				Level:            levelID,
				AllowedProviders: normalizeProviders(o.AllowedProviders),
				Scope:            append([]string(nil), o.Scope...),
			}
			return
		}

		// Lower levels may narrow allowed_providers, never widen them (§19 / P2).
		if len(existing.AllowedProviders) > 0 {
			if len(o.AllowedProviders) == 0 {
				conflicts = append(conflicts, Conflict{
					Kind:    "obligation",
					RuleID:  o.ID,
					Level:   levelID,
					Message: fmt.Sprintf("level %q cannot widen allowed_providers for obligation %q (upper allows %v)", levelID, o.ID, existing.AllowedProviders),
				})
				return
			}
			allowedSet := make(map[string]bool, len(existing.AllowedProviders))
			for _, p := range existing.AllowedProviders {
				allowedSet[p] = true
			}
			for _, p := range o.AllowedProviders {
				if !allowedSet[p] {
					conflicts = append(conflicts, Conflict{
						Kind:    "obligation",
						RuleID:  o.ID,
						Level:   levelID,
						Message: fmt.Sprintf("level %q cannot widen allowed_providers for obligation %q with provider %q (upper allows %v)", levelID, o.ID, p, existing.AllowedProviders),
					})
					return
				}
			}
		}
		if len(o.AllowedProviders) > 0 {
			existing.AllowedProviders = normalizeProviders(o.AllowedProviders)
		}
		if o.Rule != "" {
			existing.Rule = o.Rule
		}
		if len(o.Scope) > 0 {
			existing.Scope = append([]string(nil), o.Scope...)
		}
	}

	var allBindings []ResolvedBinding
	for _, step := range steps {
		if step.LevelConfig == nil {
			continue
		}
		for _, o := range step.LevelConfig.Obligations {
			applyObligation(step.ID, o)
		}
		for _, b := range step.LevelConfig.Bindings {
			allBindings = append(allBindings, ResolvedBinding{
				Obligation: b.Obligation,
				Provider:   b.Provider,
				Level:      step.ID,
				Requires:   b.Requires,
				Scope:      append([]string(nil), b.Scope...),
			})
		}
	}
	for _, o := range cfg.Obligations {
		applyObligation("project", o)
	}
	for _, b := range cfg.Bindings {
		allBindings = append(allBindings, ResolvedBinding{
			Obligation: b.Obligation,
			Provider:   b.Provider,
			Level:      "project",
			Requires:   b.Requires,
			Scope:      append([]string(nil), b.Scope...),
		})
	}

	if len(obMap) == 0 && len(allBindings) == 0 {
		return nil, nil, nil
	}

	sort.SliceStable(allBindings, func(i, j int) bool {
		if allBindings[i].Obligation != allBindings[j].Obligation {
			return allBindings[i].Obligation < allBindings[j].Obligation
		}
		if allBindings[i].Provider != allBindings[j].Provider {
			return allBindings[i].Provider < allBindings[j].Provider
		}
		if allBindings[i].Level != allBindings[j].Level {
			return allBindings[i].Level < allBindings[j].Level
		}
		return strings.Join(allBindings[i].Scope, ",") < strings.Join(allBindings[j].Scope, ",")
	})

	for _, b := range allBindings {
		ob, ok := obMap[b.Obligation]
		if !ok {
			conflicts = append(conflicts, Conflict{
				Kind:    "obligation",
				RuleID:  b.Obligation,
				Target:  b.Provider,
				Level:   b.Level,
				Message: fmt.Sprintf("binding in level %q references unknown obligation %q", b.Level, b.Obligation),
			})
			continue
		}
		if len(ob.AllowedProviders) > 0 && !containsProvider(ob.AllowedProviders, b.Provider) {
			conflicts = append(conflicts, Conflict{
				Kind:    "obligation",
				RuleID:  b.Obligation,
				Target:  b.Provider,
				Level:   b.Level,
				Message: fmt.Sprintf("binding for obligation %q uses disallowed provider %q (allowed: %v)", b.Obligation, b.Provider, ob.AllowedProviders),
			})
			continue
		}
		ob.Bindings = append(ob.Bindings, b)
		ob.Satisfied = true
	}

	obIDs := make([]string, 0, len(obMap))
	for id := range obMap {
		obIDs = append(obIDs, id)
	}
	sort.Strings(obIDs)

	resolvedObligations := make([]ResolvedObligation, 0, len(obIDs))
	for _, id := range obIDs {
		ob := obMap[id]
		if !ob.Satisfied {
			conflicts = append(conflicts, Conflict{
				Kind:    "obligation",
				RuleID:  ob.ID,
				Level:   ob.Level,
				Message: fmt.Sprintf("obligation %q (%s) has no valid binding", ob.ID, ob.Level),
			})
		}
		resolvedObligations = append(resolvedObligations, *ob)
	}

	return resolvedObligations, allBindings, conflicts
}

func containsProvider(list []string, target string) bool {
	for _, p := range list {
		if p == target {
			return true
		}
	}
	return false
}

func normalizeProviders(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(list))
	out := make([]string, 0, len(list))
	for _, p := range list {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

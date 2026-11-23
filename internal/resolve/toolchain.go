package resolve

import (
	"fmt"
	"sort"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func resolveToolchain(cfg schema.PyramidConfig, steps []PyramidStep) (*ResolvedToolchain, []Conflict) {
	reqMap := make(map[string]ResolvedToolchainItem)
	recMap := make(map[string]ResolvedToolchainItem)
	forbMap := make(map[string]ResolvedToolchainItem)

	addItems := func(levelID string, tc *schema.ToolchainConfig) {
		if tc == nil {
			return
		}
		for _, it := range tc.Required {
			key := it.Type + "\x00" + it.ID
			if _, exists := reqMap[key]; !exists {
				reqMap[key] = ResolvedToolchainItem{Type: it.Type, ID: it.ID, Level: levelID, Reason: it.Reason}
			}
		}
		for _, it := range tc.Recommended {
			key := it.Type + "\x00" + it.ID
			if _, exists := recMap[key]; !exists {
				recMap[key] = ResolvedToolchainItem{Type: it.Type, ID: it.ID, Level: levelID, Reason: it.Reason}
			}
		}
		for _, it := range tc.Forbidden {
			key := it.Type + "\x00" + it.ID
			if _, exists := forbMap[key]; !exists {
				forbMap[key] = ResolvedToolchainItem{Type: it.Type, ID: it.ID, Level: levelID, Reason: it.Reason}
			}
		}
	}

	for _, step := range steps {
		if step.LevelConfig != nil {
			addItems(step.ID, step.LevelConfig.Toolchain)
		}
	}
	addItems("project", cfg.Toolchain)

	if len(reqMap) == 0 && len(recMap) == 0 && len(forbMap) == 0 {
		return nil, nil
	}

	toSortedSlice := func(m map[string]ResolvedToolchainItem) []ResolvedToolchainItem {
		if len(m) == 0 {
			return nil
		}
		out := make([]ResolvedToolchainItem, 0, len(m))
		for _, v := range m {
			out = append(out, v)
		}
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Type != out[j].Type {
				return out[i].Type < out[j].Type
			}
			if out[i].ID != out[j].ID {
				return out[i].ID < out[j].ID
			}
			return out[i].Level < out[j].Level
		})
		return out
	}

	out := &ResolvedToolchain{
		Required:    toSortedSlice(reqMap),
		Recommended: toSortedSlice(recMap),
		Forbidden:   toSortedSlice(forbMap),
	}

	var conflicts []Conflict
	checkAgainstForbidden := func(section string, items []ResolvedToolchainItem) {
		for _, it := range items {
			for _, f := range out.Forbidden {
				if it.Type != f.Type {
					continue
				}
				matched, err := doublestar.Match(f.ID, it.ID)
				if err == nil && matched {
					conflicts = append(conflicts, Conflict{
						Kind:    "toolchain",
						RuleID:  it.Type + ":" + it.ID,
						Target:  f.ID,
						Level:   it.Level,
						Message: fmt.Sprintf("level %q declares %s %s %q which matches forbidden pattern %q (declared at %s)", it.Level, section, it.Type, it.ID, f.ID, f.Level),
					})
				}
			}
		}
	}

	checkAgainstForbidden("required", out.Required)
	checkAgainstForbidden("recommended", out.Recommended)

	return out, conflicts
}

package compile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// RenderHandbook deterministically renders a human-readable Markdown onboarding handbook
// from a resolved Gradus pyramid (X-12, D-38).
func RenderHandbook(view resolve.ResolvedView) string {
	var sb strings.Builder

	sb.WriteString("# Gradus Onboarding Handbook\n\n")
	fmt.Fprintf(&sb, "- **Schema:** `%s`\n", view.Schema)
	fmt.Fprintf(&sb, "- **View Hash:** `%s`\n", view.ViewHash)
	if view.LockHash != "" {
		fmt.Fprintf(&sb, "- **Lock Hash:** `%s`\n", view.LockHash)
	}
	sb.WriteString("\n## 1. Pyramid Levels & Charters\n\n")
	sb.WriteString("Rules flow from the highest (broadest) level to the lowest (most specific) level. Lower levels may tighten or specialize upper levels, never relax them.\n\n")

	for i, lvl := range view.Levels {
		fmt.Fprintf(&sb, "### %d. `%s`", i+1, lvl.ID)
		if lvl.Version != "" {
			fmt.Fprintf(&sb, " (`%s`)", lvl.Version)
		}
		sb.WriteString("\n\n")
		if lvl.Charter != nil {
			if lvl.Charter.Purpose != "" {
				fmt.Fprintf(&sb, "- **Purpose:** %s\n", lvl.Charter.Purpose)
			}
			if len(lvl.Charter.Includes) > 0 {
				fmt.Fprintf(&sb, "- **Includes:** %s\n", strings.Join(lvl.Charter.Includes, ", "))
			}
			if len(lvl.Charter.Excludes) > 0 {
				fmt.Fprintf(&sb, "- **Excludes:** %s\n", strings.Join(lvl.Charter.Excludes, ", "))
			}
			if len(lvl.Charter.AllowedSeverities) > 0 {
				fmt.Fprintf(&sb, "- **Allowed Severities:** %s\n", strings.Join(lvl.Charter.AllowedSeverities, ", "))
			}
		}
		sb.WriteString("\n")
	}

	sectionNum := 2
	if len(view.Overlays) > 0 {
		fmt.Fprintf(&sb, "## %d. Overlays\n\n", sectionNum)
		sectionNum++
		for _, ov := range view.Overlays {
			state := "inactive"
			if ov.Active {
				state = "active"
			}
			fmt.Fprintf(&sb, "- **`%s`** (strength: `%s`, state: `%s`)\n", ov.ID, ov.Strength, state)
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "## %d. Active Rules by Level\n\n", sectionNum)
	sectionNum++
	rulesByLevel := make(map[string][]resolve.ResolvedRule)
	var levelOrder []string
	seenLevel := make(map[string]bool)
	for _, lvl := range view.Levels {
		seenLevel[lvl.ID] = true
		levelOrder = append(levelOrder, lvl.ID)
	}
	for _, r := range view.Rules {
		if r.Waived {
			continue
		}
		if !seenLevel[r.Level] {
			seenLevel[r.Level] = true
			levelOrder = append(levelOrder, r.Level)
		}
		rulesByLevel[r.Level] = append(rulesByLevel[r.Level], r)
	}

	for _, lvlID := range levelOrder {
		rules := rulesByLevel[lvlID]
		if len(rules) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "### Level `%s`\n\n", lvlID)
		for _, r := range rules {
			fmt.Fprintf(&sb, "#### %s — %s\n", r.ID, r.Title)
			fmt.Fprintf(&sb, "- **Severity:** `%s` | **Enforcement:** `%s` | **Scope:** `%s` | **Source:** `%s`\n\n",
				r.Severity, r.Enforcement, strings.Join(r.Scope, ", "), r.Provenance.Source)
			fmt.Fprintf(&sb, "%s\n\n", r.Text)
			if r.Why != "" {
				fmt.Fprintf(&sb, "_Why:_ %s\n\n", r.Why)
			}
		}
	}

	if len(view.Obligations) > 0 {
		fmt.Fprintf(&sb, "## %d. Obligations & Bindings\n\n", sectionNum)
		sectionNum++
		for _, ob := range view.Obligations {
			fmt.Fprintf(&sb, "### %s (level `%s`, satisfied: `%t`)\n", ob.ID, ob.Level, ob.Satisfied)
			fmt.Fprintf(&sb, "- **Rule:** %s\n", ob.Rule)
			if len(ob.AllowedProviders) > 0 {
				fmt.Fprintf(&sb, "- **Allowed providers:** `%s`\n", strings.Join(ob.AllowedProviders, "`, `"))
			}
			for _, b := range ob.Bindings {
				fmt.Fprintf(&sb, "- **Binding:** provider `%s` (level `%s`)", b.Provider, b.Level)
				if len(b.Requires) > 0 {
					keys := make([]string, 0, len(b.Requires))
					for k := range b.Requires {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					var reqParts []string
					for _, k := range keys {
						reqParts = append(reqParts, fmt.Sprintf("%s=%s", k, b.Requires[k]))
					}
					fmt.Fprintf(&sb, " | requires: `%s`", strings.Join(reqParts, ", "))
				}
				if len(b.Scope) > 0 {
					fmt.Fprintf(&sb, " | scope: `%s`", strings.Join(b.Scope, "`, `"))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	fmt.Fprintf(&sb, "## %d. Exceptions & Break-Glass Procedure\n\n", sectionNum)
	sb.WriteString("- Request a time-limited exception: `gradusctl exception request --rule <ID> --reason \"...\" --expires YYYY-MM-DD`\n")
	sb.WriteString("- Approve a pending exception: `gradusctl exception approve <EXCEPTION_ID> --by <approver>`\n")
	sb.WriteString("- Record an emergency break-glass waiver (when permitted by the level): `gradusctl exception break-glass --rule <ID> --reason \"...\" --expires YYYY-MM-DD --post-review-due YYYY-MM-DD --by <responder>`\n")
	sb.WriteString("- List or expire waivers: `gradusctl exception list` / `gradusctl exception expire <EXCEPTION_ID>`\n")

	return sb.String()
}

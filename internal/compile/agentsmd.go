package compile

import (
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// PlanAgentsMD renders the managed section for AGENTS.md containing all active,
// non-waived, non-user rules in pyramid order (ADR 0002).
func PlanAgentsMD(existing string, view resolve.ResolvedView) []byte {
	var body strings.Builder
	body.WriteString(RenderPrecedenceHeader(view))
	body.WriteString("\n")

	for _, r := range view.Rules {
		if r.Level == "user" || r.Waived {
			continue
		}
		body.WriteString(RenderRuleMarkdown(r))
	}

	block := RenderManagedBlock(view.ViewHash, body.String())
	return []byte(ApplyManagedSection(existing, block))
}

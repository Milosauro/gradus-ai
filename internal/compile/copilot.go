package compile

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// PlanCopilot builds the in-memory file map for the GitHub Copilot target:
// .github/copilot-instructions.md, scoped .github/instructions/gradus-*.instructions.md,
// and embedded skills under .github/skills/.
func PlanCopilot(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)

	var globalBody strings.Builder
	globalBody.WriteString(RenderPrecedenceHeader(view))
	globalBody.WriteString("\n")

	var scopedIDs []string
	for _, r := range view.Rules {
		if r.Level == "user" || r.Waived {
			continue
		}
		if isGlobalScope(r.Scope) {
			globalBody.WriteString(RenderRuleMarkdown(r))
		} else {
			scopedIDs = append(scopedIDs, r.ID)
			var sb strings.Builder
			fmt.Fprintf(&sb, "---\napplyTo: %q\n---\n\n", strings.Join(r.Scope, ","))
			sb.WriteString(RenderRuleMarkdown(r))
			rel := filepath.ToSlash(filepath.Join(".github", "instructions", fmt.Sprintf("gradus-%s.instructions.md", strings.ToLower(r.ID))))
			files[rel] = []byte(sb.String())
		}
	}

	if len(scopedIDs) > 0 {
		fmt.Fprintf(&globalBody, "> **Scoped instructions:** %d path-scoped rule(s) (`%s`) are compiled into `.github/instructions/gradus-*.instructions.md`.\n",
			len(scopedIDs), strings.Join(scopedIDs, "`, `"))
	}

	copilotRel := filepath.ToSlash(filepath.Join(".github", "copilot-instructions.md"))
	existing, err := readOptionalFile(filepath.Join(rootDir, filepath.FromSlash(copilotRel)))
	if err != nil {
		return nil, err
	}
	block := RenderManagedBlock(view.ViewHash, globalBody.String())
	files[copilotRel] = []byte(ApplyManagedSection(existing, block))

	skillFiles, err := PlanEmbeddedSkills(skillsFS, filepath.ToSlash(filepath.Join(".github", "skills")))
	if err != nil {
		return nil, err
	}
	for k, v := range skillFiles {
		files[k] = v
	}

	return files, nil
}

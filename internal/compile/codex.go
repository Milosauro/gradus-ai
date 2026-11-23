package compile

import (
	"io/fs"
	"path/filepath"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// PlanCodex builds the in-memory file map for the codex target (AGENTS.md + .agents/skills/).
func PlanCodex(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)

	existing, err := readOptionalFile(filepath.Join(rootDir, "AGENTS.md"))
	if err != nil {
		return nil, err
	}
	files["AGENTS.md"] = PlanAgentsMD(existing, view)

	skillFiles, err := PlanEmbeddedSkills(skillsFS, filepath.ToSlash(filepath.Join(".agents", "skills")))
	if err != nil {
		return nil, err
	}
	for k, v := range skillFiles {
		files[k] = v
	}

	return files, nil
}

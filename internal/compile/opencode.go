package compile

import (
	"io/fs"
	"path/filepath"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// PlanOpenCode builds the in-memory file map for the opencode target (AGENTS.md + .opencode/skills/).
func PlanOpenCode(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)

	existing, err := readOptionalFile(filepath.Join(rootDir, "AGENTS.md"))
	if err != nil {
		return nil, err
	}
	files["AGENTS.md"] = PlanAgentsMD(existing, view)

	skillFiles, err := PlanEmbeddedSkills(skillsFS, filepath.ToSlash(filepath.Join(".opencode", "skills")))
	if err != nil {
		return nil, err
	}
	for k, v := range skillFiles {
		files[k] = v
	}

	return files, nil
}

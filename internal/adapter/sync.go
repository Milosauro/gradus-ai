package adapter

import (
	"fmt"
	"os"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// SyncResult describes the outcome of synchronizing a single adapter's target_file.
type SyncResult struct {
	Provider   string `json:"provider"`
	TargetFile string `json:"target_file"`
	Changed    bool   `json:"changed"`
}

// RenderConstitution renders the managed-section content for an adapter target_file,
// preserving all surrounding human-written prose byte-for-byte.
func RenderConstitution(existing string, view resolve.ResolvedView) []byte {
	return compile.PlanAgentsMD(existing, view)
}

// Sync writes the resolved Gradus view into m.TargetFile inside a managed section.
func Sync(rootDir string, m Manifest, view resolve.ResolvedView) (SyncResult, error) {
	res := SyncResult{
		Provider:   m.ID,
		TargetFile: m.TargetFile,
	}

	absPath, err := ResolveTargetPath(rootDir, m.TargetFile)
	if err != nil {
		return res, err
	}

	var existing string
	if raw, err := os.ReadFile(absPath); err == nil {
		existing = string(raw)
	} else if !os.IsNotExist(err) {
		return res, fmt.Errorf("failed to read %s: %w", absPath, err)
	}

	content := RenderConstitution(existing, view)
	changed, err := compile.WriteIfChanged(absPath, content)
	if err != nil {
		return res, err
	}
	res.Changed = changed
	return res, nil
}

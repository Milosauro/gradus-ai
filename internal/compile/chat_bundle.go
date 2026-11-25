package compile

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

// ChatBundleManifest describes the contents of a portable chat-agent skill bundle ZIP (§20).
type ChatBundleManifest struct {
	Schema    string                 `json:"schema"`
	ViewHash  string                 `json:"view_hash"`
	LockHash  string                 `json:"lock_hash,omitempty"`
	Levels    []resolve.LevelSummary `json:"levels"`
	RuleCount int                    `json:"rule_count"`
	Skills    []string               `json:"skills"`
}

// ChatBundleResult summarizes the written chat-agent skill bundle ZIP.
type ChatBundleResult struct {
	Path      string   `json:"path"`
	ViewHash  string   `json:"view_hash"`
	RuleCount int      `json:"rule_count"`
	Skills    []string `json:"skills"`
	Entries   int      `json:"entries"`
}

// WriteChatBundleZIP packages shared rules (excluding personal user-level rules) and embedded
// Gradus skills into a byte-reproducible ZIP archive for chat products (§20).
func WriteChatBundleZIP(outPath string, view resolve.ResolvedView, skillsFS fs.FS) (ChatBundleResult, error) {
	var res ChatBundleResult

	// Filter out personal user-level rules and level summaries.
	var sharedLevels []resolve.LevelSummary
	for _, l := range view.Levels {
		if l.ID != "user" {
			sharedLevels = append(sharedLevels, l)
		}
	}
	var sharedRules []resolve.ResolvedRule
	for _, r := range view.Rules {
		if r.Waived || r.Level == "user" {
			continue
		}
		sharedRules = append(sharedRules, r)
	}

	sharedView := view
	sharedView.Levels = sharedLevels
	sharedView.Rules = sharedRules

	var rulesMD strings.Builder
	rulesMD.WriteString(RenderPrecedenceHeader(sharedView))
	rulesMD.WriteString("\n")
	for _, r := range sharedRules {
		rulesMD.WriteString(RenderRuleMarkdown(r))
	}

	entries := make(map[string][]byte)
	entries["RULES.md"] = []byte(rulesMD.String())

	skillFiles, err := PlanEmbeddedSkills(skillsFS, "skills")
	if err != nil {
		return res, err
	}
	skillSet := make(map[string]bool)
	for path, data := range skillFiles {
		entries[path] = data
		if strings.HasSuffix(path, "/SKILL.md") {
			skillSet[filepath.Base(filepath.Dir(path))] = true
		}
	}
	skillsList := make([]string, 0, len(skillSet))
	for name := range skillSet {
		skillsList = append(skillsList, name)
	}
	sort.Strings(skillsList)

	manifest := ChatBundleManifest{
		Schema:    schema.SchemaVersion,
		ViewHash:  view.ViewHash,
		LockHash:  view.LockHash,
		Levels:    sharedLevels,
		RuleCount: len(sharedRules),
		Skills:    skillsList,
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return res, err
	}
	entries["manifest.json"] = append(manifestBytes, '\n')

	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fixedTime := time.Unix(0, 0).UTC()
	for _, k := range keys {
		hdr := &zip.FileHeader{
			Name:     k,
			Method:   zip.Deflate,
			Modified: fixedTime,
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return res, err
		}
		if _, err := w.Write(entries[k]); err != nil {
			return res, err
		}
	}
	if err := zw.Close(); err != nil {
		return res, err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return res, err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return res, err
	}

	return ChatBundleResult{
		Path:      outPath,
		ViewHash:  view.ViewHash,
		RuleCount: len(sharedRules),
		Skills:    skillsList,
		Entries:   len(keys),
	}, nil
}

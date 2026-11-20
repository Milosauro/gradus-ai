package skills_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/skills"
)

var canonicalSkills = []string{
	"gradus-onboard",
	"gradus-author",
	"gradus-review",
	"gradus-learn",
	"gradus-reconcile",
	"gradus-conflicts",
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type evalScenario struct {
	ID                     string   `yaml:"id"`
	Skill                  string   `yaml:"skill"`
	Title                  string   `yaml:"title"`
	Prompt                 string   `yaml:"prompt"`
	BaselineFailures       []string `yaml:"baseline_failures"`
	BaselineRationalizations []string `yaml:"baseline_rationalizations"`
	RequiredBehaviors      []string `yaml:"required_behaviors"`
	ForbiddenBehaviors     []string `yaml:"forbidden_behaviors"`
}

func TestEmbeddedSkills(t *testing.T) {
	if _, err := fs.Stat(skills.FS, "gradus-placeholder/SKILL.md"); err == nil {
		t.Fatalf("gradus-placeholder must be removed from embedded skills.FS")
	}

	requiredSnippets := map[string][]string{
		"gradus-onboard":   {"approval", "Redact", "gradusctl init"},
		"gradus-author":    {"locked: true", "Rejected Rationalizations", "gradusctl resolve"},
		"gradus-review":    {"confidence", "provenance", "gradusctl evidence write"},
		"gradus-learn":     {"Redact", "gradusctl lessons add", "explicit approval"},
		"gradus-reconcile": {"gradusctl reconcile", "locked: true", "gradusctl verify"},
		"gradus-conflicts": {"gradusctl conflicts --candidates", "confidence", "confirm"},
	}

	for _, name := range canonicalSkills {
		raw, err := fs.ReadFile(skills.FS, name+"/SKILL.md")
		if err != nil {
			t.Fatalf("missing embedded skill %s/SKILL.md: %v", name, err)
		}
		text := strings.ReplaceAll(string(raw), "\r\n", "\n")
		if !strings.HasPrefix(text, "---\n") {
			t.Fatalf("%s/SKILL.md missing YAML frontmatter", name)
		}
		parts := strings.SplitN(text[4:], "\n---\n", 2)
		if len(parts) != 2 {
			t.Fatalf("%s/SKILL.md malformed YAML frontmatter", name)
		}
		var fm skillFrontmatter
		if err := yaml.Unmarshal([]byte(parts[0]), &fm); err != nil {
			t.Fatalf("%s/SKILL.md invalid YAML frontmatter: %v", name, err)
		}
		if fm.Name != name {
			t.Fatalf("%s/SKILL.md frontmatter name %q != %q", name, fm.Name, name)
		}
		if strings.TrimSpace(fm.Description) == "" {
			t.Fatalf("%s/SKILL.md frontmatter description cannot be empty", name)
		}

		for _, snip := range requiredSnippets[name] {
			if !strings.Contains(strings.ToLower(parts[1]), strings.ToLower(snip)) {
				t.Fatalf("%s/SKILL.md missing required guardrail/CLI snippet %q", name, snip)
			}
		}
	}
}

func TestSkillEvalScenarios(t *testing.T) {
	evalDir := filepath.Join("..", "evals", "skills")
	entries, err := os.ReadDir(evalDir)
	if err != nil {
		t.Fatalf("failed to read %s: %v", evalDir, err)
	}

	coveredSkills := make(map[string]bool)
	hasInjection := false

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(evalDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var sc evalScenario
		if err := yaml.Unmarshal(raw, &sc); err != nil {
			t.Fatalf("invalid eval scenario %s: %v", e.Name(), err)
		}
		if sc.ID == "" || sc.Skill == "" || sc.Title == "" || strings.TrimSpace(sc.Prompt) == "" {
			t.Fatalf("incomplete eval scenario metadata in %s: %+v", e.Name(), sc)
		}
		if len(sc.BaselineFailures) == 0 || len(sc.BaselineRationalizations) == 0 || len(sc.RequiredBehaviors) == 0 || len(sc.ForbiddenBehaviors) == 0 {
			t.Fatalf("eval scenario %s missing baseline or behavior assertions", e.Name())
		}
		coveredSkills[sc.Skill] = true
		if strings.Contains(e.Name(), "injection") {
			hasInjection = true
		}
	}

	for _, name := range canonicalSkills {
		if !coveredSkills[name] {
			t.Fatalf("missing eval scenario under evals/skills/ for %s", name)
		}
	}
	if !hasInjection {
		t.Fatalf("missing prompt-injection eval scenario under evals/skills/")
	}
}

func TestEmbeddedSkillsAndEvals(t *testing.T) {
	tmp := t.TempDir()
	view := resolve.ResolvedView{
		ViewHash: "sha256:skills1",
		Levels:   []resolve.LevelSummary{{ID: "project"}},
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-01",
				Title:       "Canonical English",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "Write repository files in English.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
		},
	}

	res, err := compile.ApplyCompile(tmp, view, []string{"claude-code", "antigravity"}, skills.FS)
	if err != nil {
		t.Fatalf("ApplyCompile with embedded skills.FS failed: %v", err)
	}
	if len(res.ChangedFiles) == 0 {
		t.Fatalf("expected compiled files")
	}

	for _, name := range canonicalSkills {
		for _, base := range []string{".claude/skills", ".agents/skills"} {
			p := filepath.Join(tmp, base, name, "SKILL.md")
			if _, err := os.Stat(p); err != nil {
				t.Fatalf("expected extracted skill %s: %v", p, err)
			}
		}
	}
	for _, base := range []string{".claude/skills", ".agents/skills"} {
		if _, err := os.Stat(filepath.Join(tmp, base, "gradus-placeholder")); !os.IsNotExist(err) {
			t.Fatalf("expected no gradus-placeholder in %s", base)
		}
	}
}

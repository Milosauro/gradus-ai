package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

// FindingRecord persists a semantic conflict judgment under .gradus/findings/*.yaml.
type FindingRecord struct {
	Schema    string `yaml:"schema" json:"schema"`
	ID        string `yaml:"id" json:"id"`
	RuleA     string `yaml:"rule_a" json:"rule_a"`
	RuleB     string `yaml:"rule_b" json:"rule_b"`
	HashA     string `yaml:"hash_a,omitempty" json:"hash_a,omitempty"`
	HashB     string `yaml:"hash_b,omitempty" json:"hash_b,omitempty"`
	Status    string `yaml:"status" json:"status"` // suspected | confirmed | dismissed
	Message   string `yaml:"message" json:"message"`
	UpdatedAt string `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// CandidatePair describes two rules from distinct levels/overlays that may conflict semantically.
type CandidatePair struct {
	RuleA  string `json:"rule_a"`
	RuleB  string `json:"rule_b"`
	LevelA string `json:"level_a"`
	LevelB string `json:"level_b"`
	HashA  string `json:"hash_a"`
	HashB  string `json:"hash_b"`
	Reason string `json:"reason"`
}

// CanonicalPair orders two rule IDs and their content hashes deterministically.
func CanonicalPair(ruleA, hashA, ruleB, hashB string) (string, string, string, string) {
	if ruleA <= ruleB {
		return ruleA, hashA, ruleB, hashB
	}
	return ruleB, hashB, ruleA, hashA
}

// ListFindings reads all .gradus/findings/*.yaml records sorted by ID.
func ListFindings(rootDir string) ([]FindingRecord, error) {
	dir := filepath.Join(rootDir, ".gradus", "findings")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []FindingRecord{}, nil
		}
		return nil, err
	}

	var out []FindingRecord
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var rec FindingRecord
		if err := yaml.Unmarshal(raw, &rec); err != nil {
			return nil, fmt.Errorf("invalid finding %s: %w", e.Name(), err)
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// SaveFinding writes or updates a finding record under .gradus/findings/<id>.yaml.
func SaveFinding(rootDir string, rec FindingRecord, now time.Time) (FindingRecord, string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if rec.Status != "suspected" && rec.Status != "confirmed" && rec.Status != "dismissed" {
		return rec, "", fmt.Errorf("invalid finding status %q (expected suspected, confirmed, or dismissed)", rec.Status)
	}
	if strings.TrimSpace(rec.RuleA) == "" || strings.TrimSpace(rec.RuleB) == "" {
		return rec, "", fmt.Errorf("finding requires both rule_a and rule_b")
	}

	rec.RuleA, rec.HashA, rec.RuleB, rec.HashB = CanonicalPair(rec.RuleA, rec.HashA, rec.RuleB, rec.HashB)
	rec.Schema = schema.SchemaVersion
	if strings.TrimSpace(rec.ID) == "" {
		rec.ID = fmt.Sprintf("F-%s-%s", sanitizeIDPart(rec.RuleA), sanitizeIDPart(rec.RuleB))
	}
	rec.UpdatedAt = now.UTC().Format(time.RFC3339)

	dir := filepath.Join(rootDir, ".gradus", "findings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return rec, "", fmt.Errorf("failed to create findings dir: %w", err)
	}

	data, err := yaml.Marshal(rec)
	if err != nil {
		return rec, "", err
	}
	dest := filepath.Join(dir, rec.ID+".yaml")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return rec, "", err
	}
	return rec, filepath.ToSlash(filepath.Join(".gradus", "findings", rec.ID+".yaml")), nil
}

// FindCandidates identifies active, non-waived rule pairs across distinct levels/sources
// with overlapping scopes and shared keywords, skipping pairs already confirmed/dismissed
// with identical content hashes (§10.1).
func FindCandidates(view resolve.ResolvedView, existing []FindingRecord) []CandidatePair {
	judged := make(map[string]FindingRecord)
	for _, f := range existing {
		if f.Status != "confirmed" && f.Status != "dismissed" {
			continue
		}
		ra, ha, rb, hb := CanonicalPair(f.RuleA, f.HashA, f.RuleB, f.HashB)
		judged[ra+"|"+rb] = FindingRecord{RuleA: ra, HashA: ha, RuleB: rb, HashB: hb, Status: f.Status}
	}

	var active []resolve.ResolvedRule
	for _, r := range view.Rules {
		if !r.Waived {
			active = append(active, r)
		}
	}

	var out []CandidatePair
	for i := 0; i < len(active); i++ {
		for j := i + 1; j < len(active); j++ {
			r1 := active[i]
			r2 := active[j]
			if r1.Level == r2.Level && r1.Provenance.Source == r2.Provenance.Source {
				continue
			}
			if !scopesOverlap(r1.Scope, r2.Scope) {
				continue
			}
			shared := sharedKeywords(r1, r2)
			if len(shared) == 0 {
				continue
			}

			ra, ha, rb, hb := CanonicalPair(r1.ID, r1.Provenance.Hash, r2.ID, r2.Provenance.Hash)
			if f, ok := judged[ra+"|"+rb]; ok && f.HashA == ha && f.HashB == hb {
				continue
			}

			lvlA, lvlB := r1.Level, r2.Level
			if ra != r1.ID {
				lvlA, lvlB = r2.Level, r1.Level
			}

			out = append(out, CandidatePair{
				RuleA:  ra,
				RuleB:  rb,
				LevelA: lvlA,
				LevelB: lvlB,
				HashA:  ha,
				HashB:  hb,
				Reason: "overlapping scope and shared terms: " + strings.Join(shared, ", "),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].RuleA != out[j].RuleA {
			return out[i].RuleA < out[j].RuleA
		}
		return out[i].RuleB < out[j].RuleB
	})
	if out == nil {
		return []CandidatePair{}
	}
	return out
}

func sanitizeIDPart(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

func scopesOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	for _, pa := range a {
		if pa == "**/*" || pa == "**" {
			return true
		}
		for _, pb := range b {
			if pb == "**/*" || pb == "**" || pa == pb {
				return true
			}
			if ok, _ := doublestar.Match(pa, strings.TrimSuffix(pb, "/**")); ok {
				return true
			}
			if ok, _ := doublestar.Match(pb, strings.TrimSuffix(pa, "/**")); ok {
				return true
			}
		}
	}
	return false
}

var stopwords = map[string]bool{
	"must": true, "should": true, "never": true, "always": true, "with": true,
	"from": true, "that": true, "this": true, "when": true, "into": true,
	"only": true, "without": true, "using": true, "files": true, "file": true,
	"rule": true, "rules": true, "code": true, "source": true, "explicit": true,
	"before": true, "after": true, "changes": true, "change": true, "application": true,
	"project": true, "repository": true, "repo": true, "user": true, "local": true,
	"running": true, "require": true, "required": true, "confirm": true, "confirmation": true,
	"approval": true, "human": true, "agent": true, "agents": true, "written": true,
	"write": true, "edit": true, "update": true, "check": true, "verify": true,
	"every": true, "each": true, "also": true, "then": true, "where": true,
	"keep": true, "ensure": true, "follow": true, "avoid": true, "prefer": true,
}

func sharedKeywords(r1, r2 resolve.ResolvedRule) []string {
	k1 := extractKeywords(r1.Title + " " + r1.Text)
	k2 := extractKeywords(r2.Title + " " + r2.Text)

	var shared []string
	for w := range k1 {
		if k2[w] {
			shared = append(shared, w)
		}
	}
	sort.Strings(shared)
	if len(shared) > 5 {
		shared = shared[:5]
	}
	return shared
}

func extractKeywords(s string) map[string]bool {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	set := make(map[string]bool)
	for _, w := range words {
		if len(w) >= 4 && !stopwords[w] {
			set[w] = true
		}
	}
	return set
}

package schema

import (
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

var ruleFrontmatterSpec = FieldSpec{
	Allowed: map[string]FieldSpec{
		"id":           {},
		"title":        {},
		"severity":     {},
		"enforcement":  {},
		"scope":        {},
		"overrides":    {},
		"overridable":  {},
		"applies_when": {},
		"source":       {},
		"controls":     {},
		"owner":        {},
		"status":       {},
		"review_by":    {},
		"expires":      {},
		"checker": {
			Allowed: map[string]FieldSpec{
				"name":    {},
				"config":  {},
				"pattern": {},
			},
		},
		"guard": {
			Allowed: map[string]FieldSpec{
				"action":   {},
				"pattern":  {},
				"decision": {},
			},
		},
		"tags":    {},
		"version": {},
	},
}

// ParseRuleFile reads a Markdown rule file and derives its level strictly from derivedLevel (ADR 0004).
func ParseRuleFile(path, derivedLevel, provenanceSource string) (ParseResult[Rule], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ParseResult[Rule]{}, fmt.Errorf("failed to read rule file %s: %w", path, err)
	}
	return ParseRuleBytes(data, path, derivedLevel, provenanceSource)
}

// ParseRuleBytes parses rule Markdown bytes and derives its level strictly from derivedLevel (ADR 0004).
func ParseRuleBytes(data []byte, filePath, derivedLevel, provenanceSource string) (ParseResult[Rule], error) {
	var res ParseResult[Rule]

	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return res, fmt.Errorf("%s: %w", filePath, err)
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &node); err != nil {
		return res, fmt.Errorf("%s: invalid rule frontmatter YAML: %w", filePath, err)
	}
	res.Notes = CollectUnknownKeys(&node, ruleFrontmatterSpec, "", false)

	if err := yaml.Unmarshal([]byte(frontmatter), &res.Value); err != nil {
		return res, fmt.Errorf("%s: failed to decode rule frontmatter: %w", filePath, err)
	}

	// Apply defaults (§9.4).
	if res.Value.Enforcement == "" {
		res.Value.Enforcement = "prompt"
	}
	if len(res.Value.Scope) == 0 {
		res.Value.Scope = []string{"**/*"}
	}
	if res.Value.Status == "" {
		res.Value.Status = "active"
	}

	// Derive level strictly from container (ADR 0004).
	res.Value.Level = derivedLevel
	res.Value.FilePath = filePath
	res.Value.ProvenanceSource = provenanceSource

	parseRuleBody(body, &res.Value)

	if err := ValidateRule(&res.Value); err != nil {
		return res, fmt.Errorf("%s: %w", filePath, err)
	}
	return res, nil
}

func splitFrontmatter(raw string) (string, string, error) {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("missing YAML frontmatter opening '---'")
	}
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		return "", "", fmt.Errorf("missing YAML frontmatter closing '---'")
	}
	fm := strings.Join(lines[1:endIdx], "\n")
	body := strings.Join(lines[endIdx+1:], "\n")
	return fm, body, nil
}

func parseRuleBody(body string, r *Rule) {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")

	var statementLines []string
	var whyLines []string
	section := "statement"
	subSection := ""
	inFence := false
	var fenceLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if !inFence {
				inFence = true
				fenceLines = nil
				continue
			}
			inFence = false
			block := strings.TrimSpace(strings.Join(fenceLines, "\n"))
			if section == "examples" && block != "" {
				switch subSection {
				case "compliant":
					r.Examples.Compliant = append(r.Examples.Compliant, block)
				case "non-compliant":
					r.Examples.NonCompliant = append(r.Examples.NonCompliant, block)
				}
			}
			fenceLines = nil
			continue
		}

		if inFence {
			fenceLines = append(fenceLines, line)
			continue
		}

		if strings.HasPrefix(trimmed, "## ") {
			heading := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			subSection = ""
			switch heading {
			case "why":
				section = "why"
			case "rationalizations":
				section = "rationalizations"
			case "examples":
				section = "examples"
			default:
				section = "other"
			}
			continue
		}

		if strings.HasPrefix(trimmed, "### ") && section == "examples" {
			subHeading := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))
			switch subHeading {
			case "compliant":
				subSection = "compliant"
			case "non-compliant":
				subSection = "non-compliant"
			default:
				subSection = ""
			}
			continue
		}

		switch section {
		case "statement":
			if trimmed == "" {
				if len(statementLines) > 0 {
					section = "post-statement"
				}
				continue
			}
			statementLines = append(statementLines, trimmed)
		case "why":
			whyLines = append(whyLines, line)
		case "rationalizations":
			if strings.HasPrefix(trimmed, "- ") {
				r.Rationalizations = append(r.Rationalizations, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			}
		}
	}

	r.Statement = strings.TrimSpace(strings.Join(statementLines, " "))
	r.Why = strings.TrimSpace(strings.Join(whyLines, "\n"))
}

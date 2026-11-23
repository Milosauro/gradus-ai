package toolchain

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// Finding describes a blocking toolchain violation (missing required or forbidden present).
type Finding struct {
	Kind    string `json:"kind"` // missing_required | forbidden_present
	Type    string `json:"type"`
	ID      string `json:"id"`
	Level   string `json:"level,omitempty"`
	Message string `json:"message"`
}

// Report describes the result of `gradusctl toolchain check`.
type Report struct {
	Verdict  string    `json:"verdict"` // pass | fail
	Checked  int       `json:"checked"`
	Findings []Finding `json:"findings"`
	Warnings []string  `json:"warnings,omitempty"`
}

type inventory struct {
	skills  map[string]bool
	mcps    map[string]bool
	plugins map[string]bool
	warns   []string
}

// Check inspects the local workspace and PATH against tc without network I/O (NFR2).
func Check(rootDir string, tc *resolve.ResolvedToolchain) Report {
	rep := Report{
		Verdict:  "pass",
		Findings: []Finding{},
	}
	if tc == nil {
		return rep
	}

	inv := discoverInventory(rootDir)
	rep.Warnings = append(rep.Warnings, inv.warns...)

	isPresent := func(itemType, id string) bool {
		switch itemType {
		case "skill":
			return inv.skills[id]
		case "tool":
			_, err := exec.LookPath(id)
			return err == nil
		case "mcp":
			return inv.mcps[id]
		case "plugin":
			return inv.plugins[id]
		default:
			return false
		}
	}

	for _, req := range tc.Required {
		rep.Checked++
		if !isPresent(req.Type, req.ID) {
			rep.Verdict = "fail"
			rep.Findings = append(rep.Findings, Finding{
				Kind:    "missing_required",
				Type:    req.Type,
				ID:      req.ID,
				Level:   req.Level,
				Message: fmt.Sprintf("required %s %q (level %s) is not installed or configured", req.Type, req.ID, req.Level),
			})
		}
	}

	for _, rec := range tc.Recommended {
		rep.Checked++
		if !isPresent(rec.Type, rec.ID) {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("recommended %s %q (level %s) is not installed or configured", rec.Type, rec.ID, rec.Level))
		}
	}

	for _, forb := range tc.Forbidden {
		rep.Checked++
		if forb.Type == "tool" {
			if !strings.ContainsAny(forb.ID, "*?[") {
				if _, err := exec.LookPath(forb.ID); err == nil {
					rep.Verdict = "fail"
					rep.Findings = append(rep.Findings, Finding{
						Kind:    "forbidden_present",
						Type:    forb.Type,
						ID:      forb.ID,
						Level:   forb.Level,
						Message: fmt.Sprintf("forbidden tool %q (level %s) is present on PATH", forb.ID, forb.Level),
					})
				}
			}
			continue
		}

		var candidates []string
		switch forb.Type {
		case "skill":
			candidates = sortedKeys(inv.skills)
		case "mcp":
			candidates = sortedKeys(inv.mcps)
		case "plugin":
			candidates = sortedKeys(inv.plugins)
		}
		for _, cand := range candidates {
			if matched, err := doublestar.Match(forb.ID, cand); err == nil && matched {
				rep.Verdict = "fail"
				rep.Findings = append(rep.Findings, Finding{
					Kind:    "forbidden_present",
					Type:    forb.Type,
					ID:      cand,
					Level:   forb.Level,
					Message: fmt.Sprintf("forbidden %s %q matches pattern %q (level %s)", forb.Type, cand, forb.ID, forb.Level),
				})
			}
		}
	}

	return rep
}

func discoverInventory(rootDir string) inventory {
	inv := inventory{
		skills:  make(map[string]bool),
		mcps:    make(map[string]bool),
		plugins: make(map[string]bool),
	}

	for _, skillsRoot := range []string{
		filepath.Join(rootDir, ".claude", "skills"),
		filepath.Join(rootDir, ".agents", "skills"),
	} {
		entries, err := os.ReadDir(skillsRoot)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if _, statErr := os.Stat(filepath.Join(skillsRoot, e.Name(), "SKILL.md")); statErr == nil {
				inv.skills[e.Name()] = true
			}
		}
	}

	for _, mcpFile := range []string{
		filepath.Join(rootDir, ".mcp.json"),
		filepath.Join(rootDir, ".claude", "settings.json"),
		filepath.Join(rootDir, ".agents", "mcp.json"),
	} {
		raw, err := os.ReadFile(mcpFile)
		if err != nil {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			inv.warns = append(inv.warns, fmt.Sprintf("could not parse %s: %v", mcpFile, err))
			continue
		}
		if servers, ok := doc["mcpServers"].(map[string]any); ok {
			for k := range servers {
				inv.mcps[k] = true
			}
		}
		if filepath.Base(mcpFile) == "settings.json" {
			if plugins, ok := doc["enabledPlugins"].(map[string]any); ok {
				for k, v := range plugins {
					if enabled, ok := v.(bool); ok && enabled {
						inv.plugins[k] = true
					}
				}
			}
		}
	}

	return inv
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

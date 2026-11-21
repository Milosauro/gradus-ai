package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocalPyramid holds the parsed gradus.yaml, loaded level configs, rules, and diagnostic notes.
type LocalPyramid struct {
	Config PyramidConfig
	Levels map[string]LevelConfig
	Rules  []Rule
	Notes  []string
}

// LoadLocalPyramid reads gradus.yaml and loads rules from all local level paths.
// Home-directory paths (~/) are skipped when the CI environment variable is set (§9.2).
func LoadLocalPyramid(rootDir string) (LocalPyramid, error) {
	var out LocalPyramid

	cfgPath := filepath.Join(rootDir, "gradus.yaml")
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return out, fmt.Errorf("failed to read %s: %w", cfgPath, err)
	}

	cfgRes, err := ParsePyramidConfig(cfgBytes)
	if err != nil {
		return out, err
	}
	out.Config = cfgRes.Value
	out.Notes = append(out.Notes, cfgRes.Notes...)

	inCI := os.Getenv("CI") != ""

	for _, lvl := range out.Config.Levels {
		if lvl.Source.Path == "" {
			out.Notes = append(out.Notes, fmt.Sprintf("skipping non-local level %q in local-only load", lvl.ID))
			continue
		}

		levelDir, skipNote, err := resolveLevelPath(rootDir, lvl.ID, lvl.Source.Path, inCI)
		if err != nil {
			return out, err
		}
		if skipNote != "" {
			out.Notes = append(out.Notes, skipNote)
			continue
		}

		levelYAMLPath := filepath.Join(levelDir, "level.yaml")
		if lvlBytes, err := os.ReadFile(levelYAMLPath); err == nil {
			lvlRes, parseErr := ParseLevelConfig(lvlBytes)
			if parseErr != nil {
				return out, fmt.Errorf("level %q (%s): %w", lvl.ID, levelYAMLPath, parseErr)
			}
			out.Notes = append(out.Notes, lvlRes.Notes...)
			if out.Levels == nil {
				out.Levels = make(map[string]LevelConfig)
			}
			out.Levels[lvl.ID] = lvlRes.Value
		} else if !os.IsNotExist(err) {
			return out, fmt.Errorf("failed to read %s: %w", levelYAMLPath, err)
		}

		rulesDir := filepath.Join(levelDir, "rules")
		entries, err := os.ReadDir(rulesDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return out, fmt.Errorf("failed to read rules directory %s: %w", rulesDir, err)
		}

		var mdFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				mdFiles = append(mdFiles, entry.Name())
			}
		}
		sort.Strings(mdFiles)

		for _, name := range mdFiles {
			rulePath := filepath.Join(rulesDir, name)
			ruleRes, err := ParseRuleFile(rulePath, lvl.ID, lvl.Source.Path)
			if err != nil {
				return out, err
			}
			out.Rules = append(out.Rules, ruleRes.Value)
			out.Notes = append(out.Notes, ruleRes.Notes...)
		}
	}

	return out, nil
}

func resolveLevelPath(rootDir, levelID, rawPath string, inCI bool) (string, string, error) {
	if rawPath == "~" || strings.HasPrefix(rawPath, "~/") || strings.HasPrefix(rawPath, `~\`) {
		if inCI {
			return "", fmt.Sprintf("skipping home-directory level %q (%s) in CI", levelID, rawPath), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("level %q: failed to resolve home directory: %w", levelID, err)
		}
		trimmed := strings.TrimPrefix(strings.TrimPrefix(rawPath, "~/"), `~\`)
		if rawPath == "~" {
			trimmed = ""
		}
		resolved := filepath.Join(home, filepath.FromSlash(trimmed))
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			return "", fmt.Sprintf("home-directory level %q (%s) does not exist yet", levelID, resolved), nil
		}
		return resolved, "", nil
	}

	if filepath.IsAbs(rawPath) {
		return rawPath, "", nil
	}
	return filepath.Join(rootDir, filepath.FromSlash(rawPath)), "", nil
}

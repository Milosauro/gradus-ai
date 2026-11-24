package adapter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

var adapterIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Manifest describes a declarative Gradus provider adapter (§19).
type Manifest struct {
	Schema          string   `yaml:"schema" json:"schema"`
	ID              string   `yaml:"id" json:"id"`
	Name            string   `yaml:"name,omitempty" json:"name,omitempty"`
	Description     string   `yaml:"description,omitempty" json:"description,omitempty"`
	TargetFile      string   `yaml:"target_file" json:"target_file"`
	VersionCommand  []string `yaml:"version_command,omitempty" json:"version_command,omitempty"`
	EvidenceCommand []string `yaml:"evidence_command,omitempty" json:"evidence_command,omitempty"`
	Source          string   `yaml:"-" json:"source"` // official | custom
}

// LoadRegistry loads embedded official adapters and optional custom manifests from
// .gradus/adapters/*.yaml and .gradus/adapters/*/adapter.yaml.
func LoadRegistry(rootDir string, officialFS fs.FS) (map[string]Manifest, error) {
	reg := make(map[string]Manifest)

	if officialFS != nil {
		matches, err := fs.Glob(officialFS, "*/adapter.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to glob official adapters: %w", err)
		}
		sort.Strings(matches)
		for _, p := range matches {
			data, err := fs.ReadFile(officialFS, p)
			if err != nil {
				return nil, fmt.Errorf("failed to read official adapter %s: %w", p, err)
			}
			m, err := parseAndValidateManifest(data, "official", p, rootDir)
			if err != nil {
				return nil, err
			}
			reg[m.ID] = m
		}
	}

	customDir := filepath.Join(rootDir, ".gradus", "adapters")
	entries, err := os.ReadDir(customDir)
	if err != nil {
		if os.IsNotExist(err) {
			return reg, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", customDir, err)
	}

	var customPaths []string
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(customDir, e.Name(), "adapter.yaml")
			if _, statErr := os.Stat(p); statErr == nil {
				customPaths = append(customPaths, p)
			}
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
			customPaths = append(customPaths, filepath.Join(customDir, e.Name()))
		}
	}
	sort.Strings(customPaths)

	for _, p := range customPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("failed to read custom adapter %s: %w", p, err)
		}
		m, err := parseAndValidateManifest(data, "custom", p, rootDir)
		if err != nil {
			return nil, err
		}
		reg[m.ID] = m
	}

	return reg, nil
}

// ResolveTargetPath validates and resolves an adapter target_file strictly inside rootDir (NFR2).
func ResolveTargetPath(rootDir, targetFile string) (string, error) {
	trimmed := strings.TrimSpace(targetFile)
	if trimmed == "" {
		return "", fmt.Errorf("adapter target_file is empty")
	}
	slash := filepath.ToSlash(trimmed)
	if filepath.IsAbs(trimmed) || strings.HasPrefix(slash, "/") {
		return "", fmt.Errorf("adapter target_file %q must be a relative path", targetFile)
	}
	for _, seg := range strings.Split(slash, "/") {
		if seg == ".." {
			return "", fmt.Errorf("adapter target_file %q cannot contain '..' traversal", targetFile)
		}
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(slash)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return "", fmt.Errorf("adapter target_file %q escapes repository root", targetFile)
	}
	return absTarget, nil
}

func parseAndValidateManifest(data []byte, source, label, rootDir string) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("adapter %s: invalid YAML: %w", label, err)
	}
	if m.Schema != schema.SchemaVersion {
		return m, fmt.Errorf("adapter %s: unsupported schema %q (expected %q)", label, m.Schema, schema.SchemaVersion)
	}
	if !adapterIDPattern.MatchString(m.ID) {
		return m, fmt.Errorf("adapter %s: id %q must match ^[a-z][a-z0-9-]*$", label, m.ID)
	}
	if _, err := ResolveTargetPath(rootDir, m.TargetFile); err != nil {
		return m, fmt.Errorf("adapter %s (%s): %w", label, m.ID, err)
	}
	m.TargetFile = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(m.TargetFile))))
	m.Source = source
	return m, nil
}

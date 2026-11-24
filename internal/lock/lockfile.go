package lock

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

// Entry represents a single pinned level or overlay in gradus.lock (§9.6).
type Entry struct {
	Kind      string `yaml:"kind" json:"kind"`
	ID        string `yaml:"id" json:"id"`
	Source    string `yaml:"source" json:"source"`
	Ref       string `yaml:"ref,omitempty" json:"ref,omitempty"`
	Commit    string `yaml:"commit,omitempty" json:"commit,omitempty"`
	Hash      string `yaml:"hash" json:"hash"`
	Signature string `yaml:"signature,omitempty" json:"signature,omitempty"`
}

// Lockfile represents gradus.lock (§9.6).
type Lockfile struct {
	Schema         string  `yaml:"schema" json:"schema"`
	DefinitionHash string  `yaml:"definition_hash" json:"definition_hash"`
	Entries        []Entry `yaml:"entries" json:"entries"`
	ViewHash       string  `yaml:"view_hash" json:"view_hash"`
}

// ReadLockfile reads and parses gradus.lock from path.
func ReadLockfile(path string) (Lockfile, error) {
	var lf Lockfile
	data, err := os.ReadFile(path)
	if err != nil {
		return lf, err
	}
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return lf, fmt.Errorf("invalid gradus.lock YAML: %w", err)
	}
	if lf.Schema != schema.SchemaVersion {
		return lf, fmt.Errorf("gradus.lock: unsupported schema %q", lf.Schema)
	}
	return lf, nil
}

// WriteLockfile writes gradus.lock to path, strictly excluding any user-level entry (§9.6).
func WriteLockfile(path string, lf Lockfile) error {
	lf.Schema = schema.SchemaVersion
	filtered := make([]Entry, 0, len(lf.Entries))
	for _, e := range lf.Entries {
		if e.ID == "user" {
			continue
		}
		filtered = append(filtered, e)
	}
	lf.Entries = filtered

	out, err := yaml.Marshal(&lf)
	if err != nil {
		return fmt.Errorf("failed to marshal gradus.lock: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("failed to write gradus.lock: %w", err)
	}
	return nil
}

// FindEntry looks up an entry by kind ("level" or "overlay") and id.
func (lf Lockfile) FindEntry(kind, id string) (Entry, bool) {
	for _, e := range lf.Entries {
		if e.Kind == kind && e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}

// VerifyLockedEntry verifies that actualHash matches the recorded lock entry hash.
func VerifyLockedEntry(entry Entry, actualHash string) error {
	if entry.Hash == "" {
		return fmt.Errorf("locked %s %q has empty hash in gradus.lock", entry.Kind, entry.ID)
	}
	if entry.Hash != actualHash {
		return fmt.Errorf("lock integrity mismatch for locked %s %q: expected %s, got %s", entry.Kind, entry.ID, entry.Hash, actualHash)
	}
	return nil
}

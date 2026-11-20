package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/lock"
)

// ResolvedSource describes a resolved level or overlay directory on disk.
type ResolvedSource struct {
	Kind       string
	ID         string
	Dir        string
	SourceSpec string
	Ref        string
	Commit     string
	Hash       string
	Skipped    bool
	Notes      []string
}

// ResolveLocalSource resolves a local path source (expanding ~/ outside CI).
func ResolveLocalSource(rootDir, kind, id, rawPath string, inCI bool) (ResolvedSource, error) {
	out := ResolvedSource{
		Kind:       kind,
		ID:         id,
		SourceSpec: rawPath,
	}

	if rawPath == "~" || strings.HasPrefix(rawPath, "~/") || strings.HasPrefix(rawPath, `~\`) {
		if inCI {
			out.Skipped = true
			out.Notes = append(out.Notes, fmt.Sprintf("skipping home-directory %s %q (%s) in CI", kind, id, rawPath))
			return out, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return out, fmt.Errorf("%s %q: failed to resolve home directory: %w", kind, id, err)
		}
		trimmed := strings.TrimPrefix(strings.TrimPrefix(rawPath, "~/"), `~\`)
		if rawPath == "~" {
			trimmed = ""
		}
		resolved := filepath.Join(home, filepath.FromSlash(trimmed))
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			out.Skipped = true
			out.Notes = append(out.Notes, fmt.Sprintf("home-directory %s %q (%s) does not exist yet", kind, id, resolved))
			return out, nil
		}
		out.Dir = resolved
	} else if filepath.IsAbs(rawPath) {
		out.Dir = rawPath
	} else {
		out.Dir = filepath.Join(rootDir, filepath.FromSlash(rawPath))
	}

	info, err := os.Stat(out.Dir)
	if err != nil {
		return out, fmt.Errorf("%s %q source directory %s not accessible: %w", kind, id, out.Dir, err)
	}
	if !info.IsDir() {
		return out, fmt.Errorf("%s %q source path %s is not a directory", kind, id, out.Dir)
	}

	hash, err := lock.HashDirectory(out.Dir)
	if err != nil {
		return out, err
	}
	out.Hash = hash
	return out, nil
}

package lock

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HashBytes computes a normalized SHA-256 digest ("sha256:<hex>") with CRLF normalized to LF.
func HashBytes(data []byte) string {
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	sum := sha256.Sum256(normalized)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// HashDirectory computes a deterministic, cross-platform SHA-256 digest over all regular files
// inside dir, skipping .git directories and transient metadata (.fetched_at, .DS_Store).
func HashDirectory(dir string) (string, error) {
	type fileEntry struct {
		relPath string
		absPath string
	}

	var files []fileEntry
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink %q is not allowed in governed directory %s (Threat Model T2)", path, dir)
		}
		if d.IsDir() {
			if name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if name == ".fetched_at" || name == ".DS_Store" || strings.HasPrefix(name, "._") {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, fileEntry{
			relPath: filepath.ToSlash(rel),
			absPath: path,
		})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to hash directory %s: %w", dir, err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})

	h := sha256.New()
	for _, f := range files {
		raw, err := os.ReadFile(f.absPath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s for hashing: %w", f.absPath, err)
		}
		norm := bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
		fmt.Fprintf(h, "%s\n%d\n", f.relPath, len(norm))
		h.Write(norm)
		h.Write([]byte("\n"))
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

package source

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

const defaultCacheMaxAge = 7 * 24 * time.Hour

// ResolveGitSource fetches or reuses a cached git level/overlay under .gradus/cache/<id>/repo
// using the system git binary via os/exec (D-69).
func ResolveGitSource(rootDir, kind, id string, src schema.SourceRef, offline *schema.OfflinePolicy, lockEntry *lock.Entry) (ResolvedSource, error) {
	return resolveGitSource(rootDir, kind, id, src, offline, lockEntry, false)
}

// ResolveGitSourceOffline resolves a cached git level/overlay without any network calls.
// Used by latency-sensitive commands such as gradusctl guard (< 100ms).
func ResolveGitSourceOffline(rootDir, kind, id string, src schema.SourceRef, offline *schema.OfflinePolicy, lockEntry *lock.Entry) (ResolvedSource, error) {
	return resolveGitSource(rootDir, kind, id, src, offline, lockEntry, true)
}

func resolveGitSource(rootDir, kind, id string, src schema.SourceRef, offline *schema.OfflinePolicy, lockEntry *lock.Entry, offlineOnly bool) (ResolvedSource, error) {
	out := ResolvedSource{
		Kind:       kind,
		ID:         id,
		SourceSpec: src.Git,
		Ref:        src.Ref,
	}

	if strings.TrimSpace(src.Git) == "" {
		return out, fmt.Errorf("%s %q: git source URL is empty", kind, id)
	}
	ref := strings.TrimSpace(src.Ref)
	if ref == "" {
		ref = "HEAD"
		out.Ref = ref
	}

	cacheBase := filepath.Join(rootDir, ".gradus", "cache", id)
	repoDir := filepath.Join(cacheBase, "repo")
	fetchedAtPath := filepath.Join(cacheBase, ".fetched_at")

	// Fast path: if lockEntry is present and cache already matches commit + hash + ref, avoid network fetch (§8).
	if lockEntry != nil && lockEntry.Commit != "" && lockEntry.Hash != "" && (lockEntry.Ref == "" || lockEntry.Ref == ref) {
		if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
			if head, err := gitOutput(repoDir, "rev-parse", "HEAD"); err == nil && head == lockEntry.Commit {
				if hash, err := lock.HashDirectory(repoDir); err == nil && hash == lockEntry.Hash {
					out.Dir = repoDir
					out.Commit = head
					out.Hash = hash
					return out, nil
				}
			}
		}
	}

	if offlineOnly {
		if _, statErr := os.Stat(filepath.Join(repoDir, ".git")); statErr == nil {
			commit, _ := gitOutput(repoDir, "rev-parse", "HEAD")
			hash, hashErr := lock.HashDirectory(repoDir)
			if hashErr == nil {
				out.Dir = repoDir
				out.Commit = commit
				out.Hash = hash
				return out, nil
			}
			return out, fmt.Errorf("%s %q cached repository hash failed: %w", kind, id, hashErr)
		}
		return out, fmt.Errorf("%s %q is not cached for offline evaluation; run gradusctl resolve", kind, id)
	}

	if err := os.MkdirAll(cacheBase, 0o755); err != nil {
		return out, fmt.Errorf("%s %q: failed to create cache dir: %w", kind, id, err)
	}

	fetchErr := syncGitRepo(repoDir, src.Git, ref)
	if fetchErr != nil {
		// Apply offline policy.
		if offline != nil && offline.Mode == "cache" {
			if _, statErr := os.Stat(filepath.Join(repoDir, ".git")); statErr == nil {
				ok, ageErr := isCacheFresh(fetchedAtPath, offline.MaxAge)
				if ok {
					commit, _ := gitOutput(repoDir, "rev-parse", "HEAD")
					hash, hashErr := lock.HashDirectory(repoDir)
					if hashErr == nil {
						out.Dir = repoDir
						out.Commit = commit
						out.Hash = hash
						out.Notes = append(out.Notes, fmt.Sprintf("offline cache fallback used for %s %q: %v", kind, id, fetchErr))
						return out, nil
					}
				} else if ageErr != nil {
					return out, fmt.Errorf("%s %q unreachable and cache invalid (%v): %w", kind, id, ageErr, fetchErr)
				}
			}
		}
		return out, fmt.Errorf("%s %q git fetch failed: %w", kind, id, fetchErr)
	}

	commit, err := gitOutput(repoDir, "rev-parse", "HEAD")
	if err != nil {
		return out, fmt.Errorf("%s %q: failed to resolve HEAD commit: %w", kind, id, err)
	}
	hash, err := lock.HashDirectory(repoDir)
	if err != nil {
		return out, err
	}

	_ = os.WriteFile(fetchedAtPath, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644)

	out.Dir = repoDir
	out.Commit = commit
	out.Hash = hash
	return out, nil
}

func syncGitRepo(repoDir, url, ref string) error {
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		if err := runGit("", "clone", "--no-checkout", "--", url, repoDir); err != nil {
			return err
		}
	} else {
		if err := runGit(repoDir, "remote", "set-url", "origin", url); err != nil {
			return err
		}
		if err := runGit(repoDir, "fetch", "--tags", "--force", "origin"); err != nil {
			return err
		}
	}
	if err := runGit(repoDir, "checkout", "--detach", ref, "--"); err != nil {
		return fmt.Errorf("checkout %q: %w", ref, err)
	}
	return nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func isCacheFresh(fetchedAtPath, maxAgeStr string) (bool, error) {
	data, err := os.ReadFile(fetchedAtPath)
	if err != nil {
		return false, fmt.Errorf("missing .fetched_at: %w", err)
	}
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return false, fmt.Errorf("invalid .fetched_at timestamp: %w", err)
	}
	maxAge := defaultCacheMaxAge
	if strings.TrimSpace(maxAgeStr) != "" {
		parsed, err := parseDurationWithDays(maxAgeStr)
		if err != nil {
			return false, err
		}
		maxAge = parsed
	}
	if time.Since(ts) > maxAge {
		return false, fmt.Errorf("cache expired (age %s > max_age %s)", time.Since(ts).Round(time.Second), maxAge)
	}
	return true, nil
}

func parseDurationWithDays(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(numStr)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

package source

import (
	"fmt"
	"strings"
)

// VerifyGitSignature verifies the GPG or SSH signature on a git tag or commit
// using the system git binary via os/exec (PRJ-GO-001).
// Returns "verified" when signature verification succeeds, or "none" with an error.
func VerifyGitSignature(repoDir, ref string) (string, error) {
	cleanRef := strings.TrimSpace(ref)
	if cleanRef == "" {
		cleanRef = "HEAD"
	}
	if strings.HasPrefix(cleanRef, "-") {
		return "none", fmt.Errorf("invalid git ref %q (cannot start with '-')", cleanRef)
	}

	objType, err := gitOutput(repoDir, "cat-file", "-t", "--", cleanRef)
	if err != nil {
		return "none", fmt.Errorf("failed to inspect git ref %q: %w", cleanRef, err)
	}

	if objType == "tag" {
		if err := runGit(repoDir, "verify-tag", "--", cleanRef); err != nil {
			return "none", fmt.Errorf("git verify-tag %q failed: %w", cleanRef, err)
		}
		return "verified", nil
	}

	if err := runGit(repoDir, "verify-commit", "--", cleanRef); err != nil {
		return "none", fmt.Errorf("git verify-commit %q failed: %w", cleanRef, err)
	}
	return "verified", nil
}

package trailers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// GenerateOptions configures optional attribution trailers (X-05, D-65).
type GenerateOptions struct {
	Agent     string
	ModelTier string
	ReviewID  string
}

// TrailerSet holds generated Gradus commit trailers bound to the current lock/view hashes.
type TrailerSet struct {
	LockHash string            `json:"lock_hash,omitempty"`
	ViewHash string            `json:"view_hash"`
	Trailers map[string]string `json:"trailers"`
	Lines    []string          `json:"lines"`
}

// VerifyOptions configures commit trailer verification.
type VerifyOptions struct {
	RequireLockHash bool
}

// VerifyReport holds the result of verifying Gradus trailers on a git commit.
type VerifyReport struct {
	Ref              string            `json:"ref"`
	Verdict          string            `json:"verdict"` // pass | fail
	Trailers         map[string]string `json:"trailers"`
	ExpectedLockHash string            `json:"expected_lock_hash,omitempty"`
	Issues           []string          `json:"issues"`
}

var trailerOrder = []string{
	"Gradus-Lock-Hash",
	"Gradus-View-Hash",
	"Gradus-Agent",
	"Gradus-Model-Tier",
	"Gradus-Review-ID",
}

// Generate produces deterministic Gradus-* git commit trailer lines for rootDir.
func Generate(rootDir string, opts GenerateOptions) (TrailerSet, error) {
	var out TrailerSet
	if mt := strings.TrimSpace(opts.ModelTier); mt != "" && resolve.TierRank(mt) == 0 {
		return out, fmt.Errorf("invalid --model-tier %q (expected tier-high, tier-standard, or tier-light)", opts.ModelTier)
	}

	res, err := resolve.Resolve(rootDir, resolve.Options{OfflineOnly: true})
	if err != nil {
		return out, err
	}

	lockHash := res.View.LockHash
	if lockHash == "" {
		if b, rerr := os.ReadFile(filepath.Join(rootDir, "gradus.lock")); rerr == nil {
			lockHash = lock.HashBytes(b)
		}
	}

	m := map[string]string{
		"Gradus-View-Hash": res.View.ViewHash,
	}
	if lockHash != "" {
		m["Gradus-Lock-Hash"] = lockHash
	}
	if a := strings.TrimSpace(opts.Agent); a != "" {
		m["Gradus-Agent"] = a
	}
	if mt := strings.TrimSpace(opts.ModelTier); mt != "" {
		m["Gradus-Model-Tier"] = mt
	}
	if rid := strings.TrimSpace(opts.ReviewID); rid != "" {
		m["Gradus-Review-ID"] = rid
	}

	var lines []string
	for _, k := range trailerOrder {
		if v, ok := m[k]; ok {
			lines = append(lines, fmt.Sprintf("%s: %s", k, v))
		}
	}

	return TrailerSet{
		LockHash: lockHash,
		ViewHash: res.View.ViewHash,
		Trailers: m,
		Lines:    lines,
	}, nil
}

// ParseCommitMessage extracts Gradus-* key-value trailers from a commit message.
func ParseCommitMessage(msg string) map[string]string {
	out := make(map[string]string)
	for _, rawLine := range strings.Split(strings.ReplaceAll(msg, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "Gradus-") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}

// VerifyCommit inspects git commit ref via os/exec git (PRJ-GO-001) and checks Gradus-* trailers.
func VerifyCommit(rootDir, ref string, opts VerifyOptions) (VerifyReport, error) {
	if strings.TrimSpace(ref) == "" {
		ref = "HEAD"
	}
	cmd := exec.Command("git", "log", "-1", "--format=%B", ref)
	cmd.Dir = rootDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return VerifyReport{}, fmt.Errorf("git log %s failed: %w (%s)", ref, err, strings.TrimSpace(stderr.String()))
	}

	parsed := ParseCommitMessage(stdout.String())
	issues := []string{}

	if mt, ok := parsed["Gradus-Model-Tier"]; ok {
		if resolve.TierRank(mt) == 0 {
			issues = append(issues, fmt.Sprintf("invalid Gradus-Model-Tier %q", mt))
		}
	}

	expectedLockHash := ""
	showCmd := exec.Command("git", "show", ref+":gradus.lock")
	showCmd.Dir = rootDir
	if lockAtRef, serr := showCmd.Output(); serr == nil {
		expectedLockHash = lock.HashBytes(lockAtRef)
	} else if b, err := os.ReadFile(filepath.Join(rootDir, "gradus.lock")); err == nil {
		expectedLockHash = lock.HashBytes(b)
	}

	if gotLock, ok := parsed["Gradus-Lock-Hash"]; ok && gotLock != "" {
		if expectedLockHash != "" && gotLock != expectedLockHash {
			issues = append(issues, fmt.Sprintf("Gradus-Lock-Hash %s does not match gradus.lock (%s)", gotLock, expectedLockHash))
		}
	} else if opts.RequireLockHash {
		issues = append(issues, "missing required Gradus-Lock-Hash trailer")
	}

	verdict := "pass"
	if len(issues) > 0 {
		verdict = "fail"
	}

	return VerifyReport{
		Ref:              ref,
		Verdict:          verdict,
		Trailers:         parsed,
		ExpectedLockHash: expectedLockHash,
		Issues:           issues,
	}, nil
}

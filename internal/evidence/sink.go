package evidence

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/version"
)

const defaultLocalRetention = 30 * 24 * time.Hour

// Record matches spec/v1/schema/evidence.schema.json (§9.9) with zero personal activity metrics (P16).
type Record struct {
	Schema    string    `json:"schema"`
	ID        string    `json:"id"`
	Kind      string    `json:"kind"` // verify | check | review | semantic-conflicts | exception-use
	Timestamp string    `json:"timestamp"`
	Commit    string    `json:"commit,omitempty"`
	Base      string    `json:"base,omitempty"`
	LockHash  string    `json:"lock_hash"`
	ViewHash  string    `json:"view_hash"`
	Producer  Producer  `json:"producer"`
	Findings  []Finding `json:"findings"`
	Verdict   string    `json:"verdict"` // pass | fail | error
}

// Producer identifies the tool that generated an evidence record.
type Producer struct {
	Tool      string  `json:"tool"`
	Version   string  `json:"version"`
	Skill     *string `json:"skill"`
	ModelTier *string `json:"model_tier"`
}

// Finding represents a single rule check or verification finding (§9.9).
type Finding struct {
	RuleID     string  `json:"rule_id"`
	Severity   string  `json:"severity"`
	Level      string  `json:"level,omitempty"`
	Source     string  `json:"source,omitempty"`
	Path       string  `json:"path,omitempty"`
	Line       int     `json:"line,omitempty"`
	Message    string  `json:"message"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"` // open | dismissed | confirmed
}

// NewRecord constructs a canonical gradus/v1 evidence record.
func NewRecord(rootDir, kind, verdict, base, lockHash, viewHash string, findings []Finding, now time.Time) Record {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if findings == nil {
		findings = []Finding{}
	}
	commit := gitShortHead(rootDir)

	var randBuf [2]byte
	_, _ = rand.Read(randBuf[:])
	id := fmt.Sprintf("ev-%s-%s", now.UTC().Format("20060102-150405"), hex.EncodeToString(randBuf[:]))

	return Record{
		Schema:    schema.SchemaVersion,
		ID:        id,
		Kind:      kind,
		Timestamp: now.UTC().Format(time.RFC3339),
		Commit:    commit,
		Base:      base,
		LockHash:  lockHash,
		ViewHash:  viewHash,
		Producer: Producer{
			Tool:    "gradusctl",
			Version: version.Get().Version,
		},
		Findings: findings,
		Verdict:  verdict,
	}
}

// ActiveSinks selects CI sinks when CI is set and evidence.ci.sinks is configured;
// otherwise returns evidence.sinks (defaulting to local if empty).
func ActiveSinks(cfg *schema.EvidenceConfig, inCI bool) []schema.SinkConfig {
	if cfg != nil {
		if inCI && cfg.CI != nil && len(cfg.CI.Sinks) > 0 {
			return cfg.CI.Sinks
		}
		if len(cfg.Sinks) > 0 {
			return cfg.Sinks
		}
	}
	return []schema.SinkConfig{{Type: "local"}}
}

// WriteRecord persists rec to all configured sinks.
func WriteRecord(rootDir string, cfg *schema.EvidenceConfig, rec Record, now time.Time) ([]string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	inCI := os.Getenv("CI") != ""
	sinks := ActiveSinks(cfg, inCI)

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal evidence record: %w", err)
	}
	data = append(data, '\n')

	var written []string
	for _, s := range sinks {
		switch s.Type {
		case "none":
			continue
		case "local":
			dir := filepath.Join(rootDir, ".gradus", "evidence")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return written, fmt.Errorf("failed to create local evidence dir: %w", err)
			}
			dest := filepath.Join(dir, rec.ID+".json")
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return written, fmt.Errorf("failed to write local evidence %s: %w", dest, err)
			}
			written = append(written, filepath.ToSlash(filepath.Join(".gradus", "evidence", rec.ID+".json")))
			_ = pruneExpired(dir, s.Retention, now)
		case "ci-artifact":
			rel := s.Path
			if rel == "" {
				rel = filepath.ToSlash(filepath.Join(".gradus", "ci-artifacts"))
			}
			dir := resolveSinkPath(rootDir, rel)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return written, fmt.Errorf("failed to create ci-artifact dir: %w", err)
			}
			dest := filepath.Join(dir, rec.ID+".json")
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return written, fmt.Errorf("failed to write ci-artifact %s: %w", dest, err)
			}
			written = append(written, filepath.ToSlash(filepath.Join(rel, rec.ID+".json")))
		case "repo":
			rel := s.Path
			if rel == "" {
				rel = "evidence"
			}
			dir := resolveSinkPath(rootDir, rel)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return written, fmt.Errorf("failed to create repo evidence dir: %w", err)
			}
			dest := filepath.Join(dir, rec.ID+".json")
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return written, fmt.Errorf("failed to write repo evidence %s: %w", dest, err)
			}
			written = append(written, filepath.ToSlash(filepath.Join(rel, rec.ID+".json")))
		case "command":
			if strings.TrimSpace(s.Command) == "" {
				return written, fmt.Errorf("command evidence sink requires non-empty command")
			}
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/C", s.Command)
			} else {
				cmd = exec.Command("sh", "-c", s.Command)
			}
			cmd.Dir = rootDir
			cmd.Stdin = bytes.NewReader(data)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				return written, fmt.Errorf("command evidence sink failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
			}
			written = append(written, "command:"+s.Command)
		default:
			return written, fmt.Errorf("unsupported evidence sink type %q", s.Type)
		}
	}

	return written, nil
}

// ListLocalRecords returns all local evidence records sorted newest-first.
func ListLocalRecords(rootDir string) ([]Record, error) {
	dir := filepath.Join(rootDir, ".gradus", "evidence")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, err
	}

	var out []Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var rec Record
		if json.Unmarshal(raw, &rec) == nil {
			out = append(out, rec)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Timestamp != out[j].Timestamp {
			return out[i].Timestamp > out[j].Timestamp
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

func resolveSinkPath(rootDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(rootDir, filepath.FromSlash(p))
}

func pruneExpired(dir, retentionStr string, now time.Time) error {
	maxAge := defaultLocalRetention
	if strings.TrimSpace(retentionStr) != "" {
		if d, err := parseRetention(retentionStr); err == nil {
			maxAge = d
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var rec Record
		if json.Unmarshal(raw, &rec) != nil || rec.Timestamp == "" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, rec.Timestamp)
		if err != nil {
			continue
		}
		if now.Sub(ts) > maxAge {
			_ = os.Remove(p)
		}
	}
	return nil
}

func parseRetention(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid retention %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func gitShortHead(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

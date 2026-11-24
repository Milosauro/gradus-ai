package state

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

// ExceptionRecord represents a persisted exception or break-glass waiver under .gradus/exceptions/.
type ExceptionRecord struct {
	Schema        string   `yaml:"schema" json:"schema"`
	ID            string   `yaml:"id" json:"id"`
	RuleID        string   `yaml:"rule_id" json:"rule_id"`
	Scope         []string `yaml:"scope" json:"scope"`
	Reason        string   `yaml:"reason" json:"reason"`
	ApprovedBy    []string `yaml:"approved_by" json:"approved_by"`
	Created       string   `yaml:"created" json:"created"`
	Expires       string   `yaml:"expires" json:"expires"`
	Kind          string   `yaml:"kind" json:"kind"`
	PostReviewDue string   `yaml:"post_review_due,omitempty" json:"post_review_due,omitempty"`
}

// ExceptionSummary pairs an ExceptionRecord with its computed lifecycle status (pending, active, expired).
type ExceptionSummary struct {
	ExceptionRecord `json:",inline"`
	Status          string `json:"status"`
}

// NewExceptionID generates a unique timestamped exception or break-glass identifier.
func NewExceptionID(kind string, now time.Time) string {
	prefix := "EX"
	if kind == "break-glass" {
		prefix = "BG"
	}
	var suffix [2]byte
	_, _ = rand.Read(suffix[:])
	return fmt.Sprintf("%s-%s-%s", prefix, now.UTC().Format("20060102-150405"), hex.EncodeToString(suffix[:]))
}

// ComputeExceptionStatus returns "expired", "pending", or "active" for an ExceptionRecord.
func ComputeExceptionStatus(rec ExceptionRecord, now time.Time) string {
	today := now.UTC().Format("2006-01-02")
	if rec.Expires != "" && rec.Expires < today {
		return "expired"
	}
	if (rec.Kind == "" || rec.Kind == "exception") && len(rec.ApprovedBy) == 0 {
		return "pending"
	}
	return "active"
}

// WriteException writes rec to .gradus/exceptions/<id>.yaml.
func WriteException(rootDir string, rec ExceptionRecord) (string, error) {
	if rec.Schema == "" {
		rec.Schema = schema.SchemaVersion
	}
	if len(rec.Scope) == 0 {
		rec.Scope = []string{"**/*"}
	}
	if rec.ApprovedBy == nil {
		rec.ApprovedBy = []string{}
	}
	dir := filepath.Join(rootDir, ".gradus", "exceptions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", dir, err)
	}
	data, err := yaml.Marshal(rec)
	if err != nil {
		return "", err
	}
	outPath := filepath.Join(dir, rec.ID+".yaml")
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", outPath, err)
	}
	return outPath, nil
}

// ListExceptions reads all YAML exception records under .gradus/exceptions/ sorted by ID.
func ListExceptions(rootDir string, now time.Time) ([]ExceptionSummary, error) {
	dir := filepath.Join(rootDir, ".gradus", "exceptions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ExceptionSummary{}, nil
		}
		return nil, err
	}

	var out []ExceptionSummary
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var rec ExceptionRecord
		if err := yaml.Unmarshal(raw, &rec); err != nil {
			return nil, fmt.Errorf("invalid exception %s: %w", e.Name(), err)
		}
		out = append(out, ExceptionSummary{
			ExceptionRecord: rec,
			Status:          ComputeExceptionStatus(rec, now),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if out == nil {
		out = []ExceptionSummary{}
	}
	return out, nil
}

// ApproveException appends approver to approved_by in .gradus/exceptions/<id>.yaml.
func ApproveException(rootDir, id, approver string) (ExceptionRecord, error) {
	path, rec, err := readExceptionByID(rootDir, id)
	if err != nil {
		return ExceptionRecord{}, err
	}
	if strings.TrimSpace(approver) == "" {
		return ExceptionRecord{}, fmt.Errorf("approver must not be empty")
	}
	if !slices.Contains(rec.ApprovedBy, approver) {
		rec.ApprovedBy = append(rec.ApprovedBy, approver)
	}
	data, err := yaml.Marshal(rec)
	if err != nil {
		return ExceptionRecord{}, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ExceptionRecord{}, err
	}
	return rec, nil
}

// ExpireException sets expires to yesterday UTC in .gradus/exceptions/<id>.yaml so it expires immediately.
func ExpireException(rootDir, id string, now time.Time) (ExceptionRecord, error) {
	path, rec, err := readExceptionByID(rootDir, id)
	if err != nil {
		return ExceptionRecord{}, err
	}
	rec.Expires = now.UTC().AddDate(0, 0, -1).Format("2006-01-02")
	data, err := yaml.Marshal(rec)
	if err != nil {
		return ExceptionRecord{}, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ExceptionRecord{}, err
	}
	return rec, nil
}

func readExceptionByID(rootDir, id string) (string, ExceptionRecord, error) {
	dir := filepath.Join(rootDir, ".gradus", "exceptions")
	for _, ext := range []string{".yaml", ".yml"} {
		candidate := filepath.Join(dir, id+ext)
		raw, err := os.ReadFile(candidate)
		if err == nil {
			var rec ExceptionRecord
			if uerr := yaml.Unmarshal(raw, &rec); uerr != nil {
				return "", ExceptionRecord{}, fmt.Errorf("invalid exception %s: %w", candidate, uerr)
			}
			return candidate, rec, nil
		}
	}
	return "", ExceptionRecord{}, fmt.Errorf("exception %q not found in %s", id, dir)
}

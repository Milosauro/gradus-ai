package state

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

// LessonRecord matches spec/v1/schema/lesson.schema.json with zero personal metrics (P16).
type LessonRecord struct {
	Schema               string   `json:"schema"`
	ID                   string   `json:"id"`
	Created              string   `json:"created"`
	Kind                 string   `json:"kind"` // correction | repeated-mistake | preference | frustration
	Statement            string   `json:"statement"`
	Evidence             []string `json:"evidence,omitempty"`
	Occurrences          int      `json:"occurrences"`
	SuggestedLevel       string   `json:"suggested_level,omitempty"`
	SuggestedEnforcement string   `json:"suggested_enforcement,omitempty"`
	Status               string   `json:"status"` // open | proposed | promoted | dismissed
}

// ProjectKey computes a deterministic cross-platform directory key for rootDir.
func ProjectKey(rootDir string) string {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		abs = rootDir
	}
	cleaned := filepath.ToSlash(filepath.Clean(abs))
	sum := sha256.Sum256([]byte(cleaned))
	base := sanitizeIDPart(filepath.Base(abs))
	if base == "" || base == "." {
		base = "project"
	}
	return fmt.Sprintf("%s-%s", base, hex.EncodeToString(sum[:6]))
}

// LessonsDir returns overrideDir when provided, or ~/.gradus/lessons/<ProjectKey(rootDir)>.
func LessonsDir(rootDir, overrideDir string) (string, error) {
	if strings.TrimSpace(overrideDir) != "" {
		return overrideDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory for lessons store: %w", err)
	}
	return filepath.Join(home, ".gradus", "lessons", ProjectKey(rootDir)), nil
}

// AddLesson redacts secrets/PII (T4) and persists a new lesson record in dir.
func AddLesson(dir string, rec LessonRecord, now time.Time) (LessonRecord, string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !isValidLessonKind(rec.Kind) {
		return rec, "", fmt.Errorf("invalid lesson kind %q (expected correction, repeated-mistake, preference, or frustration)", rec.Kind)
	}
	if strings.TrimSpace(rec.Statement) == "" {
		return rec, "", fmt.Errorf("lesson statement cannot be empty")
	}

	rec.Schema = schema.SchemaVersion
	rec.Statement = RedactText(strings.TrimSpace(rec.Statement))
	if len(rec.Evidence) > 0 {
		redactedEv := make([]string, len(rec.Evidence))
		for i, ev := range rec.Evidence {
			redactedEv[i] = RedactText(ev)
		}
		rec.Evidence = redactedEv
	}

	if rec.Occurrences < 1 {
		rec.Occurrences = 1
	}
	if rec.Status == "" {
		rec.Status = "open"
	} else if !isValidLessonStatus(rec.Status) {
		return rec, "", fmt.Errorf("invalid lesson status %q", rec.Status)
	}

	if strings.TrimSpace(rec.ID) == "" {
		var randBuf [2]byte
		_, _ = rand.Read(randBuf[:])
		rec.ID = fmt.Sprintf("les-%s-%s", now.UTC().Format("20060102-150405"), hex.EncodeToString(randBuf[:]))
	}
	if rec.Created == "" {
		rec.Created = now.UTC().Format(time.RFC3339)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return rec, "", fmt.Errorf("failed to create lessons dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return rec, "", err
	}
	data = append(data, '\n')
	dest := filepath.Join(dir, rec.ID+".json")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return rec, "", err
	}
	return rec, dest, nil
}

// ListLessons reads all lesson JSON records in dir sorted newest-first.
func ListLessons(dir string) ([]LessonRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LessonRecord{}, nil
		}
		return nil, err
	}

	var out []LessonRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var rec LessonRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return nil, fmt.Errorf("invalid lesson file %s: %w", e.Name(), err)
		}
		out = append(out, rec)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Created != out[j].Created {
			return out[i].Created > out[j].Created
		}
		return out[i].ID > out[j].ID
	})
	if out == nil {
		return []LessonRecord{}, nil
	}
	return out, nil
}

// UpdateLesson updates an existing lesson's status and/or increments its occurrence count.
func UpdateLesson(dir, id, status string, increment bool) (LessonRecord, error) {
	if strings.TrimSpace(id) == "" {
		return LessonRecord{}, fmt.Errorf("lesson id is required")
	}
	p := filepath.Join(dir, id+".json")
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return LessonRecord{}, fmt.Errorf("lesson %q not found", id)
		}
		return LessonRecord{}, err
	}
	var rec LessonRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return LessonRecord{}, fmt.Errorf("invalid lesson %s: %w", id, err)
	}

	if strings.TrimSpace(status) != "" {
		if !isValidLessonStatus(status) {
			return LessonRecord{}, fmt.Errorf("invalid lesson status %q", status)
		}
		rec.Status = status
	}
	if increment {
		rec.Occurrences++
	}

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return LessonRecord{}, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return LessonRecord{}, err
	}
	return rec, nil
}

func isValidLessonKind(k string) bool {
	switch k {
	case "correction", "repeated-mistake", "preference", "frustration":
		return true
	default:
		return false
	}
}

func isValidLessonStatus(s string) bool {
	switch s {
	case "open", "proposed", "promoted", "dismissed":
		return true
	default:
		return false
	}
}

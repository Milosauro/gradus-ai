package v1_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func assertValidJSONSchemaFiles(t *testing.T, files []string) {
	t.Helper()
	for _, name := range files {
		path := filepath.Join("schema", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read schema %s: %v", path, err)
		}
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("invalid JSON in %s: %v", path, err)
		}
		if _, ok := doc["$schema"]; !ok {
			t.Fatalf("schema %s missing $schema", path)
		}
		if _, ok := doc["$id"]; !ok {
			t.Fatalf("schema %s missing $id", path)
		}
		if _, ok := doc["type"]; !ok {
			t.Fatalf("schema %s missing top-level type", path)
		}
	}
}

func TestConfigSchemasValid(t *testing.T) {
	specData, err := os.ReadFile("SPEC.md")
	if err != nil {
		t.Fatalf("failed to read SPEC.md: %v", err)
	}
	if len(specData) == 0 {
		t.Fatalf("SPEC.md must not be empty")
	}

	assertValidJSONSchemaFiles(t, []string{
		"gradus.schema.json",
		"level.schema.json",
		"overlay.schema.json",
		"rule-frontmatter.schema.json",
	})
}

func TestStateSchemasValid(t *testing.T) {
	assertValidJSONSchemaFiles(t, []string{
		"lock.schema.json",
		"resolved-view.schema.json",
		"evidence.schema.json",
		"lesson.schema.json",
		"exception.schema.json",
	})
}

func TestJSONSchemasCompileAndValidate(t *testing.T) {
	t.Run("config", TestConfigSchemasValid)
	t.Run("state", TestStateSchemasValid)
}

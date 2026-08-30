package packagecachefixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPackageCacheSchemaRequiresClosedRoot(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "package-cache", "schema-v7.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		AdditionalProperties bool     `json:"additionalProperties"`
		Required             []string `json:"required"`
		Properties           map[string]struct {
			Const string `json:"const"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.AdditionalProperties || !slices.Equal(schema.Required, []string{"$schema", "schemaVersion", "contracts", "base", "cases"}) {
		t.Fatalf("schema closure = additionalProperties %v required %v", schema.AdditionalProperties, schema.Required)
	}
	if schema.Properties["schemaVersion"].Const != schemaVersion {
		t.Fatalf("schema version = %q", schema.Properties["schemaVersion"].Const)
	}
}

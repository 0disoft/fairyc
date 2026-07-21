package projectmanifestfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestProjectManifestFixtureSchemaIsClosed(t *testing.T) {
	t.Parallel()
	contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "project-manifest", "schema-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		AdditionalProperties bool     `json:"additionalProperties"`
		Required             []string `json:"required"`
		Properties           map[string]struct {
			Const string `json:"const"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.AdditionalProperties || !slices.Equal(schema.Required, []string{"$schema", "schemaVersion", "contracts", "cases"}) || schema.Properties["schemaVersion"].Const != schemaVersion {
		t.Fatalf("schema closure = additionalProperties %v required %v version %q", schema.AdditionalProperties, schema.Required, schema.Properties["schemaVersion"].Const)
	}
}

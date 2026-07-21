package packageidentityfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPackageIdentitySchemaRequiresClosedContractFields(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "package-identity", "schema-v1.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		AdditionalProperties bool `json:"additionalProperties"`
		Required             []string
		Defs                 map[string]struct {
			AdditionalProperties bool     `json:"additionalProperties"`
			Required             []string `json:"required"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.AdditionalProperties || !slices.Equal(schema.Required, []string{"$schema", "schemaVersion", "contracts", "cases"}) {
		t.Fatalf("root schema closure = additionalProperties %v required %v", schema.AdditionalProperties, schema.Required)
	}
	for name, required := range map[string][]string{
		"identity":          {"packageUid", "sourceIdentity", "versionIdentity", "features"},
		"passExpectation":   {"status", "encodingHex", "instanceId"},
		"rejectExpectation": {"status", "reason"},
		"case":              {"id", "identity", "expect"},
	} {
		definition, ok := schema.Defs[name]
		if !ok || definition.AdditionalProperties || !slices.Equal(definition.Required, required) {
			t.Fatalf("definition %q closure = %+v, present %v", name, definition, ok)
		}
	}
}

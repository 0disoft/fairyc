package mirverifyfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMIRVerifierSchemaMatchesRunnerVocabulary(t *testing.T) {
	t.Parallel()
	contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "mir-verifier", "schema-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Dialect string `json:"$schema"`
		ID      string `json:"$id"`
		Defs    struct {
			EntityLabel struct {
				Pattern string `json:"pattern"`
			} `json:"entityLabel"`
			Mutation struct {
				Properties struct {
					Kind struct {
						Enum []string `json:"enum"`
					} `json:"kind"`
				} `json:"properties"`
			} `json:"mutation"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:mir-verifier-fixtures:v0" {
		t.Fatalf("schema identity = %q, %q", schema.Dialect, schema.ID)
	}
	if !slices.Equal(schema.Defs.Mutation.Properties.Kind.Enum, mutationKinds) {
		t.Fatalf("schema mutation kinds = %v, want %v", schema.Defs.Mutation.Properties.Kind.Enum, mutationKinds)
	}
	if schema.Defs.EntityLabel.Pattern != "^(block|flag|form|loan|local|operation|origin|place|region|resource-type|scope|span|statement|terminator|unsafe):[a-z][a-z0-9]*(?:-[a-z0-9]+)*$" {
		t.Fatalf("entity-label pattern = %q", schema.Defs.EntityLabel.Pattern)
	}
}

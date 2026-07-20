package packagefixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestPackageJSONSchemaMatchesRunnerVocabulary(t *testing.T) {
	t.Parallel()
	contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "package", "schema-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Dialect  string   `json:"$schema"`
		ID       string   `json:"$id"`
		Required []string `json:"required"`
		Defs     struct {
			FailExpectation struct {
				Properties struct {
					Reason struct {
						Enum []string `json:"enum"`
					} `json:"reason"`
				} `json:"properties"`
			} `json:"failExpectation"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:package-fixtures:v0" {
		t.Fatalf("schema identity = %q, %q", schema.Dialect, schema.ID)
	}
	slices.Sort(schema.Required)
	wantRequired := []string{"$schema", "cases", "contracts", "deferred", "schemaVersion"}
	if !slices.Equal(schema.Required, wantRequired) {
		t.Fatalf("schema required = %v, want %v", schema.Required, wantRequired)
	}
	wantReasons := []string{
		packagegraph.ReasonAmbiguousVisibleModule,
		packagegraph.ReasonDuplicateBinding,
		packagegraph.ReasonMissingVisibleModule,
	}
	if !slices.Equal(schema.Defs.FailExpectation.Properties.Reason.Enum, wantReasons) {
		t.Fatalf("schema reasons = %v, want %v", schema.Defs.FailExpectation.Properties.Reason.Enum, wantReasons)
	}
}

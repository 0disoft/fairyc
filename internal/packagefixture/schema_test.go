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
	v1Contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "package", "schema-v1.json"))
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
						Const string `json:"const"`
					} `json:"reason"`
				} `json:"properties"`
			} `json:"failExpectation"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(v1Contents, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:package-fixtures:v1" {
		t.Fatalf("schema identity = %q, %q", schema.Dialect, schema.ID)
	}
	slices.Sort(schema.Required)
	wantRequired := []string{"$schema", "cases", "contracts", "deferred", "includes", "schemaVersion"}
	if !slices.Equal(schema.Required, wantRequired) {
		t.Fatalf("schema required = %v, want %v", schema.Required, wantRequired)
	}
	if schema.Defs.FailExpectation.Properties.Reason.Const != packagegraph.ReasonDependencyCycle {
		t.Fatalf("v1 schema reason = %q", schema.Defs.FailExpectation.Properties.Reason.Const)
	}

	v0Contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "package", "schema-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v0 struct {
		Defs struct {
			FailExpectation struct {
				Properties struct {
					Reason struct {
						Enum []string `json:"enum"`
					} `json:"reason"`
				} `json:"properties"`
			} `json:"failExpectation"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(v0Contents, &v0); err != nil {
		t.Fatalf("decode v0 schema: %v", err)
	}
	wantV0Reasons := []string{
		packagegraph.ReasonAmbiguousVisibleModule,
		packagegraph.ReasonDuplicateBinding,
		packagegraph.ReasonMissingVisibleModule,
	}
	if !slices.Equal(v0.Defs.FailExpectation.Properties.Reason.Enum, wantV0Reasons) {
		t.Fatalf("v0 schema reasons = %v, want %v", v0.Defs.FailExpectation.Properties.Reason.Enum, wantV0Reasons)
	}
}

package contractstateconflictfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestContractStateConflictSchemaMatchesRunnerVocabulary(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct {
		version string
		actions []string
	}{{"v0", actionKindsV0}, {"v1", actionKindsV1}} {
		fixture := fixture
		t.Run(fixture.version, func(t *testing.T) {
			t.Parallel()
			contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "contract-state-conflict", "schema-"+fixture.version+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var schema struct {
				Dialect string `json:"$schema"`
				ID      string `json:"$id"`
				Defs    struct {
					Action struct {
						Enum []string `json:"enum"`
					} `json:"action"`
					Status struct {
						Enum []string `json:"enum"`
					} `json:"status"`
				} `json:"$defs"`
			}
			if err := json.Unmarshal(contents, &schema); err != nil {
				t.Fatalf("decode schema: %v", err)
			}
			if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:contract-state-conflict-fixtures:"+fixture.version {
				t.Fatalf("schema identity = %q, %q", schema.Dialect, schema.ID)
			}
			if !slices.Equal(schema.Defs.Action.Enum, fixture.actions) || !slices.Equal(schema.Defs.Status.Enum, resultStatuses) {
				t.Fatalf("schema vocabulary does not match runner")
			}
		})
	}
}

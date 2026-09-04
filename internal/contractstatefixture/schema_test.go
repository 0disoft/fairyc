package contractstatefixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestContractStateSchemaMatchesRunnerVocabulary(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"v0", "v1", "v2"} {
		version := version
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "contract-state", "schema-"+version+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var schema struct {
				Dialect string `json:"$schema"`
				ID      string `json:"$id"`
				Defs    struct {
					OperationKind struct {
						Enum []string `json:"enum"`
					} `json:"operationKind"`
					ExitKind struct {
						Enum []string `json:"enum"`
					} `json:"exitKind"`
					Status struct {
						Enum []string `json:"enum"`
					} `json:"status"`
				} `json:"$defs"`
			}
			if err := json.Unmarshal(contents, &schema); err != nil {
				t.Fatalf("decode schema: %v", err)
			}
			if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:contract-state-fixtures:"+version {
				t.Fatalf("schema identity = %q, %q", schema.Dialect, schema.ID)
			}
			if !slices.Equal(schema.Defs.OperationKind.Enum, operationKinds) || !slices.Equal(schema.Defs.ExitKind.Enum, exitKinds) || !slices.Equal(schema.Defs.Status.Enum, resultStatuses) {
				t.Fatalf("schema vocabulary does not match runner")
			}
		})
	}
}

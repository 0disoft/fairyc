package contractstatelayoutfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestContractStateLayoutSchemaV0IsValidJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "contract-state-layout", "schema-v0.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("schema JSON: %v", err)
	}
	if got := schema["$id"]; got != "https://fairyc.dev/spec/fixtures/contract-state-layout/schema-v0.json" {
		t.Fatalf("schema $id = %v", got)
	}
}

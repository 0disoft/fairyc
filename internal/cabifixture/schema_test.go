package cabifixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cabi"
	"github.com/0disoft/fairyc/internal/target"
)

func TestCABIJSONSchemaMatchesRunnerVocabulary(t *testing.T) {
	t.Parallel()
	contents, err := os.ReadFile(filepath.Join("..", "..", "spec", "fixtures", "c-abi", "schema-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Dialect    string   `json:"$schema"`
		ID         string   `json:"$id"`
		Required   []string `json:"required"`
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
		Defs struct {
			Signature struct {
				Properties struct {
					Result struct {
						OneOf []struct {
							Ref string `json:"$ref"`
						} `json:"oneOf"`
					} `json:"result"`
					Parameters struct {
						Items struct {
							Ref string `json:"$ref"`
						} `json:"items"`
					} `json:"parameters"`
				} `json:"properties"`
			} `json:"signature"`
			Expect struct {
				OneOf []struct {
					Properties map[string]struct {
						Enum []string `json:"enum"`
					} `json:"properties"`
				} `json:"oneOf"`
			} `json:"expect"`
			Location struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			} `json:"location"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:c-abi-fixtures:v0" {
		t.Fatalf("schema identity = %q, %q", schema.Dialect, schema.ID)
	}
	wantRequired := []string{"$schema", "cases", "contracts", "deferred", "schemaVersion", "target"}
	slices.Sort(schema.Required)
	if !slices.Equal(schema.Required, wantRequired) {
		t.Fatalf("schema required = %v, want %v", schema.Required, wantRequired)
	}
	wantTargets := []string{string(target.X8664LinuxGNU), string(target.X8664WindowsMSVC)}
	if !slices.Equal(schema.Properties["target"].Enum, wantTargets) {
		t.Fatalf("schema targets = %v, want %v", schema.Properties["target"].Enum, wantTargets)
	}
	if want := []string{"rdi", "rcx"}; !slices.Equal(schema.Defs.Location.Properties["hiddenPointerRegister"].Enum, want) {
		t.Fatalf("schema hidden result registers = %v, want %v", schema.Defs.Location.Properties["hiddenPointerRegister"].Enum, want)
	}
	typeRef := "../layout/schema-v0.json#/$defs/type"
	if len(schema.Defs.Signature.Properties.Result.OneOf) != 2 || schema.Defs.Signature.Properties.Result.OneOf[1].Ref != typeRef || schema.Defs.Signature.Properties.Parameters.Items.Ref != typeRef {
		t.Fatalf("layout type references drifted: %+v", schema.Defs.Signature.Properties)
	}
	var schemaCodes []string
	for _, variant := range schema.Defs.Expect.OneOf {
		if property, found := variant.Properties["errorCode"]; found {
			schemaCodes = property.Enum
		}
	}
	wantCodes := []string{
		string(cabi.ErrorUnsupportedTarget),
		string(cabi.ErrorLayoutRejected),
		string(cabi.ErrorInvalidCEnumBase),
		string(cabi.ErrorDepthLimit),
		string(cabi.ErrorInvalidType),
	}
	if !reflect.DeepEqual(schemaCodes, wantCodes) {
		t.Fatalf("schema error codes = %v, want %v", schemaCodes, wantCodes)
	}
}

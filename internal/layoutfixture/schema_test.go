package layoutfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLayoutSchemaIdentityAndErrorCodesMatchRunner(t *testing.T) {
	t.Parallel()
	schemaPath := filepath.Join("..", "..", "spec", "fixtures", "layout", "schema-v0.json")
	contents, err := os.ReadFile(schemaPath)
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
			Expect struct {
				OneOf []struct {
					Properties map[string]struct {
						Enum []string `json:"enum"`
					} `json:"properties"`
				} `json:"oneOf"`
			} `json:"expect"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:layout-fixtures:v0" {
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
	if len(schema.Defs.Expect.OneOf) != 2 {
		t.Fatalf("expectation variants = %d", len(schema.Defs.Expect.OneOf))
	}
	var schemaCodes []string
	for _, variant := range schema.Defs.Expect.OneOf {
		if property, found := variant.Properties["errorCode"]; found {
			schemaCodes = property.Enum
		}
	}
	wantCodes := []string{
		string(layout.ErrorInvalidType),
		string(layout.ErrorUnknownScalar),
		string(layout.ErrorDuplicateField),
		string(layout.ErrorEmptyAggregate),
		string(layout.ErrorInvalidArrayLength),
		string(layout.ErrorOverflow),
		string(layout.ErrorDepthLimit),
	}
	if !reflect.DeepEqual(schemaCodes, wantCodes) {
		t.Fatalf("schema error codes = %v, want %v", schemaCodes, wantCodes)
	}
	var raw struct {
		Defs map[string]json.RawMessage `json:"$defs"`
	}
	if err := json.Unmarshal(contents, &raw); err != nil {
		t.Fatal(err)
	}
	named := decodeKindDefinition(t, raw.Defs, "namedLeaf")
	unnamed := decodeKindDefinition(t, raw.Defs, "unnamedLeaf")
	array := decodeKindDefinition(t, raw.Defs, "array")
	structure := decodeKindDefinition(t, raw.Defs, "struct")
	if !reflect.DeepEqual(named.Enum, []string{string(layout.KindLanguageScalar), string(layout.KindCScalar), string(layout.KindCEnum)}) ||
		!reflect.DeepEqual(unnamed.Enum, []string{string(layout.KindPointer), string(layout.KindFunctionPtr)}) ||
		array.Const != string(layout.KindArray) || structure.Const != string(layout.KindStruct) {
		t.Fatalf("schema kind vocabulary = named %v, unnamed %v, array %q, struct %q", named.Enum, unnamed.Enum, array.Const, structure.Const)
	}
}

type kindDefinition struct {
	Properties map[string]struct {
		Enum  []string `json:"enum"`
		Const string   `json:"const"`
	} `json:"properties"`
}

func decodeKindDefinition(t *testing.T, definitions map[string]json.RawMessage, name string) struct {
	Enum  []string
	Const string
} {
	t.Helper()
	var definition kindDefinition
	if err := json.Unmarshal(definitions[name], &definition); err != nil {
		t.Fatalf("decode %s definition: %v", name, err)
	}
	kind, found := definition.Properties["kind"]
	if !found {
		t.Fatalf("%s definition has no kind", name)
	}
	return struct {
		Enum  []string
		Const string
	}{Enum: kind.Enum, Const: kind.Const}
}

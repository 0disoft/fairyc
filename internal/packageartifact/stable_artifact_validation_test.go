package packageartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/mir"
)

func TestStableArtifactReferenceCoverageAndOrder(t *testing.T) {
	local, first, second, missing := FunctionID{1}, FunctionID{2}, FunctionID{3}, FunctionID{4}
	calls := func(ids ...FunctionID) []StableBodyCall {
		result := make([]StableBodyCall, 0, len(ids))
		for _, id := range ids {
			result = append(result, StableBodyCall{Target: bodyInternalCallPrefix + id.String()})
		}
		return result
	}
	repeated := make([]FunctionID, 64)
	for i := range repeated {
		repeated[i] = first
	}
	unknown := fmt.Sprintf("stable package MIR reference %s is neither owned nor imported", missing)
	for _, test := range []struct {
		name       string
		imports    []FunctionID
		calls      []StableBodyCall
		operations []mir.Operation
		resources  []ContextResource
		foreign    []ContextForeignFunction
		want       string
	}{
		{name: "repeated imports are counted once", imports: []FunctionID{first, second}, calls: calls(append(repeated, second)...)},
		{name: "repetition cannot hide unused import", imports: []FunctionID{first, second}, calls: calls(repeated...), want: "stable package MIR imports are not the exact external reference set"},
		{name: "local references need no import", calls: calls(local, local)},
		{name: "unknown reference before unused import", imports: []FunctionID{first}, calls: calls(missing), want: unknown},
		{name: "import order before references", imports: []FunctionID{second, first}, calls: calls(missing), want: "stable package MIR imports are not sorted and unique"},
		{name: "duplicate imports before references", imports: []FunctionID{first, first}, calls: calls(missing), want: "stable package MIR imports are not sorted and unique"},
		{name: "zero import before references", imports: []FunctionID{{}}, calls: calls(missing), want: "stable package MIR imports are not sorted and unique"},
		{name: "owned import before references", imports: []FunctionID{local}, calls: calls(missing), want: fmt.Sprintf("stable package MIR FunctionID %s is both owned and imported", local)},
		{name: "callback before call references", calls: calls(first), operations: []mir.Operation{{Kind: mir.OperationCCallback, Variant: missing.String()}}, want: unknown},
		{name: "closure and cleanup share import usage", imports: []FunctionID{first, second}, operations: []mir.Operation{{Kind: mir.OperationClosureCreate, Variant: first.String()}}, resources: []ContextResource{{DropFunctionID: second}, {DropFunctionID: second}}},
		{name: "calls before resource references", calls: calls(missing), resources: []ContextResource{{DropFunctionID: first}}, want: unknown},
		{name: "foreign index at bound", calls: []StableBodyCall{{Target: bodyForeignCallPrefix + "1"}}, foreign: []ContextForeignFunction{{}}, want: "stable foreign call index 1 is outside the type context"},
		{name: "foreign index below bound", calls: []StableBodyCall{{Target: bodyForeignCallPrefix + "0"}}, foreign: []ContextForeignFunction{{}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			artifact := StableArtifact{Imports: test.imports,
				Functions: []StableBodyArtifact{{Calls: test.calls, Function: mir.Function{Blocks: []mir.Block{{Operations: test.operations}}}}},
				Context:   ContextDescriptor{Resources: test.resources, ForeignFunctions: test.foreign}}
			owned := map[FunctionID]struct{}{local: {}}
			before, err := json.Marshal(artifact)
			if err != nil {
				t.Fatal(err)
			}
			got := validateStableArtifactReferences(artifact, owned)
			if (got == nil) != (test.want == "") || got != nil && got.Error() != test.want {
				t.Fatalf("got %v, want %q", got, test.want)
			}
			after, err := json.Marshal(artifact)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) || !reflect.DeepEqual(owned, map[FunctionID]struct{}{local: {}}) {
				t.Fatal("reference validation mutated caller-owned artifact or ownership index")
			}
		})
	}
}

func TestStableArtifactBodyFailureOrder(t *testing.T) {
	artifact, err := DescribeStableArtifact(typedProjectionArtifact(t, 2), stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*StableArtifact) string
	}{
		{"package identity before coordinate", func(a *StableArtifact) string {
			a.Package = [32]byte{}
			a.Coordinate = [32]byte{}
			return "stable package MIR artifact has an empty PackageInstanceId"
		}},
		{"body order before identity", func(a *StableArtifact) string {
			a.Functions[1].ID = a.Functions[0].ID
			a.Functions[1].Package = [32]byte{}
			return "stable package MIR functions are not sorted and unique"
		}},
		{"body validation before imports", func(a *StableArtifact) string {
			a.Functions[0].ID = FunctionID{}
			a.Imports = []FunctionID{{}}
			return fmt.Sprintf("stable function %s: %v", a.Functions[0].ID, ValidateStableBody(a.Functions[0]))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := artifact
			changed.Functions = append([]StableBodyArtifact(nil), artifact.Functions...)
			want := test.change(&changed)
			before, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			got := ValidateStableArtifact(changed)
			if got == nil || got.Error() != want {
				t.Fatalf("got %v, want %q", got, want)
			}
			after, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("failed validation mutated artifact")
			}
		})
	}
}

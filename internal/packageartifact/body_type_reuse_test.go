package packageartifact

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestMapFunctionTypesReusesOnlyOriginalTypesWithinOneWalk(t *testing.T) {
	// A maps to B, while original B maps to C: cached outputs must not become
	// keys or cause a second transformation of repeated original A slots.
	a, b, c := hir.TypeI32, hir.TypeU8, hir.TypeBool
	for _, targets := range []map[hir.Type]hir.Type{{a: b, b: c}, {a: c, b: a}} {
		function := mir.Function{ParameterTypes: []hir.Type{a, b, a, b}}
		calls := make(map[hir.Type]int)
		err := mapFunctionTypes(&function, func(value hir.Type) (hir.Type, error) {
			calls[value]++
			return targets[value], nil
		})
		if err != nil || !reflect.DeepEqual(function.ParameterTypes, []hir.Type{targets[a], targets[b], targets[a], targets[b]}) {
			t.Fatalf("wrong original-type mapping or stale conversion bindings: %v", err)
		}
		if calls[a] != 1 || calls[b] != 1 || len(calls) != 2 {
			t.Fatalf("transform calls = %v, want one per original type", calls)
		}
	}
}

func TestMapFunctionTypesDoesNotHideLaterErrors(t *testing.T) {
	wantErr := errors.New("unresolved type binding")
	function := mir.Function{ParameterTypes: []hir.Type{hir.TypeI32, hir.TypeI32, hir.TypeU8, hir.TypeBool}}
	err := mapFunctionTypes(&function, func(value hir.Type) (hir.Type, error) {
		if value != hir.TypeI32 {
			return hir.TypeInvalid, wantErr
		}
		return hir.TypeBool, nil
	})
	if !errors.Is(err, wantErr) || !reflect.DeepEqual(function.ParameterTypes, []hir.Type{hir.TypeBool, hir.TypeBool, hir.TypeU8, hir.TypeBool}) {
		t.Fatalf("first failure or partial rewrite behavior changed: %v", err)
	}
}

func BenchmarkStableBodyRepeatedTypes(b *testing.B) {
	for _, depth := range []int{0, 16} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			artifact := typedProjectionArtifact(b, 1)
			value := artifact.Context.StructTypes[0].Type
			for range depth {
				value = hir.RawPointer(value, false)
			}
			function := artifact.Functions[0].Body
			function.Parameters = []mir.LocalID{1}
			function.ParameterTypes = []hir.Type{value}
			for i := range 256 {
				function.Blocks[0].Operations = append(function.Blocks[0].Operations, mir.Operation{
					Kind: mir.OperationAssign, Type: value, DestinationLocal: mir.LocalID(i + 2), Span: function.Span,
					Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 1, Type: value}},
				})
			}
			program := artifact.Context
			program.Functions = []mir.Function{function}
			artifact, err := Extract(program, artifact.PackageID, artifact.PackageIdentities, artifact.PackageCoordinates)
			if err != nil {
				b.Fatal(err)
			}
			body, err := DescribeStableLeafBody(artifact.Functions[0], artifact.StableContext, artifact.PackageCoordinates, stableBodyLimit)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				rebound, err := RebindStableLeafBody(body, artifact.StableContext, artifact.PackageCoordinates, 30000, 1, stableBodyLimit)
				if err != nil || len(rebound.Blocks[0].Operations) != 256 {
					b.Fatalf("rebind repeated body types: %v", err)
				}
			}
		})
	}
}

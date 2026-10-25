package packageartifact

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableRebindForeignBindingsStayPackageLocal(t *testing.T) {
	artifacts, coordinates := foreignRebindArtifacts(t, 2, 4)
	freshCoordinates := make(map[hir.PackageID]packagegraph.CoordinateID, len(coordinates))
	for packageID, coordinate := range coordinates {
		freshCoordinates[packageID+100] = coordinate
	}
	for _, firstForeign := range []hir.SymbolID{40000, 50000} {
		rebound, err := RebindStableArtifactsWithBindings(artifacts, freshCoordinates, 30000, firstForeign, 1)
		if err != nil {
			t.Fatal(err)
		}
		foreign := make(map[hir.SymbolID]mir.ForeignFunction)
		for _, declaration := range rebound.MIR.ForeignFunctions {
			foreign[declaration.Symbol] = declaration
		}
		expected := make(map[hir.SymbolID]string)
		for _, artifact := range artifacts {
			for _, body := range artifact.Functions {
				index, err := parseForeignCallTarget(body.Calls[0].Target)
				if err != nil {
					t.Fatal(err)
				}
				for _, binding := range rebound.Functions {
					if binding.Package == body.Package && binding.Name == body.Function.Name {
						expected[binding.Symbol] = artifact.Context.ForeignFunctions[index].LinkName
					}
				}
			}
		}
		if len(expected) != 8 {
			t.Fatalf("expected eight independently bound callers, got %d", len(expected))
		}
		for _, function := range rebound.MIR.Functions {
			callee := function.Blocks[0].Terminator.Callee
			if got := foreign[callee]; got.LinkName != expected[function.Symbol] || callee < firstForeign {
				t.Fatalf("%s calls foreign symbol %d (%s), want %s in fresh namespace", function.Name, callee, got.LinkName, expected[function.Symbol])
			}
		}
	}
}

func BenchmarkRebindStableArtifactsForeignContext(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			artifacts, coordinates := foreignRebindArtifacts(b, 1, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				rebound, err := RebindStableArtifactsWithBindings(artifacts, coordinates, 30000, 40000, 1)
				if err != nil || len(rebound.MIR.Functions) != count || len(rebound.MIR.ForeignFunctions) != count {
					b.Fatalf("rebind %d foreign callers: %v", count, err)
				}
			}
		})
	}
}

func foreignRebindArtifacts(t testing.TB, packageCount, count int) ([]StableArtifact, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	_, original, identities, coordinates := checkedWorkspaceProgram(t)
	spans := make([]hir.Span, 0, packageCount)
	seen := make(map[hir.PackageID]bool)
	for _, function := range original.Functions {
		if !seen[function.Span.Package] && len(spans) < packageCount {
			seen[function.Span.Package] = true
			spans = append(spans, function.Span)
		}
	}
	if len(spans) != packageCount {
		t.Fatalf("fixture needs %d function-owning packages, got %d", packageCount, len(spans))
	}
	program := mir.Program{}
	for packageIndex, span := range spans {
		prefix := hir.QualifiedModulePath(span.Package, "main")
		entity := func(kind string) string {
			return fmt.Sprintf("%s:%s:%d:%d", kind, hir.QualifiedSourcePath(span.Package, span.Source), span.StartByte, span.EndByte)
		}
		for i := range count {
			index := packageIndex*count + i
			callee := hir.SymbolID(20000 + index)
			program.ForeignFunctions = append(program.ForeignFunctions, mir.ForeignFunction{
				Symbol: callee, Name: fmt.Sprintf("%s::foreign_%04d", prefix, i), Span: span,
				LinkName: fmt.Sprintf("foreign_%d_%04d", packageIndex, i), ABI: "C", ResultType: hir.TypeVoid,
			})
			program.Functions = append(program.Functions, mir.Function{
				Symbol: hir.SymbolID(10000 + index), Name: fmt.Sprintf("%s::caller_%04d", prefix, i), Span: span,
				ResultType: hir.TypeI32, Entry: 0,
				UnsafeOperations: []mir.UnsafeOperation{{Entity: entity("call"), SourceSpanEntity: entity("span"), ScopeEntity: entity("unsafe-scope")}},
				Calls:            []mir.CallFact{{Call: span, ForeignABI: "C", UnsafeScope: span, NormalSuccessor: true}},
				Blocks: []mir.Block{
					{ID: 0, Terminator: mir.Terminator{
						Span: span, UnsafeScope: span, Entity: entity("call"),
						Kind: mir.TerminatorCall, Callee: callee, ForeignABI: "C", DestinationLocal: mir.NoLocal,
						Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock,
					}},
					{ID: 1, Terminator: mir.Terminator{
						Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
						ReturnValue: mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: int32(i)},
					}},
				},
			})
		}
	}
	extracted, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := make([]StableArtifact, len(extracted))
	for index, artifact := range extracted {
		artifacts[index], err = DescribeStableArtifact(artifact, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
	}
	return artifacts, coordinates
}

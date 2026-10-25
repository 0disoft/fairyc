package packageartifact

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestForeignIndicesResolveSortedDeclarationsAndRejectDrift(t *testing.T) {
	program, stable, coordinates := foreignIndexFixture(t, 32)
	for _, projected := range []bool{false, true} {
		t.Run(fmt.Sprintf("projected=%v", projected), func(t *testing.T) {
			required := make(map[hir.SymbolID]struct{})
			for _, foreign := range program.ForeignFunctions {
				required[foreign.Symbol] = struct{}{}
			}
			resolve := func(context ContextDescriptor) (map[hir.SymbolID]uint64, error) {
				if projected {
					return contextForeignIndicesForSymbols(program, context, coordinates, required)
				}
				return contextForeignIndices(program, context, coordinates)
			}
			resolved, err := resolve(stable)
			if err != nil || len(resolved) != len(program.ForeignFunctions) {
				t.Fatalf("resolve complete context: %v", err)
			}
			for _, foreign := range program.ForeignFunctions {
				declaration := stable.ForeignFunctions[resolved[foreign.Symbol]]
				if declaration.Package != coordinates[foreign.Span.Package] || declaration.LinkName != foreign.LinkName {
					t.Fatalf("foreign symbol %d resolved to the wrong package or declaration", foreign.Symbol)
				}
			}
			for _, change := range []string{"link name", "ABI", "parameters", "result", "missing"} {
				t.Run(change, func(t *testing.T) {
					changed := cloneContextDescriptor(stable)
					switch change {
					case "link name":
						changed.ForeignFunctions[0].LinkName += "_changed"
					case "ABI":
						changed.ForeignFunctions[0].ABI = "other"
					case "parameters":
						changed.ForeignFunctions[0].ParameterTypes = []string{"i32"}
					case "result":
						changed.ForeignFunctions[0].ResultType = "usize"
					case "missing":
						changed.ForeignFunctions = changed.ForeignFunctions[1:]
					}
					if _, err := resolve(changed); err == nil {
						t.Fatal("accepted absent or mismatched foreign declaration")
					}
				})
			}
			if projected {
				// A projected context need only contain the required foreign calls.
				for symbol := range required {
					if resolved[symbol] != 0 {
						delete(required, symbol)
					}
				}
				subset := ContextDescriptor{ForeignFunctions: stable.ForeignFunctions[:1]}
				if got, err := resolve(subset); err != nil || len(got) != 1 {
					t.Fatalf("resolve required subset: %v", err)
				}
				required[hir.SymbolID(999999)] = struct{}{}
				if _, err := resolve(subset); err == nil {
					t.Fatal("accepted an unresolved required symbol")
				}
			}
		})
	}
}

func BenchmarkContextForeignIndices(b *testing.B) {
	for _, count := range []int{256, 1024} {
		program, stable, coordinates := foreignIndexFixture(b, count)
		required := make(map[hir.SymbolID]struct{}, count)
		for _, foreign := range program.ForeignFunctions {
			required[foreign.Symbol] = struct{}{}
		}
		for _, projected := range []bool{false, true} {
			b.Run(fmt.Sprintf("%d/projected=%v", count, projected), func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					var result map[hir.SymbolID]uint64
					var err error
					if projected {
						result, err = contextForeignIndicesForSymbols(program, stable, coordinates, required)
					} else {
						result, err = contextForeignIndices(program, stable, coordinates)
					}
					if err != nil || len(result) != count {
						b.Fatalf("resolve %d foreign declarations: %v", count, err)
					}
				}
			})
		}
	}
}

func foreignIndexFixture(t testing.TB, count int) (mir.Program, ContextDescriptor, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{7: {1}, 8: {2}}
	program := mir.Program{}
	for i := range count {
		packageID := hir.PackageID(7 + i%2)
		// The same logical names in different packages must remain distinct.
		name := fmt.Sprintf("call_%06d", i/2)
		program.ForeignFunctions = append(program.ForeignFunctions, mir.ForeignFunction{
			Symbol: hir.SymbolID(10000 + i), Name: hir.QualifiedModulePath(packageID, "ffi") + "::" + name,
			Span: hir.Span{Package: packageID, Source: "ffi.fy"}, LinkName: fmt.Sprintf("ffi_%d_%s", packageID, name),
			ABI: "C", ParameterTypes: []hir.Type{hir.TypeUSize}, ResultType: hir.TypeI32,
		})
	}
	stable, err := DescribeContext(program, coordinates, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Source order deliberately differs from canonical package/name order.
	slices.Reverse(program.ForeignFunctions)
	return program, stable, coordinates
}

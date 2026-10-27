package packageartifact

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// Preserve the old scan as an independent oracle for lookup and error order.
func scannedStableDropBindings(process mir.Program, stable ContextDescriptor, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.SymbolID]FunctionID, error) {
	bindings := make(map[hir.SymbolID]FunctionID)
	for _, resource := range process.ResourceTypes {
		if resource.DropSymbol == hir.NoSymbol {
			continue
		}
		name, err := stableType(resource.Type, coordinates)
		if err != nil {
			return nil, err
		}
		position := slices.IndexFunc(stable.Resources, func(candidate ContextResource) bool { return candidate.Type == name })
		if position < 0 || stable.Resources[position].DropFunctionID == (FunctionID{}) {
			return nil, fmt.Errorf("resource %q has no stable drop-function binding", resource.Type)
		}
		bindings[resource.DropSymbol] = stable.Resources[position].DropFunctionID
	}
	return bindings, nil
}

func TestStableContextDropBindingsPreserveScanBehavior(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*mir.Program, *ContextDescriptor, map[hir.PackageID]packagegraph.CoordinateID)
	}{
		{"all drops", func(*mir.Program, *ContextDescriptor, map[hir.PackageID]packagegraph.CoordinateID) {}},
		{"reordered context", func(_ *mir.Program, stable *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID) {
			slices.Reverse(stable.Resources)
		}},
		{"missing resource", func(_ *mir.Program, stable *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID) {
			stable.Resources = stable.Resources[1:]
		}},
		{"missing drop", func(_ *mir.Program, stable *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID) {
			stable.Resources[0].DropFunctionID = FunctionID{}
		}},
		{"first duplicate wins", func(_ *mir.Program, stable *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID) {
			duplicate := stable.Resources[0]
			duplicate.DropFunctionID = FunctionID{255}
			stable.Resources = append([]ContextResource{duplicate}, stable.Resources...)
		}},
		{"invalid first duplicate", func(_ *mir.Program, stable *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID) {
			duplicate := stable.Resources[0]
			duplicate.DropFunctionID = FunctionID{}
			stable.Resources = append([]ContextResource{duplicate}, stable.Resources...)
		}},
		{"empty drops", func(process *mir.Program, _ *ContextDescriptor, coordinates map[hir.PackageID]packagegraph.CoordinateID) {
			for index := range process.ResourceTypes {
				process.ResourceTypes[index].DropSymbol = hir.NoSymbol
			}
			clear(coordinates) // Skipped resources must not attempt type resolution.
		}},
		{"missing package", func(_ *mir.Program, _ *ContextDescriptor, coordinates map[hir.PackageID]packagegraph.CoordinateID) {
			clear(coordinates)
		}},
		{"mixed drops", func(process *mir.Program, _ *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID) {
			for index := range process.ResourceTypes {
				if index%2 == 0 {
					process.ResourceTypes[index].DropSymbol = hir.NoSymbol
				}
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stable, coordinates, symbols := resourceRebindFixture(32)
			process, err := RebindContextDescriptor(stable, coordinates, symbols, 90000)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&process, &stable, coordinates)
			want, wantErr := scannedStableDropBindings(process, stable, coordinates)
			got, gotErr := bindingsForStableContext(process, stable, coordinates)
			if !reflect.DeepEqual(got, want) || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
				t.Fatalf("got %v / %v, want %v / %v", got, gotErr, want, wantErr)
			}
			if gotErr == nil {
				for symbol := range got {
					got[symbol] = FunctionID{}
				}
				again, err := bindingsForStableContext(process, stable, coordinates)
				if !reflect.DeepEqual(again, want) || err != nil {
					t.Fatal("caller mutation changed a later binding lookup")
				}
			}
		})
	}
}

func BenchmarkStableContextDropBindings(b *testing.B) {
	for _, count := range []int{512, 2048, 8192} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			stable, coordinates, symbols := resourceRebindFixture(count)
			process, err := RebindContextDescriptor(stable, coordinates, symbols, 90000)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				bindings, err := bindingsForStableContext(process, stable, coordinates)
				if err != nil || len(bindings) != count {
					b.Fatalf("bindings=%d, want %d, error=%v", len(bindings), count, err)
				}
			}
		})
	}
}

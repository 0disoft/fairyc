package packageartifact

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestReboundResourceBindingsRoundTripMixedDrops(t *testing.T) {
	descriptor, coordinates, symbols := resourceRebindFixture(64)
	for index := range descriptor.Resources {
		if index%3 == 0 {
			descriptor.Resources[index].DropKind = string(hir.ResourceDropEmpty)
			descriptor.Resources[index].DropFunctionID = FunctionID{}
		} else {
			descriptor.Resources[index].DropEffects = []string{string(hir.EffectFilesystem)}
		}
	}
	program, err := RebindContextDescriptor(descriptor, coordinates, symbols, 90000)
	if err != nil {
		t.Fatal(err)
	}
	bindings := make(map[hir.SymbolID]FunctionID)
	for index, resource := range program.ResourceTypes {
		original := descriptor.Resources[index]
		if resource.DropKind == hir.ResourceDropEmpty {
			if resource.DropSymbol != hir.NoSymbol {
				t.Fatalf("empty resource %d gained a drop symbol", index)
			}
			continue
		}
		if resource.DropSymbol != symbols[original.DropFunctionID] {
			t.Fatalf("resource %d rebound to the wrong drop function", index)
		}
		bindings[resource.DropSymbol] = original.DropFunctionID
	}
	roundTrip, err := DescribeContext(program, coordinates, bindings)
	if err != nil || !reflect.DeepEqual(roundTrip, descriptor) {
		t.Fatalf("mixed drop context changed after rebinding: %v", err)
	}
}

func BenchmarkRebindContextResourceDrops(b *testing.B) {
	for _, count := range []int{512, 2048, 8192} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			descriptor, coordinates, symbols := resourceRebindFixture(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				program, err := RebindContextDescriptor(descriptor, coordinates, symbols, 90000)
				if err != nil || len(program.ResourceTypes) != count {
					b.Fatalf("rebind %d resource drop bindings: %v", count, err)
				}
			}
		})
	}
}

func resourceRebindFixture(count int) (ContextDescriptor, map[hir.PackageID]packagegraph.CoordinateID, map[FunctionID]hir.SymbolID) {
	coordinate := packagegraph.CoordinateID{1}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{77: coordinate}
	symbols := make(map[FunctionID]hir.SymbolID, count)
	descriptor := ContextDescriptor{Resources: make([]ContextResource, count)}
	for index := range count {
		ordinal := index + 1
		id := FunctionID{byte(ordinal), byte(ordinal >> 8), byte(ordinal >> 16)}
		symbols[id] = hir.SymbolID(10000 + index)
		descriptor.Resources[index] = ContextResource{
			Type:     fmt.Sprintf("pkg[%s]::types::Resource_%06d", coordinate, index),
			DropKind: string(hir.ResourceDropNonEmpty), DropFunctionID: id,
		}
	}
	return descriptor, coordinates, symbols
}

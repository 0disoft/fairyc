package packageartifact

import (
	"fmt"
	"maps"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestContextRebindPreservesErrorOrderAndInputs(t *testing.T) {
	t.Parallel()
	unknown := fmt.Sprintf("pkg[%s]::types::Missing", packagegraph.CoordinateID{9})
	type mutation func(*ContextDescriptor, map[hir.PackageID]packagegraph.CoordinateID, map[FunctionID]hir.SymbolID, *hir.SymbolID)
	cases := []struct {
		name   string
		mutate mutation
		want   string
	}{
		{"descriptor before symbol base", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			d.Structs[0].Name = ""
			*base = hir.NoSymbol
		}, "stable type-context has an invalid struct"},
		{"base before function bindings", func(_ *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, symbols map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			*base = hir.NoSymbol
			symbols[FunctionID{}] = 1
		}, "stable type-context rebind requires a foreign symbol base"},
		{"bindings before package lookup", func(_ *ContextDescriptor, coordinates map[hir.PackageID]packagegraph.CoordinateID, symbols map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			symbols[FunctionID{}] = 1
			delete(coordinates, 3)
		}, "stable type-context rebind has an invalid function symbol binding"},
		{"duplicate function symbols", func(_ *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, symbols map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			symbols[FunctionID{1}] = 801
		}, "function symbol 801 is duplicated by"},
		{"struct before enum", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			d.Structs[0].Fields[0].Type = unknown
			d.Enums[0].Variants[0].Fields[0].Type = unknown
		}, "rebind struct "},
		{"enum before resource drop", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, symbols map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			d.Enums[0].Variants[0].Fields[0].Type = unknown
			clear(symbols)
		}, "rebind enum "},
		{"resource field before drop", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, symbols map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			d.Resources[0].Fields[0].Type = unknown
			clear(symbols)
		}, "rebind resource "},
		{"unknown effect before symbol base", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			d.Resources[0].DropEffects = []string{"unknown"}
			*base = hir.NoSymbol
		}, "stable resource "},
		{"resource drop before foreign collision", func(_ *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, symbols map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			clear(symbols)
			symbols[FunctionID{1}] = *base
		}, "stable resource "},
		{"foreign owner before collision", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			d.ForeignFunctions[0].Package = packagegraph.CoordinateID{9}
			*base = 801
		}, "PackageCoordinateId "},
		{"foreign collision before parameter", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			*base = 801
			d.ForeignFunctions[0].ParameterTypes[0] = unknown
		}, "foreign symbol 801 collides with"},
		{"later foreign collision", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, symbols map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			next := d.ForeignFunctions[0]
			next.Name, next.LinkName = "ffi::zlast", "zlast"
			d.ForeignFunctions = append(d.ForeignFunctions, next)
			symbols[FunctionID{1}] = *base + 1
		}, "foreign symbol 902 collides with"},
		{"foreign parameter before result", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			d.ForeignFunctions[0].ParameterTypes[0] = unknown
			d.ForeignFunctions[0].ResultType = unknown
		}, "rebind foreign function \"ffi::write\" parameter:"},
		{"foreign result", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			d.ForeignFunctions[0].ResultType = unknown
		}, "rebind foreign function \"ffi::write\" result:"},
		{"foreign symbol overflow", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, base *hir.SymbolID) {
			next := d.ForeignFunctions[0]
			next.Name, next.LinkName = "ffi::zlast", "zlast"
			d.ForeignFunctions = append(d.ForeignFunctions, next)
			*base = math.MaxInt
		}, "stable foreign function symbol namespace overflows"},
		{"round trip remains mandatory", func(d *ContextDescriptor, _ map[hir.PackageID]packagegraph.CoordinateID, _ map[FunctionID]hir.SymbolID, _ *hir.SymbolID) {
			d.Structs[0].ReprC = false
			d.Structs[0].Fields = []ContextField{}
		}, "rebound stable type context does not round-trip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			context, coordinates, bindings := stableContextFixture(3, 41, 51)
			descriptor, err := DescribeContext(context, coordinates, bindings)
			if err != nil {
				t.Fatal(err)
			}
			symbols := map[FunctionID]hir.SymbolID{bindings[41]: 801}
			base := hir.SymbolID(901)
			tc.mutate(&descriptor, coordinates, symbols, &base)
			before := cloneContextDescriptor(descriptor)
			beforeCoordinates, beforeSymbols := maps.Clone(coordinates), maps.Clone(symbols)
			program, err := RebindContextDescriptor(descriptor, coordinates, symbols, base)
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("error = %v, want prefix %q", err, tc.want)
			}
			if !reflect.DeepEqual(program, mir.Program{}) {
				t.Fatal("failed rebind exposed a partial program")
			}
			if !reflect.DeepEqual(descriptor, before) || !maps.Equal(coordinates, beforeCoordinates) || !maps.Equal(symbols, beforeSymbols) {
				t.Fatal("failed rebind changed caller-owned input")
			}
		})
	}
}

func TestContextRebindResultDoesNotAliasInputs(t *testing.T) {
	t.Parallel()
	context, coordinates, bindings := stableContextFixture(3, 41, 51)
	descriptor, err := DescribeContext(context, coordinates, bindings)
	if err != nil {
		t.Fatal(err)
	}
	before := cloneContextDescriptor(descriptor)
	symbols := map[FunctionID]hir.SymbolID{bindings[41]: 801}
	beforeCoordinates, beforeSymbols := maps.Clone(coordinates), maps.Clone(symbols)
	program, err := RebindContextDescriptor(descriptor, coordinates, symbols, 901)
	if err != nil {
		t.Fatal(err)
	}
	program.StructTypes[0].Fields[0].Name = "changed"
	program.EnumTypes[0].Variants[0].Fields[0].Name = "changed"
	program.ResourceTypes[0].Fields[0].Name = "changed"
	program.ResourceTypes[0].DropEffects[0] = hir.Effect("changed")
	program.ForeignFunctions[0].ParameterTypes[0] = hir.TypeInvalid
	if !reflect.DeepEqual(descriptor, before) || !maps.Equal(coordinates, beforeCoordinates) || !maps.Equal(symbols, beforeSymbols) {
		t.Fatal("rebound program aliases caller-owned input")
	}
}

func TestContextRebindEmptyAndLastForeignSymbol(t *testing.T) {
	t.Parallel()
	program, err := RebindContextDescriptor(ContextDescriptor{}, nil, nil, math.MaxInt)
	if err != nil || !reflect.DeepEqual(program, mir.Program{}) {
		t.Fatalf("empty context = %+v, %v", program, err)
	}
	_, descriptor, coordinates := foreignIndexFixture(t, 1)
	program, err = RebindContextDescriptor(descriptor, coordinates, nil, math.MaxInt)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.ForeignFunctions) != 1 || program.ForeignFunctions[0].Symbol != math.MaxInt {
		t.Fatalf("last valid foreign symbol = %+v", program.ForeignFunctions)
	}
}

func BenchmarkRebindContextDeclarations(b *testing.B) {
	for _, count := range []int{256, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			_, descriptor, coordinates := foreignIndexFixture(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				program, err := RebindContextDescriptor(descriptor, coordinates, nil, 50000)
				if err != nil || len(program.ForeignFunctions) != count {
					b.Fatalf("rebind %d foreign declarations: %v", count, err)
				}
			}
		})
	}
}

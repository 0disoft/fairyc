package packageartifact

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestExtractContextPreservesDeclarationsAndIsolation(t *testing.T) {
	artifact := typedProjectionArtifact(t, 1)
	program := artifact.Context
	program.Functions = []mir.Function{artifact.Functions[0].Body}
	before := mir.Clone(program)
	want := mir.Clone(program)
	want.Functions = nil
	context := contextOf(program)
	if !reflect.DeepEqual(context, want) {
		t.Fatal("context declarations or nil function representation changed")
	}
	if !reflect.DeepEqual(mir.Clone(program), before) {
		t.Fatal("extracting context changed its source program")
	}
	context.StructTypes[0].Fields[0].Name = "changed"
	if program.StructTypes[0].Fields[0].Name == "changed" {
		t.Fatal("context fields alias the source program")
	}
}

func BenchmarkExtractAllContextBodies(b *testing.B) {
	for _, packages := range []int{1, 8, 32} {
		b.Run(fmt.Sprintf("%d_packages", packages), func(b *testing.B) {
			program, identities, coordinates := extractionContextFixture(packages, 16)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				artifacts, err := ExtractAll(program, identities, coordinates)
				if err != nil || len(artifacts) != packages {
					b.Fatalf("extract %d packages: %v", packages, err)
				}
			}
		})
	}
}

func extractionContextFixture(packages, functionsPerPackage int) (mir.Program, map[hir.PackageID]packagegraph.InstanceID, map[hir.PackageID]packagegraph.CoordinateID) {
	program := mir.Program{}
	identities := make(map[hir.PackageID]packagegraph.InstanceID, packages)
	coordinates := make(map[hir.PackageID]packagegraph.CoordinateID, packages)
	for p := range packages {
		owner := hir.PackageID(100 + p)
		identities[owner] = packagegraph.InstanceID{1, byte(p)}
		coordinates[owner] = packagegraph.CoordinateID{2, byte(p)}
		for f := range functionsPerPackage {
			program.Functions = append(program.Functions, mir.Function{
				Symbol: hir.SymbolID(1000 + p*functionsPerPackage + f),
				Name:   hir.QualifiedModulePath(owner, fmt.Sprintf("main::leaf_%d", f)),
				Span:   hir.Span{Package: owner}, ResultType: hir.TypeI32, Entry: 0,
				Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{
					Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
					ReturnValue: mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: 42},
				}}},
			})
		}
	}
	return program, identities, coordinates
}

package packageartifact

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func BenchmarkDescribeStableArtifact(b *testing.B) {
	for _, count := range []int{64, 256, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			_, program, identities, coordinates := checkedWorkspaceProgram(b)
			span := program.Functions[0].Span
			program.Functions = make([]mir.Function, count)
			for i := range count {
				program.Functions[i] = mir.Function{
					Symbol: hir.SymbolID(10000 + i),
					Name:   fmt.Sprintf("%s::bench_%d", hir.QualifiedModulePath(span.Package, "main"), i),
					Span:   span, ResultType: hir.TypeI32, Entry: 0,
					Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{
						Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
						ReturnValue: mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: int32(i)},
					}}},
				}
			}
			artifact, err := Extract(program, span.Package, identities, coordinates)
			if err != nil {
				b.Fatal(err)
			}
			if _, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
				if err != nil || len(stable.Functions) != count {
					b.Fatalf("project %d functions: %v", count, err)
				}
			}
		})
	}
}

package packageartifact

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func BenchmarkProcessCodecDecode(b *testing.B) {
	for _, branches := range []int{64, 256} {
		b.Run(fmt.Sprint(branches), func(b *testing.B) {
			_, program, identities, coordinates := checkedWorkspaceProgram(b)
			artifacts, err := ExtractAll(program, identities, coordinates)
			if err != nil {
				b.Fatal(err)
			}
			artifact := artifacts[0]
			body := &artifact.Functions[0].Body
			if body.ResultType != hir.TypeI32 {
				b.Fatalf("benchmark requires an i32 function, got %s", body.ResultType)
			}
			// Keep the real package signature/context and vary only its verified CFG.
			*body = mir.Function{
				Symbol: body.Symbol, Name: body.Name, Span: body.Span,
				Parameters: body.Parameters, ParameterTypes: body.ParameterTypes,
				ResultType: body.ResultType, Effects: body.Effects,
				Entry: 0, Blocks: make([]mir.Block, branches*3+1),
			}
			for i := range branches {
				entry := mir.BlockID(i * 3)
				body.Blocks[entry] = mir.Block{ID: entry, Terminator: mir.Terminator{
					Kind: mir.TerminatorSwitch, SwitchKind: "bool", UnwindSuccessor: mir.NoBlock,
					SwitchOperands: []mir.Operand{{Kind: mir.OperandI32, Type: hir.TypeBool, I32: 1}},
					Successors:     []mir.BlockID{entry + 1, entry + 2},
				}}
				for _, offset := range []mir.BlockID{1, 2} {
					body.Blocks[entry+offset] = mir.Block{ID: entry + offset, Terminator: mir.Terminator{
						Kind: mir.TerminatorGoto, UnwindSuccessor: mir.NoBlock, Successors: []mir.BlockID{entry + 3},
					}}
				}
			}
			last := mir.BlockID(len(body.Blocks) - 1)
			body.Blocks[last] = mir.Block{ID: last, Terminator: mir.Terminator{
				Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
				ReturnValue: mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32},
			}}
			codec, err := NewProcessCodec(processArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			encoded, err := codec.Encode(artifact)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				decoded, err := codec.Decode(encoded)
				if err != nil || len(decoded.Functions) != len(artifact.Functions) {
					b.Fatalf("decode package: %v", err)
				}
			}
		})
	}
}

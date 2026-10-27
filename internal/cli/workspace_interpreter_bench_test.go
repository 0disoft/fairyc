package cli

import (
	"fmt"
	"io"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
)

// Include all per-suite preparation; source compilation is outside the timer.
func BenchmarkInterpreterWorkspaceTests(b *testing.B) {
	for _, count := range []int{32, 128} {
		b.Run(fmt.Sprintf("tests=%d", count), func(b *testing.B) {
			program := mir.Program{}
			tests := make([]workspaceTest, count)
			for i := range tests {
				name := fmt.Sprintf("main::test_%04d", i)
				program.Functions = append(program.Functions, mir.Function{
					Symbol: hir.SymbolID(i + 1), Name: name, ResultType: hir.TypeVoid, Entry: 0,
					Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{
						Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock,
					}}},
				})
				tests[i] = workspaceTest{display: name, entry: name}
			}
			b.ReportAllocs()
			for b.Loop() {
				outcomes, code := runInterpreterWorkspaceTests(program, tests, interpreter.Options{}, io.Discard)
				if code != ExitSuccess || len(outcomes) != count {
					b.Fatalf("test suite: exit=%d, outcomes=%d", code, len(outcomes))
				}
			}
		})
	}
}

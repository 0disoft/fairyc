package compiler_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestSharedReborrowArtifactAndInterpreter(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "shared_reborrow", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(data))})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("execute: %v %+v", err, outcome)
	}
	for name, mutate := range map[string]func(*mir.Operation){
		"exclusive result":   func(op *mir.Operation) { op.Type = op.OwnerType },
		"shared source":      func(op *mir.Operation) { op.OwnerType = op.Type },
		"different referent": func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeBool) },
		"missing operand":    func(op *mir.Operation) { op.Operands = nil },
	} {
		t.Run(name, func(t *testing.T) {
			program, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == mir.OperationSharedReborrow && !found {
							mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(program)) == 0 {
				t.Fatal("malformed reborrow accepted or operation absent")
			}
		})
	}
}

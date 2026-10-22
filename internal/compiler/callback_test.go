package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const boundedCallbackSource = `module main;
unsafe extern C fn i32 visit(i32 value,unsafe extern C fn i32(i32,ptr mut void) callback,ptr mut void data);
unsafe fn i32 handle(i32 value,ptr mut void data){return value+1;}
fn i32 main(){return unsafe{visit(41,handle,null)};}`

func callbackMIR(t *testing.T) mir.Program {
	t.Helper()
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", boundedCallbackSource)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestBoundedCallbackEmission(t *testing.T) {
	program := callbackMIR(t)
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{"typedef int32_t (*fy_c_callback_i32_userdata)(int32_t, void *);", "fy_cb_", "= &fy_cb_", "fy_c_callback_i32_userdata fy_a_1"} {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("missing %q", fragment)
			}
		}
	}
	if _, err := interpreter.Execute(program, "main::main"); err == nil {
		t.Fatal("interpreter fabricated a callback integration")
	}
}

func TestBoundedCallbackMIRRejectsEscape(t *testing.T) {
	for _, mode := range []string{"target", "duplicate-use", "null", "overwrite"} {
		t.Run(mode, func(t *testing.T) {
			program := callbackMIR(t)
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					block := &program.Functions[fi].Blocks[bi]
					for oi := range block.Operations {
						op := &block.Operations[oi]
						if op.Kind != mir.OperationCCallback {
							continue
						}
						switch mode {
						case "target":
							op.Symbol = hir.NoSymbol
						case "duplicate-use":
							block.Operations = append(block.Operations, mir.Operation{Kind: mir.OperationAssign, Symbol: hir.NoSymbol, DestinationLocal: 999, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: op.DestinationLocal}}})
						case "null":
							block.Terminator.Arguments[len(block.Terminator.Arguments)-2] = mir.Operand{Kind: mir.OperandNull, Type: hir.TypeCCallbackI32UserData}
						case "overwrite":
							block.Operations = append(block.Operations, *op)
						}
						goto mutated
					}
				}
			}
		mutated:
			if len(mir.Verify(program)) == 0 {
				t.Fatal("forged callback MIR verified")
			}
		})
	}
}

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

const exclusiveSubviewSource = `module main;
fn ref mut u8[] tail(ref mut u8[] view) { return view[1..]; }
fn void fill(ref mut u8[] view) { view[0] = 9; }
fn void read(ref u8[] view) { assert(view.len > 0); }
fn i32 main() {
    var u8[4] data = [1,2,3,4];
    let ref mut u8[] view = ref mut data;
    let ref mut u8[] child = view[1..3];
    child[0] = 7;
    let ref mut u8[] leaf = child[..1];
    fill(leaf);
    assert(child[0] == 9);
    assert(view[1] == 9);
    let ref mut u8[] returned = tail(view);
    fill(returned);
    read(returned);
    let ref mut u8[] whole = view[..];
    let usize end = whole.len;
    let ref mut u8[] empty = whole[end..end];
    assert(empty.len == 0);
    whole[3] = 8;
    assert(view[3] == 8);
    var usize order = 0;
    let ref mut u8[] ordered = view[{ order = order * 10 + 1; 1 }..{ order = order * 10 + 2; 3 }];
    assert(order == 12);
    fill(ordered);
    assert(view[1] == 9);
    return 0;
}
`

func TestExclusiveSubviewArtifactAndInterpreter(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", exclusiveSubviewSource)})
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
	if _, err := cbackend.Emit(decoded, "main::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
		t.Fatalf("C emission after artifact round trip: %v", err)
	}
	for name, mutate := range map[string]func(*mir.Operation){
		"shared result":   func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeByteSlice) },
		"shared owner":    func(op *mir.Operation) { op.OwnerType = hir.SharedBorrow(hir.TypeByteSlice) },
		"missing operand": func(op *mir.Operation) { op.Operands = nil },
		"unrelated local": func(op *mir.Operation) { op.Operands[0].Local = op.DestinationLocal },
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
						if op.Kind == mir.OperationViewSlice && !found {
							mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(program)) == 0 {
				t.Fatal("malformed slice accepted or absent")
			}
		})
	}
}

func TestExclusiveSubviewBounds(t *testing.T) {
	for _, bounds := range []string{"3..2", "0..5", "5..", "..5"} {
		t.Run(bounds, func(t *testing.T) {
			source := strings.Replace(exclusiveSubviewSource, "view[1..3]", "view["+bounds+"]", 1)
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			outcome, err := interpreter.Execute(checked.MIR, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "bounds" {
				t.Fatalf("bounds: %v %+v", err, outcome)
			}
		})
	}
}

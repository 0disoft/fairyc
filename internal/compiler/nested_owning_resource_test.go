package compiler_test

import (
	"fmt"
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

const nestedOwningPureSource = `module main;
resource Leaf{i32 value;cleanup{}}
resource Empty{cleanup{}}
fn Option<Result<Leaf,Empty>> make(){return .some(.ok(Leaf{.value=42}));}
fn Option<Result<Leaf,Empty>> forward(Option<Result<Leaf,Empty>> value){return value;}
fn Result<Leaf,Empty> extract(Option<Result<Leaf,Empty>> value){return switch(value){case .some(inner):{inner}case .none:{assert(false);}};}
fn i32 main(){
 {let Option<Option<Leaf>> outer_none=.none;}
 {let Option<Option<Leaf>> inner_none=.some(.none);}
 {let Option<Result<Leaf,Empty>> leaf=forward(make());let Option<Result<Leaf,Empty>> moved=leaf;}
 {let Option<Result<Leaf,Empty>> empty=.some(.err(Empty{}));}
 {let Result<Option<Leaf>,Result<Leaf,Empty>> left=.ok(.some(Leaf{.value=1}));discard left;}
 {let Result<Option<Leaf>,Result<Leaf,Empty>> right=.err(.ok(Leaf{.value=2}));discard right;}
 {let Result<Option<Leaf>,Result<Leaf,Empty>> zero=.err(.err(Empty{}));discard zero;}
 {let Result<Leaf,Empty> extracted=extract(make());assert(switch(extracted){case .ok(leaf):{leaf.value==42}case .err(_):{false}});}
 {var Option<Option<Leaf>> replaced=.some(.some(Leaf{.value=3}));replaced=.some(.some(Leaf{.value=4}));}
 return 42;
}`

func nestedOwningMIR(t *testing.T) mir.Program {
	t.Helper()
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", nestedOwningPureSource)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	b, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(b, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestNestedOwningResourceMoveDropAndCodec(t *testing.T) {
	program := nestedOwningMIR(t)
	out, err := interpreter.Execute(program, "main::main")
	if err != nil || out.Value != 42 {
		t.Fatalf("interpreter: %+v %v", out, err)
	}
	var names []string
	for _, e := range out.DropTrace {
		names = append(names, e.ResourceType.String())
	}
	want := "main::Leaf,main::Empty,main::Leaf,main::Leaf,main::Empty,main::Leaf,main::Leaf,main::Leaf"
	if strings.Join(names, ",") != want {
		t.Fatalf("active payload drops: %v", names)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0DropTraceProfile(), cbackend.Stage0WindowsMSVCDropTraceProfile()} {
		if _, err := cbackend.Emit(program, "main::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	// The ownership bit on a nested payload extraction must survive cached MIR.
	found := false
	for fi := range program.Functions {
		for bi := range program.Functions[fi].Blocks {
			for oi := range program.Functions[fi].Blocks[bi].Operations {
				op := &program.Functions[fi].Blocks[bi].Operations[oi]
				if op.Kind == mir.OperationOptionPayload && op.MovePayload && strings.HasPrefix(op.Type.String(), "Result<") {
					op.MovePayload = false
					found = true
				}
			}
		}
	}
	if !found || len(mir.Verify(program)) == 0 {
		t.Fatal("forged nested payload copy admitted")
	}
	if _, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
		t.Fatal("forged nested payload copy emitted")
	}
}

func TestNestedOwningResourceDepthBoundary(t *testing.T) {
	for _, depth := range []int{8, 9} {
		typ := "Leaf"
		init := "Leaf{}"
		for i := 0; i < depth; i++ {
			typ = "Option<" + typ + ">"
			init = ".some(" + init + ")"
		}
		text := fmt.Sprintf("module main;resource Leaf{cleanup{}} fn i32 main(){let %s value=%s;return 42;}", typ, init)
		checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
		if err != nil || !checked.Verified {
			t.Fatalf("source depth %d: %v %+v", depth, err, checked.Semantic.Diagnostics)
		}
		_, err = cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
		if (depth == 8) != (err == nil) {
			t.Fatalf("depth %d: %v", depth, err)
		}
	}
}

func TestNestedOwningResourceClosedBoundaries(t *testing.T) {
	for name, text := range map[string]string{
		"use-after-move":        strings.Replace(nestedOwningPureSource, "moved=leaf;", "moved=leaf;discard leaf;", 1),
		"partial-resource-move": `module main;resource Leaf{cleanup{}}resource Parent{Leaf child;cleanup{}}fn i32 main(){let Parent p=Parent{.child=Leaf{}};let Leaf child=p.child;return 42;}`,
		"conditional-root":      `module main;resource Leaf{cleanup{}}fn void consume(Option<Option<Leaf>> v){}fn void f(bool b){let Option<Option<Leaf>> v=.some(.some(Leaf{}));if(b){consume(v);}consume(v);}fn i32 main(){f(true);return 42;}`,
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err == nil && r.Verified {
				t.Fatal("closed ownership form admitted")
			}
		})
	}
	for name, text := range map[string]string{
		"owning-field":        `module main;resource Leaf{cleanup{}}resource Parent{Option<Leaf> child;cleanup{}}fn i32 main(){let Parent p=Parent{.child=.some(Leaf{})};return 42;}`,
		"unsupported-sibling": `module main;resource Leaf{cleanup{}}fn i32 main(){let Option<Result<Leaf,f64>> p=.some(.ok(Leaf{}));return 42;}`,
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || !r.Verified {
				t.Fatalf("source: %v %+v", err, r.Semantic.Diagnostics)
			}
			if _, err := cbackend.Emit(r.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("unsupported representation admitted")
			}
		})
	}
}

func TestNestedOwningResourcePayloadMIRMutations(t *testing.T) {
	program := nestedOwningMIR(t)
	for _, mode := range []string{"option-move", "result-move", "payload-type", "container-type"} {
		t.Run(mode, func(t *testing.T) {
			forged := mir.Clone(program)
			changed := false
			for fi := range forged.Functions {
				for bi := range forged.Functions[fi].Blocks {
					for oi := range forged.Functions[fi].Blocks[bi].Operations {
						op := &forged.Functions[fi].Blocks[bi].Operations[oi]
						if changed || !op.MovePayload {
							continue
						}
						if mode == "result-move" {
							if op.Kind != mir.OperationResultPayload {
								continue
							}
						} else if op.Kind != mir.OperationOptionPayload {
							continue
						}
						switch mode {
						case "option-move", "result-move":
							op.MovePayload = false
						case "payload-type":
							op.Type = hir.TypeI32
						case "container-type":
							op.Operands = []mir.Operand{{Kind: mir.OperandI32, I32: 1}}
						}
						changed = true
					}
				}
			}
			if !changed || len(mir.Verify(forged)) == 0 {
				t.Fatal("forged payload extraction verified")
			}
			if _, err := cbackend.Emit(forged, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("forged payload extraction emitted")
			}
		})
	}
}

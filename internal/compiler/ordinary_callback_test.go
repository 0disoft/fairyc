package compiler_test

import (
	"context"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

const ordinaryCallbackSource = `module main;
unsafe extern C fn i32 visit(unsafe extern C fn c::double(c::double, c::int, ptr c::int) callback,ptr c::int data);
unsafe fn c::double handle(c::double value,c::int other,ptr c::int data){return value;}
fn i32 main(){return unsafe{visit(handle,null)};}`

func TestOrdinaryCallbackCodecAndPartition(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", ordinaryCallbackSource)})
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
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(unit.Source, "(double, int, const int *)") {
			t.Fatal("exact callback signature lost")
		}
		split, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 1)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(split.Header, "double fy_cb_") || !strings.Contains(split.Header, "const int * fy_a_2") {
			t.Fatal("partition callback declaration lost")
		}
	}
	for _, mode := range []string{"retype", "target-result", "effects", "escape"} {
		t.Run(mode, func(t *testing.T) {
			forged, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			for fi := range forged.Functions {
				for bi := range forged.Functions[fi].Blocks {
					block := &forged.Functions[fi].Blocks[bi]
					for oi := range block.Operations {
						op := &block.Operations[oi]
						if op.Kind != mir.OperationCCallback {
							continue
						}
						switch mode {
						case "retype":
							op.Type = hir.TypeCCallbackI32UserData
						case "escape":
							block.Terminator.ReturnValue = mir.Operand{Kind: mir.OperandLocal, Local: op.DestinationLocal, Type: op.Type}
						default:
							for ti := range forged.Functions {
								target := &forged.Functions[ti]
								if target.Symbol == op.Symbol {
									if mode == "target-result" {
										target.ResultType = hir.TypeI32
									} else {
										target.Effects = []hir.Effect{hir.EffectConsole}
									}
								}
							}
						}
					}
				}
			}
			if len(mir.Verify(forged)) == 0 {
				t.Fatal("forged callback accepted")
			}
			if _, err := cbackend.Emit(forged, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("forged callback emitted")
			}
		})
	}
}

func TestOrdinaryCallbackRejectsClosedShapes(t *testing.T) {
	for name, text := range map[string]string{
		"mismatch":            strings.Replace(ordinaryCallbackSource, "handle(c::double value,c::int other", "handle(c::double value,c::uint other", 1),
		"context-qualifier":   strings.Replace(ordinaryCallbackSource, "callback,ptr c::int data", "callback,ptr mut c::int data", 1),
		"null":                strings.Replace(ordinaryCallbackSource, "visit(handle,null)", "visit(null,null)", 1),
		"effectful":           strings.Replace(ordinaryCallbackSource, "unsafe fn c::double handle", "@effects(console) unsafe fn c::double handle", 1),
		"aggregate":           `module main;@repr(C) struct P{i32 x;} unsafe extern C fn i32 bad(unsafe extern C fn P(P,ptr mut void) cb,ptr mut void data);`,
		"indirect-expression": strings.Replace(ordinaryCallbackSource, "visit(handle,null)", "visit({handle},null)", 1),
		"no-user-data":        `module main;unsafe extern C fn i32 bad(unsafe extern C fn i32(i32) cb,ptr mut void data);`,
		"too-many":            `module main;unsafe extern C fn i32 bad(unsafe extern C fn i32(i32,i32,i32,i32,i32,i32,i32,i32,i32,ptr mut void) cb,ptr mut void data);`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err == nil && checked.Verified {
				t.Fatal("closed callback form admitted")
			}
		})
	}
}

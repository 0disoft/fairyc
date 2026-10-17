package compiler_test

import (
	"bytes"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"testing"
)

func TestSharedIntegerForRoundTrip(t *testing.T) {
	src, e := os.ReadFile("testdata/shared_integer_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	files := []semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(src))}
	checked, e := compiler.Check(files)
	if e != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", e, checked.Semantic)
	}
	again, e := compiler.Check(files)
	if e != nil || !again.Verified {
		t.Fatalf("repeated source check: %v %+v %+v", e, again.Semantic.Diagnostics, again.Semantic.Unsupported)
	}
	data, e := mircodec.Encode(checked.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	againData, e := mircodec.Encode(again.MIR, 1<<20)
	if e != nil || !bytes.Equal(data, againData) {
		t.Fatalf("repeated analysis changed MIR: %v", e)
	}
	p, e := mircodec.Decode(data, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	out, e := interpreter.Execute(p, "main::main")
	if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("execute: %+v %v", out, e)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, e := cbackend.Emit(p, "main::main", profile); e != nil {
			t.Fatal(e)
		}
	}
	types := map[hir.Type]int{}
	for _, fn := range p.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationViewElementBorrow {
					element, mut, ok := hir.BorrowElement(op.Type)
					if !ok || mut {
						t.Fatal("loop yielded value or mutable ref")
					}
					types[element]++
				}
			}
		}
	}
	for _, typ := range []hir.Type{hir.TypeU8, hir.TypeI32, hir.TypeI64, hir.TypeU32, hir.TypeUSize} {
		if types[typ] == 0 {
			t.Fatalf("missing stored reference for %s", typ)
		}
	}
	for _, name := range []string{"bounds", "mutable-result", "wrong-width", "array-owner"} {
		t.Run(name, func(t *testing.T) {
			bad, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for fi := range bad.Functions {
				for bi := range bad.Functions[fi].Blocks {
					b := &bad.Functions[fi].Blocks[bi]
					for oi := range b.Operations {
						if found {
							continue
						}
						op := &b.Operations[oi]
						if name == "array-owner" {
							if op.Kind == mir.OperationArrayBorrowView {
								op.OwnerType = hir.ParseType("[bool; 1]")
								found = true
							}
							continue
						}
						if name == "bounds" && op.Kind == mir.OperationViewIndexCheck {
							b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
							found = true
							break
						}
						if name == "bounds" || op.Kind != mir.OperationViewElementBorrow {
							continue
						}
						found = true
						if name == "mutable-result" {
							element, _, _ := hir.BorrowElement(op.Type)
							op.Type = hir.MutableBorrow(element)
						} else {
							op.Type = hir.MutableBorrow(hir.TypeBool)
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid loop ref verified")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid loop ref crossed codec")
			}
			if _, e := cbackend.Emit(bad, "main::main", cbackend.Stage0WindowsMSVCProfile()); e == nil {
				t.Fatal("invalid loop ref emitted")
			}
		})
	}
}

// An authenticated Vec provider must not capture integer view/array loops.
func TestSharedIntegerForWithSDKVector(t *testing.T) {
	const src = `module main;
import std::vec as v;
import std::bytes as bytes;
fn i32 visit(ref i32[] numbers, ref v::Vec<i32> vector) {
 var i32 total=0;
 for(let ref i32 item in numbers){total+=*item;}
 for(let ref i32 item in vector){total+=*item;}
 let i32[2] data=[3,4];
 for(let ref i32 data in data){total+=*data;}
 {let ref i32[] data=numbers;for(let ref i32 item in data){total+=*item;}}
 for(var i32 count=0;count<1;count+=1){let i32[1] data=[5];for(let ref i32 item in data){total+=*item;}}
 return total;
}
fn i32 main(){var v::Vec<i32> values=v::empty<i32>();let i32[1] data=[1];return visit(ref data,ref values);}`
	input, coordinates := vectorSDKInputWithCandidate(t, src, true)
	result, err := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
	if err != nil || len(result.Analysis.Semantic.Diagnostics)+len(result.Analysis.Semantic.Unsupported) != 0 {
		t.Fatalf("mixed iteration: %v %+v", err, result.Analysis.Semantic)
	}
	lowered, err := compiler.LowerAnalyzedStorageTransfers(result.Analysis)
	if err != nil || !lowered.Verified {
		t.Fatalf("mixed MIR: %v %+v", err, lowered)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, err := cbackend.EmitStorageTransfers(lowered.MIR, hir.QualifiedModulePath(1, "main")+"::main", profile); err != nil {
			t.Fatal(err)
		}
	}
}

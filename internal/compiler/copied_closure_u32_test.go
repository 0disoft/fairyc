package compiler_test

import (
	"encoding/json"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const copiedClosureU32Source = `module main;fn i32 main(){
var u32 offset=40;let add=fn [u32 offset] u32(u32 value){let u32 sum=offset+value;return sum;};offset=100;
assert(add(1)==41);assert(add(2)==42);
let identity=fn [] u32(u32 value){return value;};assert(identity(4294967295)==4294967295);
let u32 a=1;let u32 b=2;let u32 c=3;let u32 d=4;
let full=fn [u32 a,u32 b,u32 c,u32 d] u32(u32 w,u32 x,u32 y,u32 z){return a+b+c+d+w+x+y+z;};assert(full(5,6,7,14)==42);
return if(add(2)==42){42}else{1};}`

func TestCopiedClosureU32NativeAndRestore(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", copiedClosureU32Source)
	before, _ := json.Marshal(file.Module)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("u32 closure: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(file.Module)
	if string(before) != string(after) {
		t.Fatal("closure lifting changed source syntax")
	}
	value, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || value.Value != 42 {
		t.Fatalf("snapshot or signature lost: %+v %v", value, err)
	}
	runExpandedLanguageNative(t, result, "main::main", "")
	wire, err := mircodec.Encode(result.MIR, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(wire, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	value, err = interpreter.Execute(restored, "main::main")
	if err != nil || value.Value != 42 {
		t.Fatalf("restored capture: %+v %v", value, err)
	}
}

func TestCopiedClosureU32DoesNotWidenValueABI(t *testing.T) {
	for _, text := range []string{
		`module main;fn i32 main(){let u32 capture=40;let fn i32(i32) value=fn [u32 capture] i32(i32 argument){return argument;};return value(42);}`,
		`module main;fn i32 main(){let fn i32(i32) value=fn [] i32(i32 argument){let u32 unused=0;return argument;};return value(42);}`,
	} {
		file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
		result, err := compiler.CheckExplicitFiles([]semantic.File{file})
		if err == nil && result.Verified {
			t.Fatal("direct u32 permission escaped into the first-class closure family")
		}
	}
}

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
)

func TestControlOperandsRunWithMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("control operand native regression requires Windows amd64")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("control operand native regression requires local x64 MSVC: %v", err)
	}
	const text = `module control_operand;
fn i32 sum(i32 a,i32 b){return a+b;}
fn i32 id(i32 x){return x;}
fn i32 fail(){assert(false);return 99;}
fn i32 binary_read(){var i32 x=5;return x+if(true){x=2;3}else{4};}
fn i32 call_read(){var i32 x=5;return sum(x,if(true){x=2;3}else{4});}
fn i32 condition_return(){if({return 9;true}){discard fail();}return 0;}
fn i32 while_condition_return(){while({return 9;true}){discard fail();}return 0;}
fn i32 and_return(){discard {return 9;true}&&{discard fail();true};return 0;}
fn i32 or_return(){discard {return 9;false}||{discard fail();true};return 0;}
fn i32 binary_left(){return {return 9;1}+fail();}
fn i32 binary_right(){return id(1)+{return 9;2};}
fn i32 call_first(){return sum({return 9;1},fail());}
fn i32 call_last(){return sum(id(1),{return 9;2});}
fn i32 call_both(){return sum(if(true){return 9;1}else{return 10;2},fail());}
fn i32 call_fallthrough(){return sum(if(false){return 9;1}else{2},3);}
fn i32 main(){
    assert(binary_read()==8);assert(call_read()==8);
    assert(condition_return()==9);assert(and_return()==9);assert(or_return()==9);
    assert(while_condition_return()==9);
    assert(binary_left()==9);assert(binary_right()==9);
    assert(call_first()==9);assert(call_last()==9);assert(call_both()==9);
    assert(call_fallthrough()==5);return 42;
}
`
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "control_operand.fy"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(root, "control operands "+profile, "control_operand.fy", "", profile,
			cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			t.Fatal(err)
		}
	}
}

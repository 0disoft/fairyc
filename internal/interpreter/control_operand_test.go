package interpreter_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestControlOperandsPreserveSourceOrderAndReturn(t *testing.T) {
	t.Parallel()
	const helpers = `
fn i32 sum(i32 a, i32 b) { return a+b; }
fn i32 id(i32 x) { return x; }
fn i32 fail() { assert(false); return 99; }
`
	for _, test := range []struct {
		name, body string
		want       int32
	}{
		{"binary-read-before-if", `var i32 x=5; return x+if(true){x=2;3}else{4};`, 8},
		{"call-read-before-if", `var i32 x=5; return sum(x,if(true){x=2;3}else{4});`, 8},
		{"if-condition-return", `if({return 9;true}){discard 1;}return 0;`, 9},
		{"and-left-return", `discard {return 9;true}&&{discard fail();true};return 0;`, 9},
		{"or-left-return", `discard {return 9;false}||{discard fail();true};return 0;`, 9},
		{"binary-left-return", `return {return 9;1}+fail();`, 9},
		{"binary-right-return", `return id(1)+{return 9;2};`, 9},
		{"call-first-argument-return", `return sum({return 9;1},fail());`, 9},
		{"call-last-argument-return", `return sum(id(1),{return 9;2});`, 9},
		{"call-conditional-argument-returns", `return sum(if(true){return 9;1}else{return 10;2},fail());`, 9},
		{"call-conditional-argument-fallthrough", `return sum(if(false){return 9;1}else{2},3);`, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			program := lowerSource(t, "module main; fn i32 main(){"+test.body+"}"+helpers)
			if failures := mir.Verify(program); len(failures) != 0 {
				t.Fatalf("Verify() = %+v", failures)
			}
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != test.want || outcome.TrapCategory != "" {
				t.Fatalf("Execute() = (%+v, %v), want return %d", outcome, err, test.want)
			}
		})
	}
}

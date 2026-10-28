package interpreter_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestWideEnumInitializersPreserveSideEffectOrder(t *testing.T) {
	const fields = 64
	var text strings.Builder
	text.WriteString("module main; enum Packet { Data {")
	for i := range fields {
		fmt.Fprintf(&text, "i32 f%d;", i)
	}
	text.WriteString(`}; }
fn i32 next(ref mut i32 n) { *n += 1; return *n; }
fn i32 main() { var i32 n = 0; let Packet packet = .Data {`)
	for i := fields - 1; i >= 0; i-- {
		fmt.Fprintf(&text, ".f%d = next(ref mut n),", i)
	}
	text.WriteString("}; return switch(packet) { case .Data {")
	for i := range fields {
		fmt.Fprintf(&text, ".f%d = v%d,", i, i)
	}
	text.WriteString("}: { v0 * 1000 + v63 } }; }")
	program := lowerSource(t, text.String())
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 64001 {
		t.Fatalf("wide enum execution order: %+v, %v", outcome, err)
	}
}

func TestEnumNamedInitializersPreserveEvaluationAndPayloadOrder(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, source string
		want         int32
	}{
		{"side-effects", `module main;
enum Reading { Values { i32 first; i32 second; }; }
fn i32 next(ref mut i32 n) { *n += 1; return *n; }
fn i32 main() {
    var i32 n = 0;
    let Reading value = .Values { .second = next(ref mut n), .first = next(ref mut n) };
    return switch (value) { case .Values { .first = a, .second = b }: { a * 10 + b } };
}`, 21},
		{"nested-payload", `module main;
struct Pair { i32 first; i32 second; }
enum State { Empty; Value { Pair pair; i32 bias; }; }
fn i32 main() {
    let State value = .Value { .bias = 3, .pair = Pair { .first = 20, .second = 19 } };
    return switch (value) { case .Empty: { 0 } case .Value { .pair = p, .bias = b }: { p.first + p.second + b } };
}`, 42},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := lowerSource(t, test.source)
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != test.want {
				t.Fatalf("Execute() = %+v, %v; want return %d", outcome, err, test.want)
			}
		})
	}
}

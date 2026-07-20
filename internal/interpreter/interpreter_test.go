package interpreter_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestExecuteRestrictedEnumMatch(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
enum Message {
    Empty;
    Value { amount: i32; };
    Pair { left: i32; right: i32; };
}
fn score(message: Message) -> i32 {
    match message {
        Message::Empty => 0,
        Message::Value { amount: value } => value,
        Message::Pair { left: first, right: second } => first + second,
    }
}
fn main() -> i32 { score(Message::Pair { left: 20, right: 22 }) }
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteImmutableByteViewIndexWithinBounds(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let value: u8 = b"\x00A"[1];
    if value == 65 { 42 } else { 1 }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteImmutableByteViewIndexOutOfBoundsTraps(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let value: u8 = b"A"[1];
    42
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "bounds" {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteImmutableByteViewSlice(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let bytes: &[u8] = b"fairy"[1..4];
    if bytes.len == 3 && bytes[0] == 97 && bytes[2] == 114 { 42 } else { 1 }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteEmptyImmutableByteViewSliceAtEnd(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let bytes: &[u8] = b"fairy"[5..5];
    if bytes.is_empty { 42 } else { 1 }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteImmutableByteViewSliceBoundsTraps(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		bounds string
	}{
		{name: "start after end", bounds: "4..2"},
		{name: "end after length", bounds: "0..6"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := lowerSource(t, "module main;\nfn main() -> i32 { let bytes: &[u8] = b\"fairy\"["+test.bounds+"]; 42 }\n")
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "bounds" {
				t.Fatalf("Execute() outcome = %+v", outcome)
			}
		})
	}
}

func TestExecuteValuePreservingIntegerCast(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 { b"*"[0] as i32 }
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteTypedScalarComparisons(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let byte: u8 = b"*"[0];
    let word: u32 = byte as u32;
    let size: usize = word as usize;
    let pointer: *u8 = null;
    if true == true && -1 < 0 && byte > 41 && word >= 42 && size <= 42 && pointer == null { 42 } else { 1 }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteCheckedUnsignedArithmeticAndByteIteration(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn iterate() -> i32 {
    let bytes: &[u8] = b"\x13\x17";
    var index: usize = 0;
    var total: i32 = 0;
    while index < bytes.len {
        total = total + (bytes[index] as i32);
        index = index + 1;
    }
    total
}
fn u8_multiply_overflow() -> i32 {
    let value: u8 = 128;
    let overflow: u8 = value * 2;
    42
}
fn u32_subtract_underflow() -> i32 {
    let value: u32 = 0;
    let underflow: u32 = value - 1;
    42
}
fn usize_add_overflow() -> i32 {
    let value: usize = 18446744073709551615;
    let overflow: usize = value + 1;
    42
}
fn unsigned_division() -> i32 {
    let byte_input: u8 = 84;
    let byte: u8 = byte_input / 2;
    let word_input: u32 = 85;
    let word: u32 = word_input % 43;
    let size_input: usize = 84;
    let size: usize = size_input / 2;
    if byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}
fn u8_divide_zero() -> i32 { let numerator: u8 = 42; let zero: u8 = 0; let value: u8 = numerator / zero; 42 }
fn u32_remainder_zero() -> i32 { let numerator: u32 = 42; let zero: u32 = 0; let value: u32 = numerator % zero; 42 }
fn usize_divide_zero() -> i32 { let numerator: usize = 42; let zero: usize = 0; let value: usize = numerator / zero; 42 }
`)
	for _, testCase := range []struct {
		entry string
		kind  string
		value int32
		trap  string
	}{
		{entry: "main::iterate", kind: interpreter.OutcomeReturn, value: 42},
		{entry: "main::u8_multiply_overflow", kind: interpreter.OutcomeTrap, trap: "integer-overflow"},
		{entry: "main::u32_subtract_underflow", kind: interpreter.OutcomeTrap, trap: "integer-overflow"},
		{entry: "main::usize_add_overflow", kind: interpreter.OutcomeTrap, trap: "integer-overflow"},
		{entry: "main::unsigned_division", kind: interpreter.OutcomeReturn, value: 42},
		{entry: "main::u8_divide_zero", kind: interpreter.OutcomeTrap, trap: "division-by-zero"},
		{entry: "main::u32_remainder_zero", kind: interpreter.OutcomeTrap, trap: "division-by-zero"},
		{entry: "main::usize_divide_zero", kind: interpreter.OutcomeTrap, trap: "division-by-zero"},
	} {
		outcome, err := interpreter.Execute(program, testCase.entry)
		if err != nil {
			t.Fatalf("Execute(%s) error = %v", testCase.entry, err)
		}
		if outcome.Kind != testCase.kind || outcome.Value != testCase.value || outcome.TrapCategory != testCase.trap {
			t.Fatalf("Execute(%s) outcome = %+v", testCase.entry, outcome)
		}
	}
}

func TestExecuteAlwaysCheckedAssert(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn passes() -> i32 { assert(true); 42 }
fn fails() -> i32 { assert(false); 42 }
`)
	passed, err := interpreter.Execute(program, "main::passes")
	if err != nil || passed.Kind != interpreter.OutcomeReturn || passed.Value != 42 {
		t.Fatalf("Execute(passes) = %+v, %v", passed, err)
	}
	failed, err := interpreter.Execute(program, "main::fails")
	if err != nil || failed.Kind != interpreter.OutcomeTrap || failed.TrapCategory != "assertion-failed" {
		t.Fatalf("Execute(fails) = %+v, %v", failed, err)
	}
}

func lowerSource(t *testing.T, contents string) mir.Program {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(contents)), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
		t.Fatalf("AnalyzeSyntax() diagnostics = %+v, AST = %+v", parsed.Tree.Diagnostics, parsed.AST)
	}
	result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("Analyze() result = %+v", result)
	}
	program, err := mir.Lower(result.HIR)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	return program
}

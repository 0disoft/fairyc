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

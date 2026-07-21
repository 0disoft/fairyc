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
    Value { i32 amount; };
    Pair { i32 left; i32 right; };
}
fn i32 score(Message message) {
    return switch (message) {
        case .Empty : { 0 }
        case .Value { .amount = value } : { value }
        case .Pair { .left = first, .right = second } : { first + second }
    };
}
fn i32 main() { return score(.Pair { .left = 20, .right = 22 }); }
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("Execute() outcome = %+v", outcome)
	}
}

func TestExecuteNonEmptyResourceDropBody(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn i32 observe(i32 value) { return value; }
resource Token {
    i32 value;
    cleanup {
        let i32 observed = observe(this.value);
    }
}
fn i32 main() {
    let Token token = Token { .value = 65 };
    return 42;
}
`)
	outcome, err := interpreter.ExecuteWithOptions(program, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{
			"main::observe": interpreter.TestIntrinsicWriteStdoutByte,
		},
		OutputLimit: 1,
	})
	if err != nil {
		t.Fatalf("ExecuteWithOptions() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || string(outcome.Stdout) != "A" || len(outcome.DropTrace) != 1 {
		t.Fatalf("ExecuteWithOptions() outcome = %+v", outcome)
	}
}

func TestExecuteImmutableByteViewIndexWithinBounds(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn i32 main() {
    let u8 value = b"\x00A"[1];
    return if (value == 65) { 42 } else { 1 };
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
fn i32 main() {
    let u8 value = b"A"[1];
    return 42;
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

func TestExecuteScalarFixedArrayAcrossCallAndIndex(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn u8 pick(u8[3] values, usize index) { return values[index]; }
fn i32 main() {
    let u8[3] values = [40, 41, 42];
    return if (values.len == 3) { pick(values, 2) as i32 } else { 1 };
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

func TestExecuteFixedU8ArrayBorrowedAsByteView(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn i32 select(ref u8[] bytes) {
    return if (bytes.len == 3) { bytes[2] as i32 } else { 1 };
}
fn i32 main() {
    let u8[3] values = [40, 41, 42];
    return select(ref values);
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestExecuteMutableFixedU8ArrayBorrowedAsByteView(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn usize length(ref mut u8[] bytes) { return bytes.len; }
fn i32 main() {
    var u8[3] values = [40, 41, 42];
    return if (length(ref mut values) == 3) { values[2] as i32 } else { 1 };
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestExecuteCheckedMutableIndexedStores(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn u8 update(ref mut u8[] bytes) {
    bytes[1] = 42;
    return bytes[1];
}
fn i32 main() {
    var u8[3] values = [1, 2, 3];
    values[0] = 40;
    let u8 changed = update(ref mut values);
    return if (values[0] == 40 && values[1] == changed && values[1] == 42) { 42 } else { 1 };
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestExecuteFixedArrayCallPreservesValueSemanticsAcrossMutation(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn u8 changed(u8[2] input) {
    var u8[2] values = input;
    values[0] = 42;
    return values[0];
}
fn i32 main() {
    let u8[2] values = [1, 2];
    let u8 result = changed(values);
    return if (result == 42 && values[0] == 1) { 42 } else { 1 };
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestExecuteIndexedStoreChecksBoundsBeforeEvaluatingValue(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn i32 observe(i32 value) { return value; }
fn i32 main() {
    var i32[1] values = [0];
    values[1] = observe(65);
    return 42;
}
`)
	outcome, err := interpreter.ExecuteWithOptions(program, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{
			"main::observe": interpreter.TestIntrinsicWriteStdoutByte,
		},
		OutputLimit: 1,
	})
	if err != nil {
		t.Fatalf("ExecuteWithOptions() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "bounds" || len(outcome.Stdout) != 0 {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestExecuteReassignedFixedArrayByteView(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn i32 main() {
    let u8[3] first = [1, 2, 3];
    let u8[3] second = [40, 41, 42];
    var ref u8[] bytes = ref first;
    bytes = ref second;
    return bytes[2] as i32;
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestExecuteScalarFixedArrayOutOfBoundsTraps(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn i32 main() {
    let i32[2] values = [40, 42];
    return values[2];
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
fn i32 main() {
    let ref u8[] bytes = b"fairy"[1..4];
    return if (bytes.len == 3 && bytes[0] == 97 && bytes[2] == 114) { 42 } else { 1 };
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

func TestExecuteImmutableByteViewSliceWithOmittedBounds(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		bounds string
		want   string
	}{
		{name: "start", bounds: "..2", want: "bytes.len == 2 && bytes[1] == 97"},
		{name: "end", bounds: "2..", want: "bytes.len == 3 && bytes[0] == 105"},
		{name: "both", bounds: "..", want: "bytes.len == 5 && bytes[4] == 121"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			program := lowerSource(t, "module main;\nfn i32 main() { let ref u8[] bytes = b\"fairy\"["+test.bounds+"]; return if ("+test.want+") { 42 } else { 1 }; }\n")
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
				t.Fatalf("Execute() outcome = %+v", outcome)
			}
		})
	}
}

func TestExecuteEmptyImmutableByteViewSliceAtEnd(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn i32 main() {
    let ref u8[] bytes = b"fairy"[5..5];
    return if (bytes.is_empty) { 42 } else { 1 };
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
			program := lowerSource(t, "module main;\nfn i32 main() { let ref u8[] bytes = b\"fairy\"["+test.bounds+"]; return 42; }\n")
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
fn i32 main() { return b"*"[0] as i32; }
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
fn i32 main() {
    let u8 byte = b"*"[0];
    let u32 word = byte as u32;
    let usize size = word as usize;
    let ptr u8 pointer = null;
    return if (true == true && -1 < 0 && byte > 41 && word >= 42 && size <= 42 && pointer == null) { 42 } else { 1 };
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
fn i32 iterate() {
    let ref u8[] bytes = b"\x13\x17";
    var usize index = 0;
    var i32 total = 0;
    while (index < bytes.len) {
        total = total + (bytes[index] as i32);
        index = index + 1;
    }
    return total;
}
fn i32 u8_multiply_overflow() {
    let u8 value = 128;
    let u8 overflow = value * 2;
    return 42;
}
fn i32 u32_subtract_underflow() {
    let u32 value = 0;
    let u32 underflow = value - 1;
    return 42;
}
fn i32 usize_add_overflow() {
    let usize value = 18446744073709551615;
    let usize overflow = value + 1;
    return 42;
}
fn i32 unsigned_division() {
    let u8 byte_input = 84;
    let u8 byte = byte_input / 2;
    let u32 word_input = 85;
    let u32 word = word_input % 43;
    let usize size_input = 84;
    let usize size = size_input / 2;
    return if (byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
}
fn i32 u8_divide_zero() { let u8 numerator = 42; let u8 zero = 0; let u8 value = numerator / zero; return 42; }
fn i32 u32_remainder_zero() { let u32 numerator = 42; let u32 zero = 0; let u32 value = numerator % zero; return 42; }
fn i32 usize_divide_zero() { let usize numerator = 42; let usize zero = 0; let usize value = numerator / zero; return 42; }
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
fn i32 passes() { assert(true); return 42; }
fn i32 fails() { assert(false); return 42; }
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

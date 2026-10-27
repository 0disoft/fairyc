package interpreter_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
)

const preparedSource = `module main;
struct Box { i32 value; }
fn i32 emit(i32 value) { return value; }
resource Token { i32 value; cleanup { discard emit(this.value); } }
fn void passes() {
    let Box box = Box { .value = 65 };
    let Token token = Token { .value = box.value };
    assert(token.value == 65);
}
fn void fails() { assert(false); }
fn i32 wrong_result() { return 0; }
fn void with_argument(i32 value) {}
`

func preparedOptions() interpreter.Options {
	return interpreter.Options{MaxSteps: 64, OutputLimit: 1,
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{
			"main::emit": interpreter.TestIntrinsicWriteStdoutByte,
		},
	}
}

func TestPreparedProgramDetachesVerifiedMIR(t *testing.T) {
	program := lowerSource(t, preparedSource)
	options := preparedOptions()
	want, err := interpreter.ExecuteVoidWithOptions(program, "main::passes", options)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := interpreter.Prepare(program)
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt nested bodies and type definitions after preparation. The immutable
	// prepared copy must still run, while ordinary execution must reject the input.
	for i := range program.Functions {
		function := &program.Functions[i]
		function.Name = "changed"
		function.Entry = mir.NoBlock
		for j := range function.Blocks {
			block := &function.Blocks[j]
			block.Terminator.Successors = append(block.Terminator.Successors, mir.NoBlock)
			for k := range block.Operations {
				for n := range block.Operations[k].Operands {
					block.Operations[k].Operands[n].Type = hir.TypeVoid
				}
			}
		}
	}
	program.StructTypes[0].Fields[0].Type = hir.TypeBool
	program.ResourceTypes[0].Fields[0].Type = hir.TypeBool
	if _, err := interpreter.ExecuteVoid(program, "main::passes"); err == nil {
		t.Fatal("ordinary execution admitted corrupted MIR")
	}
	if _, err := interpreter.Prepare(program); err == nil {
		t.Fatal("preparation admitted corrupted MIR")
	}
	got, err := prepared.ExecuteVoidWithOptions("main::passes", options)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("caller mutation changed prepared execution: %+v %v", got, err)
	}
}

func TestPreparedProgramFreshStateAndConcurrentReuse(t *testing.T) {
	program := lowerSource(t, preparedSource)
	prepared, err := interpreter.Prepare(program)
	if err != nil {
		t.Fatal(err)
	}
	options := preparedOptions()
	want, err := interpreter.ExecuteVoidWithOptions(program, "main::passes", options)
	if err != nil || string(want.Stdout) != "A" || len(want.DropTrace) != 1 {
		t.Fatalf("reference execution: %+v %v", want, err)
	}
	for i := 0; i < 12; i++ {
		failed, err := prepared.ExecuteVoidWithOptions("main::fails", options)
		if err != nil || failed.Kind != interpreter.OutcomeTrap || failed.TrapCategory != "assertion-failed" {
			t.Fatalf("trap: %+v %v", failed, err)
		}
		got, err := prepared.ExecuteVoidWithOptions("main::passes", options)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d leaked runtime state or budgets: %+v %v", i, got, err)
		}
		got.Stdout[0] = 'Z'
		got.DropTrace[0].ResourceType = hir.TypeVoid
	}
	// Options, including fixture hooks, are selected independently on every run.
	plain, err := prepared.ExecuteVoidWithOptions("main::passes", interpreter.Options{})
	if err != nil || plain.Kind != interpreter.OutcomeReturn || len(plain.Stdout) != 0 {
		t.Fatalf("fixture hooks leaked: %+v %v", plain, err)
	}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("concurrent=%d", i), func(t *testing.T) {
			t.Parallel()
			got, err := prepared.ExecuteVoidWithOptions("main::passes", options)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("concurrent execution: %+v %v", got, err)
			}
		})
	}
}

func TestPreparedProgramPreservesEntryAndExecutionGuards(t *testing.T) {
	program := lowerSource(t, preparedSource)
	prepared, err := interpreter.Prepare(program)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{"missing", "main::wrong_result", "main::with_argument"} {
		_, want := interpreter.ExecuteVoid(program, entry)
		_, got := prepared.ExecuteVoidWithOptions(entry, interpreter.Options{})
		if want == nil || got == nil || got.Error() != want.Error() {
			t.Fatalf("entry %q: got %v, want %v", entry, got, want)
		}
	}
	for _, options := range []interpreter.Options{{MaxSteps: -1}, {MaxCallDepth: -1}} {
		if _, err := prepared.ExecuteVoidWithOptions("main::passes", options); err == nil {
			t.Fatal("negative execution budget admitted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := prepared.ExecuteVoidWithOptions("main::passes", interpreter.Options{Context: ctx})
	if !errors.Is(err, context.Canceled) || got.Kind != interpreter.OutcomeCanceled {
		t.Fatalf("cancellation: %+v %v", got, err)
	}
	loop, err := interpreter.Prepare(loopingProgram(hir.TypeVoid))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		got, err := loop.ExecuteVoidWithOptions("main", interpreter.Options{MaxSteps: 3})
		var limit interpreter.CapabilityError
		if !errors.As(err, &limit) || limit.Kind != interpreter.LimitStep || got.Kind != interpreter.OutcomeLimit {
			t.Fatalf("step limit: %+v %v", got, err)
		}
	}
	for _, unprepared := range []*interpreter.PreparedProgram{nil, {}} {
		if _, err := unprepared.ExecuteVoidWithOptions("main", interpreter.Options{}); err == nil {
			t.Fatal("unprepared execution admitted")
		}
	}
}

package interpreter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
)

func loopingProgram(result hir.Type) mir.Program {
	return mir.Program{Functions: []mir.Function{{Symbol: 1, Name: "main", ResultType: result, Entry: 0,
		Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{0}, UnwindSuccessor: mir.NoBlock}}},
	}}}
}

func TestCallerStepLimitsApplyToBothEntryPoints(t *testing.T) {
	for _, void := range []bool{false, true} {
		var outcome interpreter.Outcome
		var err error
		options := interpreter.Options{MaxSteps: 3}
		if void {
			outcome, err = interpreter.ExecuteVoidWithOptions(loopingProgram(hir.TypeVoid), "main", options)
		} else {
			outcome, err = interpreter.ExecuteWithOptions(loopingProgram(hir.TypeI32), "main", options)
		}
		var limit interpreter.CapabilityError
		if !errors.As(err, &limit) || limit.Kind != interpreter.LimitStep || outcome.Kind != interpreter.OutcomeLimit {
			t.Fatalf("void=%t: %+v %v", void, outcome, err)
		}
	}
}

func TestCallerDepthLimitCanBeLoweredAndRaised(t *testing.T) {
	program := lowerSource(t, `module main;
fn i32 recur(i32 count) { if (count == 0) { return 42; } return recur(count - 1); }
fn i32 main() { return recur(5); }
fn void test_entry() { assert(main() == 42); }
`)
	for _, depth := range []int{3, 12, 0} {
		for _, void := range []bool{false, true} {
			var outcome interpreter.Outcome
			var err error
			options := interpreter.Options{MaxCallDepth: depth}
			if void {
				outcome, err = interpreter.ExecuteVoidWithOptions(program, "main::test_entry", options)
			} else {
				outcome, err = interpreter.ExecuteWithOptions(program, "main::main", options)
			}
			if depth == 3 {
				var limit interpreter.CapabilityError
				if !errors.As(err, &limit) || limit.Kind != interpreter.LimitCallDepth || outcome.Kind != interpreter.OutcomeLimit {
					t.Fatalf("depth=%d void=%t: %+v %v", depth, void, outcome, err)
				}
			} else if err != nil || outcome.Kind != interpreter.OutcomeReturn {
				t.Fatalf("depth=%d void=%t: %+v %v", depth, void, outcome, err)
			}
		}
	}
}

// Cancel on a deterministic execution checkpoint instead of relying on timing.
type checkpointContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (ctx *checkpointContext) Err() error {
	ctx.checks++
	if ctx.checks == 8 {
		ctx.cancel()
	}
	return ctx.Context.Err()
}

func TestCallerCancellationBeforeAndDuringExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	expired, release := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer release()
	active, stop := context.WithCancel(context.Background())
	defer stop()
	during := &checkpointContext{Context: active, cancel: stop}
	for _, item := range []struct {
		ctx    context.Context
		reason error
	}{{ctx, context.Canceled}, {expired, context.DeadlineExceeded}, {during, context.Canceled}} {
		outcome, err := interpreter.ExecuteVoidWithOptions(loopingProgram(hir.TypeVoid), "main", interpreter.Options{Context: item.ctx, MaxSteps: 100})
		if !errors.Is(err, item.reason) || outcome.Kind != interpreter.OutcomeCanceled {
			t.Fatalf("cancellation lost: %+v %v", outcome, err)
		}
	}
	if during.checks != 8 {
		t.Fatalf("did not cancel during execution: %d checks", during.checks)
	}
}

func TestInvalidExecutionBudgetsAreRejected(t *testing.T) {
	for _, options := range []interpreter.Options{{MaxSteps: -1}, {MaxCallDepth: -1}} {
		if _, err := interpreter.ExecuteWithOptions(loopingProgram(hir.TypeI32), "main", options); err == nil {
			t.Fatal("negative budget accepted")
		}
	}
}

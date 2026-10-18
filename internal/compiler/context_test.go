package compiler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// Cancel on a deterministic checkpoint, so in-flight tests need neither a
// giant source fixture nor timing assumptions about the test machine.
type checkpointContext struct {
	context.Context
	cancel    context.CancelFunc
	remaining int
}

func (c *checkpointContext) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		c.cancel()
	}
	return c.Context.Err()
}

func cancelAfter(t *testing.T, checkpoints int) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &checkpointContext{Context: ctx, cancel: cancel, remaining: checkpoints}
}

func TestContextCancellationDiscardsCompilerResults(t *testing.T) {
	files := []semantic.File{parseCompilerFile(t, 1, "main.fy", "module main; fn i32 main() { var i32 value = 1; value += 2; return value; }")}
	checked, err := compiler.Check(files)
	if err != nil || !checked.Verified {
		t.Fatalf("fixture: %v %+v", err, checked.Semantic.Diagnostics)
	}
	for _, checkpoint := range []int{1, 8, 20} {
		ctx := cancelAfter(t, checkpoint)
		result, err := compiler.CheckContext(ctx, files)
		var failure *compiler.Error
		if !errors.Is(err, context.Canceled) || !errors.As(err, &failure) || failure.Kind() != compiler.ErrorCanceled || result.Verified || len(result.MIR.Functions) != 0 || len(result.Semantic.HIR.Modules) != 0 {
			t.Fatalf("checkpoint %d: result=%+v error=%v", checkpoint, result, err)
		}
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := compiler.CheckContext(ctx, files); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline cause: %v", err)
	}
}

func TestContextCancellationInMIRAndCEmission(t *testing.T) {
	files := []semantic.File{parseCompilerFile(t, 1, "main.fy", "module main; fn i32 main() { var i32 x = 0; x += 1; x += 2; return x; }")}
	checked, err := compiler.Check(files)
	if err != nil || !checked.Verified {
		t.Fatalf("fixture: %v", err)
	}
	lowered, err := mir.LowerContext(cancelAfter(t, 5), checked.Semantic.HIR)
	if !errors.Is(err, context.Canceled) || len(lowered.Functions) != 0 {
		t.Fatalf("partial lowering: %v %+v", err, lowered)
	}
	failures, err := mir.VerifyWithCapabilitiesContext(cancelAfter(t, 3), checked.MIR, admission.Capabilities{})
	if !errors.Is(err, context.Canceled) || len(failures) != 0 {
		t.Fatalf("partial verification: %v %v", err, failures)
	}
	unit, err := cbackend.EmitContext(cancelAfter(t, 7), checked.MIR, checked.MIR.Functions[0].Name, cbackend.Stage0WindowsMSVCProfile())
	if !errors.Is(err, context.Canceled) || unit.Source != "" {
		t.Fatalf("partial C emission: %v %q", err, unit.Source)
	}
}

func TestContextCancellationInGenericExpansion(t *testing.T) {
	file := parseCompilerFile(t, 1, "main.fy", "module main; fn T identity<T>(T value) { return value; } fn i32 main() { return identity<i32>(42); }")
	input := semantic.PackageInput{Files: []semantic.File{file}}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: genericCoordinate(t)}
	result, _, err := compiler.CheckWithPolicy(input, coordinates, genericplan.DefaultPlanLimits(), admission.Policy[hir.PackageID]{})
	if err != nil || !result.Verified {
		t.Fatalf("fixture: %v %+v", err, result.Semantic.Diagnostics)
	}
	for _, checkpoint := range []int{1, 4, 12} {
		result, _, err := compiler.CheckWithPolicyContext(cancelAfter(t, checkpoint), input, coordinates, genericplan.DefaultPlanLimits(), admission.Policy[hir.PackageID]{})
		if !errors.Is(err, context.Canceled) || result.Verified || len(result.MIR.Functions) != 0 {
			t.Fatalf("generic checkpoint %d: %v %+v", checkpoint, err, result)
		}
	}
}

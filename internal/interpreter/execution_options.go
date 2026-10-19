package interpreter

import (
	"context"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

const (
	DefaultMaxSteps     = 1_000_000
	DefaultMaxCallDepth = 256
	OutcomeCanceled     = "canceled"
	LimitCallDepth      = "call-depth-limit"
	// Value-shape cycle protection is independent of the executable call budget.
	maxValueNesting = 256
)

// Options selects interpreter budgets, fixture intrinsics and authenticated SDK adapters.
// Zero budgets retain the defaults; negative budgets are rejected. A nil Context
// means no caller cancellation. Steps count block entries and MIR operations.
type Options struct {
	MaxSteps       int
	MaxCallDepth   int
	Context        context.Context
	TestIntrinsics map[string]TestIntrinsicKind
	OutputLimit    int
	// AbstractArenaSDKPackages must come from authenticated bundled SDK owners.
	// They enable host-free Arena construction, not arbitrary foreign calls.
	AbstractArenaSDKPackages []hir.PackageID
}

func (options Options) normalized() (Options, error) {
	if options.MaxSteps < 0 || options.MaxCallDepth < 0 {
		return Options{}, fmt.Errorf("interpreter step and call-depth limits must be non-negative")
	}
	if options.MaxSteps == 0 {
		options.MaxSteps = DefaultMaxSteps
	}
	if options.MaxCallDepth == 0 {
		options.MaxCallDepth = DefaultMaxCallDepth
	}
	if options.Context == nil {
		options.Context = context.Background()
	}
	return options, nil
}

func (r *executionState) checkCancellation() error {
	if r.context != nil {
		return r.context.Err()
	}
	return nil
}

func (r *executionState) consumeStep() error {
	if err := r.checkCancellation(); err != nil {
		return err
	}
	if r.steps <= 0 {
		return capabilityErrorKind(LimitStep, "interpreter step limit exceeded")
	}
	r.steps--
	return nil
}

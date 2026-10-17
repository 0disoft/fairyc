package nominalhir

import (
	"fmt"
	"reflect"
	"time"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// BoundaryPatch contains only annotated functions, not a cloned source closure.
// The baseline and its semantic resolutions must come from the same completed
// Flow analysis (and verified compilation for MIR reuse). This build-local
// experiment does not load patches from caches.
type BoundaryPatch struct {
	Declarations []Declaration
	Functions    []hir.Function
}

// LowerBoundaryPatch validates edited functions and their caller coverage before
// cloning its verified MIR instead of lowering the baseline HIR again. Whole-HIR/MIR equality is
// a test oracle; exact erasure of each edited function remains an admission check.
func LowerBoundaryPatch(baseline compiler.Result, patch BoundaryPatch) (mir.Program, error) {
	return lowerBoundaryPatch(baseline, patch, nil, true)
}

// PatchTimings separates retained-caller coverage, edited-function validation,
// and result materialization. Ordinary LowerBoundaryPatch does not read a clock.
type PatchTimings struct {
	Coverage        time.Duration `json:"coverage_ns"`
	Validation      time.Duration `json:"validation_ns"`
	Materialization time.Duration `json:"materialization_ns"`
}

// MeasureBoundaryPatch is an experimental measurement entry. Both alternatives
// execute identical validation; reuse=false retains the old lowering control.
func MeasureBoundaryPatch(baseline compiler.Result, patch BoundaryPatch, reuse bool) (mir.Program, PatchTimings, error) {
	var timings PatchTimings
	program, err := lowerBoundaryPatch(baseline, patch, &timings, reuse)
	return program, timings, err
}

func lowerBoundaryPatch(baseline compiler.Result, patch BoundaryPatch, timings *PatchTimings, reuse bool) (mir.Program, error) {
	if !baseline.Verified {
		return mir.Program{}, fmt.Errorf("nominal patch requires a verified baseline")
	}
	if err := validateAnalysisPatch(baseline.Semantic, patch, &baseline.MIR, timings); err != nil {
		return mir.Program{}, err
	}
	var started time.Time
	if timings != nil {
		started = time.Now()
	}
	var program mir.Program
	var err error
	if reuse {
		program = mir.Clone(baseline.MIR)
	} else {
		program, err = mir.Lower(baseline.Semantic.HIR)
	}
	if timings != nil {
		timings.Materialization = time.Since(started)
	}
	return program, err
}

// validateAnalysisPatch accepts only an in-process completed semantic analysis.
// Source resolutions cover all uses before lowering, including unreachable calls.
// Retained MIR, when available in the comparison harness, adds a redundant check.
func validateAnalysisPatch(analysis semantic.Result, patch BoundaryPatch, retained *mir.Program, timings *PatchTimings) error {
	var started time.Time
	if timings != nil {
		started = time.Now()
	}
	if analysis.CompletedThrough != semantic.PhaseFlow || analysis.FailedPhase != semantic.PhaseNone || len(analysis.Diagnostics)+len(analysis.Unsupported) != 0 || len(patch.Declarations) == 0 || len(patch.Functions) == 0 {
		return fmt.Errorf("nominal patch requires completed Flow analysis, declarations and functions")
	}
	type owner struct {
		packageID hir.PackageID
		module    string
		function  hir.Function
	}
	owners := map[hir.SymbolID]owner{}
	modules := map[string]bool{}
	b := boundary{types: map[hir.Type]Declaration{}, functions: map[hir.SymbolID]hir.Function{}, remaining: 1000000}
	for _, m := range analysis.HIR.Modules {
		modules[hir.QualifiedModulePath(m.Package, m.Path)] = true
		for _, f := range m.Functions {
			if _, ok := owners[f.Symbol]; ok {
				return fmt.Errorf("duplicate baseline function")
			}
			owners[f.Symbol] = owner{m.Package, m.Path, f}
			b.functions[f.Symbol] = f
		}
	}
	for _, d := range patch.Declarations {
		_, exists := b.types[d.Type()]
		if exists || !validName(d.Name) || !validPath(d.Module) || d.Underlying != hir.TypeUSize || !modules[hir.QualifiedModulePath(d.Package, d.Module)] {
			return fmt.Errorf("invalid patch declaration %s", d.Type())
		}
		b.types[d.Type()] = d
	}
	edited := map[hir.SymbolID]bool{}
	changed := map[hir.SymbolID]bool{}
	for _, f := range patch.Functions {
		o, ok := owners[f.Symbol]
		if !ok || edited[f.Symbol] {
			return fmt.Errorf("unknown or duplicate patch function")
		}
		edited[f.Symbol] = true
		b.functions[f.Symbol] = f
		if !reflect.DeepEqual(f.Parameters, o.function.Parameters) || f.ResultType != o.function.ResultType {
			if o.function.OwnerType != hir.ParseType("") {
				return fmt.Errorf("method signature patches are not supported")
			}
			changed[f.Symbol] = true
		}
	}
	// Resolutions include source uses that MIR may have eliminated (for example,
	// an unreachable call). Never use executable call edges as the sole coverage
	// oracle. A changed callee with no retained resolution is outside this slice.
	resolved := map[hir.SymbolID]bool{}
	for _, r := range analysis.Resolutions {
		for id := range changed {
			o := owners[id]
			if r.Target.Package != o.packageID || r.Target.Module != o.module || r.Target.Name != o.function.Name {
				continue
			}
			resolved[id] = true
			covered := false
			for caller := range edited {
				if within(r.Use, owners[caller].function.Body.Span) {
					covered = true
					break
				}
			}
			if !covered {
				return fmt.Errorf("nominal signature change has an unpatched source caller at %s:%d", r.Use.Source, r.Use.StartByte)
			}
		}
	}
	for id := range changed {
		if !resolved[id] {
			return fmt.Errorf("changed signature lacks retained source resolutions")
		}
	}
	if retained != nil {
		for _, f := range retained.Functions {
			for _, block := range f.Blocks {
				if block.Terminator.Kind == mir.TerminatorCall && changed[block.Terminator.Callee] && !edited[f.Symbol] {
					return fmt.Errorf("nominal signature change has an unpatched MIR caller")
				}
			}
		}
	}
	if timings != nil {
		timings.Coverage = time.Since(started)
		started = time.Now()
	}
	for _, f := range patch.Functions {
		o := owners[f.Symbol]
		b.owner = o.packageID
		erased, err := b.walk(reflect.ValueOf(f), 0)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(erased.Interface().(hir.Function), o.function) {
			return fmt.Errorf("nominal patch changed the analyzed function %s", o.function.Name)
		}
	}
	if timings != nil {
		timings.Validation = time.Since(started)
	}
	return nil
}

func within(use, body hir.Span) bool {
	return use.Package == body.Package && use.Source == body.Source && use.StartByte >= body.StartByte && use.EndByte <= body.EndByte
}

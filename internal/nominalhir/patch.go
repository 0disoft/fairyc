package nominalhir

import (
	"fmt"
	"reflect"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// BoundaryPatch contains only annotated functions, not a cloned source closure.
// The baseline and its semantic resolutions must come from the same verified
// compilation. This build-local experiment does not load patches from caches.
type BoundaryPatch struct {
	Declarations []Declaration
	Functions    []hir.Function
}

// LowerBoundaryPatch validates edited functions and their caller coverage before
// reusing the baseline HIR for ordinary MIR lowering. Whole-HIR/MIR equality is
// a test oracle; exact erasure of each edited function remains an admission check.
func LowerBoundaryPatch(baseline compiler.Result, patch BoundaryPatch) (mir.Program, error) {
	if !baseline.Verified || len(baseline.Semantic.Diagnostics)+len(baseline.Semantic.Unsupported) != 0 || len(patch.Declarations) == 0 || len(patch.Functions) == 0 {
		return mir.Program{}, fmt.Errorf("nominal patch requires a verified baseline, declarations and functions")
	}
	type owner struct {
		packageID hir.PackageID
		module    string
		function  hir.Function
	}
	owners := map[hir.SymbolID]owner{}
	modules := map[string]bool{}
	b := boundary{types: map[hir.Type]Declaration{}, functions: map[hir.SymbolID]hir.Function{}, remaining: 1000000}
	for _, m := range baseline.Semantic.HIR.Modules {
		modules[hir.QualifiedModulePath(m.Package, m.Path)] = true
		for _, f := range m.Functions {
			if _, ok := owners[f.Symbol]; ok {
				return mir.Program{}, fmt.Errorf("duplicate baseline function")
			}
			owners[f.Symbol] = owner{m.Package, m.Path, f}
			b.functions[f.Symbol] = f
		}
	}
	for _, d := range patch.Declarations {
		_, exists := b.types[d.Type()]
		if exists || !validName(d.Name) || !validPath(d.Module) || d.Underlying != hir.TypeUSize || !modules[hir.QualifiedModulePath(d.Package, d.Module)] {
			return mir.Program{}, fmt.Errorf("invalid patch declaration %s", d.Type())
		}
		b.types[d.Type()] = d
	}
	edited := map[hir.SymbolID]bool{}
	changed := map[hir.SymbolID]bool{}
	for _, f := range patch.Functions {
		o, ok := owners[f.Symbol]
		if !ok || edited[f.Symbol] {
			return mir.Program{}, fmt.Errorf("unknown or duplicate patch function")
		}
		edited[f.Symbol] = true
		b.functions[f.Symbol] = f
		if !reflect.DeepEqual(f.Parameters, o.function.Parameters) || f.ResultType != o.function.ResultType {
			if o.function.OwnerType != "" {
				return mir.Program{}, fmt.Errorf("method signature patches are not supported")
			}
			changed[f.Symbol] = true
		}
	}
	// Resolutions include source uses that MIR may have eliminated (for example,
	// an unreachable call). Never use executable call edges as the sole coverage
	// oracle. A changed callee with no retained resolution is outside this slice.
	resolved := map[hir.SymbolID]bool{}
	for _, r := range baseline.Semantic.Resolutions {
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
				return mir.Program{}, fmt.Errorf("nominal signature change has an unpatched source caller at %s:%d", r.Use.Source, r.Use.StartByte)
			}
		}
	}
	for id := range changed {
		if !resolved[id] {
			return mir.Program{}, fmt.Errorf("changed signature lacks retained source resolutions")
		}
	}
	for _, f := range baseline.MIR.Functions {
		for _, block := range f.Blocks {
			if block.Terminator.Kind == mir.TerminatorCall && changed[block.Terminator.Callee] && !edited[f.Symbol] {
				return mir.Program{}, fmt.Errorf("nominal signature change has an unpatched MIR caller")
			}
		}
	}
	for _, f := range patch.Functions {
		o := owners[f.Symbol]
		b.owner = o.packageID
		erased, err := b.walk(reflect.ValueOf(f), 0)
		if err != nil {
			return mir.Program{}, err
		}
		if !reflect.DeepEqual(erased.Interface().(hir.Function), o.function) {
			return mir.Program{}, fmt.Errorf("nominal patch changed the verified function %s", o.function.Name)
		}
	}
	return mir.Lower(baseline.Semantic.HIR)
}

func within(use, body hir.Span) bool {
	return use.Package == body.Package && use.Source == body.Source && use.StartByte >= body.StartByte && use.EndByte <= body.EndByte
}

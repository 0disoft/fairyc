package workspacecompiler

import (
	"context"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

const stateGraphSourceAdmission = "experimental-source-state-graph/1"

// NewExperimentalStateGraphSession selects current source graph analysis and
// the existing verified package artifact/cache owner. It never selects ordinary
// program execution for the call-only MIR projection or caches state outcomes.
// Existing workspace constructors retain their ordinary source admission.
func NewExperimentalStateGraphSession(configuration Configuration) (*Session, error) {
	session, err := NewSession(configuration)
	if err != nil {
		return nil, err
	}
	// The admission profile participates in every memory/disk artifact key.
	// Projection MIR must never be reused under ordinary source semantics.
	session.compilerBuild += ";" + stateGraphSourceAdmission
	session.analyze = analyzeStateGraphWorkspace
	disk := session.storage == StorageDisk
	session.restore = func(compiled compiler.Result) (compiler.Result, error) {
		return restoreStateGraphNamespace(compiled, disk)
	}
	return session, nil
}

func analyzeStateGraphWorkspace(ctx context.Context, input semantic.PackageInput) (compiler.Analysis, error) {
	if err := ctx.Err(); err != nil {
		return compiler.Analysis{}, err
	}
	if len(input.Files) != 1 || len(input.Imports) != 0 {
		return compiler.Analysis{}, fmt.Errorf("experimental state graph workspace requires one source file without imports")
	}
	return compiler.AnalyzeExperimentalStateGraphContext(ctx, input.Files[0])
}

// Cache reassembly allocates fresh function symbols. Restore the current HIR
// namespace before native handoff, without lowering MIR or altering cached
// objects. The source subset has only parameterless void call projections;
// aggregate, foreign, callback and local-value namespaces are not admitted.
func restoreStateGraphNamespace(compiled compiler.Result, disk bool) (compiler.Result, error) {
	if len(compiled.Semantic.HIR.Modules) != 1 || len(compiled.MIR.StructTypes)+len(compiled.MIR.EnumTypes)+len(compiled.MIR.ResourceTypes)+len(compiled.MIR.ForeignFunctions) != 0 {
		return compiler.Result{}, fmt.Errorf("state graph cache has an unadmitted ordinary context")
	}
	module := compiled.Semantic.HIR.Modules[0]
	current := make(map[string]hir.SymbolID, len(module.Functions))
	for _, function := range module.Functions {
		name := hir.QualifiedModulePath(module.Package, module.Path) + "::" + function.Name
		if _, duplicate := current[name]; duplicate {
			return compiler.Result{}, fmt.Errorf("state graph current function %q is ambiguous", name)
		}
		current[name] = function.Symbol
	}
	if len(current) != len(compiled.MIR.Functions) {
		return compiler.Result{}, fmt.Errorf("state graph cache does not cover current functions")
	}
	symbols := make(map[hir.SymbolID]hir.SymbolID, len(current))
	seen := make(map[string]bool, len(current))
	for _, function := range compiled.MIR.Functions {
		fresh, found := current[function.Name]
		if !found || seen[function.Name] || function.Span.Package != module.Package || len(function.Parameters) != 0 || function.ResultType != hir.TypeVoid {
			return compiler.Result{}, fmt.Errorf("state graph cached function %q has no exact current owner", function.Name)
		}
		if _, duplicate := symbols[function.Symbol]; duplicate {
			return compiler.Result{}, fmt.Errorf("state graph cache has duplicate function symbols")
		}
		seen[function.Name], symbols[function.Symbol] = true, fresh
	}
	program := mir.Clone(compiled.MIR)
	for i := range program.Functions {
		function := &program.Functions[i]
		function.Symbol = symbols[function.Symbol]
		for j := range function.Blocks {
			call := &function.Blocks[j].Terminator
			if call.Kind == mir.TerminatorReturn {
				// Stable local rebasing also numbers this inactive field. A void
				// return in the source projection has its original zero value.
				call.DestinationLocal = 0
				continue
			}
			if call.Kind != mir.TerminatorCall {
				return compiler.Result{}, fmt.Errorf("state graph cache has a non-projection terminator")
			}
			fresh, found := symbols[call.Callee]
			if !found || call.ForeignABI != "" {
				return compiler.Result{}, fmt.Errorf("state graph cached call has no current callee")
			}
			call.Callee = fresh
		}
	}
	transactions := slices.Clone(compiled.StateTransactions)
	for i := range transactions {
		// Memory hits lower state MIR from current HIR; disk hits rebind it to
		// artifact symbols. Both must resolve to the same current root owner.
		fresh, found := transactions[i].Function, false
		if disk {
			fresh, found = symbols[fresh]
		} else {
			for _, symbol := range current {
				found = found || symbol == fresh
			}
		}
		if !found || transactions[i].Package != module.Package {
			return compiler.Result{}, fmt.Errorf("state graph cached transaction has no current owner")
		}
		transactions[i].Function = fresh
		if len(transactions[i].Program.Helpers) == 0 {
			// Stable decoding materializes an empty list; fresh HIR lowering
			// uses the absent helper list for the same source-owned program.
			transactions[i].Program.Helpers = nil
		}
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return compiler.Result{}, fmt.Errorf("state graph current MIR rebind: %v", failures[0])
	}
	compiled.MIR, compiled.StateTransactions = program, transactions
	return compiled, nil
}

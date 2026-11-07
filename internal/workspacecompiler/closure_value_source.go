package workspacecompiler

import (
	"context"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packageartifact"
)

// Reconstruct only closure-bearing functions from freshly checked HIR. This is
// a bounded source-validation cost, not a second whole-package cache lowering.
// The ordinary lower hook and unrelated functions are never invoked here.
func verifyClosureValueSource(ctx context.Context, live hir.Package, program mir.Program) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	carriers := map[hir.Type]bool{hir.TypeClosureI32: true}
	declarations := make(map[hir.Type]hir.Struct)
	for _, module := range live.Modules {
		for _, record := range module.Structs {
			for _, field := range record.Fields {
				if field.Type == hir.TypeClosureI32 {
					carriers[record.Type] = true
					declarations[record.Type] = record
				}
			}
		}
	}
	for _, record := range program.StructTypes {
		closure := false
		for _, field := range record.Fields {
			closure = closure || field.Type == hir.TypeClosureI32
		}
		if !closure {
			continue
		}
		want, found := declarations[record.Type]
		if !found || record.ReprC != want.ReprC || len(record.Fields) != len(want.Fields) {
			return fmt.Errorf("closure carrier declaration differs from current source")
		}
		for i, field := range record.Fields {
			if field.Name != want.Fields[i].Name || field.Type != want.Fields[i].Type {
				return fmt.Errorf("closure carrier field differs from current source")
			}
		}
	}
	// Every admitted construction is a typed local or has a lifted target;
	// passing/returning values exposes the type in a declaration. Legacy sources
	// therefore need no expression walk, body reconstruction or symbol index.
	feature := len(declarations) != 0
	for _, module := range live.Modules {
		for _, fn := range module.Functions {
			feature = feature || fn.ClosureTarget != nil || carriers[fn.ResultType]
			for _, p := range fn.Parameters {
				feature = feature || carriers[p.Type]
			}
			for _, local := range fn.Locals {
				feature = feature || carriers[local.Type]
			}
		}
	}
	if !feature {
		if mir.HasCopiedClosureValues(program) {
			return fmt.Errorf("cached closure values have no current source owner")
		}
		return ctx.Err()
	}
	selected := hir.Package{Modules: make([]hir.Module, 0, len(live.Modules))}
	symbols := make(map[hir.SymbolID]string)
	count := 0
	for _, module := range live.Modules {
		copy := module
		copy.Functions = nil
		for _, fn := range module.Functions {
			name := hir.QualifiedModulePath(module.Package, module.Path) + "::" + fn.Name
			symbols[fn.Symbol] = "fn:" + name
			if closureSourceFunction(fn, carriers) {
				copy.Functions = append(copy.Functions, fn)
				count++
			}
		}
		for _, fn := range module.ForeignFunctions {
			symbols[fn.Symbol] = "foreign:" + fn.ABI + ":" + fn.LinkName
		}
		selected.Modules = append(selected.Modules, copy)
	}
	if count == 0 {
		for _, fn := range program.Functions {
			if mir.HasCopiedClosureValues(mir.Program{Functions: []mir.Function{fn}}) {
				return fmt.Errorf("cached closure values have no current source owner")
			}
		}
		return ctx.Err()
	}
	expected, err := mir.LowerContext(ctx, selected)
	if err != nil {
		return fmt.Errorf("validate closure source bodies: %w", err)
	}
	proofs := make(map[string][32]byte, len(expected.Functions))
	for _, fn := range expected.Functions {
		proof, err := packageartifact.ClosureSourceProof(fn, symbols)
		if err != nil {
			return err
		}
		proofs[fn.Name] = proof
	}
	restoredSymbols := make(map[hir.SymbolID]string, len(program.Functions)+len(program.ForeignFunctions))
	names := make(map[string]bool, len(program.Functions))
	for _, fn := range program.Functions {
		if names[fn.Name] {
			return fmt.Errorf("closure source proof has an ambiguous function name")
		}
		names[fn.Name] = true
		restoredSymbols[fn.Symbol] = "fn:" + fn.Name
	}
	for _, fn := range program.ForeignFunctions {
		restoredSymbols[fn.Symbol] = "foreign:" + fn.ABI + ":" + fn.LinkName
	}
	for _, fn := range program.Functions {
		if err := ctx.Err(); err != nil {
			return err
		}
		want, selected := proofs[fn.Name]
		if !selected {
			if fn.ClosureTarget || mir.HasCopiedClosureValues(mir.Program{Functions: []mir.Function{fn}}) {
				return fmt.Errorf("cached closure function %s has no current source owner", fn.Name)
			}
			continue
		}
		got, err := packageartifact.ClosureSourceProof(fn, restoredSymbols)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("cached closure body differs from current source: %s", fn.Name)
		}
		delete(proofs, fn.Name)
	}
	if len(proofs) != 0 {
		return fmt.Errorf("current source closure functions are missing from cached MIR")
	}
	return nil
}

func closureSourceFunction(fn hir.Function, carriers map[hir.Type]bool) bool {
	if fn.ClosureTarget != nil || carriers[fn.ResultType] {
		return true
	}
	for _, p := range fn.Parameters {
		if carriers[p.Type] {
			return true
		}
	}
	for _, local := range fn.Locals {
		if carriers[local.Type] {
			return true
		}
	}
	var expression func(*hir.Expression) bool
	var block func(hir.Block) bool
	var statement func(hir.Statement) bool
	block = func(b hir.Block) bool {
		for _, s := range b.Statements {
			if statement(s) {
				return true
			}
		}
		return expression(b.Tail)
	}
	statement = func(s hir.Statement) bool {
		if expression(s.Target) || expression(s.Expression) {
			return true
		}
		if s.For != nil {
			f := s.For
			return f.Initializer != nil && statement(*f.Initializer) || expression(f.Condition) || f.Post != nil && statement(*f.Post) || block(f.Body)
		}
		return false
	}
	expression = func(e *hir.Expression) bool {
		if e == nil {
			return false
		}
		if carriers[e.Type] || e.Kind == hir.ExpressionClosureValue || e.Kind == hir.ExpressionClosureCall {
			return true
		}
		for i := range e.Children {
			if expression(&e.Children[i]) {
				return true
			}
		}
		if e.Block != nil && block(*e.Block) {
			return true
		}
		for i := range e.Arms {
			if expression(&e.Arms[i].Expression) {
				return true
			}
		}
		return false
	}
	return block(fn.Body)
}

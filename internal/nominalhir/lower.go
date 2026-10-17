// Package nominalhir owns the isolated nominal-scalar HIR experiment. It has no
// source parser, cache format or stable compiler integration. Lower validates the
// complete closed input before handing scalar-only HIR to the existing backend.
package nominalhir

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Declaration identifies one same-package nominal usize domain.
type Declaration struct {
	Package      hir.PackageID
	Module, Name string
	Underlying   hir.Type
	Span         hir.Span
}

func (d Declaration) Type() hir.Type {
	return hir.ParseType("@nominal:" + hir.QualifiedModulePath(d.Package, d.Module) + "::" + d.Name)
}

// Program is experimental typed HIR. Unlike ordinary hir.Package, it can contain
// nominal-wrap/nominal-unwrap expressions and nominal value types.
type Program struct {
	Declarations []Declaration
	Modules      []hir.Module
}

type checker struct {
	types     map[hir.Type]Declaration
	functions map[hir.SymbolID]hir.Function
	remaining int
}

// Lower is the only experimental lowering entry. Errors publish no partial MIR.
// The initial slice accepts straight-line scalar functions; unsupported HIR is
// rejected rather than traversed with incomplete nominal checks.
func Lower(input Program) (mir.Program, error) {
	c := checker{map[hir.Type]Declaration{}, map[hir.SymbolID]hir.Function{}, 10000}
	if len(input.Modules) == 0 || len(input.Declarations) == 0 {
		return mir.Program{}, fmt.Errorf("nominal HIR requires modules and declarations")
	}
	owner := input.Modules[0].Package
	modules := map[string]bool{}
	for _, m := range input.Modules {
		if m.Package != owner || !validPath(m.Path) || modules[m.Path] || len(m.Structs)+len(m.Enums)+len(m.Resources)+len(m.ForeignFunctions) != 0 {
			return mir.Program{}, fmt.Errorf("unsupported or duplicate nominal module %q", m.Path)
		}
		modules[m.Path] = true
	}
	for _, d := range input.Declarations {
		if d.Package != owner || !modules[d.Module] || !validName(d.Name) || d.Underlying != hir.TypeUSize {
			return mir.Program{}, fmt.Errorf("invalid nominal declaration %s::%s", d.Module, d.Name)
		}
		if _, exists := c.types[d.Type()]; exists {
			return mir.Program{}, fmt.Errorf("duplicate nominal declaration %s", d.Type())
		}
		c.types[d.Type()] = d
	}
	for _, m := range input.Modules {
		names := map[string]bool{}
		for _, f := range m.Functions {
			_, duplicate := c.functions[f.Symbol]
			if duplicate || !validName(f.Name) || names[f.Name] || f.OwnerType != hir.ParseType("") || f.Receiver != "" || f.DropGlueFor != hir.ParseType("") || f.Unsafe || len(f.Effects) != 0 {
				return mir.Program{}, fmt.Errorf("unsupported or duplicate function %q", f.Name)
			}
			for _, d := range input.Declarations {
				if d.Module == m.Path && d.Name == f.Name {
					return mir.Program{}, fmt.Errorf("duplicate declaration %s", f.Name)
				}
			}
			if !c.valueType(f.ResultType) && f.ResultType != hir.TypeVoid {
				return mir.Program{}, fmt.Errorf("unsupported result type %s", f.ResultType)
			}
			names[f.Name] = true
			c.functions[f.Symbol] = f
		}
	}
	output := hir.Package{}
	for _, m := range input.Modules {
		out := hir.Module{Package: m.Package, Path: m.Path, SourcePath: m.SourcePath}
		for _, f := range m.Functions {
			lowered, err := c.function(f)
			if err != nil {
				return mir.Program{}, fmt.Errorf("nominal HIR %s::%s: %w", m.Path, f.Name, err)
			}
			out.Functions = append(out.Functions, lowered)
		}
		output.Modules = append(output.Modules, out)
	}
	result, err := mir.Lower(output)
	if err != nil {
		return mir.Program{}, err
	}
	if failures := mir.Verify(result); len(failures) != 0 {
		return mir.Program{}, fmt.Errorf("nominal scalar MIR: %v", failures)
	}
	return result, nil
}

func validName(s string) bool {
	if s == "" {
		return false
	}
	for i, ch := range s {
		if ch != '_' && !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(i > 0 && ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

func validPath(s string) bool {
	for _, part := range strings.Split(s, "::") {
		if !validName(part) {
			return false
		}
	}
	return true
}

func (c *checker) valueType(t hir.Type) bool {
	_, nominal := c.types[t]
	return nominal || t == hir.TypeUSize || t == hir.TypeI32 || t == hir.TypeBool
}

func (c *checker) scalar(t hir.Type) hir.Type {
	if _, ok := c.types[t]; ok {
		return hir.TypeUSize
	}
	return t
}

func mismatch(want, got hir.Type, span hir.Span) error {
	return fmt.Errorf("%s:%d: expected %s, got %s", span.Source, span.StartByte, want, got)
}

type binding struct {
	typ            hir.Type
	mutable, ready bool
}

func (c *checker) function(f hir.Function) (hir.Function, error) {
	bindings := map[hir.SymbolID]binding{}
	out := f
	out.Parameters = slices.Clone(f.Parameters)
	out.Locals = slices.Clone(f.Locals)
	out.ResultType = c.scalar(f.ResultType)
	add := func(id hir.SymbolID, typ hir.Type, mutable, ready bool) error {
		if _, exists := bindings[id]; exists || !c.valueType(typ) {
			return fmt.Errorf("duplicate binding or unsupported type %s", typ)
		}
		bindings[id] = binding{typ, mutable, ready}
		return nil
	}
	for i, p := range f.Parameters {
		if err := add(p.Symbol, p.Type, p.Mutable, true); err != nil {
			return hir.Function{}, err
		}
		out.Parameters[i].Type = c.scalar(p.Type)
	}
	for i, l := range f.Locals {
		if err := add(l.Symbol, l.Type, l.Mutable, false); err != nil {
			return hir.Function{}, err
		}
		out.Locals[i].Type = c.scalar(l.Type)
	}
	out.Body = hir.Block{Span: f.Body.Span}
	returned := false
	for _, s := range f.Body.Statements {
		if returned || s.For != nil {
			return hir.Function{}, fmt.Errorf("unsupported statement flow")
		}
		if s.Expression == nil {
			return hir.Function{}, fmt.Errorf("statement requires a value")
		}
		value, err := c.expression(*s.Expression, bindings, 0)
		if err != nil {
			return hir.Function{}, err
		}
		copy := hir.Statement{Kind: s.Kind, Span: s.Span, Binding: s.Binding, Expression: &value}
		switch s.Kind {
		case hir.StatementBinding, hir.StatementAssignment:
			id := s.Binding
			if s.Kind == hir.StatementAssignment {
				if s.Target == nil || s.Target.Kind != hir.ExpressionPath || s.Operator != "=" {
					return hir.Function{}, fmt.Errorf("only direct assignment is supported")
				}
				target, err := c.expression(*s.Target, bindings, 0)
				if err != nil {
					return hir.Function{}, err
				}
				id = s.Target.Symbol
				copy.Target = &target
				copy.Operator = "="
			} else if s.Target != nil || s.Operator != "" {
				return hir.Function{}, fmt.Errorf("malformed binding")
			}
			b, found := bindings[id]
			if !found || (s.Kind == hir.StatementBinding && b.ready) || (s.Kind == hir.StatementAssignment && (!b.ready || !b.mutable)) {
				return hir.Function{}, fmt.Errorf("invalid binding or assignment %d", id)
			}
			if b.typ != s.Expression.Type {
				return hir.Function{}, mismatch(b.typ, s.Expression.Type, s.Span)
			}
			b.ready = true
			bindings[id] = b
		case hir.StatementReturn:
			if s.Target != nil || s.Operator != "" {
				return hir.Function{}, fmt.Errorf("malformed return")
			}
			if f.ResultType != s.Expression.Type {
				return hir.Function{}, mismatch(f.ResultType, s.Expression.Type, s.Span)
			}
			returned = true
		case hir.StatementDiscard, hir.StatementExpressionStatement:
			if s.Target != nil || s.Operator != "" {
				return hir.Function{}, fmt.Errorf("malformed expression statement")
			}
		default:
			return hir.Function{}, fmt.Errorf("unsupported statement %s", s.Kind)
		}
		out.Body.Statements = append(out.Body.Statements, copy)
	}
	if f.Body.Tail != nil {
		if returned {
			return hir.Function{}, fmt.Errorf("tail after return")
		}
		value, err := c.expression(*f.Body.Tail, bindings, 0)
		if err != nil {
			return hir.Function{}, err
		}
		if f.Body.Tail.Type != f.ResultType {
			return hir.Function{}, mismatch(f.ResultType, f.Body.Tail.Type, f.Body.Tail.Span)
		}
		out.Body.Tail = &value
		returned = true
	}
	if !returned && f.ResultType != hir.TypeVoid {
		return hir.Function{}, fmt.Errorf("missing return")
	}
	return out, nil
}

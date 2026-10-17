package nominalhir

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// LowerBoundary checks a nominal overlay against an already verified compilation.
// It is an experiment, not a frontend: the caller supplies explicit conversions.
// Erasure must recover the exact retained HIR, including source and ownership facts.
// This permits ordinary resource/control-flow HIR without reimplementing its checker.
// The full reflective walk is deliberately measurable overhead, not a fast-path claim.
func LowerBoundary(baseline compiler.Result, input Program) (mir.Program, error) {
	if !baseline.Verified || len(baseline.Semantic.Diagnostics)+len(baseline.Semantic.Unsupported) != 0 || len(input.Declarations) == 0 {
		return mir.Program{}, fmt.Errorf("nominal boundary requires a verified baseline and declarations")
	}
	b := boundary{types: map[hir.Type]Declaration{}, functions: map[hir.SymbolID]hir.Function{}, remaining: 1000000}
	owners := map[string]bool{}
	for _, m := range input.Modules {
		owners[hir.QualifiedModulePath(m.Package, m.Path)] = true
	}
	for _, d := range input.Declarations {
		_, exists := b.types[d.Type()]
		if exists || !validName(d.Name) || !validPath(d.Module) || d.Underlying != hir.TypeUSize || !owners[hir.QualifiedModulePath(d.Package, d.Module)] {
			return mir.Program{}, fmt.Errorf("invalid boundary declaration %s", d.Type())
		}
		b.types[d.Type()] = d
	}
	for _, m := range input.Modules {
		for _, f := range m.Functions {
			if _, exists := b.functions[f.Symbol]; exists {
				return mir.Program{}, fmt.Errorf("duplicate function symbol")
			}
			b.functions[f.Symbol] = f
		}
	}
	erased, err := b.walk(reflect.ValueOf(input.Modules), 0)
	if err != nil {
		return mir.Program{}, err
	}
	modules := erased.Interface().([]hir.Module)
	if !reflect.DeepEqual(modules, baseline.Semantic.HIR.Modules) {
		return mir.Program{}, fmt.Errorf("nominal erasure changed the verified baseline HIR")
	}
	output := baseline.Semantic.HIR
	output.Modules = modules
	program, err := mir.Lower(output)
	if err != nil {
		return mir.Program{}, err
	}
	// Exact equality also retains the baseline's storage-aware MIR verification.
	if !reflect.DeepEqual(program, baseline.MIR) {
		return mir.Program{}, fmt.Errorf("nominal boundary changed verified MIR")
	}
	return program, nil
}

type boundary struct {
	types     map[hir.Type]Declaration
	functions map[hir.SymbolID]hir.Function
	bindings  map[hir.SymbolID]hir.Type
	result    hir.Type
	owner     hir.PackageID
	remaining int
}

func (b *boundary) nominal(t hir.Type) bool { return strings.Contains(t.String(), "@nominal:") }
func (b *boundary) scalar(t hir.Type) (hir.Type, error) {
	if d, ok := b.types[t]; ok {
		if d.Package != b.owner {
			return hir.ParseType(""), fmt.Errorf("cross-package nominal boundary")
		}
		return hir.TypeUSize, nil
	}
	if b.nominal(t) {
		return hir.ParseType(""), fmt.Errorf("unknown or compound nominal type %s", t)
	}
	return t, nil
}

func (b *boundary) expression(e hir.Expression) (hir.Expression, error) {
	for _, t := range []hir.Type{e.TargetType} {
		if b.nominal(t) {
			return e, fmt.Errorf("unsupported nominal target type")
		}
	}
	if e.Kind == hir.ExpressionNominalWrap || e.Kind == hir.ExpressionNominalUnwrap {
		if len(e.Children) != 1 || e.Operator != "" || e.Block != nil || len(e.Arms)+len(e.Fields)+len(e.Effects) != 0 || e.UnsafeScope != nil || e.ForeignABI != "" || e.TargetType != hir.ParseType("") || e.Cooked != "" || e.Variant != "" {
			return e, fmt.Errorf("invalid boundary conversion")
		}
		child := e.Children[0]
		_, outer := b.types[e.Type]
		_, inner := b.types[child.Type]
		if !(e.Kind == hir.ExpressionNominalWrap && outer && child.Type == hir.TypeUSize || e.Kind == hir.ExpressionNominalUnwrap && inner && e.Type == hir.TypeUSize) {
			return e, fmt.Errorf("invalid boundary conversion types")
		}
		if _, err := b.scalar(e.Type); err != nil {
			return e, err
		}
		return child, nil
	}
	if e.Kind == hir.ExpressionPath {
		if want, exists := b.bindings[e.Symbol]; exists && (b.nominal(want) || b.nominal(e.Type)) && want != e.Type {
			return e, mismatch(want, e.Type, e.Span)
		}
		if b.nominal(e.Type) {
			if _, exists := b.bindings[e.Symbol]; !exists {
				return e, fmt.Errorf("unknown nominal binding")
			}
		}
	}
	if e.Kind == hir.ExpressionCall {
		f, exists := b.functions[e.Symbol]
		if exists {
			if len(e.Children) != len(f.Parameters)+1 {
				return e, fmt.Errorf("boundary call arity mismatch")
			}
			for i, p := range f.Parameters {
				actual := e.Children[i+1].Type
				if (b.nominal(p.Type) || b.nominal(actual)) && p.Type != actual {
					return e, mismatch(p.Type, actual, e.Children[i+1].Span)
				}
			}
			if (b.nominal(f.ResultType) || b.nominal(e.Type)) && f.ResultType != e.Type {
				return e, mismatch(f.ResultType, e.Type, e.Span)
			}
		} else {
			if b.nominal(e.Type) {
				return e, fmt.Errorf("nominal result from foreign/unknown call")
			}
			for _, child := range e.Children {
				if b.nominal(child.Type) {
					return e, fmt.Errorf("nominal argument to foreign/unknown call")
				}
			}
		}
	}
	if b.nominal(e.Type) {
		switch e.Kind {
		case hir.ExpressionPath, hir.ExpressionCall, hir.ExpressionGroup:
		default:
			return e, fmt.Errorf("unsupported nominal expression %s", e.Kind)
		}
	}
	for _, child := range e.Children {
		if !b.nominal(child.Type) {
			continue
		}
		switch e.Kind {
		case hir.ExpressionCall: // Checked against the exact callee above.
		case hir.ExpressionGroup:
			if len(e.Children) != 1 || e.Type != child.Type {
				return e, fmt.Errorf("nominal group mismatch")
			}
		case hir.ExpressionBinary:
			if len(e.Children) != 2 || e.Children[0].Type != e.Children[1].Type || e.Type != hir.TypeBool {
				return e, fmt.Errorf("nominal comparison mismatch")
			}
			switch e.Operator {
			case "==", "!=", "<", "<=", ">", ">=":
			default:
				return e, fmt.Errorf("nominal arithmetic needs extraction")
			}
		default:
			return e, fmt.Errorf("nominal operand requires explicit extraction in %s", e.Kind)
		}
	}
	return e, nil
}

var hirType = reflect.TypeOf(hir.ParseType(""))

func (b *boundary) walk(v reflect.Value, depth int) (reflect.Value, error) {
	b.remaining--
	if depth > 256 || b.remaining < 0 {
		return reflect.Value{}, fmt.Errorf("nominal boundary walk budget exceeded")
	}
	if v.Type() == hirType {
		if b.nominal(v.Interface().(hir.Type)) {
			return reflect.Value{}, fmt.Errorf("nominal type in unsupported HIR position")
		}
		return v, nil
	}
	// Special cases validate before generic recursion erases child types.
	if v.CanInterface() {
		switch x := v.Interface().(type) {
		case hir.Module:
			old := b.owner
			b.owner = x.Package
			defer func() { b.owner = old }()
		case hir.Function:
			old, result := b.bindings, b.result
			b.bindings = map[hir.SymbolID]hir.Type{}
			b.result = x.ResultType
			defer func() { b.bindings = old; b.result = result }()
			for _, p := range x.Parameters {
				b.bindings[p.Symbol] = p.Type
			}
			for _, l := range x.Locals {
				b.bindings[l.Symbol] = l.Type
			}
			x.Parameters = append([]hir.Parameter(nil), x.Parameters...)
			x.Locals = append([]hir.Local(nil), x.Locals...)
			var err error
			for i := range x.Parameters {
				x.Parameters[i].Type, err = b.scalar(x.Parameters[i].Type)
				if err != nil {
					return reflect.Value{}, err
				}
			}
			for i := range x.Locals {
				x.Locals[i].Type, err = b.scalar(x.Locals[i].Type)
				if err != nil {
					return reflect.Value{}, err
				}
			}
			x.ResultType, err = b.scalar(x.ResultType)
			if err != nil {
				return reflect.Value{}, err
			}
			v = reflect.ValueOf(x)
		case hir.Statement:
			if x.Expression != nil {
				actual := x.Expression.Type
				want := hir.ParseType("")
				switch x.Kind {
				case hir.StatementBinding:
					want = b.bindings[x.Binding]
				case hir.StatementReturn:
					want = b.result
				case hir.StatementAssignment:
					if x.Target != nil {
						want = x.Target.Type
					}
					if b.nominal(want) && x.Operator != "=" {
						return reflect.Value{}, fmt.Errorf("nominal compound assignment")
					}
				}
				if want != hir.ParseType("") && (b.nominal(want) || b.nominal(actual)) && want != actual {
					return reflect.Value{}, mismatch(want, actual, x.Span)
				}
			}
		case hir.Block:
			if x.Tail != nil && b.nominal(x.Tail.Type) {
				return reflect.Value{}, fmt.Errorf("nominal block tails require a later boundary slice")
			}
		case hir.Expression:
			checked, err := b.expression(x)
			if err != nil {
				return reflect.Value{}, err
			}
			if x.Kind == hir.ExpressionNominalWrap || x.Kind == hir.ExpressionNominalUnwrap {
				return b.walk(reflect.ValueOf(checked), depth+1)
			}
			checked.Type, err = b.scalar(checked.Type)
			if err != nil {
				return reflect.Value{}, err
			}
			v = reflect.ValueOf(checked)
		}
	}
	switch v.Kind() {
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			child, err := b.walk(v.Field(i), depth+1)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Field(i).Set(child)
		}
		return out, nil
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type()), nil
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			child, err := b.walk(v.Index(i), depth+1)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(child)
		}
		return out, nil
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type()), nil
		}
		child, err := b.walk(v.Elem(), depth+1)
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(child)
		return out, nil
	default:
		return v, nil
	}
}

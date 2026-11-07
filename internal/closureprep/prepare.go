// Package closureprep structurally lifts bounded copied local closures before typing.
package closureprep

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/syntax"
)

const prefix = "_fy_closure_"

type Error struct {
	Span   diagnostic.Span
	Detail string
}

func (e *Error) Error() string                       { return e.Detail }
func fail(span diagnostic.Span, detail string) error { return &Error{span, detail} }

type local struct {
	helper   string
	captures []string
}
type lifter struct {
	functions []ast.Function
	next      int
	rootNames map[string]bool
}

// Prepare leaves input syntax untouched; files without direct closure bindings take a fast path.
func Prepare(m *ast.Module) (*ast.Module, error) {
	if m == nil {
		return m, nil
	}
	active := false
	for _, b := range m.Bindings {
		active = active || b.Expression != nil && b.Expression.Kind == syntax.KindClosure
	}
	if !active {
		return m, nil
	}
	x := &lifter{}
	out := *m
	out.Functions = slices.Clone(m.Functions)
	for i, fn := range out.Functions {
		if strings.HasPrefix(fn.Name, prefix) {
			return nil, fail(fn.NameSpan, "reserved closure helper name")
		}
		x.rootNames = map[string]bool{}
		for _, p := range fn.Parameters {
			x.rootNames[p.Name] = true
		}
		body, err := x.block(fn.Body, map[string]local{}, true, fn.Name)
		if err != nil {
			return nil, err
		}
		out.Functions[i].Body = body
	}
	out.Functions = append(out.Functions, x.functions...)
	return &out, nil
}
func (x *lifter) block(b *ast.Block, env map[string]local, top bool, owner string) (*ast.Block, error) {
	if b == nil {
		return nil, nil
	}
	out := *b
	out.Statements = nil
	env = maps.Clone(env)
	declared := map[string]bool{}
	if top {
		declared = maps.Clone(x.rootNames)
	}
	for _, s := range b.Statements {
		if s.Binding != nil && s.Binding.Expression != nil && s.Binding.Expression.Kind == syntax.KindClosure {
			if !top || s.Binding.Mutable || s.Binding.Type != nil || declared[s.Binding.Name] {
				return nil, fail(s.Span, "closure requires a unique untyped let in the outer function body")
			}
			if strings.HasPrefix(s.Binding.Name, prefix) {
				return nil, fail(s.Span, "reserved closure binding name")
			}
			c := s.Binding.Expression.Closure
			if err := validate(c, s.Span); err != nil {
				return nil, err
			}
			x.next++
			name := fmt.Sprintf("%s%s_%d", prefix, owner, x.next)
			fn := c.Function
			fn.Name = name
			fn.NameSpan = s.Binding.NameSpan
			fn.Parameters = slices.Clone(c.Function.Parameters)
			info := local{helper: name}
			for i, cap := range c.Captures {
				hidden := fmt.Sprintf("%scapture_%d_%d", prefix, x.next, i)
				info.captures = append(info.captures, hidden)
				typ := cap.Type
				snapshot := &ast.Binding{Name: hidden, Span: cap.Span, NameSpan: cap.NameSpan, Type: &typ, Expression: &ast.Expression{Kind: syntax.KindPath, Text: cap.Name, Span: cap.NameSpan}}
				// Reading another closure as a scalar capture is not a value conversion.
				if _, ok := env[cap.Name]; ok {
					return nil, fail(cap.Span, "closure values cannot be captured")
				}
				out.Statements = append(out.Statements, ast.Statement{Kind: syntax.KindBinding, Span: cap.Span, Binding: snapshot})
			}
			fn.Parameters = append(slices.Clone(c.Captures), fn.Parameters...)
			x.functions = append(x.functions, fn)
			env[s.Binding.Name] = info
			declared[s.Binding.Name] = true
			continue
		}
		// A written binding may shadow an outer closure, but cannot redeclare a closure in this scope.
		if s.Binding != nil {
			if declared[s.Binding.Name] {
				if _, ok := env[s.Binding.Name]; ok {
					return nil, fail(s.Span, "duplicate closure binding")
				}
			}
		}
		next, err := x.statement(s, env, owner)
		if err != nil {
			return nil, err
		}
		out.Statements = append(out.Statements, next)
		if s.Binding != nil {
			delete(env, s.Binding.Name)
			declared[s.Binding.Name] = true
		}
	}
	var err error
	out.Tail, err = x.expr(b.Tail, env, owner)
	return &out, err
}
func (x *lifter) statement(s ast.Statement, env map[string]local, owner string) (ast.Statement, error) {
	originalExpression := s.Expression
	var err error
	if s.Binding != nil {
		if strings.HasPrefix(s.Binding.Name, prefix) {
			return s, fail(s.Span, "reserved closure local name")
		}
		b := *s.Binding
		b.Expression, err = x.expr(b.Expression, env, owner)
		if err != nil {
			return s, err
		}
		s.Binding = &b
	}
	s.Target, err = x.expr(s.Target, env, owner)
	if err != nil {
		return s, err
	}
	s.Expression, err = x.expr(s.Expression, env, owner)
	if err != nil {
		return s, err
	}
	if s.For != nil {
		v := *s.For
		env = maps.Clone(env)
		if v.Initializer != nil {
			a, e := x.statement(*v.Initializer, env, owner)
			if e != nil {
				return s, e
			}
			v.Initializer = &a
			if a.Binding != nil {
				delete(env, a.Binding.Name)
			}
		}
		v.Condition, err = x.expr(v.Condition, env, owner)
		if err != nil {
			return s, err
		}
		if v.Post != nil {
			a, e := x.statement(*v.Post, env, owner)
			if e != nil {
				return s, e
			}
			v.Post = &a
		}
		v.Body, err = x.block(v.Body, env, false, owner)
		if err != nil {
			return s, err
		}
		s.For = &v
	}
	if s.ByteFor != nil {
		v := *s.ByteFor
		v.View, err = x.expr(v.View, env, owner)
		if err != nil {
			return s, err
		}
		// The iterable sees the enclosing scope. The written item binding shadows
		// a closure only inside the body, just like the generated scalar loop.
		bodyEnv := maps.Clone(env)
		delete(bodyEnv, v.Name)
		v.Body, err = x.block(v.Body, bodyEnv, false, owner)
		if err != nil {
			return s, err
		}
		if v.PreparedVector != nil {
			prepared, e := x.expr(v.PreparedVector, env, owner)
			if e != nil {
				return s, e
			}
			v.PreparedVector = prepared
			if prepared.Block != nil && len(prepared.Block.Statements) > 0 && prepared.Block.Statements[0].Binding != nil {
				prepared.Block.Statements[0].Binding.Expression = v.View
			}
			if s.ByteFor.PreparedVector == originalExpression {
				s.Expression = prepared
			}
		}
		s.ByteFor = &v
	}
	return s, nil
}
func (x *lifter) expr(e *ast.Expression, env map[string]local, owner string) (*ast.Expression, error) {
	if e == nil {
		return nil, nil
	}
	r := *e
	if e.Kind == syntax.KindClosure {
		return nil, fail(e.Span, "closure literal is only admitted as an outer local initializer")
	}
	if e.Kind == syntax.KindPath {
		if strings.HasPrefix(e.Text, prefix) {
			return nil, fail(e.Span, "reserved closure name")
		}
		if _, ok := env[e.Text]; ok {
			return nil, fail(e.Span, "closure is callable locally but is not a first-class value")
		}
	}
	if e.Kind == syntax.KindCall && len(e.Children) > 0 && e.Children[0].Kind == syntax.KindPath {
		if v, ok := env[e.Children[0].Text]; ok {
			if e.CallLabels != nil {
				return nil, fail(e.Span, "closure calls require positional arguments")
			}
			r.Children = []*ast.Expression{{Kind: syntax.KindPath, Text: v.helper, Span: e.Children[0].Span}}
			for _, cap := range v.captures {
				r.Children = append(r.Children, &ast.Expression{Kind: syntax.KindPath, Text: cap, Span: e.Span})
			}
			for _, arg := range e.Children[1:] {
				a, err := x.expr(arg, env, owner)
				if err != nil {
					return nil, err
				}
				r.Children = append(r.Children, a)
			}
			return &r, nil
		}
	}
	r.Children = slices.Clone(e.Children)
	for i, c := range e.Children {
		a, err := x.expr(c, env, owner)
		if err != nil {
			return nil, err
		}
		r.Children[i] = a
	}
	var err error
	r.Block, err = x.block(e.Block, env, false, owner)
	return &r, err
}
func validate(c *ast.Closure, span diagnostic.Span) error {
	reject := func() error {
		return fail(span, "copied closure requires 0..4 i32 captures/parameters, i32 result and one pure arithmetic return of at most 64 nodes")
	}
	direct := func(t ast.TypeReference) bool {
		return t.Kind == ast.TypeKindNamed && t.Path == "i32" && len(t.Arguments) == 0
	}
	if c == nil || c.Function.Unsafe || c.Function.Public || c.Function.Owner != "" || len(c.Function.Attributes) != 0 || len(c.Function.TypeParameters) != 0 || len(c.Captures) > 4 || len(c.Function.Parameters) > 4 || !direct(c.Function.ResultType) || c.Function.Body == nil {
		return reject()
	}
	allowed := map[string]bool{}
	for _, p := range append(slices.Clone(c.Captures), c.Function.Parameters...) {
		if !direct(p.Type) || p.Mutable || allowed[p.Name] || strings.HasPrefix(p.Name, prefix) {
			return reject()
		}
		allowed[p.Name] = true
	}
	body := c.Function.Body
	value := body.Tail
	if len(body.Statements) == 1 && body.Tail == nil && body.Statements[0].Kind == syntax.KindReturn {
		value = body.Statements[0].Expression
	} else if len(body.Statements) != 0 {
		return reject()
	}
	nodes := 0
	var pure func(*ast.Expression) bool
	pure = func(e *ast.Expression) bool {
		nodes++
		if e == nil || nodes > 64 {
			return false
		}
		switch e.Kind {
		case syntax.KindLiteral:
			return len(e.Children) == 0
		case syntax.KindPath:
			return allowed[e.Text]
		case syntax.KindGroup:
			return len(e.Children) == 1 && pure(e.Children[0])
		case syntax.KindPrefix:
			return (e.Text == "+" || e.Text == "-") && len(e.Children) == 1 && pure(e.Children[0])
		case syntax.KindBinary:
			return slices.Contains([]string{"+", "-", "*", "/", "%"}, e.Text) && len(e.Children) == 2 && pure(e.Children[0]) && pure(e.Children[1])
		}
		return false
	}
	if !pure(value) {
		return reject()
	}
	return nil
}

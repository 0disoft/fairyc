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

const (
	prefix                     = "_fy_closure_"
	maxCopiedClosureCaptures   = 4
	maxCopiedClosureParameters = 4
)

type Error struct {
	Span   diagnostic.Span
	Detail string
}

func (e *Error) Error() string                       { return e.Detail }
func fail(span diagnostic.Span, detail string) error { return &Error{span, detail} }

type local struct {
	helper   string
	captures []string
	value    bool
}
type lifter struct {
	functions []ast.Function
	next      int
	rootNames map[string]bool
	generic   bool
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
		x.generic = fn.SourceGeneric || len(fn.TypeParameters) != 0
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

type blockScope struct {
	env      map[string]local
	declared map[string]bool // only the outer body can declare closures
	owned    bool
}

func (scope *blockScope) shadow(name string) {
	if scope.declared != nil {
		scope.declared[name] = true
	}
	if _, found := scope.env[name]; !found {
		return
	}
	// Nested blocks borrow the enclosing environment until a local hides a closure.
	if !scope.owned {
		scope.env = maps.Clone(scope.env)
		scope.owned = true
	}
	delete(scope.env, name)
}

func (x *lifter) block(b *ast.Block, env map[string]local, top bool, owner string) (*ast.Block, error) {
	if b == nil {
		return nil, nil
	}
	scope := blockScope{env: env}
	if top {
		scope = blockScope{env: maps.Clone(env), declared: maps.Clone(x.rootNames), owned: true}
	}
	out := *b
	out.Statements = nil
	for _, statement := range b.Statements {
		binding := statement.Binding
		if binding != nil && binding.Expression != nil && binding.Expression.Kind == syntax.KindClosure {
			if !top || binding.Mutable || scope.declared[binding.Name] {
				return nil, fail(statement.Span, "closure requires a unique immutable let in the outer function body")
			}
			statements, info, err := x.liftClosure(out.Statements, statement, scope.env, owner)
			if err != nil {
				return nil, err
			}
			out.Statements = statements
			scope.env[binding.Name], scope.declared[binding.Name] = info, true
			continue
		}
		next, err := x.rewriteBlockStatement(statement, &scope, owner)
		if err != nil {
			return nil, err
		}
		out.Statements = append(out.Statements, next)
	}
	var err error
	out.Tail, err = x.expr(b.Tail, scope.env, owner)
	return &out, err
}

func (x *lifter) rewriteBlockStatement(statement ast.Statement, scope *blockScope, owner string) (ast.Statement, error) {
	binding := statement.Binding
	if binding != nil && scope.declared[binding.Name] {
		if _, found := scope.env[binding.Name]; found {
			return statement, fail(statement.Span, "duplicate closure binding")
		}
	}
	next, err := x.statement(statement, scope.env, owner)
	if err != nil {
		return next, err
	}
	if binding != nil {
		scope.shadow(binding.Name)
	}
	return next, nil
}

// statements is the output buffer owned by the current block.
func (x *lifter) liftClosure(statements []ast.Statement, statement ast.Statement, env map[string]local, owner string) ([]ast.Statement, local, error) {
	binding := statement.Binding
	if strings.HasPrefix(binding.Name, prefix) {
		return nil, local{}, fail(statement.Span, "reserved closure binding name")
	}
	closure := binding.Expression.Closure
	if err := validate(closure, statement.Span); err != nil {
		return nil, local{}, err
	}
	if binding.Type != nil {
		if x.generic || !ClosureValueType(*binding.Type) || len(closure.Function.Parameters) != 1 {
			return nil, local{}, fail(statement.Span, "closure value requires monomorphic fn i32(i32) binding and literal signature")
		}
		prepared, err := x.valueBinding(statement, closure, env, owner)
		if err != nil {
			return nil, local{}, err
		}
		return append(statements, prepared), local{value: true}, nil
	}
	x.next++
	fn, info, snapshots, err := prepareDirectClosure(binding, closure, env, owner, x.next)
	if err != nil {
		return nil, local{}, err
	}
	x.functions = append(x.functions, fn)
	return append(statements, snapshots[:len(closure.Captures)]...), info, nil
}

func prepareDirectClosure(binding *ast.Binding, closure *ast.Closure, env map[string]local, owner string, ordinal int) (ast.Function, local, [maxCopiedClosureCaptures]ast.Statement, error) {
	var snapshots [maxCopiedClosureCaptures]ast.Statement
	name := fmt.Sprintf("%s%s_%d", prefix, owner, ordinal)
	info := local{helper: name}
	for i, capture := range closure.Captures {
		if _, found := env[capture.Name]; found {
			return ast.Function{}, local{}, snapshots, fail(capture.Span, "closure values cannot be captured")
		}
		hidden := fmt.Sprintf("%scapture_%d_%d", prefix, ordinal, i)
		info.captures = append(info.captures, hidden)
		typ := capture.Type
		snapshot := &ast.Binding{Name: hidden, Span: capture.Span, NameSpan: capture.NameSpan, Type: &typ, Expression: &ast.Expression{Kind: syntax.KindPath, Text: capture.Name, Span: capture.NameSpan}}
		snapshots[i] = ast.Statement{Kind: syntax.KindBinding, Span: capture.Span, Binding: snapshot}
	}
	fn := closure.Function
	fn.Name, fn.NameSpan = name, binding.NameSpan
	fn.Parameters = append(slices.Clone(closure.Captures), closure.Function.Parameters...)
	return fn, info, snapshots, nil
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
		if v, ok := env[e.Text]; ok && !v.value {
			return nil, fail(e.Span, "closure is callable locally but is not a first-class value")
		}
	}
	if e.Kind == syntax.KindCall && len(e.Children) > 0 && e.Children[0].Kind == syntax.KindPath {
		if v, ok := env[e.Children[0].Text]; ok && !v.value {
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
		return fail(span, "copied closure requires 0..4 copied i32/u32 captures/parameters, i32/u32 result and one pure arithmetic return of at most 64 nodes")
	}
	direct := func(t ast.TypeReference) bool {
		return t.Kind == ast.TypeKindNamed && (t.Path == "i32" || t.Path == "u32") && len(t.Arguments) == 0
	}
	if c == nil || c.Function.Unsafe || c.Function.Public || c.Function.Owner != "" || len(c.Function.Attributes) != 0 || len(c.Function.TypeParameters) != 0 ||
		len(c.Captures) > maxCopiedClosureCaptures || len(c.Function.Parameters) > maxCopiedClosureParameters || !direct(c.Function.ResultType) || c.Function.Body == nil {
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
	bindings := body.Statements
	if len(bindings) > 0 && body.Tail == nil && bindings[len(bindings)-1].Kind == syntax.KindReturn {
		value = bindings[len(bindings)-1].Expression
		bindings = bindings[:len(bindings)-1]
	}
	if len(bindings) > 16 {
		return reject()
	}
	nodes := 0
	// Existing zero-local returns retain their original admission and diagnostics.
	// Initializers must be integer expressions even when their inferred value is unused.
	initializing := false
	var pure func(*ast.Expression) bool
	pure = func(e *ast.Expression) bool {
		nodes++
		if e == nil || nodes > 64 {
			return false
		}
		switch e.Kind {
		case syntax.KindLiteral:
			if len(e.Children) != 0 {
				return false
			}
			if initializing {
				// AST literals retain their original spelling, not a token kind.
				// Syntax already validates digits/separators; discriminate integer
				// tokens without moving contextual integer range checking before typing.
				raw := strings.ToLower(e.Text)
				return len(raw) != 0 && raw[0] >= '0' && raw[0] <= '9' &&
					(strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0b") || !strings.ContainsAny(raw, ".e"))
			}
			return true
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
	initializing = true
	for _, statement := range bindings {
		binding := statement.Binding
		if statement.Kind != syntax.KindBinding || binding == nil || binding.Mutable ||
			binding.Type != nil && !direct(*binding.Type) || allowed[binding.Name] || strings.HasPrefix(binding.Name, prefix) {
			return reject()
		}
		// A name becomes visible only after its initializer has been checked.
		if !pure(binding.Expression) {
			return reject()
		}
		allowed[binding.Name] = true
	}
	initializing = false
	if !pure(value) {
		return reject()
	}
	return nil
}

package semantic

import (
	"math"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

const (
	constEvalAttribute       = "const_eval"
	compileTimeMaxCallDepth  = 32
	compileTimeMaxNodeBudget = 65536
)

type compileTimeValue struct {
	typ hir.Type
	b   bool
	i   int64
	u   uint64
}

func (v compileTimeValue) canonical() string {
	if v.typ == hir.TypeBool {
		return strconv.FormatBool(v.b)
	}
	if v.typ == hir.TypeI32 || v.typ == hir.TypeI64 {
		return strconv.FormatInt(v.i, 10)
	}
	return strconv.FormatUint(v.u, 10)
}

func compileTimeScalar(value hir.Type) bool {
	return value == hir.TypeBool || value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 || value == hir.TypeU32
}

type compileTimeEvaluator struct {
	analyzer  *analyzer
	budgets   map[hir.PackageID]int
	exhausted map[hir.PackageID]bool
	state     map[*symbol]uint8
	invalid   map[*symbol]bool
}

func (a *analyzer) prepareCompileTime() {
	evaluator := &compileTimeEvaluator{
		analyzer: a, budgets: make(map[hir.PackageID]int), exhausted: make(map[hir.PackageID]bool), state: make(map[*symbol]uint8), invalid: make(map[*symbol]bool),
	}
	for _, module := range a.moduleOrder {
		a.rejectMisplacedConstEvalAttributes(module)
		for index := range module.file.Module.Constants {
			constant := &module.file.Module.Constants[index]
			value := a.constants[constant]
			if value == nil {
				continue
			}
			value.typ = a.canonicalType(module, constant.Type)
			if !compileTimeScalar(value.typ) {
				a.addDiagnostic("E4301", span(module.file.Path, constant.Type.Span))
				evaluator.invalid[value] = true
			}
		}
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			value := a.functions[function]
			if value == nil {
				continue
			}
			seen := false
			valid := true
			reported := false
			for _, attribute := range function.Attributes {
				if attribute.Name != constEvalAttribute {
					continue
				}
				if seen || len(attribute.Arguments) != 0 {
					a.addDiagnostic("E4301", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
					valid = false
					reported = true
				}
				seen = true
			}
			if !seen {
				continue
			}
			resultType := a.canonicalType(module, function.ResultType)
			if function.Public || function.Unsafe || function.Owner != "" || !compileTimeScalar(resultType) || len(value.effects) != 0 || !singleReturnExpression(function) {
				valid = false
			}
			for parameterIndex := range function.Parameters {
				parameter := &function.Parameters[parameterIndex]
				if parameter.Mutable || !compileTimeScalar(a.canonicalType(module, parameter.Type)) {
					valid = false
				}
			}
			if !valid && !reported {
				a.addDiagnostic("E4301", span(module.file.Path, function.NameSpan))
			}
			if !valid {
				continue
			}
			a.constEvalFunctions[value] = true
		}
	}
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Constants {
			constant := &module.file.Module.Constants[index]
			if value := a.constants[constant]; value != nil && !evaluator.invalid[value] && !evaluator.validateExpression(module, constant.Expression, nil) {
				evaluator.invalid[value] = true
			}
		}
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if value := a.functions[function]; value != nil && a.constEvalFunctions[value] && !evaluator.validateExpression(module, function.Body.Statements[0].Expression, function) {
				evaluator.invalid[value] = true
			}
		}
	}
	evaluator.rejectCycles()
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Constants {
			constant := &module.file.Module.Constants[index]
			value := a.constants[constant]
			if value == nil || evaluator.invalid[value] {
				continue
			}
			computed, ok := evaluator.evalConstant(value, 0)
			if !ok {
				continue
			}
			a.constantFacts = append(a.constantFacts, ConstantFact{
				Package: value.packageID, Module: value.module, Name: value.name,
				Declaration: span(value.source, value.declaration), Type: computed.typ,
				Value: computed.canonical(), Span: span(value.source, constant.Span),
			})
		}
		for index := range module.file.Module.StaticAssertions {
			assertion := &module.file.Module.StaticAssertions[index]
			if !evaluator.validateExpression(module, assertion.Expression, nil) {
				continue
			}
			computed, ok := evaluator.eval(module, assertion.Expression, hir.TypeBool, nil, 0)
			if !ok {
				continue
			}
			if computed.typ != hir.TypeBool {
				a.addDiagnostic("E1007", span(module.file.Path, assertion.Expression.Span))
			} else if !computed.b {
				a.addDiagnostic("E4305", span(module.file.Path, assertion.Span))
			}
		}
	}
}

func (a *analyzer) rejectMisplacedConstEvalAttributes(module *moduleScope) {
	reject := func(attributes []ast.Attribute, declaration hir.Span) {
		for _, attribute := range attributes {
			if attribute.Name == constEvalAttribute {
				a.addDiagnostic("E4301", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: declaration})
			}
		}
	}
	for index := range module.file.Module.Structs {
		value := &module.file.Module.Structs[index]
		reject(value.Attributes, span(module.file.Path, value.NameSpan))
	}
	for index := range module.file.Module.Enums {
		value := &module.file.Module.Enums[index]
		reject(value.Attributes, span(module.file.Path, value.NameSpan))
	}
	for index := range module.file.Module.Resources {
		value := &module.file.Module.Resources[index]
		reject(value.Attributes, span(module.file.Path, value.NameSpan))
		if value.Cleanup != nil {
			reject(value.Cleanup.Attributes, span(module.file.Path, value.Cleanup.Span))
		}
	}
	for index := range module.file.Module.ForeignFunctions {
		value := &module.file.Module.ForeignFunctions[index]
		reject(value.Attributes, span(module.file.Path, value.NameSpan))
	}
}

func (e *compileTimeEvaluator) validateExpression(module *moduleScope, expression *ast.Expression, function *ast.Function) bool {
	if expression == nil {
		e.analyzer.addDiagnostic("E4302", span(module.file.Path, module.file.Module.Span))
		return false
	}
	validChildren := func() bool {
		for _, child := range expression.Children {
			if !e.validateExpression(module, child, function) {
				return false
			}
		}
		return true
	}
	switch expression.Kind {
	case syntax.KindLiteral:
		if expression.Text == "true" || expression.Text == "false" || isIntegerLiteral(expression.Text) {
			return true
		}
	case syntax.KindPath:
		target := e.analyzer.resolved[expression]
		if target != nil && target.constant != nil && target.module == module.file.Module.Path && target.packageID == module.key.packageID {
			return true
		}
		if target != nil && target.kind == "parameter" && function != nil {
			for index := range function.Parameters {
				if e.analyzer.parameter[&function.Parameters[index]] == target {
					return true
				}
			}
		}
	case syntax.KindGroup:
		if len(expression.Children) == 1 {
			return validChildren()
		}
	case syntax.KindPrefix:
		if len(expression.Children) == 1 && (expression.Text == "+" || expression.Text == "-" || expression.Text == "!" || expression.Text == "~") {
			return validChildren()
		}
	case syntax.KindBinary:
		if len(expression.Children) == 2 && strings.Contains(" && || == != < <= > >= + - * / % << >> & ^ | ", " "+expression.Text+" ") {
			return validChildren()
		}
	case syntax.KindCast:
		if len(expression.Children) == 1 && expression.Type != nil && compileTimeScalar(e.analyzer.canonicalType(module, *expression.Type)) {
			return validChildren()
		}
	case syntax.KindIf:
		if len(expression.Children) == 3 {
			return validChildren()
		}
	case syntax.KindCall:
		if len(expression.Children) > 0 {
			callee := e.analyzer.resolved[expression.Children[0]]
			if callee != nil && e.analyzer.constEvalFunctions[callee] && callee.module == module.file.Module.Path && callee.packageID == module.key.packageID && len(expression.Children)-1 == len(callee.function.Parameters) {
				for _, argument := range expression.Children[1:] {
					if !e.validateExpression(module, argument, function) {
						return false
					}
				}
				return true
			}
		}
	case syntax.KindBlock:
		if expression.Block != nil && len(expression.Block.Statements) == 0 && expression.Block.Tail != nil {
			return e.validateExpression(module, expression.Block.Tail, function)
		}
	}
	e.analyzer.addDiagnostic("E4302", span(module.file.Path, expression.Span))
	return false
}

func singleReturnExpression(function *ast.Function) bool {
	return function.Body != nil && len(function.Body.Statements) == 1 && function.Body.Tail == nil &&
		function.Body.Statements[0].Kind == syntax.KindReturn && function.Body.Statements[0].Expression != nil
}

func (e *compileTimeEvaluator) rejectCycles() {
	visiting := make(map[*symbol]bool)
	visited := make(map[*symbol]bool)
	var visit func(*symbol)
	visit = func(node *symbol) {
		if node == nil || visited[node] || e.invalid[node] {
			return
		}
		if visiting[node] {
			e.analyzer.addDiagnostic("E4304", span(node.source, node.declaration))
			e.invalid[node] = true
			return
		}
		visiting[node] = true
		for _, dependency := range e.dependencies(node) {
			visit(dependency)
			if e.invalid[dependency] {
				e.invalid[node] = true
			}
		}
		delete(visiting, node)
		visited[node] = true
	}
	for _, module := range e.analyzer.moduleOrder {
		for index := range module.file.Module.Constants {
			visit(e.analyzer.constants[&module.file.Module.Constants[index]])
		}
		for index := range module.file.Module.Functions {
			node := e.analyzer.functions[&module.file.Module.Functions[index]]
			if e.analyzer.constEvalFunctions[node] {
				visit(node)
			}
		}
	}
}

func (e *compileTimeEvaluator) dependencies(node *symbol) []*symbol {
	var root *ast.Expression
	if node.constant != nil {
		root = node.constant.Expression
	} else if node.function != nil && singleReturnExpression(node.function) {
		root = node.function.Body.Statements[0].Expression
	}
	seen := make(map[*symbol]bool)
	var result []*symbol
	var walk func(*ast.Expression)
	walk = func(expression *ast.Expression) {
		if expression == nil {
			return
		}
		if target := e.analyzer.resolved[expression]; target != nil && (target.constant != nil || e.analyzer.constEvalFunctions[target]) && !seen[target] {
			seen[target] = true
			result = append(result, target)
		}
		for _, child := range expression.Children {
			walk(child)
		}
		if expression.Block != nil {
			walk(expression.Block.Tail)
			for index := range expression.Block.Statements {
				walk(expression.Block.Statements[index].Expression)
			}
		}
	}
	walk(root)
	return result
}

func (e *compileTimeEvaluator) evalConstant(value *symbol, depth int) (compileTimeValue, bool) {
	if computed, ok := e.analyzer.constantValues[value]; ok {
		return computed, true
	}
	if value == nil || value.constant == nil || e.invalid[value] {
		return compileTimeValue{}, false
	}
	if e.state[value] == 1 {
		e.analyzer.addDiagnostic("E4304", span(value.source, value.declaration))
		e.invalid[value] = true
		return compileTimeValue{}, false
	}
	e.state[value] = 1
	module := e.analyzer.symbolModule(value)
	computed, ok := e.eval(module, value.constant.Expression, value.typ, nil, depth)
	e.state[value] = 2
	if !ok {
		return compileTimeValue{}, false
	}
	if computed.typ != value.typ {
		e.analyzer.addDiagnostic("E1006", span(value.source, value.constant.Expression.Span), RelatedSpan{Role: "expected-type", Span: span(value.source, value.constant.Type.Span)})
		return compileTimeValue{}, false
	}
	e.analyzer.constantValues[value] = computed
	return computed, true
}

func (e *compileTimeEvaluator) eval(module *moduleScope, expression *ast.Expression, expected hir.Type, environment map[*symbol]compileTimeValue, depth int) (compileTimeValue, bool) {
	if e.analyzer.canceled() {
		return compileTimeValue{}, false
	}
	if expression == nil || module == nil {
		return compileTimeValue{}, false
	}
	if e.exhausted[module.key.packageID] {
		return compileTimeValue{}, false
	}
	e.budgets[module.key.packageID]++
	if e.budgets[module.key.packageID] > compileTimeMaxNodeBudget {
		e.analyzer.addDiagnostic("E4304", span(module.file.Path, expression.Span))
		e.exhausted[module.key.packageID] = true
		return compileTimeValue{}, false
	}
	switch expression.Kind {
	case syntax.KindLiteral:
		return e.literal(module, expression, expected)
	case syntax.KindPath:
		target := e.analyzer.resolved[expression]
		if value, ok := environment[target]; ok {
			return value, true
		}
		if target != nil && target.constant != nil && target.module == module.file.Module.Path && target.packageID == module.key.packageID {
			return e.evalConstant(target, depth)
		}
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	case syntax.KindGroup:
		if len(expression.Children) == 1 {
			return e.eval(module, expression.Children[0], expected, environment, depth)
		}
	case syntax.KindPrefix:
		return e.prefix(module, expression, expected, environment, depth)
	case syntax.KindBinary:
		return e.binary(module, expression, expected, environment, depth)
	case syntax.KindCast:
		return e.cast(module, expression, environment, depth)
	case syntax.KindIf:
		return e.conditional(module, expression, expected, environment, depth)
	case syntax.KindCall:
		return e.call(module, expression, environment, depth)
	case syntax.KindBlock:
		if expression.Block != nil && len(expression.Block.Statements) == 0 && expression.Block.Tail != nil {
			return e.eval(module, expression.Block.Tail, expected, environment, depth)
		}
	}
	e.rejectExpression(module, expression)
	return compileTimeValue{}, false
}

func (e *compileTimeEvaluator) rejectExpression(module *moduleScope, expression *ast.Expression) {
	e.analyzer.addDiagnostic("E4302", span(module.file.Path, expression.Span))
}

func (e *compileTimeEvaluator) literal(module *moduleScope, expression *ast.Expression, expected hir.Type) (compileTimeValue, bool) {
	if expression.Text == "true" || expression.Text == "false" {
		return compileTimeValue{typ: hir.TypeBool, b: expression.Text == "true"}, true
	}
	typ := expected
	if !compileTimeScalar(typ) || typ == hir.TypeBool {
		typ = hir.TypeI32
	}
	if !isIntegerLiteral(expression.Text) || !integerLiteralFits(expression.Text, typ) {
		e.analyzer.addDiagnostic("E4303", span(module.file.Path, expression.Span))
		return compileTimeValue{}, false
	}
	raw := strings.ReplaceAll(expression.Text, "_", "")
	base := 10
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		base, raw = 16, raw[2:]
	} else if strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B") {
		base, raw = 2, raw[2:]
	}
	value, err := strconv.ParseUint(raw, base, 64)
	if err != nil {
		e.analyzer.addDiagnostic("E4303", span(module.file.Path, expression.Span))
		return compileTimeValue{}, false
	}
	if typ == hir.TypeI32 || typ == hir.TypeI64 {
		return compileTimeValue{typ: typ, i: int64(value)}, true
	}
	return compileTimeValue{typ: typ, u: value}, true
}

func (e *compileTimeEvaluator) prefix(module *moduleScope, expression *ast.Expression, expected hir.Type, environment map[*symbol]compileTimeValue, depth int) (compileTimeValue, bool) {
	if len(expression.Children) != 1 {
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	width := uint(32)
	minimum := int64(math.MinInt32)
	if expected == hir.TypeI64 {
		width, minimum = 64, math.MinInt64
	}
	if expression.Text == "-" && (expected == hir.TypeI32 || expected == hir.TypeI64) && directSignedMinimumLiteral(expression.Children[0], width) {
		if e.exhausted[module.key.packageID] {
			return compileTimeValue{}, false
		}
		e.budgets[module.key.packageID]++
		if e.budgets[module.key.packageID] > compileTimeMaxNodeBudget {
			e.analyzer.addDiagnostic("E4304", span(module.file.Path, expression.Children[0].Span))
			e.exhausted[module.key.packageID] = true
			return compileTimeValue{}, false
		}
		return compileTimeValue{typ: expected, i: minimum}, true
	}
	value, ok := e.eval(module, expression.Children[0], expected, environment, depth)
	if !ok {
		return compileTimeValue{}, false
	}
	switch expression.Text {
	case "!":
		if value.typ == hir.TypeBool {
			return compileTimeValue{typ: hir.TypeBool, b: !value.b}, true
		}
	case "+":
		if value.typ == hir.TypeI32 || value.typ == hir.TypeI64 {
			return value, true
		}
	case "-":
		if value.typ == hir.TypeI32 && value.i != math.MinInt32 || value.typ == hir.TypeI64 && value.i != math.MinInt64 {
			return compileTimeValue{typ: value.typ, i: -value.i}, true
		}
	case "~":
		switch value.typ {
		case hir.TypeU8:
			return compileTimeValue{typ: value.typ, u: uint64(^uint8(value.u))}, true
		case hir.TypeU32:
			return compileTimeValue{typ: value.typ, u: uint64(^uint32(value.u))}, true
		case hir.TypeI32:
			return compileTimeValue{typ: value.typ, i: int64(^int32(value.i))}, true
		case hir.TypeI64:
			return compileTimeValue{typ: value.typ, i: ^value.i}, true
		}
	}
	e.analyzer.addDiagnostic("E4303", span(module.file.Path, expression.Span))
	return compileTimeValue{}, false
}

func (e *compileTimeEvaluator) binary(module *moduleScope, expression *ast.Expression, expected hir.Type, environment map[*symbol]compileTimeValue, depth int) (compileTimeValue, bool) {
	if len(expression.Children) != 2 {
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	if contextualIntegerTree(expression.Children[0]) && !contextualIntegerTree(expression.Children[1]) && contextualIntegerOperator(expression.Text) {
		// The flexible side has no names, calls or effects. Evaluate the fixed
		// peer once to select its type without converting that peer's value.
		right, ok := e.eval(module, expression.Children[1], integerBinaryContext(expression.Text, expected), environment, depth)
		if !ok {
			return compileTimeValue{}, false
		}
		left, ok := e.eval(module, expression.Children[0], integerPeerContext(right.typ, integerBinaryContext(expression.Text, expected)), environment, depth)
		if !ok {
			return compileTimeValue{}, false
		}
		if left.typ != right.typ {
			e.analyzer.addDiagnostic("E1006", span(module.file.Path, expression.Children[1].Span))
			return compileTimeValue{}, false
		}
		return e.applyBinary(module, expression, left, right)
	}
	left, ok := e.eval(module, expression.Children[0], expected, environment, depth)
	if !ok {
		return compileTimeValue{}, false
	}
	if expression.Text == "&&" || expression.Text == "||" {
		if left.typ != hir.TypeBool {
			e.analyzer.addDiagnostic("E1007", span(module.file.Path, expression.Children[0].Span))
			return compileTimeValue{}, false
		}
		if expression.Text == "&&" && !left.b || expression.Text == "||" && left.b {
			return left, true
		}
	}
	right, ok := e.eval(module, expression.Children[1], left.typ, environment, depth)
	if !ok {
		return compileTimeValue{}, false
	}
	if left.typ != right.typ {
		e.analyzer.addDiagnostic("E1006", span(module.file.Path, expression.Children[1].Span))
		return compileTimeValue{}, false
	}
	return e.applyBinary(module, expression, left, right)
}

func (e *compileTimeEvaluator) applyBinary(module *moduleScope, expression *ast.Expression, left, right compileTimeValue) (compileTimeValue, bool) {
	op := expression.Text
	if left.typ == hir.TypeBool {
		switch op {
		case "&&":
			return compileTimeValue{typ: hir.TypeBool, b: left.b && right.b}, true
		case "||":
			return compileTimeValue{typ: hir.TypeBool, b: left.b || right.b}, true
		case "==":
			return compileTimeValue{typ: hir.TypeBool, b: left.b == right.b}, true
		case "!=":
			return compileTimeValue{typ: hir.TypeBool, b: left.b != right.b}, true
		}
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	if op == "==" || op == "!=" || op == "<" || op == "<=" || op == ">" || op == ">=" {
		comparison := 0
		if left.typ == hir.TypeI32 || left.typ == hir.TypeI64 {
			if left.i < right.i {
				comparison = -1
			} else if left.i > right.i {
				comparison = 1
			}
		} else if left.u < right.u {
			comparison = -1
		} else if left.u > right.u {
			comparison = 1
		}
		result := op == "==" && comparison == 0 || op == "!=" && comparison != 0 || op == "<" && comparison < 0 || op == "<=" && comparison <= 0 || op == ">" && comparison > 0 || op == ">=" && comparison >= 0
		return compileTimeValue{typ: hir.TypeBool, b: result}, true
	}
	var signed int64
	var unsigned uint64
	trap := false
	if left.typ == hir.TypeI64 {
		return e.applyI64Binary(module, expression, left.i, right.i)
	}
	if left.typ == hir.TypeI32 {
		l, r := left.i, right.i
		switch op {
		case "+":
			signed = l + r
			trap = signed < math.MinInt32 || signed > math.MaxInt32
		case "-":
			signed = l - r
			trap = signed < math.MinInt32 || signed > math.MaxInt32
		case "*":
			signed = l * r
			trap = signed < math.MinInt32 || signed > math.MaxInt32
		case "/":
			trap = r == 0 || l == math.MinInt32 && r == -1
			if !trap {
				signed = l / r
			}
		case "%":
			trap = r == 0 || l == math.MinInt32 && r == -1
			if !trap {
				signed = l % r
			}
		case "&":
			signed = int64(int32(l) & int32(r))
		case "^":
			signed = int64(int32(l) ^ int32(r))
		case "|":
			signed = int64(int32(l) | int32(r))
		case ">>":
			trap = r < 0 || r >= 32
			if !trap {
				signed = int64(int32(l) >> uint(r))
			}
		case "<<":
			trap = r < 0 || r >= 32
			if !trap {
				signed = l * int64(uint64(1)<<uint(r))
				trap = signed < math.MinInt32 || signed > math.MaxInt32
			}
		default:
			e.rejectExpression(module, expression)
			return compileTimeValue{}, false
		}
	} else {
		l, r := left.u, right.u
		maximum := uint64(math.MaxUint32)
		if left.typ == hir.TypeU8 {
			maximum = math.MaxUint8
		}
		switch op {
		case "+":
			unsigned = l + r
			trap = unsigned > maximum
		case "-":
			trap = l < r
			if !trap {
				unsigned = l - r
			}
		case "*":
			unsigned = l * r
			trap = r != 0 && unsigned/r != l || unsigned > maximum
		case "/":
			trap = r == 0
			if !trap {
				unsigned = l / r
			}
		case "%":
			trap = r == 0
			if !trap {
				unsigned = l % r
			}
		case "&":
			unsigned = l & r
		case "^":
			unsigned = l ^ r
		case "|":
			unsigned = l | r
		case ">>":
			trap = r >= uint64(bitWidth(left.typ))
			if !trap {
				unsigned = l >> uint(r)
			}
		case "<<":
			trap = r >= uint64(bitWidth(left.typ))
			if !trap {
				unsigned = l << uint(r)
				trap = unsigned > maximum || unsigned>>uint(r) != l
			}
		default:
			e.rejectExpression(module, expression)
			return compileTimeValue{}, false
		}
	}
	if trap {
		e.analyzer.addDiagnostic("E4303", span(module.file.Path, expression.Span))
		return compileTimeValue{}, false
	}
	return compileTimeValue{typ: left.typ, i: signed, u: unsigned}, true
}

func bitWidth(value hir.Type) int {
	if value == hir.TypeI64 {
		return 64
	}
	if value == hir.TypeU8 {
		return 8
	}
	return 32
}

func (e *compileTimeEvaluator) cast(module *moduleScope, expression *ast.Expression, environment map[*symbol]compileTimeValue, depth int) (compileTimeValue, bool) {
	if len(expression.Children) != 1 || expression.Type == nil {
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	target := e.analyzer.canonicalType(module, *expression.Type)
	value, ok := e.eval(module, expression.Children[0], hir.TypeInvalid, environment, depth)
	if !ok {
		return compileTimeValue{}, false
	}
	if target == hir.TypeI64 && (value.typ == hir.TypeI32 || value.typ == hir.TypeU8 || value.typ == hir.TypeU32) {
		signed := value.i
		if value.typ != hir.TypeI32 {
			signed = int64(value.u)
		}
		return compileTimeValue{typ: target, i: signed}, true
	}
	if value.typ == hir.TypeU8 && target == hir.TypeI32 {
		return compileTimeValue{typ: target, i: int64(value.u)}, true
	}
	if value.typ == hir.TypeU8 && target == hir.TypeU32 {
		return compileTimeValue{typ: target, u: value.u}, true
	}
	e.analyzer.addDiagnostic("E4303", span(module.file.Path, expression.Span))
	return compileTimeValue{}, false
}

func (e *compileTimeEvaluator) conditional(module *moduleScope, expression *ast.Expression, expected hir.Type, environment map[*symbol]compileTimeValue, depth int) (compileTimeValue, bool) {
	if len(expression.Children) != 3 {
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	condition, ok := e.eval(module, expression.Children[0], hir.TypeBool, environment, depth)
	if !ok {
		return compileTimeValue{}, false
	}
	if condition.typ != hir.TypeBool {
		e.analyzer.addDiagnostic("E1007", span(module.file.Path, expression.Children[0].Span))
		return compileTimeValue{}, false
	}
	selected := expression.Children[2]
	if condition.b {
		selected = expression.Children[1]
	}
	return e.eval(module, selected, expected, environment, depth)
}

func (e *compileTimeEvaluator) call(module *moduleScope, expression *ast.Expression, environment map[*symbol]compileTimeValue, depth int) (compileTimeValue, bool) {
	if !e.analyzer.checkCallArgumentLabels(module, expression) {
		return compileTimeValue{}, false
	}
	if len(expression.Children) == 0 {
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	callee := e.analyzer.resolved[expression.Children[0]]
	if callee == nil || !e.analyzer.constEvalFunctions[callee] || callee.module != module.file.Module.Path || callee.packageID != module.key.packageID {
		e.rejectExpression(module, expression)
		return compileTimeValue{}, false
	}
	if depth >= compileTimeMaxCallDepth {
		e.analyzer.addDiagnostic("E4304", span(module.file.Path, expression.Span))
		return compileTimeValue{}, false
	}
	arguments := expression.Children[1:]
	if len(arguments) != len(callee.function.Parameters) {
		e.analyzer.addDiagnostic("E1010", span(module.file.Path, expression.Span))
		return compileTimeValue{}, false
	}
	bindings := make(map[*symbol]compileTimeValue, len(arguments))
	calleeModule := e.analyzer.symbolModule(callee)
	for index, argument := range arguments {
		parameter := &callee.function.Parameters[index]
		typ := e.analyzer.canonicalType(calleeModule, parameter.Type)
		value, ok := e.eval(module, argument, typ, environment, depth)
		if !ok {
			return compileTimeValue{}, false
		}
		if value.typ != typ {
			e.analyzer.addDiagnostic("E1006", span(module.file.Path, argument.Span))
			return compileTimeValue{}, false
		}
		bindings[e.analyzer.parameter[parameter]] = value
	}
	resultType := e.analyzer.canonicalType(calleeModule, callee.function.ResultType)
	return e.eval(calleeModule, callee.function.Body.Statements[0].Expression, resultType, bindings, depth+1)
}

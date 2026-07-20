package semantic

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type typeChecker struct {
	analyzer     *analyzer
	module       *moduleScope
	function     *ast.Function
	hir          *hir.Function
	loopDepth    int
	unsafeScopes []hir.Span
}

func (a *analyzer) typeCheck() Result {
	result := Result{}
	a.prepareAggregateIdentities()
	a.prepareResourceTypes()
	a.prepareStructTypes()
	a.prepareEnumTypes()
	for _, module := range a.moduleOrder {
		hirModule := hir.Module{Path: module.file.Module.Path, SourcePath: module.file.Path}
		for index := range module.file.Module.Structs {
			structure := &module.file.Module.Structs[index]
			if a.structs[structure] == nil {
				continue
			}
			hirModule.Structs = append(hirModule.Structs, a.checkStruct(module, structure))
		}
		for index := range module.file.Module.Enums {
			enumeration := &module.file.Module.Enums[index]
			if a.enums[enumeration] == nil {
				continue
			}
			hirModule.Enums = append(hirModule.Enums, a.checkEnum(module, enumeration))
		}
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			if a.resources[resource] == nil {
				continue
			}
			hirModule.Resources = append(hirModule.Resources, a.checkResource(module, resource))
		}
		for index := range module.file.Module.ForeignFunctions {
			function := &module.file.Module.ForeignFunctions[index]
			if a.foreignFunctions[function] == nil {
				continue
			}
			hirModule.ForeignFunctions = append(hirModule.ForeignFunctions, a.checkForeignFunction(module, function))
		}
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if a.functions[function] == nil {
				continue
			}
			checker := typeChecker{analyzer: a, module: module, function: function}
			hirModule.Functions = append(hirModule.Functions, checker.checkFunction())
		}
		result.HIR.Modules = append(result.HIR.Modules, hirModule)
	}
	return result
}

func (a *analyzer) checkEnum(module *moduleScope, enumeration *ast.Enum) hir.Enum {
	value := a.enums[enumeration]
	for _, attribute := range enumeration.Attributes {
		a.addUnsupported(module.file.Path, attribute.Span, "enum attribute @"+attribute.Name)
	}
	result := hir.Enum{Name: enumeration.Name, Type: value.typ, Span: span(module.file.Path, enumeration.Span)}
	if info := a.enumTypes[value.typ]; info != nil {
		for _, variant := range info.orderedVariants {
			hirVariant := hir.EnumVariant{Name: variant.name, Span: variant.span}
			for _, field := range variant.orderedFields {
				hirVariant.Fields = append(hirVariant.Fields, hir.EnumField{Name: field.name, Type: field.typ, Span: field.span})
			}
			result.Variants = append(result.Variants, hirVariant)
		}
	}
	return result
}

func (a *analyzer) checkStruct(module *moduleScope, structure *ast.Struct) hir.Struct {
	value := a.structs[structure]
	result := hir.Struct{Name: structure.Name, Type: value.typ, Span: span(module.file.Path, structure.Span)}
	if info := a.structTypes[value.typ]; info != nil {
		result.ReprC = info.reprC
		for _, field := range info.orderedFields {
			result.Fields = append(result.Fields, hir.StructField{Name: field.name, Type: field.typ, Span: field.span})
		}
	}
	return result
}

func (a *analyzer) checkForeignFunction(module *moduleScope, function *ast.ForeignFunction) hir.ForeignFunction {
	symbol := a.foreignFunctions[function]
	linkName := function.Name
	seenLinkName := false
	for _, attribute := range function.Attributes {
		if attribute.Name != "link_name" {
			a.addUnsupported(module.file.Path, attribute.Span, "foreign function attribute @"+attribute.Name)
			continue
		}
		if seenLinkName || len(attribute.Arguments) != 1 || len(attribute.Arguments[0].Raw) < 2 || attribute.Arguments[0].Raw[0] != '"' || attribute.Arguments[0].Cooked == "" {
			a.addDiagnostic("E5106", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
			seenLinkName = true
			continue
		}
		seenLinkName = true
		linkName = attribute.Arguments[0].Cooked
	}
	result := hir.ForeignFunction{
		Symbol: symbol.id, Name: function.Name, LinkName: linkName, ABI: function.ABI,
		Span: span(module.file.Path, function.Span), ResultType: a.canonicalType(module, function.ResultType),
	}
	symbol.typ = result.ResultType
	if _, _, borrow := hir.BorrowElement(result.ResultType); borrow {
		a.addDiagnostic("E5102", span(module.file.Path, function.ResultType.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
	}
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		parameterType := a.canonicalType(module, parameter.Type)
		if _, _, borrow := hir.BorrowElement(parameterType); borrow {
			a.addDiagnostic("E5102", span(module.file.Path, parameter.Type.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
		}
		parameterSpan := span(module.file.Path, parameter.NameSpan)
		result.Parameters = append(result.Parameters, hir.ForeignParameter{Name: parameter.Name, Type: parameterType, Span: parameterSpan})
		if parameterType != hir.TypeInvalid {
			a.types = append(a.types, TypeFact{Expression: parameterSpan, CanonicalType: parameterType})
		}
	}
	if result.ResultType != hir.TypeInvalid {
		a.types = append(a.types, TypeFact{Expression: span(module.file.Path, function.ResultType.Span), CanonicalType: result.ResultType})
	}
	return result
}

func (a *analyzer) checkResource(module *moduleScope, resource *ast.Resource) hir.Resource {
	value := a.resources[resource]
	for _, attribute := range resource.Attributes {
		a.addUnsupported(module.file.Path, attribute.Span, "resource attribute @"+attribute.Name)
	}
	result := hir.Resource{
		Name:     resource.Name,
		Type:     value.typ,
		Span:     span(module.file.Path, resource.Span),
		DropKind: hir.ResourceDropMissing,
	}
	if info := a.resourceTypes[value.typ]; info != nil {
		result.Fields = make([]hir.ResourceField, 0, len(info.orderedFields))
		for _, field := range info.orderedFields {
			result.Fields = append(result.Fields, hir.ResourceField{Name: field.name, Type: field.typ, Span: field.span})
		}
	}
	if resource.Drop == nil || resource.Drop.Body == nil {
		a.addUnsupported(module.file.Path, resource.Span, "resource without a drop body")
		return result
	}
	result.DropSpan = span(module.file.Path, resource.Drop.Span)
	if len(resource.Drop.Body.Statements) == 0 && resource.Drop.Body.Tail == nil {
		result.DropKind = hir.ResourceDropEmpty
		return result
	}
	result.DropKind = hir.ResourceDropNonEmpty
	a.addUnsupported(module.file.Path, resource.Drop.Span, "non-empty resource drop body")
	return result
}

func (c *typeChecker) checkFunction() hir.Function {
	functionSymbol := c.analyzer.functions[c.function]
	for _, attribute := range c.function.Attributes {
		c.analyzer.addUnsupported(c.module.file.Path, attribute.Span, "function attribute @"+attribute.Name)
	}
	resultType := c.canonicalType(c.function.ResultType)
	functionSymbol.typ = resultType
	if errorSpan, ok := resultFailureSpan(c.function.ResultType); ok {
		functionSymbol.resultError = errorSpan
	}
	result := hir.Function{
		Symbol:     functionSymbol.id,
		Name:       c.function.Name,
		Unsafe:     c.function.Unsafe,
		Span:       span(c.module.file.Path, c.function.Span),
		ResultType: resultType,
	}
	c.hir = &result
	for index := range c.function.Parameters {
		parameter := &c.function.Parameters[index]
		symbol := c.analyzer.parameter[parameter]
		if symbol == nil {
			continue
		}
		symbol.typ = c.canonicalType(parameter.Type)
		if errorSpan, ok := resultFailureSpan(parameter.Type); ok {
			symbol.resultError = errorSpan
		}
		parameterSpan := span(c.module.file.Path, parameter.NameSpan)
		result.Parameters = append(result.Parameters, hir.Parameter{
			Symbol: symbol.id,
			Name:   parameter.Name,
			Type:   symbol.typ,
			Span:   parameterSpan,
		})
		c.addType(parameterSpan, symbol.typ)
	}
	if c.function.Body == nil {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Span, "function without a body")
		return result
	}
	if c.function.Unsafe {
		c.unsafeScopes = append(c.unsafeScopes, span(c.module.file.Path, c.function.Span))
	}
	body, bodyType, hasReturn := c.checkBlock(c.function.Body, resultType)
	if c.function.Unsafe {
		c.unsafeScopes = c.unsafeScopes[:len(c.unsafeScopes)-1]
	}
	result.Body = body
	if c.function.Body.Tail != nil {
		c.requireType(
			"E1009",
			span(c.module.file.Path, c.function.Body.Tail.Span),
			bodyType,
			resultType,
			span(c.module.file.Path, c.function.ResultType.Span),
		)
	} else if resultType != hir.TypeVoid && !hasReturn {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Body.Span, "non-void function without a checked return value")
	}
	return result
}

func (c *typeChecker) checkBlock(block *ast.Block, expectedTail hir.Type) (hir.Block, hir.Type, bool) {
	result := hir.Block{Span: span(c.module.file.Path, block.Span)}
	hasReturn := false
	for index := range block.Statements {
		statement := &block.Statements[index]
		hirStatement := hir.Statement{Kind: string(statement.Kind), Span: span(c.module.file.Path, statement.Span), Binding: hir.NoSymbol}
		switch {
		case statement.Binding != nil:
			binding := statement.Binding
			symbol := c.analyzer.bindingSymbols[binding]
			if symbol == nil {
				continue
			}
			expected := hir.TypeInvalid
			if binding.Type != nil {
				expected = c.canonicalType(*binding.Type)
			}
			expression, actual := c.checkExpression(binding.Expression, expected)
			if expected != hir.TypeInvalid {
				c.requireType(
					"E1006",
					span(c.module.file.Path, binding.Expression.Span),
					actual,
					expected,
					span(c.module.file.Path, binding.Type.Span),
				)
				symbol.typ = expected
			} else {
				symbol.typ = actual
			}
			hirStatement.Binding = symbol.id
			hirStatement.Expression = expression
			localSpan := span(c.module.file.Path, binding.NameSpan)
			c.hir.Locals = append(c.hir.Locals, hir.Local{
				Symbol:  symbol.id,
				Name:    binding.Name,
				Type:    symbol.typ,
				Mutable: binding.Mutable,
				Span:    localSpan,
			})
			c.addType(localSpan, symbol.typ)
		case statement.Kind == syntax.KindReturn:
			hasReturn = true
			expression, actual := c.checkExpression(statement.Expression, expectedTail)
			hirStatement.Expression = expression
			if statement.Expression == nil {
				actual = hir.TypeVoid
			}
			primary := statement.Span
			if statement.Expression != nil {
				primary = statement.Expression.Span
			}
			c.requireType(
				"E1009",
				span(c.module.file.Path, primary),
				actual,
				expectedTail,
				span(c.module.file.Path, c.function.ResultType.Span),
			)
		case statement.Kind == syntax.KindAssignment:
			target, targetType := c.checkExpression(statement.Target, hir.TypeInvalid)
			value, actual := c.checkExpression(statement.Expression, targetType)
			hirStatement.Target = target
			hirStatement.Expression = value
			if statement.Target != nil {
				resolved := c.analyzer.resolved[statement.Target]
				if resolved == nil || !resolved.mutable {
					c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, statement.Target.Span))
				}
			}
			if statement.Expression != nil {
				c.requireType(
					"E1006",
					span(c.module.file.Path, statement.Expression.Span),
					actual,
					targetType,
					span(c.module.file.Path, statement.Target.Span),
				)
			}
		case statement.Kind == syntax.KindExprStmt || statement.Kind == syntax.KindDiscard:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeInvalid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindWhile:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeVoid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindBreak || statement.Kind == syntax.KindContinue:
			if c.loopDepth == 0 {
				c.analyzer.addDiagnostic("E1012", span(c.module.file.Path, statement.Span))
			}
		case statement.Kind == syntax.KindRegion:
			expression, nestedReturn := c.checkRegion(statement.Expression, expectedTail)
			hirStatement.Expression = expression
			hasReturn = hasReturn || nestedReturn
		default:
			c.analyzer.addUnsupported(c.module.file.Path, statement.Span, "statement "+string(statement.Kind))
		}
		result.Statements = append(result.Statements, hirStatement)
	}
	resultType := hir.TypeVoid
	if block.Tail != nil {
		result.Tail, resultType = c.checkExpression(block.Tail, expectedTail)
	}
	return result, resultType, hasReturn
}

func (c *typeChecker) checkExpression(expression *ast.Expression, expected hir.Type) (*hir.Expression, hir.Type) {
	if expression == nil {
		return nil, hir.TypeVoid
	}
	result := &hir.Expression{
		Kind:   string(expression.Kind),
		Span:   span(c.module.file.Path, expression.Span),
		Value:  expression.Text,
		Cooked: expression.Cooked,
		Symbol: hir.NoSymbol,
	}
	var actual hir.Type
	switch expression.Kind {
	case syntax.KindLiteral:
		var supported bool
		actual, supported = literalType(expression.Text, expected)
		if !supported {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "literal "+literalClass(expression.Text))
		}
	case syntax.KindPath:
		if reference, found := c.analyzer.enumVariantPaths[expression]; found {
			actual = c.checkEnumPath(expression, result, reference)
		} else if symbol := c.analyzer.resolved[expression]; symbol != nil {
			result.Symbol = symbol.id
			actual = symbol.typ
		} else if variant, option := optionVariantPath(expression.Text); option {
			actual = c.checkOptionPath(expression, result, expected, variant)
		} else if isResultVariantPath(expression.Text) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Result variant used without construction or pattern")
			actual = hir.TypeInvalid
		} else {
			actual = hir.TypeInvalid
		}
	case syntax.KindGroup:
		if len(expression.Children) == 1 {
			child, childType := c.checkExpression(expression.Children[0], expected)
			result.Children = appendExpression(result.Children, child)
			actual = childType
		} else {
			actual = hir.TypeInvalid
		}
	case syntax.KindPrefix:
		actual = c.checkPrefix(expression, result)
	case syntax.KindBinary:
		actual = c.checkBinary(expression, result)
	case syntax.KindCast:
		actual = c.checkCast(expression, result)
	case syntax.KindCall:
		actual = c.checkCall(expression, result, expected)
	case syntax.KindIndex:
		actual = c.checkIndex(expression, result)
	case syntax.KindIf:
		actual = c.checkIf(expression, result, expected)
	case syntax.KindMatch:
		actual = c.checkMatch(expression, result, expected)
	case syntax.KindWhile:
		actual = c.checkWhile(expression, result)
	case syntax.KindRecord:
		actual = c.checkRecord(expression, result)
	case syntax.KindFieldExpr:
		actual = c.checkField(expression, result)
	case syntax.KindBlock:
		if expression.Block == nil {
			actual = hir.TypeInvalid
			break
		}
		block, blockType, _ := c.checkBlock(expression.Block, expected)
		result.Block = &block
		actual = blockType
	case syntax.KindRegion:
		var nestedReturn bool
		result, nestedReturn = c.checkRegion(expression, expected)
		_ = nestedReturn
		actual = hir.TypeVoid
	case syntax.KindPropagate:
		actual = c.checkPropagate(expression, result)
	case syntax.KindUnsafe:
		actual = c.checkUnsafe(expression, result, expected)
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "expression "+string(expression.Kind))
		actual = hir.TypeInvalid
	}
	result.Type = actual
	c.analyzer.expressionTypes[expression] = actual
	if actual != hir.TypeInvalid {
		c.addType(result.Span, actual)
	}
	return result, actual
}

func (c *typeChecker) checkIndex(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed index expression")
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	index, indexType := c.checkExpression(expression.Children[1], hir.TypeUSize)
	result.Children = appendExpression(result.Children, receiver, index)
	if receiverType != hir.SharedBorrow(hir.TypeByteSlice) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "indexing outside immutable byte views")
		return hir.TypeInvalid
	}
	c.requireType(
		"E1006",
		span(c.module.file.Path, expression.Children[1].Span),
		indexType,
		hir.TypeUSize,
		span(c.module.file.Path, expression.Span),
	)
	if indexType == hir.TypeInvalid || indexType != hir.TypeUSize {
		return hir.TypeInvalid
	}
	result.Kind = "view-index"
	result.Operator = "[]"
	return hir.TypeU8
}

func (c *typeChecker) checkCast(expression *ast.Expression, result *hir.Expression) hir.Type {
	result.Operator = "as"
	if len(expression.Children) != 1 || expression.Type == nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed cast")
		return hir.TypeInvalid
	}
	targetType := c.analyzer.canonicalType(c.module, *expression.Type)
	result.TargetType = targetType
	child, sourceType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, child)
	sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(sourceType)
	targetElement, targetMutable, targetPointer := hir.RawPointerElement(targetType)
	sameElement := sourceElement == targetElement && (sourceMutable || !targetMutable)
	byteSliceBytes := sourceElement == hir.TypeByteSlice && !sourceMutable && targetElement == hir.TypeU8 && !targetMutable
	stringBytes := sourceElement == hir.TypeStr && !sourceMutable && targetElement == hir.TypeU8 && !targetMutable
	cStringBytes := sourceElement == hir.TypeCStr && !sourceMutable && targetElement == hir.TypeU8 && !targetMutable
	if !sourceBorrow || !targetPointer || !sameElement && !byteSliceBytes && !stringBytes && !cStringBytes {
		c.analyzer.addUnsupported(
			c.module.file.Path,
			expression.Span,
			fmt.Sprintf("cast from %s to %s", sourceType, targetType),
		)
		return hir.TypeInvalid
	}
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span))
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	return targetType
}

func (c *typeChecker) checkUnsafe(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) != 1 || expression.Children[0] == nil || expression.Children[0].Kind != syntax.KindBlock {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed unsafe block")
		return hir.TypeInvalid
	}
	scope := span(c.module.file.Path, expression.Span)
	c.unsafeScopes = append(c.unsafeScopes, scope)
	child, actual := c.checkExpression(expression.Children[0], expected)
	c.unsafeScopes = c.unsafeScopes[:len(c.unsafeScopes)-1]
	result.Children = appendExpression(result.Children, child)
	return actual
}

func (c *typeChecker) checkWhile(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 2 {
		return hir.TypeInvalid
	}
	condition, conditionType := c.checkExpression(expression.Children[0], hir.TypeBool)
	result.Children = appendExpression(result.Children, condition)
	if conditionType != hir.TypeInvalid && conditionType != hir.TypeBool {
		c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
	}
	c.loopDepth++
	body, _ := c.checkExpression(expression.Children[1], hir.TypeVoid)
	c.loopDepth--
	result.Children = appendExpression(result.Children, body)
	return hir.TypeVoid
}

func (c *typeChecker) checkRecord(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	if reference, found := c.analyzer.enumVariantPaths[expression.Children[0]]; found {
		return c.checkEnumRecord(expression, result, reference)
	}
	constructor := c.analyzer.resolved[expression.Children[0]]
	if constructor != nil && constructor.typ == hir.TypeRegion {
		c.analyzer.addDiagnostic("E3104", span(c.module.file.Path, expression.Span))
		result.Symbol = constructor.id
		return hir.TypeInvalid
	}
	if constructor == nil || constructor.resource == nil && constructor.structure == nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "record constructor that is not an aggregate")
		return hir.TypeInvalid
	}
	fields := map[string]resourceField(nil)
	if info := c.analyzer.resourceTypes[constructor.typ]; info != nil {
		fields = info.fields
	} else if info := c.analyzer.structTypes[constructor.typ]; info != nil {
		fields = info.fields
	}
	if fields == nil {
		return hir.TypeInvalid
	}
	result.Symbol = constructor.id
	seen := make(map[string]struct{}, len(expression.Children)-1)
	for _, fieldExpression := range expression.Children[1:] {
		field, found := fields[fieldExpression.Text]
		if !found || len(fieldExpression.Children) != 1 {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "unknown or malformed resource field "+fieldExpression.Text)
			continue
		}
		if _, duplicate := seen[fieldExpression.Text]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "duplicate resource field "+fieldExpression.Text)
			continue
		}
		seen[fieldExpression.Text] = struct{}{}
		value, actual := c.checkExpression(fieldExpression.Children[0], field.typ)
		result.Children = appendExpression(result.Children, value)
		result.Fields = append(result.Fields, fieldExpression.Text)
		c.requireType("E1006", span(c.module.file.Path, fieldExpression.Children[0].Span), actual, field.typ, field.declaration)
	}
	if len(seen) != len(fields) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "resource construction with missing fields")
	}
	return constructor.typ
}

func (c *typeChecker) checkEnumPath(expression *ast.Expression, result *hir.Expression, reference enumVariantReference) hir.Type {
	variant, found := c.enumVariant(reference)
	if !found {
		return hir.TypeInvalid
	}
	if len(variant.orderedFields) != 0 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "enum payload variant used without record construction")
		return hir.TypeInvalid
	}
	result.Kind = "enum-variant"
	result.Symbol = reference.enumeration.id
	result.Variant = variant.name
	return reference.enumeration.typ
}

func (c *typeChecker) checkEnumRecord(expression *ast.Expression, result *hir.Expression, reference enumVariantReference) hir.Type {
	variant, found := c.enumVariant(reference)
	if !found {
		return hir.TypeInvalid
	}
	result.Kind = "enum-variant"
	result.Symbol = reference.enumeration.id
	result.Variant = variant.name
	seen := make(map[string]struct{}, len(expression.Children)-1)
	for _, fieldExpression := range expression.Children[1:] {
		field, declared := variant.fields[fieldExpression.Text]
		if !declared || len(fieldExpression.Children) != 1 {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "unknown or malformed enum payload field "+fieldExpression.Text)
			continue
		}
		if _, duplicate := seen[fieldExpression.Text]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "duplicate enum payload field "+fieldExpression.Text)
			continue
		}
		seen[fieldExpression.Text] = struct{}{}
		value, actual := c.checkExpression(fieldExpression.Children[0], field.typ)
		result.Children = appendExpression(result.Children, value)
		result.Fields = append(result.Fields, fieldExpression.Text)
		c.requireType("E1006", span(c.module.file.Path, fieldExpression.Children[0].Span), actual, field.typ, field.declaration)
	}
	if len(seen) != len(variant.fields) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "enum payload construction with missing fields")
	}
	return reference.enumeration.typ
}

func (c *typeChecker) enumVariant(reference enumVariantReference) (enumVariant, bool) {
	if reference.enumeration == nil || reference.variant == nil {
		return enumVariant{}, false
	}
	info := c.analyzer.enumTypes[reference.enumeration.typ]
	if info == nil {
		return enumVariant{}, false
	}
	variant, found := info.variants[reference.variant.Name]
	return variant, found
}

func (c *typeChecker) checkField(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, receiver)
	if element, _, borrow := hir.BorrowElement(receiverType); borrow && (element == hir.TypeStr || element == hir.TypeByteSlice) {
		if expression.Text != "len" {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown borrowed-view field "+expression.Text)
			return hir.TypeInvalid
		}
		result.Kind = "view-length"
		result.Operator = "len"
		return hir.TypeUSize
	}
	if element, _, borrow := hir.BorrowElement(receiverType); borrow {
		receiverType = element
	}
	fields := map[string]resourceField(nil)
	if info, found := c.analyzer.resourceType(receiverType); found {
		fields = info.fields
	} else if info, found := c.analyzer.structType(receiverType); found {
		fields = info.fields
	}
	if fields == nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "field access on a non-aggregate value")
		return hir.TypeInvalid
	}
	field, found := fields[expression.Text]
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown resource field "+expression.Text)
		return hir.TypeInvalid
	}
	result.Operator = expression.Text
	return field.typ
}

func (c *typeChecker) checkPrefix(expression *ast.Expression, result *hir.Expression) hir.Type {
	result.Operator = expression.Text
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	child, childType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, child)
	switch expression.Text {
	case "!":
		if childType != hir.TypeInvalid && childType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
		}
		return hir.TypeBool
	case "+", "-", "~":
		c.requireType(
			"E1006",
			span(c.module.file.Path, expression.Children[0].Span),
			childType,
			hir.TypeI32,
			span(c.module.file.Path, expression.Span),
		)
		return hir.TypeI32
	case "&":
		if c.analyzer.resolved[expression.Children[0]] == nil {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "borrow of a non-place expression")
			return hir.TypeInvalid
		}
		return hir.SharedBorrow(childType)
	case "&mut":
		target := c.analyzer.resolved[expression.Children[0]]
		if target == nil || !target.mutable {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, expression.Children[0].Span))
			return hir.TypeInvalid
		}
		return hir.MutableBorrow(childType)
	case "*":
		element, _, ok := hir.BorrowElement(childType)
		if !ok {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "dereference of a non-borrow value")
			return hir.TypeInvalid
		}
		return element
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "prefix operator "+expression.Text)
		return hir.TypeInvalid
	}
}

func (c *typeChecker) checkBinary(expression *ast.Expression, result *hir.Expression) hir.Type {
	result.Operator = expression.Text
	if len(expression.Children) != 2 {
		return hir.TypeInvalid
	}
	left, leftType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	right, rightType := c.checkExpression(expression.Children[1], leftType)
	result.Children = appendExpression(result.Children, left, right)
	switch expression.Text {
	case "&&", "||":
		if leftType != hir.TypeInvalid && leftType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
		}
		if rightType != hir.TypeInvalid && rightType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[1].Span))
		}
		return hir.TypeBool
	case "==", "!=":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
		}
		return hir.TypeBool
	case "<", "<=", ">", ">=":
		c.requireType("E1006", span(c.module.file.Path, expression.Children[0].Span), leftType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), rightType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		return hir.TypeBool
	case "<<":
		c.requireType("E1006", span(c.module.file.Path, expression.Children[0].Span), leftType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), rightType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.analyzer.addDiagnostic("E1011", span(c.module.file.Path, expression.Span))
		return hir.TypeI32
	case "+", "-", "*", "/", "%", ">>", "&", "^", "|":
		c.requireType("E1006", span(c.module.file.Path, expression.Children[0].Span), leftType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), rightType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		return hir.TypeI32
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "binary operator "+expression.Text)
		return hir.TypeInvalid
	}
}

func (c *typeChecker) checkCall(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	calleeExpression := expression.Children[0]
	if calleeExpression != nil && calleeExpression.Kind == syntax.KindPath {
		if variant, ok := resultVariantPath(calleeExpression.Text); ok {
			return c.checkResultConstructor(expression, result, expected, variant)
		}
		if variant, ok := optionVariantPath(calleeExpression.Text); ok {
			return c.checkOptionConstructor(expression, result, expected, variant)
		}
	}
	if isRegionPlaceCall(expression) {
		return c.checkRegionPlace(expression, result)
	}
	callee := c.analyzer.resolved[calleeExpression]
	if callee == nil || callee.kind != "function" && callee.kind != "foreign-function" {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "call target that is not a function")
		return hir.TypeInvalid
	}
	if callee.foreign != nil {
		return c.checkForeignCall(expression, result, callee)
	}
	if callee.function == nil {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "call target without a function declaration")
		return hir.TypeInvalid
	}
	result.Symbol = callee.id
	result.Children = append(result.Children, hir.Expression{
		Kind:   string(calleeExpression.Kind),
		Type:   hir.TypeInvalid,
		Span:   span(c.module.file.Path, calleeExpression.Span),
		Symbol: callee.id,
	})
	arguments := expression.Children[1:]
	if len(arguments) != len(callee.function.Parameters) {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration",
			Span: span(callee.source, callee.declaration),
		})
	}
	for index, argument := range arguments {
		expected := hir.TypeInvalid
		var expectedSpan = callee.typeSpan
		if index < len(callee.function.Parameters) {
			expected = c.analyzer.canonicalType(c.analyzer.modules[callee.module], callee.function.Parameters[index].Type)
			expectedSpan = callee.function.Parameters[index].Type.Span
		}
		child, actual := c.checkExpression(argument, expected)
		result.Children = appendExpression(result.Children, child)
		if expected != hir.TypeInvalid {
			c.requireType(
				"E1006",
				span(c.module.file.Path, argument.Span),
				actual,
				expected,
				span(callee.source, expectedSpan),
			)
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.modules[callee.module], callee.function.ResultType)
	if errorSpan, ok := resultFailureSpan(callee.function.ResultType); ok {
		callee.resultError = errorSpan
	}
	return callee.typ
}

func (c *typeChecker) checkForeignCall(expression *ast.Expression, result *hir.Expression, callee *symbol) hir.Type {
	declaration := callee.foreign
	result.Symbol = callee.id
	result.ForeignABI = declaration.ABI
	result.Children = append(result.Children, hir.Expression{
		Kind: string(syntax.KindPath), Type: hir.TypeInvalid,
		Span: span(c.module.file.Path, expression.Children[0].Span), Symbol: callee.id,
	})
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	arguments := expression.Children[1:]
	if len(arguments) != len(declaration.Parameters) {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
	}
	for index, argument := range arguments {
		expected := hir.TypeInvalid
		expectedSpan := declaration.ResultType.Span
		if index < len(declaration.Parameters) {
			expected = c.analyzer.canonicalType(c.analyzer.modules[callee.module], declaration.Parameters[index].Type)
			expectedSpan = declaration.Parameters[index].Type.Span
		}
		child, actual := c.checkExpression(argument, expected)
		result.Children = appendExpression(result.Children, child)
		if expected != hir.TypeInvalid {
			c.requireType("E1006", span(c.module.file.Path, argument.Span), actual, expected, span(callee.source, expectedSpan))
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.modules[callee.module], declaration.ResultType)
	return callee.typ
}

func (c *typeChecker) checkOptionPath(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantNone {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::Some used without one payload")
		return hir.TypeInvalid
	}
	_, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::None without contextual Option type")
		return hir.TypeInvalid
	}
	result.Kind = "option-variant"
	result.Variant = optionVariantNone
	return expected
}

func (c *typeChecker) checkOptionConstructor(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantSome {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::None is a bare variant, not a call")
		return hir.TypeInvalid
	}
	payloadType, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option constructor without contextual Option type")
		return hir.TypeInvalid
	}
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::Some arity other than one")
		return hir.TypeInvalid
	}
	payload, actual := c.checkExpression(expression.Children[1], payloadType)
	c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), actual, payloadType, span(c.module.file.Path, expression.Span))
	result.Kind = "option-variant"
	result.Variant = optionVariantSome
	result.Children = appendExpression(result.Children, payload)
	return expected
}

func (c *typeChecker) checkResultConstructor(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	success, failure, ok := hir.ResultElements(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Result constructor without contextual Result type")
		return hir.TypeInvalid
	}
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Result constructor arity other than one")
		return hir.TypeInvalid
	}
	payloadType := success
	if variant == resultVariantErr {
		payloadType = failure
	}
	payload, actual := c.checkExpression(expression.Children[1], payloadType)
	c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), actual, payloadType, span(c.module.file.Path, expression.Span))
	result.Kind = "result-variant"
	result.Variant = variant
	result.Children = appendExpression(result.Children, payload)
	return expected
}

func (c *typeChecker) checkMatch(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) < 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "match without arms")
		return hir.TypeInvalid
	}
	scrutinee, scrutineeType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, scrutinee)
	if _, _, ok := hir.ResultElements(scrutineeType); ok {
		if len(expression.Children) != 3 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "built-in Result match without exactly two arms")
			return hir.TypeInvalid
		}
		return c.checkResultMatch(expression, result, expected, scrutineeType)
	}
	if payloadType, ok := hir.OptionElement(scrutineeType); ok {
		if len(expression.Children) != 3 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "built-in Option match without exactly two arms")
			return hir.TypeInvalid
		}
		return c.checkOptionMatch(expression, result, expected, payloadType)
	}
	if info := c.analyzer.enumTypes[scrutineeType]; info != nil {
		return c.checkEnumMatch(expression, result, expected, info)
	}
	c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "match scrutinee outside the implemented Result, Option, and restricted enum subset")
	return hir.TypeInvalid
}

func (c *typeChecker) checkEnumMatch(expression *ast.Expression, result *hir.Expression, expected hir.Type, info *enumInfo) hir.Type {
	seen := make(map[string]struct{}, len(info.variants))
	actual := expected
	for _, armExpression := range expression.Children[1:] {
		if armExpression == nil || armExpression.Kind != syntax.KindPattern || armExpression.Text != "arm" || len(armExpression.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed enum match arm")
			return hir.TypeInvalid
		}
		pattern := armExpression.Children[0]
		reference, found := c.analyzer.enumVariantPaths[pattern]
		if !found || reference.enumeration != info.symbol || reference.variant == nil {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern from a different or unresolved enum")
			return hir.TypeInvalid
		}
		variant := info.variants[reference.variant.Name]
		if _, duplicate := seen[variant.name]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "duplicate enum match variant")
			return hir.TypeInvalid
		}
		seen[variant.name] = struct{}{}
		arm := hir.MatchArm{Variant: variant.name, Binding: hir.NoSymbol}
		boundFields := make(map[string]struct{}, len(pattern.Children)-1)
		for _, fieldExpression := range pattern.Children[1:] {
			field, declared := variant.fields[fieldExpression.Text]
			binding := c.analyzer.resolved[fieldExpression]
			if !declared || binding == nil || fieldExpression.Cooked == "" {
				c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "unknown or malformed enum pattern field "+fieldExpression.Text)
				return hir.TypeInvalid
			}
			if _, duplicate := boundFields[fieldExpression.Text]; duplicate {
				c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "duplicate enum pattern field "+fieldExpression.Text)
				return hir.TypeInvalid
			}
			boundFields[fieldExpression.Text] = struct{}{}
			binding.typ = field.typ
			bindingSpan := span(c.module.file.Path, fieldExpression.Span)
			c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding.id, Name: binding.name, Type: field.typ, Span: bindingSpan})
			c.addType(bindingSpan, field.typ)
			arm.Bindings = append(arm.Bindings, hir.MatchBinding{Field: field.name, Symbol: binding.id, Type: field.typ, Span: bindingSpan})
		}
		if len(boundFields) != len(variant.fields) {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "enum pattern with missing payload fields")
			return hir.TypeInvalid
		}
		body, bodyType := c.checkExpression(armExpression.Children[1], actual)
		if actual == hir.TypeInvalid {
			actual = bodyType
		} else {
			c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
		}
		arm.Expression = *body
		result.Arms = append(result.Arms, arm)
	}
	if len(seen) != len(info.variants) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive enum match")
		return hir.TypeInvalid
	}
	return actual
}

func (c *typeChecker) checkResultMatch(expression *ast.Expression, result *hir.Expression, expected, scrutineeType hir.Type) hir.Type {
	success, failure, ok := hir.ResultElements(scrutineeType)
	if !ok {
		return hir.TypeInvalid
	}
	seen := make(map[string]struct{}, 2)
	actual := expected
	for _, armExpression := range expression.Children[1:] {
		if armExpression == nil || armExpression.Kind != syntax.KindPattern || armExpression.Text != "arm" || len(armExpression.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed Result match arm")
			return hir.TypeInvalid
		}
		pattern := armExpression.Children[0]
		if pattern == nil || pattern.Kind != syntax.KindPattern || len(pattern.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, armExpression.Span, "Result arm without one payload binding")
			return hir.TypeInvalid
		}
		variant, variantOK := resultVariantPath(pattern.Text)
		if !variantOK {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a qualified Result variant")
			return hir.TypeInvalid
		}
		if _, duplicate := seen[variant]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "duplicate Result match variant")
			return hir.TypeInvalid
		}
		seen[variant] = struct{}{}
		bindingExpression := pattern.Children[1]
		binding := c.analyzer.resolved[bindingExpression]
		if binding == nil {
			return hir.TypeInvalid
		}
		bindingType := success
		if variant == resultVariantErr {
			bindingType = failure
		}
		binding.typ = bindingType
		bindingSpan := span(c.module.file.Path, bindingExpression.Span)
		c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding.id, Name: binding.name, Type: bindingType, Span: bindingSpan})
		c.addType(bindingSpan, bindingType)
		armExpected := actual
		body, bodyType := c.checkExpression(armExpression.Children[1], armExpected)
		if actual == hir.TypeInvalid {
			actual = bodyType
		} else {
			c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
		}
		result.Arms = append(result.Arms, hir.MatchArm{Variant: variant, Binding: binding.id, BindingSpan: bindingSpan, Expression: *body})
	}
	if len(seen) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive Result match")
		return hir.TypeInvalid
	}
	return actual
}

func (c *typeChecker) checkOptionMatch(expression *ast.Expression, result *hir.Expression, expected, payloadType hir.Type) hir.Type {
	seen := make(map[string]struct{}, 2)
	actual := expected
	for _, armExpression := range expression.Children[1:] {
		if armExpression == nil || armExpression.Kind != syntax.KindPattern || armExpression.Text != "arm" || len(armExpression.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed Option match arm")
			return hir.TypeInvalid
		}
		pattern := armExpression.Children[0]
		if pattern == nil || pattern.Kind != syntax.KindPattern {
			c.analyzer.addUnsupported(c.module.file.Path, armExpression.Span, "malformed Option pattern")
			return hir.TypeInvalid
		}
		variant, variantOK := optionVariantPath(pattern.Text)
		if !variantOK {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a qualified Option variant")
			return hir.TypeInvalid
		}
		if _, duplicate := seen[variant]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "duplicate Option match variant")
			return hir.TypeInvalid
		}
		seen[variant] = struct{}{}
		arm := hir.MatchArm{Variant: variant, Binding: hir.NoSymbol}
		switch variant {
		case optionVariantSome:
			if len(pattern.Children) != 2 {
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "Option::Some pattern without one payload binding")
				return hir.TypeInvalid
			}
			bindingExpression := pattern.Children[1]
			binding := c.analyzer.resolved[bindingExpression]
			if binding == nil {
				return hir.TypeInvalid
			}
			binding.typ = payloadType
			bindingSpan := span(c.module.file.Path, bindingExpression.Span)
			c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding.id, Name: binding.name, Type: payloadType, Span: bindingSpan})
			c.addType(bindingSpan, payloadType)
			arm.Binding = binding.id
			arm.BindingSpan = bindingSpan
		case optionVariantNone:
			if len(pattern.Children) != 1 {
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "Option::None pattern with a payload binding")
				return hir.TypeInvalid
			}
		}
		body, bodyType := c.checkExpression(armExpression.Children[1], actual)
		if actual == hir.TypeInvalid {
			actual = bodyType
		} else {
			c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
		}
		arm.Expression = *body
		result.Arms = append(result.Arms, arm)
	}
	if len(seen) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive Option match")
		return hir.TypeInvalid
	}
	return actual
}

func (c *typeChecker) checkIf(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) < 2 {
		return hir.TypeInvalid
	}
	condition, conditionType := c.checkExpression(expression.Children[0], hir.TypeBool)
	result.Children = appendExpression(result.Children, condition)
	if conditionType != hir.TypeInvalid && conditionType != hir.TypeBool {
		c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
	}
	thenExpression, thenType := c.checkExpression(expression.Children[1], expected)
	result.Children = appendExpression(result.Children, thenExpression)
	if len(expression.Children) == 2 {
		return hir.TypeVoid
	}
	elseExpression, elseType := c.checkExpression(expression.Children[2], expected)
	result.Children = appendExpression(result.Children, elseExpression)
	if thenType == hir.TypeInvalid || elseType == hir.TypeInvalid {
		return hir.TypeInvalid
	}
	if thenType != elseType {
		c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[2].Span), RelatedSpan{
			Role: "expected-type",
			Span: span(c.module.file.Path, expression.Children[1].Span),
		})
		return hir.TypeInvalid
	}
	return thenType
}

func (c *typeChecker) canonicalType(reference ast.TypeReference) hir.Type {
	return c.analyzer.canonicalType(c.module, reference)
}

func (c *typeChecker) requireType(code string, primary hir.Span, actual, expected hir.Type, expectedSpan hir.Span) {
	if actual == hir.TypeInvalid || expected == hir.TypeInvalid || actual == expected {
		return
	}
	c.analyzer.addDiagnostic(code, primary, RelatedSpan{
		Role: "expected-type",
		Span: expectedSpan,
	})
}

func (c *typeChecker) addType(expression hir.Span, value hir.Type) {
	if value == hir.TypeInvalid {
		return
	}
	c.analyzer.types = append(c.analyzer.types, TypeFact{Expression: expression, CanonicalType: value})
}

func literalType(raw string, expected hir.Type) (hir.Type, bool) {
	if raw == "true" || raw == "false" {
		return hir.TypeBool, true
	}
	if raw == "null" {
		if _, _, pointer := hir.RawPointerElement(expected); pointer {
			return expected, true
		}
		return hir.TypeInvalid, false
	}
	if strings.HasPrefix(raw, "c\"") {
		return hir.SharedBorrow(hir.TypeCStr), true
	}
	if strings.HasPrefix(raw, "b\"") {
		return hir.SharedBorrow(hir.TypeByteSlice), true
	}
	if strings.HasPrefix(raw, "\"") {
		return hir.SharedBorrow(hir.TypeStr), true
	}
	if raw != "" && raw[0] >= '0' && raw[0] <= '9' && !strings.ContainsAny(raw, ".eE") {
		switch expected {
		case hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
			return expected, true
		}
		return hir.TypeI32, true
	}
	return hir.TypeInvalid, false
}

func literalClass(raw string) string {
	switch {
	case strings.HasPrefix(raw, "\""):
		return "string"
	case strings.HasPrefix(raw, "b\""):
		return "byte-string"
	case strings.HasPrefix(raw, "c\""):
		return "c-string"
	case raw == "null":
		return "null"
	case strings.ContainsAny(raw, ".eE"):
		return "float"
	default:
		return "unknown"
	}
}

func appendExpression(values []hir.Expression, expressions ...*hir.Expression) []hir.Expression {
	for _, expression := range expressions {
		if expression != nil {
			values = append(values, *expression)
		}
	}
	return values
}

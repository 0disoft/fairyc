package semantic

import (
	"fmt"
	"slices"
	"strconv"
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
	resultType   hir.Type
	loopDepth    int
	unsafeScopes []hir.Span
}

func (a *analyzer) typeCheck() Result {
	result := Result{}
	a.prepareAggregateIdentities()
	a.prepareResourceTypes()
	a.prepareStructTypes()
	a.prepareEnumTypes()
	a.prepareCallableMustUse()
	a.prepareCallableEffects()
	a.prepareCompileTime()
	result.HIR.StateLayouts = a.checkStateLayouts()
	for _, module := range a.moduleOrder {
		_, sourcePath := hir.SourcePathParts(module.file.Path)
		hirModule := hir.Module{Package: module.key.packageID, Path: module.file.Module.Path, SourcePath: sourcePath}
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
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			function := a.dropFunctions[resource]
			if function == nil || a.functions[function] == nil {
				continue
			}
			checker := typeChecker{analyzer: a, module: module, function: function}
			dropFunction := checker.checkFunction()
			dropFunction.DropGlueFor = a.resources[resource].typ
			hirModule.Functions = append(hirModule.Functions, dropFunction)
		}
		result.HIR.Modules = append(result.HIR.Modules, hirModule)
	}
	return result
}

func (a *analyzer) checkEnum(module *moduleScope, enumeration *ast.Enum) hir.Enum {
	value := a.enums[enumeration]
	for _, attribute := range enumeration.Attributes {
		if attribute.Name == constEvalAttribute {
			continue
		}
		if attribute.Name == mustUseAttribute {
			a.addDiagnostic("E4201", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, enumeration.NameSpan)})
			continue
		}
		a.addUnsupported(module.file.Path, attribute.Span, "enum attribute @"+attribute.Name)
	}
	result := hir.Enum{
		Name: enumeration.Name, Public: enumeration.Public, Type: value.typ,
		Span:              span(module.file.Path, enumeration.Span),
		DocumentationSpan: optionalSpan(module.file.Path, enumeration.DocumentationSpan),
	}
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
	result := hir.Struct{
		Name: structure.Name, Public: structure.Public, Type: value.typ,
		Span:              span(module.file.Path, structure.Span),
		DocumentationSpan: optionalSpan(module.file.Path, structure.DocumentationSpan),
	}
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
		if attribute.Name == constEvalAttribute {
			continue
		}
		if attribute.Name == effectsAttribute || attribute.Name == mustUseAttribute {
			continue
		}
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
		Symbol: symbol.id, Name: function.Name, Public: function.Public, LinkName: linkName, ABI: function.ABI,
		Span:              span(module.file.Path, function.Span),
		DocumentationSpan: optionalSpan(module.file.Path, function.DocumentationSpan),
		Effects:           slices.Clone(symbol.effects),
		ResultType:        a.canonicalType(module, function.ResultType),
	}
	symbol.typ = result.ResultType
	result.MustUse = a.finalizeMustUse(module.file.Path, function.ResultType.Span, symbol, result.ResultType)
	if containsSafeBorrow(result.ResultType) {
		a.addDiagnostic("E5102", span(module.file.Path, function.ResultType.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, function.NameSpan)})
	}
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		parameterType := a.canonicalType(module, parameter.Type)
		if containsSafeBorrow(parameterType) {
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
		if attribute.Name == constEvalAttribute {
			continue
		}
		if attribute.Name == effectsAttribute {
			continue
		}
		if attribute.Name == mustUseAttribute {
			a.addDiagnostic("E4201", span(module.file.Path, attribute.Span), RelatedSpan{Role: "declaration", Span: span(module.file.Path, resource.NameSpan)})
			continue
		}
		a.addUnsupported(module.file.Path, attribute.Span, "resource attribute @"+attribute.Name)
	}
	if resource.Cleanup != nil {
		for _, attribute := range resource.Cleanup.Attributes {
			if attribute.Name == constEvalAttribute {
				continue
			}
			if attribute.Name == effectsAttribute {
				continue
			}
			if attribute.Name == mustUseAttribute {
				a.addDiagnostic("E4201", span(module.file.Path, attribute.Span), RelatedSpan{Role: "cleanup-declaration", Span: span(module.file.Path, resource.Cleanup.Span)})
			}
		}
	}
	result := hir.Resource{
		Name:              resource.Name,
		Public:            resource.Public,
		Type:              value.typ,
		Span:              span(module.file.Path, resource.Span),
		DocumentationSpan: optionalSpan(module.file.Path, resource.DocumentationSpan),
		DropKind:          hir.ResourceDropMissing,
		DropSymbol:        hir.NoSymbol,
		DropEffects:       slices.Clone(value.effects),
	}
	if info := a.resourceTypes[value.typ]; info != nil {
		result.Abilities = info.abilities.canonicalHIR()
		result.Fields = make([]hir.ResourceField, 0, len(info.orderedFields))
		for _, field := range info.orderedFields {
			result.Fields = append(result.Fields, hir.ResourceField{Name: field.name, Type: field.typ, Span: field.span})
		}
	}
	if resource.Cleanup == nil || resource.Cleanup.Body == nil {
		a.addUnsupported(module.file.Path, resource.Span, "resource without a cleanup block")
		return result
	}
	result.DropSpan = span(module.file.Path, resource.Cleanup.Span)
	if len(resource.Cleanup.Body.Statements) == 0 && resource.Cleanup.Body.Tail == nil {
		if len(value.effects) > 0 && value.effectsValid {
			a.addDiagnostic("E7101", span(module.file.Path, value.effectSpan), RelatedSpan{
				Role: "cleanup-declaration", Span: span(module.file.Path, resource.Cleanup.Span),
			})
		}
		result.DropKind = hir.ResourceDropEmpty
		return result
	}
	result.DropKind = hir.ResourceDropNonEmpty
	if function := a.dropFunctions[resource]; function != nil {
		if dropSymbol := a.functions[function]; dropSymbol != nil {
			result.DropSymbol = dropSymbol.id
		}
	}
	return result
}

func (c *typeChecker) checkFunction() hir.Function {
	functionSymbol := c.analyzer.functions[c.function]
	for _, attribute := range c.function.Attributes {
		if attribute.Name == effectsAttribute || attribute.Name == mustUseAttribute || attribute.Name == constEvalAttribute {
			continue
		}
		c.analyzer.addUnsupported(c.module.file.Path, attribute.Span, "function attribute @"+attribute.Name)
	}
	resultType := hir.TypeVoid
	if c.analyzer.dropOwners[c.function] == nil {
		resultType = c.canonicalType(c.function.ResultType)
	}
	c.resultType = resultType
	functionSymbol.typ = resultType
	if _, _, borrowLeaves := safeBorrowLeaf(resultType); borrowLeaves > 1 {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.ResultType.Span, "function result with more than one safe-borrow leaf")
	}
	functionSymbol.mustUse = c.analyzer.finalizeMustUse(c.module.file.Path, c.function.ResultType.Span, functionSymbol, resultType)
	if errorSpan, ok := resultFailureSpan(c.function.ResultType); ok {
		functionSymbol.resultError = errorSpan
	}
	result := hir.Function{
		Symbol:            functionSymbol.id,
		Name:              functionSymbol.name,
		Public:            c.function.Public,
		Unsafe:            c.function.Unsafe,
		Span:              span(c.module.file.Path, c.function.Span),
		DocumentationSpan: optionalSpan(c.module.file.Path, c.function.DocumentationSpan),
		Effects:           slices.Clone(functionSymbol.effects),
		MustUse:           functionSymbol.mustUse,
		ResultType:        resultType,
	}
	if c.function.Owner != "" {
		if owner := c.analyzer.ownerSymbol(c.module, c.function.Owner); owner != nil {
			result.OwnerType = owner.typ
			result.Receiver = c.checkOwnerReceiver(owner)
		}
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
			Symbol:  symbol.id,
			Name:    parameter.Name,
			Type:    symbol.typ,
			Mutable: parameter.Mutable,
			Span:    parameterSpan,
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
	body, bodyType, _ := c.checkBlock(c.function.Body, resultType)
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
	} else if resultType != hir.TypeVoid && !blockAlwaysReturns(c.function.Body) {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Body.Span, "non-void function without a checked return value")
	}
	return result
}

func blockAlwaysReturns(block *ast.Block) bool {
	if block == nil {
		return false
	}
	for index := range block.Statements {
		statement := &block.Statements[index]
		if statement.Kind == syntax.KindReturn || expressionAlwaysReturns(statement.Expression) {
			return true
		}
	}
	return block.Tail != nil
}

func expressionAlwaysReturns(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	switch expression.Kind {
	case syntax.KindIf:
		if len(expression.Children) < 3 {
			return false
		}
		thenBranch := expression.Children[1]
		elseBranch := expression.Children[2]
		return thenBranch != nil && elseBranch != nil && blockAlwaysReturns(thenBranch.Block) &&
			(elseBranch.Kind == syntax.KindIf && expressionAlwaysReturns(elseBranch) || blockAlwaysReturns(elseBranch.Block))
	case syntax.KindUnsafe, syntax.KindRegion:
		for _, child := range expression.Children {
			if child != nil && blockAlwaysReturns(child.Block) {
				return true
			}
		}
	}
	return false
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
			expression, actual := c.checkExpression(statement.Expression, c.resultType)
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
				c.resultType,
				span(c.module.file.Path, c.function.ResultType.Span),
			)
		case statement.Kind == syntax.KindAssignment:
			target, targetType := c.checkExpression(statement.Target, hir.TypeInvalid)
			value, actual := c.checkExpression(statement.Expression, targetType)
			if scalarDerefAssignmentTarget(statement.Target) &&
				(!hir.FixedArrayScalarElement(targetType) || !directDerefBorrowPath(statement.Target.Children[0])) {
				c.analyzer.addUnsupported(c.module.file.Path, statement.Target.Span, "scalar dereference assignment requires a named mutable reference to an admitted scalar")
			}
			hirStatement.Operator = statement.Operator
			if hirStatement.Operator == "" {
				hirStatement.Operator = "="
			}
			hirStatement.Target = target
			hirStatement.Expression = value
			if statement.Target != nil && !c.mutableAssignmentTarget(statement.Target, target) {
				c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, statement.Target.Span))
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
			if hirStatement.Operator != "=" {
				validPlace := target != nil && (target.Kind == "path" || target.Kind == "array-index" || target.Kind == "view-index")
				if !validPlace || !hir.CompoundAssignmentType(hirStatement.Operator, targetType) {
					c.analyzer.addUnsupported(c.module.file.Path, statement.Span, "compound assignment "+hirStatement.Operator+" on "+string(targetType))
				}
			}
		case statement.Kind == syntax.KindAssert:
			expression, actual := c.checkExpression(statement.Expression, hir.TypeBool)
			hirStatement.Expression = expression
			if statement.Expression != nil && actual != hir.TypeInvalid && actual != hir.TypeBool {
				c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, statement.Expression.Span))
			}
		case statement.Kind == syntax.KindDefer:
			expression, actual := c.checkExpression(statement.Expression, hir.TypeVoid)
			hirStatement.Expression = expression
			if !validDeferredCall(statement.Expression, expression, actual) {
				c.analyzer.addUnsupported(c.module.file.Path, statement.Span, "defer outside an internal void call with copy-scalar arguments")
			}
		case statement.Kind == syntax.KindExprStmt || statement.Kind == syntax.KindDiscard:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeInvalid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindBurn:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeInvalid)
			hirStatement.Expression = expression
			c.analyzer.validateResourceBurn(c.module, statement.Expression)
		case statement.Kind == syntax.KindWhile:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeVoid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindFor:
			hirStatement.For = c.checkFor(statement.For)
		case statement.Kind == syntax.KindByteFor:
			if reason := c.byteForSourceIssue(statement.ByteFor, statement.Expression); reason != "" {
				c.analyzer.addUnsupported(c.module.file.Path, statement.Span, reason)
			} else {
				hirStatement.Expression, _ = c.checkExpression(statement.Expression, hir.TypeVoid)
				hirStatement.Kind = string(syntax.KindExprStmt)
			}
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

func validDeferredCall(source *ast.Expression, checked *hir.Expression, actual hir.Type) bool {
	if source == nil || source.Kind != syntax.KindCall || checked == nil || checked.Kind != string(syntax.KindCall) ||
		actual != hir.TypeVoid || checked.Symbol == hir.NoSymbol || checked.ForeignABI != "" || len(checked.Children) == 0 {
		return false
	}
	for index := 1; index < len(checked.Children); index++ {
		if !hir.FixedArrayScalarElement(checked.Children[index].Type) {
			return false
		}
	}
	return true
}

func (c *typeChecker) checkFor(loop *ast.ForLoop) *hir.ForLoop {
	if loop == nil || loop.Body == nil {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Span, "malformed for statement")
		return nil
	}
	result := &hir.ForLoop{Span: span(c.module.file.Path, loop.Span)}
	if loop.Initializer != nil {
		result.Initializer = c.checkForClause(loop.Initializer)
		if !validForInitializer(loop.Initializer, result.Initializer) {
			c.analyzer.addUnsupported(c.module.file.Path, loop.Initializer.Span, "for initializer outside the scalar declaration or direct-assignment subset")
		}
	}
	if loop.Condition != nil {
		condition, actual := c.checkExpression(loop.Condition, hir.TypeBool)
		result.Condition = condition
		if !plainForHeaderExpression(loop.Condition) {
			c.analyzer.addUnsupported(c.module.file.Path, loop.Condition.Span, "for condition with calls, borrows, blocks, indexing, or other non-scalar header behavior")
		}
		if actual != hir.TypeInvalid && actual != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, loop.Condition.Span))
		}
	}
	c.loopDepth++
	body, _, _ := c.checkBlock(loop.Body, hir.TypeVoid)
	c.loopDepth--
	result.Body = body
	if loop.Post != nil {
		result.Post = c.checkForClause(loop.Post)
		if !validForPost(loop.Post, result.Post) {
			c.analyzer.addUnsupported(c.module.file.Path, loop.Post.Span, "for post clause outside direct scalar assignment")
		}
	}
	return result
}

func (c *typeChecker) checkForClause(statement *ast.Statement) *hir.Statement {
	if statement == nil {
		return nil
	}
	block := &ast.Block{Span: statement.Span, Statements: []ast.Statement{*statement}}
	checked, _, _ := c.checkBlock(block, hir.TypeVoid)
	if len(checked.Statements) != 1 {
		return nil
	}
	result := checked.Statements[0]
	return &result
}

func validForInitializer(source *ast.Statement, checked *hir.Statement) bool {
	if source == nil || checked == nil {
		return false
	}
	if source.Binding != nil {
		return checked.Kind == string(syntax.KindBinding) && checked.Expression != nil &&
			forHeaderScalar(checked.Expression.Type) && plainForHeaderExpression(source.Binding.Expression)
	}
	return validForScalarAssignment(source, checked)
}

func validForPost(source *ast.Statement, checked *hir.Statement) bool {
	return validForScalarAssignment(source, checked)
}

func validForScalarAssignment(source *ast.Statement, checked *hir.Statement) bool {
	return source != nil && checked != nil && source.Kind == syntax.KindAssignment &&
		source.Target != nil && source.Target.Kind == syntax.KindPath &&
		checked.Target != nil && checked.Target.Kind == string(syntax.KindPath) &&
		checked.Expression != nil && forHeaderScalar(checked.Target.Type) &&
		checked.Expression.Type == checked.Target.Type && plainForHeaderExpression(source.Expression)
}

func forHeaderScalar(value hir.Type) bool {
	return value == hir.TypeBool || hir.ScalarOrderedType(value)
}

func plainForHeaderExpression(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	switch expression.Kind {
	case syntax.KindLiteral, syntax.KindPath:
		return true
	case syntax.KindGroup, syntax.KindBinary, syntax.KindCast:
		for _, child := range expression.Children {
			if !plainForHeaderExpression(child) {
				return false
			}
		}
		return len(expression.Children) > 0
	case syntax.KindPrefix:
		if expression.Text != "+" && expression.Text != "-" && expression.Text != "!" && expression.Text != "~" {
			return false
		}
		return len(expression.Children) == 1 && plainForHeaderExpression(expression.Children[0])
	default:
		return false
	}
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
	case ast.PrivateStorageLayout:
		actual = c.checkStorageLayout(expression, result)
	case ast.PrivateStorageTake:
		actual = c.checkStorageTake(expression, result)
	case ast.PrivateStoragePut:
		actual = c.checkStoragePut(expression, result)
	case ast.PrivateStorageBorrow:
		actual = c.checkStorageBorrow(expression, result)
	case ast.PrivateStorageElementAddress:
		actual = c.checkStorageElementAddress(expression, result)
	case syntax.KindLiteral:
		var supported bool
		actual, supported = literalType(expression.Text, expected)
		if !supported {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "literal "+literalClass(expression.Text))
		} else if isIntegerLiteral(expression.Text) && !integerLiteralFits(expression.Text, actual) {
			c.analyzer.addDiagnostic("E1013", span(c.module.file.Path, expression.Span))
			actual = hir.TypeInvalid
		}
	case syntax.KindPath:
		if expression.Text == sourceLocationPath {
			c.analyzer.addDiagnostic("E4401", span(c.module.file.Path, expression.Span))
			actual = hir.TypeInvalid
		} else if symbol := c.analyzer.resolved[expression]; symbol != nil {
			if value, constant := c.analyzer.constantValues[symbol]; constant {
				result.Kind = string(syntax.KindLiteral)
				result.Value = value.canonical()
				result.Cooked = ""
				actual = value.typ
			} else {
				result.Symbol = symbol.id
				actual = symbol.typ
			}
		} else if variant, option := optionVariantPath(expression.Text); option {
			actual = c.checkOptionPath(expression, result, expected, variant)
		} else if strings.HasPrefix(expression.Text, ".") {
			actual = c.checkContextualEnumPath(expression, result, expected)
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
		actual = c.checkPrefix(expression, result, expected)
	case syntax.KindBinary:
		actual = c.checkBinary(expression, result)
	case syntax.KindCast:
		actual = c.checkCast(expression, result)
	case syntax.KindCall:
		actual = c.checkCall(expression, result, expected)
	case syntax.KindArray:
		actual = c.checkArray(expression, result, expected)
	case syntax.KindIndex:
		actual = c.checkIndex(expression, result)
	case syntax.KindSlice:
		actual = c.checkSlice(expression, result)
	case syntax.KindIf:
		actual = c.checkIf(expression, result, expected)
	case syntax.KindSwitch:
		actual = c.checkMatch(expression, result, expected)
		result.Kind = string(syntax.KindMatch)
	case syntax.KindWhile:
		actual = c.checkWhile(expression, result)
	case syntax.KindRecord:
		actual = c.checkRecord(expression, result, expected)
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

func (c *typeChecker) checkArray(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if expression.Text != "list" || len(expression.Children) == 0 {
		feature := "empty fixed-array literal"
		if expression.Text == "repeat" {
			feature = "repeated fixed-array literal"
		} else if expression.Text != "empty" {
			feature = "malformed fixed-array literal"
		}
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, feature)
		return hir.TypeInvalid
	}
	expectedElement, _, hasExpectedArray := hir.FixedArrayParts(expected)
	elementType := expectedElement
	valid := true
	for index := range expression.Children {
		childExpected := elementType
		if index == 0 && !hasExpectedArray {
			childExpected = hir.TypeInvalid
		}
		child, childType := c.checkExpression(expression.Children[index], childExpected)
		result.Children = appendExpression(result.Children, child)
		if index == 0 && !hasExpectedArray {
			elementType = childType
		}
		if childType == hir.TypeInvalid || elementType == hir.TypeInvalid {
			valid = false
			continue
		}
		if childType != elementType {
			c.requireType(
				"E1006",
				span(c.module.file.Path, expression.Children[index].Span),
				childType,
				elementType,
				span(c.module.file.Path, expression.Span),
			)
			valid = false
		}
	}
	if !valid {
		return hir.TypeInvalid
	}
	if !hir.FixedArrayScalarElement(elementType) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "fixed-array element type "+string(elementType))
		return hir.TypeInvalid
	}
	length := uint64(len(expression.Children))
	if !hir.ExecutableFixedArrayShape(elementType, length) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, fmt.Sprintf("fixed-array literal with %d elements", length))
		return hir.TypeInvalid
	}
	result.Kind = "array"
	result.Operator = "list"
	return hir.FixedArray(elementType, length)
}

func (c *typeChecker) checkIndex(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed index expression")
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	index, indexType := c.checkExpression(expression.Children[1], hir.TypeUSize)
	result.Children = appendExpression(result.Children, receiver, index)
	arrayElement, _, fixedArray := hir.FixedArrayParts(receiverType)
	sharedBytes := hir.SharedBorrow(hir.TypeByteSlice)
	mutableBytes := hir.MutableBorrow(hir.TypeByteSlice)
	if receiverType != sharedBytes && receiverType != mutableBytes && !fixedArray {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "indexing outside byte views")
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
	if fixedArray {
		result.Kind = "array-index"
		result.Operator = "[]"
		return arrayElement
	}
	result.Kind = "view-index"
	result.Operator = "[]"
	return hir.TypeU8
}

func (c *typeChecker) mutableAssignmentTarget(source *ast.Expression, target *hir.Expression) bool {
	if source == nil || target == nil {
		return false
	}
	switch target.Kind {
	case "path":
		resolved := c.analyzer.resolved[source]
		return resolved != nil && resolved.mutable
	case "array-index":
		if len(source.Children) != 2 || len(target.Children) != 2 {
			return false
		}
		receiver := assignmentRootPath(source.Children[0])
		resolved := c.analyzer.resolved[receiver]
		_, _, fixedArray := hir.FixedArrayParts(target.Children[0].Type)
		return fixedArray && resolved != nil && resolved.mutable
	case "view-index":
		return len(target.Children) == 2 && target.Children[0].Type == hir.MutableBorrow(hir.TypeByteSlice)
	case "prefix":
		if !scalarDerefAssignmentTarget(source) || target.Operator != "*" || len(target.Children) != 1 ||
			!hir.FixedArrayScalarElement(target.Type) || !directDerefBorrowPath(source.Children[0]) {
			return false
		}
		element, mutable, borrowed := hir.BorrowElement(target.Children[0].Type)
		return borrowed && mutable && element == target.Type
	case "field-expression":
		if len(source.Children) != 1 || len(target.Children) != 1 {
			return false
		}
		if _, mutable, borrow := hir.BorrowElement(target.Children[0].Type); borrow {
			return mutable
		}
		receiver := assignmentRootPath(source.Children[0])
		resolved := c.analyzer.resolved[receiver]
		return resolved != nil && resolved.mutable
	default:
		return false
	}
}

func scalarDerefAssignmentTarget(expression *ast.Expression) bool {
	return expression != nil && expression.Kind == syntax.KindPrefix && expression.Text == "*" && len(expression.Children) == 1
}

func directDerefBorrowPath(expression *ast.Expression) bool {
	for expression != nil && expression.Kind == syntax.KindGroup && len(expression.Children) == 1 {
		expression = expression.Children[0]
	}
	return expression != nil && expression.Kind == syntax.KindPath
}

func assignmentRootPath(expression *ast.Expression) *ast.Expression {
	for expression != nil && expression.Kind == syntax.KindGroup && len(expression.Children) == 1 {
		expression = expression.Children[0]
	}
	if expression == nil || expression.Kind != syntax.KindPath {
		return nil
	}
	return expression
}

func (c *typeChecker) checkSlice(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 3 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed slice expression")
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	start, startType := c.checkSliceBound(expression.Children[1], "start")
	end, endType := c.checkSliceBound(expression.Children[2], "end")
	result.Children = appendExpression(result.Children, receiver, start, end)
	wantReceiver := hir.SharedBorrow(hir.TypeByteSlice)
	if receiverType != wantReceiver {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "slicing outside immutable byte views")
		return hir.TypeInvalid
	}
	c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), startType, hir.TypeUSize, span(c.module.file.Path, expression.Span))
	c.requireType("E1006", span(c.module.file.Path, expression.Children[2].Span), endType, hir.TypeUSize, span(c.module.file.Path, expression.Span))
	if startType != hir.TypeUSize || endType != hir.TypeUSize {
		return hir.TypeInvalid
	}
	result.Kind = "view-slice"
	result.Operator = ".."
	return wantReceiver
}

func (c *typeChecker) checkSliceBound(expression *ast.Expression, position string) (*hir.Expression, hir.Type) {
	if expression != nil && expression.Kind == syntax.KindOmitBound {
		if expression.Text != position {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed omitted slice bound")
			return nil, hir.TypeInvalid
		}
		return &hir.Expression{
			Kind: "omitted-range-bound", Type: hir.TypeUSize, Span: span(c.module.file.Path, expression.Span),
			Operator: position, Symbol: hir.NoSymbol,
		}, hir.TypeUSize
	}
	return c.checkExpression(expression, hir.TypeUSize)
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
	if hir.ValuePreservingIntegerCast(sourceType, targetType) {
		result.Kind = "scalar-cast"
		return targetType
	}
	sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(sourceType)
	targetElement, targetMutable, targetPointer := hir.RawPointerElement(targetType)
	sameElement := sourceElement == targetElement && (sourceMutable || !targetMutable)
	directLocalBorrow := child != nil && child.Kind == "prefix" && (child.Operator == "&" || child.Operator == "&mut") &&
		len(child.Children) == 1 && child.Children[0].Kind == "path"
	borrowValue := child != nil && child.Kind == "path"
	opaqueElement := targetElement == hir.TypeVoid && (sourceMutable || !targetMutable) &&
		(directLocalBorrow || borrowValue)
	byteSliceBytes := sourceElement == hir.TypeByteSlice && targetElement == hir.TypeU8 && (sourceMutable || !targetMutable)
	stringBytes := sourceElement == hir.TypeStr && !sourceMutable && targetElement == hir.TypeU8 && !targetMutable
	cStringBytes := sourceElement == hir.TypeCStr && !sourceMutable && targetElement == hir.TypeU8 && !targetMutable
	if !sourceBorrow || !targetPointer || !sameElement && !opaqueElement && !byteSliceBytes && !stringBytes && !cStringBytes {
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

func (c *typeChecker) checkRecord(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.TypeArguments) != 0 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "generic-aggregate-specialization")
		return hir.TypeInvalid
	}
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	if path := expression.Children[0]; path != nil && strings.HasPrefix(path.Text, ".") {
		return c.checkContextualEnumRecord(expression, result, expected)
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
	if constructor.resource != nil && (constructor.packageID != c.module.key.packageID || constructor.module != c.module.file.Module.Path) {
		c.analyzer.addDiagnostic("E0103", span(c.module.file.Path, expression.Children[0].Span), RelatedSpan{
			Role: "declaration",
			Span: span(constructor.source, constructor.declaration),
		})
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

func (c *typeChecker) checkContextualEnumPath(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	info := c.analyzer.enumTypes[expected]
	name := strings.TrimPrefix(expression.Text, ".")
	if info == nil || name == "" {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "contextual enum variant without an expected enum type")
		return hir.TypeInvalid
	}
	variant, found := info.variants[name]
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown contextual enum variant "+name)
		return hir.TypeInvalid
	}
	if len(variant.orderedFields) != 0 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "enum payload variant used without record construction")
		return hir.TypeInvalid
	}
	result.Kind = "enum-variant"
	result.Symbol = info.symbol.id
	result.Variant = name
	return expected
}

func (c *typeChecker) checkContextualEnumRecord(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	info := c.analyzer.enumTypes[expected]
	name := strings.TrimPrefix(expression.Children[0].Text, ".")
	if info == nil || name == "" {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "contextual enum construction without an expected enum type")
		return hir.TypeInvalid
	}
	variant, found := info.variants[name]
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown contextual enum variant "+name)
		return hir.TypeInvalid
	}
	result.Kind = "enum-variant"
	result.Symbol = info.symbol.id
	result.Variant = name
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
	return expected
}

func (c *typeChecker) checkField(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, receiver)
	if successType, errorType, isResult := hir.ResultElements(receiverType); isResult && expression.Text == "is_ok" {
		result.Kind = "match"
		for _, arm := range []struct {
			variant string
			value   string
		}{{"ok", "true"}, {"err", "false"}} {
			binding := c.analyzer.allocateSymbol()
			bindingSpan := span(c.module.file.Path, expression.Span)
			payloadType := successType
			if arm.variant == "err" {
				payloadType = errorType
			}
			c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding, Name: "_", Type: payloadType, Span: bindingSpan})
			result.Arms = append(result.Arms, hir.MatchArm{
				Variant: arm.variant, Binding: binding, BindingSpan: bindingSpan,
				Expression: hir.Expression{Kind: "literal", Type: hir.TypeBool, Span: bindingSpan, Value: arm.value, Symbol: hir.NoSymbol},
			})
		}
		return hir.TypeBool
	}
	if element, _, borrow := hir.BorrowElement(receiverType); borrow && (element == hir.TypeStr || element == hir.TypeByteSlice) {
		switch expression.Text {
		case "len":
			result.Kind = "view-length"
			result.Operator = "len"
			return hir.TypeUSize
		case "is_empty":
			result.Kind = "view-empty"
			result.Operator = "is_empty"
			return hir.TypeBool
		default:
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown borrowed-view field "+expression.Text)
			return hir.TypeInvalid
		}
	}
	if element, length, array := hir.FixedArrayParts(receiverType); array && hir.ExecutableFixedArrayShape(element, length) {
		if expression.Text == "len" {
			result.Kind = "array-length"
			result.Operator = "len"
			return hir.TypeUSize
		}
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown fixed-array field "+expression.Text)
		return hir.TypeInvalid
	}
	if element, _, borrow := hir.BorrowElement(receiverType); borrow {
		receiverType = element
	}
	fields := map[string]resourceField(nil)
	if info, found := c.analyzer.resourceType(receiverType); found {
		owner := info.symbol
		if owner.packageID != c.module.key.packageID || owner.module != c.module.file.Module.Path {
			c.analyzer.addDiagnostic("E0103", span(c.module.file.Path, expression.Span), RelatedSpan{
				Role: "declaration", Span: span(owner.source, owner.declaration),
			})
			return hir.TypeInvalid
		}
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

func (c *typeChecker) checkPrefix(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	result.Operator = expression.Text
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	if expression.Text == "-" && directI32MinimumLiteral(expression.Children[0]) {
		result.Kind = "literal"
		result.Value = "-2147483648"
		result.Operator = ""
		return hir.TypeI32
	}
	child, childType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, child)
	switch expression.Text {
	case "!":
		if childType != hir.TypeInvalid && childType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
		}
		return hir.TypeBool
	case "+", "-":
		c.requireType(
			"E1006",
			span(c.module.file.Path, expression.Children[0].Span),
			childType,
			hir.TypeI32,
			span(c.module.file.Path, expression.Span),
		)
		return hir.TypeI32
	case "~":
		if childType != hir.TypeInvalid && !hir.BitwiseNotType(childType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "bitwise not on "+string(childType))
		}
		return childType
	case "&":
		if child != nil {
			if field, ok := sharedFieldBorrow(*child); ok {
				*result = field
				result.Span = span(c.module.file.Path, expression.Span)
				return field.Type
			}
		}
		if c.analyzer.resolved[expression.Children[0]] == nil {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "borrow of a non-place expression")
			return hir.TypeInvalid
		}
		if element, length, array := hir.FixedArrayParts(childType); expected == hir.SharedBorrow(hir.TypeByteSlice) &&
			array && element == hir.TypeU8 && hir.ExecutableFixedArrayShape(element, length) {
			result.Kind = "array-view-borrow"
			return expected
		}
		return hir.SharedBorrow(childType)
	case "&mut":
		target := c.analyzer.resolved[expression.Children[0]]
		if target == nil || !target.mutable {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, expression.Children[0].Span))
			return hir.TypeInvalid
		}
		if element, length, array := hir.FixedArrayParts(childType); expected == hir.MutableBorrow(hir.TypeByteSlice) &&
			array && element == hir.TypeU8 && hir.ExecutableFixedArrayShape(element, length) {
			result.Kind = "array-view-borrow"
			return expected
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
		} else if leftType != hir.TypeInvalid && !hir.ScalarEqualityType(leftType) {
			if _, _, pointer := hir.RawPointerElement(leftType); !pointer {
				c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "equality comparison on "+string(leftType))
			}
		}
		return hir.TypeBool
	case "<", "<=", ">", ">=":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
		} else if leftType != hir.TypeInvalid && !hir.ScalarOrderedType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "ordered comparison on "+string(leftType))
		}
		return hir.TypeBool
	case "<<":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedShiftType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked left shift on "+string(leftType))
		}
		return leftType
	case "+", "-", "*":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedArithmeticType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked arithmetic on "+string(leftType))
		}
		return leftType
	case "/", "%":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedDivisionType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked division on "+string(leftType))
		}
		return leftType
	case ">>":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedShiftType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked right shift on "+string(leftType))
		}
		return leftType
	case "&", "^", "|":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.BitwiseIntegerType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "bitwise operation on "+string(leftType))
		}
		return leftType
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "binary operator "+expression.Text)
		return hir.TypeInvalid
	}
}

func (c *typeChecker) checkCall(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if !c.analyzer.checkCallArgumentLabels(c.module, expression) {
		return hir.TypeInvalid
	}
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	calleeExpression := expression.Children[0]
	if calleeExpression != nil && calleeExpression.Kind == syntax.KindFieldExpr && !isRegionPlaceCall(expression) {
		return c.checkMethodCall(expression, result)
	}
	if calleeExpression != nil && calleeExpression.Kind == syntax.KindPath {
		if calleeExpression.Text == sourceLocationPath {
			return c.checkSourceLocation(expression, result)
		}
		if variant, ok := resultVariantPath(calleeExpression.Text); ok {
			return c.checkResultConstructor(expression, result, expected, variant)
		}
		if variant, ok := optionVariantPath(calleeExpression.Text); ok {
			return c.checkOptionConstructor(expression, result, expected, variant)
		}
		if constructor, ok := borrowedViewConstructorPath(calleeExpression.Text); ok {
			return c.checkBorrowedViewConstructor(expression, result, calleeExpression.Text, constructor)
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
		c.checkCallEffects(calleeExpression, result, callee)
		return c.checkForeignCall(expression, result, callee)
	}
	if callee.function == nil {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "call target without a function declaration")
		return hir.TypeInvalid
	}
	result.Symbol = callee.id
	c.checkCallEffects(calleeExpression, result, callee)
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
			expected = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.Parameters[index].Type)
			expectedSpan = callee.function.Parameters[index].Type.Span
		}
		child, actual := c.checkExpression(argument, expected)
		result.Children = appendExpression(result.Children, child)
		if expected != hir.TypeInvalid {
			c.requireCallArgumentType(
				span(c.module.file.Path, argument.Span),
				actual,
				expected,
				span(callee.source, expectedSpan),
			)
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.ResultType)
	if errorSpan, ok := resultFailureSpan(callee.function.ResultType); ok {
		callee.resultError = errorSpan
	}
	return callee.typ
}

func (c *typeChecker) checkOwnerReceiver(owner *symbol) string {
	for index := range c.function.Parameters {
		parameter := &c.function.Parameters[index]
		if parameter.Name != "this" {
			continue
		}
		if index != 0 {
			c.analyzer.addUnsupported(c.module.file.Path, parameter.NameSpan, "method receiver that is not the first parameter")
			return "invalid"
		}
		receiverType := c.canonicalType(parameter.Type)
		baseType := receiverType
		mode := "value"
		if element, mutable, borrow := hir.BorrowElement(receiverType); borrow {
			baseType = element
			mode = "shared"
			if mutable {
				mode = "exclusive"
			}
		}
		if baseType != owner.typ {
			c.analyzer.addUnsupported(c.module.file.Path, parameter.Type.Span, "method receiver type that does not match its owner")
			return "invalid"
		}
		return mode
	}
	return ""
}

func (c *typeChecker) checkMethodCall(expression *ast.Expression, result *hir.Expression) hir.Type {
	calleeExpression := expression.Children[0]
	if len(calleeExpression.Children) != 1 {
		return hir.TypeInvalid
	}
	receiverExpression := calleeExpression.Children[0]
	receiver, receiverType := c.checkExpression(receiverExpression, hir.TypeInvalid)
	ownerType := receiverType
	if element, _, borrow := hir.BorrowElement(ownerType); borrow {
		ownerType = element
	}
	callee := c.analyzer.methodSymbol(ownerType, calleeExpression.Text)
	if callee == nil || callee.function == nil || len(callee.function.Parameters) == 0 || callee.function.Parameters[0].Name != "this" {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "unknown method "+calleeExpression.Text)
		return hir.TypeInvalid
	}
	if callee.module != c.module.file.Module.Path && !callee.public {
		c.analyzer.addDiagnostic("E0103", span(c.module.file.Path, calleeExpression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
		return hir.TypeInvalid
	}
	expectedReceiver := c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.Parameters[0].Type)
	if element, mutable, borrow := hir.BorrowElement(expectedReceiver); borrow && receiverType == element {
		target := c.analyzer.resolved[receiverExpression]
		if target == nil || mutable && !target.mutable {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, receiverExpression.Span))
			return hir.TypeInvalid
		}
		operator := "&"
		if mutable {
			operator = "&mut"
		}
		borrowExpression := &ast.Expression{
			Kind: syntax.KindPrefix, Span: receiverExpression.Span, Text: operator, Children: []*ast.Expression{receiverExpression},
		}
		calleeExpression.Children[0] = borrowExpression
		receiver = &hir.Expression{
			Kind: string(syntax.KindPrefix), Span: span(c.module.file.Path, receiverExpression.Span), Operator: operator,
			Type: expectedReceiver, Symbol: hir.NoSymbol, Children: []hir.Expression{*receiver},
		}
		receiverType = expectedReceiver
	}
	c.requireType("E1006", span(c.module.file.Path, receiverExpression.Span), receiverType, expectedReceiver, span(callee.source, callee.function.Parameters[0].Type.Span))
	c.analyzer.methodCalls[expression] = callee
	result.Symbol = callee.id
	c.checkCallEffects(calleeExpression, result, callee)
	result.Children = append(result.Children, hir.Expression{
		Kind: string(syntax.KindPath), Type: hir.TypeInvalid, Span: span(c.module.file.Path, calleeExpression.Span), Symbol: callee.id,
	})
	result.Children = appendExpression(result.Children, receiver)
	arguments := expression.Children[1:]
	if len(arguments)+1 != len(callee.function.Parameters) {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
	}
	for index, argument := range arguments {
		parameterIndex := index + 1
		expectedType := hir.TypeInvalid
		expectedSpan := callee.typeSpan
		if parameterIndex < len(callee.function.Parameters) {
			expectedType = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.Parameters[parameterIndex].Type)
			expectedSpan = callee.function.Parameters[parameterIndex].Type.Span
		}
		child, actual := c.checkExpression(argument, expectedType)
		result.Children = appendExpression(result.Children, child)
		if expectedType != hir.TypeInvalid {
			c.requireType("E1006", span(c.module.file.Path, argument.Span), actual, expectedType, span(callee.source, expectedSpan))
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.ResultType)
	return callee.typ
}

func (a *analyzer) methodSymbol(ownerType hir.Type, name string) *symbol {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if function.Owner == "" || function.Name != name || len(function.Parameters) == 0 || function.Parameters[0].Name != "this" {
				continue
			}
			owner := a.ownerSymbol(module, function.Owner)
			if owner != nil && owner.typ == ownerType {
				return a.functions[function]
			}
		}
	}
	return nil
}

func (a *analyzer) callableForCall(expression *ast.Expression) *symbol {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 {
		return nil
	}
	if method := a.methodCalls[expression]; method != nil {
		return method
	}
	return a.resolved[expression.Children[0]]
}

func (a *analyzer) callArguments(expression *ast.Expression) []*ast.Expression {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 {
		return nil
	}
	arguments := expression.Children[1:]
	if a.methodCalls[expression] == nil {
		return arguments
	}
	callee := expression.Children[0]
	if callee == nil || callee.Kind != syntax.KindFieldExpr || len(callee.Children) != 1 {
		return arguments
	}
	result := make([]*ast.Expression, 0, len(arguments)+1)
	result = append(result, callee.Children[0])
	return append(result, arguments...)
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
			expected = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), declaration.Parameters[index].Type)
			expectedSpan = declaration.Parameters[index].Type.Span
		}
		child, actual := c.checkExpression(argument, expected)
		result.Children = appendExpression(result.Children, child)
		if expected != hir.TypeInvalid {
			c.requireType("E1006", span(c.module.file.Path, argument.Span), actual, expected, span(callee.source, expectedSpan))
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), declaration.ResultType)
	return callee.typ
}

func (c *typeChecker) checkOptionPath(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantNone {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".some used without one payload")
		return hir.TypeInvalid
	}
	_, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".none without contextual Option type")
		return hir.TypeInvalid
	}
	result.Kind = "option-variant"
	result.Variant = optionVariantNone
	return expected
}

func (c *typeChecker) checkOptionConstructor(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantSome {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".none is a bare variant, not a call")
		return hir.TypeInvalid
	}
	payloadType, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option constructor without contextual Option type")
		return hir.TypeInvalid
	}
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".some arity other than one")
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
		variantName := strings.TrimPrefix(pattern.Text, ".")
		variant, found := info.variants[variantName]
		if !found {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern from a different or unresolved enum")
			return hir.TypeInvalid
		}
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
	continuing := 0
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
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a contextual Result variant")
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
		if !returnOnlyMatchArm(armExpression.Children[1]) {
			continuing++
			if actual == hir.TypeInvalid {
				actual = bodyType
			} else {
				c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
			}
		}
		result.Arms = append(result.Arms, hir.MatchArm{Variant: variant, Binding: binding.id, BindingSpan: bindingSpan, Expression: *body})
	}
	if len(seen) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive Result match")
		return hir.TypeInvalid
	}
	if continuing == 0 {
		return hir.TypeVoid
	}
	return actual
}

func returnOnlyMatchArm(expression *ast.Expression) bool {
	if expression == nil || expression.Kind != syntax.KindBlock || expression.Block == nil || expression.Block.Tail != nil {
		return false
	}
	statements := expression.Block.Statements
	return len(statements) > 0 && statements[len(statements)-1].Kind == syntax.KindReturn
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
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a contextual Option variant")
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
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, ".some pattern without one payload binding")
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
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, ".none pattern with a payload binding")
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
	if isIntegerLiteral(raw) {
		switch expected {
		case hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
			return expected, true
		}
		return hir.TypeI32, true
	}
	return hir.TypeInvalid, false
}

func isIntegerLiteral(raw string) bool {
	if raw == "" || raw[0] < '0' || raw[0] > '9' {
		return false
	}
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") || strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B") {
		return true
	}
	return !strings.ContainsAny(raw, ".eE")
}

func integerLiteralFits(raw string, selected hir.Type) bool {
	var maximum uint64
	switch selected {
	case hir.TypeI32:
		maximum = 1<<31 - 1
	case hir.TypeU8:
		maximum = 1<<8 - 1
	case hir.TypeU32:
		maximum = 1<<32 - 1
	case hir.TypeUSize:
		maximum = ^uint64(0)
	default:
		return true
	}
	comparison, valid := compareIntegerLiteralMagnitude(raw, maximum)
	return valid && comparison <= 0
}

func directI32MinimumLiteral(expression *ast.Expression) bool {
	if expression == nil || expression.Kind != syntax.KindLiteral || !isIntegerLiteral(expression.Text) {
		return false
	}
	comparison, valid := compareIntegerLiteralMagnitude(expression.Text, 1<<31)
	return valid && comparison == 0
}

func compareIntegerLiteralMagnitude(raw string, maximum uint64) (int, bool) {
	base, start := 10, 0
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		base, start = 16, 2
	} else if strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B") {
		base, start = 2, 2
	}
	maximumDigits := strconv.FormatUint(maximum, base)
	significantDigits := 0
	seenNonZero := false
	for index := start; index < len(raw); index++ {
		if raw[index] == '_' {
			continue
		}
		digit, valid := integerDigitValue(raw[index])
		if !valid || digit >= base {
			return 0, false
		}
		if !seenNonZero && digit == 0 {
			continue
		}
		seenNonZero = true
		significantDigits++
	}
	if !seenNonZero {
		return -1, true
	}
	if significantDigits < len(maximumDigits) {
		return -1, true
	}
	if significantDigits > len(maximumDigits) {
		return 1, true
	}
	maximumIndex := 0
	seenNonZero = false
	for index := start; index < len(raw); index++ {
		if raw[index] == '_' {
			continue
		}
		digit, _ := integerDigitValue(raw[index])
		if !seenNonZero && digit == 0 {
			continue
		}
		seenNonZero = true
		maximumDigit, _ := integerDigitValue(maximumDigits[maximumIndex])
		if digit < maximumDigit {
			return -1, true
		}
		if digit > maximumDigit {
			return 1, true
		}
		maximumIndex++
	}
	return 0, true
}

func integerDigitValue(value byte) (int, bool) {
	switch {
	case value >= '0' && value <= '9':
		return int(value - '0'), true
	case value >= 'a' && value <= 'f':
		return int(value-'a') + 10, true
	case value >= 'A' && value <= 'F':
		return int(value-'A') + 10, true
	default:
		return 0, false
	}
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

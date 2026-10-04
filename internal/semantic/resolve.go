package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type symbol struct {
	id           hir.SymbolID
	packageID    hir.PackageID
	kind         string
	name         string
	module       string
	source       string
	declaration  diagnostic.Span
	typeSpan     diagnostic.Span
	resultError  diagnostic.Span
	effectSpan   diagnostic.Span
	typ          hir.Type
	effects      []hir.Effect
	effectsValid bool
	mustUse      bool
	mustUseSpan  diagnostic.Span
	mutable      bool
	public       bool
	constant     *ast.Constant
	function     *ast.Function
	foreign      *ast.ForeignFunction
	structure    *ast.Struct
	resource     *ast.Resource
	enumeration  *ast.Enum
}

type moduleKey struct {
	packageID hir.PackageID
	path      string
}

type moduleScope struct {
	key       moduleKey
	file      File
	functions map[string]*symbol
	constants map[string]*symbol
	structs   map[string]*symbol
	resources map[string]*symbol
	enums     map[string]*symbol
	imports   map[string]*moduleScope
}

func (m *moduleScope) identityPath() string {
	return hir.QualifiedModulePath(m.key.packageID, m.file.Module.Path)
}

type lexicalScope struct {
	parent *lexicalScope
	names  map[string]*symbol
}

type analyzer struct {
	nominalBindings       map[string]NominalTypeBinding
	vectorForTypes        map[vectorTypeKey]bool
	files                 []File
	modules               map[moduleKey]*moduleScope
	packageImports        map[packageImportKey]hir.PackageID
	moduleOrder           []*moduleScope
	nextSymbol            hir.SymbolID
	resolved              map[*ast.Expression]*symbol
	bindingSymbols        map[*ast.Binding]*symbol
	parameter             map[*ast.Parameter]*symbol
	parameterIndex        map[*symbol]int
	functions             map[*ast.Function]*symbol
	constants             map[*ast.Constant]*symbol
	foreignFunctions      map[*ast.ForeignFunction]*symbol
	structs               map[*ast.Struct]*symbol
	resources             map[*ast.Resource]*symbol
	enums                 map[*ast.Enum]*symbol
	regions               map[*ast.Expression]*symbol
	regionType            *symbol
	diagnostics           []Diagnostic
	resolutions           []Resolution
	types                 []TypeFact
	borrows               []BorrowFact
	unsupported           []Unsupported
	flowStarted           bool
	resourceTypes         map[hir.Type]*resourceInfo
	structTypes           map[hir.Type]*structInfo
	enumTypes             map[hir.Type]*enumInfo
	resourceStates        []ResourceStateFact
	expressionTypes       map[*ast.Expression]hir.Type
	regionOrigins         []RegionOriginFact
	resultHandling        []ResultHandlingFact
	effectFacts           []EffectFact
	resourceLoops         []resourceLoopFrame
	resourceScopes        []map[*symbol]struct{}
	resourceCaller        *symbol
	resourceCleanupExempt *symbol
	resultLoops           []resultLoopFrame
	resultNeutralScalars  map[*ast.Expression]bool
	borrowReturns         map[*symbol]borrowReturnSummary
	dropFunctions         map[*ast.Resource]*ast.Function
	dropOwners            map[*ast.Function]*ast.Resource
	methodCalls           map[*ast.Expression]*symbol
	constantFacts         []ConstantFact
	constantValues        map[*symbol]compileTimeValue
	constEvalFunctions    map[*symbol]bool
}

func newAnalyzer(files []File, packageImports map[packageImportKey]hir.PackageID) *analyzer {
	return &analyzer{
		files:              files,
		modules:            make(map[moduleKey]*moduleScope, len(files)),
		packageImports:     packageImports,
		resolved:           make(map[*ast.Expression]*symbol),
		bindingSymbols:     make(map[*ast.Binding]*symbol),
		parameter:          make(map[*ast.Parameter]*symbol),
		parameterIndex:     make(map[*symbol]int),
		functions:          make(map[*ast.Function]*symbol),
		constants:          make(map[*ast.Constant]*symbol),
		foreignFunctions:   make(map[*ast.ForeignFunction]*symbol),
		structs:            make(map[*ast.Struct]*symbol),
		resources:          make(map[*ast.Resource]*symbol),
		enums:              make(map[*ast.Enum]*symbol),
		regions:            make(map[*ast.Expression]*symbol),
		resourceTypes:      make(map[hir.Type]*resourceInfo),
		structTypes:        make(map[hir.Type]*structInfo),
		enumTypes:          make(map[hir.Type]*enumInfo),
		expressionTypes:    make(map[*ast.Expression]hir.Type),
		borrowReturns:      make(map[*symbol]borrowReturnSummary),
		dropFunctions:      make(map[*ast.Resource]*ast.Function),
		dropOwners:         make(map[*ast.Function]*ast.Resource),
		methodCalls:        make(map[*ast.Expression]*symbol),
		constantValues:     make(map[*symbol]compileTimeValue),
		constEvalFunctions: make(map[*symbol]bool),
		regionType:         &symbol{id: hir.NoSymbol, kind: "capability", name: "Region", typ: hir.TypeRegion},
	}
}

func (a *analyzer) resolve() {
	a.declareModules()
	a.declareEnums()
	a.declareStructs()
	a.declareResources()
	a.declareConstants()
	a.declareCallables()
	a.declareResourceDropFunctions()
	a.resolveImports()
	if len(a.diagnostics) > 0 {
		return
	}
	for _, module := range a.moduleOrder {
		constantScope := &lexicalScope{names: make(map[string]*symbol)}
		for index := range module.file.Module.Constants {
			a.resolveExpression(module, constantScope, module.file.Module.Constants[index].Expression)
		}
		for index := range module.file.Module.StaticAssertions {
			a.resolveExpression(module, constantScope, module.file.Module.StaticAssertions[index].Expression)
		}
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			if function := a.dropFunctions[resource]; function != nil {
				a.resolveFunction(module, function)
			}
		}
		for index := range module.file.Module.Functions {
			a.resolveFunction(module, &module.file.Module.Functions[index])
		}
	}
}

func (a *analyzer) declareConstants() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Constants {
			constant := &module.file.Module.Constants[index]
			if a.rejectSourceLocationName(module.file.Path, constant.Name, constant.NameSpan) {
				continue
			}
			if previous := module.item(constant.Name); previous != nil {
				a.addDiagnostic("E4301", span(module.file.Path, constant.NameSpan), RelatedSpan{
					Role: "previous-declaration", Span: span(previous.source, previous.declaration),
				})
				continue
			}
			value := &symbol{
				id: a.allocateSymbol(), packageID: module.key.packageID, kind: "constant", name: constant.Name,
				module: module.file.Module.Path, source: module.file.Path, declaration: constant.NameSpan,
				typeSpan: constant.Type.Span, public: constant.Public, constant: constant,
			}
			module.constants[constant.Name] = value
			a.constants[constant] = value
			if constant.Public || len(constant.Attributes) != 0 || constant.Expression == nil {
				a.addDiagnostic("E4301", span(module.file.Path, constant.Span))
			}
		}
		for index := range module.file.Module.StaticAssertions {
			assertion := &module.file.Module.StaticAssertions[index]
			if assertion.Public || len(assertion.Attributes) != 0 || assertion.Expression == nil {
				a.addDiagnostic("E4301", span(module.file.Path, assertion.Span))
			}
		}
	}
}

func (a *analyzer) declareResourceDropFunctions() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			if a.resources[resource] == nil || resource.Cleanup == nil || resource.Cleanup.Body == nil ||
				len(resource.Cleanup.Body.Statements) == 0 && resource.Cleanup.Body.Tail == nil {
				continue
			}
			receiverName := resource.Cleanup.ReceiverName
			if receiverName == "" {
				receiverName = "this"
			}
			function := &ast.Function{
				Name:     "$drop$" + resource.Name,
				Span:     resource.Cleanup.Span,
				NameSpan: resource.Cleanup.Span,
				Parameters: []ast.Parameter{{
					Name:     receiverName,
					Mutable:  true,
					Span:     resource.Cleanup.Span,
					NameSpan: resource.Cleanup.Span,
					Type:     ast.NamedTypeReference(resource.Name, resource.NameSpan),
				}},
				ResultType: ast.NamedTypeReference("void", resource.Cleanup.Span),
				Body:       resource.Cleanup.Body,
			}
			value := &symbol{
				id:          a.allocateSymbol(),
				packageID:   module.key.packageID,
				kind:        "resource-drop",
				name:        function.Name,
				module:      module.file.Module.Path,
				source:      module.file.Path,
				declaration: resource.Cleanup.Span,
				typeSpan:    resource.Cleanup.Span,
				function:    function,
			}
			a.functions[function] = value
			a.dropFunctions[resource] = function
			a.dropOwners[function] = resource
		}
	}
}

func (a *analyzer) declareModules() {
	for _, file := range a.files {
		if file.Module == nil {
			continue
		}
		key := moduleKey{packageID: file.Package, path: file.Module.Path}
		if previous, found := a.modules[key]; found {
			a.addDiagnostic("E0102", span(file.Path, file.Module.Span), RelatedSpan{
				Role: "previous-declaration",
				Span: span(previous.file.Path, previous.file.Module.Span),
			})
			continue
		}
		module := &moduleScope{
			key:       key,
			file:      file,
			functions: make(map[string]*symbol),
			constants: make(map[string]*symbol),
			structs:   make(map[string]*symbol),
			resources: make(map[string]*symbol),
			enums:     make(map[string]*symbol),
			imports:   make(map[string]*moduleScope),
		}
		a.modules[key] = module
		a.moduleOrder = append(a.moduleOrder, module)
	}
}

func (a *analyzer) declareEnums() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Enums {
			enumeration := &module.file.Module.Enums[index]
			if previous := module.item(enumeration.Name); previous != nil {
				a.addDiagnostic("E0102", span(module.file.Path, enumeration.NameSpan), RelatedSpan{Role: "previous-declaration", Span: span(previous.source, previous.declaration)})
				continue
			}
			value := &symbol{
				id:          a.allocateSymbol(),
				packageID:   module.key.packageID,
				kind:        "enum",
				name:        enumeration.Name,
				module:      module.file.Module.Path,
				source:      module.file.Path,
				declaration: enumeration.NameSpan,
				public:      enumeration.Public,
				enumeration: enumeration,
			}
			module.enums[enumeration.Name] = value
			a.enums[enumeration] = value
		}
	}
}

func (a *analyzer) declareStructs() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Structs {
			structure := &module.file.Module.Structs[index]
			if len(structure.TypeParameters) != 0 {
				a.addUnsupported(module.file.Path, structure.Span, "generic-aggregate-specialization")
			}
			if previous := module.item(structure.Name); previous != nil {
				a.addDiagnostic("E0102", span(module.file.Path, structure.NameSpan), RelatedSpan{Role: "previous-declaration", Span: span(previous.source, previous.declaration)})
				continue
			}
			value := &symbol{
				id:          a.allocateSymbol(),
				packageID:   module.key.packageID,
				kind:        "struct",
				name:        structure.Name,
				module:      module.file.Module.Path,
				source:      module.file.Path,
				declaration: structure.NameSpan,
				public:      structure.Public,
				structure:   structure,
			}
			module.structs[structure.Name] = value
			a.structs[structure] = value
		}
	}
}

func (a *analyzer) declareResources() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			if len(resource.TypeParameters) != 0 {
				a.addUnsupported(module.file.Path, resource.Span, "generic-aggregate-specialization")
			}
			if previous := module.item(resource.Name); previous != nil {
				a.addDiagnostic("E0102", span(module.file.Path, resource.NameSpan), RelatedSpan{
					Role: "previous-declaration",
					Span: span(previous.source, previous.declaration),
				})
				continue
			}
			value := &symbol{
				id:          a.allocateSymbol(),
				packageID:   module.key.packageID,
				kind:        "resource",
				name:        resource.Name,
				module:      module.file.Module.Path,
				source:      module.file.Path,
				declaration: resource.NameSpan,
				public:      resource.Public,
				resource:    resource,
			}
			module.resources[resource.Name] = value
			a.resources[resource] = value
		}
	}
}

func (a *analyzer) declareCallables() {
	for _, module := range a.moduleOrder {
		functionIndex := 0
		foreignIndex := 0
		for functionIndex < len(module.file.Module.Functions) || foreignIndex < len(module.file.Module.ForeignFunctions) {
			if foreignIndex >= len(module.file.Module.ForeignFunctions) ||
				functionIndex < len(module.file.Module.Functions) && module.file.Module.Functions[functionIndex].NameSpan.StartByte < module.file.Module.ForeignFunctions[foreignIndex].NameSpan.StartByte {
				a.declareFunction(module, &module.file.Module.Functions[functionIndex])
				functionIndex++
				continue
			}
			a.declareForeignFunction(module, &module.file.Module.ForeignFunctions[foreignIndex])
			foreignIndex++
		}
	}
}

func (a *analyzer) declareFunction(module *moduleScope, function *ast.Function) {
	name := function.Name
	if function.Owner == "" && a.rejectSourceLocationName(module.file.Path, name, function.NameSpan) {
		return
	}
	if function.Owner != "" {
		name = function.Owner + "::" + function.Name
		if a.ownerSymbol(module, function.Owner) == nil {
			a.addDiagnostic("E0101", span(module.file.Path, function.OwnerSpan))
			return
		}
	}
	if previous := module.item(name); previous != nil {
		code := "E0102"
		if previous.kind == "constant" {
			code = "E4301"
		}
		a.addDiagnostic(code, span(module.file.Path, function.NameSpan), RelatedSpan{
			Role: "previous-declaration",
			Span: span(previous.source, previous.declaration),
		})
		return
	}
	value := &symbol{
		id:          a.allocateSymbol(),
		packageID:   module.key.packageID,
		kind:        "function",
		name:        name,
		module:      module.file.Module.Path,
		source:      module.file.Path,
		declaration: function.NameSpan,
		typeSpan:    function.ResultType.Span,
		public:      function.Public,
		function:    function,
	}
	module.functions[name] = value
	a.functions[function] = value
}

func (a *analyzer) declareForeignFunction(module *moduleScope, function *ast.ForeignFunction) {
	if a.rejectSourceLocationName(module.file.Path, function.Name, function.NameSpan) {
		return
	}
	if previous := module.item(function.Name); previous != nil {
		code := "E0102"
		if previous.kind == "constant" {
			code = "E4301"
		}
		a.addDiagnostic(code, span(module.file.Path, function.NameSpan), RelatedSpan{
			Role: "previous-declaration", Span: span(previous.source, previous.declaration),
		})
		return
	}
	value := &symbol{
		id: a.allocateSymbol(), packageID: module.key.packageID, kind: "foreign-function", name: function.Name,
		module: module.file.Module.Path, source: module.file.Path,
		declaration: function.NameSpan, typeSpan: function.ResultType.Span,
		public: function.Public, foreign: function,
	}
	module.functions[function.Name] = value
	a.foreignFunctions[function] = value
	parameters := make(map[string]diagnostic.Span, len(function.Parameters))
	for parameterIndex := range function.Parameters {
		parameter := &function.Parameters[parameterIndex]
		if a.rejectSourceLocationName(module.file.Path, parameter.Name, parameter.NameSpan) {
			continue
		}
		if previous, duplicate := parameters[parameter.Name]; duplicate {
			a.addDiagnostic("E0102", span(module.file.Path, parameter.NameSpan), RelatedSpan{
				Role: "previous-declaration", Span: span(module.file.Path, previous),
			})
			continue
		}
		parameters[parameter.Name] = parameter.NameSpan
	}
}

func (a *analyzer) resolveImports() {
	for _, module := range a.moduleOrder {
		for _, imported := range module.file.Module.Imports {
			if a.rejectSourceLocationName(module.file.Path, imported.Alias, imported.AliasSpan) {
				continue
			}
			if previous := module.item(imported.Alias); previous != nil {
				a.addDiagnostic("E0102", span(module.file.Path, imported.AliasSpan), RelatedSpan{
					Role: "previous-declaration",
					Span: span(previous.source, previous.declaration),
				})
				continue
			}
			if _, found := module.imports[imported.Alias]; found {
				a.addDiagnostic("E0102", span(module.file.Path, imported.AliasSpan))
				continue
			}
			targetPackage := module.key.packageID
			if a.packageImports != nil {
				bound, exists := a.packageImports[packageImportKey{packageID: module.key.packageID, path: imported.Path}]
				if !exists {
					a.addDiagnostic("E0101", span(module.file.Path, imported.PathSpan))
					continue
				}
				targetPackage = bound
			}
			target, found := a.modules[moduleKey{packageID: targetPackage, path: imported.Path}]
			if !found {
				a.addDiagnostic("E0101", span(module.file.Path, imported.PathSpan))
				continue
			}
			module.imports[imported.Alias] = target
		}
	}
}

func (a *analyzer) resolveFunction(module *moduleScope, function *ast.Function) {
	if len(function.TypeParameters) != 0 {
		a.addUnsupported(module.file.Path, function.Span, "generic-function-specialization")
	}
	parameters := &lexicalScope{names: make(map[string]*symbol)}
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		if a.rejectSourceLocationName(module.file.Path, parameter.Name, parameter.NameSpan) {
			continue
		}
		if previous, found := parameters.names[parameter.Name]; found {
			a.addDiagnostic("E0102", span(module.file.Path, parameter.NameSpan), RelatedSpan{
				Role: "previous-declaration",
				Span: span(previous.source, previous.declaration),
			})
			continue
		}
		value := &symbol{
			id:          a.allocateSymbol(),
			packageID:   module.key.packageID,
			kind:        "parameter",
			name:        parameter.Name,
			module:      module.file.Module.Path,
			source:      module.file.Path,
			declaration: parameter.NameSpan,
			typeSpan:    parameter.Type.Span,
			mutable:     parameter.Mutable,
		}
		parameters.names[parameter.Name] = value
		a.parameter[parameter] = value
		a.parameterIndex[value] = index
	}
	if function.Body != nil {
		a.resolveBlock(module, function.Body, parameters)
	}
}

func (a *analyzer) resolveBlock(module *moduleScope, block *ast.Block, parent *lexicalScope) {
	scope := &lexicalScope{parent: parent, names: make(map[string]*symbol)}
	for index := range block.Statements {
		statement := &block.Statements[index]
		if statement.Binding != nil {
			a.resolveBinding(module, scope, statement.Binding)
			continue
		}
		if statement.For != nil {
			a.resolveFor(module, scope, statement.For)
			continue
		}
		resetArrayForPreparation(statement)
		if statement.ByteFor != nil && statement.Expression == nil {
			statement.Expression = ast.PrepareIntegerReferenceLoop(statement.ByteFor)
		}
		a.resolveExpression(module, scope, statement.Target)
		a.resolveExpression(module, scope, statement.Expression)
	}
	a.resolveExpression(module, scope, block.Tail)
}

func (a *analyzer) resolveBinding(module *moduleScope, scope *lexicalScope, binding *ast.Binding) {
	if binding == nil {
		return
	}
	a.resolveExpression(module, scope, binding.Expression)
	if a.rejectSourceLocationName(module.file.Path, binding.Name, binding.NameSpan) {
		return
	}
	if previous, found := scope.names[binding.Name]; found {
		a.addDiagnostic("E0102", span(module.file.Path, binding.NameSpan), RelatedSpan{
			Role: "previous-declaration",
			Span: span(previous.source, previous.declaration),
		})
		return
	}
	value := &symbol{
		id:          a.allocateSymbol(),
		packageID:   module.key.packageID,
		kind:        "local",
		name:        binding.Name,
		module:      module.file.Module.Path,
		source:      module.file.Path,
		declaration: binding.NameSpan,
		mutable:     binding.Mutable,
	}
	if binding.Type != nil {
		value.typeSpan = binding.Type.Span
	}
	scope.names[binding.Name] = value
	a.bindingSymbols[binding] = value
}

func (a *analyzer) resolveFor(module *moduleScope, parent *lexicalScope, loop *ast.ForLoop) {
	if loop == nil {
		return
	}
	scope := &lexicalScope{parent: parent, names: make(map[string]*symbol)}
	if loop.Initializer != nil {
		if loop.Initializer.Binding != nil {
			a.resolveBinding(module, scope, loop.Initializer.Binding)
		} else {
			a.resolveExpression(module, scope, loop.Initializer.Target)
			a.resolveExpression(module, scope, loop.Initializer.Expression)
		}
	}
	a.resolveExpression(module, scope, loop.Condition)
	if loop.Post != nil {
		a.resolveExpression(module, scope, loop.Post.Target)
		a.resolveExpression(module, scope, loop.Post.Expression)
	}
	a.resolveBlock(module, loop.Body, scope)
}

func (a *analyzer) resolveExpression(module *moduleScope, scope *lexicalScope, expression *ast.Expression) {
	if expression == nil {
		return
	}
	if expression.Kind == syntax.KindSwitch {
		a.resolveMatch(module, scope, expression)
		return
	}
	if expression.Kind == syntax.KindRegion {
		a.resolveRegion(module, scope, expression)
		return
	}
	if expression.Kind == syntax.KindPath && !strings.HasPrefix(expression.Text, ".") && !isCompilerBuiltinPath(expression.Text) {
		a.resolvePath(module, scope, expression)
	}
	if expression.Block != nil {
		a.resolveBlock(module, expression.Block, scope)
	}
	for _, child := range expression.Children {
		a.resolveExpression(module, scope, child)
	}
}

func (a *analyzer) resolveMatch(module *moduleScope, scope *lexicalScope, expression *ast.Expression) {
	if len(expression.Children) == 0 {
		return
	}
	a.resolveExpression(module, scope, expression.Children[0])
	for _, arm := range expression.Children[1:] {
		if arm == nil || arm.Kind != syntax.KindPattern || arm.Text != "arm" || len(arm.Children) != 2 {
			a.resolveExpression(module, scope, arm)
			continue
		}
		pattern := arm.Children[0]
		body := arm.Children[1]
		armScope := &lexicalScope{parent: scope, names: make(map[string]*symbol)}
		if pattern != nil && pattern.Kind == syntax.KindPattern {
			if strings.HasPrefix(pattern.Text, ".") && !isBuiltinVariantPath(pattern.Text) {
				for _, field := range pattern.Children[1:] {
					if field == nil || field.Kind != syntax.KindField || field.Cooked == "" {
						continue
					}
					if a.rejectSourceLocationName(module.file.Path, field.Cooked, field.Span) {
						continue
					}
					if previous := armScope.names[field.Cooked]; previous != nil {
						a.addDiagnostic("E0102", span(module.file.Path, field.Span), RelatedSpan{
							Role: "previous-declaration", Span: span(previous.source, previous.declaration),
						})
						continue
					}
					value := &symbol{
						id: a.allocateSymbol(), packageID: module.key.packageID, kind: "match-binding", name: field.Cooked,
						module: module.file.Module.Path, source: module.file.Path, declaration: field.Span,
					}
					armScope.names[field.Cooked] = value
					a.resolved[field] = value
				}
				a.resolveExpression(module, armScope, body)
				continue
			}
		}
		if pattern == nil || pattern.Kind != syntax.KindPattern || !isBuiltinVariantPath(pattern.Text) {
			a.resolveExpression(module, armScope, pattern)
			a.resolveExpression(module, armScope, body)
			continue
		}
		if variant, option := optionVariantPath(pattern.Text); option && variant == optionVariantNone && len(pattern.Children) == 1 {
			a.resolveExpression(module, armScope, body)
			continue
		}
		if len(pattern.Children) != 2 {
			a.resolveExpression(module, armScope, body)
			continue
		}
		binding := pattern.Children[1]
		if binding == nil || binding.Kind != syntax.KindPath || binding.Text == "" {
			a.resolveExpression(module, armScope, body)
			continue
		}
		if a.rejectSourceLocationName(module.file.Path, binding.Text, binding.Span) {
			a.resolveExpression(module, armScope, body)
			continue
		}
		value := &symbol{
			id:          a.allocateSymbol(),
			packageID:   module.key.packageID,
			kind:        "match-binding",
			name:        binding.Text,
			module:      module.file.Module.Path,
			source:      module.file.Path,
			declaration: binding.Span,
		}
		armScope.names[binding.Text] = value
		a.resolved[binding] = value
		a.resolveExpression(module, armScope, body)
	}
}

func (a *analyzer) resolveRegion(module *moduleScope, scope *lexicalScope, expression *ast.Expression) {
	if len(expression.Children) != 2 {
		return
	}
	a.resolveExpression(module, scope, expression.Children[0])
	if a.rejectSourceLocationName(module.file.Path, expression.Text, expression.Span) {
		return
	}
	regionScope := &lexicalScope{parent: scope, names: make(map[string]*symbol)}
	value := &symbol{id: a.allocateSymbol(), packageID: module.key.packageID, kind: "region", name: expression.Text, module: module.file.Module.Path, source: module.file.Path, declaration: expression.Span, typ: hir.TypeRegion}
	regionScope.names[expression.Text] = value
	a.regions[expression] = value
	body := expression.Children[1]
	if body.Block != nil {
		a.resolveBlock(module, body.Block, regionScope)
	}
}

func (a *analyzer) resolvePath(module *moduleScope, scope *lexicalScope, expression *ast.Expression) {
	parts := strings.Split(expression.Text, "::")
	var target *symbol
	if len(parts) == 1 {
		if parts[0] == "Region" {
			target = a.regionType
		} else {
			target = lookupLexical(scope, parts[0])
			if target == nil {
				target = module.item(parts[0])
			}
		}
	} else if len(parts) == 2 {
		if local := module.functions[expression.Text]; local != nil {
			target = local
		} else if imported := module.imports[parts[0]]; imported != nil {
			target = imported.item(parts[1])
			if target != nil && imported != module && !target.public {
				a.addDiagnostic("E0103", span(module.file.Path, expression.Span), RelatedSpan{
					Role: "declaration",
					Span: span(target.source, target.declaration),
				})
				return
			}
		}
	} else if len(parts) == 3 {
		if imported := module.imports[parts[0]]; imported != nil {
			target = imported.functions[parts[1]+"::"+parts[2]]
			if target != nil && imported != module && !target.public {
				a.addDiagnostic("E0103", span(module.file.Path, expression.Span), RelatedSpan{
					Role: "declaration", Span: span(target.source, target.declaration),
				})
				return
			}
		}
	} else {
		a.addUnsupported(module.file.Path, expression.Span, "qualified paths deeper than one import alias")
		return
	}
	if target == nil {
		a.addDiagnostic("E0101", span(module.file.Path, expression.Span))
		return
	}
	a.resolved[expression] = target
	if target == a.regionType {
		return
	}
	a.resolutions = append(a.resolutions, Resolution{
		Use: span(module.file.Path, expression.Span),
		Target: SymbolCoordinate{
			Package:     target.packageID,
			Module:      target.module,
			Name:        target.name,
			Kind:        target.kind,
			Declaration: span(target.source, target.declaration),
		},
	})
}

func (a *analyzer) ownerSymbol(module *moduleScope, name string) *symbol {
	if module == nil {
		return nil
	}
	if value := module.structs[name]; value != nil {
		return value
	}
	if value := module.resources[name]; value != nil {
		return value
	}
	return module.enums[name]
}

func (m *moduleScope) item(name string) *symbol {
	if value := m.constants[name]; value != nil {
		return value
	}
	if value := m.functions[name]; value != nil {
		return value
	}
	if value := m.structs[name]; value != nil {
		return value
	}
	if value := m.enums[name]; value != nil {
		return value
	}
	return m.resources[name]
}

func (a *analyzer) symbolModule(value *symbol) *moduleScope {
	if value == nil {
		return nil
	}
	return a.modules[moduleKey{packageID: value.packageID, path: value.module}]
}

func lookupLexical(scope *lexicalScope, name string) *symbol {
	for current := scope; current != nil; current = current.parent {
		if value := current.names[name]; value != nil {
			return value
		}
	}
	return nil
}

func (a *analyzer) allocateSymbol() hir.SymbolID {
	value := a.nextSymbol
	a.nextSymbol++
	return value
}

func (a *analyzer) addDiagnostic(code string, primary hir.Span, related ...RelatedSpan) {
	a.diagnostics = append(a.diagnostics, Diagnostic{Code: code, Primary: primary, Related: related})
}

func (a *analyzer) addUnsupported(source string, value diagnostic.Span, feature string) {
	a.unsupported = append(a.unsupported, Unsupported{Feature: feature, Span: span(source, value)})
}

func span(source string, value diagnostic.Span) hir.Span {
	return sourceSpan(source, value.StartByte, value.EndByte)
}

func optionalSpan(source string, value diagnostic.Span) hir.Span {
	if value.EndByte <= value.StartByte {
		return hir.Span{}
	}
	return span(source, value)
}

func sourceSpan(source string, start, end int) hir.Span {
	packageID, sourcePath := hir.SourcePathParts(source)
	return hir.Span{Package: packageID, Source: sourcePath, StartByte: start, EndByte: end}
}

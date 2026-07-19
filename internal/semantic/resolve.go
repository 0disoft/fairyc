package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type symbol struct {
	id          hir.SymbolID
	kind        string
	name        string
	module      string
	source      string
	declaration diagnostic.Span
	typeSpan    diagnostic.Span
	resultError diagnostic.Span
	typ         hir.Type
	mutable     bool
	public      bool
	function    *ast.Function
	foreign     *ast.ForeignFunction
	structure   *ast.Struct
	resource    *ast.Resource
	enumeration *ast.Enum
}

type moduleScope struct {
	file      File
	functions map[string]*symbol
	structs   map[string]*symbol
	resources map[string]*symbol
	enums     map[string]*symbol
	imports   map[string]*moduleScope
}

type lexicalScope struct {
	parent *lexicalScope
	names  map[string]*symbol
}

type analyzer struct {
	files            []File
	modules          map[string]*moduleScope
	moduleOrder      []*moduleScope
	nextSymbol       hir.SymbolID
	resolved         map[*ast.Expression]*symbol
	bindingSymbols   map[*ast.Binding]*symbol
	parameter        map[*ast.Parameter]*symbol
	parameterIndex   map[*symbol]int
	functions        map[*ast.Function]*symbol
	foreignFunctions map[*ast.ForeignFunction]*symbol
	structs          map[*ast.Struct]*symbol
	resources        map[*ast.Resource]*symbol
	enums            map[*ast.Enum]*symbol
	regions          map[*ast.Expression]*symbol
	regionType       *symbol
	diagnostics      []Diagnostic
	resolutions      []Resolution
	types            []TypeFact
	borrows          []BorrowFact
	unsupported      []Unsupported
	flowStarted      bool
	resourceTypes    map[hir.Type]*resourceInfo
	structTypes      map[hir.Type]*structInfo
	enumTypes        map[hir.Type]*enumInfo
	resourceStates   []ResourceStateFact
	expressionTypes  map[*ast.Expression]hir.Type
	regionOrigins    []RegionOriginFact
	resultHandling   []ResultHandlingFact
	resourceLoops    []resourceLoopFrame
	resultLoops      []resultLoopFrame
}

func newAnalyzer(files []File) *analyzer {
	return &analyzer{
		files:            files,
		modules:          make(map[string]*moduleScope, len(files)),
		resolved:         make(map[*ast.Expression]*symbol),
		bindingSymbols:   make(map[*ast.Binding]*symbol),
		parameter:        make(map[*ast.Parameter]*symbol),
		parameterIndex:   make(map[*symbol]int),
		functions:        make(map[*ast.Function]*symbol),
		foreignFunctions: make(map[*ast.ForeignFunction]*symbol),
		structs:          make(map[*ast.Struct]*symbol),
		resources:        make(map[*ast.Resource]*symbol),
		enums:            make(map[*ast.Enum]*symbol),
		regions:          make(map[*ast.Expression]*symbol),
		resourceTypes:    make(map[hir.Type]*resourceInfo),
		structTypes:      make(map[hir.Type]*structInfo),
		enumTypes:        make(map[hir.Type]*enumInfo),
		expressionTypes:  make(map[*ast.Expression]hir.Type),
		regionType:       &symbol{id: hir.NoSymbol, kind: "capability", name: "Region", typ: hir.TypeRegion},
	}
}

func (a *analyzer) resolve() {
	a.declareModules()
	a.declareEnums()
	a.declareStructs()
	a.declareResources()
	a.declareCallables()
	a.resolveImports()
	if len(a.diagnostics) > 0 {
		return
	}
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Functions {
			a.resolveFunction(module, &module.file.Module.Functions[index])
		}
	}
}

func (a *analyzer) declareModules() {
	for _, file := range a.files {
		if file.Module == nil {
			continue
		}
		if previous, found := a.modules[file.Module.Path]; found {
			a.addDiagnostic("E0102", span(file.Path, file.Module.Span), RelatedSpan{
				Role: "previous-declaration",
				Span: span(previous.file.Path, previous.file.Module.Span),
			})
			continue
		}
		module := &moduleScope{
			file:      file,
			functions: make(map[string]*symbol),
			structs:   make(map[string]*symbol),
			resources: make(map[string]*symbol),
			enums:     make(map[string]*symbol),
			imports:   make(map[string]*moduleScope),
		}
		a.modules[file.Module.Path] = module
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
			if previous := module.item(structure.Name); previous != nil {
				a.addDiagnostic("E0102", span(module.file.Path, structure.NameSpan), RelatedSpan{Role: "previous-declaration", Span: span(previous.source, previous.declaration)})
				continue
			}
			value := &symbol{
				id:          a.allocateSymbol(),
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
			if previous := module.item(resource.Name); previous != nil {
				a.addDiagnostic("E0102", span(module.file.Path, resource.NameSpan), RelatedSpan{
					Role: "previous-declaration",
					Span: span(previous.source, previous.declaration),
				})
				continue
			}
			value := &symbol{
				id:          a.allocateSymbol(),
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
	if previous := module.item(function.Name); previous != nil {
		a.addDiagnostic("E0102", span(module.file.Path, function.NameSpan), RelatedSpan{
			Role: "previous-declaration",
			Span: span(previous.source, previous.declaration),
		})
		return
	}
	value := &symbol{
		id:          a.allocateSymbol(),
		kind:        "function",
		name:        function.Name,
		module:      module.file.Module.Path,
		source:      module.file.Path,
		declaration: function.NameSpan,
		typeSpan:    function.ResultType.Span,
		public:      function.Public,
		function:    function,
	}
	module.functions[function.Name] = value
	a.functions[function] = value
}

func (a *analyzer) declareForeignFunction(module *moduleScope, function *ast.ForeignFunction) {
	if previous := module.item(function.Name); previous != nil {
		a.addDiagnostic("E0102", span(module.file.Path, function.NameSpan), RelatedSpan{
			Role: "previous-declaration", Span: span(previous.source, previous.declaration),
		})
		return
	}
	value := &symbol{
		id: a.allocateSymbol(), kind: "foreign-function", name: function.Name,
		module: module.file.Module.Path, source: module.file.Path,
		declaration: function.NameSpan, typeSpan: function.ResultType.Span,
		public: function.Public, foreign: function,
	}
	module.functions[function.Name] = value
	a.foreignFunctions[function] = value
	parameters := make(map[string]diagnostic.Span, len(function.Parameters))
	for parameterIndex := range function.Parameters {
		parameter := &function.Parameters[parameterIndex]
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
			target, found := a.modules[imported.Path]
			if !found {
				a.addDiagnostic("E0101", span(module.file.Path, imported.PathSpan))
				continue
			}
			module.imports[imported.Alias] = target
		}
	}
}

func (a *analyzer) resolveFunction(module *moduleScope, function *ast.Function) {
	parameters := &lexicalScope{names: make(map[string]*symbol)}
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		if previous, found := parameters.names[parameter.Name]; found {
			a.addDiagnostic("E0102", span(module.file.Path, parameter.NameSpan), RelatedSpan{
				Role: "previous-declaration",
				Span: span(previous.source, previous.declaration),
			})
			continue
		}
		value := &symbol{
			id:          a.allocateSymbol(),
			kind:        "parameter",
			name:        parameter.Name,
			module:      module.file.Module.Path,
			source:      module.file.Path,
			declaration: parameter.NameSpan,
			typeSpan:    parameter.Type.Span,
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
			a.resolveExpression(module, scope, statement.Binding.Expression)
			binding := statement.Binding
			if previous, found := scope.names[binding.Name]; found {
				a.addDiagnostic("E0102", span(module.file.Path, binding.NameSpan), RelatedSpan{
					Role: "previous-declaration",
					Span: span(previous.source, previous.declaration),
				})
				continue
			}
			value := &symbol{
				id:          a.allocateSymbol(),
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
			continue
		}
		a.resolveExpression(module, scope, statement.Target)
		a.resolveExpression(module, scope, statement.Expression)
	}
	a.resolveExpression(module, scope, block.Tail)
}

func (a *analyzer) resolveExpression(module *moduleScope, scope *lexicalScope, expression *ast.Expression) {
	if expression == nil {
		return
	}
	if expression.Kind == syntax.KindMatch {
		a.resolveMatch(module, scope, expression)
		return
	}
	if expression.Kind == syntax.KindRegion {
		a.resolveRegion(module, scope, expression)
		return
	}
	if expression.Kind == syntax.KindPath && !isBuiltinVariantPath(expression.Text) {
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
		value := &symbol{
			id:          a.allocateSymbol(),
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
	regionScope := &lexicalScope{parent: scope, names: make(map[string]*symbol)}
	value := &symbol{id: a.allocateSymbol(), kind: "region", name: expression.Text, module: module.file.Module.Path, source: module.file.Path, declaration: expression.Span, typ: hir.TypeRegion}
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
		if imported := module.imports[parts[0]]; imported != nil {
			target = imported.item(parts[1])
			if target != nil && imported != module && !target.public {
				a.addDiagnostic("E0103", span(module.file.Path, expression.Span), RelatedSpan{
					Role: "declaration",
					Span: span(target.source, target.declaration),
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
			Module:      target.module,
			Name:        target.name,
			Kind:        target.kind,
			Declaration: span(target.source, target.declaration),
		},
	})
}

func (m *moduleScope) item(name string) *symbol {
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
	return hir.Span{Source: source, StartByte: value.StartByte, EndByte: value.EndByte}
}

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
	mutable      bool
	public       bool
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

type enumVariantReference struct {
	enumeration *symbol
	variant     *ast.EnumVariant
}

type analyzer struct {
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
	foreignFunctions      map[*ast.ForeignFunction]*symbol
	structs               map[*ast.Struct]*symbol
	resources             map[*ast.Resource]*symbol
	enums                 map[*ast.Enum]*symbol
	enumVariantPaths      map[*ast.Expression]enumVariantReference
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
	borrowReturns         map[*symbol]borrowReturnSummary
	dropFunctions         map[*ast.Resource]*ast.Function
	dropOwners            map[*ast.Function]*ast.Resource
}

func newAnalyzer(files []File, packageImports map[packageImportKey]hir.PackageID) *analyzer {
	return &analyzer{
		files:            files,
		modules:          make(map[moduleKey]*moduleScope, len(files)),
		packageImports:   packageImports,
		resolved:         make(map[*ast.Expression]*symbol),
		bindingSymbols:   make(map[*ast.Binding]*symbol),
		parameter:        make(map[*ast.Parameter]*symbol),
		parameterIndex:   make(map[*symbol]int),
		functions:        make(map[*ast.Function]*symbol),
		foreignFunctions: make(map[*ast.ForeignFunction]*symbol),
		structs:          make(map[*ast.Struct]*symbol),
		resources:        make(map[*ast.Resource]*symbol),
		enums:            make(map[*ast.Enum]*symbol),
		enumVariantPaths: make(map[*ast.Expression]enumVariantReference),
		regions:          make(map[*ast.Expression]*symbol),
		resourceTypes:    make(map[hir.Type]*resourceInfo),
		structTypes:      make(map[hir.Type]*structInfo),
		enumTypes:        make(map[hir.Type]*enumInfo),
		expressionTypes:  make(map[*ast.Expression]hir.Type),
		borrowReturns:    make(map[*symbol]borrowReturnSummary),
		dropFunctions:    make(map[*ast.Resource]*ast.Function),
		dropOwners:       make(map[*ast.Function]*ast.Resource),
		regionType:       &symbol{id: hir.NoSymbol, kind: "capability", name: "Region", typ: hir.TypeRegion},
	}
}

func (a *analyzer) resolve() {
	a.declareModules()
	a.declareEnums()
	a.declareStructs()
	a.declareResources()
	a.declareCallables()
	a.declareResourceDropFunctions()
	a.resolveImports()
	if len(a.diagnostics) > 0 {
		return
	}
	for _, module := range a.moduleOrder {
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

func (a *analyzer) declareResourceDropFunctions() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			if a.resources[resource] == nil || resource.Drop == nil || resource.Drop.Body == nil ||
				len(resource.Drop.Body.Statements) == 0 && resource.Drop.Body.Tail == nil {
				continue
			}
			function := &ast.Function{
				Name:     "$drop$" + resource.Name,
				Span:     resource.Drop.Span,
				NameSpan: resource.Drop.Span,
				Parameters: []ast.Parameter{{
					Name:     "self",
					Mutable:  true,
					Span:     resource.Drop.Span,
					NameSpan: resource.Drop.Span,
					Type:     ast.TypeReference{Text: resource.Name, Span: resource.NameSpan},
				}},
				ResultType: ast.TypeReference{Text: "void", Span: resource.Drop.Span},
				Body:       resource.Drop.Body,
			}
			value := &symbol{
				id:          a.allocateSymbol(),
				packageID:   module.key.packageID,
				kind:        "resource-drop",
				name:        function.Name,
				module:      module.file.Module.Path,
				source:      module.file.Path,
				declaration: resource.Drop.Span,
				typeSpan:    resource.Drop.Span,
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
	if previous := module.item(function.Name); previous != nil {
		a.addDiagnostic("E0102", span(module.file.Path, function.NameSpan), RelatedSpan{
			Role: "previous-declaration",
			Span: span(previous.source, previous.declaration),
		})
		return
	}
	value := &symbol{
		id:          a.allocateSymbol(),
		packageID:   module.key.packageID,
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
		if _, recognized := a.resolveEnumVariantPath(module, expression); !recognized {
			a.resolvePath(module, scope, expression)
		}
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
			if reference, recognized := a.resolveEnumVariantPath(module, pattern); recognized {
				if reference.variant != nil {
					for _, field := range pattern.Children[1:] {
						if field == nil || field.Kind != syntax.KindField || field.Cooked == "" {
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

func (a *analyzer) resolveEnumVariantPath(module *moduleScope, expression *ast.Expression) (enumVariantReference, bool) {
	if expression == nil {
		return enumVariantReference{}, false
	}
	parts := strings.Split(expression.Text, "::")
	var owner *symbol
	switch len(parts) {
	case 2:
		owner = module.enums[parts[0]]
	case 3:
		imported := module.imports[parts[0]]
		if imported == nil {
			return enumVariantReference{}, false
		}
		owner = imported.enums[parts[1]]
		if owner != nil && imported != module && !owner.public {
			a.addDiagnostic("E0103", span(module.file.Path, expression.Span), RelatedSpan{
				Role: "declaration", Span: span(owner.source, owner.declaration),
			})
			return enumVariantReference{}, true
		}
	default:
		return enumVariantReference{}, false
	}
	if owner == nil || owner.enumeration == nil {
		return enumVariantReference{}, false
	}
	variantName := parts[len(parts)-1]
	for index := range owner.enumeration.Variants {
		variant := &owner.enumeration.Variants[index]
		if variant.Name != variantName {
			continue
		}
		reference := enumVariantReference{enumeration: owner, variant: variant}
		a.enumVariantPaths[expression] = reference
		a.resolutions = append(a.resolutions, Resolution{
			Use: span(module.file.Path, expression.Span),
			Target: SymbolCoordinate{
				Package: owner.packageID, Module: owner.module, Name: owner.name + "::" + variant.Name, Kind: "enum-variant",
				Declaration: span(owner.source, variant.NameSpan),
			},
		})
		return reference, true
	}
	a.addDiagnostic("E0101", span(module.file.Path, expression.Span), RelatedSpan{
		Role: "enum-declaration", Span: span(owner.source, owner.declaration),
	})
	return enumVariantReference{}, true
}

func (a *analyzer) resolveRegion(module *moduleScope, scope *lexicalScope, expression *ast.Expression) {
	if len(expression.Children) != 2 {
		return
	}
	a.resolveExpression(module, scope, expression.Children[0])
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
			Package:     target.packageID,
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
	return hir.Span{Source: source, StartByte: value.StartByte, EndByte: value.EndByte}
}

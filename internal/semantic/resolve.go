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
	typ         hir.Type
	mutable     bool
	public      bool
	function    *ast.Function
	resource    *ast.Resource
}

type moduleScope struct {
	file      File
	functions map[string]*symbol
	resources map[string]*symbol
	imports   map[string]*moduleScope
}

type lexicalScope struct {
	parent *lexicalScope
	names  map[string]*symbol
}

type analyzer struct {
	files          []File
	modules        map[string]*moduleScope
	moduleOrder    []*moduleScope
	nextSymbol     hir.SymbolID
	resolved       map[*ast.Expression]*symbol
	bindingSymbols map[*ast.Binding]*symbol
	parameter      map[*ast.Parameter]*symbol
	parameterIndex map[*symbol]int
	functions      map[*ast.Function]*symbol
	resources      map[*ast.Resource]*symbol
	diagnostics    []Diagnostic
	resolutions    []Resolution
	types          []TypeFact
	borrows        []BorrowFact
	unsupported    []Unsupported
	flowStarted    bool
	resourceTypes  map[hir.Type]*resourceInfo
	resourceStates []ResourceStateFact
}

func newAnalyzer(files []File) *analyzer {
	return &analyzer{
		files:          files,
		modules:        make(map[string]*moduleScope, len(files)),
		resolved:       make(map[*ast.Expression]*symbol),
		bindingSymbols: make(map[*ast.Binding]*symbol),
		parameter:      make(map[*ast.Parameter]*symbol),
		parameterIndex: make(map[*symbol]int),
		functions:      make(map[*ast.Function]*symbol),
		resources:      make(map[*ast.Resource]*symbol),
		resourceTypes:  make(map[hir.Type]*resourceInfo),
	}
}

func (a *analyzer) resolve() {
	a.declareModules()
	a.declareResources()
	a.declareFunctions()
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
			resources: make(map[string]*symbol),
			imports:   make(map[string]*moduleScope),
		}
		a.modules[file.Module.Path] = module
		a.moduleOrder = append(a.moduleOrder, module)
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

func (a *analyzer) declareFunctions() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if previous := module.item(function.Name); previous != nil {
				a.addDiagnostic("E0102", span(module.file.Path, function.NameSpan), RelatedSpan{
					Role: "previous-declaration",
					Span: span(previous.source, previous.declaration),
				})
				continue
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
	if expression.Kind == syntax.KindPath {
		a.resolvePath(module, scope, expression)
	}
	if expression.Block != nil {
		a.resolveBlock(module, expression.Block, scope)
	}
	for _, child := range expression.Children {
		a.resolveExpression(module, scope, child)
	}
}

func (a *analyzer) resolvePath(module *moduleScope, scope *lexicalScope, expression *ast.Expression) {
	parts := strings.Split(expression.Text, "::")
	var target *symbol
	if len(parts) == 1 {
		target = lookupLexical(scope, parts[0])
		if target == nil {
			target = module.item(parts[0])
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

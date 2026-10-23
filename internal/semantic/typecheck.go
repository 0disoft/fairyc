package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
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
	a.prepareCEnumBases()
	a.prepareResourceTypes()
	a.prepareStructTypes()
	a.prepareEnumTypes()
	a.prepareCallableMustUse()
	a.prepareCallableEffects()
	a.prepareCompileTime()
	result.HIR.StateLayouts = a.checkStateLayouts()
	for _, module := range a.moduleOrder {
		if a.canceled() {
			return Result{}
		}
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
			if a.canceled() {
				return Result{}
			}
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

package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

// Lower normalizes typed HIR into executable mutable-local MIR.
func Lower(input hir.Package) (Program, error) {
	structTypes, err := lowerStructTypes(input)
	if err != nil {
		return Program{}, err
	}
	enumTypes, err := lowerEnumTypes(input)
	if err != nil {
		return Program{}, err
	}
	resourceTypes, err := lowerResourceTypes(input)
	if err != nil {
		return Program{}, err
	}
	foreignFunctions, err := lowerForeignFunctions(input)
	if err != nil {
		return Program{}, err
	}
	program := Program{StructTypes: structTypes, EnumTypes: enumTypes, ResourceTypes: resourceTypes, ForeignFunctions: foreignFunctions}
	properties := newTypeProperties(resourceTypes, structTypes, enumTypes)
	for _, module := range input.Modules {
		moduleIdentity := hir.QualifiedModulePath(module.Package, module.Path)
		for index := range module.Functions {
			function := &module.Functions[index]
			lowerer := newFunctionLowerer(moduleIdentity, function, properties)
			lowered, err := lowerer.lower()
			if err != nil {
				return Program{}, fmt.Errorf("lower %s::%s: %w", moduleIdentity, function.Name, err)
			}
			program.Functions = append(program.Functions, lowered)
		}
	}
	slices.SortFunc(program.Functions, func(left, right Function) int {
		if left.Name < right.Name {
			return -1
		}
		if left.Name > right.Name {
			return 1
		}
		return 0
	})
	return program, nil
}

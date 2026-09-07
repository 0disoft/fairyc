package compiler

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericsource"
)

type genericHeader struct {
	span, nameSpan, documentation diagnostic.Span
	parameters                    []string
	public                        bool
	signature                     GenericSignature
}

func originalGenericHeader(module *ast.Module, origin genericsource.DeclarationOrigin) (genericHeader, error) {
	var candidates []genericHeader
	switch origin.Kind {
	case genericsource.OriginFunction:
		for _, fn := range module.Functions {
			if fn.Name != origin.Name {
				continue
			}
			if fn.Owner != "" {
				return genericHeader{}, fmt.Errorf("generic declaration cannot be a method")
			}
			candidates = append(candidates, genericHeader{fn.Span, fn.NameSpan, fn.DocumentationSpan, fn.TypeParameters, fn.Public,
				GenericSignature{Attributes: fn.Attributes, Arguments: fn.Parameters, Result: fn.ResultType, Unsafe: fn.Unsafe}})
		}
	case genericsource.OriginStruct:
		for _, structure := range module.Structs {
			if structure.Name == origin.Name {
				candidates = append(candidates, genericHeader{structure.Span, structure.NameSpan, structure.DocumentationSpan, structure.TypeParameters, structure.Public,
					GenericSignature{Attributes: structure.Attributes, Fields: structure.Fields}})
			}
		}
	case genericsource.OriginResource:
		for _, resource := range module.Resources {
			if resource.Name == origin.Name {
				candidates = append(candidates, genericHeader{resource.Span, resource.NameSpan, resource.DocumentationSpan, resource.TypeParameters, resource.Public,
					GenericSignature{Attributes: resource.Attributes, Fields: resource.Fields, HasCleanup: resource.Cleanup != nil}})
			}
		}
	default:
		return genericHeader{}, fmt.Errorf("generic declaration has an unsupported kind %q", origin.Kind)
	}
	if len(candidates) != 1 {
		return genericHeader{}, fmt.Errorf("generic declaration has no unique original header")
	}
	header := candidates[0]
	if header.span != origin.Span || header.nameSpan != origin.NameSpan || header.public != origin.Public || !slices.Equal(header.parameters, origin.Parameters) {
		return genericHeader{}, fmt.Errorf("generic declaration does not match its original header")
	}
	return header, nil
}

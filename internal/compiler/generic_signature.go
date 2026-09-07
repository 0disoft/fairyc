package compiler

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
)

// GenericSignature retains written types before specialization erases binder references.
// These are source-declared syntax, not resolved semantic types or proof that an unused
// template body is valid. Consumers must resolve names in the original package/module.
// No body is retained, and getters detach every mutable type and attribute node.
type GenericSignature struct {
	Fields     []ast.FieldDeclaration
	HasCleanup bool
	Attributes []ast.Attribute
	Arguments  []ast.Parameter
	Result     ast.TypeReference
	Unsafe     bool
}

func cloneGenericSignature(value GenericSignature) GenericSignature {
	value.Fields = slices.Clone(value.Fields)
	for i := range value.Fields {
		value.Fields[i].Type = cloneGenericType(value.Fields[i].Type)
	}
	value.Attributes = slices.Clone(value.Attributes)
	for i := range value.Attributes {
		value.Attributes[i].Arguments = slices.Clone(value.Attributes[i].Arguments)
	}
	value.Arguments = slices.Clone(value.Arguments)
	for i := range value.Arguments {
		value.Arguments[i].Type = cloneGenericType(value.Arguments[i].Type)
	}
	value.Result = cloneGenericType(value.Result)
	return value
}

func cloneGenericType(value ast.TypeReference) ast.TypeReference {
	clonePointer := func(input *ast.TypeReference) *ast.TypeReference {
		if input == nil {
			return nil
		}
		result := cloneGenericType(*input)
		return &result
	}
	value.Element = clonePointer(value.Element)
	value.Result = clonePointer(value.Result)
	value.Arguments = slices.Clone(value.Arguments)
	for i := range value.Arguments {
		value.Arguments[i] = cloneGenericType(value.Arguments[i])
	}
	value.Parameters = slices.Clone(value.Parameters)
	for i := range value.Parameters {
		value.Parameters[i] = cloneGenericType(value.Parameters[i])
	}
	if value.Length != nil {
		length := *value.Length
		value.Length = &length
	}
	return value
}

package languageserver

import (
	"iter"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

type authoredName struct {
	span diagnostic.Span
	kind tokenKind
}

func authoredDeclarations(module *ast.Module) iter.Seq[authoredName] {
	return func(yield func(authoredName) bool) {
		emit := func(span diagnostic.Span, kind tokenKind) bool {
			return yield(authoredName{span, kind})
		}
		fields := func(items []ast.FieldDeclaration) bool {
			for _, item := range items {
				if !emit(item.NameSpan, tokenProperty) {
					return false
				}
			}
			return true
		}
		function := func(span diagnostic.Span, kind tokenKind, parameters []ast.Parameter) bool {
			if !emit(span, kind) {
				return false
			}
			for _, parameter := range parameters {
				if !emit(parameter.NameSpan, tokenParameter) {
					return false
				}
			}
			return true
		}
		for _, item := range module.Imports {
			if !emit(item.AliasSpan, tokenNamespace) {
				return
			}
		}
		for _, item := range module.Structs {
			if !emit(item.NameSpan, tokenStruct) || !fields(item.Fields) {
				return
			}
		}
		for _, item := range module.Resources {
			if !emit(item.NameSpan, tokenType) || !fields(item.Fields) {
				return
			}
		}
		for _, item := range module.Enums {
			if !emit(item.NameSpan, tokenEnum) {
				return
			}
			for _, variant := range item.Variants {
				if !emit(variant.NameSpan, tokenEnumMember) || !fields(variant.Fields) {
					return
				}
			}
		}
		for _, state := range module.States {
			for _, field := range state.Fields {
				if !emit(field.NameSpan, tokenProperty) {
					return
				}
			}
		}
		for _, fn := range module.Functions {
			if !function(fn.NameSpan, tokenFunction, fn.Parameters) {
				return
			}
		}
		for _, fn := range module.ForeignFunctions {
			if !function(fn.NameSpan, tokenFunction, fn.Parameters) {
				return
			}
		}
		for _, binding := range module.Bindings {
			if !emit(binding.NameSpan, tokenVariable) {
				return
			}
		}
		for _, constant := range module.Constants {
			if !emit(constant.NameSpan, tokenConstant) {
				return
			}
		}
	}
}

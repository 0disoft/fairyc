package semantic

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
)

const testAttribute = "test"

// TestDeclarations is a read-only index of harness entries and declaration
// rejections. Indices refer to the caller's original module function slice.
type TestDeclarations struct {
	Functions   []int
	Diagnostics []diagnostic.Diagnostic
}

// InspectTestDeclarations owns @test shape validation. Callers select test
// source explicitly; source paths or module names never grant test admission.
// The returned metadata is detached and the module is never modified.
func InspectTestDeclarations(module *ast.Module) TestDeclarations {
	result := TestDeclarations{}
	if module == nil {
		return result
	}
	reject := func(attributes []ast.Attribute) {
		for _, attribute := range attributes {
			if attribute.Name == testAttribute {
				result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{Code: "E9001", Span: attribute.Span, Repair: diagnostic.RepairNone})
			}
		}
	}
	for _, value := range module.Structs {
		reject(value.Attributes)
	}
	for _, value := range module.Enums {
		reject(value.Attributes)
	}
	for _, value := range module.Resources {
		reject(value.Attributes)
		if value.Cleanup != nil {
			reject(value.Cleanup.Attributes)
		}
	}
	for _, value := range module.ForeignFunctions {
		reject(value.Attributes)
	}
	for _, value := range module.Constants {
		reject(value.Attributes)
	}
	for _, value := range module.StaticAssertions {
		reject(value.Attributes)
	}
	for index, function := range module.Functions {
		var markers []ast.Attribute
		for _, attribute := range function.Attributes {
			if attribute.Name == testAttribute {
				markers = append(markers, attribute)
			}
		}
		if len(markers) == 0 {
			continue
		}
		valid := len(markers) == 1 && len(markers[0].Arguments) == 0 && !function.Public && !function.Unsafe && function.Owner == "" && len(function.Parameters) == 0 && len(function.TypeParameters) == 0 && function.ResultType.Kind == ast.TypeKindNamed && function.ResultType.Path == "void" && len(function.ResultType.Arguments) == 0
		if !valid {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{Code: "E9001", Span: markers[0].Span, Repair: diagnostic.RepairNone})
			continue
		}
		result.Functions = append(result.Functions, index)
	}
	return result
}

// TestDeclarationDiagnostics validates original source before semantic analysis
// or generic expansion can erase declarations. File.Test is caller-owned metadata.
func TestDeclarationDiagnostics(files []File) []Diagnostic {
	var rejections []Diagnostic
	for _, file := range files {
		if !file.Test {
			continue
		}
		for _, rejection := range InspectTestDeclarations(file.Module).Diagnostics {
			rejections = append(rejections, Diagnostic{Code: rejection.Code, Primary: hir.Span{Package: file.Package, Source: file.Path, StartByte: rejection.Span.StartByte, EndByte: rejection.Span.EndByte}})
		}
	}
	slices.SortFunc(rejections, compareDiagnostic)
	return rejections
}

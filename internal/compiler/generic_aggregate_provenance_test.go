package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestGenericAggregateProvenanceRetainsKindsAndOriginalFields(t *testing.T) {
	text := genericCleanupProgram() + "\n/// Not instantiated.\nstruct Unused<U> { ptr mut U data; }\n"
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	declarations := result.GenericProvenance.Declarations()
	origins := result.GenericProvenance.Specializations()
	if len(declarations) != 5 || len(origins) != 4 || !result.Semantic.RequiresGenericProvenance {
		t.Fatalf("missing provenance: %+v %+v", declarations, origins)
	}
	expected := map[string]string{"Cell": genericsource.OriginStruct, "Token": genericsource.OriginResource,
		"make": genericsource.OriginFunction, "finish": genericsource.OriginFunction, "Unused": genericsource.OriginStruct}
	byID := map[genericplan.DeclarationID]compiler.GenericDeclaration{}
	for _, declaration := range declarations {
		if declaration.Kind != expected[declaration.Name] || declaration.Package != hir.DefaultPackage || declaration.Span.Source != "main.fy" || text[declaration.NameSpan.StartByte:declaration.NameSpan.EndByte] != declaration.Name {
			t.Fatalf("wrong original identity: %+v", declaration)
		}
		byID[declaration.ID] = declaration
		if declaration.Kind != genericsource.OriginFunction && (len(declaration.Signature.Fields) != 1 || len(declaration.Signature.Arguments) != 0 || declaration.Signature.Result.Kind != "") {
			t.Fatalf("aggregate signature confused with function: %+v", declaration.Signature)
		}
		if declaration.Name == "Token" {
			field := declaration.Signature.Fields[0]
			if !declaration.Signature.HasCleanup || field.Name != "payload" || field.Type.Path != "Cell" || field.Type.Arguments[0].Path != "T" {
				t.Fatalf("resource header lost: %+v", declaration)
			}
			declaration.Signature.Fields[0].Type.Arguments[0].Path = "bool"
			declaration.Signature.Fields[0].Name = "changed"
		}
		if declaration.Name == "Unused" && (declaration.Signature.HasCleanup || declaration.DocumentationSpan.EndByte <= declaration.DocumentationSpan.StartByte || declaration.Signature.Fields[0].Type.Element.Path != "U") {
			t.Fatalf("unused source header lost: %+v", declaration)
		}
	}
	for _, origin := range origins {
		declaration, found := byID[origin.Declaration]
		if !found {
			t.Fatal("specialization has no original declaration")
		}
		matches := 0
		switch declaration.Kind {
		case genericsource.OriginFunction:
			for _, fn := range result.MIR.Functions {
				if fn.Name == "main::"+origin.GeneratedName {
					matches++
				}
			}
		case genericsource.OriginStruct:
			for _, typ := range result.MIR.StructTypes {
				if typ.Name == origin.GeneratedName {
					matches++
				}
			}
		case genericsource.OriginResource:
			for _, typ := range result.MIR.ResourceTypes {
				if typ.Type == hir.Type("main::"+origin.GeneratedName) {
					matches++
				}
			}
		}
		if matches != 1 {
			t.Fatalf("origin did not join exactly one %s: %+v", declaration.Kind, origin)
		}
	}
	file.Module.Resources[0].Fields[0].Type.Arguments[0].Path = "u32"
	for _, declaration := range result.GenericProvenance.Declarations() {
		if declaration.Name == "Token" && (declaration.Signature.Fields[0].Name != "payload" || declaration.Signature.Fields[0].Type.Arguments[0].Path != "T") {
			t.Fatal("aggregate provenance aliases source or getter")
		}
	}
	invalid := parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(text, "finish<T>(this.payload.value)", "finish<Unknown>(this.payload.value)", 1))
	rejected, _, _ := compiler.CheckGenericFunctions(invalid, genericplan.DefaultPlanLimits())
	if rejected.Verified || len(rejected.GenericProvenance.Declarations()) != 0 || len(rejected.GenericProvenance.Specializations()) != 0 {
		t.Fatal("rejection published partial provenance")
	}
}

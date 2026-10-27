package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestGenericProvenanceSourceIdentity(t *testing.T) {
	for _, scenario := range []string{"distinct-owners", "distinct-modules", "nil-module", "missing-owner", "duplicate", "owner-collision", "missing", "empty-path", "invalid-span", "header-mismatch", "unrelated-duplicate", "error-order"} {
		t.Run(scenario, func(t *testing.T) {
			span := diagnostic.Span{StartByte: 1, EndByte: 8}
			name := diagnostic.Span{StartByte: 2, EndByte: 4}
			function := ast.Function{Name: "item", TypeParameters: []string{"T"}, Span: span, NameSpan: name}
			input := semantic.PackageInput{Files: []semantic.File{
				{Package: 1, Path: "left.fy", Module: &ast.Module{Path: "shared", SourceSize: 16, Functions: []ast.Function{function}}},
				{Package: 2, Path: "right.fy", Module: &ast.Module{Path: "shared", SourceSize: 16, Functions: []ast.Function{function}}},
			}}
			owners := map[hir.PackageID]genericplan.PackageID{1: "left", 2: "right"}
			expansion := genericsource.PackageExpansion{Declarations: []genericsource.DeclarationOrigin{
				{Kind: genericsource.OriginFunction, ID: "left-item", Owner: "left", Module: "shared", Name: "item", Span: span, NameSpan: name, Parameters: []string{"T"}},
				{Kind: genericsource.OriginFunction, ID: "right-item", Owner: "right", Module: "shared", Name: "item", Span: span, NameSpan: name, Parameters: []string{"T"}},
			}}
			wantError := ""
			switch scenario {
			case "distinct-modules":
				owners[2] = "left"
				input.Files[1].Module.Path = "other"
				expansion.Declarations[1].Owner = "left"
				expansion.Declarations[1].Module = "other"
			case "nil-module":
				input.Files = append(input.Files, semantic.File{Package: 1})
			case "missing-owner":
				delete(owners, 2)
				expansion.Declarations[1].Owner = ""
			case "duplicate":
				input.Files = append(input.Files, input.Files[0])
				wantError = "ambiguous generic declaration source"
			case "owner-collision":
				owners[2] = "left"
				wantError = "ambiguous generic declaration source"
			case "missing":
				expansion.Declarations[0].Module = "absent"
				wantError = "generic declaration has no exact source"
			case "empty-path":
				input.Files[0].Path = ""
				wantError = "generic declaration has no exact source"
			case "invalid-span":
				expansion.Declarations[0].Span.EndByte = 17
				wantError = "generic declaration has no exact source"
			case "header-mismatch":
				expansion.Declarations[0].NameSpan.EndByte++
				wantError = "generic declaration does not match its original header"
			case "unrelated-duplicate", "error-order":
				input.Files = append(input.Files, input.Files[1])
				if scenario == "error-order" {
					expansion.Declarations[0].NameSpan.EndByte++
					wantError = "generic declaration does not match its original header"
				} else {
					expansion.Declarations = expansion.Declarations[:1]
				}
			}
			got, err := genericProvenance(input, owners, expansion, nil)
			if wantError != "" {
				if err == nil || err.Error() != wantError {
					t.Fatalf("error=%v, want %q", err, wantError)
				}
				return
			}
			if err != nil || len(got.declarations) != len(expansion.Declarations) {
				t.Fatalf("declarations=%d, error=%v", len(got.declarations), err)
			}
			for index, declaration := range got.declarations {
				if declaration.Package != input.Files[index].Package || declaration.Span.Source != input.Files[index].Path {
					t.Fatalf("source ownership changed: %+v", declaration)
				}
			}
		})
	}
}

func BenchmarkGenericProvenanceSourceFiles(b *testing.B) {
	for _, files := range []int{64, 512, 2048} {
		b.Run(fmt.Sprint(files), func(b *testing.B) {
			var text strings.Builder
			text.WriteString("module template;")
			for function := range 2 {
				fmt.Fprintf(&text, "fn T item%d<T>(T value) { return value; }", function)
			}
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("template.fy", []byte(text.String())), source.DefaultLimits())
			if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				b.Fatalf("parse source fixture: %v", err)
			}
			var input semantic.PackageInput
			owners := make(map[hir.PackageID]genericplan.PackageID)
			var expansion genericsource.PackageExpansion
			for file := range files {
				module := *parsed.AST
				module.Path = fmt.Sprintf("module_%d", file)
				pkg := hir.PackageID(file + 1)
				owner := genericplan.PackageID(fmt.Sprintf("package_%d", file))
				prepared, err := genericsource.PrepareModule(&module, owner, genericplan.DefaultPlanLimits())
				if err != nil {
					b.Fatal(err)
				}
				input.Files = append(input.Files, semantic.File{Package: pkg, Path: fmt.Sprintf("source_%d.fy", file), Module: &module})
				owners[pkg] = owner
				expansion.Declarations = append(expansion.Declarations, prepared.Declarations...)
			}
			b.ReportAllocs()
			for b.Loop() {
				result, err := genericProvenance(input, owners, expansion, nil)
				if err != nil || len(result.declarations) != files*2 {
					b.Fatalf("declarations=%d, error=%v", len(result.declarations), err)
				}
			}
		})
	}
}

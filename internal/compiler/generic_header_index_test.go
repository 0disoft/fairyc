package compiler

import (
	"fmt"
	"reflect"
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

func TestGenericHeaderIndexMatchesOriginalValidation(t *testing.T) {
	for _, kind := range []string{genericsource.OriginFunction, genericsource.OriginStruct, genericsource.OriginResource} {
		for _, mutation := range []string{"unchanged", "missing", "duplicate", "method", "duplicate-method", "span", "binders", "visibility", "unsupported-kind"} {
			t.Run(kind+"/"+mutation, func(t *testing.T) {
				span := diagnostic.Span{StartByte: 1, EndByte: 8}
				nameSpan := diagnostic.Span{StartByte: 2, EndByte: 4}
				module := &ast.Module{
					Functions: []ast.Function{{Name: "item", Span: span, NameSpan: nameSpan, Public: true, TypeParameters: []string{"T"}}},
					Structs:   []ast.Struct{{Name: "item", Span: span, NameSpan: nameSpan, Public: true, TypeParameters: []string{"T"}}},
					Resources: []ast.Resource{{Name: "item", Span: span, NameSpan: nameSpan, Public: true, TypeParameters: []string{"T"}}},
				}
				origin := genericsource.DeclarationOrigin{Kind: kind, Name: "item", Span: span, NameSpan: nameSpan, Public: true, Parameters: []string{"T"}}
				switch mutation {
				case "missing":
					origin.Name = "absent"
				case "duplicate":
					module.Functions = append(module.Functions, module.Functions[0])
					module.Structs = append(module.Structs, module.Structs[0])
					module.Resources = append(module.Resources, module.Resources[0])
				case "method":
					module.Functions[0].Owner = "Owner"
				case "duplicate-method":
					method := module.Functions[0]
					method.Owner = "Owner"
					module.Functions = append(module.Functions, method)
				case "span":
					origin.Span.EndByte++
				case "binders":
					origin.Parameters = []string{"U"}
				case "visibility":
					origin.Public = false
				case "unsupported-kind":
					origin.Kind = "unknown"
				}
				want, wantErr := originalGenericHeader(module, origin)
				got, gotErr := indexGenericHeaders(module).original(origin)
				if !reflect.DeepEqual(got, want) || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
					t.Fatalf("indexed header=%+v / %v, scan=%+v / %v", got, gotErr, want, wantErr)
				}
			})
		}
	}
}

func BenchmarkGenericProvenanceHeaders(b *testing.B) {
	for _, count := range []int{64, 512, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			var text strings.Builder
			text.WriteString("module main; fn i32 main() { return 0; }")
			for index := range count {
				fmt.Fprintf(&text, "fn T header_%06d<T>(T value) { return value; }", index)
			}
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
			if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				b.Fatalf("parse generic declarations: %v", err)
			}
			owner := genericplan.PackageID("headerbench")
			expansion, err := genericsource.PrepareModule(parsed.AST, owner, genericplan.DefaultPlanLimits())
			if err != nil {
				b.Fatal(err)
			}
			input := semantic.PackageInput{Files: []semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: parsed.AST}}}
			owners := map[hir.PackageID]genericplan.PackageID{hir.DefaultPackage: owner}
			b.ReportAllocs()
			for b.Loop() {
				provenance, err := genericProvenance(input, owners, expansion, nil)
				if err != nil || len(provenance.declarations) != count {
					b.Fatalf("declarations=%d, want %d, error=%v", len(provenance.declarations), count, err)
				}
			}
		})
	}
}

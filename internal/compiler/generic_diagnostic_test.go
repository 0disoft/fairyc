package compiler_test

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericPackageSourceFailuresPublishOnlyDiagnostics(t *testing.T) {
	for _, test := range []struct{ source, primary, code string }{
		{"fn T id<T>(T v) { return v; } fn i32 main() { return id<i32,bool>(42); }", "id<i32,bool>", "E4502"},
		{"fn T id<T,T>(T v) { return v; } fn i32 main() { return 0; }", "fn T id<T,T>(T v) { return v; }", "E4501"},
		{"fn T id<T>(T v) { return id<T>(v); } fn i32 main() { return id<i32>(42); }", "id<i32>", "E4503"},
		{"fn i32 main() { return missing<i32>(42); }", "missing<i32>", "E0101"},
		{"fn T id<T>(T v) { return v; } fn i32 main() { return id(42); }", "id(42)", "E4502"},
	} {
		t.Run(test.code+test.primary, func(t *testing.T) {
			// Non-ASCII trivia and CRLF retain original byte coordinates.
			text := "module main; // 한글\r\n" + test.source
			input := semantic.PackageInput{Files: []semantic.File{parseCompilerFile(t, 7, "main.fy", text)}}
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{7: genericCoordinate(t)}
			analysis, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
			semanticResult := analysis.Analysis.Semantic
			if err != nil || semanticResult.FailedPhase != semantic.PhaseResolve || semanticResult.CompletedThrough != semantic.PhaseNone || len(semanticResult.Diagnostics) != 1 || !reflect.DeepEqual(semanticResult.HIR, hir.Package{}) || analysis.Interfaces != nil || analysis.OrdinaryInterfaces != nil || analysis.Specializations != nil || len(analysis.Plan.Specializations) != 0 {
				t.Fatalf("source error published compiler facts: %v %+v", err, analysis)
			}
			d := semanticResult.Diagnostics[0]
			if d.Code != test.code || d.Primary.Package != 7 || fixtureSpanText(t, text, d.Primary) != test.primary {
				t.Fatalf("diagnostic mismatch: %+v", d)
			}
			compiled, plan, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err != nil || compiled.Verified || len(compiled.MIR.Functions) != 0 || len(compiled.StateTransactions) != 0 || len(plan.Specializations) != 0 {
				t.Fatalf("source rejection lowered: %v %+v", err, compiled)
			}
		})
	}
}

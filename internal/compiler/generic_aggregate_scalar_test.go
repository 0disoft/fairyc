package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

const genericAggregateScalarProgram = `module main;
fn i32 observe(i32 value) { return value; }
fn i32 check_drop() { return 1; }
struct Cell<T> { T value; }
resource Token<T> {
    Cell<T> payload;
    cleanup { let T observed = finish<T>(this.payload.value); }
}
fn T finish<T>(T value) { let i32 observed = observe(65); return value; }
fn Token<T> make<T>(T value) { return Token<T> { .payload = Cell<T> { .value = value } }; }
fn i32 consume<T>(Token<T> owned, T expected) {
    if (owned.payload.value == expected) { return 41; }
    return 0;
}
fn i32 main() {
    let Token<SCALAR> original = make<SCALAR>(VALUE);
    let Token<SCALAR> moved = original;
    return consume<SCALAR>(moved, VALUE) + check_drop();
}
`

// The scalar domain is the proposed stable aggregate boundary, not arbitrary nominal payloads.
// Interpreter checks the package-aware route; the native helper independently checks both MSVC
// profiles and a C-side exactly-once cleanup marker through the explicit-file route.
func TestGenericAggregateScalarDomainMovesAndDrops(t *testing.T) {
	for _, item := range []struct{ scalar, value string }{{"bool", "true"}, {"u8", "255"}, {"i32", "-2147483647"}, {"u32", "4294967295"}, {"usize", "65537"}} {
		t.Run(item.scalar, func(t *testing.T) {
			text := strings.NewReplacer("SCALAR", item.scalar, "VALUE", item.value).Replace(genericAggregateScalarProgram)
			file := parseCompilerFile(t, 1, "main.fy", text)
			coordinate := genericCoordinate(t)
			result, plan, err := compiler.CheckGenericPackages(semantic.PackageInput{Files: []semantic.File{file}}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate}, genericplan.DefaultPlanLimits())
			if err != nil || !result.Verified {
				t.Fatalf("package compile: %v %+v", err, result.Semantic)
			}
			if len(plan.Specializations) != 5 || len(result.MIR.ResourceTypes) != 1 || len(result.MIR.StructTypes) != 1 {
				t.Fatalf("domain expansion: %+v", plan)
			}
			module := hir.QualifiedModulePath(1, "main")
			outcome, err := interpreter.ExecuteWithOptions(result.MIR, module+"::main", interpreter.Options{TestIntrinsics: map[string]interpreter.TestIntrinsicKind{module + "::observe": interpreter.TestIntrinsicWriteStdoutByte}, OutputLimit: 10})
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 1 || string(outcome.Stdout) != "A" {
				t.Fatalf("package execution: %+v %v", outcome, err)
			}
			t.Run("native", func(t *testing.T) { runGenericAggregateNative(t, text) })
			// The same path must not admit a consumed resource a second time.
			invalid := strings.Replace(text, "return consume<"+item.scalar+">(moved, "+item.value+") + check_drop();", "let i32 consumed = consume<"+item.scalar+">(moved, "+item.value+"); return consume<"+item.scalar+">(original, "+item.value+");", 1)
			badFile := parseCompilerFile(t, 1, "main.fy", invalid)
			bad, _, err := compiler.CheckGenericPackages(semantic.PackageInput{Files: []semantic.File{badFile}}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate}, genericplan.DefaultPlanLimits())
			if err == nil && bad.Verified {
				t.Fatal("scalar aggregate use-after-move accepted")
			}
		})
	}
}

package mir_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestIgnoredResourceTemporaryCleanup(t *testing.T) {
	const declarations = `module main;
resource Bad { i32 value; cleanup { assert(false); } }
fn Bad make() { return Bad { .value = 1 }; }
fn Option<Bad> some() { return .some(make()); }
fn Option<Bad> none() { return .none; }
fn Result<Bad,i32> ok() { return .ok(make()); }
fn Result<i32,Bad> err() { return .err(make()); }
fn Result<Bad,i32> scalar_err() { return .err(1); }
fn Result<i32,Bad> scalar_ok() { return .ok(1); }
`
	for _, tc := range []struct {
		name, body string
		trap       bool
	}{
		{"discard-constructor", `discard Bad { .value = 1 };`, true},
		{"ignored-constructor", `Bad { .value = 1 };`, true},
		{"discard-call", `discard make();`, true},
		{"ignored-call", `make();`, true},
		{"grouped-call", `discard (make());`, true},
		{"block-result", `discard { let Bad value = make(); value };`, true},
		{"branch-result", `discard if (true) { make() } else { make() };`, true},
		{"option-some", `discard some();`, true},
		{"ignored-option", `some();`, true},
		{"option-none", `discard none();`, false},
		{"result-ok", `discard ok();`, true},
		{"result-err", `discard err();`, true},
		{"result-inactive-ok", `discard scalar_err();`, false},
		{"result-inactive-err", `discard scalar_ok();`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := lowerIgnoredCleanupSource(t, declarations+`fn i32 main() { `+tc.body+` return 42; }`)
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil {
				t.Fatal(err)
			}
			if tc.trap {
				if outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "assertion-failed" || len(outcome.DropTrace) != 1 {
					t.Fatalf("ignored resource must run cleanup: %+v", outcome)
				}
			} else if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 0 {
				t.Fatalf("inactive resource payload must not run cleanup: %+v", outcome)
			}
		})
	}
}

func TestIgnoredResourceTemporaryCleanupOrder(t *testing.T) {
	program := lowerIgnoredCleanupSource(t, `module main;
fn i32 observe(i32 value) { return value; }
resource Mark { i32 id; cleanup { discard observe(this.id); } }
fn Mark make(i32 id) { return Mark { .id = id }; }
fn Option<Mark> some(i32 id) { return .some(make(id)); }
fn Result<Mark,i32> ok(i32 id) { return .ok(make(id)); }
fn i32 main() {
    let Mark local = make(70);
    discard (local);
    discard Mark { .id = 65 };
    make(66);
    discard some(67);
    discard ok(68);
    var i32 index = 0;
    while (index < 2) { discard make(71); index += 1; }
    discard 123;
    456;
    discard observe(69);
    return 42;
}`)
	outcome, err := interpreter.ExecuteWithOptions(program, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{"main::observe": interpreter.TestIntrinsicWriteStdoutByte}, OutputLimit: 10,
	})
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || string(outcome.Stdout) != "ABCDGGEF" || len(outcome.DropTrace) != 7 {
		t.Fatalf("immediate, exactly-once cleanup and scalar evaluation: %+v %v", outcome, err)
	}
}

func lowerIgnoredCleanupSource(t *testing.T, text string) mir.Program {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse: %v %+v", err, parsed.Tree.Diagnostics)
	}
	result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("analyze: %+v", result)
	}
	program, err := mir.Lower(result.HIR)
	if err != nil {
		t.Fatal(err)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("verify: %+v", failures)
	}
	return program
}

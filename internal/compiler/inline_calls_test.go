package compiler_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/inlinebody"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const inlineCallsValueSource = `module main;
@inline pub fn i32 outer(i32 x){let a=add(leaf(x),leaf(x));return add(a,zero());}
@inline pub fn i32 leaf(i32 x){return x+1;}
@inline pub fn i32 add(i32 x,i32 y){return x+y;}
@inline pub fn i32 zero(){return 2;}
fn i32 main(){return outer(19);}`

func checkInlineCalls(t *testing.T, text string) compiler.Result {
	t.Helper()
	r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
	if err != nil || !r.Verified {
		t.Fatalf("inline calls: %v diagnostics=%+v unsupported=%+v", err, r.Semantic.Diagnostics, r.Semantic.Unsupported)
	}
	return r
}

func TestInlineCallsCanonicalVectors(t *testing.T) {
	for _, test := range []struct{ name, declarations, vector string }{
		{"direct", `@inline pub fn i32 f(i32 x){return leaf(x);}@inline pub fn i32 leaf(i32 x){return x+1;}`,
			`{"Version":"fairyc.inline-function/v3","Arity":1,"Root":{"Op":"call","Value":"leaf","Children":[{"Op":"path","Value":"0","Children":null}]}}`},
		{"zero", `@inline pub fn i32 f(){return leaf();}@inline pub fn i32 leaf(){return 42;}`,
			`{"Version":"fairyc.inline-function/v3","Arity":0,"Root":{"Op":"call","Value":"leaf","Children":null}}`},
		{"bindings", `@inline pub fn i32 f(i32 x){let a=zero();return add(x,a);}@inline pub fn i32 zero(){return 2;}@inline pub fn i32 add(i32 x,i32 y){return x+y;}`,
			`{"Version":"fairyc.inline-function/v3","Arity":1,"Root":{"Op":"bindings","Value":"","Children":[{"Op":"call","Value":"zero","Children":null},{"Op":"call","Value":"add","Children":[{"Op":"path","Value":"0","Children":null},{"Op":"local","Value":"0","Children":null}]}]}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := checkInlineCalls(t, "module main;"+test.declarations+"fn i32 main(){return 0;}")
			fn := r.Semantic.HIR.Modules[0].Functions[0]
			want := fmt.Sprintf("%s%x", hir.InlineFunctionPrefix, sha256.Sum256([]byte(test.vector)))
			if fn.InlineBody != want {
				t.Fatalf("canonical vector: %s want %s", fn.InlineBody, want)
			}
			if _, err := hir.InlineFunctionID(fn); err == nil {
				t.Fatal("call identity accepted without resolved declaration membership")
			}
		})
	}
	base := `module main;@inline pub fn i32 f(i32 x){return leaf(x);}@inline pub fn i32 leaf(i32 x){return x+1;}fn i32 main(){return f(41);}`
	baseline := checkInlineCalls(t, base).Semantic.HIR.Modules[0].Functions[0].InlineBody
	for _, test := range []struct {
		name, text string
		same       bool
	}{
		{"parameter", strings.Replace(base, "f(i32 x){return leaf(x);}", "f(i32 input){return leaf(input);}", 1), true},
		{"callee-body", strings.Replace(base, "x+1", "x+2", 1), true},
		{"callee-name", strings.ReplaceAll(base, "leaf", "renamed"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := checkInlineCalls(t, test.text).Semantic.HIR.Modules[0].Functions[0].InlineBody
			if (got == baseline) != test.same {
				t.Fatal("callee declaration name or independent body identity changed incorrectly")
			}
		})
	}
}

func TestInlineCallsExecutionAndRestoration(t *testing.T) {
	r := checkInlineCalls(t, inlineCallsValueSource)
	out, err := interpreter.Execute(r.MIR, "main::main")
	if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 42 {
		t.Fatalf("execute: %+v %v", out, err)
	}
	// Each source occurrence lowers once, preserving actual arguments as
	// values rather than substituting their expression trees into a callee.
	for _, fn := range r.MIR.Functions {
		if fn.Name == "main::outer" {
			calls := 0
			for _, block := range fn.Blocks {
				if block.Terminator.Kind == mir.TerminatorCall {
					calls++
				}
			}
			if calls != 5 || len(fn.Calls) != 5 {
				t.Fatalf("source call occurrences duplicated or lost: %d / %d", calls, len(fn.Calls))
			}
		}
	}
	wire, err := mircodec.Encode(r.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(wire, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if failures := mir.Verify(restored); len(failures) != 0 {
		t.Fatal(failures)
	}
	if err := inlinebody.VerifyProgram(r.Semantic.HIR, restored); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(restored, "main::main", profile); err != nil {
			t.Fatal(err)
		}
		parts, err := cbackend.EmitBuildContext(context.Background(), restored, "main::main", profile, false, 1)
		if err != nil || !strings.Contains(parts.Header, "static inline int32_t") {
			t.Fatalf("partitioned inline calls: %v", err)
		}
	}
}

func inlineCallsDAG(count int, reverse bool) string {
	var text strings.Builder
	text.WriteString("module main;")
	for row := 0; row < count; row++ {
		i := row
		if reverse {
			i = count - 1 - row
		}
		expr := "x"
		if i != 0 {
			expr = fmt.Sprintf("f%d(x)+f%d(x)", i-1, i-1)
		}
		fmt.Fprintf(&text, "@inline pub fn i32 f%d(i32 x){return %s;}", i, expr)
	}
	text.WriteString("fn i32 main(){return 0;}")
	return text.String()
}

func TestInlineCallsGraphAndNodeBounds(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, count := range []int{64, 65} {
			t.Run(fmt.Sprintf("dag-%d-reverse-%t", count, reverse), func(t *testing.T) {
				text := inlineCallsDAG(count, reverse)
				r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
				if err != nil || r.Verified != (count == 64) {
					t.Fatalf("distinct graph bound: verified=%t error=%v unsupported=%+v", r.Verified, err, r.Semantic.Unsupported)
				}
				if count == 65 && len(r.Semantic.Unsupported) == 0 {
					t.Fatal("graph rejection lacks source unsupported diagnostic")
				}
			})
		}
	}
	argument := strings.Repeat("x+", 31) + "x"
	for _, over := range []bool{false, true} {
		expr := argument
		if over {
			expr = "(" + expr + ")"
		}
		text := "module main;@inline pub fn i32 f(i32 x){return leaf(" + expr + ");}@inline pub fn i32 leaf(i32 x){return x;}fn i32 main(){return 0;}"
		r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
		if err != nil || r.Verified == over {
			t.Fatalf("64/65 nodes: over=%t verified=%t error=%v", over, r.Verified, err)
		}
	}
}

func TestInlineCallsMaxNestedNodeBudget(t *testing.T) {
	for _, calls := range []int{63, 64} {
		t.Run(fmt.Sprintf("nodes-%d", calls+1), func(t *testing.T) {
			expression := strings.Repeat("leaf(", calls) + "x" + strings.Repeat(")", calls)
			text := "module main;@inline pub fn i32 outer(i32 x){return " + expression + ";}@inline pub fn i32 leaf(i32 x){return x;}fn i32 main(){return outer(42);}"
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil {
				t.Fatal(err)
			}
			if calls == 64 {
				if r.Verified || len(r.Semantic.Unsupported) == 0 {
					t.Fatal("65 typed expression nodes escaped source admission")
				}
				return
			}
			if !r.Verified {
				t.Fatalf("64 typed nodes rejected: diagnostics=%+v unsupported=%+v", r.Semantic.Diagnostics, r.Semantic.Unsupported)
			}
			found := false
			for _, fn := range r.MIR.Functions {
				if fn.Name != "main::outer" {
					continue
				}
				found = true
				operations, callTerminators := 0, 0
				for _, block := range fn.Blocks {
					operations += len(block.Operations)
					if block.Terminator.Kind == mir.TerminatorCall {
						callTerminators++
					}
				}
				if operations > 128 || len(fn.Blocks) > 256 || callTerminators != calls {
					t.Fatalf("nested call lowering: operations=%d blocks=%d calls=%d", operations, len(fn.Blocks), callTerminators)
				}
				t.Logf("64 typed nodes: operations=%d/128 blocks=%d/256 call terminators=%d", operations, len(fn.Blocks), callTerminators)
			}
			if !found {
				t.Fatal("nested inline caller missing")
			}
			out, err := interpreter.Execute(r.MIR, "main::main")
			if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 42 {
				t.Fatalf("nested calls: %+v %v", out, err)
			}
		})
	}
}

func TestInlineCallsClosedFormsAndResolvedMembership(t *testing.T) {
	const caller = "@inline pub fn i32 f(i32 x){return leaf(x);}"
	for name, declarations := range map[string]string{
		"ordinary":         caller + "pub fn i32 leaf(i32 x){return x;}",
		"private":          caller + "@inline fn i32 leaf(i32 x){return x;}",
		"generic":          caller + "@inline pub fn i32 leaf<T>(i32 x){return x;}",
		"effects":          caller + "@inline @effects(console) pub fn i32 leaf(i32 x){return x;}",
		"unsafe":           "@inline pub fn i32 f(i32 x){return unsafe{leaf(x)};}@inline pub unsafe fn i32 leaf(i32 x){return x;}",
		"foreign":          "unsafe extern C fn i32 leaf(i32 x);@inline pub fn i32 f(i32 x){return unsafe{leaf(x)};}",
		"self":             "@inline pub fn i32 f(i32 x){return f(x);}",
		"mutual":           caller + "@inline pub fn i32 leaf(i32 x){return f(x);}",
		"parameter-shadow": "@inline pub fn i32 leaf(i32 x){return x;}@inline pub fn i32 f(i32 leaf){return leaf(1);}",
		"local-shadow":     "@inline pub fn i32 leaf(i32 x){return x;}@inline pub fn i32 f(){let leaf=1;return leaf(1);}",
		"arity":            "@inline pub fn i32 f(){return leaf();}@inline pub fn i32 leaf(i32 x){return x;}",
		"label":            "@inline pub fn i32 f(i32 x){return leaf(.x=x);}@inline pub fn i32 leaf(i32 x){return x;}",
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+declarations+"fn i32 main(){return 0;}")})
			if err != nil {
				t.Fatal(err)
			}
			if r.Verified || len(r.Semantic.Diagnostics)+len(r.Semantic.Unsupported) == 0 {
				t.Fatal("closed call accepted or missing source diagnostic")
			}
		})
	}
	t.Run("other-module", func(t *testing.T) {
		r, err := compiler.Check([]semantic.File{
			parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;import other as h;@inline pub fn i32 f(i32 x){return h::leaf(x);}fn i32 main(){return 0;}"),
			parseCompilerFile(t, hir.DefaultPackage, "other.fy", "module other;@inline pub fn i32 leaf(i32 x){return x;}"),
		})
		if err != nil || r.Verified {
			t.Fatalf("other-module target admitted: %v", err)
		}
	})
	r := checkInlineCalls(t, "module main;@inline pub fn i32 f(i32 x){return leaf(x);}@inline pub fn i32 leaf(i32 x){return x;}pub fn i32 ordinary(i32 x){return x;}fn i32 main(){return 0;}")
	m := r.Semantic.HIR.Modules[0]
	fn := m.Functions[0]
	call := *fn.Body.Statements[0].Expression
	call.Children = append([]hir.Expression(nil), call.Children...)
	call.Symbol = m.Functions[2].Symbol
	call.Children[0].Symbol = call.Symbol
	fn.Body.Statements = append([]hir.Statement(nil), fn.Body.Statements...)
	fn.Body.Statements[0].Expression = &call
	if _, err := hir.InlineFunctionID(fn, hir.InlineModuleResolver(m)); err == nil {
		t.Fatal("ordinary resolved symbol bypassed inline membership")
	}
	call.Symbol = m.Functions[1].Symbol
	call.Children[0].Symbol = call.Symbol
	call.Children = call.Children[:1]
	if _, err := hir.InlineFunctionID(fn, hir.InlineModuleResolver(m)); err == nil {
		t.Fatal("forged HIR call arity admitted")
	}
}

func TestInlineCallsRejectMalformedMIR(t *testing.T) {
	r := checkInlineCalls(t, "module main;@inline pub fn i32 f(i32 x){return leaf(x);}@inline pub fn i32 leaf(i32 x){return x;}fn i32 main(){return f(42);}")
	for _, name := range []string{"unmarked", "other-module", "effects", "unsafe", "foreign", "cycle", "bad-identity"} {
		t.Run(name, func(t *testing.T) {
			program := mir.Clone(r.MIR)
			var caller, target *mir.Function
			for i := range program.Functions {
				fn := &program.Functions[i]
				if fn.Name == "main::f" {
					caller = fn
				}
				if fn.Name == "main::leaf" {
					target = fn
				}
			}
			if caller == nil || target == nil {
				t.Fatal("fixture functions missing")
			}
			switch name {
			case "unmarked":
				target.InlineBody = ""
			case "other-module":
				target.Name = "elsewhere::leaf"
			case "effects":
				target.Effects = []hir.Effect{hir.EffectConsole}
			case "unsafe":
				target.UnsafeOperations = []mir.UnsafeOperation{{Entity: "forged"}}
			case "bad-identity":
				target.InlineBody = "bad"
			case "foreign", "cycle":
				for i := range caller.Blocks {
					term := &caller.Blocks[i].Terminator
					if term.Kind == mir.TerminatorCall {
						if name == "foreign" {
							term.ForeignABI = "C"
						} else {
							term.Callee = caller.Symbol
						}
					}
				}
			}
			inlineRejected := false
			for _, failure := range mir.Verify(program) {
				inlineRejected = inlineRejected || strings.HasPrefix(failure.Detail, "inline body:")
			}
			if !inlineRejected {
				t.Fatal("malformed inline MIR lacked inline rejection")
			}
			if _, err := mircodec.Encode(program, 1<<20); err == nil {
				t.Fatal("malformed inline MIR encoded")
			}
			// Bypass the encoder to simulate an externally altered artifact.
			wire, err := json.Marshal(struct {
				Schema     string      `json:"schema"`
				MIRVersion string      `json:"mirVersion"`
				Program    mir.Program `json:"program"`
			}{mircodec.Schema, mircodec.MIRVersion, program})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := mircodec.Decode(wire, 1<<20); err == nil {
				t.Fatal("malformed inline artifact decoded")
			}
		})
	}
}

func TestInlineCallsMIRGraphBoundAndUnmarkedFunctions(t *testing.T) {
	r := checkInlineCalls(t, inlineCallsDAG(64, false))
	program := mir.Clone(r.MIR)
	var top *mir.Function
	var nextSymbol hir.SymbolID
	for i := range program.Functions {
		fn := &program.Functions[i]
		if fn.Symbol >= nextSymbol {
			nextSymbol = fn.Symbol + 1
		}
		if fn.Name == "main::f63" {
			top = fn
		}
	}
	if top == nil {
		t.Fatal("graph root missing")
	}
	// Add an individually valid marked caller above the already verified
	// 64-function DAG. Counting cached subgraphs as one would admit this.
	extra := mir.Clone(mir.Program{Functions: []mir.Function{*top}}).Functions[0]
	extra.Name, extra.Symbol = "main::extra", nextSymbol
	for i := range extra.Blocks {
		if extra.Blocks[i].Terminator.Kind == mir.TerminatorCall {
			extra.Blocks[i].Terminator.Callee = top.Symbol
		}
	}
	program.Functions = append(program.Functions, extra)
	rejected := false
	for _, failure := range mir.Verify(program) {
		if failure.Entity == extra.Name && strings.HasPrefix(failure.Detail, "inline body:") {
			rejected = true
		}
	}
	if !rejected {
		t.Fatal("65 reachable MIR inline declarations admitted")
	}
	// Ordinary calls have no new inline graph restriction, even with this DAG.
	for i := range program.Functions {
		program.Functions[i].InlineBody = ""
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("unmarked functions acquired inline restrictions: %v", failures)
	}
}

var inlineCallsTrapCases = []struct{ name, body, trap string }{
	{"arguments-division-first", "return add(divide(0),overflow(1));", "division-by-zero"},
	{"arguments-overflow-first", "return add(overflow(1),divide(0));", "integer-overflow"},
	{"unused-local", "let unused=divide(0);return 42;", "division-by-zero"},
	{"locals-source-order", "let unused=overflow(1);let second=divide(0);return 42;", "integer-overflow"},
}

func inlineCallsTrapSource(body string) string {
	return "module main;@inline pub fn i32 f(){" + body + "}@inline pub fn i32 divide(i32 x){return 10/x;}@inline pub fn i32 overflow(i32 x){return 2147483647+x;}@inline pub fn i32 add(i32 x,i32 y){return x+y;}fn i32 main(){return f();}"
}

func TestInlineCallsTrapOrder(t *testing.T) {
	for _, test := range inlineCallsTrapCases {
		t.Run(test.name, func(t *testing.T) {
			r := checkInlineCalls(t, inlineCallsTrapSource(test.body))
			out, err := interpreter.Execute(r.MIR, "main::main")
			if err != nil || out.Kind != interpreter.OutcomeTrap || out.TrapCategory != test.trap {
				t.Fatalf("trap order: %+v %v", out, err)
			}
		})
	}
}

func TestInlineCallsWindowsNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 native; Linux validation is batched separately")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		name, text string
		exit       int32
	}{
		{"value", inlineCallsValueSource, 42},
		{"division-first", inlineCallsTrapSource(inlineCallsTrapCases[0].body), 195},
		{"overflow-first", inlineCallsTrapSource(inlineCallsTrapCases[1].body), 194},
	} {
		r := checkInlineCalls(t, sample.text)
		unit, err := cbackend.Emit(r.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(sample.name+"/"+string(mode)+"/"+profile.Name, func(t *testing.T) {
					artifact, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(tc, profile, mode, unit.Source, support)
					if err != nil {
						t.Fatal(err)
					}
					out, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
					if err != nil || out.ExitCode != sample.exit || out.Stdout != "" || out.Stderr != "" {
						t.Fatalf("native: %+v %v", out, err)
					}
				})
			}
		}
	}
}

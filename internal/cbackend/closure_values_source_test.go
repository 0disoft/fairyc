package cbackend_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

const closureValueSourceFactories = `module main;
struct Holder {fn i32(i32) action;}
fn fn i32(i32) make(i32 seed) {
 var i32 a=seed;let i32 b=3;let i32 c=2;let i32 d=4;
 let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 x) {
  let reduced=a-b*x;return reduced+c-d;
 };
 a=999;return action;
}
fn fn i32(i32) other() {
 let fn i32(i32) action=fn [] i32(i32 x){return x-1;};return action;
}
fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
fn i32 main(){
 let original=make(50);let holder=Holder{.action=original};let copied=holder;
 let extracted=copied.action;let zero=other();assert(apply(zero,43)==42);
 return apply(extracted,2);
}`

func closureValueSourceCases() []struct {
	name, code, trap string
	want             int32
} {
	return []struct {
		name, code, trap string
		want             int32
	}{
		{"factories-struct-snapshot", closureValueSourceFactories, "", 42},
		{"argument-once", `module main;
fn fn i32(i32) make(i32 a){let fn i32(i32) f=fn [i32 a] i32(i32 x){return a+x;};return f;}
fn i32 main(){let f=make(40);var i32 count=0;let result=f({count=count+1;2});assert(count==1);return result;}`, "", 42},
		{"argument-trap-before-body", `module main;
fn fn i32(i32) make(){let fn i32(i32) f=fn [] i32(i32 x){return x+2147483647;};return f;}
fn i32 main(){let f=make();return f(1/0);}`, "division-by-zero", 0},
		{"unused-trapping-body", `module main;
fn fn i32(i32) make(){let fn i32(i32) f=fn [] i32(i32 x){let unused=1/0;return x;};return f;}
fn i32 main(){let f=make();return 42;}`, "", 42},
		{"invoked-trapping-local", `module main;
fn fn i32(i32) make(){let fn i32(i32) f=fn [] i32(i32 x){let unused=1/0;return x;};return f;}
fn i32 main(){let f=make();return f(42);}`, "division-by-zero", 0},
	}
}

func TestClosureValueSourceExecutionAndEmission(t *testing.T) {
	for _, tc := range closureValueSourceCases() {
		t.Run(tc.name, func(t *testing.T) {
			// This path parses source, prepares closures, checks semantics and lowers
			// HIR normally. No MIR fields or lifted-target metadata are patched.
			p := lowerSource(t, tc.code)
			if failures := mir.Verify(p); len(failures) != 0 {
				t.Fatalf("source MIR gate: %+v", failures)
			}
			before, _ := json.Marshal(p)
			out, err := interpreter.Execute(p, "main::main")
			if err != nil || out.TrapCategory != tc.trap || (tc.trap == "" && (out.Kind != interpreter.OutcomeReturn || out.Value != tc.want)) {
				t.Fatalf("source interpreter %+v %v", out, err)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				u, err := cbackend.Emit(p, "main::main", profile)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(u.Source, "fy_closure_i32") {
					t.Fatal("source closure transport absent")
				}
			}
			after, _ := json.Marshal(p)
			if string(before) != string(after) {
				t.Fatal("source execution/emission mutated MIR")
			}
		})
	}
}

func TestClosureValueSourceNative(t *testing.T) {
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	toolchain, err := cdriver.ResolveNativeToolchain(id, "")
	if err != nil {
		t.Fatal(err)
	}
	profile := cbackend.Stage0Profile()
	render := cruntime.LinuxRunSource
	if id == target.X8664WindowsMSVC {
		profile = cbackend.Stage0WindowsMSVCProfile()
		render = cruntime.WindowsRunSource
	}
	support, err := render(cdriver.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range closureValueSourceCases() {
		t.Run(tc.name, func(t *testing.T) {
			p := lowerSource(t, tc.code)
			u, err := cbackend.Emit(p, "main::main", profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, optimization := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: optimization, UnitSource: u.Source, RuntimeSource: support, TrapControl: true})
				if err != nil {
					t.Fatalf("%v\n%s", err, u.Source)
				}
				out, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{RejectOutput: true})
				if err != nil || out.TrapCategory != tc.trap || (tc.trap == "" && out.ExitCode != tc.want) {
					t.Fatalf("%s: %+v %v", optimization.Name, out, err)
				}
			}
		})
	}
}

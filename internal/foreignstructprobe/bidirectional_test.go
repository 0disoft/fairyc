package foreignstructprobe_test

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/foreignstructprobe"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func emitBidirectional(t *testing.T, posix bool) (foreignstructprobe.Bidirectional, cbackend.Unit, string) {
	t.Helper()
	p, err := foreignstructprobe.GenerateBidirectional(posix)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(p.FairySource)), source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		t.Fatal(err)
	}
	if !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		t.Fatalf("compile: %+v %+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	profile := cbackend.Stage0WindowsMSVCProfile()
	if posix {
		profile = cbackend.Stage0Profile()
	}
	u, err := cbackend.Emit(checked.MIR, "main::main", profile)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := p.Bind(u.Source, u.Functions)
	if err != nil {
		t.Fatal(err)
	}
	return p, u, bound
}

func TestBidirectionalAggregateEmission(t *testing.T) {
	for _, posix := range []bool{false, true} {
		name := "windows"
		if posix {
			name = "linux"
		}
		t.Run(name, func(t *testing.T) {
			p, u, bound := emitBidirectional(t, posix)
			want := 133
			if posix {
				want = 139
			}
			if len(p.Records) != want {
				t.Fatalf("aggregate inventory changed: %d, want %d; review the documented matrix", len(p.Records), want)
			}
			again, err := foreignstructprobe.GenerateBidirectional(posix)
			if err != nil || !reflect.DeepEqual(p, again) {
				t.Fatalf("nondeterministic probe: %v", err)
			}
			t.Logf("%d aggregate records; three value sets; both call directions; %d generated functions", len(p.Records), len(u.Functions))
			for _, tag := range p.Records {
				if !strings.Contains(bound, "struct "+tag+" reverse_bridge_"+tag+"(") {
					t.Fatalf("missing bridge %s", tag)
				}
			}
			if _, err := p.Bind(u.Source, nil); err == nil {
				t.Fatal("accepted missing Fairy function metadata")
			}
			if _, err := p.Bind("", u.Functions); err == nil {
				t.Fatal("accepted missing Fairy signature")
			}
			for _, fragment := range p.RequiredCFragments {
				if !strings.Contains(u.Source, fragment) {
					t.Fatalf("lost forward expectation %q", fragment)
				}
			}
		})
	}
}

func TestBidirectionalAggregateRejectsClosedShapes(t *testing.T) {
	for name, body := range map[string]string{
		"empty":                "@repr(C) struct Bad{}",
		"fairy-bool":           "@repr(C) struct Bad{bool value;}",
		"safe-borrow":          "@repr(C) struct Bad{ref u8 value;}",
		"owning-tag":           "@repr(C) struct Bad{Option<i32> value;}",
		"private-record-array": "struct Element{u8 value;} @repr(C) struct Bad{Element[2] values;}",
		"bare-array":           "unsafe extern C fn u8[2] bad();",
	} {
		t.Run(name, func(t *testing.T) {
			use := "unsafe extern C fn Bad make_bad();fn i32 main(){let Bad value=unsafe{make_bad()};return 0;}"
			if name == "bare-array" {
				use = "fn i32 main(){let u8[2] value=unsafe{bad()};return 0;}"
			}
			code := "module main;" + body + use
			p, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(code)), source.DefaultLimits())
			if err != nil || p.AST == nil || len(p.Tree.Diagnostics) != 0 {
				t.Fatalf("negative case must parse: %v %+v", err, p.Tree.Diagnostics)
			}
			checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: p.AST}})
			if err != nil {
				t.Fatal(err)
			}
			if !checked.Verified {
				if len(checked.Semantic.Diagnostics) == 0 && len(checked.Semantic.Unsupported) == 0 {
					t.Fatal("unverified negative case without diagnostics")
				}
				return
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				if _, err := cbackend.Emit(checked.MIR, "main::main", profile); err == nil {
					t.Fatalf("unsupported aggregate shape emitted for %s", profile.Target)
				}
			}
		})
	}
}

func TestBidirectionalAggregateWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows amd64 MSVC; Linux native runs are batched separately")
	}
	p, _, bound := emitBidirectional(t, false)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := cruntime.ScalarSource(128)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MSVC %s", tc.Version)
	for _, mode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(string(mode)+"/"+profile.Name, func(t *testing.T) {
				a, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(tc, profile, mode, bound, p.CompanionC+"\n"+rt)
				if err != nil {
					t.Fatal(err)
				}
				if len(a.Objects) != 2 || a.Objects[0].Name != "caller.obj" || a.Objects[1].Name != "callee.obj" {
					t.Fatal("lost separate object receipts")
				}
				if _, err := pecoff.InspectExecutable(a.Bytes); err != nil {
					t.Fatal(err)
				}
				if _, err := pecoff.InspectAMD64Unwind(a.Bytes); err != nil {
					t.Fatal(err)
				}
				run, err := cdriver.ExecuteWindowsMSVCABIProbe(a)
				if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
					t.Fatalf("aggregate execution: %+v %v", run, err)
				}
				t.Logf("%d records x 3 value sets; two independent objects, PE and unwind inspected; exit 42, empty streams", len(p.Records))
			})
		}
	}
}

func TestBidirectionalAggregateWindowsTrap(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows amd64 MSVC")
	}
	p, _, bound := emitBidirectional(t, false)
	// The first mark comes from the Fairy caller. The second is reached only
	// when the independent C observer invokes the generated Fairy body.
	needle := "++fy_reverse_counts[id];return 42;"
	if strings.Count(p.CompanionC, needle) != 1 {
		t.Fatal("trap injection point drifted")
	}
	companion := strings.Replace(p.CompanionC, needle, "++fy_reverse_counts[id];return fy_reverse_counts[id]==2?0:42;", 1)
	rt, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			a, err := cdriver.BuildWindowsMSVCABIProbe(tc, profile, bound, companion+"\n"+rt)
			if err != nil {
				t.Fatal(err)
			}
			a.TrapControl = true
			run, err := cdriver.ExecuteWindowsMSVCABIProbe(a)
			if err != nil || run.ExitCode != 134 || len(run.TrapControl) == 0 || run.TrapControl[0] != 6 || run.Stdout != "" || run.Stderr != "" {
				t.Fatalf("C -> Fairy assertion receipt: %+v %v", run, err)
			}
		})
	}
}

// Execute later on the shared Linux validation host. This test is never a
// substitute for Windows execution and does not attempt WSL or provisioning.
func TestBidirectionalAggregateLinux(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("deferred to batched native Linux validation")
	}
	p, _, bound := emitBidirectional(t, true)
	rt, err := cruntime.ScalarSource(cruntime.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	clang, err := cdriver.Resolve(target.X8664LinuxGNU, "clang")
	if err != nil {
		t.Fatal(err)
	}
	gcc, err := cdriver.Resolve(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]cdriver.Toolchain{{clang, gcc}, {gcc, clang}} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(pair[0].Name+"-"+pair[1].Name+"/"+profile.Name, func(t *testing.T) {
				a, err := cdriver.BuildCrossCompilerWithRuntime(pair[0], pair[1], profile, bound, p.CompanionC, rt)
				if err != nil {
					t.Fatal(err)
				}
				run, err := cdriver.ExecuteLinuxTarget(cdriver.Artifact{Bytes: a.Bytes, Toolchain: pair[0], Profile: profile})
				if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
					t.Fatalf("Linux aggregate crossing: %+v %v", run, err)
				}
			})
		}
	}
}

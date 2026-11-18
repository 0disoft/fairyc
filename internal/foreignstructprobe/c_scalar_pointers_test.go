package foreignstructprobe_test

import (
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

func emitCScalarPointers(t *testing.T, posix bool) (foreignstructprobe.PointerUnit, string) {
	t.Helper()
	p := foreignstructprobe.GenerateCScalarPointers(posix)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(p.FairySource)), source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v diagnostics=%+v unsupported=%+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	profile := cbackend.Stage0WindowsMSVCProfile()
	if posix {
		profile = cbackend.Stage0Profile()
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", profile)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range p.RequiredCFragments {
		if !strings.Contains(unit.Source, fragment) {
			t.Fatalf("missing exact pointer declaration %s", fragment)
		}
	}
	bound, err := p.Bind(unit.Source, unit.Functions)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Bind(unit.Source, nil); err == nil {
		t.Fatal("missing function metadata accepted")
	}
	if _, err := p.Bind("", unit.Functions); err == nil {
		t.Fatal("missing generated signatures accepted")
	}
	return p, bound
}

func TestCScalarPointersEmission(t *testing.T) {
	for _, posix := range []bool{false, true} {
		p, bound := emitCScalarPointers(t, posix)
		want := 15
		if posix {
			want = 16
		}
		if len(p.RequiredCFragments) != want*3 || strings.Contains(p.CompanionC, "%!") {
			t.Fatal("pointer matrix incomplete or malformed")
		}
		if !strings.Contains(bound, "reverse_pointer_slot_long") || !strings.Contains(bound, "sizeof(long)") {
			t.Fatal("target long/provider assertions lost")
		}
		t.Logf("posix=%v: %d scalar targets, three shapes, three value rotations; native execution is separate", posix, want)
	}
}

func TestCScalarPointersWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Windows amd64 MSVC")
	}
	p, bound := emitCScalarPointers(t, false)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := cruntime.ScalarSource(128)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(string(mode)+"/"+profile.Name, func(t *testing.T) {
				artifact, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(tc, profile, mode, bound, p.CompanionC+"\n"+rt)
				if err != nil {
					t.Fatal(err)
				}
				if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "callee.obj" {
					t.Fatal("independent objects missing")
				}
				if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
					t.Fatal(err)
				}
				if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
					t.Fatal(err)
				}
				out, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
				if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
					t.Fatalf("pointer crossing: %+v %v", out, err)
				}
				t.Logf("MSVC %s: 15 C scalar types x 3 value rotations, const/mutable/T** through Fairy and C bridges; C writes observed; exit42 empty streams", tc.Version)
			})
		}
	}
}

// Batched with the other native Linux work; this never attempts local WSL setup.
func TestCScalarPointersLinux(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("deferred to batched native Linux validation")
	}
	p, bound := emitCScalarPointers(t, true)
	rt, err := cruntime.ScalarSource(128)
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
				artifact, err := cdriver.BuildCrossCompilerWithRuntime(pair[0], pair[1], profile, bound, p.CompanionC, rt)
				if err != nil {
					t.Fatal(err)
				}
				out, err := cdriver.ExecuteLinuxTarget(cdriver.Artifact{
					Bytes: artifact.Bytes, Toolchain: artifact.CallerToolchain, Profile: artifact.Profile, ConsumerMode: artifact.ConsumerMode,
				})
				if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
					t.Fatalf("Linux pointer crossing: %+v %v", out, err)
				}
			})
		}
	}
}

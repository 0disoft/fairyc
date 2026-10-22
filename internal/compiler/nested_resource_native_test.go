package compiler_test

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

// Reuse the independent C observer from the Windows test. Only the production
// emitter/runtime perform cleanup and propagate the parent's modified receiver.
func runNestedResourceCleanupNative(t *testing.T, source, probe string) {
	t.Helper()
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("requires Linux GNU or Windows MSVC amd64")
	}
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)
	checked, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !checked.Verified {
		t.Fatalf("nested cleanup source: %v %+v", err, checked.Semantic.Diagnostics)
	}
	emission := cbackend.Stage0Profile()
	compilers := []string{"gcc", "clang"}
	if runtime.GOOS == "windows" {
		emission = cbackend.Stage0WindowsMSVCProfile()
		compilers = []string{"msvc"}
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", emission)
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	support := scalar + "\n#include <string.h>\n" + probe
	for _, cc := range compilers {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				if cc == "msvc" {
					tc, err := cdriver.ResolveWindowsHostMSVC()
					if err != nil {
						t.Fatal(err)
					}
					artifact, err := cdriver.BuildWindowsHost(tc, profile, unit.Source, support)
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(artifact)
					if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("nested cleanup: %+v %v", outcome, err)
					}
					return
				}
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, unit.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				check := func(artifact cdriver.Artifact) {
					t.Helper()
					outcome, err := cdriver.ExecuteLinuxTarget(artifact)
					if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("nested cleanup: %+v %v", outcome, err)
					}
				}
				check(artifact)
				// One ASan+UBSan execution per compiler; do not repeat it at O2.
				if profile == cdriver.LowProfile {
					sanitized, err := cdriver.BuildSanitized(tc, unit.Source, support)
					if err != nil {
						t.Fatal(err)
					}
					check(sanitized)
					t.Log("nested cleanup ASan+UBSan passed")
				}
				t.Log("nested cleanup: 10,2,99 repeated three times; inactive payloads produced no cleanup")
			})
		}
	}
}

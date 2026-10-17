package compiler_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPrivateVectorNestedResourceDropTrace(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
resource Leaf { i32 value; cleanup {} }
resource Box {
    Leaf first; Leaf last;
    cleanup { let Leaf temporary = Leaf { .value = 3 }; }
}
fn i32 main() {
    let Box value = Box { .first = Leaf { .value = 1 }, .last = Leaf { .value = 2 } };
    return 42;
}`)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic.Diagnostics)
	}
	oracle, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || oracle.Kind != interpreter.OutcomeReturn || oracle.Value != 42 || len(oracle.DropTrace) != 4 {
		t.Fatalf("interpreter: %+v %v", oracle, err)
	}
	profile := cbackend.Stage0WindowsMSVCDropTraceProfile()
	unit, err := cbackend.Emit(result.MIR, "main::main", profile)
	if err != nil {
		t.Fatal(err)
	}
	var expected strings.Builder
	for _, event := range oracle.DropTrace {
		found := false
		for _, site := range unit.DropSites {
			if site.ResourceType == event.ResourceType.String() && site.Package == event.Binding.Package &&
				site.Source == event.Binding.Source && site.StartByte == event.Binding.StartByte && site.EndByte == event.Binding.EndByte {
				fmt.Fprintf(&expected, "%d\n", site.ID)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing native drop site for %+v", event)
		}
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	const trace = "\n#include <stdio.h>\nvoid fy_runtime_trace_drop(uint32_t id) { printf(\"%u\\n\", (unsigned)id); }\n"
	for _, nativeProfile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsHost(toolchain, nativeProfile, unit.Source, scalar+trace)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := cdriver.ExecuteWindowsHost(artifact)
		if err != nil || outcome.ExitCode != 42 || strings.ReplaceAll(outcome.Stdout, "\r\n", "\n") != expected.String() || outcome.Stderr != "" {
			t.Fatalf("native %s: %+v %v; expected trace %q", nativeProfile.Name, outcome, err, expected.String())
		}
	}
}

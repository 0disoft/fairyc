package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const generatedFairyScalarFormattingFailureSource = `module main;

import std::bytes as bytes;

fn bool view_equals(ref u8[] left, ref u8[] right) {
    if (left.len != right.len) {
        return false;
    }
    var usize index = 0;
    while (index < left.len) {
        if (left[index] != right[index]) {
            return false;
        }
        index += 1;
    }
    return true;
}

fn i32 exercise_formatting(bytes::Buffer input) {
    var bytes::Buffer buffer = input;
    let usize initial_capacity = bytes::capacity(ref buffer);
    return switch (bytes::append_i32(ref mut buffer, -2147483648)) {
        case .ok(length) : {
            let ref u8[] view = bytes::view(ref buffer);
            if (length == 18 && view_equals(view, b"prefix:-2147483648")) {
                42
            } else {
                4
            }
        }
        case .err(error) : {
            let ref u8[] view = bytes::view(ref buffer);
            let bool preserved = bytes::error_status(error) == 1 &&
                bytes::len(ref buffer) == 7 &&
                bytes::capacity(ref buffer) == initial_capacity &&
                view_equals(view, b"prefix:");
            if (preserved) { 42 } else { 5 }
        }
    };
}

fn i32 main() {
    return switch (bytes::copy(b"prefix:")) {
        case .ok(buffer) : { exercise_formatting(buffer) }
        case .err(_) : { 6 }
    };
}
`

func TestGeneratedFairyScalarFormattingFailureProbeSource(t *testing.T) {
	t.Parallel()

	generated := "int fy_std_buffer_initialize_v0(void);\n" +
		"int fy_std_buffer_append_v0(void);\n" +
		"int fy_std_buffer_drop_v0(void);\n" +
		"int main(void) { return 42; }\n"
	probe, err := generatedFairyScalarFormattingFailureProbeSource(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"#define main fy_generated_scalar_formatting_main",
		generated,
		"fy_std_system_heap_probe_fail_after_allocations_v0(1)",
		"fy_std_system_heap_probe_live_allocations_v0() != 0",
	} {
		if !strings.Contains(probe, fragment) {
			t.Fatalf("probe source lost required fragment %q", fragment)
		}
	}
	if strings.Count(probe, "int main(void)") != 2 {
		t.Fatalf("probe source must contain the renamed generated entry and one test entry: %q", probe)
	}
}

func TestGeneratedFairyScalarFormattingFailureNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("generated Fairy scalar formatting native proof requires windows/amd64")
	}

	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve scalar formatting test path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", "..", ".."))
	images, err := runGeneratedFairyScalarFormattingFailureProbe(repositoryRoot, toolchain)
	if err != nil {
		t.Fatal(err)
	}
	if images != 2 {
		t.Fatalf("generated Fairy scalar formatting failure images = %d; want 2", images)
	}
}

func runGeneratedFairyScalarFormattingFailureProbe(
	repositoryRoot string,
	toolchain cdriver.WindowsHostToolchain,
) (int, error) {
	manifest, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(checkedInStdBytesManifestPath)))
	if err != nil {
		return 0, fmt.Errorf("read generated-Fairy scalar formatting failure manifest: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-scalar-formatting-failure-probe-")
	if err != nil {
		return 0, fmt.Errorf("create generated-Fairy scalar formatting failure workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": []byte(generatedFairyScalarFormattingFailureSource),
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, fmt.Errorf("create generated-Fairy scalar formatting failure path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, fmt.Errorf("write generated-Fairy scalar formatting failure path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, fmt.Errorf("load generated-Fairy scalar formatting failure workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, fmt.Errorf("prepare generated-Fairy scalar formatting failure semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, fmt.Errorf("compile generated-Fairy scalar formatting failure source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow ||
		checked.Semantic.FailedPhase != semantic.PhaseNone ||
		len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return 0, fmt.Errorf(
			"generated-Fairy scalar formatting failure source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v",
			checked.Semantic.CompletedThrough,
			checked.Semantic.FailedPhase,
			checked.Semantic.Diagnostics,
			checked.Semantic.Unsupported,
		)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, fmt.Errorf("resolve generated-Fairy scalar formatting failure root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, fmt.Errorf("emit generated-Fairy scalar formatting failure C: %w", err)
	}
	probeSource, err := generatedFairyScalarFormattingFailureProbeSource(unit.Source)
	if err != nil {
		return 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, err
	}
	return runWindowsSeparateTranslationUnitProbe(
		toolchain,
		"generated-Fairy scalar formatting allocation failure",
		probeSource,
		implementation,
		0,
		cdriver.C11ConsumerMode,
	)
}

func generatedFairyScalarFormattingFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy scalar formatting failure C must contain a main wrapper")
	}
	for _, required := range []string{
		"fy_std_buffer_initialize_v0",
		"fy_std_buffer_append_v0",
		"fy_std_buffer_drop_v0",
	} {
		if !strings.Contains(generated, required) {
			return "", fmt.Errorf("generated-Fairy scalar formatting failure C lost required call %s", required)
		}
	}
	return "#define main fy_generated_scalar_formatting_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    if (fy_generated_scalar_formatting_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    fy_std_system_heap_probe_fail_after_allocations_v0(1);
    if (fy_generated_scalar_formatting_main() != 42) {
        return 12;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 13;
    }
    return 0;
}
`, nil
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const generatedFairyReadAppendFailureSource = `module main;

import std::bytes as bytes;
import std::fs as fs;

@effects(filesystem)
fn i32 close_after_setup_error(fs::File file) {
    return switch (fs::close(file)) {
        case .ok(_) : { 5 }
        case .err(_) : { 6 }
    };
}

@effects(filesystem)
fn i32 exercise_read_append_failure(fs::File input, bytes::Buffer initial) {
    var fs::File file = input;
    var bytes::Buffer buffer = initial;
    let bool failed = switch (fs::read_append(ref mut file, ref mut buffer, 8)) {
        case .ok(_) : { false }
        case .err(error) : {
            fs::read_append_error_file_status(error) == 0 &&
                fs::read_append_error_os_code(error) == 0 &&
                fs::read_append_error_allocation_status(error) == 1
        }
    };
    let bool preserved = bytes::len(ref buffer) == 0 && bytes::capacity(ref buffer) == 0;
    return switch (fs::close(file)) {
        case .ok(_) : { if (failed && preserved) { 42 } else { 7 } }
        case .err(_) : { 8 }
    };
}

@effects(filesystem)
fn i32 main() {
    return switch (fs::create_truncate(c"NUL")) {
        case .ok(file) : { switch (bytes::with_capacity(0)) {
            case .ok(buffer) : { exercise_read_append_failure(file, buffer) }
            case .err(_) : { close_after_setup_error(file) }
        } }
        case .err(_) : { 9 }
    };
}
`

func runGeneratedFairyReadAppendFailureProbe(
	repositoryRoot string,
	sourceRoot string,
	toolchain cdriver.WindowsHostToolchain,
) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdFSManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy read_append failure manifest: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-read-append-failure-probe-")
	if err != nil {
		return 0, 0, fmt.Errorf("create generated-Fairy read_append failure workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": []byte(generatedFairyReadAppendFailureSource),
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, 0, fmt.Errorf("create generated-Fairy read_append failure path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, 0, fmt.Errorf("write generated-Fairy read_append failure path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("load generated-Fairy read_append failure workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, 0, fmt.Errorf("prepare generated-Fairy read_append failure semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, 0, fmt.Errorf("compile generated-Fairy read_append failure source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow ||
		checked.Semantic.FailedPhase != semantic.PhaseNone || len(checked.Semantic.Diagnostics) != 0 ||
		len(checked.Semantic.Unsupported) != 0 {
		return 0, 0, fmt.Errorf(
			"generated-Fairy read_append failure source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v",
			checked.Semantic.CompletedThrough,
			checked.Semantic.FailedPhase,
			checked.Semantic.Diagnostics,
			checked.Semantic.Unsupported,
		)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve generated-Fairy read_append failure root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, 0, fmt.Errorf("emit generated-Fairy read_append failure C: %w", err)
	}
	probeSource, err := generatedFairyReadAppendFailureProbeSource(unit.Source)
	if err != nil {
		return 0, 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, 0, err
	}
	images, err := runWindowsSeparateTranslationUnitProbe(
		toolchain,
		"generated-Fairy read_append allocation failure",
		probeSource,
		implementation,
		0,
		cdriver.C11ConsumerMode,
	)
	if err != nil {
		return images, images, err
	}
	return images, images, nil
}

func generatedFairyReadAppendFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy read_append failure C must contain a main wrapper")
	}
	for _, required := range []string{
		"fy_std_file_open_v0",
		"fy_std_file_read_append_v0",
		"fy_std_file_close_v0",
		"fy_std_buffer_initialize_v0",
		"fy_std_buffer_drop_v0",
	} {
		if !strings.Contains(generated, required) {
			return "", fmt.Errorf("generated-Fairy read_append failure C lost required call %s", required)
		}
	}
	return "#define main fy_generated_read_append_failure_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    if (fy_generated_read_append_failure_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    return 0;
}
`, nil
}

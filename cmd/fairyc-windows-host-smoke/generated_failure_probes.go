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

const generatedFairyRegionFailureSource = `module main;

import std::alloc as alloc;

fn i32 exercise_region_failure(alloc::Arena arena) {
    var alloc::Arena mutable_arena = arena;
    var i32 result = 2;
    region scratch in ref mut mutable_arena {
        result = switch (scratch.place(42)) {
            case .ok(_): { 3 }
            case .err(error): {
                if (alloc::error_status(error) == 1) { 42 } else { 4 }
            }
        };
    }
    return result;
}

fn i32 main() {
    return switch (alloc::create()) {
        case .ok(arena): { exercise_region_failure(arena) }
        case .err(_): { 5 }
    };
}
`

const generatedFairyBufferInitialFailureSource = `module main;

import std::bytes as bytes;

fn i32 discard_buffer(bytes::Buffer value) {
    return 0;
}

fn i32 main() {
    return switch (bytes::with_capacity(16)) {
        case .ok(buffer) : { discard_buffer(buffer) }
        case .err(error) : {
            if (bytes::error_status(error) == 1) { 42 } else { 4 }
        }
    };
}
`

const generatedFairyBufferResizeFailureSource = `module main;

import std::bytes as bytes;

fn i32 discard_buffer(bytes::Buffer value) {
    return 0;
}

fn i32 check_resize_failure(bytes::Buffer input) {
    var bytes::Buffer buffer = input;
    let bool failed = switch (bytes::resize_zeroed(ref mut buffer, 16)) {
        case .ok(_) : { false }
        case .err(error) : { bytes::error_status(error) == 1 }
    };
    let bool preserved = bytes::len(ref buffer) == 0 &&
        bytes::capacity(ref buffer) == 0;
    return if (failed && preserved) { 42 } else { 4 };
}

fn i32 main() {
    return switch (bytes::with_capacity(0)) {
        case .ok(buffer) : { check_resize_failure(buffer) }
        case .err(_) : { 5 }
    };
}
`

const generatedFairyStringAppendFailureSource = `module main;

import std::text as text;

fn i32 exercise_append_failure(text::String input) {
    var text::String value = input;
    return switch (text::append(ref mut value, "Fairy")) {
        case .ok(_) : { 4 }
        case .err(error) : {
            let ref str view = text::string_as_str(ref value);
            let bool ok_status = text::error_status(error) == 1 &&
                text::error_allocation_status(error) == 1;
            let bool preserved = text::string_len(ref value) == 5 &&
                !text::string_is_empty(ref value) &&
                view.len == 5;
            if (ok_status && preserved) { 42 } else { 5 }
        }
    };
}

fn i32 main() {
    return switch (text::copy("Fairy")) {
        case .ok(value) : { exercise_append_failure(value) }
        case .err(_) : { 6 }
    };
}
`

const generatedFairyCStringFailureSource = `module main;

import std::text as text;

fn i32 exercise_cstring_failure() {
    return switch (text::cstring_from_text("Fairy")) {
        case .ok(_) : { 4 }
        case .err(error) : {
            if (text::error_status(error) == 1 && text::error_allocation_status(error) == 1) {
                42
            } else {
                5
            }
        }
    };
}

fn i32 main() {
    return exercise_cstring_failure();
}
`

const generatedFairyStringCopyFailureSource = `module main;

import std::text as text;

fn i32 exercise_copy_failure() {
    return switch (text::copy("Fairy")) {
        case .ok(_) : { 4 }
        case .err(error) : {
            if (text::error_status(error) == 1 && text::error_allocation_status(error) == 1) {
                42
            } else {
                5
            }
        }
    };
}

fn i32 main() {
    return exercise_copy_failure();
}
`

func runGeneratedFairyRegionFailureProbe(repositoryRoot, sourceRoot string, toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInRegionsManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy Region failure manifest: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-region-failure-probe-")
	if err != nil {
		return 0, 0, fmt.Errorf("create generated-Fairy Region failure workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": []byte(generatedFairyRegionFailureSource),
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, 0, fmt.Errorf("create generated-Fairy Region failure path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, 0, fmt.Errorf("write generated-Fairy Region failure path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("load generated-Fairy Region failure workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, 0, fmt.Errorf("prepare generated-Fairy Region failure semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, 0, fmt.Errorf("compile generated-Fairy Region failure source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
		len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return 0, 0, fmt.Errorf("generated-Fairy Region failure source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve generated-Fairy Region failure root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, 0, fmt.Errorf("emit generated-Fairy Region failure C: %w", err)
	}
	probe, err := generatedFairyRegionFailureProbeSource(unit.Source)
	if err != nil {
		return 0, 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, 0, err
	}
	images, err := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy Region failure", probe, implementation, 0, cdriver.C11ConsumerMode)
	if err != nil {
		return images, images * 2, err
	}
	return images, images * 2, nil
}

func generatedFairyRegionFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy Region failure C must contain a main wrapper")
	}
	if !strings.Contains(generated, "fy_std_region_allocate_zeroed_v0") ||
		!strings.Contains(generated, "fy_std_region_reset_v0") ||
		!strings.Contains(generated, "fy_std_arena_drop_v0") {
		return "", fmt.Errorf("generated-Fairy Region failure C lost required placement, reset, or drop calls")
	}
	return "#define main fy_generated_region_failure_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    if (fy_generated_region_failure_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }

    fy_std_system_heap_probe_fail_after_allocations_v0(1);
    if (fy_generated_region_failure_main() != 42) {
        return 12;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 13;
    }
    return 0;
}
`, nil
}

func runGeneratedFairyBufferFailureProbe(repositoryRoot, sourceRoot string, toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdBytesManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy Buffer failure manifest: %w", err)
	}
	totalImages := 0
	totalExecutions := 0
	for _, probe := range []struct {
		name   string
		source string
		wrap   func(string) (string, error)
	}{
		{name: "with_capacity initial allocation failure", source: generatedFairyBufferInitialFailureSource, wrap: generatedFairyBufferInitialFailureProbeSource},
		{name: "resize_zeroed growth failure", source: generatedFairyBufferResizeFailureSource, wrap: generatedFairyBufferResizeFailureProbeSource},
	} {
		workspaceRoot, err := os.MkdirTemp("", "fairyc-buffer-failure-probe-")
		if err != nil {
			return totalImages, totalExecutions, fmt.Errorf("create generated-Fairy Buffer failure workspace: %w", err)
		}
		for name, contents := range map[string][]byte{
			"fairy.toml":  manifest,
			"src/main.fy": []byte(probe.source),
		} {
			destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
				os.RemoveAll(workspaceRoot)
				return totalImages, totalExecutions, fmt.Errorf("create generated-Fairy Buffer failure path %s: %w", name, err)
			}
			if err := os.WriteFile(destination, contents, 0o600); err != nil {
				os.RemoveAll(workspaceRoot)
				return totalImages, totalExecutions, fmt.Errorf("write generated-Fairy Buffer failure path %s: %w", name, err)
			}
		}
		workspace, err := workspaceloader.Load(workspaceRoot)
		os.RemoveAll(workspaceRoot)
		if err != nil {
			return totalImages, totalExecutions, fmt.Errorf("load generated-Fairy Buffer failure workspace: %w", err)
		}
		input, err := workspace.SemanticInput()
		if err != nil {
			return totalImages, totalExecutions, fmt.Errorf("prepare generated-Fairy Buffer failure semantic input: %w", err)
		}
		checked, err := compiler.CheckPackages(input)
		if err != nil {
			return totalImages, totalExecutions, fmt.Errorf("compile generated-Fairy Buffer failure source: %w", err)
		}
		if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
			len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
			return totalImages, totalExecutions, fmt.Errorf("generated-Fairy Buffer failure source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		rootPackage, err := workspace.RootPackageID()
		if err != nil {
			return totalImages, totalExecutions, fmt.Errorf("resolve generated-Fairy Buffer failure root package: %w", err)
		}
		entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
		unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			return totalImages, totalExecutions, fmt.Errorf("emit generated-Fairy Buffer failure C: %w", err)
		}
		probeSource, err := probe.wrap(unit.Source)
		if err != nil {
			return totalImages, totalExecutions, err
		}
		implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
		if err != nil {
			return totalImages, totalExecutions, err
		}
		images, err := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy Buffer "+probe.name, probeSource, implementation, 0, cdriver.C11ConsumerMode)
		if err != nil {
			return totalImages + images, totalExecutions + images, err
		}
		totalImages += images
		totalExecutions += images
	}
	return totalImages, totalExecutions, nil
}

func generatedFairyBufferInitialFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy Buffer initial-failure C must contain a main wrapper")
	}
	if !strings.Contains(generated, "fy_std_buffer_initialize_v0") ||
		!strings.Contains(generated, "fy_std_buffer_drop_v0") {
		return "", fmt.Errorf("generated-Fairy Buffer initial-failure C lost required initialize or drop calls")
	}
	return "#define main fy_generated_buffer_initial_failure_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    if (fy_generated_buffer_initial_failure_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    return 0;
}
`, nil
}

func generatedFairyBufferResizeFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy Buffer resize-failure C must contain a main wrapper")
	}
	if !strings.Contains(generated, "fy_std_buffer_initialize_v0") ||
		!strings.Contains(generated, "fy_std_buffer_resize_zeroed_v0") ||
		!strings.Contains(generated, "fy_std_buffer_drop_v0") {
		return "", fmt.Errorf("generated-Fairy Buffer resize-failure C lost required initialize, resize, or drop calls")
	}
	return "#define main fy_generated_buffer_resize_failure_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    if (fy_generated_buffer_resize_failure_main() != 42) {
        return 12;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 13;
    }
    return 0;
}
`, nil
}

func runGeneratedFairyStringAppendFailureProbe(repositoryRoot, sourceRoot string, toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdTextManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy String append failure manifest: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-string-append-failure-probe-")
	if err != nil {
		return 0, 0, fmt.Errorf("create generated-Fairy String append failure workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": []byte(generatedFairyStringAppendFailureSource),
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, 0, fmt.Errorf("create generated-Fairy String append failure path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, 0, fmt.Errorf("write generated-Fairy String append failure path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("load generated-Fairy String append failure workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, 0, fmt.Errorf("prepare generated-Fairy String append failure semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, 0, fmt.Errorf("compile generated-Fairy String append failure source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
		len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return 0, 0, fmt.Errorf("generated-Fairy String append failure source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve generated-Fairy String append failure root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, 0, fmt.Errorf("emit generated-Fairy String append failure C: %w", err)
	}
	probeSource, err := generatedFairyStringAppendFailureProbeSource(unit.Source)
	if err != nil {
		return 0, 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, 0, err
	}
	images, err := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy String append failure", probeSource, implementation, 0, cdriver.C11ConsumerMode)
	if err != nil {
		return images, images, err
	}
	return images, images, nil
}

func generatedFairyStringAppendFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy String append-failure C must contain a main wrapper")
	}
	if !strings.Contains(generated, "fy_std_string_copy_storage_v0") ||
		!strings.Contains(generated, "fy_std_string_append_storage_v0") ||
		!strings.Contains(generated, "fy_std_string_drop_storage_v0") {
		return "", fmt.Errorf("generated-Fairy String append-failure C lost required copy, append, or drop calls")
	}
	return "#define main fy_generated_string_append_failure_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(1);
    if (fy_generated_string_append_failure_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    return 0;
}
`, nil
}

func runGeneratedFairyCStringFailureProbe(repositoryRoot, sourceRoot string, toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdTextManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy CString failure manifest: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-cstring-failure-probe-")
	if err != nil {
		return 0, 0, fmt.Errorf("create generated-Fairy CString failure workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": []byte(generatedFairyCStringFailureSource),
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, 0, fmt.Errorf("create generated-Fairy CString failure path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, 0, fmt.Errorf("write generated-Fairy CString failure path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("load generated-Fairy CString failure workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, 0, fmt.Errorf("prepare generated-Fairy CString failure semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, 0, fmt.Errorf("compile generated-Fairy CString failure source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
		len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return 0, 0, fmt.Errorf("generated-Fairy CString failure source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve generated-Fairy CString failure root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, 0, fmt.Errorf("emit generated-Fairy CString failure C: %w", err)
	}
	probeSource, err := generatedFairyCStringFailureProbeSource(unit.Source)
	if err != nil {
		return 0, 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, 0, err
	}
	images, err := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy CString failure", probeSource, implementation, 0, cdriver.C11ConsumerMode)
	if err != nil {
		return images, images, err
	}
	return images, images, nil
}

func generatedFairyCStringFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy CString failure C must contain a main wrapper")
	}
	if strings.Count(generated, "fy_std_cstring_from_text_storage_v0(") < 2 ||
		strings.Count(generated, "fy_std_cstring_drop_storage_v0(") < 2 {
		return "", fmt.Errorf("generated-Fairy CString failure C lost required from-text or drop calls (expected prototype and body call per symbol)")
	}
	return "#define main fy_generated_cstring_failure_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    if (fy_generated_cstring_failure_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    return 0;
}
`, nil
}

func runGeneratedFairyStringCopyFailureProbe(repositoryRoot, sourceRoot string, toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdTextManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy String copy failure manifest: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-string-copy-failure-probe-")
	if err != nil {
		return 0, 0, fmt.Errorf("create generated-Fairy String copy failure workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": []byte(generatedFairyStringCopyFailureSource),
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, 0, fmt.Errorf("create generated-Fairy String copy failure path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, 0, fmt.Errorf("write generated-Fairy String copy failure path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("load generated-Fairy String copy failure workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, 0, fmt.Errorf("prepare generated-Fairy String copy failure semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, 0, fmt.Errorf("compile generated-Fairy String copy failure source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
		len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return 0, 0, fmt.Errorf("generated-Fairy String copy failure source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve generated-Fairy String copy failure root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, 0, fmt.Errorf("emit generated-Fairy String copy failure C: %w", err)
	}
	probeSource, err := generatedFairyStringCopyFailureProbeSource(unit.Source)
	if err != nil {
		return 0, 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, 0, err
	}
	images, err := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy String copy failure", probeSource, implementation, 0, cdriver.C11ConsumerMode)
	if err != nil {
		return images, images, err
	}
	return images, images, nil
}

func generatedFairyStringCopyFailureProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy String copy failure C must contain a main wrapper")
	}
	if strings.Count(generated, "fy_std_string_copy_storage_v0(") < 2 ||
		strings.Count(generated, "fy_std_string_drop_storage_v0(") < 2 {
		return "", fmt.Errorf("generated-Fairy String copy failure C lost required copy or drop calls (expected prototype and body call per symbol)")
	}
	return "#define main fy_generated_string_copy_failure_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    if (fy_generated_string_copy_failure_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    return 0;
}
`, nil
}

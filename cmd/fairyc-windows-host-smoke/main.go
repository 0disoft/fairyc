package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/foreignstructprobe"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/windowsabiprobe"
	"github.com/0disoft/fairyc/internal/windowslayoutprobe"
	"github.com/0disoft/fairyc/internal/windowsunwindprobe"
)

const negativeSource = `module negative;

fn i32 main() {
    return -1;
}
`

const divisionByZeroSource = `module division_by_zero;

fn i32 main() {
    return 42 / 0;
}
`

const remainderByZeroSource = `module remainder_by_zero;

fn i32 main() {
    return 42 % 0;
}
`

const signedDivisionOverflowSource = `module signed_division_overflow;

fn i32 main() {
    return (-2147483647 - 1) / -1;
}
`

const integerOverflowSource = `module integer_overflow;

fn i32 main() {
    return 2147483647 + 1;
}
`

const usizeOverflowSource = `module usize_overflow;

fn i32 main() {
    let usize value = 18446744073709551615;
    let usize overflow = value + 1;
    return 42;
}
`

const unsignedDivisionSource = `module unsigned_division;

fn i32 main() {
    let u8 byte_input = 84;
    let u8 byte = byte_input / 2;
    let u32 word_input = 85;
    let u32 word = word_input % 43;
    let usize size_input = 84;
    let usize size = size_input / 2;
    return if (byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
}
`

const unsignedDivisionZeroSource = `module unsigned_division_zero;

fn i32 main() {
    let usize numerator = 42;
    let usize zero = 0;
    let usize value = numerator / zero;
    return 42;
}
`

const unsignedBitwiseShiftSource = `module unsigned_bitwise_shift;

fn i32 main() {
    let u8 byte_input = 84;
    let u8 byte = byte_input >> 1;
    let u8 byte_not_input = 213;
    let u8 byte_not = ~byte_not_input;
    let u32 word_input = 43;
    let u32 word = word_input & 63;
    let u32 word_not_input = 4294967253;
    let u32 word_not = ~word_not_input;
    let u32 word_shift_input = 84;
    let u32 word_shift = word_shift_input >> 1;
    let usize size_input = 43;
    let usize size = (size_input ^ 1) | 0;
    let usize size_not_input = 18446744073709551573;
    let usize size_not = ~size_not_input;
    let usize size_shift_input = 84;
    let usize size_shift = size_shift_input >> 1;
    return if (byte == 42 && byte_not == 42 && word == 43 && word_not == 42 && word_shift == 42 && size == 42 && size_not == 42 && size_shift == 42) { 42 } else { 1 };
}
`

const unsignedInvalidShiftSource = `module unsigned_invalid_shift;

fn i32 main() {
    let usize value = 42;
    let usize amount = 64;
    let usize shifted = value >> amount;
    return 42;
}
`

const leftShiftSource = `module left_shift;

fn i32 main() {
    let i32 signed_input = 21;
    let i32 signed = signed_input << 1;
    let u8 byte_input = 21;
    let u8 byte = byte_input << 1;
    let u32 word_input = 21;
    let u32 word = word_input << 1;
    let usize size_input = 21;
    let usize size = size_input << 1;
    return if (signed == 42 && byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
}
`

const leftShiftOverflowSource = `module left_shift_overflow;

fn i32 main() {
    let i32 value = 1073741824;
    return value << 1;
}
`

const invalidShiftSource = `module invalid_shift;

fn i32 main() {
    return 1 >> -1;
}
`

const boundsSource = `module bounds;

fn i32 main() {
    let u8 value = b"A"[1];
    return 42;
}
`

const arrayBoundsSource = `module array_bounds;

fn i32 main() {
    let i32[2] values = [40, 42];
    return values[2];
}
`

const arrayStoreBoundsSource = `module array_store_bounds;

fn i32 main() {
    var u8[2] values = [40, 41];
    values[2] = 42;
    return 0;
}
`

const sliceBoundsSource = `module slice_bounds;

fn i32 slice(usize start, usize end) {
    let ref u8[] value = b"fairy"[start..end];
    return if (value.is_empty) { 42 } else { 1 };
}

fn i32 main() { return slice(4, 2); }
`

const assertionFailureSource = `module assertion_failure;

fn i32 check(bool condition) {
    assert(condition);
    return 42;
}

fn i32 main() { return check(false); }
`

const checkedInSuccessPath = "examples/windows_smoke/main.fy"
const checkedInModuleMainPath = "examples/run/modules/main.fy"
const checkedInModuleSupportPath = "examples/run/modules/math.fy"
const checkedInWindowsTargetPath = "examples/windows_target/main.fy"
const checkedInWindowsQuickstartPath = "examples/windows_quickstart/main.fy"
const checkedInWindowsOutputPath = "examples/windows_output/main.fy"
const checkedInWindowsOutputSupportPath = "examples/windows_output/windows_console.fy"
const checkedInWindowsFilePath = "examples/windows_file/main.fy"
const checkedInWindowsFileSupportPath = "examples/windows_file/windows_file.fy"
const checkedInOwnedTextPath = "examples/owned_text/main.fy"
const checkedInOwnedTextSupportPath = "examples/owned_text/owned_text.fy"
const checkedInWorkspaceManifestPath = "examples/workspace_quickstart/fairy.toml"
const checkedInWorkspaceMainPath = "examples/workspace_quickstart/src/main.fy"
const checkedInWorkspaceMemberManifestPath = "examples/workspace_quickstart/libs/answer/fairy.toml"
const checkedInWorkspaceMemberSourcePath = "examples/workspace_quickstart/libs/answer/src/answer.fy"
const workspaceManifestPath = "fairy.toml"
const workspaceMainPath = "src/main.fy"
const workspaceMemberManifestPath = "libs/answer/fairy.toml"
const workspaceMemberSourcePath = "libs/answer/src/answer.fy"
const windowsLayoutManifestPath = "spec/fixtures/layout/manifest-windows-msvc-v0.json"
const windowsABIManifestPath = "spec/fixtures/c-abi/manifest-windows-msvc-v0.json"
const subprocessOutputLimit = 64 * 1024

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fairyc Windows host smoke failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	repositoryRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	successSource, err := os.ReadFile(filepath.FromSlash(checkedInSuccessPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows smoke source: %w", err)
	}
	moduleMainSource, err := os.ReadFile(filepath.FromSlash(checkedInModuleMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in multi-source entry: %w", err)
	}
	moduleSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInModuleSupportPath))
	if err != nil {
		return fmt.Errorf("read checked-in multi-source support: %w", err)
	}
	windowsTargetSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsTargetPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows target source: %w", err)
	}
	windowsQuickstartSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsQuickstartPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows quickstart source: %w", err)
	}
	windowsOutputSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsOutputPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows output source: %w", err)
	}
	windowsOutputSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsOutputSupportPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows output support source: %w", err)
	}
	windowsFileSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsFilePath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows File source: %w", err)
	}
	windowsFileSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsFileSupportPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows File support source: %w", err)
	}
	ownedTextSource, err := os.ReadFile(filepath.FromSlash(checkedInOwnedTextPath))
	if err != nil {
		return fmt.Errorf("read checked-in owned-text source: %w", err)
	}
	ownedTextSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInOwnedTextSupportPath))
	if err != nil {
		return fmt.Errorf("read checked-in owned-text support source: %w", err)
	}
	workspaceManifest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in workspace manifest: %w", err)
	}
	workspaceMain, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in workspace entry: %w", err)
	}
	workspaceMemberManifest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMemberManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in workspace member manifest: %w", err)
	}
	workspaceMemberSource, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMemberSourcePath))
	if err != nil {
		return fmt.Errorf("read checked-in workspace member source: %w", err)
	}
	root, err := os.MkdirTemp("", "fairyc-windows-host-smoke-")
	if err != nil {
		return fmt.Errorf("create temporary source root: %w", err)
	}
	defer os.RemoveAll(root)
	for name, source := range map[string]string{
		workspaceManifestPath:             string(workspaceManifest),
		workspaceMainPath:                 string(workspaceMain),
		workspaceMemberManifestPath:       string(workspaceMemberManifest),
		workspaceMemberSourcePath:         string(workspaceMemberSource),
		checkedInSuccessPath:              string(successSource),
		checkedInModuleMainPath:           string(moduleMainSource),
		checkedInModuleSupportPath:        string(moduleSupportSource),
		checkedInWindowsTargetPath:        string(windowsTargetSource),
		checkedInWindowsQuickstartPath:    string(windowsQuickstartSource),
		checkedInWindowsOutputPath:        string(windowsOutputSource),
		checkedInWindowsOutputSupportPath: string(windowsOutputSupportSource),
		checkedInWindowsFilePath:          string(windowsFileSource),
		checkedInWindowsFileSupportPath:   string(windowsFileSupportSource),
		checkedInOwnedTextPath:            string(ownedTextSource),
		checkedInOwnedTextSupportPath:     string(ownedTextSupportSource),
		"negative.fy":                     negativeSource,
		"division_by_zero.fy":             divisionByZeroSource,
		"remainder_by_zero.fy":            remainderByZeroSource,
		"signed_division_overflow.fy":     signedDivisionOverflowSource,
		"integer_overflow.fy":             integerOverflowSource,
		"usize_overflow.fy":               usizeOverflowSource,
		"unsigned_division.fy":            unsignedDivisionSource,
		"unsigned_division_zero.fy":       unsignedDivisionZeroSource,
		"unsigned_bitwise_shift.fy":       unsignedBitwiseShiftSource,
		"unsigned_invalid_shift.fy":       unsignedInvalidShiftSource,
		"left_shift.fy":                   leftShiftSource,
		"left_shift_overflow.fy":          leftShiftOverflowSource,
		"invalid_shift.fy":                invalidShiftSource,
		"bounds.fy":                       boundsSource,
		"array_bounds.fy":                 arrayBoundsSource,
		"array_store_bounds.fy":           arrayStoreBoundsSource,
		"slice_bounds.fy":                 sliceBoundsSource,
		"assertion_failure.fy":            assertionFailureSource,
	} {
		destination := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return fmt.Errorf("create %s smoke source directory: %w", name, err)
		}
		if err := os.WriteFile(destination, []byte(source), 0o600); err != nil {
			return fmt.Errorf("write %s smoke source: %w", name, err)
		}
	}

	var buildStdout bytes.Buffer
	var buildStderr bytes.Buffer
	buildCode := cli.Run([]string{
		"experimental", "build-host", checkedInModuleMainPath,
		"--source", checkedInModuleSupportPath,
		"--profile", "optimized", "--output", "build/main.exe",
	}, root, &buildStdout, &buildStderr)
	if buildCode != cli.ExitSuccess || buildStderr.Len() != 0 || !strings.HasPrefix(buildStdout.String(), "built: build/main.exe (windows/amd64 host-smoke, msvc ") {
		return fmt.Errorf("build-host returned code %d, stdout %q, stderr %q", buildCode, buildStdout.String(), buildStderr.String())
	}
	artifactPath := filepath.Join(root, "build", "main.exe")
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("read build-host artifact: %w", err)
	}
	peReport, err := pecoff.InspectExecutable(artifact)
	if err != nil {
		return fmt.Errorf("inspect build-host PE executable image: %w", err)
	}
	unwindReport, err := pecoff.InspectAMD64Unwind(artifact)
	if err != nil {
		return fmt.Errorf("inspect build-host Microsoft x64 unwind data: %w", err)
	}
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return err
	}

	targetBuildExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/windows-target-%s.exe", profile)
		var targetBuildStdout bytes.Buffer
		var targetBuildStderr bytes.Buffer
		targetBuildCode := cli.Run([]string{
			"experimental", "build-target", checkedInWindowsTargetPath,
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}, root, &targetBuildStdout, &targetBuildStderr)
		if targetBuildCode != cli.ExitSuccess || targetBuildStderr.Len() != 0 || !strings.HasPrefix(targetBuildStdout.String(), "built: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("build-target %s returned code %d, stdout %q, stderr %q", profile, targetBuildCode, targetBuildStdout.String(), targetBuildStderr.String())
		}
		targetArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		targetArtifact, err := os.ReadFile(targetArtifactPath)
		if err != nil {
			return fmt.Errorf("read build-target %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(targetArtifact); err != nil {
			return fmt.Errorf("inspect build-target %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(targetArtifact); err != nil {
			return fmt.Errorf("inspect build-target %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(targetArtifactPath, 42); err != nil {
			return fmt.Errorf("execute build-target %s artifact: %w", profile, err)
		}
		targetBuildExecutions++
	}
	workspaceBuildExecutions := 0
	for index, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/workspace-%s.exe", profile)
		arguments := []string{
			"experimental", "build-workspace",
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}
		if index != 0 {
			arguments = append(arguments, "--locked")
		}
		var workspaceBuildStdout bytes.Buffer
		var workspaceBuildStderr bytes.Buffer
		workspaceBuildCode := cli.Run(arguments, root, &workspaceBuildStdout, &workspaceBuildStderr)
		if workspaceBuildCode != cli.ExitSuccess || workspaceBuildStderr.Len() != 0 || !strings.HasPrefix(workspaceBuildStdout.String(), "built workspace: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("build-workspace %s returned code %d, stdout %q, stderr %q", profile, workspaceBuildCode, workspaceBuildStdout.String(), workspaceBuildStderr.String())
		}
		workspaceArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		workspaceArtifact, err := os.ReadFile(workspaceArtifactPath)
		if err != nil {
			return fmt.Errorf("read build-workspace %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(workspaceArtifact); err != nil {
			return fmt.Errorf("inspect build-workspace %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(workspaceArtifact); err != nil {
			return fmt.Errorf("inspect build-workspace %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(workspaceArtifactPath, 42); err != nil {
			return fmt.Errorf("execute build-workspace %s artifact: %w", profile, err)
		}
		workspaceBuildExecutions++
	}
	windowsFileSourceExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/windows-file-%s.exe", profile)
		var fileBuildStdout bytes.Buffer
		var fileBuildStderr bytes.Buffer
		fileBuildCode := cli.Run([]string{
			"experimental", "build-target", checkedInWindowsFilePath,
			"--source", checkedInWindowsFileSupportPath,
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}, root, &fileBuildStdout, &fileBuildStderr)
		if fileBuildCode != cli.ExitSuccess || fileBuildStderr.Len() != 0 || !strings.HasPrefix(fileBuildStdout.String(), "built: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("Windows File build-target %s returned code %d, stdout %q, stderr %q", profile, fileBuildCode, fileBuildStdout.String(), fileBuildStderr.String())
		}
		fileArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		fileArtifact, err := os.ReadFile(fileArtifactPath)
		if err != nil {
			return fmt.Errorf("read Windows File build-target %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(fileArtifact); err != nil {
			return fmt.Errorf("inspect Windows File build-target %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(fileArtifact); err != nil {
			return fmt.Errorf("inspect Windows File build-target %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(fileArtifactPath, 42); err != nil {
			return fmt.Errorf("execute Windows File build-target %s artifact: %w", profile, err)
		}
		windowsFileSourceExecutions++
	}
	ownedTextSourceExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/owned-text-%s.exe", profile)
		var ownedTextBuildStdout bytes.Buffer
		var ownedTextBuildStderr bytes.Buffer
		ownedTextBuildCode := cli.Run([]string{
			"experimental", "build-target", checkedInOwnedTextPath,
			"--source", checkedInOwnedTextSupportPath,
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}, root, &ownedTextBuildStdout, &ownedTextBuildStderr)
		if ownedTextBuildCode != cli.ExitSuccess || ownedTextBuildStderr.Len() != 0 || !strings.HasPrefix(ownedTextBuildStdout.String(), "built: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("owned-text build-target %s returned code %d, stdout %q, stderr %q", profile, ownedTextBuildCode, ownedTextBuildStdout.String(), ownedTextBuildStderr.String())
		}
		ownedTextArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		ownedTextArtifact, err := os.ReadFile(ownedTextArtifactPath)
		if err != nil {
			return fmt.Errorf("read owned-text build-target %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(ownedTextArtifact); err != nil {
			return fmt.Errorf("inspect owned-text build-target %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(ownedTextArtifact); err != nil {
			return fmt.Errorf("inspect owned-text build-target %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(ownedTextArtifactPath, 42); err != nil {
			return fmt.Errorf("execute owned-text build-target %s artifact: %w", profile, err)
		}
		ownedTextSourceExecutions++
	}
	targetRunExecutions := 0
	var targetRunStdout bytes.Buffer
	var targetRunStderr bytes.Buffer
	targetRunCode := cli.Run([]string{
		"experimental", "run-target", checkedInWindowsTargetPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &targetRunStdout, &targetRunStderr)
	if targetRunCode != cli.ExitSuccess || targetRunStderr.Len() != 0 || !strings.HasPrefix(targetRunStdout.String(), "native result: 42 (x86_64-windows-msvc, msvc ") {
		return fmt.Errorf("run-target returned code %d, stdout %q, stderr %q", targetRunCode, targetRunStdout.String(), targetRunStderr.String())
	}
	targetRunExecutions++
	workspaceRunExecutions := 0
	var workspaceRunStdout bytes.Buffer
	var workspaceRunStderr bytes.Buffer
	workspaceRunCode := cli.Run([]string{
		"experimental", "run-workspace", "--locked",
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &workspaceRunStdout, &workspaceRunStderr)
	if workspaceRunCode != cli.ExitSuccess || workspaceRunStderr.Len() != 0 || !strings.HasPrefix(workspaceRunStdout.String(), "native workspace result: 42 (x86_64-windows-msvc, msvc ") {
		return fmt.Errorf("run-workspace returned code %d, stdout %q, stderr %q", workspaceRunCode, workspaceRunStdout.String(), workspaceRunStderr.String())
	}
	workspaceRunExecutions++
	targetExecExecutions := 0
	var targetExecStdout bytes.Buffer
	var targetExecStderr bytes.Buffer
	targetExecCode := cli.Run([]string{
		"experimental", "exec-target", checkedInWindowsOutputPath,
		"--source", checkedInWindowsOutputSupportPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &targetExecStdout, &targetExecStderr)
	if targetExecCode != 42 || targetExecStdout.String() != "Fairy C says hi\n" || targetExecStderr.Len() != 0 {
		return fmt.Errorf("exec-target returned code %d, stdout %q, stderr %q", targetExecCode, targetExecStdout.String(), targetExecStderr.String())
	}
	targetExecExecutions++
	standaloneCompilerBytes, err := runStandaloneCompiler(repositoryRoot, root)
	if err != nil {
		return err
	}
	targetRunExecutions++
	targetExecExecutions++
	workspaceRunExecutions++

	for _, testCase := range []struct {
		name         string
		source       string
		support      string
		profile      string
		code         int
		stdoutPrefix string
		stderr       string
	}{
		{name: "success low", source: checkedInSuccessPath, profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "multi-source success optimized", source: checkedInModuleMainPath, support: checkedInModuleSupportPath, profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "byte-view iteration low", source: checkedInWindowsQuickstartPath, profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "byte-view iteration optimized", source: checkedInWindowsQuickstartPath, profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "negative low", source: "negative.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "negative optimized", source: "negative.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "division by zero low", source: "division_by_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "division by zero optimized", source: "division_by_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "remainder by zero low", source: "remainder_by_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "remainder by zero optimized", source: "remainder_by_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "signed division overflow low", source: "signed_division_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: signed-division-overflow\n"},
		{name: "signed division overflow optimized", source: "signed_division_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: signed-division-overflow\n"},
		{name: "integer overflow low", source: "integer_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "integer overflow optimized", source: "integer_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "usize overflow low", source: "usize_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "usize overflow optimized", source: "usize_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "unsigned division low", source: "unsigned_division.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned division optimized", source: "unsigned_division.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned division by zero low", source: "unsigned_division_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "unsigned division by zero optimized", source: "unsigned_division_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "unsigned bitwise shift low", source: "unsigned_bitwise_shift.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned bitwise shift optimized", source: "unsigned_bitwise_shift.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned invalid shift low", source: "unsigned_invalid_shift.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "unsigned invalid shift optimized", source: "unsigned_invalid_shift.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "left shift low", source: "left_shift.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "left shift optimized", source: "left_shift.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "left shift overflow low", source: "left_shift_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "left shift overflow optimized", source: "left_shift_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "invalid shift low", source: "invalid_shift.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "invalid shift optimized", source: "invalid_shift.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "byte-view bounds low", source: "bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "byte-view bounds optimized", source: "bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array bounds low", source: "array_bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array bounds optimized", source: "array_bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array store bounds low", source: "array_store_bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array store bounds optimized", source: "array_store_bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "byte-view slice bounds low", source: "slice_bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "byte-view slice bounds optimized", source: "slice_bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "assertion failure low", source: "assertion_failure.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: assertion-failed\n"},
		{name: "assertion failure optimized", source: "assertion_failure.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: assertion-failed\n"},
	} {
		if err := runHostCase(root, testCase.name, testCase.source, testCase.support, testCase.profile, testCase.code, testCase.stdoutPrefix, testCase.stderr); err != nil {
			return err
		}
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return fmt.Errorf("resolve Windows MSVC probe toolchain: %w", err)
	}
	windowsFileExecutions, err := runWindowsFileProbe(toolchain)
	if err != nil {
		return err
	}
	bufferProbeExecutions, err := runBufferFailureProbe(toolchain)
	if err != nil {
		return err
	}
	ownedTextProbeExecutions, err := runOwnedTextStorageProbe(toolchain)
	if err != nil {
		return err
	}
	arenaProbeExecutions, err := runArenaFailureProbe(toolchain)
	if err != nil {
		return err
	}
	layoutExecutions, layoutCases, err := runWindowsLayoutProbe(toolchain)
	if err != nil {
		return err
	}
	abiExecutions, abiCases, err := runWindowsABIProbe(toolchain)
	if err != nil {
		return err
	}
	structResultExecutions, err := runFairyStructResultProbe(toolchain)
	if err != nil {
		return err
	}
	unwindExecutions, unwindCases, err := runWindowsUnwindProbe(toolchain)
	if err != nil {
		return err
	}
	inspectedPEImages := 1 + targetBuildExecutions + workspaceBuildExecutions + windowsFileSourceExecutions + ownedTextSourceExecutions + targetRunExecutions + workspaceRunExecutions + targetExecExecutions + windowsFileExecutions + bufferProbeExecutions + ownedTextProbeExecutions + arenaProbeExecutions + layoutExecutions + abiExecutions + structResultExecutions + unwindExecutions
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes with %d validated sections, %d runtime-function records, and %d version-2 unwind records, %d source-level Windows target builds, %d package-workspace builds, %d package-workspace runs, %d source-level safe File lifecycle builds, %d source-level safe owned-text lifecycle builds, %d direct run-target executions, and %d exec-target executions checked LLP64 c.long emission, move-only pointer-width resource transport, closed SystemHeap, Buffer, Arena/Region, String/CString/CStr, UTF-8 crossings, safe borrowed-view internal calls, byte lengths, typed scalar comparisons, scalar fixed-array calls, checked reads and mutable indexed stores, checked byte indexing and explicit or omitted-bound slicing, always-checked assertions, value-preserving integer casts, checked unsigned arithmetic, division, remainder, bitwise operations, and left/right shifts, exact stdout bytes, and native result 42, including one %d-byte standalone Go-built fairyc.exe quickstart, output, and locked workspace execution, 40 run-host outcomes covered fixed arrays, byte-view iteration and slicing, checked indexed stores, assertions in low and optimized profiles, usize arithmetic and overflow, unsigned division and zero traps, typed bitwise operations, checked shift-width and left-shift overflow traps, signed results, and six trap categories, Windows File partial I/O and consuming-close passed %d separate-translation-unit MSVC executions, Buffer initialization, reserve, self-overlap append, and zeroed-resize allocation rollback passed %d separate-translation-unit MSVC executions, owned-text opaque-storage failure preservation, forced allocation rollback, and repeated drop passed %d separate-translation-unit MSVC executions, Arena zeroing, two-stage allocation rollback, stale Region rejection, and active-drop preservation passed %d separate-translation-unit MSVC executions, %d LLP64 layout cases passed %d MSVC probe executions, %d Microsoft x64 ABI cases passed %d separate-translation-unit MSVC probe executions, generated-Fairy small and large struct results passed %d separate-translation-unit MSVC executions across C11 and C17 consumer modes, %d Windows x64 unwind cases passed %d actual exception-dispatch and virtual-unwind executions, %d PE32+ target or probe images passed structural and unwind inspection, and every MSVC compilation object passed COFF inspection across low/optimized with msvc %s\n", len(artifact), len(peReport.Sections), unwindReport.RuntimeFunctions, unwindReport.Version2Infos, targetBuildExecutions, workspaceBuildExecutions, workspaceRunExecutions, windowsFileSourceExecutions, ownedTextSourceExecutions, targetRunExecutions, targetExecExecutions, standaloneCompilerBytes, windowsFileExecutions, bufferProbeExecutions, ownedTextProbeExecutions, arenaProbeExecutions, layoutCases, layoutExecutions, abiCases, abiExecutions, structResultExecutions, unwindCases, unwindExecutions, inspectedPEImages, toolchain.Version)
	return nil
}

func runStandaloneCompiler(repositoryRoot, sourceRoot string) (int64, error) {
	temporaryRoot, err := os.MkdirTemp("", "fairyc-standalone-compiler-smoke-")
	if err != nil {
		return 0, fmt.Errorf("create standalone compiler directory: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	executablePath := filepath.Join(temporaryRoot, "fairyc.exe")
	buildContext, cancelBuild := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelBuild()
	build := exec.CommandContext(buildContext, "go", "build", "-mod=readonly", "-trimpath", "-o", executablePath, "./cmd/fairyc")
	build.Dir = repositoryRoot
	build.Stdin = nil
	var buildStdout boundedBuffer
	var buildStderr boundedBuffer
	buildStdout.limit = subprocessOutputLimit
	buildStderr.limit = subprocessOutputLimit
	build.Stdout = &buildStdout
	build.Stderr = &buildStderr
	if err := build.Run(); err != nil {
		if errors.Is(buildContext.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("standalone compiler build exceeded timeout")
		}
		return 0, fmt.Errorf("build standalone compiler: %w; stdout %q; stderr %q", err, buildStdout.String(), buildStderr.String())
	}
	if buildStdout.Len() != 0 || buildStderr.Len() != 0 {
		return 0, fmt.Errorf("standalone compiler build produced unexpected output: stdout %q; stderr %q", buildStdout.String(), buildStderr.String())
	}
	info, err := os.Stat(executablePath)
	if err != nil {
		return 0, fmt.Errorf("stat standalone compiler: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return 0, fmt.Errorf("standalone compiler is not a nonempty regular file")
	}

	helpCode, helpStdout, helpStderr, err := runStandaloneCommand(sourceRoot, executablePath, "--help")
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler help: %w", err)
	}
	if helpCode != cli.ExitSuccess || helpStderr != "" ||
		!strings.Contains(helpStdout, "fairyc experimental run-target") ||
		!strings.Contains(helpStdout, "fairyc experimental exec-target") ||
		!strings.Contains(helpStdout, "fairyc experimental build-workspace") ||
		!strings.Contains(helpStdout, "fairyc experimental run-workspace") {
		return 0, fmt.Errorf("standalone compiler help returned code %d, stdout %q, stderr %q", helpCode, helpStdout, helpStderr)
	}

	runCode, runStdout, runStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "run-target", checkedInWindowsQuickstartPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler quickstart: %w", err)
	}
	if runCode != cli.ExitSuccess || runStderr != "" || !strings.HasPrefix(runStdout, "native result: 42 (x86_64-windows-msvc, msvc ") {
		return 0, fmt.Errorf("standalone compiler run-target returned code %d, stdout %q, stderr %q", runCode, runStdout, runStderr)
	}

	execCode, execStdout, execStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "exec-target", checkedInWindowsOutputPath,
		"--source", checkedInWindowsOutputSupportPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler output program: %w", err)
	}
	if execCode != 42 || execStdout != "Fairy C says hi\n" || execStderr != "" {
		return 0, fmt.Errorf("standalone compiler exec-target returned code %d, stdout %q, stderr %q", execCode, execStdout, execStderr)
	}

	workspaceCode, workspaceStdout, workspaceStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "run-workspace", "--locked",
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler workspace: %w", err)
	}
	if workspaceCode != cli.ExitSuccess || workspaceStderr != "" || !strings.HasPrefix(workspaceStdout, "native workspace result: 42 (x86_64-windows-msvc, msvc ") {
		return 0, fmt.Errorf("standalone compiler run-workspace returned code %d, stdout %q, stderr %q", workspaceCode, workspaceStdout, workspaceStderr)
	}
	return info.Size(), nil
}

func runStandaloneCommand(directory, executablePath string, arguments ...string) (int, string, string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath, arguments...)
	command.Dir = directory
	command.Stdin = nil
	var stdout boundedBuffer
	var stderr boundedBuffer
	stdout.limit = subprocessOutputLimit
	stderr.limit = subprocessOutputLimit
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return -1, stdout.String(), stderr.String(), fmt.Errorf("command exceeded timeout")
	}
	if err == nil {
		return 0, stdout.String(), stderr.String(), nil
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return -1, stdout.String(), stderr.String(), err
	}
	return exitError.ExitCode(), stdout.String(), stderr.String(), nil
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	originalLength := len(value)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = buffer.buffer.Write(value)
	}
	if originalLength > remaining {
		buffer.truncated = true
	}
	return originalLength, nil
}

func (buffer *boundedBuffer) Len() int {
	return buffer.buffer.Len()
}

func (buffer *boundedBuffer) String() string {
	if buffer.truncated {
		return buffer.buffer.String() + "\n[output truncated]"
	}
	return buffer.buffer.String()
}

func runWindowsUnwindProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit := windowsunwindprobe.Generate()
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCUnwindProbe(toolchain, profile, unit.CatcherSource, unit.FrameSource, unit.RaiserSource)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC unwind probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 3 || artifact.Objects[0].Name != "catcher.obj" || artifact.Objects[1].Name != "frame.obj" || artifact.Objects[2].Name != "raiser.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind probe PE executable image with %s: %w", profile.Name, err)
		}
		report, err := pecoff.InspectAMD64Unwind(artifact.Bytes)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind probe unwind data with %s: %w", profile.Name, err)
		}
		if report.HandlerInfos == 0 {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s has no handler-bearing unwind records", profile.Name)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCUnwindProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC unwind probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}

func runWindowsFileProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsFileSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows File implementation: %w", err)
	}
	probe, err := cstdlib.WindowsFileProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows File probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "Windows File", probe, implementation, 0, cdriver.C11ConsumerMode)
}

func runOwnedTextStorageProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsMSVCOwnedTextProbeImplementationSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows MSVC owned-text implementation: %w", err)
	}
	probe, err := cstdlib.OwnedTextProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load owned-text storage probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "owned-text storage", probe, implementation, 0, cdriver.C11ConsumerMode)
}

func runBufferFailureProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsMSVCBufferProbeImplementationSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows MSVC Buffer implementation: %w", err)
	}
	probe, err := cstdlib.BufferProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load Buffer failure probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "Buffer failure", probe, implementation, 0, cdriver.C11ConsumerMode)
}

func runArenaFailureProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsMSVCArenaProbeImplementationSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows MSVC Arena implementation: %w", err)
	}
	probe, err := cstdlib.ArenaProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load Arena failure probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "Arena failure", probe, implementation, 0, cdriver.C11ConsumerMode)
}

func runFairyStructResultProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	probe := foreignstructprobe.Generate()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(probe.FairySource)), source.DefaultLimits())
	if err != nil {
		return 0, fmt.Errorf("parse generated-Fairy struct-result source: %w", err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return 0, fmt.Errorf("generated-Fairy struct-result source is not parser-valid: %+v", parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil {
		return 0, fmt.Errorf("compile generated-Fairy struct-result source: %w", err)
	}
	semanticResult := checked.Semantic
	if semanticResult.CompletedThrough != semantic.PhaseFlow || semanticResult.FailedPhase != semantic.PhaseNone ||
		len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return 0, fmt.Errorf("generated-Fairy struct-result source did not complete Flow: phase=%q failed=%q diagnostics=%+v unsupported=%+v", semanticResult.CompletedThrough, semanticResult.FailedPhase, semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	if !checked.Verified {
		return 0, fmt.Errorf("generated-Fairy struct-result source did not produce verified MIR")
	}
	caller, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, fmt.Errorf("emit generated-Fairy Windows struct-result C: %w", err)
	}
	for _, required := range probe.RequiredCFragments {
		if !strings.Contains(caller.Source, required) {
			return 0, fmt.Errorf("generated-Fairy Windows struct-result C lost required fragment %q", required)
		}
	}
	executions := 0
	for _, consumerMode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
		count, runErr := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy struct-result", caller.Source, probe.CompanionC, probe.ExpectedExit, consumerMode)
		if runErr != nil {
			return executions, runErr
		}
		executions += count
	}
	return executions, nil
}

func runWindowsSeparateTranslationUnitProbe(
	toolchain cdriver.WindowsHostToolchain,
	name string,
	probe string,
	implementation string,
	expectedExit int,
	consumerMode cdriver.ConsumerMode,
) (int, error) {
	if name == "" || probe == "" || implementation == "" {
		return 0, fmt.Errorf("Windows separate-translation-unit probe identity and sources must be non-empty")
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(toolchain, profile, consumerMode, probe, implementation)
		if err != nil {
			return executions, fmt.Errorf("build %s probe with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		if artifact.ConsumerMode != consumerMode {
			return executions, fmt.Errorf("%s probe with %s/%s lost its consumer-mode identity", name, consumerMode, profile.Name)
		}
		if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "callee.obj" {
			return executions, fmt.Errorf("%s probe with %s/%s returned unexpected COFF object receipts", name, consumerMode, profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect %s probe PE with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect %s probe unwind data with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
		if err != nil {
			return executions, fmt.Errorf("execute %s probe with %s/%s: %w", name, consumerMode, profile.Name, err)
		}
		if outcome.ExitCode != int32(expectedExit) || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, fmt.Errorf("%s probe with %s/%s returned %d, stdout %q, and stderr %q", name, consumerMode, profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, nil
}

func runWindowsLayoutProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit, err := windowslayoutprobe.Generate(filepath.FromSlash(windowsLayoutManifestPath))
	if err != nil {
		return 0, 0, fmt.Errorf("generate Windows MSVC layout probe: %w", err)
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, unit.Source)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 1 || artifact.Objects[0].Name != "probe.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC layout probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC layout probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC layout probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC layout probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}

func runWindowsABIProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit, err := windowsabiprobe.Generate(filepath.FromSlash(windowsABIManifestPath))
	if err != nil {
		return 0, 0, fmt.Errorf("generate Windows MSVC ABI probe: %w", err)
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCABIProbe(toolchain, profile, unit.CallerSource, unit.CalleeSource)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC ABI probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "callee.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC ABI probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC ABI probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC ABI probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC ABI probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC ABI probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}

func runHostCase(root, name, source, support, profile string, wantCode int, wantStdoutPrefix, wantStderr string) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	arguments := []string{"experimental", "run-host", source}
	if support != "" {
		arguments = append(arguments, "--source", support)
	}
	arguments = append(arguments, "--profile", profile)
	code := cli.Run(arguments, root, &stdout, &stderr)
	stdoutMatches := stdout.Len() == 0
	if wantStdoutPrefix != "" {
		stdoutMatches = strings.HasPrefix(stdout.String(), wantStdoutPrefix)
	}
	if code != wantCode || stderr.String() != wantStderr || !stdoutMatches {
		return fmt.Errorf("%s run-host returned code %d, stdout %q, stderr %q", name, code, stdout.String(), stderr.String())
	}
	return nil
}

func executeExpectResult(path string, expected int32) error {
	contextValue, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, path)
	command.Dir = filepath.Dir(path)
	command.Stdin = nil
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("build-host artifact exceeded execution timeout")
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return fmt.Errorf("build-host artifact execution failed: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	actual := int32(uint32(exitCode))
	if actual != expected || output.Len() != 0 {
		exitCode := -1
		if command.ProcessState != nil {
			exitCode = command.ProcessState.ExitCode()
		}
		return fmt.Errorf("build-host artifact returned error %v, exit %d (i32 %d), output %q", err, exitCode, actual, output.String())
	}
	return nil
}

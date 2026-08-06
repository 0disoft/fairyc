package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/pecoff"
)

const negativeSource = `module negative;

fn i32 main() {
    return -1;
}
`

const generatedFairyProcessArgsSource = `module main;

import std::process as process;

fn i32 main() {
    var u8[8] buffer = [0, 0, 0, 0, 0, 0, 0, 0];
    let bool called = switch (process::arg(0, ref mut buffer)) {
        case .ok(read) : { read <= buffer.len && process::arg_count() >= 1 }
        case .err(_) : { false }
    };
    return if (called) { 42 } else { 1 };
}
`

var processArgsDecodeRoundTripArguments = []string{"Fairy", "a b\"c\\d\\", "한글λ", ""}

const windowsProcessArgsDecodeProbeMain = `
#include <stdint.h>
#include <process.h>
#include <string.h>

int main(void) {
    const wchar_t *arguments[4] = { L"probe", L"Fairy", L"bad\xD800" L"arg", L"" };
    uint8_t buffer[8] = {
        UINT8_C(0x11), UINT8_C(0x22), UINT8_C(0x33), UINT8_C(0x44),
        UINT8_C(0x55), UINT8_C(0x66), UINT8_C(0x77), UINT8_C(0x88)
    };
    struct fy_std_process_args_result_v0 result;

    result = fy_std_process_arg_wide_v0(1, 4, arguments, buffer, sizeof(buffer));
    if (result.status != FY_STD_PROCESS_ARGS_OK_V0 || result.needed != (size_t)5 ||
        buffer[0] != UINT8_C('F') || buffer[1] != UINT8_C('a') || buffer[2] != UINT8_C('i') ||
        buffer[3] != UINT8_C('r') || buffer[4] != UINT8_C('y')) {
        return 1;
    }
    result = fy_std_process_arg_wide_v0(2, 4, arguments, buffer, sizeof(buffer));
    if (result.status != FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0 || result.needed != (size_t)0 ||
        buffer[0] != UINT8_C('F') || buffer[1] != UINT8_C('a') || buffer[2] != UINT8_C('i') ||
        buffer[3] != UINT8_C('r') || buffer[4] != UINT8_C('y') ||
        buffer[5] != UINT8_C(0x66) || buffer[6] != UINT8_C(0x77) || buffer[7] != UINT8_C(0x88)) {
        return 2;
    }
    result = fy_std_process_arg_wide_v0(3, 4, arguments, buffer, sizeof(buffer));
    if (result.status != FY_STD_PROCESS_ARGS_OK_V0 || result.needed != (size_t)0) {
        return 3;
    }
    result = fy_std_process_arg_wide_v0(4, 4, arguments, buffer, sizeof(buffer));
    if (result.status != FY_STD_PROCESS_ARGS_INDEX_OUT_OF_RANGE_V0) {
        return 4;
    }
    /* Process-global UCRT argv under int main(void) with static /MT. The probe
       runs once with no explicit program arguments (argc == 1) and once with four
       explicit arguments (argc == 5). The narrow mainCRTStartup policy must leave
       *__p___wargv() NULL; _configure_wide_argv then populates it, which is the
       minimal safe fix under test here. */
    int *process_argc = __p___argc();
    wchar_t ***process_wargv = __p___wargv();
    if (process_argc == NULL || (*process_argc != 1 && *process_argc != 5)) {
        return 5;
    }
    if (process_wargv == NULL) {
        return 9;
    }
    if (*process_wargv != NULL) {
        return 10;
    }
    if (_configure_wide_argv(_crt_argv_unexpanded_arguments) != 0) {
        return 11;
    }
    if (*process_wargv == NULL) {
        return 12;
    }
    result = fy_std_process_arg_v0(0, buffer, sizeof(buffer));
    if (result.status == FY_STD_PROCESS_ARGS_INDEX_OUT_OF_RANGE_V0 ||
        result.status == FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0) {
        return 6;
    }
    if (result.status != FY_STD_PROCESS_ARGS_OK_V0 &&
        result.status != FY_STD_PROCESS_ARGS_BUFFER_TOO_SMALL_V0) {
        return 7;
    }
    if (result.needed == (size_t)0) {
        return 8;
    }
    if (*process_argc == 5) {
        uint8_t round_buffer[64] = { 0 };
        static const uint8_t expected[4][8] = {
            { UINT8_C('F'), UINT8_C('a'), UINT8_C('i'), UINT8_C('r'), UINT8_C('y'), 0, 0, 0 },
            { UINT8_C('a'), UINT8_C(' '), UINT8_C('b'), UINT8_C('"'), UINT8_C('c'), UINT8_C('\\'), UINT8_C('d'), UINT8_C('\\') },
            { UINT8_C(0xED), UINT8_C(0x95), UINT8_C(0x9C), UINT8_C(0xEA), UINT8_C(0xB8), UINT8_C(0x80), UINT8_C(0xCE), UINT8_C(0xBB) },
            { 0, 0, 0, 0, 0, 0, 0, 0 }
        };
        static const size_t expected_length[4] = { (size_t)5, (size_t)8, (size_t)8, (size_t)0 };
        size_t index;
        for (index = 1; index <= 4; index++) {
            result = fy_std_process_arg_v0(index, round_buffer, sizeof(round_buffer));
            if (result.status != FY_STD_PROCESS_ARGS_OK_V0 ||
                result.needed != expected_length[index - 1]) {
                return 13;
            }
            if (result.needed != (size_t)0 &&
                memcmp(round_buffer, expected[index - 1], result.needed) != 0) {
                return 14;
            }
        }
    }
    return 0;
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

const lexicalDeferSource = `module lexical_defer;

fn void record(i32 value) {
    assert(value == 42);
}

fn i32 main() {
    var i32 value = 40;
    defer record(value + 2);
    value = 0;
    return 42;
}
`

const checkedInSuccessPath = "examples/windows_smoke/main.fy"
const checkedInModuleMainPath = "examples/run/modules/main.fy"
const checkedInModuleSupportPath = "examples/run/modules/math.fy"
const checkedInWindowsTargetPath = "examples/windows_target/main.fy"
const checkedInWindowsQuickstartPath = "examples/windows_quickstart/main.fy"
const checkedInWindowsOutputPath = "examples/windows_output/main.fy"
const checkedInWindowsOutputSupportPath = "examples/windows_output/windows_console.fy"
const checkedInStdIOManifestPath = "examples/std_io_quickstart/fairy.toml"
const checkedInStdIOMainPath = "examples/std_io_quickstart/src/main.fy"
const checkedInStdIoScalarManifestPath = "examples/std_io_scalar_quickstart/fairy.toml"
const checkedInStdIoScalarMainPath = "examples/std_io_scalar_quickstart/src/main.fy"
const checkedInStdAllocManifestPath = "examples/std_alloc_quickstart/fairy.toml"
const checkedInStdAllocMainPath = "examples/std_alloc_quickstart/src/main.fy"
const checkedInStdBytesManifestPath = "examples/std_bytes_quickstart/fairy.toml"
const checkedInStdBytesMainPath = "examples/std_bytes_quickstart/src/main.fy"
const checkedInStdTextManifestPath = "examples/std_text_quickstart/fairy.toml"
const checkedInStdTextMainPath = "examples/std_text_quickstart/src/main.fy"
const checkedInStdTextFromBufferManifestPath = "examples/std_text_from_buffer_quickstart/fairy.toml"
const checkedInStdTextFromBufferMainPath = "examples/std_text_from_buffer_quickstart/src/main.fy"
const checkedInStdFSManifestPath = "examples/std_fs_quickstart/fairy.toml"
const checkedInStdFSMainPath = "examples/std_fs_quickstart/src/main.fy"
const checkedInStdFSPathOpsPath = "examples/std_fs_quickstart/src/path_ops.fy"
const checkedInStdFSPositioningPath = "examples/std_fs_quickstart/src/positioning.fy"
const checkedInStdFSMetadataPath = "examples/std_fs_quickstart/src/metadata.fy"
const checkedInStdFSLinkOpsPath = "examples/std_fs_quickstart/src/link_ops.fy"
const checkedInRegionsManifestPath = "examples/regions/fairy.toml"
const checkedInRegionsMainPath = "examples/regions/src/main.fy"
const checkedInProcessArgsManifestPath = "examples/process_args_quickstart/fairy.toml"
const checkedInProcessArgsMainPath = "examples/process_args_quickstart/src/main.fy"
const checkedInWindowsFilePath = "examples/windows_file/main.fy"
const checkedInWindowsFileSupportPath = "examples/windows_file/windows_file.fy"
const checkedInOwnedTextPath = "examples/owned_text/main.fy"
const checkedInOwnedTextSupportPath = "examples/owned_text/owned_text.fy"
const checkedInWorkspaceManifestPath = "examples/workspace_quickstart/fairy.toml"
const checkedInWorkspaceMainPath = "examples/workspace_quickstart/src/main.fy"
const checkedInWorkspaceMemberManifestPath = "examples/workspace_quickstart/libs/answer/fairy.toml"
const checkedInWorkspaceMemberSourcePath = "examples/workspace_quickstart/libs/answer/src/answer.fy"
const checkedInWorkspaceMemberTestPath = "examples/workspace_quickstart/libs/answer/tests/answer.fy"
const checkedInWorkspaceTestPath = "examples/workspace_quickstart/tests/answer.fy"
const workspaceManifestPath = "fairy.toml"
const workspaceMainPath = "src/main.fy"
const workspaceMemberManifestPath = "libs/answer/fairy.toml"
const workspaceMemberSourcePath = "libs/answer/src/answer.fy"
const workspaceMemberTestPath = "libs/answer/tests/answer.fy"
const workspaceTestPath = "tests/answer.fy"
const stableWorkspaceOutputPath = "build/workspace-quickstart.exe"
const windowsLayoutManifestPath = "spec/fixtures/layout/manifest-windows-msvc-v0.json"
const windowsABIManifestPath = "spec/fixtures/c-abi/manifest-windows-msvc-v0.json"
const subprocessOutputLimit = 64 * 1024

// generatedFairyProbeTrapExitBase matches the Windows product trap base in internal/cli/build.go
// so generated-Fairy probe closures keep the same bounded process-status meaning as product runs.
const generatedFairyProbeTrapExitBase = 128

const (
	processCacheMaxEntries       = 8
	processCacheMaxBytes         = 32 << 20
	processCacheMaxArtifactBytes = 8 << 20
)

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
	stdIOManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdIOManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::io workspace manifest: %w", err)
	}
	stdIOMain, err := os.ReadFile(filepath.FromSlash(checkedInStdIOMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::io workspace entry: %w", err)
	}
	stdIoScalarManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdIoScalarManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::io scalar workspace manifest: %w", err)
	}
	stdIoScalarMain, err := os.ReadFile(filepath.FromSlash(checkedInStdIoScalarMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::io scalar workspace entry: %w", err)
	}
	stdAllocManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdAllocManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::alloc workspace manifest: %w", err)
	}
	stdAllocMain, err := os.ReadFile(filepath.FromSlash(checkedInStdAllocMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::alloc workspace entry: %w", err)
	}
	stdBytesManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdBytesManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::bytes workspace manifest: %w", err)
	}
	stdBytesMain, err := os.ReadFile(filepath.FromSlash(checkedInStdBytesMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::bytes workspace entry: %w", err)
	}
	stdTextManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdTextManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::text workspace manifest: %w", err)
	}
	stdTextMain, err := os.ReadFile(filepath.FromSlash(checkedInStdTextMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::text workspace entry: %w", err)
	}
	stdTextFromBufferManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdTextFromBufferManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::text from-buffer workspace manifest: %w", err)
	}
	stdTextFromBufferMain, err := os.ReadFile(filepath.FromSlash(checkedInStdTextFromBufferMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::text from-buffer workspace entry: %w", err)
	}
	stdFSManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdFSManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::fs workspace manifest: %w", err)
	}
	stdFSMain, err := os.ReadFile(filepath.FromSlash(checkedInStdFSMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::fs workspace entry: %w", err)
	}
	stdFSPathOps, err := os.ReadFile(filepath.FromSlash(checkedInStdFSPathOpsPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::fs path operations: %w", err)
	}
	stdFSPositioning, err := os.ReadFile(filepath.FromSlash(checkedInStdFSPositioningPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::fs positioning operations: %w", err)
	}
	stdFSMetadata, err := os.ReadFile(filepath.FromSlash(checkedInStdFSMetadataPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::fs metadata operations: %w", err)
	}
	stdFSLinkOps, err := os.ReadFile(filepath.FromSlash(checkedInStdFSLinkOpsPath))
	if err != nil {
		return fmt.Errorf("read checked-in std::fs hard-link operations: %w", err)
	}
	regionsManifest, err := os.ReadFile(filepath.FromSlash(checkedInRegionsManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in Region workspace manifest: %w", err)
	}
	regionsMain, err := os.ReadFile(filepath.FromSlash(checkedInRegionsMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in Region workspace entry: %w", err)
	}
	processArgsManifest, err := os.ReadFile(filepath.FromSlash(checkedInProcessArgsManifestPath))
	if err != nil {
		return fmt.Errorf("read checked-in process args workspace manifest: %w", err)
	}
	processArgsMain, err := os.ReadFile(filepath.FromSlash(checkedInProcessArgsMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in process args workspace entry: %w", err)
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
	workspaceMemberTest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMemberTestPath))
	if err != nil {
		return fmt.Errorf("read checked-in workspace member test: %w", err)
	}
	workspaceTest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceTestPath))
	if err != nil {
		return fmt.Errorf("read checked-in workspace test: %w", err)
	}
	root, err := os.MkdirTemp("", "fairyc-windows-host-smoke-")
	if err != nil {
		return fmt.Errorf("create temporary source root: %w", err)
	}
	defer os.RemoveAll(root)
	for name, source := range map[string]string{
		workspaceManifestPath:                  string(workspaceManifest),
		workspaceMainPath:                      string(workspaceMain),
		workspaceMemberManifestPath:            string(workspaceMemberManifest),
		workspaceMemberSourcePath:              string(workspaceMemberSource),
		workspaceMemberTestPath:                string(workspaceMemberTest),
		workspaceTestPath:                      string(workspaceTest),
		checkedInSuccessPath:                   string(successSource),
		checkedInModuleMainPath:                string(moduleMainSource),
		checkedInModuleSupportPath:             string(moduleSupportSource),
		checkedInWindowsTargetPath:             string(windowsTargetSource),
		checkedInWindowsQuickstartPath:         string(windowsQuickstartSource),
		checkedInWindowsOutputPath:             string(windowsOutputSource),
		checkedInWindowsOutputSupportPath:      string(windowsOutputSupportSource),
		checkedInStdIOManifestPath:             string(stdIOManifest),
		checkedInStdIOMainPath:                 string(stdIOMain),
		checkedInStdIoScalarManifestPath:       string(stdIoScalarManifest),
		checkedInStdIoScalarMainPath:           string(stdIoScalarMain),
		checkedInStdAllocManifestPath:          string(stdAllocManifest),
		checkedInStdAllocMainPath:              string(stdAllocMain),
		checkedInStdBytesManifestPath:          string(stdBytesManifest),
		checkedInStdBytesMainPath:              string(stdBytesMain),
		checkedInStdTextManifestPath:           string(stdTextManifest),
		checkedInStdTextMainPath:               string(stdTextMain),
		checkedInStdTextFromBufferManifestPath: string(stdTextFromBufferManifest),
		checkedInStdTextFromBufferMainPath:     string(stdTextFromBufferMain),
		checkedInStdFSManifestPath:             string(stdFSManifest),
		checkedInStdFSMainPath:                 string(stdFSMain),
		checkedInStdFSPathOpsPath:              string(stdFSPathOps),
		checkedInStdFSPositioningPath:          string(stdFSPositioning),
		checkedInStdFSMetadataPath:             string(stdFSMetadata),
		checkedInStdFSLinkOpsPath:              string(stdFSLinkOps),
		checkedInRegionsManifestPath:           string(regionsManifest),
		checkedInRegionsMainPath:               string(regionsMain),
		checkedInProcessArgsManifestPath:       string(processArgsManifest),
		checkedInProcessArgsMainPath:           string(processArgsMain),
		checkedInWindowsFilePath:               string(windowsFileSource),
		checkedInWindowsFileSupportPath:        string(windowsFileSupportSource),
		checkedInOwnedTextPath:                 string(ownedTextSource),
		checkedInOwnedTextSupportPath:          string(ownedTextSupportSource),
		"negative.fy":                          negativeSource,
		"division_by_zero.fy":                  divisionByZeroSource,
		"remainder_by_zero.fy":                 remainderByZeroSource,
		"signed_division_overflow.fy":          signedDivisionOverflowSource,
		"integer_overflow.fy":                  integerOverflowSource,
		"usize_overflow.fy":                    usizeOverflowSource,
		"unsigned_division.fy":                 unsignedDivisionSource,
		"unsigned_division_zero.fy":            unsignedDivisionZeroSource,
		"unsigned_bitwise_shift.fy":            unsignedBitwiseShiftSource,
		"unsigned_invalid_shift.fy":            unsignedInvalidShiftSource,
		"left_shift.fy":                        leftShiftSource,
		"left_shift_overflow.fy":               leftShiftOverflowSource,
		"invalid_shift.fy":                     invalidShiftSource,
		"bounds.fy":                            boundsSource,
		"array_bounds.fy":                      arrayBoundsSource,
		"array_store_bounds.fy":                arrayStoreBoundsSource,
		"slice_bounds.fy":                      sliceBoundsSource,
		"assertion_failure.fy":                 assertionFailureSource,
		"lexical_defer.fy":                     lexicalDeferSource,
		"enum_result.fy":                       enumResultSource,
		"enum_result_propagation.fy":           enumResultPropagationSource,
		"enum_option.fy":                       enumOptionSource,
		"tagged_array.fy":                      taggedArraySource,
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
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return fmt.Errorf("resolve Windows MSVC probe toolchain: %w", err)
	}
	clockResolution, err := observeMonotonicClockResolution()
	if err != nil {
		return fmt.Errorf("observe process-cache measurement clock: %w", err)
	}
	cacheMeasurement, err := measureWorkspaceProcessCache(root, clockResolution)
	if err != nil {
		return fmt.Errorf("measure process-local workspace cache: %w", err)
	}
	cacheMeasurementJSON, err := buildmeasure.MarshalCache(cacheMeasurement)
	if err != nil {
		return fmt.Errorf("encode process-local workspace cache measurement: %w", err)
	}
	fmt.Printf("Fairy C process-cache measurement: %s\n", cacheMeasurementJSON)
	cacheEditMeasurement, err := measureWorkspaceProcessCacheEdits(root, clockResolution)
	if err != nil {
		return fmt.Errorf("measure process-local workspace cache edits: %w", err)
	}
	cacheEditMeasurementJSON, err := buildmeasure.MarshalCacheEdits(cacheEditMeasurement)
	if err != nil {
		return fmt.Errorf("encode process-local workspace cache edit measurement: %w", err)
	}
	fmt.Printf("Fairy C process-cache edit measurement: %s\n", cacheEditMeasurementJSON)
	atlasRewriteMeasurement, err := measureAtlasRewrite(root, clockResolution)
	if err != nil {
		return fmt.Errorf("measure Atlas whole-package rewrite: %w", err)
	}
	atlasRewriteMeasurementJSON, err := buildmeasure.MarshalAtlasRewrite(atlasRewriteMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas whole-package rewrite measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas rewrite measurement: %s\n", atlasRewriteMeasurementJSON)
	atlasCallScaleMeasurement, err := measureAtlasCallScale(clockResolution)
	if err != nil {
		return fmt.Errorf("measure Atlas workspace-call scaling: %w", err)
	}
	atlasCallScaleMeasurementJSON, err := buildmeasure.MarshalAtlasCallScale(atlasCallScaleMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas workspace-call scaling measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas call scale measurement: %s\n", atlasCallScaleMeasurementJSON)
	measurement := buildmeasure.Report{
		Schema:              buildmeasure.Schema,
		Workload:            "workspace_quickstart",
		Clock:               "process-monotonic-wall-nanoseconds",
		CacheImplementation: "none",
		Decision:            "baseline-only",
		Environment: buildmeasure.Environment{
			OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc", Toolchain: "msvc " + toolchain.Version,
		},
	}
	workspaceBuildExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		for _, condition := range []string{buildmeasure.ConditionOrdinary, buildmeasure.ConditionLocked} {
			label := "ordinary"
			arguments := []string{
				"experimental", "build-workspace",
				"--target", "x86_64-windows-msvc", "--profile", profile,
			}
			if condition == buildmeasure.ConditionLocked {
				label = "locked"
				arguments = append(arguments, "--locked")
			}
			output := fmt.Sprintf("build/workspace-%s-%s.exe", profile, label)
			arguments = append(arguments, "--output", output)
			var workspaceBuildStdout bytes.Buffer
			var workspaceBuildStderr bytes.Buffer
			started := time.Now()
			workspaceBuildCode := cli.Run(arguments, root, &workspaceBuildStdout, &workspaceBuildStderr)
			duration := time.Since(started)
			lockState := "verified"
			if condition == buildmeasure.ConditionOrdinary {
				lockState = "unchanged"
				if profile == "low" {
					lockState = "updated"
				}
			}
			if workspaceBuildCode != cli.ExitSuccess || workspaceBuildStderr.Len() != 0 ||
				!strings.HasPrefix(workspaceBuildStdout.String(), "built workspace: "+output+" (x86_64-windows-msvc, msvc ") ||
				!strings.Contains(workspaceBuildStdout.String(), ", lock "+lockState+")") {
				return fmt.Errorf("build-workspace %s %s returned code %d, stdout %q, stderr %q", profile, label, workspaceBuildCode, workspaceBuildStdout.String(), workspaceBuildStderr.String())
			}
			workspaceArtifactPath := filepath.Join(root, filepath.FromSlash(output))
			workspaceArtifact, err := os.ReadFile(workspaceArtifactPath)
			if err != nil {
				return fmt.Errorf("read build-workspace %s %s artifact: %w", profile, label, err)
			}
			if _, err := pecoff.InspectExecutable(workspaceArtifact); err != nil {
				return fmt.Errorf("inspect build-workspace %s %s PE executable image: %w", profile, label, err)
			}
			if _, err := pecoff.InspectAMD64Unwind(workspaceArtifact); err != nil {
				return fmt.Errorf("inspect build-workspace %s %s Microsoft x64 unwind data: %w", profile, label, err)
			}
			if err := executeExpectResult(workspaceArtifactPath, 42); err != nil {
				return fmt.Errorf("execute build-workspace %s %s artifact: %w", profile, label, err)
			}
			measurement.Samples = append(measurement.Samples, buildmeasure.Sample{
				Sequence: len(measurement.Samples) + 1, Profile: profile, Condition: condition,
				LockState: lockState, DurationNS: duration.Nanoseconds(), ArtifactBytes: int64(len(workspaceArtifact)), NativeResult: 42,
			})
			workspaceBuildExecutions++
		}
	}
	stableBuildExecutions, err := runStableWorkspaceBuildProof(root)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stableBuildExecutions
	measurementJSON, err := buildmeasure.Marshal(measurement)
	if err != nil {
		return fmt.Errorf("encode Windows workspace build measurement: %w", err)
	}
	fmt.Printf("Fairy C build measurement: %s\n", measurementJSON)
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
	stableRunExecutions, err := runStableWorkspaceRunProof(root)
	if err != nil {
		return err
	}
	workspaceRunExecutions += stableRunExecutions
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
	stdIORoot := filepath.Join(root, "examples", "std_io_quickstart")
	var stdIOBuildStdout bytes.Buffer
	var stdIOBuildStderr bytes.Buffer
	stdIOBuildCode := cli.Run([]string{"build", "--release"}, stdIORoot, &stdIOBuildStdout, &stdIOBuildStderr)
	if stdIOBuildCode != cli.ExitSuccess || stdIOBuildStderr.Len() != 0 ||
		stdIOBuildStdout.String() != "built: build/std-io-quickstart.exe (x86_64-windows-msvc, release, lock updated)\n" {
		return fmt.Errorf("std::io stable build returned code %d, stdout %q, stderr %q", stdIOBuildCode, stdIOBuildStdout.String(), stdIOBuildStderr.String())
	}
	stdIOArtifactPath := filepath.Join(stdIORoot, "build", "std-io-quickstart.exe")
	stdIOArtifact, err := os.ReadFile(stdIOArtifactPath)
	if err != nil {
		return fmt.Errorf("read std::io stable build artifact: %w", err)
	}
	if _, err := pecoff.InspectExecutable(stdIOArtifact); err != nil {
		return fmt.Errorf("inspect std::io stable build PE image: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(stdIOArtifact); err != nil {
		return fmt.Errorf("inspect std::io stable build unwind data: %w", err)
	}
	if err := executeExpectOutput(
		stdIOArtifactPath,
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\n",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return fmt.Errorf("execute std::io stable build artifact: %w", err)
	}
	if err := executeExpectOutputWithInput(
		stdIOArtifactPath,
		"stdin!\n",
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\nstdin!\n",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return fmt.Errorf("execute std::io stable build artifact with short stdin: %w", err)
	}
	if err := executeExpectOutputWithInput(
		stdIOArtifactPath,
		"123456789",
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\n12345678",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return fmt.Errorf("execute std::io stable build artifact with bounded stdin: %w", err)
	}
	var stdIORunStdout bytes.Buffer
	var stdIORunStderr bytes.Buffer
	stdIORunCode := cli.RunWithInput(
		[]string{"run", "--release", "--locked"},
		stdIORoot,
		strings.NewReader("stdin!\n"),
		&stdIORunStdout,
		&stdIORunStderr,
	)
	if stdIORunCode != cli.ExitSuccess || stdIORunStdout.String() != "Fairy C says hi\nbytes on stdout\n-2147483648\nstdin!\n" ||
		stdIORunStderr.String() != "Fairy C warns too\nbytes on stderr\n" {
		return fmt.Errorf("std::io stable run returned code %d, stdout %q, stderr %q", stdIORunCode, stdIORunStdout.String(), stdIORunStderr.String())
	}
	workspaceRunExecutions++
	workspaceBuildExecutions++
	stdAllocBuildExecutions, err := runStdlibQuickstartProof(root, "std_alloc_quickstart", "std-alloc-quickstart", "std::alloc", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdAllocBuildExecutions
	stdBytesBuildExecutions, err := runStdlibQuickstartProof(root, "std_bytes_quickstart", "std-bytes-quickstart", "std::bytes", 0)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdBytesBuildExecutions
	stdTextBuildExecutions, err := runStdlibQuickstartProof(root, "std_text_quickstart", "std-text-quickstart", "std::text", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdTextBuildExecutions
	stdTextFromBufferBuildExecutions, err := runStdlibQuickstartProof(root, "std_text_from_buffer_quickstart", "std-text-from-buffer-quickstart", "std::bytes", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdTextFromBufferBuildExecutions
	stdFSBuildExecutions, err := runStdlibQuickstartProofWithFixtures(
		root,
		"std_fs_quickstart",
		"std-fs-quickstart",
		"std::fs",
		42,
		[]string{"fairyc-std-fs-read-only-metadata.tmp"},
		[]hardLinkFixture{{
			path:  "fairyc-std-fs-hard-link-metadata.tmp",
			alias: "fairyc-std-fs-hard-link-metadata-alias.tmp",
		}},
		"fairyc-std-fs-data.tmp",
		"fairyc-std-fs-drop.tmp",
		"fairyc-std-fs-open-or-create.tmp",
		"fairyc-std-fs-create-new.tmp",
		"fairyc-std-fs-append.tmp",
		"fairyc-std-fs-read-append-error.tmp",
		"fairyc-std-fs-copy-source.tmp",
		"fairyc-std-fs-copy-destination.tmp",
		"fairyc-std-fs-copy-missing-destination.tmp",
		"fairyc-std-fs-hard-link-source.tmp",
		"fairyc-std-fs-hard-link-destination.tmp",
		"fairyc-std-fs-hard-link-missing.tmp",
		"fairyc-std-fs-hard-link-missing-destination.tmp",
		"fairyc-std-fs-read-only-mutation.tmp",
		"fairyc-std-fs-remove-all\\nested\\b.tmp",
		"fairyc-std-fs-remove-all\\a.tmp",
		"fairyc-std-fs-remove-all\\nested",
		"fairyc-std-fs-remove-all",
		"fairyc-std-fs-remove-all-blocked\\child.tmp",
		"fairyc-std-fs-remove-all-blocked",
		"fairyc-std-fs-remove-all-file.tmp",
		"fairyc-std-fs-directory-all\\collision.tmp",
		"fairyc-std-fs-directory-all\\nested\\leaf",
		"fairyc-std-fs-directory-all\\nested",
		"fairyc-std-fs-directory-all",
		"fairyc-std-fs-directory-all-partial\\created",
		"fairyc-std-fs-directory-all-partial",
	)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdFSBuildExecutions
	stdIoScalarBuildExecutions, err := runStdIoScalarQuickstartProof(root)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdIoScalarBuildExecutions
	regionBuildExecutions, err := runStdlibQuickstartProof(root, "regions", "regions", "Region", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += regionBuildExecutions
	var atlasCallProcessMeasurement buildmeasure.AtlasCallProcessReport
	standaloneCompilerBytes, err := runStandaloneCompiler(repositoryRoot, root, clockResolution, &atlasCallProcessMeasurement)
	if err != nil {
		return err
	}
	atlasCallProcessMeasurementJSON, err := buildmeasure.MarshalAtlasCallProcess(atlasCallProcessMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas call process measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas call process measurement: %s\n", atlasCallProcessMeasurementJSON)
	atlasCallRetainedMeasurement, err := measureAtlasCallRetainedHeap()
	if err != nil {
		return fmt.Errorf("measure Atlas workspace-call retained heap: %w", err)
	}
	atlasCallRetainedMeasurementJSON, err := buildmeasure.MarshalAtlasCallRetained(atlasCallRetainedMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas workspace-call retained-heap measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas call retained-heap measurement: %s\n", atlasCallRetainedMeasurementJSON)
	targetRunExecutions++
	targetExecExecutions++
	workspaceBuildExecutions += 2
	workspaceRunExecutions += 3

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
		{name: "lexical defer low", source: "lexical_defer.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "lexical defer optimized", source: "lexical_defer.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
	} {
		if err := runHostCase(root, testCase.name, testCase.source, testCase.support, testCase.profile, testCase.code, testCase.stdoutPrefix, testCase.stderr); err != nil {
			return err
		}
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(root, "user-enum Result "+profile, "enum_result.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
		if err := runHostCase(root, "user-enum Result propagation "+profile, "enum_result_propagation.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
		if err := runHostCase(root, "user-enum Option "+profile, "enum_option.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
		if err := runHostCase(root, "copy fixed-array Option/Result "+profile, "tagged_array.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
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
	regionFailureProbeImages, regionFailureCases, err := runGeneratedFairyRegionFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	bufferFailureProbeImages, bufferFailureCases, err := runGeneratedFairyBufferFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	readAppendFailureProbeImages, readAppendFailureCases, err := runGeneratedFairyReadAppendFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	readToEndFailureProbeImages, readToEndFailureCases, err := runGeneratedFairyReadToEndFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	stringAppendFailureProbeImages, stringAppendFailureCases, err := runGeneratedFairyStringAppendFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	cstringFailureProbeImages, cstringFailureCases, err := runGeneratedFairyCStringFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	stringCopyFailureProbeImages, stringCopyFailureCases, err := runGeneratedFairyStringCopyFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	stringFromBufferProbeImages, stringFromBufferProbeCases, err := runGeneratedFairyStringFromBufferProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	processArgsBuildImages, processArgsExecutions, err := runProcessArgsQuickstartProof(root)
	if err != nil {
		return err
	}
	processArgsDecodeImages, processArgsDecodeExecutions, err := runWindowsProcessArgsDecodeProbe(toolchain)
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
	relocationExecutions, err := runWindowsRelocationProbe(toolchain)
	if err != nil {
		return err
	}
	weakSearchExecutions, err := runWindowsWeakSearchProbe(toolchain)
	if err != nil {
		return err
	}
	archiveExecutions, err := runWindowsArchiveProbe(toolchain)
	if err != nil {
		return err
	}
	structResultExecutions, err := runFairyStructResultProbe(toolchain)
	if err != nil {
		return err
	}
	cIntCastExecutions, err := runCIntCastProbe(toolchain)
	if err != nil {
		return err
	}
	unwindExecutions, unwindCases, err := runWindowsUnwindProbe(toolchain)
	if err != nil {
		return err
	}
	inspectedPEImages := 1 + targetBuildExecutions + workspaceBuildExecutions + windowsFileSourceExecutions + ownedTextSourceExecutions + targetRunExecutions + workspaceRunExecutions + targetExecExecutions + windowsFileExecutions + bufferProbeExecutions + ownedTextProbeExecutions + arenaProbeExecutions + regionFailureProbeImages + bufferFailureProbeImages + readAppendFailureProbeImages + readToEndFailureProbeImages + stringAppendFailureProbeImages + cstringFailureProbeImages + stringCopyFailureProbeImages + stringFromBufferProbeImages + processArgsBuildImages + processArgsDecodeImages + layoutExecutions + abiExecutions + relocationExecutions + weakSearchExecutions + archiveExecutions + structResultExecutions + cIntCastExecutions + unwindExecutions
	fmt.Printf("Generated-Fairy Region failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering payload-allocation failure, node-allocation rollback, exact status 1, lexical reset, automatic Arena drop, and zero retained probe allocations\n", regionFailureCases, regionFailureProbeImages)
	fmt.Printf("Generated-Fairy Buffer failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering with_capacity initial-allocation failure plus resize_zeroed and append_byte growth failures with preserved len/capacity, exact status 1, and zero retained probe allocations\n", bufferFailureCases, bufferFailureProbeImages)
	fmt.Printf("Generated-Fairy read_append failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering growth-allocation failure before I/O, exact file/allocation statuses 0/1, preserved Buffer len/capacity, consuming File close, automatic Buffer drop, and zero retained probe allocations\n", readAppendFailureCases, readAppendFailureProbeImages)
	fmt.Printf("Generated-Fairy read_to_end failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering one committed 65536-byte prefix, second-window allocation failure before I/O, exact file/allocation statuses 0/1, preserved cursor and Buffer bytes, consuming File close, automatic Buffer drop, and zero retained probe allocations\n", readToEndFailureCases, readToEndFailureProbeImages)
	fmt.Printf("Generated-Fairy String append failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering successful copy allocation, append growth-allocation failure with preserved owner and len/is_empty/view, exact statuses 1/1, automatic String drop, and zero retained probe allocations\n", stringAppendFailureCases, stringAppendFailureProbeImages)
	fmt.Printf("Generated-Fairy CString failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering cstring_from_text initial-allocation failure, exact statuses 1/1, automatic CString drop, and zero retained probe allocations\n", cstringFailureCases, cstringFailureProbeImages)
	fmt.Printf("Generated-Fairy String copy failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering text::copy initial-allocation failure, exact statuses 1/1, automatic String drop, and zero retained probe allocations\n", stringCopyFailureCases, stringCopyFailureProbeImages)
	fmt.Printf("Generated-Fairy Buffer-to-String conversion proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering zero-copy valid UTF-8 transfer under a fail-after-two schedule, invalid UTF-8 ownership return with preserved len/capacity/bytes, empty String success, automatic String and Buffer drop, and zero retained probe allocations\n", stringFromBufferProbeCases, stringFromBufferProbeImages)
	fmt.Printf("Generated-Fairy std::fs one-call write, owned-Buffer read append/read-to-end, file and directory creation/removal/no-replace rename, recursive parent creation, no-replace byte copy and hard-link creation, path-kind snapshot, absolute/current/end seek, same-handle read-only observation/mutation, hard-link, and allocated-size metadata, append, file-kind, exact-read, and file-identity proof passed: %d native executions across %d inspected low/optimized MSVC images, covering one-call prefix success, read-only write failure with zero progress, write-all completion, initialized-prefix Buffer append, current-position read-to-end, short read, EOF, write-only read failure with preserved bytes, open-or-create preservation, exclusive create collision, exact file remove success plus missing-path and open-sharing failures, no-replace file rename collision preservation plus open-sharing failure and exact final bytes, exact byte-copy completion plus collision preservation and missing-source no-create behavior, no-replace hard-link collision plus missing-source no-create behavior, shared identity, link-count and bytes, source-name removal, and surviving destination alias, path-level ordinary file and directory classification plus exact missing failure, one-level and nested directory creation, existing-directory success, missing-parent and file/directory collision failures, partial-prefix preservation after a later invalid-name failure, non-empty removal failure, child cleanup, exact empty-directory removal, no-replace directory rename collision, moved-child preservation, old-path disappearance, bounded negative seek, start-preserving failure, beyond-EOF cursor movement with unchanged size, write-capable read-only true/false mutation, exact read-only-handle access denial with unchanged metadata, ordinary and host-created read-only metadata plus both names of a host-created two-link file with filesystem-reported allocation at least logical size and unchanged cursor/size, write-only append creation, seek-resistant EOF writes, exact retained bytes, disk files, the NUL character device, complete reads, unexpected-EOF prefix preservation, same-file handles, and distinct live files\n", stdFSBuildExecutions, stdFSBuildExecutions)
	fmt.Printf("std::fs recursive removal proof covered generated-Fairy exact four-object success, missing-path storage, open-child sharing failure with zero progress, read-only preservation, raw actual junction-leaf target survival, and retry cleanup across the same %d native executions\n", stdFSBuildExecutions)
	fmt.Printf("Stable run program-argument forwarding proof passed: %d native executions across %d inspected low/optimized MSVC images, covering argv[0] inclusion and ordering, empty, flag-looking, space/quote/trailing-backslash, and non-ASCII UTF-8 argument round-trips, out-of-range status 1, too-small status 3 with exact needed and an unchanged caller buffer, CLI embedded-NUL and delimiter rejection, and low-level invalid-surrogate status 2 with an unchanged buffer\n", processArgsExecutions+processArgsDecodeExecutions, processArgsBuildImages+processArgsDecodeImages)
	fmt.Printf("std::io scalar output proof passed: %d native executions across %d inspected low/optimized MSVC images, covering stdout u32 0/1/max, usize 0/1/max, bool true/false, plus stderr i32 0/negative/min/max, u32 0/max, usize 0/max, and bool true/false exact ASCII bytes with exact written counts\n", stdIoScalarBuildExecutions, stdIoScalarBuildExecutions)
	fmt.Println("Generated-Fairy user-enum Result proof passed: 4 native executions across low/optimized MSVC, covering .ok enum construction, enum payload extraction, exhaustive enum switching, .err inert-struct extraction, and ? propagation of enum errors")
	fmt.Println("Generated-Fairy user-enum Option proof passed: 2 native executions across low/optimized MSVC, covering Some enum construction, payload extraction, exhaustive enum switching, and None")
	fmt.Println("Generated-Fairy copy fixed-array Option/Result proof passed: 2 native executions across low/optimized MSVC, covering Some and None, Ok and Err, indexed payload reads, and success/error propagation")
	fmt.Printf("Generated-Fairy c::int cast proof passed: %d native executions across low/optimized MSVC, covering imported negative-capable C int identity, explicit c::int-to-i32 conversion, exact value round-trip, explicit i32-to-c::int conversion, and a separate C consumer\n", cIntCastExecutions)
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes with %d validated sections, %d runtime-function records, and %d version-2 unwind records, %d source-level Windows target builds, %d package-workspace builds, %d package-workspace runs, %d source-level safe File lifecycle builds, %d source-level safe owned-text lifecycle builds, %d direct run-target executions, and %d exec-target executions checked LLP64 c.long emission, move-only pointer-width resource transport, closed SystemHeap, public safe std::bytes Buffer ownership and views, Arena/Region, String/CString/CStr, allocation-free owner-bound text-to-byte views, UTF-8 crossings, safe borrowed-view internal calls, byte lengths, typed scalar comparisons, scalar fixed-array calls, checked reads and mutable indexed stores, checked byte indexing and explicit or omitted-bound slicing, always-checked assertions, bounded lexical defer, value-preserving integer casts, checked unsigned arithmetic, division, remainder, bitwise operations, and left/right shifts, exact stdout bytes, stable-run stdin forwarding, bounded allocation-free stdin byte reads with EOF and partial-buffer evidence, and native result 42, including one %d-byte standalone Go-built fairyc.exe quickstart, output, locked workspace execution, semantic text/JSON tests, ten isolated native-test commands covering fourteen executions across low/optimized MSVC including exact portable test-file and recursive test-directory selection, exact-current source anchors, and optimized JSON trap continuation, selected workspace-member tests, and package-qualified workspace aggregation, two target-explicit C emissions, nine paired cold/warm persistent-cache process measurements, one nine-process bounded persistent-cache growth staircase, an eight-request editor-shaped workspace session with Windows process-memory peaks, %d fresh locked Atlas workspace-call catalog processes with per-process memory peaks, %d generated Atlas workspace-call scaling shapes, and %d post-GC retained-index batches, 42 run-host outcomes covered fixed arrays, byte-view iteration and slicing, checked indexed stores, assertions and lexical defer in low and optimized profiles, usize arithmetic and overflow, unsigned division and zero traps, typed bitwise operations, checked shift-width and left-shift overflow traps, signed results, and six trap categories, Windows File partial I/O and consuming-close passed %d separate-translation-unit MSVC executions, Buffer initialization, reserve, self-overlap append, and zeroed-resize allocation rollback passed %d separate-translation-unit MSVC executions, owned-text opaque-storage failure preservation, forced allocation rollback, and repeated drop passed %d separate-translation-unit MSVC executions, Arena zeroing, two-stage allocation rollback, stale Region rejection, and active-drop preservation passed %d separate-translation-unit MSVC executions, %d LLP64 layout cases passed %d MSVC probe executions, %d Microsoft x64 ABI cases passed %d separate-translation-unit MSVC probe executions, exact object addends and final-image REL32 through REL32_5, weak-alias fallback/strong override, ADDR64, and unwind ADDR32NB values plus CodeView SECTION/SECREL pairs, PE/PDB identity, and exact DbgHelp function/line symbolication passed %d separate-translation-unit MSVC executions, weak NOLIBRARY suppression and LIBRARY extraction passed %d separate-translation-unit MSVC executions, archive extraction, pick-any ownership, selected-child retention, and discarded-child elimination passed %d MSVC input-order executions, generated-Fairy small and large struct results passed %d separate-translation-unit MSVC executions across C11 and C17 consumer modes, %d Windows x64 unwind cases plus exact profile-specific handler-tail ADDR32NB binding passed %d actual exception-dispatch and virtual-unwind executions, %d PE32+ target or probe images passed structural and unwind inspection, and every MSVC compilation object passed COFF inspection across low/optimized with msvc %s\n", len(artifact), len(peReport.Sections), unwindReport.RuntimeFunctions, unwindReport.Version2Infos, targetBuildExecutions, workspaceBuildExecutions, workspaceRunExecutions, windowsFileSourceExecutions, ownedTextSourceExecutions, targetRunExecutions, targetExecExecutions, standaloneCompilerBytes, buildmeasure.AtlasCallProcessIterations, len(atlasCallScaleMeasurement.Shapes), len(atlasCallRetainedMeasurement.Shapes)*buildmeasure.AtlasCallRetainedIterationsPerShape, windowsFileExecutions, bufferProbeExecutions, ownedTextProbeExecutions, arenaProbeExecutions, layoutCases, layoutExecutions, abiCases, abiExecutions, relocationExecutions, weakSearchExecutions, archiveExecutions, structResultExecutions, unwindCases, unwindExecutions, inspectedPEImages, toolchain.Version)
	return nil
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

func executeExpectResultWithArgs(path string, expected int32, args ...string) error {
	contextValue, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, path)
	command.Args = append([]string{path}, args...)
	command.Dir = filepath.Dir(path)
	command.Stdin = nil
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("build-host artifact with arguments exceeded execution timeout")
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return fmt.Errorf("build-host artifact with arguments execution failed: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	actual := int32(uint32(exitCode))
	if actual != expected || output.Len() != 0 {
		exitCode := -1
		if command.ProcessState != nil {
			exitCode = command.ProcessState.ExitCode()
		}
		return fmt.Errorf("build-host artifact with arguments returned error %v, exit %d (i32 %d), output %q", err, exitCode, actual, output.String())
	}
	return nil
}

func executeExpectOutput(path string, expected int32, expectedStdout, expectedStderr string) error {
	return executeExpectOutputWithInput(path, "", expected, expectedStdout, expectedStderr)
}

func executeExpectOutputWithInput(path, input string, expected int32, expectedStdout, expectedStderr string) error {
	contextValue, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, path)
	command.Dir = filepath.Dir(path)
	command.Stdin = strings.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("artifact exceeded execution timeout")
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return fmt.Errorf("artifact execution failed: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	actual := int32(uint32(exitCode))
	if actual != expected || stdout.String() != expectedStdout || stderr.String() != expectedStderr {
		return fmt.Errorf("artifact returned error %v, exit %d (i32 %d), stdout %q, stderr %q", err, exitCode, actual, stdout.String(), stderr.String())
	}
	return nil
}

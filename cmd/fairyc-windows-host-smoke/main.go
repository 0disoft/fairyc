package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/foreignstructprobe"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/windowsabiprobe"
	"github.com/0disoft/fairyc/internal/windowsarchiveprobe"
	"github.com/0disoft/fairyc/internal/windowslayoutprobe"
	"github.com/0disoft/fairyc/internal/windowsrelocationprobe"
	"github.com/0disoft/fairyc/internal/windowsunwindprobe"
	"github.com/0disoft/fairyc/internal/windowsweaksearchprobe"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
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
	stdFSBuildExecutions, err := runStdlibQuickstartProof(
		root,
		"std_fs_quickstart",
		"std-fs-quickstart",
		"std::fs",
		42,
		"fairyc-std-fs-data.tmp",
		"fairyc-std-fs-drop.tmp",
		"fairyc-std-fs-open-or-create.tmp",
		"fairyc-std-fs-create-new.tmp",
		"fairyc-std-fs-append.tmp",
		"fairyc-std-fs-read-append-error.tmp",
		"fairyc-std-fs-copy-source.tmp",
		"fairyc-std-fs-copy-destination.tmp",
		"fairyc-std-fs-copy-missing-destination.tmp",
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
	fmt.Printf("Generated-Fairy Buffer failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering with_capacity initial-allocation failure and resize_zeroed growth failure with preserved len/capacity, exact status 1, and zero retained probe allocations\n", bufferFailureCases, bufferFailureProbeImages)
	fmt.Printf("Generated-Fairy read_append failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering growth-allocation failure before I/O, exact file/allocation statuses 0/1, preserved Buffer len/capacity, consuming File close, automatic Buffer drop, and zero retained probe allocations\n", readAppendFailureCases, readAppendFailureProbeImages)
	fmt.Printf("Generated-Fairy read_to_end failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering one committed 65536-byte prefix, second-window allocation failure before I/O, exact file/allocation statuses 0/1, preserved cursor and Buffer bytes, consuming File close, automatic Buffer drop, and zero retained probe allocations\n", readToEndFailureCases, readToEndFailureProbeImages)
	fmt.Printf("Generated-Fairy String append failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering successful copy allocation, append growth-allocation failure with preserved owner and len/is_empty/view, exact statuses 1/1, automatic String drop, and zero retained probe allocations\n", stringAppendFailureCases, stringAppendFailureProbeImages)
	fmt.Printf("Generated-Fairy CString failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering cstring_from_text initial-allocation failure, exact statuses 1/1, automatic CString drop, and zero retained probe allocations\n", cstringFailureCases, cstringFailureProbeImages)
	fmt.Printf("Generated-Fairy String copy failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering text::copy initial-allocation failure, exact statuses 1/1, automatic String drop, and zero retained probe allocations\n", stringCopyFailureCases, stringCopyFailureProbeImages)
	fmt.Printf("Generated-Fairy Buffer-to-String conversion proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering zero-copy valid UTF-8 transfer under a fail-after-two schedule, invalid UTF-8 ownership return with preserved len/capacity/bytes, empty String success, automatic String and Buffer drop, and zero retained probe allocations\n", stringFromBufferProbeCases, stringFromBufferProbeImages)
	fmt.Printf("Generated-Fairy std::fs one-call write, owned-Buffer read append/read-to-end, file and directory creation/removal/no-replace rename, recursive parent creation, no-replace byte copy, path-kind snapshot, append, file-kind, exact-read, and file-identity proof passed: %d native executions across %d inspected low/optimized MSVC images, covering one-call prefix success, read-only write failure with zero progress, write-all completion, initialized-prefix Buffer append, current-position read-to-end, short read, EOF, write-only read failure with preserved bytes, open-or-create preservation, exclusive create collision, exact file remove success plus missing-path and open-sharing failures, no-replace file rename collision preservation plus open-sharing failure and exact final bytes, exact byte-copy completion plus collision preservation and missing-source no-create behavior, path-level ordinary file and directory classification plus exact missing failure, one-level and nested directory creation, existing-directory success, missing-parent and file/directory collision failures, partial-prefix preservation after a later invalid-name failure, non-empty removal failure, child cleanup, exact empty-directory removal, no-replace directory rename collision, moved-child preservation, old-path disappearance, write-only append creation, seek-resistant EOF writes, exact retained bytes, disk files, the NUL character device, complete reads, unexpected-EOF prefix preservation, same-file handles, and distinct live files\n", stdFSBuildExecutions, stdFSBuildExecutions)
	fmt.Printf("Stable run program-argument forwarding proof passed: %d native executions across %d inspected low/optimized MSVC images, covering argv[0] inclusion and ordering, empty, flag-looking, space/quote/trailing-backslash, and non-ASCII UTF-8 argument round-trips, out-of-range status 1, too-small status 3 with exact needed and an unchanged caller buffer, CLI embedded-NUL and delimiter rejection, and low-level invalid-surrogate status 2 with an unchanged buffer\n", processArgsExecutions+processArgsDecodeExecutions, processArgsBuildImages+processArgsDecodeImages)
	fmt.Printf("std::io scalar output proof passed: %d native executions across %d inspected low/optimized MSVC images, covering u32 0/1/max, usize 0/1/max, and bool true/false exact ASCII bytes with exact written counts and empty stderr\n", stdIoScalarBuildExecutions, stdIoScalarBuildExecutions)
	fmt.Println("Generated-Fairy user-enum Result proof passed: 4 native executions across low/optimized MSVC, covering .ok enum construction, enum payload extraction, exhaustive enum switching, .err inert-struct extraction, and ? propagation of enum errors")
	fmt.Println("Generated-Fairy user-enum Option proof passed: 2 native executions across low/optimized MSVC, covering Some enum construction, payload extraction, exhaustive enum switching, and None")
	fmt.Println("Generated-Fairy copy fixed-array Option/Result proof passed: 2 native executions across low/optimized MSVC, covering Some and None, Ok and Err, indexed payload reads, and success/error propagation")
	fmt.Printf("Generated-Fairy c::int cast proof passed: %d native executions across low/optimized MSVC, covering imported negative-capable C int identity, explicit c::int-to-i32 conversion, exact value round-trip, explicit i32-to-c::int conversion, and a separate C consumer\n", cIntCastExecutions)
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes with %d validated sections, %d runtime-function records, and %d version-2 unwind records, %d source-level Windows target builds, %d package-workspace builds, %d package-workspace runs, %d source-level safe File lifecycle builds, %d source-level safe owned-text lifecycle builds, %d direct run-target executions, and %d exec-target executions checked LLP64 c.long emission, move-only pointer-width resource transport, closed SystemHeap, public safe std::bytes Buffer ownership and views, Arena/Region, String/CString/CStr, UTF-8 crossings, safe borrowed-view internal calls, byte lengths, typed scalar comparisons, scalar fixed-array calls, checked reads and mutable indexed stores, checked byte indexing and explicit or omitted-bound slicing, always-checked assertions, bounded lexical defer, value-preserving integer casts, checked unsigned arithmetic, division, remainder, bitwise operations, and left/right shifts, exact stdout bytes, stable-run stdin forwarding, bounded allocation-free stdin byte reads with EOF and partial-buffer evidence, and native result 42, including one %d-byte standalone Go-built fairyc.exe quickstart, output, locked workspace execution, semantic text/JSON tests, eight isolated native-test commands covering twelve executions across low/optimized MSVC including optimized JSON trap continuation, selected workspace-member tests, and package-qualified workspace aggregation, two target-explicit C emissions, nine paired cold/warm persistent-cache process measurements, one nine-process bounded persistent-cache growth staircase, an eight-request editor-shaped workspace session with Windows process-memory peaks, %d fresh locked Atlas workspace-call catalog processes with per-process memory peaks, %d generated Atlas workspace-call scaling shapes, and %d post-GC retained-index batches, 42 run-host outcomes covered fixed arrays, byte-view iteration and slicing, checked indexed stores, assertions and lexical defer in low and optimized profiles, usize arithmetic and overflow, unsigned division and zero traps, typed bitwise operations, checked shift-width and left-shift overflow traps, signed results, and six trap categories, Windows File partial I/O and consuming-close passed %d separate-translation-unit MSVC executions, Buffer initialization, reserve, self-overlap append, and zeroed-resize allocation rollback passed %d separate-translation-unit MSVC executions, owned-text opaque-storage failure preservation, forced allocation rollback, and repeated drop passed %d separate-translation-unit MSVC executions, Arena zeroing, two-stage allocation rollback, stale Region rejection, and active-drop preservation passed %d separate-translation-unit MSVC executions, %d LLP64 layout cases passed %d MSVC probe executions, %d Microsoft x64 ABI cases passed %d separate-translation-unit MSVC probe executions, exact object addends and final-image REL32 through REL32_5, weak-alias fallback/strong override, ADDR64, and unwind ADDR32NB values plus CodeView SECTION/SECREL pairs, PE/PDB identity, and exact DbgHelp function/line symbolication passed %d separate-translation-unit MSVC executions, weak NOLIBRARY suppression and LIBRARY extraction passed %d separate-translation-unit MSVC executions, archive extraction, pick-any ownership, selected-child retention, and discarded-child elimination passed %d MSVC input-order executions, generated-Fairy small and large struct results passed %d separate-translation-unit MSVC executions across C11 and C17 consumer modes, %d Windows x64 unwind cases plus exact profile-specific handler-tail ADDR32NB binding passed %d actual exception-dispatch and virtual-unwind executions, %d PE32+ target or probe images passed structural and unwind inspection, and every MSVC compilation object passed COFF inspection across low/optimized with msvc %s\n", len(artifact), len(peReport.Sections), unwindReport.RuntimeFunctions, unwindReport.Version2Infos, targetBuildExecutions, workspaceBuildExecutions, workspaceRunExecutions, windowsFileSourceExecutions, ownedTextSourceExecutions, targetRunExecutions, targetExecExecutions, standaloneCompilerBytes, buildmeasure.AtlasCallProcessIterations, len(atlasCallScaleMeasurement.Shapes), len(atlasCallRetainedMeasurement.Shapes)*buildmeasure.AtlasCallRetainedIterationsPerShape, windowsFileExecutions, bufferProbeExecutions, ownedTextProbeExecutions, arenaProbeExecutions, layoutCases, layoutExecutions, abiCases, abiExecutions, relocationExecutions, weakSearchExecutions, archiveExecutions, structResultExecutions, unwindCases, unwindExecutions, inspectedPEImages, toolchain.Version)
	return nil
}

func runStableWorkspaceBuildProof(root string) (int, error) {
	const output = stableWorkspaceOutputPath
	artifactPath := filepath.Join(root, filepath.FromSlash(output))

	var developmentStdout bytes.Buffer
	var developmentStderr bytes.Buffer
	developmentCode := cli.Run([]string{"build"}, root, &developmentStdout, &developmentStderr)
	if developmentCode != cli.ExitSuccess || developmentStderr.Len() != 0 ||
		developmentStdout.String() != "built: "+stableWorkspaceOutputPath+" (x86_64-windows-msvc, development, lock unchanged)\n" {
		return 0, fmt.Errorf("stable development build returned code %d, stdout %q, stderr %q", developmentCode, developmentStdout.String(), developmentStderr.String())
	}
	developmentArtifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable development build artifact: %w", err)
	}
	if _, err := pecoff.InspectExecutable(developmentArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable development PE executable image: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(developmentArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable development Microsoft x64 unwind data: %w", err)
	}
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return 0, fmt.Errorf("execute stable development build artifact: %w", err)
	}

	sourcePath := filepath.Join(root, filepath.FromSlash(workspaceMainPath))
	originalSource, err := os.ReadFile(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("read stable build source before failure proof: %w", err)
	}
	if err := os.WriteFile(sourcePath, []byte("module main;\nfn i32 broken( { return 0; }\n"), 0o600); err != nil {
		return 0, fmt.Errorf("write invalid stable build source: %w", err)
	}
	var invalidStdout bytes.Buffer
	var invalidStderr bytes.Buffer
	invalidCode := cli.Run([]string{"build", "--locked"}, root, &invalidStdout, &invalidStderr)
	restoreErr := os.WriteFile(sourcePath, originalSource, 0o600)
	if restoreErr != nil {
		return 0, fmt.Errorf("restore stable build source: %w", restoreErr)
	}
	if invalidCode != cli.ExitSourceDiagnostics || invalidStdout.Len() != 0 || !strings.Contains(invalidStderr.String(), "E0007") {
		return 0, fmt.Errorf("invalid stable build returned code %d, stdout %q, stderr %q", invalidCode, invalidStdout.String(), invalidStderr.String())
	}
	afterFailure, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable artifact after failed replacement: %w", err)
	}
	if !bytes.Equal(afterFailure, developmentArtifact) {
		return 0, fmt.Errorf("failed stable build modified the previous executable")
	}

	var releaseStdout bytes.Buffer
	var releaseStderr bytes.Buffer
	releaseCode := cli.Run([]string{"build", "--release", "--locked"}, root, &releaseStdout, &releaseStderr)
	if releaseCode != cli.ExitSuccess || releaseStderr.Len() != 0 ||
		releaseStdout.String() != "built: "+stableWorkspaceOutputPath+" (x86_64-windows-msvc, release, lock verified)\n" {
		return 0, fmt.Errorf("stable release build returned code %d, stdout %q, stderr %q", releaseCode, releaseStdout.String(), releaseStderr.String())
	}
	releaseArtifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable release build artifact: %w", err)
	}
	if _, err := pecoff.InspectExecutable(releaseArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable release PE executable image: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(releaseArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable release Microsoft x64 unwind data: %w", err)
	}
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return 0, fmt.Errorf("execute stable release build artifact: %w", err)
	}
	staging, err := filepath.Glob(filepath.Join(root, "build", ".fairyc-*.tmp"))
	if err != nil {
		return 0, fmt.Errorf("inspect stable build staging files: %w", err)
	}
	if len(staging) != 0 {
		return 0, fmt.Errorf("stable build left staging files: %v", staging)
	}
	return 2, nil
}

func runStableWorkspaceRunProof(root string) (int, error) {
	artifactPath := filepath.Join(root, filepath.FromSlash(stableWorkspaceOutputPath))
	retainedArtifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read retained stable build before run proof: %w", err)
	}
	lockPath := filepath.Join(root, "fairy.lock")
	originalLock, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, fmt.Errorf("read lock before stable run proof: %w", err)
	}

	var developmentStdout bytes.Buffer
	var developmentStderr bytes.Buffer
	developmentCode := cli.Run([]string{"run"}, root, &developmentStdout, &developmentStderr)
	if developmentCode != 42 || developmentStderr.Len() != 0 || developmentStdout.Len() != 0 {
		return 0, fmt.Errorf("stable development run returned code %d, stdout %q, stderr %q", developmentCode, developmentStdout.String(), developmentStderr.String())
	}

	sourcePath := filepath.Join(root, filepath.FromSlash(workspaceMainPath))
	originalSource, err := os.ReadFile(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("read source before stable run trap proof: %w", err)
	}
	reservedResultSource := []byte("module main;\n\nfn i32 main() {\n    return 131;\n}\n")
	if err := os.WriteFile(sourcePath, reservedResultSource, 0o600); err != nil {
		return 0, fmt.Errorf("write stable run reserved-result source: %w", err)
	}
	var reservedStdout bytes.Buffer
	var reservedStderr bytes.Buffer
	reservedCode := cli.Run([]string{"run"}, root, &reservedStdout, &reservedStderr)
	if reservedCode != 131 || reservedStderr.Len() != 0 || reservedStdout.Len() != 0 {
		return 0, fmt.Errorf("stable reserved-result run returned code %d, stdout %q, stderr %q", reservedCode, reservedStdout.String(), reservedStderr.String())
	}
	trapSource := []byte(`module main;

import std::io as io;

@effects(console)
fn i32 main() {
    let ref str message = "before trap\n";
    return switch (io::write(message)) {
        case .ok(written) : { if (written == message.len) { 42 / 0 } else { 1 } }
        case .err(error) : { if (error.status != 0) { 2 } else { 3 } }
    };
}
`)
	if err := os.WriteFile(sourcePath, trapSource, 0o600); err != nil {
		return 0, fmt.Errorf("write stable run trap source: %w", err)
	}
	var trapStdout bytes.Buffer
	var trapStderr bytes.Buffer
	trapCode := cli.Run([]string{"run"}, root, &trapStdout, &trapStderr)
	restoreErr := os.WriteFile(sourcePath, originalSource, 0o600)
	if restoreErr != nil {
		return 0, fmt.Errorf("restore source after stable run trap proof: %w", restoreErr)
	}
	if trapCode != cli.ExitProgramTrap || trapStdout.String() != "before trap\n" || trapStderr.String() != "trap: division-by-zero\n" {
		return 0, fmt.Errorf("stable trap run returned code %d, stdout %q, stderr %q", trapCode, trapStdout.String(), trapStderr.String())
	}

	var releaseStdout bytes.Buffer
	var releaseStderr bytes.Buffer
	releaseCode := cli.Run([]string{"run", "--release"}, root, &releaseStdout, &releaseStderr)
	if releaseCode != 42 || releaseStderr.Len() != 0 || releaseStdout.Len() != 0 {
		return 0, fmt.Errorf("stable release run returned code %d, stdout %q, stderr %q", releaseCode, releaseStdout.String(), releaseStderr.String())
	}
	restoredLock, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, fmt.Errorf("read restored lock after stable run proof: %w", err)
	}
	if !bytes.Equal(restoredLock, originalLock) {
		return 0, fmt.Errorf("stable run proof did not restore the original canonical lock")
	}
	afterRun, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read retained stable build after run proof: %w", err)
	}
	if !bytes.Equal(afterRun, retainedArtifact) {
		return 0, fmt.Errorf("stable run modified the retained build artifact")
	}
	return 4, nil
}

func measureWorkspaceProcessCache(root string, clockResolution int64) (buildmeasure.CacheReport, error) {
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		return buildmeasure.CacheReport{}, err
	}
	session, err := workspacecompiler.NewSession(workspacecompiler.Configuration{
		Target:           target.X8664WindowsMSVC,
		Cache:            packagecache.Limits{MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes},
		MaxArtifactBytes: processCacheMaxArtifactBytes,
	})
	if err != nil {
		return buildmeasure.CacheReport{}, err
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return buildmeasure.CacheReport{}, err
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	report := buildmeasure.CacheReport{
		Schema: buildmeasure.CacheSchema, Workload: "workspace_quickstart",
		Scope: "semantic-through-verified-mir", Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "observed-only-no-speedup-claim",
		Environment: buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: string(target.X8664WindowsMSVC)},
		Policy: buildmeasure.CachePolicy{
			MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes, MaxArtifactBytes: processCacheMaxArtifactBytes,
		},
	}
	for index, condition := range []string{buildmeasure.CacheConditionCold, buildmeasure.CacheConditionWarm} {
		started := time.Now()
		checked, err := session.Check(workspace)
		duration := time.Since(started)
		if err != nil {
			return buildmeasure.CacheReport{}, err
		}
		if !checked.Compiler.Verified {
			return buildmeasure.CacheReport{}, fmt.Errorf("%s process-cache sample did not produce verified MIR", condition)
		}
		outcome, err := interpreter.Execute(checked.Compiler.MIR, entry)
		if err != nil {
			return buildmeasure.CacheReport{}, err
		}
		stats := session.Stats()
		durationNS := duration.Nanoseconds()
		report.Samples = append(report.Samples, buildmeasure.CacheSample{
			Sequence: index + 1, Condition: condition, DurationNS: duration.Nanoseconds(),
			BelowClockResolution: durationNS < clockResolution,
			Outcome:              checked.Cache.Outcome, Packages: checked.Cache.Packages,
			Hits: checked.Cache.Hits, Misses: checked.Cache.Misses, Rejected: checked.Cache.Rejected,
			Stored: checked.Cache.Stored, StoreSkipped: checked.Cache.StoreSkipped,
			Lowered: checked.Cache.Lowered, ReassemblyFallback: checked.Cache.ReassemblyFallback,
			MIRFunctions: len(checked.Compiler.MIR.Functions), InterpreterResult: int(outcome.Value),
			RetainedEntries: stats.Store.Entries, RetainedBytes: stats.Store.Bytes, VerifiedHits: stats.VerifiedHits,
		})
	}
	return report, nil
}

func measureAtlasRewrite(root string, clockResolution int64) (buildmeasure.AtlasRewriteReport, error) {
	const maxMapBytesPerPackage = uint64(64 << 20)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	if len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 || !checked.Verified {
		return buildmeasure.AtlasRewriteReport{}, fmt.Errorf("Atlas rewrite workload did not produce clean verified MIR")
	}
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	_, partitions, err := atlasmap.PrepareWorkspace(workspace, checked, compilerBuild)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	shape, err := atlasRewriteShape(workspace, partitions)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	warmup, _, err := encodeAtlasPartitions(partitions, maxMapBytesPerPackage)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	report := buildmeasure.AtlasRewriteReport{
		Schema: buildmeasure.AtlasRewriteSchema, Workload: "workspace_quickstart",
		Scope: "canonical-fymap-encode-all-packages;workspace-analysis-map-planning-and-publication-excluded",
		Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		Decision: "raw-distribution-only-no-format-or-incremental-policy",
		Setup:    "one-untimed-warmup;fixed-preplanned-partitions;gc-uncontrolled", Iterations: buildmeasure.AtlasRewriteIterations,
		CompilerBuild: compilerBuild.String(), LogicalSchema: atlasmap.LogicalSchema,
		TransportSchema: atlasmap.TransportSchema, TransportEncoding: atlasmap.TransportEncoding,
		MaxMapBytesPerPackage: maxMapBytesPerPackage,
		Environment:           buildmeasure.AtlasRewriteEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH},
		Shape:                 shape,
	}
	for index := 0; index < buildmeasure.AtlasRewriteIterations; index++ {
		started := time.Now()
		encoded, encodedBytes, encodeErr := encodeAtlasPartitions(partitions, maxMapBytesPerPackage)
		duration := time.Since(started).Nanoseconds()
		if encodeErr != nil {
			return buildmeasure.AtlasRewriteReport{}, encodeErr
		}
		if !equalAtlasEncodings(warmup, encoded) {
			return buildmeasure.AtlasRewriteReport{}, fmt.Errorf("Atlas rewrite sample %d changed canonical map bytes", index+1)
		}
		report.Samples = append(report.Samples, buildmeasure.AtlasRewriteSample{
			Sequence: index + 1, DurationNS: duration, BelowClockResolution: duration < clockResolution,
			Packages: shape.Packages, MapRecords: shape.MapRecords, EncodedBytes: encodedBytes,
		})
	}
	if err := buildmeasure.ValidateAtlasRewrite(report); err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	return report, nil
}

func atlasRewriteShape(workspace workspaceloader.Workspace, partitions []atlasmap.Partition) (buildmeasure.AtlasRewriteShape, error) {
	shape := buildmeasure.AtlasRewriteShape{Packages: len(partitions)}
	for _, loaded := range workspace.Packages {
		for _, file := range loaded.Sources {
			shape.SourceFiles++
			if uint64(len(file.Bytes)) > ^uint64(0)-shape.SourceBytes {
				return buildmeasure.AtlasRewriteShape{}, fmt.Errorf("Atlas rewrite source-byte count overflow")
			}
			shape.SourceBytes += uint64(len(file.Bytes))
		}
	}
	for _, partition := range partitions {
		counts := partition.Header().Counts
		values := []int{counts.Files, counts.Modules, counts.Symbols, counts.Functions, counts.Relations, counts.Completeness}
		for _, value := range values {
			if value < 0 || uint64(value) > ^uint64(0)-shape.MapRecords {
				return buildmeasure.AtlasRewriteShape{}, fmt.Errorf("Atlas rewrite record count overflow")
			}
			shape.MapRecords += uint64(value)
		}
	}
	return shape, nil
}

func encodeAtlasPartitions(partitions []atlasmap.Partition, maxBytes uint64) ([][]byte, uint64, error) {
	result := make([][]byte, 0, len(partitions))
	var total uint64
	for _, partition := range partitions {
		encoded, err := atlasmap.Encode(partition, maxBytes)
		if err != nil {
			return nil, 0, err
		}
		if uint64(len(encoded)) > ^uint64(0)-total {
			return nil, 0, fmt.Errorf("Atlas rewrite encoded-byte count overflow")
		}
		total += uint64(len(encoded))
		result = append(result, encoded)
	}
	return result, total, nil
}

func equalAtlasEncodings(left, right [][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !bytes.Equal(left[index], right[index]) {
			return false
		}
	}
	return true
}

func measureWorkspaceProcessCacheEdits(sourceRoot string, clockResolution int64) (buildmeasure.CacheEditReport, error) {
	report := buildmeasure.CacheEditReport{
		Schema: buildmeasure.CacheEditSchema, Workload: "workspace_quickstart",
		Scope: "semantic-through-verified-mir", Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "raw-distributions-only-no-speedup-claim",
		Setup: "fresh-session-per-sample;edit-and-load-outside-timer", WarmupRequests: 1,
		IterationsPerCondition: buildmeasure.CacheEditIterationsPerCondition,
		Environment:            buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: string(target.X8664WindowsMSVC)},
		Policy: buildmeasure.CachePolicy{
			MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes, MaxArtifactBytes: processCacheMaxArtifactBytes,
		},
	}
	conditions := []string{
		buildmeasure.CacheEditNoOpWarm,
		buildmeasure.CacheEditPrivateRoot,
		buildmeasure.CacheEditPrivateDependency,
		buildmeasure.CacheEditPublicDependency,
	}
	for _, condition := range conditions {
		for iteration := 1; iteration <= buildmeasure.CacheEditIterationsPerCondition; iteration++ {
			sample, err := measureWorkspaceProcessCacheEditSample(sourceRoot, condition, iteration, clockResolution)
			if err != nil {
				return buildmeasure.CacheEditReport{}, fmt.Errorf("%s iteration %d: %w", condition, iteration, err)
			}
			sample.Sequence = len(report.Samples) + 1
			report.Samples = append(report.Samples, sample)
		}
	}
	return report, nil
}

func measureWorkspaceProcessCacheEditSample(sourceRoot, condition string, iteration int, clockResolution int64) (buildmeasure.CacheEditSample, error) {
	measurementRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	defer os.RemoveAll(measurementRoot)

	workspace, err := workspaceloader.Load(measurementRoot)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	session, err := workspacecompiler.NewSession(workspacecompiler.Configuration{
		Target:           target.X8664WindowsMSVC,
		Cache:            packagecache.Limits{MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes},
		MaxArtifactBytes: processCacheMaxArtifactBytes,
	})
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	warmup, err := session.Check(workspace)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	if !warmup.Compiler.Verified || warmup.Cache.Outcome != workspacecompiler.OutcomeFallback || !warmup.Cache.Lowered {
		return buildmeasure.CacheEditSample{}, fmt.Errorf("untimed warmup did not populate a verified fallback")
	}
	if condition != buildmeasure.CacheEditNoOpWarm {
		if err := mutateWorkspaceMeasurementFixture(measurementRoot, condition); err != nil {
			return buildmeasure.CacheEditSample{}, err
		}
		workspace, err = workspaceloader.Load(measurementRoot)
		if err != nil {
			return buildmeasure.CacheEditSample{}, err
		}
	}
	started := time.Now()
	checked, err := session.Check(workspace)
	duration := time.Since(started)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	if !checked.Compiler.Verified {
		return buildmeasure.CacheEditSample{}, fmt.Errorf("measured request did not produce verified MIR")
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	outcome, err := interpreter.Execute(checked.Compiler.MIR, entry)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	stats := session.Stats()
	durationNS := duration.Nanoseconds()
	return buildmeasure.CacheEditSample{
		Condition: condition, Iteration: iteration, DurationNS: durationNS, BelowClockResolution: durationNS < clockResolution,
		Outcome: checked.Cache.Outcome, Packages: checked.Cache.Packages,
		Hits: checked.Cache.Hits, Misses: checked.Cache.Misses, Rejected: checked.Cache.Rejected,
		Stored: checked.Cache.Stored, StoreSkipped: checked.Cache.StoreSkipped,
		Lowered: checked.Cache.Lowered, ReassemblyFallback: checked.Cache.ReassemblyFallback,
		MIRFunctions: len(checked.Compiler.MIR.Functions), InterpreterResult: int(outcome.Value),
		RetainedEntries: stats.Store.Entries, RetainedBytes: stats.Store.Bytes, VerifiedHits: stats.VerifiedHits,
	}, nil
}

func observeMonotonicClockResolution() (int64, error) {
	const (
		maximumObservations      = 10_000_000
		requiredTransitions      = 32
		cooperativeYieldInterval = 4_096
	)
	var minimumPositive int64
	transitions := 0
	previous := time.Now()
	for attempts := 0; attempts < maximumObservations && transitions < requiredTransitions; attempts++ {
		if attempts != 0 && attempts%cooperativeYieldInterval == 0 {
			runtime.Gosched()
		}
		current := time.Now()
		if elapsed := current.Sub(previous).Nanoseconds(); elapsed > 0 {
			if minimumPositive == 0 || elapsed < minimumPositive {
				minimumPositive = elapsed
			}
			transitions++
		}
		previous = current
	}
	if transitions < requiredTransitions {
		return 0, fmt.Errorf("monotonic clock produced %d of %d required positive transitions during %d resolution observations", transitions, requiredTransitions, maximumObservations)
	}
	return minimumPositive, nil
}

func copyWorkspaceMeasurementFixture(sourceRoot string) (string, error) {
	root, err := os.MkdirTemp("", "fairyc-process-cache-edit-")
	if err != nil {
		return "", err
	}
	for _, name := range []string{workspaceManifestPath, workspaceMainPath, workspaceMemberManifestPath, workspaceMemberSourcePath} {
		contents, readErr := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(name)))
		if readErr != nil {
			os.RemoveAll(root)
			return "", readErr
		}
		destination := filepath.Join(root, filepath.FromSlash(name))
		if mkdirErr := os.MkdirAll(filepath.Dir(destination), 0o700); mkdirErr != nil {
			os.RemoveAll(root)
			return "", mkdirErr
		}
		if writeErr := os.WriteFile(destination, contents, 0o600); writeErr != nil {
			os.RemoveAll(root)
			return "", writeErr
		}
	}
	return root, nil
}

func mutateWorkspaceMeasurementFixture(root, condition string) error {
	path := workspaceMainPath
	needle := "\nfn i32 main()"
	replacement := "\nstruct cache_private_root { i32 value; }\n\nfn i32 main()"
	if condition == buildmeasure.CacheEditPrivateDependency || condition == buildmeasure.CacheEditPublicDependency {
		path = workspaceMemberSourcePath
		needle = "\npub fn i32 value()"
		replacement = "\nstruct cache_private_dependency { i32 value; }\n\npub fn i32 value()"
	}
	absolute := filepath.Join(root, filepath.FromSlash(path))
	contents, err := os.ReadFile(absolute)
	if err != nil {
		return err
	}
	updated := string(contents)
	switch condition {
	case buildmeasure.CacheEditPrivateRoot, buildmeasure.CacheEditPrivateDependency:
		updated = strings.Replace(updated, needle, replacement, 1)
	case buildmeasure.CacheEditPublicDependency:
		updated += "\npub fn i32 cache_public_probe() {\n    return 7;\n}\n"
	default:
		return fmt.Errorf("unknown cache edit condition %q", condition)
	}
	if updated == string(contents) {
		return fmt.Errorf("cache edit condition %q did not change its source", condition)
	}
	return os.WriteFile(absolute, []byte(updated), 0o600)
}

func measureStandaloneWorkspaceCache(executablePath, sourceRoot string, clockResolution int64) (buildmeasure.PersistentCacheProcessReport, error) {
	workspaceRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("copy standalone persistent-cache fixture: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	prepareCode, prepareStdout, prepareStderr, err := runStandaloneCommand(workspaceRoot, executablePath, "experimental", "check-workspace")
	if err != nil {
		return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("prepare standalone persistent-cache lockfile: %w", err)
	}
	if prepareCode != cli.ExitSuccess || prepareStderr != "" {
		return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("prepare standalone persistent-cache lockfile returned code %d, stdout %q, stderr %q", prepareCode, prepareStdout, prepareStderr)
	}

	measurement := buildmeasure.PersistentCacheProcessReport{
		Schema: buildmeasure.PersistentCacheProcessSchema, Workload: "workspace_quickstart",
		Scope: "caller-process-run-through-exit;workspace-and-cache-io-included", Clock: "caller-monotonic-wall-nanoseconds",
		ClockResolutionNS: clockResolution, CacheImplementation: "bounded-immutable-disk-package-mir-v2",
		Decision: "raw-paired-distributions-only-no-speedup-claim",
		Setup:    "fresh-cache-per-pair;separate-fairyc-process-per-sample", RetryPolicy: "none;preserve-first-observation",
		IterationsPerCondition: buildmeasure.PersistentCacheProcessIterations,
		Environment:            buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc"},
		Policy: buildmeasure.PersistentCachePolicy{
			MaxEntries: 8, MaxBytes: 67108864, MaxRecordBytes: 8388608,
			MaxArtifactBytes: 4194304, TempMaxAgeSecs: 60,
		},
	}
	for pair := 1; pair <= buildmeasure.PersistentCacheProcessIterations; pair++ {
		cacheDirectory := fmt.Sprintf("build/persistent-cache-pair-%d", pair)
		arguments := []string{
			"experimental", "check-workspace-cache", "--locked",
			"--target", "x86_64-windows-msvc",
			"--cache-dir", cacheDirectory,
			"--cache-max-entries", "8",
			"--cache-max-bytes", "67108864",
			"--cache-max-record-bytes", "8388608",
			"--cache-max-artifact-bytes", "4194304",
			"--cache-temp-max-age-seconds", "60",
		}
		for _, condition := range []string{buildmeasure.PersistentCacheProcessCold, buildmeasure.PersistentCacheProcessWarm} {
			started := time.Now()
			code, stdout, stderr, runErr := runStandaloneCommand(workspaceRoot, executablePath, arguments...)
			duration := time.Since(started).Nanoseconds()
			if runErr != nil {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("execute standalone persistent-cache pair %d %s: %w", pair, condition, runErr)
			}
			if code != cli.ExitSuccess || stderr != "" {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("standalone persistent-cache pair %d %s returned code %d, stdout %q, stderr %q", pair, condition, code, stdout, stderr)
			}
			var report standaloneWorkspaceCacheReport
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("decode standalone persistent-cache pair %d %s report: %w", pair, condition, err)
			}
			if report.SchemaVersion != "fairyc.experimental.workspace-cache-check/v0" || report.Cache.Storage != workspacecompiler.StorageDisk {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("standalone persistent-cache pair %d %s report is incompatible: %+v", pair, condition, report)
			}
			if measurement.CompilerBuild == "" {
				measurement.CompilerBuild = report.CompilerBuild
			} else if measurement.CompilerBuild != report.CompilerBuild {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("standalone persistent-cache compiler identity changed within measurement")
			}
			measurement.Samples = append(measurement.Samples, buildmeasure.PersistentCacheProcessSample{
				Sequence: len(measurement.Samples) + 1, Pair: pair, Condition: condition,
				DurationNS: duration, BelowClockResolution: duration < clockResolution,
				Outcome: report.Cache.Outcome, Packages: report.Cache.Packages,
				Hits: report.Cache.Hits, Misses: report.Cache.Misses, Rejected: report.Cache.Rejected,
				Stored: report.Cache.Stored, StoreSkipped: report.Cache.StoreSkipped, StoreFailed: report.Cache.StoreFailed,
				Lowered: report.Cache.Lowered, ReassemblyFallback: report.Cache.ReassemblyFallback,
				MIRFunctions: report.MIRFunctions, RetainedEntries: report.Cache.RetainedEntries,
				RetainedBytes: report.Cache.RetainedBytes, RecordHits: report.Cache.RecordHits,
				RecordMisses: report.Cache.RecordMisses, RecordRejected: report.Cache.RecordRejected,
				Evictions: report.Cache.Evictions, VerifiedHits: report.Cache.VerifiedHits,
			})
		}
	}
	if err := buildmeasure.ValidatePersistentCacheProcess(measurement); err != nil {
		return buildmeasure.PersistentCacheProcessReport{}, err
	}
	return measurement, nil
}

func measureStandaloneWorkspaceCacheGrowth(executablePath, sourceRoot string) (buildmeasure.PersistentCacheGrowthReport, error) {
	workspaceRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("copy standalone persistent-cache growth fixture: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	prepareCode, prepareStdout, prepareStderr, err := runStandaloneCommand(workspaceRoot, executablePath, "experimental", "check-workspace")
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("prepare standalone persistent-cache growth lockfile: %w", err)
	}
	if prepareCode != cli.ExitSuccess || prepareStderr != "" {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("prepare standalone persistent-cache growth lockfile returned code %d, stdout %q, stderr %q", prepareCode, prepareStdout, prepareStderr)
	}
	baseRoot, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(workspaceMainPath)))
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("read persistent-cache growth root source: %w", err)
	}
	baseDependency, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(workspaceMemberSourcePath)))
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("read persistent-cache growth dependency source: %w", err)
	}
	report := buildmeasure.PersistentCacheGrowthReport{
		Schema: buildmeasure.PersistentCacheGrowthSchema, Workload: "workspace_quickstart-retention-sequence",
		Scope:               "retained-disk-state-after-completed-process;source-mutation-excluded",
		CacheImplementation: "bounded-immutable-disk-package-mir-v2", Decision: "bounded-growth-control-only-no-default-policy",
		Setup: "one-cache;separate-fairyc-process-per-sample;mutation-before-process", RetryPolicy: "none;preserve-first-observation",
		Environment: buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc"},
		Policy: buildmeasure.PersistentCachePolicy{
			MaxEntries: 6, MaxBytes: 67108864, MaxRecordBytes: 8388608,
			MaxArtifactBytes: 4194304, TempMaxAgeSecs: 60,
		},
	}
	conditions := []string{
		buildmeasure.PersistentGrowthBaselineCold,
		buildmeasure.PersistentGrowthPrivateRootOne,
		buildmeasure.PersistentGrowthPrivateRootTwo,
		buildmeasure.PersistentGrowthPrivateDependency,
		buildmeasure.PersistentGrowthPublicDependency,
		buildmeasure.PersistentGrowthPublicNoOp,
		buildmeasure.PersistentGrowthPrivateRootPublic,
		buildmeasure.PersistentGrowthPrivateRootNoOp,
		buildmeasure.PersistentGrowthRepeatedNoOp,
	}
	arguments := []string{
		"experimental", "check-workspace-cache",
		"--target", "x86_64-windows-msvc",
		"--cache-dir", "build/persistent-cache-growth",
		"--cache-max-entries", "6",
		"--cache-max-bytes", "67108864",
		"--cache-max-record-bytes", "8388608",
		"--cache-max-artifact-bytes", "4194304",
		"--cache-temp-max-age-seconds", "60",
	}
	previousEntries := 0
	previousBytes := uint64(0)
	for index, condition := range conditions {
		if err := writePersistentCacheGrowthSources(workspaceRoot, condition, baseRoot, baseDependency); err != nil {
			return buildmeasure.PersistentCacheGrowthReport{}, err
		}
		code, stdout, stderr, runErr := runStandaloneCommand(workspaceRoot, executablePath, arguments...)
		if runErr != nil {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("execute standalone persistent-cache growth sample %d %s: %w", index+1, condition, runErr)
		}
		if code != cli.ExitSuccess || stderr != "" {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("standalone persistent-cache growth sample %d %s returned code %d, stdout %q, stderr %q", index+1, condition, code, stdout, stderr)
		}
		var observation standaloneWorkspaceCacheReport
		if err := json.Unmarshal([]byte(stdout), &observation); err != nil {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("decode standalone persistent-cache growth sample %d %s: %w", index+1, condition, err)
		}
		if observation.SchemaVersion != "fairyc.experimental.workspace-cache-check/v0" || observation.Cache.Storage != workspacecompiler.StorageDisk {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("standalone persistent-cache growth sample %d %s is incompatible: %+v", index+1, condition, observation)
		}
		if report.CompilerBuild == "" {
			report.CompilerBuild = observation.CompilerBuild
		} else if report.CompilerBuild != observation.CompilerBuild {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("standalone persistent-cache compiler identity changed within growth measurement")
		}
		byteDelta := int64(observation.Cache.RetainedBytes) - int64(previousBytes)
		report.Samples = append(report.Samples, buildmeasure.PersistentCacheGrowthSample{
			Sequence: index + 1, Condition: condition,
			Outcome: observation.Cache.Outcome, Packages: observation.Cache.Packages,
			Hits: observation.Cache.Hits, Misses: observation.Cache.Misses, Rejected: observation.Cache.Rejected,
			Stored: observation.Cache.Stored, StoreSkipped: observation.Cache.StoreSkipped, StoreFailed: observation.Cache.StoreFailed,
			Lowered: observation.Cache.Lowered, ReassemblyFallback: observation.Cache.ReassemblyFallback,
			MIRFunctions: observation.MIRFunctions, RetainedEntries: observation.Cache.RetainedEntries,
			RetainedBytes:      observation.Cache.RetainedBytes,
			RetainedEntryDelta: observation.Cache.RetainedEntries - previousEntries, RetainedByteDelta: byteDelta,
			RecordHits: observation.Cache.RecordHits, RecordMisses: observation.Cache.RecordMisses,
			RecordRejected: observation.Cache.RecordRejected, Evictions: observation.Cache.Evictions,
			VerifiedHits: observation.Cache.VerifiedHits,
		})
		previousEntries = observation.Cache.RetainedEntries
		previousBytes = observation.Cache.RetainedBytes
	}
	if err := buildmeasure.ValidatePersistentCacheGrowth(report); err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, err
	}
	return report, nil
}

func writePersistentCacheGrowthSources(root, condition string, baseRoot, baseDependency []byte) error {
	rootSource := string(baseRoot)
	dependencySource := string(baseDependency)
	switch condition {
	case buildmeasure.PersistentGrowthBaselineCold:
	case buildmeasure.PersistentGrowthPrivateRootOne:
		rootSource = strings.Replace(rootSource, "\nfn i32 main()", "\nstruct retention_root_v1 { i32 value; }\n\nfn i32 main()", 1)
	case buildmeasure.PersistentGrowthPrivateRootTwo:
		rootSource = strings.Replace(rootSource, "\nfn i32 main()", "\nstruct retention_root_v2 { i32 first; i32 second; }\n\nfn i32 main()", 1)
	case buildmeasure.PersistentGrowthPrivateDependency:
		dependencySource = strings.Replace(dependencySource, "\npub fn i32 value()", "\nstruct retention_dependency_v1 { i32 value; }\n\npub fn i32 value()", 1)
	case buildmeasure.PersistentGrowthPublicDependency, buildmeasure.PersistentGrowthPublicNoOp:
		dependencySource += "\npub fn i32 retention_public_v1() {\n    return 7;\n}\n"
	case buildmeasure.PersistentGrowthPrivateRootPublic, buildmeasure.PersistentGrowthPrivateRootNoOp, buildmeasure.PersistentGrowthRepeatedNoOp:
		rootSource = strings.Replace(rootSource, "\nfn i32 main()", "\nstruct retention_root_public_v1 { i32 value; }\n\nfn i32 main()", 1)
		dependencySource += "\npub fn i32 retention_public_v1() {\n    return 7;\n}\n"
	default:
		return fmt.Errorf("unknown persistent-cache growth condition %q", condition)
	}
	if !strings.Contains(rootSource, "fn i32 main()") || !strings.Contains(dependencySource, "pub fn i32 value()") {
		return fmt.Errorf("persistent-cache growth condition %q did not preserve required entry functions", condition)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(workspaceMainPath)), []byte(rootSource), 0o600); err != nil {
		return fmt.Errorf("write persistent-cache growth root source: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(workspaceMemberSourcePath)), []byte(dependencySource), 0o600); err != nil {
		return fmt.Errorf("write persistent-cache growth dependency source: %w", err)
	}
	return nil
}

func measureStandaloneWorkspaceSession(executablePath, sourceRoot string) (buildmeasure.SessionProcessReport, error) {
	workspaceRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("copy standalone workspace-session fixture: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)

	contextValue, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(
		contextValue,
		executablePath,
		"experimental", "check-workspace-session",
		"--target", "x86_64-windows-msvc", "--max-requests", "8",
		"--cache-max-entries", "8", "--cache-max-bytes", "33554432", "--cache-max-artifact-bytes", "8388608",
	)
	command.Dir = workspaceRoot
	stdin, err := command.StdinPipe()
	if err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("open standalone workspace-session stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("open standalone workspace-session stdout: %w", err)
	}
	var stderr boundedBuffer
	stderr.limit = subprocessOutputLimit
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("start standalone workspace session: %w", err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = stdin.Close()
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	probe, err := openProcessMemoryProbe(command.Process.Pid)
	if err != nil {
		return buildmeasure.SessionProcessReport{}, err
	}
	defer probe.close()

	report := buildmeasure.SessionProcessReport{
		Schema: buildmeasure.SessionProcessSchema, Workload: "workspace_quickstart-editor-sequence",
		Scope: "request-write-through-response-read;mutation-excluded", Clock: "caller-monotonic-wall-nanoseconds",
		Setup: "one-standalone-fairyc-process;sequential-requests", RetryPolicy: "none;preserve-first-observation",
		MemoryMetric:        "windows-process-memory-counters-peak-working-set-and-pagefile-bytes",
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "local-observation-only-no-reference-budget-or-speedup-claim",
		Environment: buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc"},
		Policy:      buildmeasure.CachePolicy{MaxEntries: 8, MaxBytes: 33554432, MaxArtifactBytes: 8388608},
	}
	conditions := []string{
		buildmeasure.SessionProcessCold,
		buildmeasure.SessionProcessNoOp,
		buildmeasure.SessionProcessPrivateRoot,
		buildmeasure.SessionProcessNoOpAfterRoot,
		buildmeasure.SessionProcessPrivateDependency,
		buildmeasure.SessionProcessNoOpAfterPrivate,
		buildmeasure.SessionProcessPublicDependency,
		buildmeasure.SessionProcessNoOpAfterPublic,
	}
	reader := bufio.NewReaderSize(stdout, subprocessOutputLimit)
	for index, condition := range conditions {
		switch condition {
		case buildmeasure.SessionProcessPrivateRoot:
			err = mutateWorkspaceMeasurementFixture(workspaceRoot, buildmeasure.CacheEditPrivateRoot)
		case buildmeasure.SessionProcessPrivateDependency:
			err = mutateWorkspaceMeasurementFixture(workspaceRoot, buildmeasure.CacheEditPrivateDependency)
		case buildmeasure.SessionProcessPublicDependency:
			err = mutateWorkspaceMeasurementFixture(workspaceRoot, buildmeasure.CacheEditPublicDependency)
		default:
			err = nil
		}
		if err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("prepare standalone workspace-session %s: %w", condition, err)
		}
		sequence := index + 1
		request := fmt.Sprintf("{\"schemaVersion\":\"fairyc.experimental.workspace-check-session/v0\",\"kind\":\"check\",\"sequence\":%d}\n", sequence)
		started := time.Now()
		if _, err := io.WriteString(stdin, request); err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("write standalone workspace-session request %d: %w", sequence, err)
		}
		line, err := reader.ReadBytes('\n')
		duration := time.Since(started).Nanoseconds()
		if err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("read standalone workspace-session response %d: %w", sequence, err)
		}
		var response standaloneWorkspaceSessionResponse
		if err := json.Unmarshal(line, &response); err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("decode standalone workspace-session response %d: %w", sequence, err)
		}
		if response.Sequence != sequence || response.Status != "ok" || response.ExitCode != cli.ExitSuccess {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf(
				"standalone workspace-session response %d is not successful: %s; stderr %q",
				sequence,
				line,
				stderr.String(),
			)
		}
		report.Samples = append(report.Samples, buildmeasure.SessionProcessSample{
			Sequence: sequence, Condition: condition, DurationNS: duration,
			Outcome: response.Cache.Outcome, Hits: response.Cache.Hits, Misses: response.Cache.Misses,
			Lowered: response.Cache.Lowered, MIRFunctions: response.MIRFunctions,
			RetainedEntries: response.Cache.RetainedEntries, RetainedBytes: response.Cache.RetainedBytes,
			VerifiedHits: response.Cache.VerifiedHits,
		})
		workingSet, pagefile, err := probe.snapshot()
		if err != nil {
			return buildmeasure.SessionProcessReport{}, err
		}
		if workingSet > report.Memory.PeakWorkingSetBytes {
			report.Memory.PeakWorkingSetBytes = workingSet
		}
		if pagefile > report.Memory.PeakPagefileBytes {
			report.Memory.PeakPagefileBytes = pagefile
		}
	}
	if err := stdin.Close(); err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("close standalone workspace-session stdin: %w", err)
	}
	if err := command.Wait(); err != nil {
		waited = true
		if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("standalone workspace session exceeded timeout")
		}
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("standalone workspace session failed: %w; stderr %q", err, stderr.String())
	}
	waited = true
	if stderr.Len() != 0 {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("standalone workspace session produced stderr %q", stderr.String())
	}
	return report, nil
}

func runStandaloneCommand(directory, executablePath string, arguments ...string) (int, string, string, error) {
	return runStandaloneCommandWithInput(directory, executablePath, "", arguments...)
}

func runStandaloneCommandWithInput(directory, executablePath, input string, arguments ...string) (int, string, string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath, arguments...)
	command.Dir = directory
	command.Stdin = strings.NewReader(input)
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
		handlerSymbol, ok := unit.HandlerSymbolForProfile(profile.Name)
		if !ok {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe has no handler identity for profile %q", profile.Name)
		}
		handlerReport, err := pecoff.InspectAMD64HandlerLink(artifact.Objects[1].Bytes, artifact.Bytes, artifact.LinkMap, pecoff.AMD64HandlerLinkSpec{
			SourceSymbol: unit.FrameSymbol, SourceObject: unit.FrameObject, HandlerSymbol: handlerSymbol,
		})
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind handler relocation with %s: %w", profile.Name, err)
		}
		if handlerReport.LinkedHandlerRVA != handlerReport.ExpectedHandlerRVA || handlerReport.HandlerMapOwner == "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind handler relocation with %s returned an incomplete receipt", profile.Name)
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

// generatedFairyProbeImplementation builds the test-only standard-library closure for a
// generated-Fairy probe: the existing probe-only SystemHeap allocator adapter, the exact
// standard-library components selected for the emitted C foreign symbols, and the canonical
// scalar runtime for emitted trap calls. No product source, symbol contract, or ordinary
// product linkage is changed.
func generatedFairyProbeImplementation(repositoryRoot string, symbols []string) (string, error) {
	if len(symbols) == 0 {
		return "", fmt.Errorf("generated-Fairy probe closure requires at least one referenced symbol")
	}
	implementation, err := cstdlib.WindowsMSVCSourceForSymbols(symbols)
	if err != nil {
		return "", fmt.Errorf("select generated-Fairy probe standard-library closure: %w", err)
	}
	adapterPath := filepath.Join(repositoryRoot, filepath.FromSlash("internal/cstdlib/stage0_system_heap_probe_allocator.c"))
	adapterBytes, err := os.ReadFile(adapterPath)
	if err != nil {
		return "", fmt.Errorf("read generated-Fairy probe SystemHeap adapter: %w", err)
	}
	adapter := string(adapterBytes)
	runtimeSource, err := cruntime.ScalarSource(generatedFairyProbeTrapExitBase)
	if err != nil {
		return "", fmt.Errorf("render generated-Fairy probe scalar runtime: %w", err)
	}
	return generatedFairyProbeClosureSource(adapter, implementation, runtimeSource)
}

// generatedFairyProbeClosureSource joins the three canonical C sources and rejects incomplete or
// duplicated closures before they reach the native toolchain.
func generatedFairyProbeClosureSource(adapter, implementation, runtimeSource string) (string, error) {
	if adapter == "" || strings.Contains(adapter, "\r") || !strings.HasSuffix(adapter, "\n") ||
		implementation == "" || strings.Contains(implementation, "\r") || !strings.HasSuffix(implementation, "\n") ||
		runtimeSource == "" || strings.Contains(runtimeSource, "\r") || !strings.HasSuffix(runtimeSource, "\n") {
		return "", fmt.Errorf("generated-Fairy probe closure source is empty or non-canonical")
	}
	if !strings.Contains(adapter, "fy_std_system_heap_probe_fail_after_allocations_v0") ||
		!strings.Contains(adapter, "fy_std_system_heap_probe_live_allocations_v0") {
		return "", fmt.Errorf("generated-Fairy probe SystemHeap adapter lost its failure controls")
	}
	if strings.Contains(adapter+implementation, "void fy_runtime_trap(") {
		return "", fmt.Errorf("generated-Fairy probe closure already defines fy_runtime_trap")
	}
	if !strings.Contains(runtimeSource, "void fy_runtime_trap(") {
		return "", fmt.Errorf("generated-Fairy probe scalar runtime lost its trap definition")
	}
	return adapter + "\n" + implementation + "\n" + runtimeSource, nil
}

func runGeneratedFairyStringFromBufferProbe(repositoryRoot, sourceRoot string, toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	manifest, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdTextFromBufferManifestPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy Buffer-to-String conversion manifest: %w", err)
	}
	checkedInMain, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(checkedInStdTextFromBufferMainPath)))
	if err != nil {
		return 0, 0, fmt.Errorf("read generated-Fairy Buffer-to-String conversion entry: %w", err)
	}
	workspaceRoot, err := os.MkdirTemp("", "fairyc-string-from-buffer-probe-")
	if err != nil {
		return 0, 0, fmt.Errorf("create generated-Fairy Buffer-to-String conversion workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	for name, contents := range map[string][]byte{
		"fairy.toml":  manifest,
		"src/main.fy": checkedInMain,
	} {
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return 0, 0, fmt.Errorf("create generated-Fairy Buffer-to-String conversion path %s: %w", name, err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			return 0, 0, fmt.Errorf("write generated-Fairy Buffer-to-String conversion path %s: %w", name, err)
		}
	}
	workspace, err := workspaceloader.Load(workspaceRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("load generated-Fairy Buffer-to-String conversion workspace: %w", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return 0, 0, fmt.Errorf("prepare generated-Fairy Buffer-to-String conversion semantic input: %w", err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return 0, 0, fmt.Errorf("compile generated-Fairy Buffer-to-String conversion source: %w", err)
	}
	if !checked.Verified || checked.Semantic.CompletedThrough != semantic.PhaseFlow || checked.Semantic.FailedPhase != semantic.PhaseNone ||
		len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return 0, 0, fmt.Errorf("generated-Fairy Buffer-to-String conversion source did not produce clean verified MIR: phase=%q failed=%q diagnostics=%+v unsupported=%+v", checked.Semantic.CompletedThrough, checked.Semantic.FailedPhase, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve generated-Fairy Buffer-to-String conversion root package: %w", err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		return 0, 0, fmt.Errorf("emit generated-Fairy Buffer-to-String conversion C: %w", err)
	}
	probeSource, err := generatedFairyStringFromBufferProbeSource(unit.Source)
	if err != nil {
		return 0, 0, err
	}
	implementation, err := generatedFairyProbeImplementation(repositoryRoot, unit.ForeignSymbols)
	if err != nil {
		return 0, 0, err
	}
	images, err := runWindowsSeparateTranslationUnitProbe(toolchain, "generated-Fairy Buffer-to-String conversion", probeSource, implementation, 0, cdriver.C11ConsumerMode)
	if err != nil {
		return images, images, err
	}
	return images, images, nil
}

func generatedFairyStringFromBufferProbeSource(generated string) (string, error) {
	if generated == "" || !strings.Contains(generated, "int main(void)") {
		return "", fmt.Errorf("generated-Fairy Buffer-to-String conversion C must contain a main wrapper")
	}
	for _, required := range []string{
		"fy_std_string_from_buffer_storage_v0(",
		"fy_std_string_drop_storage_v0(",
		"fy_std_buffer_drop_v0(",
	} {
		if strings.Count(generated, required) < 2 {
			return "", fmt.Errorf("generated-Fairy Buffer-to-String conversion C lost a required call to %s (expected prototype and body call)", strings.TrimSuffix(required, "("))
		}
	}
	return "#define main fy_generated_string_from_buffer_main\n" + generated + `
#undef main

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    fy_std_system_heap_probe_fail_after_allocations_v0(2);
    if (fy_generated_string_from_buffer_main() != 42) {
        return 10;
    }
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    return 0;
}
`, nil
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

func runWindowsProcessArgsDecodeProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	implementation, err := cstdlib.WindowsProcessArgsSource()
	if err != nil {
		return 0, 0, fmt.Errorf("load Windows process args implementation: %w", err)
	}
	probeSource := implementation + "\n" + windowsProcessArgsDecodeProbeMain
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, probeSource)
		if err != nil {
			return executions, 0, fmt.Errorf("build Windows process args decode probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 1 || artifact.Objects[0].Name != "probe.obj" {
			return executions, 0, fmt.Errorf("Windows process args decode probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, 0, fmt.Errorf("inspect Windows process args decode probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, 0, fmt.Errorf("inspect Windows process args decode probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCProbe(artifact)
		if err != nil {
			return executions, 0, fmt.Errorf("execute Windows process args decode probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, 0, fmt.Errorf("Windows process args decode probe with %s returned %d, stdout %q, stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
		outcomeWithArgs, err := cdriver.ExecuteWindowsMSVCTargetWithInputAndArgs(artifact, strings.NewReader(""), processArgsDecodeRoundTripArguments)
		if err != nil {
			return executions, 0, fmt.Errorf("execute Windows process args decode probe with arguments with %s: %w", profile.Name, err)
		}
		if outcomeWithArgs.ExitCode != 0 || outcomeWithArgs.Stdout != "" || outcomeWithArgs.Stderr != "" {
			return executions, 0, fmt.Errorf("Windows process args decode probe with arguments with %s returned %d, stdout %q, stderr %q", profile.Name, outcomeWithArgs.ExitCode, outcomeWithArgs.Stdout, outcomeWithArgs.Stderr)
		}
		executions++
	}
	return executions, executions, nil
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

func runWindowsRelocationProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	unit := windowsrelocationprobe.Generate()
	variantObject, err := pecoff.BuildAMD64REL32VariantProbeObject(unit.VariantSource, unit.VariantTarget)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 REL32 variants object: %w", err)
	}
	weakObject, err := pecoff.BuildAMD64WeakExternalProbeObject(
		unit.WeakFallbackEntry, unit.WeakOverrideEntry, unit.WeakMissing, unit.WeakOverride,
		unit.WeakMissingFallback, unit.WeakOverrideFallback,
	)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 weak-external object: %w", err)
	}
	overflowObject, err := pecoff.BuildAMD64RelocationOverflowProbeObject(unit.OverflowSource)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 relocation-overflow object: %w", err)
	}
	bigObject, err := pecoff.BuildAMD64BigObjectProbe(unit.BigObjectSource)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 bigobj: %w", err)
	}
	rel32Spec := pecoff.AMD64REL32LinkSpec{
		SourceSymbol: unit.SourceSymbol, TargetSymbol: unit.TargetSymbol,
		ReceiptSection: unit.ReceiptSection, ReceiptMagic: unit.ReceiptMagic,
	}
	addressSpec := pecoff.AMD64AddressLinkSpec{
		SourceSymbol: unit.SourceSymbol, TargetSymbol: unit.TargetSymbol, ReceiptSymbol: unit.ReceiptSymbol,
		SourceObject: unit.SourceObject, TargetObject: unit.TargetObject,
		ReceiptSection: unit.ReceiptSection, ReceiptMagic: unit.ReceiptMagic,
	}
	variantSpec := pecoff.AMD64REL32VariantsSpec{
		SourceSymbol: unit.VariantSource, TargetSymbol: unit.VariantTarget,
		SourceObject: unit.VariantObject, TargetObject: unit.TargetObject,
	}
	weakSpec := pecoff.AMD64WeakExternalSpec{
		ObjectName: unit.WeakObject, TargetObject: unit.TargetObject,
		FallbackEntry: unit.WeakFallbackEntry, OverrideEntry: unit.WeakOverrideEntry,
		MissingWeak: unit.WeakMissing, OverrideWeak: unit.WeakOverride,
		MissingFallback: unit.WeakMissingFallback, OverrideFallback: unit.WeakOverrideFallback,
	}
	overflowSpec := pecoff.AMD64RelocationOverflowSpec{SourceSymbol: unit.OverflowSource, SourceObject: unit.OverflowObject}
	bigObjectSpec := pecoff.AMD64BigObjectSpec{SourceSymbol: unit.BigObjectSource, SourceObject: unit.BigObject}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCRelocationProbe(toolchain, profile, unit.CallerSource, unit.TargetSource, variantObject, weakObject, overflowObject, bigObject)
		if err != nil {
			return executions, fmt.Errorf("build Windows MSVC REL32 probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 6 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "target.obj" ||
			artifact.Objects[2].Name != unit.VariantObject || artifact.Objects[3].Name != unit.WeakObject ||
			artifact.Objects[4].Name != unit.OverflowObject || artifact.Objects[5].Name != unit.BigObject {
			return executions, fmt.Errorf("Windows MSVC REL32 probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		callerCodeView, err := pecoff.InspectCodeViewRelocations(artifact.Objects[0].Bytes)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC caller CodeView relocations with %s: %w", profile.Name, err)
		}
		if !hasCodeViewRelocationPair(callerCodeView, unit.SourceSymbol) {
			return executions, fmt.Errorf("Windows MSVC caller CodeView relocations with %s omit %q; pairs=%+v", profile.Name, unit.SourceSymbol, callerCodeView.Pairs)
		}
		targetCodeView, err := pecoff.InspectCodeViewRelocations(artifact.Objects[1].Bytes)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC target CodeView relocations with %s: %w", profile.Name, err)
		}
		if !hasCodeViewRelocationPair(targetCodeView, unit.TargetSymbol) || !hasCodeViewRelocationPair(targetCodeView, unit.VariantTarget) {
			return executions, fmt.Errorf("Windows MSVC target CodeView relocations with %s omit exact target symbols; pairs=%+v", profile.Name, targetCodeView.Pairs)
		}
		if _, err := pecoff.InspectProgramDatabaseAssociation(artifact.Bytes, artifact.ProgramDatabase); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC PE/PDB association with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 probe unwind data with %s: %w", profile.Name, err)
		}
		report, err := pecoff.InspectAMD64REL32Link(artifact.Objects[0].Bytes, artifact.Bytes, rel32Spec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 object-to-image link with %s: %w", profile.Name, err)
		}
		if report.ObjectSection == "" || report.LinkedDisplacement != report.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC REL32 probe with %s returned an incomplete link receipt", profile.Name)
		}
		symbolReceipt, err := symbolicateWindowsProgram(artifact.Bytes, artifact.ProgramDatabase, report.SourceRVA)
		if err != nil {
			return executions, fmt.Errorf("symbolicate Windows MSVC REL32 source with %s: %w", profile.Name, err)
		}
		if err := validateWindowsSymbolication(symbolReceipt, unit.SourceSymbol, "caller.c", unit.SourceLine); err != nil {
			return executions, fmt.Errorf("validate Windows MSVC REL32 symbolication with %s: %w", profile.Name, err)
		}
		addressReport, err := pecoff.InspectAMD64AddressLink(artifact.Objects[0].Bytes, artifact.Bytes, artifact.LinkMap, addressSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC ADDR64 and unwind ADDR32NB object-to-image link with %s: %w", profile.Name, err)
		}
		if addressReport.ADDR64LinkedValue != addressReport.ADDR64ExpectedValue ||
			addressReport.ADDR32NBLinkedBeginRVA != addressReport.ADDR32NBExpectedBeginRVA ||
			addressReport.RuntimeFunctionEndRVA != addressReport.ExpectedFunctionEndRVA ||
			addressReport.UnwindInfoRVA != addressReport.ExpectedUnwindInfoRVA {
			return executions, fmt.Errorf("Windows MSVC address-relocation probe with %s returned an incomplete link receipt", profile.Name)
		}
		variantReport, err := pecoff.InspectAMD64REL32VariantsLink(artifact.Objects[2].Bytes, artifact.Bytes, artifact.LinkMap, variantSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC REL32 variants object-to-image link with %s: %w", profile.Name, err)
		}
		if len(variantReport.Variants) != 6 || variantReport.SourceRVA == 0 || variantReport.TargetRVA == 0 {
			return executions, fmt.Errorf("Windows MSVC REL32 variants probe with %s returned an incomplete receipt", profile.Name)
		}
		for _, variant := range variantReport.Variants {
			if variant.LinkedDisplacement != variant.ExpectedDisplacement {
				return executions, fmt.Errorf("Windows MSVC REL32_%d probe with %s returned an incomplete formula receipt", variant.TrailingBytes, profile.Name)
			}
		}
		weakReport, err := pecoff.InspectAMD64WeakExternalLink(artifact.Objects[3].Bytes, artifact.Bytes, artifact.LinkMap, weakSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-external link with %s: %w", profile.Name, err)
		}
		if !weakReport.Fallback.UsedFallback || weakReport.Override.UsedFallback ||
			weakReport.Fallback.LinkedDisplacement != weakReport.Fallback.ExpectedDisplacement ||
			weakReport.Override.LinkedDisplacement != weakReport.Override.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC weak-external probe with %s returned an incomplete receipt", profile.Name)
		}
		overflowReport, err := pecoff.InspectAMD64RelocationOverflowLink(artifact.Objects[4].Bytes, artifact.Bytes, artifact.LinkMap, overflowSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC relocation-overflow link with %s: %w", profile.Name, err)
		}
		if overflowReport.Relocations != 0xffff || overflowReport.SourceRVA == 0 ||
			overflowReport.LinkedDisplacement != overflowReport.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC relocation-overflow probe with %s returned an incomplete receipt", profile.Name)
		}
		bigObjectReport, err := pecoff.InspectAMD64BigObjectLink(artifact.Objects[5].Bytes, artifact.Bytes, artifact.LinkMap, bigObjectSpec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC bigobj link with %s: %w", profile.Name, err)
		}
		if bigObjectReport.Version != 2 || bigObjectReport.SourceRVA == 0 ||
			bigObjectReport.LinkedDisplacement != bigObjectReport.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC bigobj probe with %s returned an incomplete receipt", profile.Name)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCRelocationProbe(artifact)
		if err != nil {
			return executions, fmt.Errorf("execute Windows MSVC REL32 probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != unit.ExpectedExit || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, fmt.Errorf("Windows MSVC REL32 probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, nil
}

func runWindowsWeakSearchProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	unit := windowsweaksearchprobe.Generate()
	weakObject, err := pecoff.BuildAMD64WeakSearchProbeObject(
		unit.NoLibraryEntry, unit.LibraryEntry, unit.NoLibraryWeak, unit.LibraryWeak,
		unit.NoLibraryFallback, unit.LibraryFallback,
	)
	if err != nil {
		return 0, fmt.Errorf("build AMD64 weak-search object: %w", err)
	}
	spec := pecoff.AMD64WeakSearchSpec{
		ObjectName: unit.ObjectName, ArchiveName: unit.ArchiveName,
		NoLibraryMember: unit.NoLibraryMember, LibraryMember: unit.LibraryMember,
		NoLibraryEntry: unit.NoLibraryEntry, LibraryEntry: unit.LibraryEntry,
		NoLibraryWeak: unit.NoLibraryWeak, NoLibrarySentinel: unit.NoLibrarySentinel, LibraryWeak: unit.LibraryWeak,
		NoLibraryFallback: unit.NoLibraryFallback, LibraryFallback: unit.LibraryFallback,
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCWeakSearchProbe(
			toolchain, profile, unit.CallerSource, unit.NoLibrarySource, unit.LibrarySource, weakObject,
		)
		if err != nil {
			return executions, fmt.Errorf("build Windows MSVC weak-search probe with %s: %w", profile.Name, err)
		}
		if artifact.ArchiveName != unit.ArchiveName || len(artifact.Objects) != 4 ||
			artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != unit.NoLibraryMember ||
			artifact.Objects[2].Name != unit.LibraryMember || artifact.Objects[3].Name != unit.ObjectName {
			return executions, fmt.Errorf("Windows MSVC weak-search probe with %s returned unexpected artifact receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-search PE with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-search unwind data with %s: %w", profile.Name, err)
		}
		report, err := pecoff.InspectAMD64WeakSearchLink(artifact.Objects[3].Bytes, artifact.Archive, artifact.Bytes, artifact.LinkMap, spec)
		if err != nil {
			return executions, fmt.Errorf("inspect Windows MSVC weak-search link with %s: %w", profile.Name, err)
		}
		if report.ArchiveMembers != 2 || !report.NoLibrary.UsedFallback || report.Library.UsedFallback ||
			report.NoLibrary.LinkedDisplacement != report.NoLibrary.ExpectedDisplacement ||
			report.Library.LinkedDisplacement != report.Library.ExpectedDisplacement {
			return executions, fmt.Errorf("Windows MSVC weak-search probe with %s returned an incomplete receipt", profile.Name)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCWeakSearchProbe(artifact)
		if err != nil {
			return executions, fmt.Errorf("execute Windows MSVC weak-search probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != unit.ExpectedExit || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, fmt.Errorf("Windows MSVC weak-search probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, nil
}

func hasCodeViewRelocationPair(report pecoff.CodeViewRelocationReport, symbol string) bool {
	for _, pair := range report.Pairs {
		if pair.Symbol == symbol && pair.ResolvedSection != 0 {
			return true
		}
	}
	return false
}

func runWindowsArchiveProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	unit := windowsarchiveprobe.Generate()
	spec := pecoff.ArchiveCOMDATLinkSpec{
		ArchiveName: unit.ArchiveName, SelectedMember: unit.SelectedMember,
		UnusedMember: unit.UnusedMember, DirectObject: unit.DirectObject,
		DemandSymbol: unit.DemandSymbol, UnusedSymbol: unit.UnusedSymbol,
		COMDATSymbol: unit.COMDATSymbol, ReceiptSection: unit.ReceiptSection,
		ReceiptMagic: unit.ReceiptMagic, ArchiveCandidate: unit.ArchiveCandidate,
		DirectCandidate: unit.DirectCandidate,
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifacts, err := cdriver.BuildWindowsMSVCArchiveProbe(
			toolchain, profile, unit.CallerSource, unit.ChosenSource, unit.UnusedSource, unit.DirectSource,
		)
		if err != nil {
			return executions, fmt.Errorf("build Windows MSVC archive probe with %s: %w", profile.Name, err)
		}
		if len(artifacts) != 2 || artifacts[0].Variant != "direct-first" || artifacts[1].Variant != "archive-first" {
			return executions, fmt.Errorf("Windows MSVC archive probe with %s returned unexpected input-order variants", profile.Name)
		}
		for _, artifact := range artifacts {
			if artifact.ArchiveName != unit.ArchiveName || len(artifact.Objects) != 4 ||
				artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != unit.SelectedMember ||
				artifact.Objects[2].Name != unit.UnusedMember || artifact.Objects[3].Name != unit.DirectObject {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned unexpected artifact receipts", profile.Name, artifact.Variant)
			}
			if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC archive probe PE with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC archive probe unwind data with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			report, err := pecoff.InspectArchiveCOMDATLink(
				artifact.Archive, artifact.Objects[3].Bytes, artifact.Bytes, artifact.LinkMap, spec,
			)
			if err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC archive and COMDAT selection with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if report.ArchiveMembers != 2 || report.ExtractedMember != unit.SelectedMember || report.DemandOwner != "archive" ||
				(report.COMDATOwner != "archive" && report.COMDATOwner != "direct") || report.TargetRVA == 0 || report.COMDATRVA == 0 {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned an incomplete selection receipt", profile.Name, artifact.Variant)
			}
			association, err := pecoff.InspectAssociativeCOMDATUnwindLink(
				artifact.Objects[1].Bytes, artifact.Bytes, artifact.LinkMap, pecoff.AssociativeCOMDATUnwindSpec{
					ArchiveName: unit.ArchiveName, SourceMember: unit.SelectedMember, SourceSymbol: unit.DemandSymbol,
				},
			)
			if err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC associative COMDAT unwind selection with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if association.ParentSectionIndex <= 0 || association.PDataSectionIndex <= 0 || association.XDataSectionIndex <= 0 ||
				association.FinalEndRVA <= association.FinalBeginRVA || association.FinalUnwindRVA == 0 {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned an incomplete associative COMDAT receipt", profile.Name, artifact.Variant)
			}
			discardedAssociation, err := pecoff.InspectDiscardedAssociativeCOMDATLink(
				artifact.Objects[1].Bytes, artifact.Objects[3].Bytes, artifact.Bytes, artifact.LinkMap,
				pecoff.DiscardedAssociativeCOMDATSpec{
					ArchiveName: unit.ArchiveName, ArchiveMember: unit.SelectedMember,
					DirectObject: unit.DirectObject, SourceSymbol: unit.DiscardSymbol,
				},
			)
			if err != nil {
				return executions, fmt.Errorf("inspect Windows MSVC discarded associative COMDAT with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if discardedAssociation.SelectedOwner == discardedAssociation.DiscardedOwner ||
				discardedAssociation.SelectedBeginRVA == 0 || discardedAssociation.SelectedEndRVA <= discardedAssociation.SelectedBeginRVA ||
				discardedAssociation.SelectedUnwindRVA == 0 || discardedAssociation.DiscardedPayloadMatches != 0 || discardedAssociation.DiscardedRawMatches != 0 || discardedAssociation.OrphanPDataBytes != 0 ||
				discardedAssociation.SelectedUnwindDigest == discardedAssociation.DiscardedUnwindDigest {
				return executions, fmt.Errorf("Windows MSVC archive probe %s/%s returned an incomplete discarded associative COMDAT receipt", profile.Name, artifact.Variant)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCArchiveProbe(artifact)
			if err != nil {
				return executions, fmt.Errorf("execute Windows MSVC archive probe with %s/%s: %w", profile.Name, artifact.Variant, err)
			}
			if outcome.ExitCode != unit.ExpectedExit || outcome.Stdout != "" || outcome.Stderr != "" {
				return executions, fmt.Errorf("Windows MSVC archive probe with %s/%s returned %d, stdout %q, and stderr %q", profile.Name, artifact.Variant, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
			}
			executions++
		}
	}
	return executions, nil
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

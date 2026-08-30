package main

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
const checkedInContractCapabilitiesManifestPath = "examples/contract_capabilities_quickstart/fairy.toml"
const checkedInContractCapabilitiesMainPath = "examples/contract_capabilities_quickstart/src/main.fy"
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
const checkedInNativeTestArgsManifestPath = "examples/native_test_args_quickstart/fairy.toml"
const checkedInNativeTestArgsMainPath = "examples/native_test_args_quickstart/src/main.fy"
const checkedInNativeTestArgsTestPath = "examples/native_test_args_quickstart/tests/process_args.fy"
const checkedInNativeTestStdinManifestPath = "examples/native_test_stdin_quickstart/fairy.toml"
const checkedInNativeTestStdinMainPath = "examples/native_test_stdin_quickstart/src/main.fy"
const checkedInNativeTestStdinTestPath = "examples/native_test_stdin_quickstart/tests/stdin.fy"
const checkedInNativeTestStdinFixturePath = "examples/native_test_stdin_quickstart/tests/fixtures/input.txt"
const checkedInNativeTestOutputManifestPath = "examples/native_test_output_quickstart/fairy.toml"
const checkedInNativeTestOutputMainPath = "examples/native_test_output_quickstart/src/main.fy"
const checkedInNativeTestOutputTestPath = "examples/native_test_output_quickstart/tests/output.fy"
const checkedInNativeTestTimeoutManifestPath = "examples/native_test_timeout_quickstart/fairy.toml"
const checkedInNativeTestTimeoutMainPath = "examples/native_test_timeout_quickstart/src/main.fy"
const checkedInNativeTestTimeoutTestPath = "examples/native_test_timeout_quickstart/tests/timeout.fy"
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

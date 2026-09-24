package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPrivateVectorSDKCandidateCBOMWorkLimit(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	core, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	const probe = `
int main(void) {
    static const unsigned char input[] = {0xef, 0xbb, 0xbf, 'a'};
    struct sample { size_t length, work; unsigned phase; };
    static const struct sample cases[] = {
        {3, 0, 0}, {3, 2, 0}, {3, 3, 1},
        {4, 2, 0}, {4, 3, 2}, {4, 4, 3},
    };
    for (size_t i = 0; i < sizeof(cases) / sizeof(cases[0]); i++) {
        const struct sample *item = &cases[i];
        struct fc_token tokens[4] = {0};
        struct fc_diagnostic diagnostics[2] = {0};
        struct fc_scanner scanner = {0};
        scanner.source = input;
        scanner.length = item->length;
        scanner.tokens = tokens;
        scanner.diagnostics = diagnostics;
        scanner.work_limit = item->work;
        int status = fc_scan(&scanner);
        int bad = status != 0 || scanner.token_count != (item->phase >= 2u ? 3u : 2u) ||
                  scanner.diagnostic_count != (item->phase == 0u || item->phase == 2u ? 1u : 0u) ||
                  tokens[0].kind != (item->phase == 0u ? (unsigned)FC_INVALID : (unsigned)FC_TRIVIA) ||
                  tokens[0].trivia != (item->phase == 0u ? (unsigned)FC_NO_TRIVIA : (unsigned)FC_PREAMBLE) ||
                  tokens[0].start != 0 || tokens[0].end != (item->phase == 0u ? item->length : 3u);
        if (!bad && item->phase >= 2u)
            bad = tokens[1].kind != (item->phase == 2u ? (unsigned)FC_INVALID : (unsigned)FC_IDENTIFIER) ||
                  tokens[1].start != 3u || tokens[1].end != item->length;
        if (!bad && (item->phase == 0u || item->phase == 2u)) {
            size_t position = item->phase == 0u ? 0u : 3u;
            bad = strcmp(diagnostics[0].code, "E0010") != 0 ||
                  diagnostics[0].start != position || diagnostics[0].end != position;
        }
        if (!bad) {
            const struct fc_token *eof = &tokens[scanner.token_count - 1u];
            bad = eof->kind != FC_EOF || eof->start != item->length || eof->end != item->length;
        }
        if (bad) return (int)(i + 1u);
    }
    return 0;
}
`
	code := "#define FC_CANDIDATE_WORK_LIMIT\n#define FC_SCANNER_LIBRARY\n" + string(core) + probe
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" {
		t.Fatal("real scanner source changed")
	}
	program, entry, _ := scannerBudgetFairyProgramWithMain(t, variants[0], candidateCBOMWorkMain)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			cImage, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			cOutcome, err := cdriver.ExecuteWindowsHost(cImage)
			if err != nil || cOutcome.ExitCode != 0 || cOutcome.Stdout != "" || cOutcome.Stderr != "" || cOutcome.TrapControl != "" {
				t.Fatalf("C BOM work limit: %+v %v", cOutcome, err)
			}
			fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
			fairyOutcome, err := cdriver.ExecuteWindowsHost(fairyImage)
			if err != nil || fairyOutcome.ExitCode != 0 || fairyOutcome.Stdout != "" || fairyOutcome.Stderr != "" || fairyOutcome.TrapControl != "" {
				t.Fatalf("Fairy BOM work limit: %+v %v", fairyOutcome, err)
			}
		})
	}
}

const candidateCBOMWorkMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import diagnostics as diagnostics;

fn i32 check(ref u8[] input, usize work_limit, u32 phase) {
    let bytes::Buffer owned = switch (bytes::copy(input)) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    let scanner::Stream stream = switch (scanner::collect(owned, false, 32768, 1048576, 256, work_limit)) {
        case .ok(value): { value }
        case .err(_): { return 2; }
    };
    let usize count = if (phase >= 2) { 3 } else { 2 };
    if (scanner::token_count(ref stream) != count ||
        scanner::diagnostic_count(ref stream) != (if (phase == 0 || phase == 2) { 1 } else { 0 })) { return 3; }
    let scanner::TokenInfo first = scanner::token_info(ref stream, 0);
    if (first.kind != (if (phase == 0) { 0 } else { 2 }) ||
        first.trivia != (if (phase == 0) { 0 } else { 1 }) || first.start != 0 ||
        first.end != (if (phase == 0) { input.len } else { 3 })) { return 4; }
    if (phase >= 2) {
        let scanner::TokenInfo next = scanner::token_info(ref stream, 1);
        if (next.kind != (if (phase == 2) { 0 } else { 3 }) ||
            next.start != 3 || next.end != input.len) { return 5; }
    }
    if (phase == 0 || phase == 2) {
        let diagnostics::Info d = scanner::diagnostic_info(ref stream, 0);
        let usize position = if (phase == 0) { 0 } else { 3 };
        if (d.code != 10 || d.start != position || d.end != position) { return 6; }
    }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, count - 1);
    if (eof.kind != 1 || eof.start != input.len || eof.end != input.len) { return 7; }
    return 0;
}

fn i32 main() {
    if (check(b"\xEF\xBB\xBF"[..], 0, 0) != 0) { return 11; }
    if (check(b"\xEF\xBB\xBF"[..], 2, 0) != 0) { return 12; }
    if (check(b"\xEF\xBB\xBF"[..], 3, 1) != 0) { return 13; }
    if (check(b"\xEF\xBB\xBFa"[..], 2, 0) != 0) { return 14; }
    if (check(b"\xEF\xBB\xBFa"[..], 3, 2) != 0) { return 15; }
    if (check(b"\xEF\xBB\xBFa"[..], 4, 3) != 0) { return 16; }
    return 0;
}`

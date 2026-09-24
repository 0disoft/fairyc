package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPrivateVectorSDKCandidateCNumberWorkLimit(t *testing.T) {
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
    struct sample { const char *text; size_t work, boundary; unsigned kind; const char *code; };
    static const struct sample cases[] = {
        {"1", 0, 0, FC_INTEGER, "E0010"},
        {"1", 1, 1, FC_INTEGER, NULL},
        {"12", 1, 1, FC_INTEGER, "E0010"},
        {"12", 2, 2, FC_INTEGER, NULL},
        {"0x1", 1, 1, FC_INTEGER, "E0010"},
        {"0x1", 2, 2, FC_INTEGER, "E0010"},
        {"0x1", 3, 3, FC_INTEGER, NULL},
        {"0b12", 3, 3, FC_INTEGER, "E0010"},
        {"0b12", 4, 4, FC_INTEGER, "E0004"},
        {"1abc", 3, 3, FC_INTEGER, "E0010"},
        {"1abc", 4, 4, FC_INTEGER, "E0004"},
        {"1.5", 2, 1, FC_FLOAT, "E0010"},
        {"1.5", 3, 3, FC_FLOAT, "E0010"},
        {"1.5", 6, 3, FC_FLOAT, NULL},
        {"1e", 1, 1, FC_FLOAT, "E0010"},
        {"1e", 2, 2, FC_FLOAT, "E0004"},
        {"1e3", 6, 3, FC_FLOAT, "E0010"},
        {"1e3", 7, 3, FC_FLOAT, NULL},
        {"1e+3", 7, 4, FC_FLOAT, "E0010"},
        {"1e+3", 8, 4, FC_FLOAT, NULL},
    };
    for (size_t i = 0; i < sizeof(cases) / sizeof(cases[0]); i++) {
        const struct sample *item = &cases[i];
        size_t length = strlen(item->text);
        int exhausted = item->code != NULL && strcmp(item->code, "E0010") == 0;
        struct fc_token tokens[4] = {0};
        struct fc_diagnostic diagnostics[2] = {0};
        struct fc_scanner scanner = {0};
        scanner.source = (const unsigned char *)item->text;
        scanner.length = length;
        scanner.tokens = tokens;
        scanner.diagnostics = diagnostics;
        scanner.work_limit = item->work;
        int status = fc_scan(&scanner);
        int bad = status != 0 || scanner.token_count != (exhausted && item->boundary > 0 && item->boundary < length ? 3u : 2u) ||
                  scanner.diagnostic_count != (item->code == NULL ? 0u : 1u) ||
                  tokens[0].kind != (exhausted ? (unsigned)FC_INVALID : item->kind) ||
                  tokens[0].start != 0 ||
                  tokens[0].end != (exhausted && item->boundary > 0 ? item->boundary : length);
        if (!bad && item->code != NULL) {
            const struct fc_diagnostic *d = &diagnostics[0];
            bad = strcmp(d->code, item->code) != 0 ||
                  d->start != (exhausted ? item->boundary : 0u) ||
                  d->end != (exhausted ? item->boundary : length);
        }
        if (!bad && exhausted && item->boundary > 0 && item->boundary < length)
            bad = tokens[1].kind != FC_INVALID || tokens[1].start != item->boundary ||
                  tokens[1].end != length;
        if (!bad) {
            const struct fc_token *eof = &tokens[scanner.token_count - 1u];
            bad = eof->kind != FC_EOF || eof->start != length || eof->end != length;
        }
        fc_release_cooked(&scanner);
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
	program, entry, _ := scannerBudgetFairyProgramWithMain(t, variants[0], candidateCNumberWorkMain)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			cImage, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			cOutcome, err := cdriver.ExecuteWindowsHost(cImage)
			if err != nil || cOutcome.ExitCode != 0 || cOutcome.Stdout != "" || cOutcome.Stderr != "" || cOutcome.TrapControl != "" {
				t.Fatalf("C number work limit: %+v %v", cOutcome, err)
			}
			fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
			fairyOutcome, err := cdriver.ExecuteWindowsHost(fairyImage)
			if err != nil || fairyOutcome.ExitCode != 0 || fairyOutcome.Stdout != "" || fairyOutcome.Stderr != "" || fairyOutcome.TrapControl != "" {
				t.Fatalf("Fairy number work limit: %+v %v", fairyOutcome, err)
			}
		})
	}
}

const candidateCNumberWorkMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import diagnostics as diagnostics;

fn i32 check(ref u8[] input, usize work_limit, u32 kind, usize boundary, u32 code) {
    let bytes::Buffer owned = switch (bytes::copy(input)) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    let scanner::Stream stream = switch (scanner::collect(owned, false, 32768, 1048576, 256, work_limit)) {
        case .ok(value): { value }
        case .err(_): { return 2; }
    };
    let bool exhausted = code == 10;
    let usize count = if (exhausted && boundary > 0 && boundary < input.len) { 3 } else { 2 };
    if (scanner::token_count(ref stream) != count) { return 3; }
    let scanner::TokenInfo first = scanner::token_info(ref stream, 0);
    if (first.kind != (if (exhausted) { 0 } else { kind }) || first.start != 0 ||
        first.end != (if (exhausted && boundary > 0) { boundary } else { input.len })) { return 4; }
    if (scanner::diagnostic_count(ref stream) != (if (code == 0) { 0 } else { 1 })) { return 5; }
    if (code != 0) {
        let diagnostics::Info d = scanner::diagnostic_info(ref stream, 0);
        if (d.code != code || d.start != (if (exhausted) { boundary } else { 0 }) ||
            d.end != (if (exhausted) { boundary } else { input.len })) { return 6; }
    }
    if (exhausted && boundary > 0 && boundary < input.len) {
        let scanner::TokenInfo tail = scanner::token_info(ref stream, 1);
        if (tail.kind != 0 || tail.start != boundary || tail.end != input.len) { return 7; }
    }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, count - 1);
    if (eof.kind != 1 || eof.start != input.len || eof.end != input.len) { return 8; }
    return 0;
}

fn i32 main() {
    if (check(b"1"[..], 0, 5, 0, 10) != 0) { return 11; }
    if (check(b"1"[..], 1, 5, 1, 0) != 0) { return 12; }
    if (check(b"12"[..], 1, 5, 1, 10) != 0) { return 13; }
    if (check(b"12"[..], 2, 5, 2, 0) != 0) { return 14; }
    if (check(b"0x1"[..], 1, 5, 1, 10) != 0) { return 15; }
    if (check(b"0x1"[..], 2, 5, 2, 10) != 0) { return 16; }
    if (check(b"0x1"[..], 3, 5, 3, 0) != 0) { return 17; }
    if (check(b"0b12"[..], 3, 5, 3, 10) != 0) { return 18; }
    if (check(b"0b12"[..], 4, 5, 4, 4) != 0) { return 19; }
    if (check(b"1abc"[..], 3, 5, 3, 10) != 0) { return 20; }
    if (check(b"1abc"[..], 4, 5, 4, 4) != 0) { return 21; }
    if (check(b"1.5"[..], 2, 6, 1, 10) != 0) { return 22; }
    if (check(b"1.5"[..], 3, 6, 3, 10) != 0) { return 23; }
    if (check(b"1.5"[..], 6, 6, 3, 0) != 0) { return 24; }
    if (check(b"1e"[..], 1, 6, 1, 10) != 0) { return 25; }
    if (check(b"1e"[..], 2, 6, 2, 4) != 0) { return 26; }
    if (check(b"1e3"[..], 6, 6, 3, 10) != 0) { return 27; }
    if (check(b"1e3"[..], 7, 6, 3, 0) != 0) { return 28; }
    if (check(b"1e+3"[..], 7, 6, 4, 10) != 0) { return 29; }
    if (check(b"1e+3"[..], 8, 6, 4, 0) != 0) { return 30; }
    return 0;
}`

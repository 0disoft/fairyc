package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPrivateVectorSDKCandidateCTriviaWorkLimit(t *testing.T) {
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
    struct sample { const char *text; size_t work, boundary; unsigned trivia; };
    static const struct sample cases[] = {
        {" ", 0, 0, FC_WHITESPACE},
        {" \t", 1, 1, FC_WHITESPACE},
        {" \t", 2, 2, FC_WHITESPACE},
        {"\r\n", 1, 1, FC_WHITESPACE},
        {"\r\n", 2, 2, FC_WHITESPACE},
        {"//x", 1, 1, FC_LINE_COMMENT},
        {"//x", 2, 2, FC_LINE_COMMENT},
        {"//x", 3, 3, FC_LINE_COMMENT},
        {"///", 2, 2, FC_DOC_LINE},
        {"///", 3, 3, FC_DOC_LINE},
        {"/*x*/", 1, 1, FC_BLOCK_COMMENT},
        {"/*x*/", 3, 3, FC_BLOCK_COMMENT},
        {"/*x*/", 4, 3, FC_BLOCK_COMMENT},
        {"/*x*/", 5, 5, FC_BLOCK_COMMENT},
        {"/*a/*b*/c*/", 4, 3, FC_BLOCK_COMMENT},
        {"/*a/*b*/c*/", 7, 6, FC_BLOCK_COMMENT},
        {"/*a/*b*/c*/", 11, 11, FC_BLOCK_COMMENT},
    };
    for (size_t i = 0; i < sizeof(cases) / sizeof(cases[0]); i++) {
        const struct sample *item = &cases[i];
        size_t length = strlen(item->text);
        int success = item->boundary == length;
        struct fc_token tokens[4] = {0};
        struct fc_diagnostic diagnostics[2] = {0};
        struct fc_scanner scanner = {0};
        scanner.source = (const unsigned char *)item->text;
        scanner.length = length;
        scanner.tokens = tokens;
        scanner.diagnostics = diagnostics;
        scanner.work_limit = item->work;
        int status = fc_scan(&scanner);
        int bad = status != 0 || scanner.token_count != (success || item->boundary == 0 ? 2u : 3u) ||
                  scanner.diagnostic_count != (success ? 0u : 1u) ||
                  tokens[0].kind != (success ? (unsigned)FC_TRIVIA : (unsigned)FC_INVALID) ||
                  tokens[0].trivia != (success ? item->trivia : (unsigned)FC_NO_TRIVIA) ||
                  tokens[0].start != 0 ||
                  tokens[0].end != (success || item->boundary == 0 ? length : item->boundary);
        if (!bad && !success) {
            const struct fc_diagnostic *d = &diagnostics[0];
            bad = strcmp(d->code, "E0010") != 0 || d->start != item->boundary ||
                  d->end != item->boundary ||
                  (item->boundary != 0 &&
                   (tokens[1].kind != FC_INVALID || tokens[1].start != item->boundary ||
                    tokens[1].end != length));
        }
        if (!bad) {
            const struct fc_token *eof = &tokens[scanner.token_count - 1u];
            bad = eof->kind != FC_EOF || eof->start != length || eof->end != length;
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
	program, entry, _ := scannerBudgetFairyProgramWithMain(t, variants[0], candidateCTriviaWorkMain)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			cImage, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			cOutcome, err := cdriver.ExecuteWindowsHost(cImage)
			if err != nil || cOutcome.ExitCode != 0 || cOutcome.Stdout != "" || cOutcome.Stderr != "" || cOutcome.TrapControl != "" {
				t.Fatalf("C trivia work limit: %+v %v", cOutcome, err)
			}
			fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
			fairyOutcome, err := cdriver.ExecuteWindowsHost(fairyImage)
			if err != nil || fairyOutcome.ExitCode != 0 || fairyOutcome.Stdout != "" || fairyOutcome.Stderr != "" || fairyOutcome.TrapControl != "" {
				t.Fatalf("Fairy trivia work limit: %+v %v", fairyOutcome, err)
			}
		})
	}
}

const candidateCTriviaWorkMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import diagnostics as diagnostics;

fn i32 check(ref u8[] input, usize work_limit, u32 trivia, usize boundary) {
    let bytes::Buffer owned = switch (bytes::copy(input)) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    let scanner::Stream stream = switch (scanner::collect(owned, false, 32768, 1048576, 256, work_limit)) {
        case .ok(value): { value }
        case .err(_): { return 2; }
    };
    let bool success = boundary == input.len;
    let usize count = if (success || boundary == 0) { 2 } else { 3 };
    if (scanner::token_count(ref stream) != count) { return 3; }
    let scanner::TokenInfo first = scanner::token_info(ref stream, 0);
    if (first.kind != (if (success) { 2 } else { 0 }) ||
        first.trivia != (if (success) { trivia } else { 0 }) || first.start != 0 ||
        first.end != (if (success || boundary == 0) { input.len } else { boundary })) { return 4; }
    if (scanner::diagnostic_count(ref stream) != (if (success) { 0 } else { 1 })) { return 5; }
    if (!success) {
        let diagnostics::Info d = scanner::diagnostic_info(ref stream, 0);
        if (d.code != 10 || d.start != boundary || d.end != boundary) { return 6; }
        if (boundary != 0) {
            let scanner::TokenInfo tail = scanner::token_info(ref stream, 1);
            if (tail.kind != 0 || tail.start != boundary || tail.end != input.len) { return 7; }
        }
    }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, count - 1);
    if (eof.kind != 1 || eof.start != input.len || eof.end != input.len) { return 8; }
    return 0;
}

fn i32 main() {
    if (check(b" "[..], 0, 2, 0) != 0) { return 11; }
    if (check(b" \t"[..], 1, 2, 1) != 0) { return 12; }
    if (check(b" \t"[..], 2, 2, 2) != 0) { return 13; }
    if (check(b"\r\n"[..], 1, 2, 1) != 0) { return 14; }
    if (check(b"\r\n"[..], 2, 2, 2) != 0) { return 15; }
    if (check(b"//x"[..], 1, 3, 1) != 0) { return 16; }
    if (check(b"//x"[..], 2, 3, 2) != 0) { return 17; }
    if (check(b"//x"[..], 3, 3, 3) != 0) { return 18; }
    if (check(b"///"[..], 2, 5, 2) != 0) { return 19; }
    if (check(b"///"[..], 3, 5, 3) != 0) { return 20; }
    if (check(b"/*x*/"[..], 1, 4, 1) != 0) { return 21; }
    if (check(b"/*x*/"[..], 3, 4, 3) != 0) { return 22; }
    if (check(b"/*x*/"[..], 4, 4, 3) != 0) { return 23; }
    if (check(b"/*x*/"[..], 5, 4, 5) != 0) { return 24; }
    if (check(b"/*a/*b*/c*/"[..], 4, 4, 3) != 0) { return 25; }
    if (check(b"/*a/*b*/c*/"[..], 7, 4, 6) != 0) { return 26; }
    if (check(b"/*a/*b*/c*/"[..], 11, 4, 11) != 0) { return 27; }
    return 0;
}`

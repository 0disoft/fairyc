package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// The candidate work budget charges source bytes before a string consumes them.
// Check the exact failing byte as well as the completed cooked payload.
func TestPrivateVectorSDKCandidateCStringWorkLimit(t *testing.T) {
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
    struct sample { const char *text; size_t work, boundary; unsigned kind; const char *hex; };
    static const struct sample cases[] = {
        {"\"A\"", 0, 0, FC_STRING, ""},
        {"\"A\"", 2, 2, FC_STRING, ""},
        {"\"A\"", 3, 3, FC_STRING, "41"},
        {"c\"\"", 2, 2, FC_C_STRING, ""},
        {"c\"\"", 3, 3, FC_C_STRING, "00"},
        {"\"\\n\"", 2, 2, FC_STRING, ""},
        {"\"\\n\"", 4, 4, FC_STRING, "0a"},
        {"\"\\x41\"", 4, 3, FC_STRING, ""},
        {"\"\\x41\"", 6, 6, FC_STRING, "41"},
        {"\"\\u{41}\"", 7, 7, FC_STRING, ""},
        {"\"\\u{41}\"", 8, 8, FC_STRING, "41"},
        {"\"\xed\x95\x9c\"", 3, 1, FC_STRING, ""},
        {"\"\xed\x95\x9c\"", 5, 5, FC_STRING, "ed959c"},
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
                  tokens[0].kind != (success ? item->kind : FC_INVALID) ||
                  tokens[0].start != 0 ||
                  tokens[0].end != (success || item->boundary == 0 ? length : item->boundary);
        if (!bad && success)
            bad = tokens[0].cooked_length != strlen(item->hex) ||
                  memcmp(tokens[0].cooked, item->hex, strlen(item->hex)) != 0;
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
	program, entry, _ := scannerBudgetFairyProgramWithMain(t, variants[0], candidateCStringWorkMain)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			cImage, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			cOutcome, err := cdriver.ExecuteWindowsHost(cImage)
			if err != nil || cOutcome.ExitCode != 0 || cOutcome.Stdout != "" || cOutcome.Stderr != "" || cOutcome.TrapControl != "" {
				t.Fatalf("C string work limit: %+v %v", cOutcome, err)
			}
			fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
			fairyOutcome, err := cdriver.ExecuteWindowsHost(fairyImage)
			if err != nil || fairyOutcome.ExitCode != 0 || fairyOutcome.Stdout != "" || fairyOutcome.Stderr != "" || fairyOutcome.TrapControl != "" {
				t.Fatalf("Fairy string work limit: %+v %v", fairyOutcome, err)
			}
		})
	}
}

const candidateCStringWorkMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import diagnostics as diagnostics;

fn i32 check(ref u8[] input, usize work_limit, u32 kind, usize boundary, usize cooked_length) {
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
    if (first.kind != (if (success) { kind } else { 0 }) || first.start != 0 ||
        first.end != (if (success || boundary == 0) { input.len } else { boundary })) { return 4; }
    if (scanner::diagnostic_count(ref stream) != (if (success) { 0 } else { 1 })) { return 5; }
    if (success) {
        if (scanner::token_cooked(ref stream, 0).len != cooked_length) { return 6; }
    } else {
        let diagnostics::Info d = scanner::diagnostic_info(ref stream, 0);
        if (d.code != 10 || d.start != boundary || d.end != boundary) { return 7; }
        if (boundary != 0) {
            let scanner::TokenInfo tail = scanner::token_info(ref stream, 1);
            if (tail.kind != 0 || tail.start != boundary || tail.end != input.len) { return 8; }
        }
    }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, count - 1);
    if (eof.kind != 1 || eof.start != input.len || eof.end != input.len) { return 9; }
    return 0;
}

fn i32 main() {
    if (check(b"\"A\""[..], 0, 7, 0, 0) != 0) { return 11; }
    if (check(b"\"A\""[..], 2, 7, 2, 0) != 0) { return 12; }
    if (check(b"\"A\""[..], 3, 7, 3, 2) != 0) { return 13; }
    if (check(b"c\"\""[..], 2, 9, 2, 0) != 0) { return 14; }
    if (check(b"c\"\""[..], 3, 9, 3, 2) != 0) { return 15; }
    if (check(b"\"\\n\""[..], 2, 7, 2, 0) != 0) { return 16; }
    if (check(b"\"\\n\""[..], 4, 7, 4, 2) != 0) { return 17; }
    if (check(b"\"\\x41\""[..], 4, 7, 3, 0) != 0) { return 18; }
    if (check(b"\"\\x41\""[..], 6, 7, 6, 2) != 0) { return 19; }
    if (check(b"\"\\u{41}\""[..], 7, 7, 7, 0) != 0) { return 20; }
    if (check(b"\"\\u{41}\""[..], 8, 7, 8, 2) != 0) { return 21; }
    if (check(b"\"\xED\x95\x9C\""[..], 3, 7, 1, 0) != 0) { return 22; }
    if (check(b"\"\xED\x95\x9C\""[..], 5, 7, 5, 6) != 0) { return 23; }
    return 0;
}`

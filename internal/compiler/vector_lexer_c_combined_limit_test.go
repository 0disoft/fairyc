package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// The first exhausted limit wins; its terminal diagnostic must not be replaced
// by a token or ordinary-diagnostic limit reached on the same input.
func TestPrivateVectorSDKCandidateCCombinedLimits(t *testing.T) {
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
    unsigned char invalid[257];
    memset(invalid, 0xff, sizeof(invalid));
    static const unsigned char bom[] = {0xef, 0xbb, 0xbf};
    static const unsigned char bom_identifier[] = {0xef, 0xbb, 0xbf, 'a'};
    struct sample {
        const unsigned char *input;
        size_t length, work, token_limit, tokens, diagnostics, terminal;
        unsigned first_kind;
        size_t first_end, last_start, byte_limit;
    };
    const struct sample cases[] = {
        {(const unsigned char *)"@x", 2, 0, 0, 2, 1, 0, FC_INVALID, 2, 0},
        {(const unsigned char *)"@x", 2, 1, 0, 3, 1, 1, FC_INVALID, 1, 1},
        {(const unsigned char *)"a@", 2, 1, 1, 3, 1, 1, FC_IDENTIFIER, 1, 1},
        {(const unsigned char *)"a@", 2, 2, 1, 3, 1, 2, FC_IDENTIFIER, 1, 1},
        {(const unsigned char *)"a@", 2, 2, 0, 3, 1, 1, FC_INVALID, 1, 1},
        {invalid, 257, 256, 256, 258, 257, 256, FC_INVALID, 1, 256},
        {invalid, 257, 257, 256, 258, 257, 257, FC_INVALID, 1, 256},
        {invalid, 257, 257, 257, 258, 257, 257, FC_INVALID, 1, 256},
        {(const unsigned char *)"a@", 2, 1, 0, 2, 1, 0, FC_INVALID, 2, 0, 1},
        {(const unsigned char *)"a@", 2, 1, 0, 3, 1, 1, FC_INVALID, 1, 1, 2},
        {bom, 3, 2, 1, 2, 1, 0, FC_INVALID, 3, 0, 3},
        {bom, 3, 3, 0, 2, 1, 3, FC_INVALID, 3, 0, 3},
        {bom_identifier, 4, 3, 1, 3, 1, 3, FC_TRIVIA, 3, 3, 4},
        {bom_identifier, 4, 4, 1, 3, 1, 4, FC_TRIVIA, 3, 3, 4},
        {bom, 3, 0, 0, 2, 1, 0, FC_INVALID, 3, 0, 2},
    };
    for (size_t i = 0; i < sizeof(cases) / sizeof(cases[0]); i++) {
        const struct sample *item = &cases[i];
        struct fc_token tokens[260] = {0};
        struct fc_diagnostic diagnostics[259] = {0};
        struct fc_scanner scanner = {0};
        scanner.source = item->input;
        scanner.length = item->length;
        scanner.tokens = tokens;
        scanner.diagnostics = diagnostics;
        scanner.work_limit = item->work;
        scanner.token_limit = item->token_limit;
        scanner.byte_limit_set = item->byte_limit != 0u;
        scanner.byte_limit = item->byte_limit;
        int status = fc_scan(&scanner);
        int bad = status != 0 || scanner.token_count != item->tokens ||
                  scanner.diagnostic_count != item->diagnostics ||
                  tokens[0].kind != item->first_kind || tokens[0].start != 0 ||
                  tokens[0].end != item->first_end;
        if (!bad) {
            const struct fc_token *last = &tokens[scanner.token_count - 2u];
            const struct fc_token *eof = &tokens[scanner.token_count - 1u];
            const struct fc_diagnostic *terminal = &diagnostics[scanner.diagnostic_count - 1u];
            bad = last->kind != FC_INVALID || last->start != item->last_start ||
                  last->end != item->length || eof->kind != FC_EOF ||
                  eof->start != item->length || eof->end != item->length ||
                  strcmp(terminal->code, "E0010") != 0 ||
                  terminal->start != item->terminal || terminal->end != item->terminal;
        }
        if (!bad && item->diagnostics > 1u)
            bad = strcmp(diagnostics[0].code, "E0001") != 0 ||
                  diagnostics[0].start != 0 || diagnostics[0].end != 1u;
        if (bad) return (int)(i + 1u);
    }
    return 0;
}
`
	code := "#define FC_CANDIDATE_WORK_LIMIT\n#define FC_CANDIDATE_TOKEN_LIMIT\n#define FC_CANDIDATE_DIAGNOSTIC_LIMIT\n#define FC_CANDIDATE_BYTE_LIMIT\n#define FC_SCANNER_LIBRARY\n" + string(core) + probe
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner source changed")
	}
	invalid := strings.Repeat(`\xFF`, 257)
	fairyMain := fmt.Sprintf(candidateCCombinedLimitMain, invalid, invalid, invalid)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			cImage, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			cOutcome, err := cdriver.ExecuteWindowsHost(cImage)
			if err != nil || cOutcome.ExitCode != 0 || cOutcome.Stdout != "" || cOutcome.Stderr != "" || cOutcome.TrapControl != "" {
				t.Fatalf("C combined limits: %+v %v", cOutcome, err)
			}
			for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
				t.Run(variant.name, func(t *testing.T) {
					program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, fairyMain)
					fairyImage, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
					fairyOutcome, runErr := cdriver.ExecuteWindowsHost(fairyImage)
					if runErr != nil || fairyOutcome.ExitCode != 0 || fairyOutcome.Stdout != "" || fairyOutcome.Stderr != "" || fairyOutcome.TrapControl != "" {
						t.Fatalf("Fairy combined limits: %+v %v", fairyOutcome, runErr)
					}
				})
			}
		})
	}
}

const candidateCCombinedLimitMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import diagnostics as diagnostics;

fn i32 check(ref u8[] input, usize byte_limit, usize work_limit, usize token_limit, usize wanted_tokens,
             usize wanted_diagnostics, usize terminal, u32 first_kind, usize first_end,
             usize last_start) {
    let bytes::Buffer owned = switch (bytes::copy(input)) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    let scanner::Stream stream = switch (scanner::collect(owned, false, byte_limit, token_limit, 256, work_limit)) {
        case .ok(value): { value }
        case .err(_): { return 2; }
    };
    if (scanner::token_count(ref stream) != wanted_tokens ||
        scanner::diagnostic_count(ref stream) != wanted_diagnostics) { return 3; }
    let scanner::TokenInfo first = scanner::token_info(ref stream, 0);
    if (first.kind != first_kind || first.start != 0 || first.end != first_end) { return 4; }
    let scanner::TokenInfo last = scanner::token_info(ref stream, wanted_tokens - 2);
    if (last.kind != 0 || last.start != last_start || last.end != input.len) { return 5; }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, wanted_tokens - 1);
    if (eof.kind != 1 || eof.start != input.len || eof.end != input.len) { return 6; }
    let diagnostics::Info d = scanner::diagnostic_info(ref stream, wanted_diagnostics - 1);
    if (d.code != 10 || d.start != terminal || d.end != terminal) { return 7; }
    if (wanted_diagnostics > 1) {
        let diagnostics::Info first_d = scanner::diagnostic_info(ref stream, 0);
        if (first_d.code != 1 || first_d.start != 0 || first_d.end != 1) { return 8; }
    }
    return 0;
}

fn i32 main() {
    if (check(b"@x"[..], 32768, 0, 0, 2, 1, 0, 0, 2, 0) != 0) { return 11; }
    if (check(b"@x"[..], 32768, 1, 0, 3, 1, 1, 0, 1, 1) != 0) { return 12; }
    if (check(b"a@"[..], 32768, 1, 1, 3, 1, 1, 3, 1, 1) != 0) { return 13; }
    if (check(b"a@"[..], 32768, 2, 1, 3, 1, 2, 3, 1, 1) != 0) { return 14; }
    if (check(b"a@"[..], 32768, 2, 0, 3, 1, 1, 0, 1, 1) != 0) { return 25; }
    if (check(b"%s"[..], 32768, 256, 256, 258, 257, 256, 0, 1, 256) != 0) { return 15; }
    if (check(b"%s"[..], 32768, 257, 256, 258, 257, 257, 0, 1, 256) != 0) { return 16; }
    if (check(b"%s"[..], 32768, 257, 257, 258, 257, 257, 0, 1, 256) != 0) { return 17; }
    if (check(b"a@"[..], 1, 1, 0, 2, 1, 0, 0, 2, 0) != 0) { return 18; }
    if (check(b"a@"[..], 2, 1, 0, 3, 1, 1, 0, 1, 1) != 0) { return 19; }
    if (check(b"\xEF\xBB\xBF"[..], 3, 2, 1, 2, 1, 0, 0, 3, 0) != 0) { return 20; }
    if (check(b"\xEF\xBB\xBF"[..], 3, 3, 0, 2, 1, 3, 0, 3, 0) != 0) { return 21; }
    if (check(b"\xEF\xBB\xBFa"[..], 4, 3, 1, 3, 1, 3, 2, 3, 3) != 0) { return 22; }
    if (check(b"\xEF\xBB\xBFa"[..], 4, 4, 1, 3, 1, 4, 2, 3, 3) != 0) { return 23; }
    if (check(b"\xEF\xBB\xBF"[..], 2, 0, 0, 2, 1, 0, 0, 3, 0) != 0) { return 24; }
    return 0;
}`

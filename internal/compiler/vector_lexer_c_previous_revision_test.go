//go:build windows

package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

func TestPrivateVectorSDKCandidateCPreviousRevisionKeywordParity(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	input := []byte("defer deferred for true")
	file := source.NewFile("previous.fy", input)
	for _, testCase := range []struct {
		name     string
		previous bool
		first    lexer.Kind
	}{
		{"current", false, lexer.Keyword},
		{"previous", true, lexer.Identifier},
	} {
		budget := source.NewBudget(source.DefaultLimits(), len(input))
		var result lexer.Result
		if testCase.previous {
			result = lexer.LexPreviousRevision(file, budget)
		} else {
			result = lexer.Lex(file, budget)
		}
		if len(result.Tokens) != 8 || len(result.Diagnostics) != 0 ||
			result.Tokens[0].Kind != testCase.first || result.Tokens[2].Kind != lexer.Identifier ||
			result.Tokens[4].Kind != lexer.Keyword || result.Tokens[6].Kind != lexer.Keyword ||
			result.Tokens[7].Kind != lexer.EOF {
			t.Fatalf("%s Go lexer tokens = %+v diagnostics = %+v", testCase.name, result.Tokens, result.Diagnostics)
		}
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	core, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, geometric := range []bool{false, true} {
				name := "C/up-front"
				if geometric {
					name = "C/geometric"
				}
				t.Run(name, func(t *testing.T) {
					code := scannerCBudgetDefines(geometric) + "#define FC_SCANNER_LIBRARY\n" + string(core) + scannerPreviousRevisionCProbe
					image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("C previous-revision keyword parity: %+v %v", outcome, err)
					}
				})
			}
			for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
				t.Run("Fairy/"+variant.name, func(t *testing.T) {
					program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, scannerPreviousRevisionFairyMain)
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("Fairy previous-revision keyword parity: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}

const scannerPreviousRevisionCProbe = `
static int check_previous_revision(int previous, unsigned first_kind) {
    static const unsigned char input[] = "defer deferred for true";
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    struct fc_token tokens[8] = {0};
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    struct fc_diagnostic diagnostics[1] = {0};
#endif
    struct fc_scanner scanner = {0};
    scanner.source = input;
    scanner.length = sizeof(input) - 1u;
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    scanner.tokens = tokens;
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    scanner.diagnostics = diagnostics;
#endif
    scanner.previous_revision = previous;
    scanner.work_limit = FC_CANDIDATE_WORK;
    scanner.token_limit = FC_CANDIDATE_TOKENS;
    int status = fc_scan(&scanner);
    int bad = status != 0 || scanner.token_count != 8u || scanner.diagnostic_count != 0u ||
              scanner.tokens[0].kind != first_kind || scanner.tokens[0].start != 0u ||
              scanner.tokens[0].end != 5u || scanner.tokens[2].kind != FC_IDENTIFIER ||
              scanner.tokens[4].kind != FC_KEYWORD || scanner.tokens[6].kind != FC_KEYWORD ||
              scanner.tokens[7].kind != FC_EOF || scanner.tokens[7].start != 23u;
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
    free(scanner.tokens);
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    free(scanner.diagnostics);
#endif
    return bad;
}

int main(void) {
    if (check_previous_revision(0, FC_KEYWORD)) return 1;
    if (check_previous_revision(1, FC_IDENTIFIER)) return 2;
    return 0;
}
`

const scannerPreviousRevisionFairyMain = `module main;
import scanner as scanner;
import std::bytes as bytes;

fn i32 check(bool previous, u32 first_kind) {
    let bytes::Buffer owned = switch (bytes::copy(b"defer deferred for true"[..])) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    let scanner::Stream stream = switch (scanner::collect(owned, previous, 32768, 1048576, 256, 16777216)) {
        case .ok(value): { value }
        case .err(_): { return 2; }
    };
    if (scanner::token_count(ref stream) != 8 || scanner::diagnostic_count(ref stream) != 0) { return 3; }
    let scanner::TokenInfo first = scanner::token_info(ref stream, 0);
    if (first.kind != first_kind || first.start != 0 || first.end != 5) { return 4; }
    if (scanner::token_info(ref stream, 2).kind != 3 ||
        scanner::token_info(ref stream, 4).kind != 4 ||
        scanner::token_info(ref stream, 6).kind != 4) { return 5; }
    let scanner::TokenInfo eof = scanner::token_info(ref stream, 7);
    if (eof.kind != 1 || eof.start != 23 || eof.end != 23) { return 6; }
    return 0;
}

fn i32 main() {
    if (check(false, 4) != 0) { return 1; }
    if (check(true, 3) != 0) { return 2; }
    return 0;
}
`

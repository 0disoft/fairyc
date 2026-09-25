package compiler_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/source"
)

type scannerConfiguredByteLimitCase struct {
	name  string
	data  []byte
	limit byte
}

func scannerConfiguredByteLimitCases() []scannerConfiguredByteLimitCase {
	return []scannerConfiguredByteLimitCase{
		{name: "empty at zero", limit: 0},
		{name: "first byte rejected", data: []byte("a"), limit: 0},
		{name: "first byte admitted", data: []byte("a"), limit: 1},
		{name: "BOM rejected", data: []byte{0xef, 0xbb, 0xbf, 'a'}, limit: 3},
		{name: "BOM admitted", data: []byte{0xef, 0xbb, 0xbf}, limit: 3},
		{name: "invalid byte rejected", data: []byte{0xff}, limit: 0},
		{name: "invalid byte admitted", data: []byte{0xff}, limit: 1},
		{name: "decimal rejected", data: []byte("1e3"), limit: 2},
		{name: "decimal admitted", data: []byte("1e3"), limit: 3},
		{name: "string rejected", data: []byte("\"a\""), limit: 2},
		{name: "string admitted", data: []byte("\"a\""), limit: 3},
		{name: "CRLF rejected", data: []byte("\r\n"), limit: 1},
		{name: "CRLF admitted", data: []byte("\r\n"), limit: 2},
	}
}

// A caller-supplied byte limit is distinct from the hard 32,768-byte
// candidate cap. Compare the independent C reference and both Fairy forms
// against the Go lexer on the same accepted/rejected source boundaries.
func TestPrivateVectorSDKCandidateConfiguredCByteLimitParity(t *testing.T) {
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
	cases := scannerConfiguredByteLimitCases()
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	fairyMain := scannerConfiguredByteLimitFairyMain(cases)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, geometric := range []bool{false, true} {
				name := "preallocated"
				if geometric {
					name = "geometric"
				}
				t.Run("C/"+name, func(t *testing.T) {
					code := scannerCBudgetDefines(geometric) + "#define FC_SCANNER_LIBRARY\n" + string(core) + scannerConfiguredByteLimitCProbe
					image, buildErr := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
					if buildErr != nil {
						t.Fatal(buildErr)
					}
					for _, item := range cases {
						t.Run(item.name, func(t *testing.T) {
							frame := append([]byte{item.limit, byte(len(item.data))}, item.data...)
							outcome, runErr := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(frame))
							limits := source.DefaultLimits()
							limits.MaxBytes = int(item.limit)
							want := scannerGoObservationWithLimits(item.data, limits)
							if runErr != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" || outcome.Stdout != want {
								t.Fatalf("configured C byte limit: %+v %v\nwant %q", outcome, runErr, want)
							}
						})
					}
				})
			}
			for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
				t.Run("Fairy/"+variant.name, func(t *testing.T) {
					program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, fairyMain)
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
					outcome, runErr := cdriver.ExecuteWindowsHost(image)
					if runErr != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("configured Fairy byte limit: %+v %v", outcome, runErr)
					}
					lines := strings.Split(strings.TrimSuffix(outcome.Stdout, "\n"), "\n")
					if len(lines) != len(cases) {
						t.Fatalf("Fairy reports=%d, want %d: %q", len(lines), len(cases), outcome.Stdout)
					}
					for i, item := range cases {
						limits := source.DefaultLimits()
						limits.MaxBytes = int(item.limit)
						want := scannerFairyNumericObservationWithLimits(item.data, limits)
						if lines[i] != want {
							t.Fatalf("%s Fairy/Go byte-limit mismatch\n got: %q\nwant: %q", item.name, lines[i], want)
						}
					}
				})
			}
		})
	}
}

func scannerConfiguredByteLimitFairyMain(cases []scannerConfiguredByteLimitCase) string {
	var main strings.Builder
	main.WriteString("module main;\nimport scanner as scanner;\nimport std::io as io;\n")
	main.WriteString("@effects(console) fn i32 main() {\n")
	for i, item := range cases {
		fmt.Fprintf(&main, "    if (scanner::report_work(b\"%s\"[..], false, %d, 1048576, 256, 16777216) != 0) { return %d; }\n",
			scannerFairyByteLiteral(item.data), item.limit, i+1)
		main.WriteString("    assert(switch (io::write(\"\\n\")) { case .ok(_): { true } case .err(_): { false } });\n")
	}
	main.WriteString("    return 0;\n}\n")
	return main.String()
}

const scannerConfiguredByteLimitCProbe = `
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 ||
        _setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    unsigned char header[2];
    unsigned char input[64] = {0};
    if (fread(header, 1, sizeof(header), stdin) != sizeof(header) ||
        header[1] > sizeof(input)) return FC_BAD_INPUT;
    size_t length = header[1];
    if (fread(input, 1, length, stdin) != length || getchar() != EOF) return FC_BAD_INPUT;
    struct fc_token *tokens = NULL;
    struct fc_diagnostic *diagnostics = NULL;
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    tokens = (struct fc_token *)calloc(length + 2u, sizeof(*tokens));
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    diagnostics = (struct fc_diagnostic *)calloc(length + 2u, sizeof(*diagnostics));
#endif
    if (
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
        tokens == NULL ||
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
        diagnostics == NULL ||
#endif
        0) { free(tokens); free(diagnostics); return 3; }
    struct fc_scanner scanner = {0};
    scanner.source = input;
    scanner.length = length;
    scanner.tokens = tokens;
    scanner.diagnostics = diagnostics;
    scanner.work_limit = FC_CANDIDATE_WORK;
    scanner.token_limit = FC_CANDIDATE_TOKENS;
    scanner.byte_limit_set = 1;
    scanner.byte_limit = header[0];
    int status = fc_scan(&scanner);
    if (status == 0) {
        qsort(scanner.diagnostics, scanner.diagnostic_count,
              sizeof(*scanner.diagnostics), fc_compare_diagnostic);
        fc_print(&scanner);
    }
    fc_release_cooked(&scanner);
    fc_release_diagnostics(&scanner);
    free(scanner.tokens);
    free(scanner.diagnostics);
    return status;
}
`

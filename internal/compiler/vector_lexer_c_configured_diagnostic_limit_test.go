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

type scannerConfiguredDiagnosticLimitCase struct {
	name  string
	data  []byte
	limit byte
}

func scannerConfiguredDiagnosticLimitCases() []scannerConfiguredDiagnosticLimitCase {
	return []scannerConfiguredDiagnosticLimitCase{
		{name: "empty at zero", limit: 0},
		{name: "valid at zero", data: []byte("a"), limit: 0},
		{name: "first invalid rejected", data: []byte("#"), limit: 0},
		{name: "first invalid admitted", data: []byte("#"), limit: 1},
		{name: "second invalid rejected", data: []byte("##"), limit: 1},
		{name: "second invalid admitted", data: []byte("##"), limit: 2},
		{name: "third invalid rejected", data: []byte("###"), limit: 2},
		{name: "unclosed comment rejected", data: []byte("/*"), limit: 0},
		{name: "unclosed comment admitted", data: []byte("/*"), limit: 1},
	}
}

// Compare caller-selected diagnostic budgets, including zero, with the Go
// lexer and both real Fairy loop forms. The C default remains 256 when unset.
func TestPrivateVectorSDKCandidateConfiguredCDiagnosticLimitParity(t *testing.T) {
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
	cases := scannerConfiguredDiagnosticLimitCases()
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	fairyMain := scannerConfiguredDiagnosticLimitFairyMain(cases)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, geometric := range []bool{false, true} {
				name := "preallocated"
				if geometric {
					name = "geometric"
				}
				t.Run("C/"+name, func(t *testing.T) {
					code := scannerCBudgetDefines(geometric) + "#define FC_SCANNER_LIBRARY\n" + string(core) + scannerConfiguredDiagnosticLimitCProbe
					image, buildErr := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
					if buildErr != nil {
						t.Fatal(buildErr)
					}
					for _, item := range cases {
						t.Run(item.name, func(t *testing.T) {
							frame := append([]byte{item.limit, byte(len(item.data))}, item.data...)
							outcome, runErr := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(frame))
							limits := source.DefaultLimits()
							limits.MaxDiagnostics = int(item.limit)
							want := scannerGoObservationWithLimits(item.data, limits)
							if runErr != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" || outcome.Stdout != want {
								t.Fatalf("configured C diagnostic limit: %+v %v\nwant %q", outcome, runErr, want)
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
						t.Fatalf("configured Fairy diagnostic limit: %+v %v", outcome, runErr)
					}
					lines := strings.Split(strings.TrimSuffix(outcome.Stdout, "\n"), "\n")
					if len(lines) != len(cases) {
						t.Fatalf("Fairy reports=%d, want %d: %q", len(lines), len(cases), outcome.Stdout)
					}
					for i, item := range cases {
						limits := source.DefaultLimits()
						limits.MaxDiagnostics = int(item.limit)
						want := scannerFairyNumericObservationWithLimits(item.data, limits)
						if lines[i] != want {
							t.Fatalf("%s Fairy/Go diagnostic-limit mismatch\n got: %q\nwant: %q", item.name, lines[i], want)
						}
					}
				})
			}
		})
	}
}

func scannerConfiguredDiagnosticLimitFairyMain(cases []scannerConfiguredDiagnosticLimitCase) string {
	var main strings.Builder
	main.WriteString("module main;\nimport scanner as scanner;\nimport std::io as io;\n")
	main.WriteString("@effects(console) fn i32 main() {\n")
	for i, item := range cases {
		fmt.Fprintf(&main, "    if (scanner::report_work(b\"%s\"[..], false, 32768, 1048576, %d, 16777216) != 0) { return %d; }\n",
			scannerFairyByteLiteral(item.data), item.limit, i+1)
		main.WriteString("    assert(switch (io::write(\"\\n\")) { case .ok(_): { true } case .err(_): { false } });\n")
	}
	main.WriteString("    return 0;\n}\n")
	return main.String()
}

const scannerConfiguredDiagnosticLimitCProbe = `
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
    scanner.diagnostic_limit_set = 1;
    scanner.diagnostic_limit = header[0];
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

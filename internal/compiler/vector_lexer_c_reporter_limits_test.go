package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/source"
)

type scannerBudgetReporterLimitCase struct {
	name   string
	data   []byte
	limits source.Limits
}

func scannerBudgetReporterLimitCases() []scannerBudgetReporterLimitCase {
	base := source.DefaultLimits()
	base.MaxBytes = 32768
	var cases []scannerBudgetReporterLimitCase
	for _, item := range scannerConfiguredByteLimitCases() {
		limits := base
		limits.MaxBytes = int(item.limit)
		cases = append(cases, scannerBudgetReporterLimitCase{"bytes/" + item.name, item.data, limits})
	}
	for _, item := range scannerConfiguredDiagnosticLimitCases() {
		limits := base
		limits.MaxDiagnostics = int(item.limit)
		cases = append(cases, scannerBudgetReporterLimitCase{"diagnostics/" + item.name, item.data, limits})
	}
	add := func(name string, data []byte, byteLimit, tokenLimit, diagnosticLimit, workLimit int) {
		limits := base
		limits.MaxBytes = byteLimit
		limits.MaxTokens = tokenLimit
		limits.MaxDiagnostics = diagnosticLimit
		limits.MaxWorkUnits = workLimit
		cases = append(cases, scannerBudgetReporterLimitCase{name, data, limits})
	}
	add("work before token", []byte("@x"), 32768, 0, 256, 0)
	add("token after first work", []byte("@x"), 32768, 0, 256, 1)
	add("token before later invalid", []byte("a@"), 32768, 1, 256, 1)
	add("work before later invalid", []byte("a@"), 32768, 1, 256, 2)
	add("byte before work", []byte("a@"), 1, 0, 256, 1)
	add("work after byte admitted", []byte("a@"), 2, 0, 256, 1)
	add("BOM work exhausted", []byte{0xef, 0xbb, 0xbf}, 32768, 1, 256, 2)
	add("BOM token exhausted", []byte{0xef, 0xbb, 0xbf}, 32768, 0, 256, 3)
	add("decimal work exhausted", []byte("1e3"), 32768, 1048576, 256, 2)
	add("string work exhausted", []byte(`"abc"`), 32768, 1048576, 256, 3)
	return cases
}

func TestPrivateVectorSDKCandidateBudgetReporterLimits(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	cases := scannerBudgetReporterLimitCases()
	if len(cases) != 32 {
		t.Fatalf("reporter limit corpus changed: %d cases", len(cases))
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, geometric := range []bool{false, true} {
				name := "preallocated"
				if geometric {
					name = "geometric"
				}
				t.Run("C/"+name, func(t *testing.T) {
					image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, scannerCBudgetReporterLimitsSource(t, cases, geometric))
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("C limit reporter execution: %+v %v", outcome, err)
					}
					scannerCheckBudgetReporterLimitLines(t, outcome.Stdout, cases)
				})
			}
			for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
				t.Run("Fairy/"+variant.name, func(t *testing.T) {
					program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, scannerBudgetReporterLimitsFairyMain(cases))
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("Fairy limit reporter execution: %+v %v", outcome, err)
					}
					scannerCheckBudgetReporterLimitLines(t, outcome.Stdout, cases)
				})
			}
		})
	}
}

func scannerCBudgetReporterLimitsSource(t *testing.T, cases []scannerBudgetReporterLimitCase, geometric bool) string {
	t.Helper()
	var result strings.Builder
	result.WriteString(scannerCBudgetDefines(geometric))
	result.WriteString("#define FC_BUDGET_FAIRY_REPORT\n#define FC_SCANNER_LIBRARY\n#define FC_SCANNER_WORKLOAD_LIBRARY\n")
	for _, name := range []string{"scanner_reference.c", "scanner_workload_reference.c", "support.c"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", name))
		if err != nil {
			t.Fatal(err)
		}
		result.Write(data)
		result.WriteByte('\n')
	}
	for i, item := range cases {
		fmt.Fprintf(&result, "static const unsigned char case_%d[] = {", i)
		if len(item.data) == 0 {
			result.WriteString("0")
		}
		for j, value := range item.data {
			if j != 0 {
				result.WriteByte(',')
			}
			fmt.Fprintf(&result, "0x%02x", value)
		}
		result.WriteString("};\n")
	}
	result.WriteString(`
static int report_case(const unsigned char *input, size_t length,
                       size_t byte_limit, size_t token_limit,
                       size_t diagnostic_limit, size_t work_limit) {
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
    scanner.byte_limit_set = 1;
    scanner.byte_limit = byte_limit;
    scanner.token_limit = token_limit;
    scanner.diagnostic_limit_set = 1;
    scanner.diagnostic_limit = diagnostic_limit;
    scanner.work_limit = work_limit;
    int status = fc_scan(&scanner);
    if (status == 0) {
        qsort(scanner.diagnostics, scanner.diagnostic_count,
              sizeof(*scanner.diagnostics), fc_compare_diagnostic);
        fc_report_fairy(&scanner);
        if (putchar('\n') == EOF) status = 2;
    }
    fc_release_cooked(&scanner);
    fc_release_diagnostics(&scanner);
    free(scanner.tokens);
    free(scanner.diagnostics);
    return status;
}
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdout), _O_BINARY) == -1) return 2;
#endif
`)
	for i, item := range cases {
		limits := item.limits
		fmt.Fprintf(&result, "    if (report_case(case_%d, %d, %d, %d, %d, %d) != 0) return %d;\n",
			i, len(item.data), limits.MaxBytes, limits.MaxTokens, limits.MaxDiagnostics, limits.MaxWorkUnits, i+1)
	}
	result.WriteString("    return 0;\n}\n")
	return result.String()
}

func scannerBudgetReporterLimitsFairyMain(cases []scannerBudgetReporterLimitCase) string {
	var main strings.Builder
	main.WriteString("module main;\nimport scanner as scanner;\nimport std::io as io;\n")
	main.WriteString("@link_name(\"fairyc_fixture_vector_clean\") unsafe extern C fn i32 clean();\n")
	main.WriteString("@effects(console) fn i32 main() {\n")
	for i, item := range cases {
		limits := item.limits
		fmt.Fprintf(&main, "    if (scanner::report_work(b\"%s\"[..], false, %d, %d, %d, %d) != 0) { return %d; }\n",
			scannerFairyByteLiteral(item.data), limits.MaxBytes, limits.MaxTokens,
			limits.MaxDiagnostics, limits.MaxWorkUnits, i+1)
		main.WriteString("    assert(switch (io::write(\"\\n\")) { case .ok(_): { true } case .err(_): { false } });\n")
	}
	main.WriteString("    assert(unsafe { clean() } == 1);\n    return 0;\n}\n")
	return main.String()
}

func scannerCheckBudgetReporterLimitLines(t *testing.T, output string, cases []scannerBudgetReporterLimitCase) {
	t.Helper()
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("reporter output has no final newline: %q", output)
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != len(cases) {
		t.Fatalf("reporter lines=%d, want %d", len(lines), len(cases))
	}
	for i, item := range cases {
		want := scannerFairyNumericObservationWithLimits(item.data, item.limits)
		if lines[i] != want {
			t.Fatalf("%s reporter mismatch\n got: %q\nwant: %q", item.name, lines[i], want)
		}
	}
}

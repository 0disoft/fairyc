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

// The benchmark reporter must agree on every field, not only the timed sample.
func TestPrivateVectorSDKCandidateBudgetReporterCorpus(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	inputs := scannerCReferenceInputs()
	if len(inputs) != 38 {
		t.Fatalf("lexical corpus changed: %d cases", len(inputs))
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
					image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, scannerCBudgetReporterCorpusSource(t, inputs, geometric))
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("C reporter execution: %+v %v", outcome, err)
					}
					scannerCheckBudgetReporterLines(t, outcome.Stdout, inputs)
				})
			}
			for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
				t.Run("Fairy/"+variant.name, func(t *testing.T) {
					program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, scannerLexicalParityMain(inputs))
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, "")
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("Fairy reporter execution: %+v %v", outcome, err)
					}
					scannerCheckBudgetReporterLines(t, outcome.Stdout, inputs)
				})
			}
		})
	}
}

func scannerCBudgetReporterCorpusSource(t *testing.T, inputs []struct {
	name string
	data []byte
}, geometric bool) string {
	t.Helper()
	var source strings.Builder
	source.WriteString(scannerCBudgetDefines(geometric))
	source.WriteString("#define FC_BUDGET_FAIRY_REPORT\n#define FC_SCANNER_LIBRARY\n#define FC_SCANNER_WORKLOAD_LIBRARY\n")
	for _, name := range []string{"scanner_reference.c", "scanner_workload_reference.c", "support.c"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", name))
		if err != nil {
			t.Fatal(err)
		}
		source.Write(data)
		source.WriteByte('\n')
	}
	for i, item := range inputs {
		fmt.Fprintf(&source, "static const unsigned char case_%d[] = {", i)
		if len(item.data) == 0 {
			source.WriteString("0")
		}
		for j, value := range item.data {
			if j != 0 {
				source.WriteByte(',')
			}
			fmt.Fprintf(&source, "0x%02x", value)
		}
		source.WriteString("};\n")
	}
	source.WriteString("int main(void) {\n#ifdef _WIN32\n    if (_setmode(_fileno(stdout), _O_BINARY) == -1) return 2;\n#endif\n    uint64_t work = 0;\n    int status;\n")
	for i, item := range inputs {
		fmt.Fprintf(&source, "    status = fc_workload_round(case_%d, %d, 1, &work);\n", i, len(item.data))
		source.WriteString("    if (status != 0 || putchar('\\n') == EOF) return status != 0 ? status : 2;\n")
	}
	source.WriteString("    return 0;\n}\n")
	return source.String()
}

func scannerCheckBudgetReporterLines(t *testing.T, output string, inputs []struct {
	name string
	data []byte
}) {
	t.Helper()
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("reporter output has no final newline: %q", output)
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != len(inputs) {
		t.Fatalf("reporter lines=%d, want %d", len(lines), len(inputs))
	}
	for i, item := range inputs {
		want := scannerFairyNumericObservation(item.data)
		if lines[i] != want {
			t.Fatalf("%s reporter mismatch\n got: %q\nwant: %q", item.name, lines[i], want)
		}
	}
}

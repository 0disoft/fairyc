package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPrivateVectorSDKCandidateCSourceLimit(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	const define = "#define FC_CANDIDATE_SOURCE_LIMIT\n"
	if !strings.Contains(scannerCBenchmarkSource(t), define) ||
		!strings.Contains(scannerCGeometricBenchmarkSource(t), define) {
		t.Fatal("both C budget storage lanes must enforce the candidate source cap")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	core, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	const libraryProbe = `
int main(void) {
    static unsigned char source[FC_MAX_SOURCE + 1u];
    struct fc_token tokens[2] = {0};
    struct fc_diagnostic diagnostics[1] = {0};
    memset(source, 'a', sizeof(source));
    struct fc_scanner scanner = {0};
    scanner.source = source;
    scanner.length = FC_MAX_SOURCE;
    scanner.tokens = tokens;
    scanner.diagnostics = diagnostics;
    if (fc_scan(&scanner) != 0 || scanner.token_count != 2u ||
        tokens[0].kind != FC_IDENTIFIER || tokens[0].end != FC_MAX_SOURCE ||
        tokens[1].kind != FC_EOF) return 1;
    scanner.length++;
    scanner.position = 0;
    scanner.token_count = 0;
    if (fc_scan(&scanner) != FC_CANDIDATE_SOURCE_SIZE ||
        scanner.token_count != 0 || scanner.diagnostic_count != 0 ||
        scanner.position != 0) return 2;
    return 0;
}
`
	exact := []byte(strings.Repeat("a", 32768))
	over := []byte(strings.Repeat("a", 32769))
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, lane := range []struct {
				name, prefix string
				overStatus   int32
			}{
				{"candidate", define, 4},
				{"Go oracle", "", 8},
			} {
				t.Run(lane.name, func(t *testing.T) {
					image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, lane.prefix+string(core))
					if err != nil {
						t.Fatal(err)
					}
					exactOutcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, scannerCInput(exact))
					if err != nil || exactOutcome.ExitCode != 0 || exactOutcome.Stderr != "" ||
						exactOutcome.TrapControl != "" || exactOutcome.Stdout != scannerGoObservation(exact) {
						t.Fatalf("exact source limit: %+v %v", exactOutcome, err)
					}
					overOutcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, scannerCInput(over))
					if err != nil || overOutcome.ExitCode != lane.overStatus ||
						overOutcome.Stdout != "" || overOutcome.Stderr != "" || overOutcome.TrapControl != "" {
						t.Fatalf("over source limit: %+v %v", overOutcome, err)
					}
				})
			}
			t.Run("library", func(t *testing.T) {
				image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile,
					define+"#define FC_SCANNER_LIBRARY\n"+string(core)+libraryProbe)
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsHost(image)
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" ||
					outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("library source limit: %+v %v", outcome, err)
				}
			})
		})
	}
}

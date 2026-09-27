package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Compare the C reference's late diagnostic allocation failures with the
// full Fairy scanner, including cleanup of its earlier token allocation.
func TestPrivateVectorSDKCandidateFairyDiagnosticAllocationFailure(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		t.Run(variant.name, func(t *testing.T) {
			for _, failedCall := range []int{2, 3} {
				t.Run(fmt.Sprintf("call-%d", failedCall), func(t *testing.T) {
					main := fmt.Sprintf(scannerDiagnosticFailureMain, failedCall, failedCall)
					program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, main)
					for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
						t.Run(profile.Name, func(t *testing.T) {
							image, _ := buildRealScannerVectorNative(t, program, entry, profile, scannerDiagnosticFailureSupport)
							outcome, err := cdriver.ExecuteWindowsHost(image)
							if err != nil || outcome.ExitCode != 3 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
								t.Fatalf("Fairy diagnostic allocation failure: %+v %v", outcome, err)
							}
						})
					}
				})
			}
		})
	}
}

// The ninth standalone CR grows both token and diagnostic vectors. Fail the
// latter allocation after the eight completed diagnostic entries are owned.
func TestPrivateVectorSDKCandidateFairyDiagnosticGrowthFailure(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		t.Run(variant.name, func(t *testing.T) {
			program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, scannerDiagnosticGrowthFailureMain)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, scannerDiagnosticGrowthFailureSupport)
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 3 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("Fairy diagnostic growth failure: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}

const scannerDiagnosticGrowthFailureSupport = `
void fairyc_fixture_fail_diagnostic_growth(void) { vector_fail_at = vector_calls + 4; }
int32_t fairyc_fixture_diagnostic_growth_failure_seen(void) {
    return vector_calls == 4 && vector_failures == 1 &&
           vector_allocations == 3 && vector_allocations == vector_frees &&
           vector_live == 0 ? 1 : 0;
}
`

const scannerDiagnosticGrowthFailureMain = `module main;
import scanner as scanner;
@link_name("fairyc_fixture_fail_diagnostic_growth") unsafe extern C fn void fail_growth();
@link_name("fairyc_fixture_diagnostic_growth_failure_seen") unsafe extern C fn i32 failure_seen();
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn i32 main() {
    unsafe { fail_growth(); }
    let i32 status = scanner::report_work(b"\r\r\r\r\r\r\r\r\r"[..], false, 32768, 1048576, 256, 16777216);
    assert(unsafe { failure_seen() } == 1);
    assert(unsafe { clean() } == 1);
    return status;
}`

const scannerDiagnosticFailureSupport = `
void fairyc_fixture_fail_diagnostic_call(int32_t offset) {
    vector_fail_at = vector_calls + (size_t)offset;
}
int32_t fairyc_fixture_diagnostic_failure_seen(int32_t expected_calls) {
    return vector_calls == (size_t)expected_calls && vector_failures == 1 &&
           vector_allocations == (size_t)(expected_calls - 1) &&
           vector_allocations == vector_frees && vector_live == 0 ? 1 : 0;
}
`

const scannerDiagnosticFailureMain = `module main;
import scanner as scanner;
@link_name("fairyc_fixture_fail_diagnostic_call") unsafe extern C fn void fail_call(i32 offset);
@link_name("fairyc_fixture_diagnostic_failure_seen") unsafe extern C fn i32 failure_seen(i32 calls);
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn i32 main() {
    unsafe { fail_call(%d); }
    let i32 status = scanner::report_work(b"/*"[..], false, 32768, 1048576, 256, 16777216);
    assert(unsafe { failure_seen(%d) } == 1);
    assert(unsafe { clean() } == 1);
    return status;
}`

func TestPrivateVectorSDKCandidateCDiagnosticAllocationOrder(t *testing.T) {
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
	const injection = `
#include <stdlib.h>
static size_t probe_calls, probe_live, probe_fail_at;
static void *probe_malloc(size_t size) {
    probe_calls++;
    if (probe_calls == probe_fail_at) return NULL;
    void *value = malloc(size);
    if (value != NULL) probe_live++;
    return value;
}
static void probe_free(void *value) {
    if (value != NULL) probe_live--;
    free(value);
}
#define malloc probe_malloc
#define free probe_free
#define FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
#define FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
#define FC_SCANNER_LIBRARY
`
	const probe = `
static int check(size_t fail_at, int wanted_payload_failure,
                 int wanted_storage_failure, size_t wanted_calls) {
    struct fc_token tokens[3] = {0};
    struct fc_scanner scanner = {0};
    scanner.source = (const unsigned char *)"/*";
    scanner.length = 2;
    scanner.tokens = tokens;
    probe_calls = 0;
    probe_live = 0;
    probe_fail_at = fail_at;
    int status = fc_scan(&scanner);
    int failing = fail_at != 0;
    int bad = status != (failing ? 3 : 0) || probe_calls != wanted_calls ||
              scanner.diagnostic_payload_failed != wanted_payload_failure ||
              scanner.diagnostic_allocation_failed != wanted_storage_failure ||
              scanner.diagnostic_count != (failing ? 0u : 1u) ||
              scanner.token_count != (failing ? 1u : 2u) ||
              tokens[0].kind != FC_TRIVIA || tokens[0].start != 0 || tokens[0].end != 2u;
    if (!bad && !failing)
        bad = scanner.diagnostics[0].token_length != 2u ||
              memcmp(scanner.diagnostics[0].token, "*/", 2u) != 0 ||
              tokens[1].kind != FC_EOF;
    fc_release_cooked(&scanner);
    fc_release_diagnostics(&scanner);
    probe_free(scanner.diagnostics);
    return bad || probe_live != 0;
}
static int check_growth(size_t fail_at) {
    struct fc_token tokens[10] = {0};
    struct fc_scanner scanner = {0};
    scanner.source = (const unsigned char *)"\r\r\r\r\r\r\r\r\r";
    scanner.length = 9;
    scanner.tokens = tokens;
    probe_calls = 0;
    probe_live = 0;
    probe_fail_at = fail_at;
    int status = fc_scan(&scanner);
    int failing = fail_at != 0;
    int bad = status != (failing ? 3 : 0) || probe_calls != 2u ||
              scanner.token_count != (failing ? 9u : 10u) ||
              scanner.diagnostic_count != (failing ? 8u : 9u) ||
              scanner.diagnostic_capacity != (failing ? 8u : 16u) ||
              scanner.diagnostic_allocation_failed != failing;
    if (!bad) {
        for (size_t i = 0; i < 8u; i++) {
            if (tokens[i].kind != FC_INVALID ||
                tokens[i].start != i || tokens[i].end != i + 1u ||
                strcmp(scanner.diagnostics[i].code, "E0001") != 0 ||
                scanner.diagnostics[i].start != i ||
                scanner.diagnostics[i].end != i + 1u) bad = 1;
        }
        if (!failing && (tokens[9].kind != FC_EOF ||
                         strcmp(scanner.diagnostics[8].code, "E0001") != 0)) bad = 1;
    }
    fc_release_cooked(&scanner);
    fc_release_diagnostics(&scanner);
    probe_free(scanner.diagnostics);
    return bad || probe_live != 0;
}
int main(void) {
    if (check(1u, 1, 0, 1u)) return 1;
    if (check(2u, 0, 1, 2u)) return 2;
    if (check(0u, 0, 0, 2u)) return 3;
    if (check_growth(2u)) return 4;
    if (check_growth(0u)) return 5;
    return 0;
}
`
	code := injection + string(core) + probe
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(image)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("C diagnostic allocation order: %+v %v", outcome, err)
			}
		})
	}
}

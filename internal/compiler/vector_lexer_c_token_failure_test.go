package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPrivateVectorSDKCandidateCTokenAllocationFailureOrder(t *testing.T) {
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
#define FC_CANDIDATE_GEOMETRIC_TOKENS
#define FC_SCANNER_LIBRARY
`
	const probe = `
static int check(size_t fail_at, size_t wanted_calls) {
    struct fc_scanner scanner = {0};
    scanner.source = (const unsigned char *)"12";
    scanner.length = 2;
    probe_calls = 0;
    probe_live = 0;
    probe_fail_at = fail_at;
    int status = fc_scan(&scanner);
    int failing = fail_at != 0;
    int bad = status != (failing ? 3 : 0) || probe_calls != wanted_calls ||
              scanner.token_count != (failing ? 0u : 2u) ||
              scanner.token_allocation_failed != (fail_at == 3u);
    if (!bad && !failing)
        bad = scanner.tokens[0].kind != FC_INTEGER ||
              scanner.tokens[0].cooked_length != 2u ||
              memcmp(scanner.tokens[0].cooked, "12", 2u) != 0 ||
              scanner.tokens[1].kind != FC_EOF;
    fc_release_cooked(&scanner);
    probe_free(scanner.tokens);
    return bad || probe_live != 0;
}
static int check_growth(size_t fail_at) {
    struct fc_scanner scanner = {0};
    scanner.source = (const unsigned char *)";;;;;;;;;";
    scanner.length = 9;
    probe_calls = 0;
    probe_live = 0;
    probe_fail_at = fail_at;
    int status = fc_scan(&scanner);
    int failing = fail_at != 0;
    int bad = status != (failing ? 3 : 0) || probe_calls != 2u ||
              scanner.token_count != (failing ? 8u : 10u) ||
              scanner.token_capacity != (failing ? 8u : 16u) ||
              scanner.token_allocation_failed != failing ||
              scanner.tokens == NULL;
    if (!bad) {
        for (size_t i = 0; i < 8u; i++) {
            if (scanner.tokens[i].kind != FC_SYMBOL ||
                scanner.tokens[i].start != i ||
                scanner.tokens[i].end != i + 1u) bad = 1;
        }
        if (!failing && (scanner.tokens[8].kind != FC_SYMBOL ||
                         scanner.tokens[9].kind != FC_EOF)) bad = 1;
    }
    fc_release_cooked(&scanner);
    probe_free(scanner.tokens);
    return bad || probe_live != 0;
}
int main(void) {
    if (check(1u, 1u)) return 1;
    if (check(2u, 2u)) return 2;
    if (check(3u, 3u)) return 3;
    if (check(0u, 3u)) return 4;
    if (check_growth(2u)) return 5;
    if (check_growth(0u)) return 6;
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
				t.Fatalf("C token allocation order: %+v %v", outcome, err)
			}
		})
	}
}

func TestPrivateVectorSDKCandidateFairyFirstAllocationFailure(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		t.Run(variant.name, func(t *testing.T) {
			program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, scannerFirstAllocationFailureMain)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, scannerFirstAllocationFailureSupport)
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 3 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("Fairy scanner first allocation failure: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}

const scannerFirstAllocationFailureSupport = `
void fairyc_fixture_vector_fail_next(void) { vector_fail = 1; }
int32_t fairyc_fixture_vector_failure_seen(void) { return vector_failures == 1 ? 1 : 0; }
`

const scannerFirstAllocationFailureMain = `module main;
import scanner as scanner;
@link_name("fairyc_fixture_vector_fail_next") unsafe extern C fn void fail_next();
@link_name("fairyc_fixture_vector_failure_seen") unsafe extern C fn i32 failure_seen();
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn i32 main() {
    unsafe { fail_next(); }
    let i32 status = scanner::report_work(b"12"[..], false, 32768, 1048576, 256, 16777216);
    assert(unsafe { failure_seen() } == 1);
    assert(unsafe { clean() } == 1);
    return status;
}`

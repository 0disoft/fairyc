package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Sweep every allocation reached by one mixed cooked-token and diagnostic
// input in both independent C storage lanes. Caller-owned fixed storage is
// deliberately outside the injection boundary.
func TestPrivateVectorSDKCandidateCScannerAllocationSweep(t *testing.T) {
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
	for _, storage := range []struct {
		name      string
		geometric bool
	}{{"caller-owned", false}, {"geometric", true}} {
		t.Run(storage.name, func(t *testing.T) {
			code := cScannerAllocationSweepInjection + scannerCBudgetDefines(storage.geometric) +
				"#define FC_SCANNER_LIBRARY\n" + string(core) + cScannerAllocationSweepProbe
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" ||
						outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("C scanner allocation sweep: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}

const cScannerAllocationSweepInjection = `
#include <stdlib.h>
static size_t probe_calls, probe_live, probe_fail_at, probe_failures;
static void *probe_malloc(size_t size) {
    probe_calls++;
    if (probe_calls == probe_fail_at) { probe_failures++; return NULL; }
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
`

const cScannerAllocationSweepProbe = `
static int sweep_once(size_t fail_at) {
    static const unsigned char source[] = "1 2 3 4 5 6 7 8 9 \r\r\r\r\r\r\r\r\r/*";
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    struct fc_token fixed_tokens[32] = {0};
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    struct fc_diagnostic fixed_diagnostics[16] = {0};
#endif
    struct fc_scanner scanner = {0};
    scanner.source = source;
    scanner.length = sizeof(source) - 1u;
    scanner.token_limit = FC_CANDIDATE_TOKENS;
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    scanner.tokens = fixed_tokens;
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    scanner.diagnostics = fixed_diagnostics;
#endif
    scanner.work_limit = FC_CANDIDATE_WORK;
    probe_calls = probe_live = probe_failures = 0;
    probe_fail_at = fail_at;
    int status = fc_scan(&scanner);
    int bad = status != (fail_at == 0 ? 0 : 3) ||
              probe_failures != (fail_at == 0 ? 0u : 1u) ||
              scanner.token_count > 29u || scanner.diagnostic_count > 10u;
    if (fail_at == 0)
        bad |= probe_calls < 10u || probe_calls > 64u ||
               scanner.token_count != 29u || scanner.diagnostic_count != 10u ||
               scanner.tokens[0].kind != FC_INTEGER ||
               scanner.tokens[28].kind != FC_EOF;
    else
        bad |= probe_calls != fail_at;
    fc_release_cooked(&scanner);
    fc_release_diagnostics(&scanner);
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
    probe_free(scanner.tokens);
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    probe_free(scanner.diagnostics);
#endif
    if (probe_live != 0) bad = 1;
    if (bad)
        printf("fail=%zu status=%d calls=%zu failures=%zu live=%zu tokens=%zu diagnostics=%zu\n",
               fail_at, status, probe_calls, probe_failures, probe_live,
               scanner.token_count, scanner.diagnostic_count);
    return bad ? -1 : (int)probe_calls;
}
int main(void) {
    int calls = sweep_once(0);
    if (calls < 10 || calls > 64) return 1;
    for (int at = 1; at <= calls; at++) {
        if (sweep_once((size_t)at) != at) return at + 1;
    }
    return 0;
}
`

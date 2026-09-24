package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

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
int main(void) {
    if (check(1u, 1, 0, 1u)) return 1;
    if (check(2u, 0, 1, 2u)) return 2;
    if (check(0u, 0, 0, 2u)) return 3;
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

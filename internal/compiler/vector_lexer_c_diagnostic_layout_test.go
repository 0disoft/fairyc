package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestPrivateVectorSDKCandidateCDiagnosticEntryLayout(t *testing.T) {
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
	const probe = `
_Static_assert(sizeof(struct fc_diagnostic_buffer) == 32u, "Buffer must be four words");
_Static_assert(sizeof(struct fc_diagnostic) == 64u, "Entry must occupy 64 bytes");
_Static_assert(offsetof(struct fc_diagnostic, code) == 0u, "code offset");
_Static_assert(offsetof(struct fc_diagnostic, start) == 8u, "start offset");
_Static_assert(offsetof(struct fc_diagnostic, end) == 16u, "end offset");
_Static_assert(offsetof(struct fc_diagnostic, exit_repair) == 24u, "repair offset");
_Static_assert(offsetof(struct fc_diagnostic, token) == 32u, "Buffer offset");
int main(void) {
    static const unsigned char input[] = "/*";
    struct fc_token tokens[3] = {0};
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    struct fc_diagnostic diagnostics[2] = {0};
#endif
    struct fc_scanner scanner = {0};
    scanner.source = input;
    scanner.length = sizeof(input) - 1u;
    scanner.tokens = tokens;
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    scanner.diagnostics = diagnostics;
#endif
    int status = fc_scan(&scanner);
    int bad = status != 0 || scanner.token_count != 2u ||
              scanner.diagnostic_count != 1u ||
              scanner.diagnostics[0].code != 1u ||
              scanner.diagnostics[0].start != 0u ||
              scanner.diagnostics[0].end != 2u ||
              scanner.diagnostics[0].exit_repair != 1u ||
              scanner.diagnostics[0].token.data == NULL ||
              scanner.diagnostics[0].token.capacity != 2u ||
              scanner.diagnostics[0].token.alignment != 1u ||
              scanner.diagnostics[0].token.length != 2u;
    if (!bad)
        bad = memcmp(scanner.diagnostics[0].token.data, "*/", 2u) != 0;
    if (!bad) {
        struct fc_diagnostic plain = scanner.diagnostics[0];
        plain.exit_repair = 0u;
        bad = fc_compare_diagnostic(&plain, &scanner.diagnostics[0]) >= 0;
    }
    if (!bad) fc_print(&scanner);
    fc_release_diagnostics(&scanner);
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    free(scanner.diagnostics);
#endif
    return bad;
}
`
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		for _, storage := range []struct {
			name    string
			defines string
		}{
			{"preallocated", ""},
			{"geometric", "#define FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS\n"},
		} {
			t.Run(profile.Name+"/"+storage.name, func(t *testing.T) {
				code := "#include <stddef.h>\n#define FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT\n#define FC_CANDIDATE_DIAGNOSTIC_PAYLOAD\n#define FC_SCANNER_LIBRARY\n" + storage.defines + string(core) + probe
				image, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, code)
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsHost(image)
				want := scannerGoObservation([]byte("/*"))
				if err != nil || outcome.ExitCode != 0 || strings.ReplaceAll(outcome.Stdout, "\r\n", "\n") != want || outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("C diagnostic Entry layout: %+v %v\nwant %q", outcome, err, want)
				}
			})
		}
	}
}

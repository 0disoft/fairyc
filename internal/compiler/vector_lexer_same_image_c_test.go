package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// Establish the native C/Fairy coexistence boundary before using one image for
// adjacent timing. Existing lexical corpus tests own full reporter parity.
func TestPrivateVectorSDKCandidateSameImageCScannerParity(t *testing.T) {
	if os.Getenv("FAIRYC_SCANNER_SAME_IMAGE_C_PARITY") != "1" {
		t.Skip("opt-in same-image C/Fairy scanner parity")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	const sample = "let n=12345; /*open"
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)), source.NewBudget(source.DefaultLimits(), len(sample)))
	perRound := len(sample) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if perRound <= len(sample) || len(lexed.Diagnostics) == 0 {
		t.Fatal("scanner sample lost cooked or diagnostic coverage")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("scanner variants changed")
	}
	support := sameImageCScannerSupport(t)
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		t.Run(variant.name, func(t *testing.T) {
			program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, sameImageCScannerMain)
			artifact, _ := buildRealScannerVectorNative(t, program, entry, cdriver.OptimizedProfile, support)
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" ||
				strings.TrimSpace(outcome.Stdout) != strconv.Itoa(perRound) {
				t.Fatalf("%s same-image result: %+v %v; want %d", variant.name, outcome, err, perRound)
			}
		})
	}
}

func sameImageCScannerSupport(t *testing.T) string {
	t.Helper()
	parts := []string{scannerCBudgetDefines(false), "#define FC_SCANNER_LIBRARY\n#define FC_SCANNER_WORKLOAD_LIBRARY\n"}
	for _, name := range []string{"scanner_reference.c", "scanner_workload_reference.c"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", name))
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, string(data), "\n")
	}
	parts = append(parts, `
_Static_assert(sizeof(size_t) == sizeof(uint64_t), "same-image C scanner requires 64-bit size_t");
size_t fc_same_image_work(void) {
    static const unsigned char sample[] = "let n=12345; /*open";
    uint64_t work = 0;
    int status = fc_workload_round(sample, sizeof(sample) - 1u, 0, &work);
    if (status != 0) abort();
    return (size_t)work;
}
`)
	return strings.Join(parts, "")
}

const sameImageCScannerMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
import std::io as io;
@link_name("fc_same_image_work") unsafe extern C fn usize c_work();
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn i32 main() {
    let usize c = unsafe { c_work() };
    let bytes::Buffer owned = switch (bytes::copy(b"let n=12345; /*open")) {
        case .ok(value) : { value }
        case .err(_) : { assert(false); bytes::empty() }
    };
    let usize fairy = switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream) : {
            let scanner::Storage usage = scanner::storage(ref stream);
            usage.source_bytes + usage.cooked_bytes + usage.diagnostic_bytes + usage.tokens + usage.diagnostics
        }
        case .err(_) : { assert(false); 0 }
    };
    assert(c == fairy);
    assert(unsafe { clean() } == 1);
    assert(switch (io::write_usize(c)) { case .ok(_) : { true } case .err(_) : { false } });
    return 0;
}`

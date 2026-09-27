package compiler_test

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Exercise every allocation reached by one scan in a single native image.
// The source Buffer is created before arming the failure, so each offset
// targets scanner work rather than fixture setup.
func TestPrivateVectorSDKCandidateFairyScannerAllocationSweep(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	var wantCalls int32
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		t.Run(variant.name, func(t *testing.T) {
			program, entry, _ := scannerBudgetFairyProgramWithMain(t, variant, scannerAllocationSweepMain)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					image, _ := buildRealScannerVectorNative(t, program, entry, profile, scannerAllocationSweepSupport)
					outcome, err := cdriver.ExecuteWindowsHost(image)
					if err != nil || outcome.ExitCode < 12 || outcome.ExitCode > 64 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("Fairy scanner allocation sweep: %+v %v", outcome, err)
					}
					if wantCalls == 0 {
						wantCalls = outcome.ExitCode
					} else if outcome.ExitCode != wantCalls {
						t.Fatalf("scanner allocation calls changed: got %d, want %d", outcome.ExitCode, wantCalls)
					}
				})
			}
		})
	}
	t.Logf("swept %d scanner allocation calls in each source form and profile", wantCalls)
}

const scannerAllocationSweepSupport = `
static size_t sweep_start_calls, sweep_start_failures;
int32_t fairyc_fixture_sweep_begin(int32_t offset) {
    if (offset < 0 || vector_live != 1 || vector_allocations != vector_frees + 1) return 0;
    sweep_start_calls = vector_calls;
    sweep_start_failures = vector_failures;
    vector_fail_at = offset == 0 ? 0 : vector_calls + (size_t)offset;
    return 1;
}
int32_t fairyc_fixture_sweep_finish(int32_t offset, int32_t status) {
    size_t calls = vector_calls - sweep_start_calls;
    if (vector_live != 0 || vector_allocations != vector_frees) return -1;
    if (offset == 0) {
        if (status != 0 || vector_failures != sweep_start_failures || calls < 12 || calls > 64) return -2;
        return (int32_t)calls;
    }
    if (status != 3 || vector_failures != sweep_start_failures + 1 || calls < (size_t)offset) return -3;
    return 0;
}
`

const scannerAllocationSweepMain = `module main;
import scanner as scanner;
import std::bytes as bytes;
@link_name("fairyc_fixture_sweep_begin") unsafe extern C fn i32 begin(i32 offset);
@link_name("fairyc_fixture_sweep_finish") unsafe extern C fn i32 finish(i32 offset, i32 status);
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
fn i32 scan_once(i32 offset) {
    let bytes::Buffer owned = switch (bytes::copy(b"1 2 3 4 5 6 7 8 9 \r\r\r\r\r\r\r\r\r")) {
        case .ok(value): { value }
        case .err(_): { return -10; }
    };
    assert(unsafe { begin(offset) } == 1);
    let i32 status = switch (scanner::collect(owned, false, 32768, 1048576, 256, 16777216)) {
        case .ok(stream): {
            let scanner::Storage usage = scanner::storage(ref stream);
            assert(usage.cooked_bytes == 9 && usage.diagnostics == 9 && usage.tokens >= 19);
            scanner::release(stream);
            0
        }
        case .err(code): { code }
    };
    return unsafe { finish(offset, status) };
}
fn i32 main() {
    let i32 calls = scan_once(0);
    assert(calls >= 12 && calls <= 64);
    var i32 offset = 1;
    while (offset <= calls) {
        assert(scan_once(offset) == 0);
        offset += 1;
    }
    assert(unsafe { clean() } == 1);
    return calls;
}`

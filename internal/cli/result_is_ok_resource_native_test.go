package cli

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

type resultIsOkHeapLedger struct {
	Attempts, Allocations, Frees, Failures uint64
	PeakBytes, RetainedBytes               uint64
	LiveBytes, LiveBlocks                  uint64
	Status                                 int
}

func TestResultBranchReviewIsOkResourceFailureNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	variants := []struct{ name, test string }{
		{"switch", "switch (v::push<bytes::Buffer>(ref mut values, owned)) { case .ok(_): { true } case .err(_): { false } }"},
		{"is_ok", "v::push<bytes::Buffer>(ref mut values, owned).is_ok"},
	}
	baseline := make(map[string]resultIsOkHeapLedger)
	for _, variant := range variants {
		root := t.TempDir()
		copyLexerStreamExample(t, root)
		writeTestSource(t, root, "src/main.fy", `module main;
import std::bytes as bytes;
import std::vec as v;
@link_name("fairyc_heap_finish") unsafe extern C fn i32 finish(i32 status);
fn i32 exercise() {
    var v::Vec<bytes::Buffer> values = v::empty<bytes::Buffer>();
    var bytes::Buffer owned = bytes::empty();
    assert(switch (bytes::append_byte(ref mut owned, 120)) { case .ok(_): { true } case .err(_): { false } });
    let bool accepted = `+variant.test+`;
    assert(v::len<bytes::Buffer>(ref values) == if (accepted) { 1 } else { 0 });
    return if (accepted) { 0 } else { 3 };
}
fn i32 main() { let i32 status = exercise(); return unsafe { finish(status) }; }
`)
		var diagnostics strings.Builder
		lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "Result is_ok resource failure", &diagnostics)
		if code != ExitSuccess {
			t.Fatalf("%s lower: %s", variant.name, diagnostics.String())
		}
		unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		var symbols []string
		for _, symbol := range unit.ForeignSymbols {
			if !strings.HasPrefix(symbol, "fairyc_heap_") {
				symbols = append(symbols, symbol)
			}
		}
		support, err := windowsVectorSupportSource(symbols, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, failAt := range []int{0, 2} {
			linkedSupport := fmt.Sprintf("#define HEAP_FAIL_AT %d\n", failAt) + lexerHeapProbe + support
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit.Source, linkedSupport)
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("%s fail-at-%d %s: %+v %v", variant.name, failAt, profile.Name, outcome, err)
				}
				var row resultIsOkHeapLedger
				if err := json.Unmarshal([]byte(outcome.Stdout), &row); err != nil {
					t.Fatal(err)
				}
				wantStatus, wantAllocations, wantFailures := 0, uint64(2), uint64(0)
				if failAt == 2 {
					wantStatus, wantAllocations, wantFailures = 3, 1, 1
				}
				if row.Status != wantStatus || row.Attempts != 2 || row.Allocations != wantAllocations ||
					row.Frees != wantAllocations || row.Failures != wantFailures || row.LiveBytes != 0 || row.LiveBlocks != 0 {
					t.Fatalf("%s fail-at-%d %s ledger: %+v", variant.name, failAt, profile.Name, row)
				}
				key := fmt.Sprintf("%d/%s", failAt, profile.Name)
				if variant.name == "switch" {
					baseline[key] = row
				} else if row != baseline[key] {
					t.Fatalf("fail-at-%d %s ledger changed: switch=%+v is_ok=%+v", failAt, profile.Name, baseline[key], row)
				}
				t.Logf("%s fail-at-%d %s: status=%d alloc=%d free=%d failures=%d live=%d", variant.name, failAt, profile.Name, row.Status, row.Allocations, row.Frees, row.Failures, row.LiveBytes)
			}
		}
	}
}

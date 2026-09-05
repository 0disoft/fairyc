package contractstatenative

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

func TestVerifiedCellTransactionsRunLowAndOptimizedMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("contract-state native proof requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Skipf("contract-state native proof requires local x64 MSVC: %v", err)
	}
	runtimeSource, err := cstdlib.ContractStateRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, exit := range []contractstatemir.ExitKind{contractstatemir.ExitCommit, contractstatemir.ExitReject, contractstatemir.ExitTrap} {
		for _, deleting := range []bool{false, true} {
			input, coordinate := nativeStateInput(t, exit)
			if deleting {
				input.StateTransactions[0].Accesses[0] = hir.StateAccess{Kind: hir.StateAccessCellDelete, Field: "value"}
			}
			compiled, err := compiler.LowerAnalyzed(compiler.Analysis{Semantic: semantic.Result{HIR: input, CompletedThrough: semantic.PhaseFlow}})
			if err != nil {
				t.Fatal(err)
			}
			handoff, err := PrepareCompiled(compiled, 1, 1, coordinate)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := Emit(handoff, target.X8664WindowsMSVC)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				built, err := cdriver.BuildWindowsHost(toolchain, profile, artifact.Source+nativeProbe(exit, deleting), runtimeSource)
				if err != nil {
					t.Fatalf("build %s %s: %v", exit, profile.Name, err)
				}
				outcome, err := cdriver.ExecuteWindowsHost(built)
				if err != nil {
					t.Fatalf("execute %s %s: %v", exit, profile.Name, err)
				}
				if outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("%s %s outcome = %+v", exit, profile.Name, outcome)
				}
			}
		}
	}
}

func nativeProbe(exit contractstatemir.ExitKind, deleting bool) string {
	status := 0
	committed := 1
	trapCheck := ""
	if exit == contractstatemir.ExitReject {
		status, committed = 1, 0
	} else if exit == contractstatemir.ExitTrap {
		status, committed = 2, 0
		trapCheck = "if (sizeof(fycs_native_trap_category_v0) != sizeof(\"overflow\") || memcmp(fycs_native_trap_category_v0, \"overflow\", sizeof(\"overflow\")) != 0) { return 35; }"
	}
	storeCheck := fmt.Sprintf("if (store.present != 1u || store.length != 8u || memcmp(store.value, %d != 0 ? written : original, 8u) != 0) { return 32; }", committed)
	readCheck := "if (fycs_native_last_read_found_v0 != 1u || fycs_native_last_read_length_v0 != 8u || memcmp(fycs_native_last_read_v0, written, 8u) != 0) { return 33; }"
	if deleting {
		readCheck = "if (fycs_native_last_read_found_v0 != 0u || fycs_native_last_read_length_v0 != 0u) { return 33; }"
		if committed != 0 {
			storeCheck = "if (store.present != 0u || store.length != 0u) { return 32; }"
		}
	}
	return fmt.Sprintf(`
#include <string.h>
int main(void) {
    fycs_native_store_v0 store;
    const uint8_t original[8] = {0,0,0,0,0,0,0,7};
    const uint8_t written[8] = {0,0,0,0,0,0,0,42};
    (void)written;
    fycs_native_store_initialize_v0(&store, fycs_native_layout_hash_v0, fycs_native_body_hash_v0);
    memcpy(store.value, original, sizeof(original));
    store.length = sizeof(original);
    store.present = 1u;
    for (int repeat = 0; repeat < 2; ++repeat) {
    if (fycs_native_entry_v0(&store) != %d) { return 30; }
    %s
    %s
    }
    %s
    store.body_hash[0] ^= 0xffu;
    store.length = 8u;
    store.present = 1u;
    if (fycs_native_entry_v0(&store) != -10) { return 34; }
    store.body_hash[0] ^= 0xffu;
    for (size_t length = 0; length <= FYCS_NATIVE_VALUE_CAPACITY_V0; ++length) {
        if (length == 8u) { continue; }
        store.length = length;
        if (fycs_native_entry_v0(&store) != -10) { return 38; }
        if (store.length != length) { return 39; }
    }
    store.length = FYCS_NATIVE_VALUE_CAPACITY_V0 + 1u;
    if (fycs_native_entry_v0(&store) != -10) { return 37; }
    if (fycs_native_live_resources_v0() != 0u) { return 36; }
    return 42;
}
`, status, storeCheck, readCheck, trapCheck)
}

package contractstatenative

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
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
		input, coordinate := nativeStateInput(t, exit)
		handoff, err := Prepare(input, 1, 1, coordinate)
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := Emit(handoff, target.X8664WindowsMSVC)
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			built, err := cdriver.BuildWindowsHost(toolchain, profile, artifact.Source+nativeProbe(exit), runtimeSource)
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

func nativeProbe(exit contractstatemir.ExitKind) string {
	status := 0
	committed := 1
	trapCheck := ""
	if exit == contractstatemir.ExitReject {
		status, committed = 1, 0
	} else if exit == contractstatemir.ExitTrap {
		status, committed = 2, 0
		trapCheck = "if (fycs_native_trap_category_v0[0] != 'o' || fycs_native_trap_category_v0[8] != 0u) { return 35; }"
	}
	return fmt.Sprintf(`
int main(void) {
    fycs_native_store_v0 store;
    fycs_native_store_initialize_v0(&store, fycs_native_layout_hash_v0, fycs_native_body_hash_v0);
    if (fycs_native_entry_v0(&store) != %d) { return 30; }
    if (store.present != %du) { return 31; }
    if (%d != 0 && (store.length != 8u || store.value[7] != 42u)) { return 32; }
    if (fycs_native_last_read_found_v0 != 1u || fycs_native_last_read_length_v0 != 8u || fycs_native_last_read_v0[7] != 42u) { return 33; }
    %s
    store.body_hash[0] ^= 0xffu;
    if (fycs_native_entry_v0(&store) != -10) { return 34; }
    store.body_hash[0] ^= 0xffu;
    store.length = FYCS_NATIVE_VALUE_CAPACITY_V0 + 1u;
    if (fycs_native_entry_v0(&store) != -10) { return 37; }
    if (fycs_native_live_resources_v0() != 0u) { return 36; }
    return 42;
}
`, status, committed, committed, trapCheck)
}

package cstdlib

import (
	"strings"
	"testing"
)

func TestContractStateRuntimeIsStackOnlyAndOwnsClosedEntryPoints(t *testing.T) {
	source, err := ContractStateRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range []string{
		"fycs_native_store_initialize_v0", "fycs_native_begin_v0", "fycs_native_cell_read_v0",
		"fycs_native_cell_write_v0", "fycs_native_cell_delete_v0", "fycs_native_commit_v0",
		"fycs_native_rollback_v0", "fycs_native_live_resources_v0",
	} {
		if !strings.Contains(source, symbol) {
			t.Fatalf("contract-state runtime is missing %s", symbol)
		}
	}
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "free("} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("contract-state runtime contains dynamic allocation %q", forbidden)
		}
	}
}

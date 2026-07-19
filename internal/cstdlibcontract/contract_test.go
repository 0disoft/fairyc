package cstdlibcontract

import (
	"slices"
	"testing"
)

func TestComponentsForSymbolsClosesDependenciesInDeterministicOrder(t *testing.T) {
	t.Parallel()
	components, err := ComponentsForSymbols([]string{
		"fairyc_external_helper",
		"fy_std_string_append_v0",
		"fy_std_linux_file_read_v0",
		"fy_std_string_append_v0",
		"fy_std_arena_initialize_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []Component{
		ComponentLinuxFile,
		ComponentSystemHeap,
		ComponentBuffer,
		ComponentArena,
		ComponentTextView,
		ComponentOwnedText,
	}
	if !slices.Equal(components, want) {
		t.Fatalf("components = %v, want %v", components, want)
	}
}

func TestComponentsForSymbolsIgnoresNonStandardForeignSymbols(t *testing.T) {
	t.Parallel()
	components, err := ComponentsForSymbols([]string{"fairyc_external_helper"})
	if err != nil || len(components) != 0 {
		t.Fatalf("ComponentsForSymbols() = %v, %v", components, err)
	}
}

func TestComponentsForSymbolsRejectsUnknownStandardLibraryNamespaceMember(t *testing.T) {
	t.Parallel()
	if _, err := ComponentsForSymbols([]string{"fy_std_missing_v0"}); err == nil {
		t.Fatal("ComponentsForSymbols() accepted an unknown fy_std_ symbol")
	}
}

func TestAdmittedKeepsUnexercisedLinuxFileSymbolsClosed(t *testing.T) {
	t.Parallel()
	if !Admitted("fy_std_linux_file_close_v0") {
		t.Fatal("close symbol is not admitted")
	}
	for _, symbol := range []string{"fy_std_linux_file_write_v0", "fy_std_linux_file_drop_v0"} {
		if Admitted(symbol) {
			t.Fatalf("%s unexpectedly admitted", symbol)
		}
	}
}

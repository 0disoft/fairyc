package main

import (
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestReprCRecordFieldNativeCrossing(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Skipf("MSVC unavailable: %v", err)
	}
	executions, err := runFairyStructResultProbe(toolchain)
	if err != nil {
		t.Fatal(err)
	}
	if executions != 4 {
		t.Fatalf("record argument/result probe ran %d combinations, want 4", executions)
	}
	t.Logf("MSVC %s: C11/C17, low/optimized array, C enum, floating/integer records and bounded callbacks passed (36 C-enum-array cases, 45 C-scalar-array cases, 5 floating cases, 4 integer cases, 4 integer-array cases, 6 callback invocations per combination)", toolchain.Version)
}

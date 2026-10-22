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
	t.Logf("MSVC %s: C11/C17, low/optimized integer array and C enum record crossings passed", toolchain.Version)
}

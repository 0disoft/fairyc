package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestWindowsMixedStackBoundaries(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := runWindowsMixedStackProbe(toolchain)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 10 {
		t.Fatalf("got %d executions, want 10", len(rows))
	}
	for _, row := range rows {
		t.Logf("%s/%s: exit=%d trap=%d runtimeFunctions=%d", row.Case, row.Profile, row.Exit, row.TrapCode, row.RuntimeFunctions)
	}
	if path := os.Getenv("FAIRYC_MIXED_STACK_EVIDENCE"); path != "" {
		data, err := json.MarshalIndent(struct {
			Passed bool                    `json:"passed"`
			Rows   []mixedStackObservation `json:"rows"`
		}{true, rows}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

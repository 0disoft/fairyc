package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestWindowsChainedUnwind(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := runWindowsChainedUnwindProbe(toolchain)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d profiles", len(rows))
	}
	for _, row := range rows {
		t.Logf("%s: functions=%d untested=%d contexts=%d depth=%d integer=%d vector=%d rejected=%d", row.Profile, row.Functions, row.UntestedFunctions, row.Contexts, row.MaximumDepth, row.IntegerRestores, row.VectorRestores, row.RejectedMutations)
	}
	if path := os.Getenv("FAIRYC_CHAIN_EVIDENCE"); path != "" {
		data, err := json.MarshalIndent(struct {
			Passed bool                       `json:"passed"`
			Mode   string                     `json:"mode"`
			Rows   []chainedUnwindObservation `json:"rows"`
		}{true, "metadata-derived-controlled-contexts-at-linked-CRT-chain-prologs", rows}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

package main

import (
	"encoding/json"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"testing"
)

func TestWindowsVersion2Epilogs(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := runWindowsVersion2EpilogProbe(toolchain)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d profiles", len(rows))
	}
	for _, row := range rows {
		t.Logf("%s: functions=%d epilogs=%d contexts=%d pops=%d explicit=%d rejected=%d", row.Profile, row.Functions, row.Epilogs, row.Contexts, row.Pops, row.ExplicitOffsets, row.RejectedMutations)
	}
	if path := os.Getenv("FAIRYC_V2_EPILOG_EVIDENCE"); path != "" {
		data, err := json.MarshalIndent(struct {
			Passed bool                        `json:"passed"`
			Mode   string                      `json:"mode"`
			Rows   []version2EpilogObservation `json:"rows"`
		}{true, "controlled-contexts-at-linked-CRT-epilog-instructions", rows}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

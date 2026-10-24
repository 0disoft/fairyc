package main

import (
	"encoding/json"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"testing"
)

func TestWindowsChainedLink(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := runWindowsChainedLinkProbe(toolchain)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("profiles=%d", len(rows))
	}
	for _, row := range rows {
		t.Logf("%s: links=%d fields=%d contexts=%d rejected=%d", row.Profile, len(row.Link.Records), row.Link.RelocationFields, row.Contexts, row.RejectedMutations)
	}
	if path := os.Getenv("FAIRYC_CHAIN_LINK_EVIDENCE"); path != "" {
		data, err := json.MarshalIndent(struct {
			Passed bool                     `json:"passed"`
			Mode   string                   `json:"mode"`
			Rows   []chainedLinkObservation `json:"rows"`
		}{true, "closed-COFF-fixture-chain-relocations-and-native-unwind", rows}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

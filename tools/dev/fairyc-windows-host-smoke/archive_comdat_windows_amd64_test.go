//go:build windows && amd64

package main

import (
	"github.com/0disoft/fairyc/internal/cdriver"
	"testing"
)

func TestWindowsArchiveCOMDATForms(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	count, err := runWindowsArchiveProbe(tc)
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("archive executions=%d", count)
	}
	t.Logf("MSVC %s: four archive/direct input-order executions with indexed selection, ANY competition, retained and discarded associative unwind proofs", tc.Version)
}

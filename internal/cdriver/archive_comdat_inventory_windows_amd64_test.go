//go:build windows && amd64

package cdriver_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
)

func recordGeneratedArchiveCOMDAT(t *testing.T, a cdriver.WindowsHostArtifact) {
	t.Helper()
	first, second := a.Objects[0].Bytes, a.Objects[1].Bytes
	parents, err := pecoff.InspectGeneratedCOMDATSelection(first, second, a.Bytes, a.LinkMap, a.LinkerVerbose)
	if err != nil {
		t.Fatal(err)
	}
	children, err := pecoff.InspectGeneratedUnwindSelection(first, second, a.Bytes, a.LinkMap, a.LinkerVerbose)
	if err != nil {
		t.Fatal(err)
	}
	inventory := map[uint8]int{}
	for _, raw := range [][]byte{first, second} {
		r, err := pecoff.InspectObject(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range r.Sections {
			if s.COMDATSelection != 0 && s.COMDATSelection != pecoff.ObjectCOMDATNoDuplicates && s.COMDATSelection != pecoff.ObjectCOMDATAny && s.COMDATSelection != pecoff.ObjectCOMDATAssociative {
				t.Fatal("new generated COMDAT mode needs review")
			}
			if s.COMDATSelection != 0 {
				inventory[s.COMDATSelection]++
			}
		}
	}
	t.Logf("production COMDAT modes=%v parents=%d retained=%d discarded=%d children=%d retained=%d discarded=%d fields=%d pdata=%d xdata=%d", inventory, len(parents.Parents), parents.Retained, parents.Discarded, len(children.Children), children.Retained, children.Discarded, children.RelocationFields, children.MatchedPDataEntries, children.MatchedXDataBytes)
}

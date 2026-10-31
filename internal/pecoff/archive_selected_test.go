package pecoff

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestIndexedArchiveObject(t *testing.T) {
	base := syntheticCOFFArchive()
	r, err := InspectIndexedArchiveObject(base, "winner")
	if err != nil || r.Name != "winner.obj" {
		t.Fatalf("%+v %v", r, err)
	}
	unused, err := InspectIndexedArchiveObject(base, "unused")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "absent"} {
		if _, err := InspectIndexedArchiveObject(base, name); err == nil {
			t.Fatal("accepted missing demand")
		}
	}
	first := 8 + coffArchiveHeaderBytes
	second := nextSyntheticArchiveMemberOffset(8, syntheticFirstLinkerSize()) + coffArchiveHeaderBytes
	tests := []func([]byte){func(b []byte) { b[0] = '?' }, func(b []byte) { binary.BigEndian.PutUint32(b[first+4:], unused.HeaderOffset) }, func(b []byte) { binary.LittleEndian.PutUint32(b[second+4:], 1) }, func(b []byte) { copy(b[int(r.DataOffset)+testCOFFSymbolOffset:], "absent") }}
	for _, mutate := range tests {
		b := bytes.Clone(base)
		mutate(b)
		if _, err := InspectIndexedArchiveObject(b, "winner"); err == nil {
			t.Fatal("accepted corrupt selected archive route")
		}
	}
	// Unselected bodies are deliberately not represented as structurally verified.
	b := bytes.Clone(base)
	b[unused.DataOffset] = 0
	if _, err := InspectIndexedArchiveObject(b, "winner"); err != nil {
		t.Fatal(err)
	}
}

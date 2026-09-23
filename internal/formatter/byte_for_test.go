package formatter

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestFormatByteViewForPreservesShapeAndName(t *testing.T) {
	input := []byte("module main;  \r\nfn void show(ref u8[] view) {  for (let u8 byte in view) { assert(byte == 7); } }\r\n")
	first, err := Format("main.fy", input, source.DefaultLimits())
	if err != nil || first.Malformed {
		t.Fatalf("first format = %+v, %v", first, err)
	}
	if !bytes.Contains(first.Bytes, []byte("let u8 byte in view")) {
		t.Fatalf("formatted header = %q", first.Bytes)
	}
	second, err := Format("main.fy", first.Bytes, source.DefaultLimits())
	if err != nil || !bytes.Equal(first.Bytes, second.Bytes) || first.Tree.ShapeIdentity() != second.Tree.ShapeIdentity() {
		t.Fatalf("second format changed shape or bytes: %v, %q, %q", err, first.Bytes, second.Bytes)
	}
}

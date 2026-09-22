package formatter

import (
	"bytes"
	"github.com/0disoft/fairyc/internal/source"
	"testing"
)

func TestArgumentLabelsFormatting(t *testing.T) {
	input := []byte("module main; fn i32 f(i32 value) { return value; } fn i32 main() { return f(.value /* keep */ = 42); }\r\n")
	first, err := Format("main.fy", input, source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Format("main.fy", first.Bytes, source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes, second.Bytes) || !bytes.Contains(first.Bytes, []byte(".value /* keep */ = 42")) {
		t.Fatalf("labels changed: %s", first.Bytes)
	}
}

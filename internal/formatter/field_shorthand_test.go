package formatter

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestFieldShorthandFormatting(t *testing.T) {
	input := []byte("module main;\r\nfn i32 main() { let Pair p = Pair { .left /* keep */, .right }; return 0; }  \r\n")
	first, err := Format("main.fy", input, source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Format("main.fy", first.Bytes, source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes, second.Bytes) || !bytes.Contains(first.Bytes, []byte(".left /* keep */, .right")) {
		t.Fatalf("shorthand/comments changed: %s", first.Bytes)
	}
}

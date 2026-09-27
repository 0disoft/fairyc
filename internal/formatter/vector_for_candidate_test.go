package formatter

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestFormatPrivateVectorForPreservesBorrowedItemAndOwner(t *testing.T) {
	for _, sample := range []struct {
		name   string
		source string
		header string
	}{
		{
			name:   "explicit owner borrow",
			source: "module main;\r\nimport std::vec as v;\r\nfn void show() {\r\n    var v::Vec<i32> values = v::empty<i32>();\r\n    for (let ref i32 item in ref values) { assert(*item == 0); }\r\n}\r\n",
			header: "let ref i32 item in ref values",
		},
		{
			name:   "named shared view",
			source: "module main;\r\nimport std::vec as v;\r\nfn void show(ref v::Vec<i32> values) {\r\n    for (let ref i32 item in values) { assert(*item == 0); }\r\n}\r\n",
			header: "let ref i32 item in values",
		},
	} {
		t.Run(sample.name, func(t *testing.T) {
			first, err := Format("main.fy", []byte(sample.source), source.DefaultLimits())
			if err != nil || first.Malformed {
				t.Fatalf("first format = %+v, %v", first, err)
			}
			if !bytes.Contains(first.Bytes, []byte(sample.header)) || bytes.Contains(first.Bytes, []byte("\r")) {
				t.Fatalf("formatted source changed vector loop: %q", first.Bytes)
			}
			second, err := Format("main.fy", first.Bytes, source.DefaultLimits())
			if err != nil || second.Malformed || !bytes.Equal(first.Bytes, second.Bytes) || first.Tree.ShapeIdentity() != second.Tree.ShapeIdentity() {
				t.Fatalf("second format changed shape or bytes: %v, %q, %q", err, first.Bytes, second.Bytes)
			}
		})
	}
}

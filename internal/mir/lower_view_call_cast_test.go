package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestLowerDataAddressFromReturnedView(t *testing.T) {
	for _, tc := range []struct {
		name, view, input, pointer string
		kind                       OperationKind
	}{
		{"bytes", "ref u8[]", `b"abc"`, "ptr u8", OperationByteSliceDataAddress},
		{"mutable-bytes", "ref mut u8[]", "ref mut data[..]", "ptr mut u8", OperationByteSliceDataAddress},
		{"text", "ref str", `"abc"`, "ptr u8", OperationStrDataAddress},
		{"cstring", "ref CStr", `c"abc"`, "ptr u8", OperationCStrDataAddress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := lowerTestSource(t, fmt.Sprintf(`module main;
fn %s pass(%s value) { return value; }
fn i32 main() {
    var u8[3] data = [1, 2, 3];
    let %s address = unsafe { pass(%s) as %s };
    return 0;
}`, tc.view, tc.view, tc.pointer, tc.input, tc.pointer))
			var calls, addresses int
			for _, f := range program.Functions {
				for _, b := range f.Blocks {
					if b.Terminator.Kind == TerminatorCall {
						calls++
					}
					for _, op := range b.Operations {
						if op.Kind == tc.kind {
							addresses++
							if len(op.Operands) != 1 || op.Operands[0].Kind != OperandLocal || op.UnsafeScope.Source != "main.fy" || op.Operator != "as" {
								t.Fatalf("returned-view address = %+v", op)
							}
							_, mutable, pointer := hir.RawPointerElement(op.Type)
							if !pointer || mutable != (tc.name == "mutable-bytes") {
								t.Fatalf("address pointer type = %q", op.Type)
							}
						}
					}
				}
			}
			if calls != 1 || addresses != 1 {
				t.Fatalf("calls=%d addresses=%d; view producer must execute once", calls, addresses)
			}
			if failures := Verify(program); len(failures) != 0 {
				t.Fatalf("returned-view MIR: %+v", failures)
			}
		})
	}
}

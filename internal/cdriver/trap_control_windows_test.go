//go:build windows

package cdriver

import "testing"

func TestBoundedTrapControlDrainsAfterOverflow(t *testing.T) {
	t.Parallel()
	control := &boundedTrapControl{}
	input := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	written, err := control.Write(input)
	if err != nil || written != len(input) || len(control.bytes) != maxTrapControlBytes || !control.overflow {
		t.Fatalf("first write = (%d, %v), bytes %v, overflow %t", written, err, control.bytes, control.overflow)
	}
	written, err = control.Write([]byte{11, 12})
	if err != nil || written != 2 || len(control.bytes) != maxTrapControlBytes || !control.overflow {
		t.Fatalf("drain write = (%d, %v), bytes %v, overflow %t", written, err, control.bytes, control.overflow)
	}
}

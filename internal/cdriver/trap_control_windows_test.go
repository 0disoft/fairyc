//go:build windows

package cdriver

import (
	"bytes"
	"testing"
)

func TestBoundedTrapControlDrainsAfterOverflow(t *testing.T) {
	t.Parallel()
	control := &boundedTrapControl{}
	input := make([]byte, maxTrapControlBytes+2)
	for index := range input {
		input[index] = byte(index % 251)
	}
	written, err := control.Write(input)
	if err != nil || written != len(input) || !bytes.Equal(control.bytes, input[:maxTrapControlBytes]) || !control.overflow {
		t.Fatalf("first write = (%d, %v), bytes %v, overflow %t", written, err, control.bytes, control.overflow)
	}
	written, err = control.Write([]byte{11, 12})
	if err != nil || written != 2 || !bytes.Equal(control.bytes, input[:maxTrapControlBytes]) || !control.overflow {
		t.Fatalf("drain write = (%d, %v), bytes %v, overflow %t", written, err, control.bytes, control.overflow)
	}
}

package cdriver

import (
	"debug/elf"
	"testing"
	"time"
)

func TestLinuxProjectAdmission(t *testing.T) {
	for _, timeout := range []time.Duration{0, -1, 10*time.Minute + 1} {
		if _, err := BuildLinuxTargetParts("must-not-open", "", LowProfile, "header", "literals", []string{"body"}, "runtime", timeout); err == nil {
			t.Fatal("invalid deadline admitted")
		}
	}
	for _, data := range [][]byte{nil, []byte("MZ"), make([]byte, 64)} {
		if err := inspectLinuxProjectELF(data, elf.ET_EXEC, elf.ET_DYN); err == nil {
			t.Fatal("non-ELF image admitted")
		}
	}
}

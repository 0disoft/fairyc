package cdriver

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func TestProjectDataOnlyFallbackPreservesNameBudget(t *testing.T) {
	for _, count := range []int{4, 1024} {
		const symbolOffset = 20 + 40
		const nameSize = 64 << 10
		stringOffset := symbolOffset + count*18
		object := make([]byte, stringOffset+4+nameSize+1)
		binary.LittleEndian.PutUint16(object, 0x8664)
		binary.LittleEndian.PutUint16(object[2:], 1)
		binary.LittleEndian.PutUint32(object[8:], symbolOffset)
		binary.LittleEndian.PutUint32(object[12:], uint32(count))
		copy(object[20:], ".bss")
		binary.LittleEndian.PutUint32(object[20+36:], 0xc0000080)
		for index := range count {
			binary.LittleEndian.PutUint32(object[symbolOffset+index*18+4:], 4)
			object[symbolOffset+index*18+16] = 2
		}
		binary.LittleEndian.PutUint32(object[stringOffset:], nameSize+5)
		for index := 0; index < nameSize; index++ {
			object[stringOffset+4+index] = 'n'
		}
		err := inspectProjectObject(object)
		if count == 4 && err != nil {
			t.Fatalf("valid data-only fallback rejected: %v", err)
		}
		if count == 1024 && !errors.Is(err, pecoff.ErrNameBudgetExceeded) {
			t.Fatalf("data-only fallback bypassed name budget: %v", err)
		}
		// A different earlier failure must not bypass the fallback preflight.
		binary.LittleEndian.PutUint32(object[12:], (1<<18)+1)
		if err := inspectProjectObject(object); err == nil || !strings.Contains(err.Error(), "invalid symbol count") {
			t.Fatalf("earlier failure bypassed fallback preflight: %v", err)
		}
	}
}

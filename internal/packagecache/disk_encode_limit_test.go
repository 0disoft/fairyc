package packagecache

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestDiskEncodePreservesExactByteLimit(t *testing.T) {
	for _, size := range []int{1, 2, 3, 4, 1024} {
		t.Run(fmt.Sprintf("payload_%d", size), func(t *testing.T) {
			record := diskOwnershipRecord()
			for index := range record.Artifacts {
				record.Artifacts[index].Data = bytes.Repeat([]byte{'x'}, size)
			}
			encoded, err := encodeDiskRecord(record, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			exact, err := encodeDiskRecord(record, uint64(len(encoded)))
			if err != nil || !bytes.Equal(exact, encoded) {
				t.Fatalf("exact encoded limit changed canonical bytes: %v", err)
			}
			tooSmall, err := encodeDiskRecord(record, uint64(len(encoded)-1))
			if tooSmall != nil || !errors.Is(err, ErrRecordTooLarge) {
				t.Fatalf("one-byte undersize: got %d bytes, %v", len(tooSmall), err)
			}
		})
	}
}

func TestDiskEncodeBoundsCombinedBase64Payloads(t *testing.T) {
	record := diskOwnershipRecord()
	for index := range record.Artifacts {
		record.Artifacts[index].Data = bytes.Repeat([]byte{'x'}, 600)
	}
	// Each 600-byte payload expands to 800 bytes; the pair exceeds 1 KiB.
	encoded, err := encodeDiskRecord(record, 1024)
	if encoded != nil || !errors.Is(err, ErrRecordTooLarge) {
		t.Fatalf("combined payload limit: got %d bytes, %v", len(encoded), err)
	}
}

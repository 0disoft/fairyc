package contractstatemir

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestStableStateStringCountPhysicalBoundaries(t *testing.T) {
	for _, count := range []uint64{0, 1, 2, 3, ^uint64(0)} {
		data := make([]byte, 8+16) // space for exactly two empty string prefixes
		binary.BigEndian.PutUint64(data, count)
		reader := bytes.NewReader(data)
		values, err := readStableStateStrings(reader)
		if count <= 2 {
			if err != nil || len(values) != int(count) || reader.Len() != 16-int(count)*8 {
				t.Fatalf("count=%d values=%v remaining=%d err=%v", count, values, reader.Len(), err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "string count exceeds its bytes") || values != nil || reader.Len() != 16 {
			t.Fatalf("impossible count=%d was not rejected before item reads: remaining=%d err=%v", count, reader.Len(), err)
		}
	}
	for _, remaining := range []int{0, 7, 8, 9} {
		data := make([]byte, 8+remaining)
		binary.BigEndian.PutUint64(data, 1)
		_, err := readStableStateStrings(bytes.NewReader(data))
		if (err != nil) != (remaining < 8) {
			t.Fatalf("remaining=%d err=%v", remaining, err)
		}
	}
}

func TestStableStateStringCountRejectsAmplification(t *testing.T) {
	data := make([]byte, 8+(1<<20))
	binary.BigEndian.PutUint64(data, 1<<20)
	reader := bytes.NewReader(data)
	if values, err := readStableStateStrings(reader); err == nil || values != nil || reader.Len() != 1<<20 {
		t.Fatalf("count bomb reached allocation/item decoding: values=%d remaining=%d err=%v", len(values), reader.Len(), err)
	}
}

package atlasmap

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDecodeMapTrailingDataAcrossVersions(t *testing.T) {
	partition := decodeBenchmarkPartition(t, 2)
	for _, version := range []struct {
		name, logical string
		encode        func(Partition, uint64) ([]byte, error)
		decode        func([]byte, uint64) (Partition, error)
	}{
		{"v1", LogicalSchema, Encode, Decode},
		{"v2", LogicalSchemaV2, EncodeV2, DecodeV2},
		{"v3", LogicalSchemaV3, EncodeV3, DecodeV3},
	} {
		t.Run(version.name, func(t *testing.T) {
			partition.data.header.LogicalSchema = version.logical
			canonical, err := version.encode(partition, testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			for _, whitespace := range []string{" ", "\t", "\r", "\n", " \t\r\n"} {
				if _, err := version.decode(append(bytes.Clone(canonical), whitespace...), testMapLimit); !errors.Is(err, ErrNonCanonical) {
					t.Fatalf("noncanonical whitespace %q: %v", whitespace, err)
				}
			}
			for _, tail := range []string{"null", "false", "0", `"string"`, "{}", "[]", "[", "garbage", "\x00", "\v", "\f", "\u00a0"} {
				for _, whitespace := range []string{"", " \t\r\n"} {
					data := append(bytes.Clone(canonical), whitespace+tail...)
					if got, err := version.decode(data, testMapLimit); err == nil || !strings.Contains(err.Error(), "trailing") || got.data.declarations != nil {
						t.Fatalf("tail %q returned records or wrong error: %v", whitespace+tail, err)
					}
				}
			}
			data := append(bytes.Clone(canonical), '[')
			data = append(data, bytes.Repeat([]byte("0,"), (1<<19)-1)...)
			data = append(data, '0', ']')
			rejectionAllocations := func(input []byte) float64 {
				return testing.AllocsPerRun(1, func() {
					if _, err := version.decode(input, testMapLimit); err == nil || !strings.Contains(err.Error(), "trailing") {
						t.Fatalf("trailing array: %v", err)
					}
				})
			}
			small := rejectionAllocations(append(bytes.Clone(canonical), "[]"...))
			large := rejectionAllocations(data)
			// Normal envelope allocations remain; rejected array nodes must not grow with its size.
			if large > small+16 {
				t.Fatalf("array rejection allocations grew from %.0f to %.0f", small, large)
			}
		})
	}
}

func BenchmarkDecodeMapTrailingArray(b *testing.B) {
	partition := decodeBenchmarkPartition(b, 2)
	partition.data.header.LogicalSchema = LogicalSchemaV3
	canonical, err := EncodeV3(partition, testMapLimit)
	if err != nil {
		b.Fatal(err)
	}
	for _, elements := range []int{1 << 14, 1 << 19} {
		b.Run(fmt.Sprint(elements), func(b *testing.B) {
			data := append(bytes.Clone(canonical), '[')
			data = append(data, bytes.Repeat([]byte("0,"), elements-1)...)
			data = append(data, '0', ']')
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := DecodeV3(data, uint64(len(data))); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
					b.Fatalf("trailing array: %v", err)
				}
			}
		})
	}
}

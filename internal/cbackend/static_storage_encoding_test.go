package cbackend

import (
	"fmt"
	"strings"
	"testing"
)

func TestStaticStorageDecimalEncoding(t *testing.T) {
	allBytes := make([]byte, 256)
	for i := range allBytes {
		allBytes[i] = byte(i)
	}
	for _, emptySentinel := range []bool{false, true} {
		t.Run(fmt.Sprint(emptySentinel), func(t *testing.T) {
			storage := map[string]string{"z_bytes": strings.Repeat(string(allBytes), 3), "b_utf8": "Café 🧚", "c_zero": "\x00"}
			order := []string{"b_utf8", "c_zero", "z_bytes"}
			if emptySentinel {
				storage["a_empty"] = ""
				order = append([]string{"a_empty"}, order...)
			}
			var got, want strings.Builder
			got.WriteString("prefix\n")
			want.WriteString("prefix\n")
			emitStaticStorage(&got, storage, emptySentinel)
			for _, symbol := range order {
				value := storage[symbol]
				length := len(value)
				if emptySentinel && length == 0 {
					length = 1
				}
				fmt.Fprintf(&want, "static const uint8_t %s[%d] = { ", symbol, length)
				if len(value) == 0 {
					want.WriteString("UINT8_C(0)")
				}
				for i := range len(value) {
					if i != 0 {
						want.WriteString(", ")
					}
					fmt.Fprintf(&want, "UINT8_C(%d)", value[i])
				}
				want.WriteString(" };\n")
			}
			want.WriteByte('\n')
			if got.String() != want.String() {
				t.Fatal("static storage decimal bytes, ordering or empty sentinel changed")
			}
		})
	}
}

func BenchmarkEmitStaticStorage(b *testing.B) {
	for _, size := range []int{4096, 1 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i)
			}
			storage := map[string]string{"fy_static_bytes_test": string(data)}
			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var output strings.Builder
				emitStaticStorage(&output, storage, true)
				if output.Len() < size {
					b.Fatal("static storage output truncated")
				}
			}
		})
	}
}

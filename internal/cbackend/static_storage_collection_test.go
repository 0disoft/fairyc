package cbackend

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestStaticStorageSymbolChunkBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, 16, 31, 32, 33, 4095, 4096, 4097, 65536} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}
		want := fmt.Sprintf("fy_test_%x", sha256.Sum256(data))
		if got := staticStorageSymbol("fy_test_", string(data)); got != want {
			t.Fatalf("size=%d: got %q, want %q", size, got, want)
		}
	}
}

func TestCollectedStoragePreservesContentAndRejectsWrongTypes(t *testing.T) {
	for _, test := range []struct {
		name    string
		kind    mir.OperandKind
		typ     hir.Type
		value   string
		symbol  func(string) string
		collect func([]mir.Function) (map[string]string, error)
	}{
		{"bytes", mir.OperandByteString, hir.SharedBorrow(hir.TypeByteSlice), strings.Repeat("\x00\xff", 4097), byteStringStorageSymbol, collectByteStringStorage},
		{"utf8", mir.OperandString, hir.SharedBorrow(hir.TypeStr), strings.Repeat("Café 🧚", 1024), stringStorageSymbol, collectStringStorage},
		{"cstr", mir.OperandCString, hir.SharedBorrow(hir.TypeCStr), strings.Repeat("x", 4097) + "\x00", cStringStorageSymbol, collectCStringStorage},
	} {
		t.Run(test.name, func(t *testing.T) {
			operand := mir.Operand{Kind: test.kind, Type: test.typ, StaticData: test.value}
			functions := []mir.Function{{Name: "storage", Blocks: []mir.Block{{
				Operations: []mir.Operation{{Operands: []mir.Operand{operand, operand}}},
				Terminator: mir.Terminator{Arguments: []mir.Operand{operand}, SwitchOperands: []mir.Operand{operand}, ReturnValue: operand},
			}}}}
			storage, err := test.collect(functions)
			if err != nil || len(storage) != 1 || storage[test.symbol(test.value)] != test.value {
				t.Fatalf("static storage contents or deduplication changed: entries=%d err=%v", len(storage), err)
			}
			functions[0].Blocks[0].Terminator.ReturnValue.Type = hir.TypeI32
			if _, err := test.collect(functions); err == nil {
				t.Fatal("incorrect type accepted after a valid duplicate literal")
			}
		})
	}
}

func BenchmarkStaticStorageSymbol(b *testing.B) {
	for _, size := range []int{16, 65536} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			value := strings.Repeat("a", size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if len(staticStorageSymbol("fy_test_", value)) != len("fy_test_")+64 {
					b.Fatal("invalid symbol")
				}
			}
		})
	}
}

func BenchmarkCollectRepeatedStringStorage(b *testing.B) {
	for _, size := range []int{4096, 65536} {
		for _, unique := range []bool{false, true} {
			b.Run(fmt.Sprintf("bytes=%d/unique=%t", size, unique), func(b *testing.B) {
				value := strings.Repeat("a", size)
				operands := make([]mir.Operand, 256)
				for i := range operands {
					// Repeated contents need not share backing storage after decoding.
					data := strings.Clone(value)
					if unique {
						data = fmt.Sprintf("%04d", i) + value[4:]
					}
					operands[i] = mir.Operand{Kind: mir.OperandString, Type: hir.SharedBorrow(hir.TypeStr), StaticData: data}
				}
				functions := []mir.Function{{Name: "storage", Blocks: []mir.Block{{Operations: []mir.Operation{{Operands: operands}}}}}}
				want := 1
				if unique {
					want = len(operands)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					storage, err := collectStringStorage(functions)
					if err != nil || len(storage) != want {
						b.Fatalf("storage=%d err=%v", len(storage), err)
					}
				}
			})
		}
	}
}

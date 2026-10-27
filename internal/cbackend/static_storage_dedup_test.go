package cbackend

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestStaticStorageValidatesEachContentOncePerCollection(t *testing.T) {
	for _, test := range []struct {
		name    string
		kind    mir.OperandKind
		typ     hir.Type
		valid   func(string) bool
		content string
		invalid string
	}{
		{"utf8", mir.OperandString, hir.SharedBorrow(hir.TypeStr), utf8.ValidString, "Café 🧚", "\xff"},
		{"cstr", mir.OperandCString, hir.SharedBorrow(hir.TypeCStr), validCStringStorage, "value\x00", "bad\x00tail\x00"},
	} {
		t.Run(test.name, func(t *testing.T) {
			checks := 0
			spec := staticStorageSpec{
				operandKind: test.kind, valueType: test.typ, label: test.name, symbolPrefix: "fy_test_",
				valid: func(value string) bool { checks++; return test.valid(value) },
			}
			operand := mir.Operand{Kind: test.kind, Type: test.typ, StaticData: test.content}
			functions := []mir.Function{{Name: "first", Blocks: []mir.Block{{
				Operations: []mir.Operation{{Operands: []mir.Operand{operand}}},
				Terminator: mir.Terminator{Arguments: []mir.Operand{operand}, SwitchOperands: []mir.Operand{operand}, ReturnValue: operand},
			}}}, {Name: "second", Blocks: []mir.Block{{Operations: []mir.Operation{{Operands: []mir.Operand{
				{Kind: test.kind, Type: test.typ, StaticData: strings.Clone(test.content)},
			}}}}}}}
			for pass := 1; pass <= 2; pass++ {
				storage, err := collectStaticStorage(functions, spec)
				if err != nil || len(storage) != 1 || checks != pass {
					t.Fatalf("pass %d: entries=%d checks=%d err=%v", pass, len(storage), checks, err)
				}
				if storage[staticStorageSymbol(spec.symbolPrefix, test.content)] != test.content {
					t.Fatal("content-addressed storage changed")
				}
			}
			// A new invalid value must still be checked after repeated valid data.
			functions[1].Blocks[0].Operations[0].Operands[0].StaticData = test.invalid
			if _, err := collectStaticStorage(functions, spec); err == nil {
				t.Fatal("invalid content admitted after a valid repeated literal")
			}
		})
	}
}

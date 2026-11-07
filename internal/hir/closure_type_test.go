package hir

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClosureValueLeafCanonicalIdentityAndTransport(t *testing.T) {
	if TypeClosureI32.String() != "fn i32(i32)" || !IsClosureI32(TypeClosureI32) || ParseType("fn i32(i32)") != TypeClosureI32 || TypeClosureI32.Descriptor().Kind != TypeKindBuiltin || TypeClosureI32.Descriptor().Name != "fn i32(i32)" {
		t.Fatal("closure canonical leaf identity lost")
	}
	if TypeClosureI32 == TypeCCallbackI32UserData || IsCCallback(TypeClosureI32) {
		t.Fatal("closure aliases C callback")
	}
	if _, _, ok := CallbackSignature(TypeClosureI32); ok {
		t.Fatal("safe closure decomposed as foreign callback")
	}
	raw, err := json.Marshal(TypeClosureI32)
	if err != nil || string(raw) != `"fn i32(i32)"` {
		t.Fatalf("closure JSON changed: %s %v", raw, err)
	}
	var restored Type
	if err := json.Unmarshal(raw, &restored); err != nil || restored != TypeClosureI32 {
		t.Fatalf("closure JSON round trip: %v", err)
	}
	text, err := TypeClosureI32.MarshalText()
	if err != nil || string(text) != "fn i32(i32)" {
		t.Fatal("closure text transport changed")
	}
	restored = Type{}
	if err := restored.UnmarshalText(text); err != nil || restored != TypeClosureI32 {
		t.Fatal("closure text round trip lost identity")
	}
	var chunks strings.Builder
	if !TypeClosureI32.VisitStringChunks(func(s string) bool { chunks.WriteString(s); return true }) || chunks.String() != string(text) {
		t.Fatal("closure chunk spelling differs")
	}
	values := map[Type]int{TypeClosureI32: 1, TypeCCallbackI32UserData: 2}
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[Type]int
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded[TypeClosureI32] != 1 || decoded[TypeCCallbackI32UserData] != 2 {
		t.Fatal("closure/callback map keys lost distinct identities")
	}
}

func TestClosureValueLeafDoesNotNormalizeOtherSignatures(t *testing.T) {
	for _, text := range []string{"fn bool(i32)", "fn i32()", "fn i32(i32, i32)", "fn i32(i32,)", "fn i32( i32)", "fn  i32(i32)", "fn i32(i32) ", "fn [i32 x] i32(i32)", "fn i32(i32", "unsafe extern C fn i32(i32)", "unsafe fn i32(i32)", "fn i32(i32, *mut void)"} {
		value := ParseType(text)
		if IsClosureI32(value) || value.String() != text || value.Descriptor().Kind != TypeKindOpaque {
			t.Fatalf("other/malformed signature was normalized: %q (%v)", text, value.Descriptor())
		}
		encoded, _ := json.Marshal(value)
		var restored Type
		if err := json.Unmarshal(encoded, &restored); err != nil || restored != value {
			t.Fatalf("malformed spelling round trip %q: %v", text, err)
		}
	}
	for _, value := range []Type{TypeI32, TypeInvalid, TypeCCallbackI32UserData, Option(TypeClosureI32), Result(TypeClosureI32, TypeU32), SharedBorrow(TypeClosureI32), RawPointer(TypeClosureI32, false), FixedArray(TypeClosureI32, 2)} {
		if IsClosureI32(value) {
			t.Fatal("exact closure helper admitted another type or wrapper")
		}
	}
	if ValidCallbackSignature(TypeClosureI32, []Type{RawPointer(TypeVoid, true)}) || ValidCallbackSignature(TypeI32, []Type{TypeClosureI32, RawPointer(TypeVoid, true)}) {
		t.Fatal("closure gained callback FFI admission")
	}
}

func TestClosureValueLeafPreservesExistingTypeBytes(t *testing.T) {
	for _, tc := range []struct {
		value Type
		text  string
	}{
		{TypeI32, "i32"}, {TypeBool, "bool"}, {Option(TypeI32), "Option<i32>"}, {Result(TypeI32, TypeBool), "Result<i32, bool>"}, {SharedBorrow(TypeI32), "&i32"},
		{TypeCCallbackI32UserData, "unsafe extern C fn i32(i32, *mut void)"},
		{CCallback(TypeCInt, []Type{TypeCUInt, RawPointer(TypeCInt, false)}), "unsafe extern C fn c::int(c::uint, *c::int)"},
	} {
		want, _ := json.Marshal(tc.text)
		got, err := json.Marshal(tc.value)
		if err != nil || string(got) != string(want) || ParseType(tc.text) != tc.value {
			t.Fatalf("existing type bytes changed: %q -> %s (%v)", tc.text, got, err)
		}
	}
	if !IsCCallback(TypeCCallbackI32UserData) || CCallback(TypeI32, []Type{TypeI32, RawPointer(TypeVoid, true)}) != TypeCCallbackI32UserData {
		t.Fatal("legacy callback identity changed")
	}
}

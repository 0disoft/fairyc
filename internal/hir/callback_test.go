package hir

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestOrdinaryCallbackTypeIdentity(t *testing.T) {
	data := RawPointer(TypeCInt, false)
	types := append(CScalarTypes(), TypeU8, TypeI32, TypeI64, TypeU32, TypeUSize, TypeI8, TypeI16, TypeU16, TypeU64, TypeISize, TypeF32, TypeF64, RawPointer(TypeVoid, true), RawPointer(TypeCInt, false), RawPointer(RawPointer(TypeCInt, true), true))
	for _, typ := range types {
		signature := CCallback(typ, []Type{typ, data})
		result, params, ok := CallbackSignature(signature)
		if !ok || result != typ || !slices.Equal(params, []Type{typ, data}) {
			t.Fatal("signature lost", typ)
		}
		b, err := json.Marshal(signature)
		if err != nil {
			t.Fatal(err)
		}
		var restored Type
		if err = json.Unmarshal(b, &restored); err != nil || restored != signature {
			t.Fatal("identity lost", signature, err)
		}
		if !ValidBoundedCallbackParameters([]Type{signature, data}) || ValidBoundedCallbackParameters([]Type{signature, RawPointer(TypeCInt, true)}) {
			t.Fatal("context pointer mismatch lost")
		}
	}
	if CCallback(TypeI32, []Type{TypeI32, RawPointer(TypeVoid, true)}) != TypeCCallbackI32UserData {
		t.Fatal("legacy identity changed")
	}
	if CCallback(TypeVoid, []Type{data}) == TypeInvalid {
		t.Fatal("zero value arguments rejected")
	}
	for _, text := range []string{"unsafe extern C fn i32(i32)", "unsafe extern C fn bool(*mut void)", "unsafe extern C fn c::int(c::int,*mut void)", "unsafe extern C fn i32(" + strings.Repeat("i32, ", 9) + "*mut void)"} {
		if IsCCallback(ParseType(text)) {
			t.Fatal("malformed callback admitted", text)
		}
	}
}

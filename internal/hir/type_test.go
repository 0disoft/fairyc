package hir

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestTypeIDsShareStructuredIdentity(t *testing.T) {
	token := ParseType("$p7::main::Token")
	value := Result(Option(MutableBorrow(Slice(token))), FixedArray(RawPointer(TypeCInt, true), 3))
	text := "Result<Option<&mut [$p7::main::Token]>, [*mut c::int; 3]>"
	if value.String() != text || value != ParseType(text) {
		t.Fatalf("type identity mismatch: %s", value)
	}
	d := value.Descriptor()
	if d.Kind != TypeKindResult || d.Element != Option(MutableBorrow(Slice(token))) || d.Failure != FixedArray(RawPointer(TypeCInt, true), 3) {
		t.Fatalf("descriptor: %+v", d)
	}
	d.Element = TypeBool
	if value.Descriptor().Element == TypeBool {
		t.Fatal("descriptor mutation escaped")
	}
	if MutableBorrow(token) == SharedBorrow(token) || RawPointer(token, true) == RawPointer(token, false) || FixedArray(token, 3) == FixedArray(token, 4) {
		t.Fatal("distinct shapes collapsed")
	}
	if TypeByteSlice != Slice(TypeU8) || ParseType("[u8]") != TypeByteSlice {
		t.Fatal("builtin slice identity mismatch")
	}
	if (Type{}).String() != "" || ParseType("") != (Type{}) || TypeInvalid == (Type{}) {
		t.Fatal("empty and invalid identities collapsed")
	}
}

func TestTypeIDsAreConcurrentAndComparable(t *testing.T) {
	want := Option(Result(TypeI32, MutableBorrow(TypeCStr)))
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 100 {
				got := Option(Result(TypeI32, MutableBorrow(TypeCStr)))
				if got != want || map[Type]bool{want: true}[got] != true {
					t.Error("interning lost identity")
				}
			}
		})
	}
	wg.Wait()
}

func TestTypeTransportPreservesSpellings(t *testing.T) {
	for _, text := range []string{"", "<invalid>", "i32", "main::Token", "&mut [i32; 42]", "Result<Option<i32>, main::Error>", "Result< i32,bool >", "Option<i32, bool>", "[i32; nope]", "[i32; 0]", strings.Repeat("Option<", 1024) + "i32" + strings.Repeat(">", 1024)} {
		value := ParseType(text)
		if value.String() != text {
			t.Fatalf("rewrote %q", text)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := json.Marshal(text)
		if string(encoded) != string(want) {
			t.Fatalf("transport changed: %s", encoded)
		}
		var decoded Type
		if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != value {
			t.Fatalf("round trip %q: %v", text, err)
		}
	}
	original := map[Type]int{Result(TypeI32, TypeBool): 7}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[Type]int
	if err := json.Unmarshal(data, &decoded); err != nil || decoded[Result(TypeI32, TypeBool)] != 7 {
		t.Fatalf("type map round trip %s: %v", data, err)
	}
	value := TypeI32
	if err := json.Unmarshal([]byte("null"), &value); err != nil || value != TypeI32 {
		t.Fatalf("null changed identity: %s %v", value, err)
	}
}

func TestParsedTypeUsesStructuredChildren(t *testing.T) {
	value := ParseType("Result< [i32; 3],Option<bool> >")
	success, failure, ok := ResultElements(value)
	if !ok || success != FixedArray(TypeI32, 3) || failure != Option(TypeBool) {
		t.Fatalf("parsed children: %s %s %v", success, failure, ok)
	}
	for _, text := range []string{"Result<i32, bool, u8>", "Result<[i32>, bool]>", "Option<i32, bool>", "Option<i32>>"} {
		if ParseType(text).Descriptor().Kind != TypeKindOpaque {
			t.Fatalf("malformed type admitted: %s", text)
		}
	}
}

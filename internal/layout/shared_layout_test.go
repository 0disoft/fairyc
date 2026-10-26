package layout

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

// Backend layout descriptors share each nested aggregate's field slice.
func sharedLayout(depth int, kind Kind) Type {
	value := Type{Kind: KindLanguageScalar, Name: "i32"}
	for range depth {
		value = Type{Kind: kind, Fields: []Field{{Name: "left", Type: value}, {Name: "right", Type: value}}}
	}
	return value
}

func TestSharedAggregateLayout(t *testing.T) {
	engine := testEngine(t)
	for _, kind := range []Kind{KindStruct, KindUnion} {
		value := sharedLayout(12, kind)
		result, err := engine.Of(value)
		wantSize := uint64(4)
		if kind == KindStruct {
			wantSize <<= 12
		}
		if err != nil || result.SizeBytes != wantSize || result.AlignmentBytes != 4 || len(result.Fields) != 2 {
			t.Fatalf("%s shared layout = %+v, %v; want size %d, alignment 4 and two fields", kind, result, err, wantSize)
		}
	}
}

func TestSharedLayoutStillChecksDeeperOccurrences(t *testing.T) {
	engine := testEngine(t)
	shared := sharedLayout(1, KindStruct)
	deep := shared
	for range maxLayoutDepth - 2 {
		child := deep
		deep = Type{Kind: KindArray, Element: &child, Length: 1}
	}
	root := Type{Kind: KindStruct, Fields: []Field{{Name: "shallow", Type: shared}, {Name: "deep", Type: deep}}}
	if _, err := engine.Of(root); errorCode(t, err) != ErrorDepthLimit {
		t.Fatalf("shared subtree bypassed depth limit: %v", err)
	}
}

func TestSharedLayoutDistinguishesMetadata(t *testing.T) {
	engine := testEngine(t)
	structure := sharedLayout(1, KindStruct)
	scalar := Type{Kind: KindLanguageScalar, Name: "i32"}
	array := Type{Kind: KindArray, Element: &scalar, Length: 1}
	for _, test := range []struct {
		name string
		base Type
		edit func(*Type)
		size uint64
		code ErrorCode
	}{
		{"field slice length", structure, func(v *Type) { v.Fields = v.Fields[:1] }, 12, ""},
		{"aggregate kind", structure, func(v *Type) { v.Kind = KindUnion }, 12, ""},
		{"array length", array, func(v *Type) { v.Length = 2 }, 12, ""},
		{"invalid array name", array, func(v *Type) { v.Name = "invalid" }, 0, ErrorInvalidType},
		{"invalid aggregate element", structure, func(v *Type) { v.Element = &scalar }, 0, ErrorInvalidType},
		{"invalid aggregate length", structure, func(v *Type) { v.Length = 1 }, 0, ErrorInvalidType},
	} {
		t.Run(test.name, func(t *testing.T) {
			other := test.base
			test.edit(&other)
			root := Type{Kind: KindStruct, Fields: []Field{{Name: "first", Type: test.base}, {Name: "second", Type: other}}}
			result, err := engine.Of(root)
			if test.code != "" {
				if errorCode(t, err) != test.code {
					t.Fatalf("Of() = %v, want %s", err, test.code)
				}
			} else if err != nil || result.SizeBytes != test.size {
				t.Fatalf("Of() = %+v, %v, want size %d", result, err, test.size)
			}
		})
	}
}

func TestSharedLayoutDoesNotRetainMutableInputsOrResults(t *testing.T) {
	engine := testEngine(t)
	value := sharedLayout(2, KindStruct)
	first, err := engine.Of(value)
	if err != nil {
		t.Fatal(err)
	}
	first.Fields[0].SizeBytes = 999
	if result, err := engine.Of(value); err != nil || result.Fields[0].SizeBytes != 8 {
		t.Fatalf("returned fields leaked between calls: %+v, %v", result, err)
	}
	value.Fields[0].Type.Fields[0].Type.Name = "i64"
	if result, err := engine.Of(value); err != nil || result.SizeBytes != 32 {
		t.Fatalf("mutated input reused stale layout: %+v, %v", result, err)
	}
	if _, err := engine.Of(sharedLayout(64, KindStruct)); errorCode(t, err) != ErrorOverflow {
		t.Fatalf("shared aggregate overflow was not rejected: %v", err)
	}
}

func BenchmarkSharedAggregateLayout(b *testing.B) {
	spec, _ := target.Lookup(target.X8664LinuxGNU)
	engine, err := New(spec)
	if err != nil {
		b.Fatal(err)
	}
	for _, depth := range []int{1, 12, 18} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			value := sharedLayout(depth, KindStruct)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				result, err := engine.Of(value)
				if err != nil || result.SizeBytes != uint64(4)<<depth {
					b.Fatalf("Of() = %+v, %v", result, err)
				}
			}
		})
	}
}

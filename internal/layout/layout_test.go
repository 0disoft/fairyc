package layout

import (
	"math"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestStructLayoutIncludesInternalAndTailPadding(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	result, err := engine.Of(Type{Kind: KindStruct, Name: "Padded", Fields: []Field{
		{Name: "small", Type: Type{Kind: KindLanguageScalar, Name: "u8"}},
		{Name: "wide", Type: Type{Kind: KindLanguageScalar, Name: "i32"}},
		{Name: "tail", Type: Type{Kind: KindLanguageScalar, Name: "u16"}},
	}})
	if err != nil {
		t.Fatalf("Of() error = %v", err)
	}
	wantFields := []FieldResult{
		{Name: "small", OffsetBytes: 0, SizeBytes: 1, AlignmentBytes: 1},
		{Name: "wide", OffsetBytes: 4, SizeBytes: 4, AlignmentBytes: 4},
		{Name: "tail", OffsetBytes: 8, SizeBytes: 2, AlignmentBytes: 2},
	}
	wantPadding := []Padding{{OffsetBytes: 1, SizeBytes: 3}, {OffsetBytes: 10, SizeBytes: 2}}
	if result.SizeBytes != 12 || result.AlignmentBytes != 4 || !slices.Equal(result.Fields, wantFields) || !slices.Equal(result.Padding, wantPadding) {
		t.Fatalf("Of() = %+v", result)
	}
}

func TestArrayLayoutRejectsOverflowAndZeroLength(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	element := Type{Kind: KindLanguageScalar, Name: "i32"}
	for _, test := range []struct {
		name   string
		length uint64
		code   ErrorCode
	}{
		{name: "zero", length: 0, code: ErrorInvalidArrayLength},
		{name: "overflow", length: math.MaxUint64, code: ErrorOverflow},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := engine.Of(Type{Kind: KindArray, Element: &element, Length: test.length})
			code, ok := Code(err)
			if !ok || code != test.code {
				t.Fatalf("Of() error = %v, code = %q", err, code)
			}
		})
	}
}

func TestUnionLayoutUsesLargestAlignedField(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	result, err := engine.Of(Type{Kind: KindUnion, Fields: []Field{
		{Name: "small", Type: Type{Kind: KindLanguageScalar, Name: "u8"}},
		{Name: "pair", Type: Type{Kind: KindStruct, Fields: []Field{
			{Name: "left", Type: Type{Kind: KindLanguageScalar, Name: "i32"}},
			{Name: "right", Type: Type{Kind: KindLanguageScalar, Name: "i32"}},
		}}},
	}})
	if err != nil {
		t.Fatalf("Of() error = %v", err)
	}
	wantFields := []FieldResult{
		{Name: "small", OffsetBytes: 0, SizeBytes: 1, AlignmentBytes: 1},
		{Name: "pair", OffsetBytes: 0, SizeBytes: 8, AlignmentBytes: 4},
	}
	if result.SizeBytes != 8 || result.AlignmentBytes != 4 || !slices.Equal(result.Fields, wantFields) || len(result.Padding) != 0 {
		t.Fatalf("Of() = %+v", result)
	}
}

func TestLayoutRejectsUnknownScalarAndDuplicateField(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	if _, err := engine.Of(Type{Kind: KindLanguageScalar, Name: "i128"}); errorCode(t, err) != ErrorUnknownScalar {
		t.Fatalf("unknown scalar error = %v", err)
	}
	_, err := engine.Of(Type{Kind: KindStruct, Fields: []Field{
		{Name: "same", Type: Type{Kind: KindLanguageScalar, Name: "i32"}},
		{Name: "same", Type: Type{Kind: KindLanguageScalar, Name: "i32"}},
	}})
	if errorCode(t, err) != ErrorDuplicateField {
		t.Fatalf("duplicate field error = %v", err)
	}
}

func testEngine(t *testing.T) *Engine {
	t.Helper()
	spec, found := target.Lookup(target.X8664LinuxGNU)
	if !found {
		t.Fatal("target is missing")
	}
	engine, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if engine.Target() != target.X8664LinuxGNU {
		t.Fatalf("engine target = %q", engine.Target())
	}
	return engine
}

func errorCode(t *testing.T, err error) ErrorCode {
	t.Helper()
	code, ok := Code(err)
	if !ok {
		t.Fatalf("error %v has no layout code", err)
	}
	return code
}

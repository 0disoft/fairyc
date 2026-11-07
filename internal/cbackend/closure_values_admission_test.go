package cbackend

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestClosureValuePrivateLayoutBoundaries(t *testing.T) {
	engine := taggedCandidateEngine(t)
	box := mir.StructType{Type: hir.ParseType("main::Box"), Name: "Box", Fields: []mir.StructField{{Name: "action", Type: hir.TypeClosureI32}}}
	structures, byType, _, _, err := prepareAggregateTypes([]mir.StructType{box}, nil, engine)
	if err != nil || len(structures) != 1 {
		t.Fatal("ordinary closure field", err)
	}
	computed, err := engine.Of(closureValueLayout())
	if err != nil || structures[0].layout.SizeBytes != computed.SizeBytes || structures[0].layout.AlignmentBytes != computed.AlignmentBytes {
		t.Fatal("private closure struct layout", err)
	}
	analysis := emissionAnalysis{structTypes: structures, structByType: byType}
	if !analysis.supportedCValueType(hir.TypeClosureI32) {
		t.Fatal("closure leaf transport missing")
	}
	for _, value := range []hir.Type{hir.TypeClosureI32, box.Type, hir.RawPointer(hir.TypeClosureI32, false), hir.RawPointer(box.Type, true), hir.Option(hir.TypeClosureI32), hir.Result(hir.TypeClosureI32, hir.TypeU32)} {
		for _, result := range []bool{false, true} {
			if _, err := analysis.foreignCType(value, result); err == nil {
				t.Fatalf("closure exposed through FFI: %s", value)
			}
		}
	}
	for _, value := range []hir.Type{hir.SharedBorrow(hir.TypeClosureI32), hir.MutableBorrow(hir.TypeClosureI32), hir.Option(hir.TypeClosureI32), hir.Result(hir.TypeClosureI32, hir.TypeU32)} {
		if analysis.supportedCValueType(value) {
			t.Fatalf("closure wrapper opened: %s", value)
		}
	}
	for _, value := range []hir.Type{hir.Option(box.Type), hir.Result(box.Type, hir.TypeI32), hir.Option(hir.SharedBorrow(box.Type)), hir.Result(hir.TypeI32, hir.Option(box.Type))} {
		tagged, _, err := prepareTaggedTypes([]mir.Function{{ParameterTypes: []hir.Type{value}}}, structures, byType, nil, nil, nil, nil, nil, engine)
		if err != nil || len(tagged) != 0 {
			t.Fatalf("closure struct wrapper opened: %s %+v %v", value, tagged, err)
		}
	}
	if _, err := analysis.cValueType(hir.SharedBorrow(box.Type)); err == nil {
		t.Fatal("closure struct gained pointer transport")
	}
	if _, _, err := prepareScalarArrayTypes([]mir.Function{{ParameterTypes: []hir.Type{hir.FixedArray(box.Type, 2)}}}, structures); err == nil {
		t.Fatal("closure struct array opened")
	}
	box.ReprC = true
	if _, _, _, _, err := prepareAggregateTypes([]mir.StructType{box}, nil, engine); err == nil {
		t.Fatal("closure gained repr(C)")
	}
	box.ReprC = false
	outer := mir.StructType{Type: hir.ParseType("main::Outer"), Name: "Outer", Fields: []mir.StructField{{Name: "box", Type: box.Type}}}
	if _, _, _, _, err := prepareAggregateTypes([]mir.StructType{box, outer}, nil, engine); err == nil {
		t.Fatal("nested closure struct opened")
	}
}

package cbackend

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestCompositeResultCandidateBoundaries(t *testing.T) {
	for _, scalar := range []hir.Type{hir.TypeI8, hir.TypeI16, hir.TypeISize, hir.TypeU16, hir.TypeU64, hir.TypeF32, hir.TypeF64} {
		t.Run(scalar.String(), func(t *testing.T) {
			flat := hir.Result(scalar, hir.TypeU32)
			blocked := []hir.Type{hir.Result(scalar, hir.TypeI32), hir.Result(hir.TypeU32, scalar), hir.Result(scalar, scalar), hir.Result(hir.Option(scalar), hir.TypeU32), hir.Option(flat), hir.Result(flat, hir.TypeU32), hir.Result(hir.TypeU32, flat), hir.Result(hir.SharedBorrow(scalar), hir.TypeU32)}
			for _, first := range []bool{true, false} {
				params := append([]hir.Type{}, blocked...)
				if first {
					params = append([]hir.Type{flat}, params...)
				} else {
					params = append(params, flat)
				}
				got, byType, err := prepareTaggedTypes([]mir.Function{{ParameterTypes: params}}, nil, nil, nil, nil, nil, nil, nil, taggedCandidateEngine(t))
				if err != nil || len(got) != 1 || len(byType) != 1 || got[0].typ != flat || got[0].kind != taggedResult || got[0].needsDrop {
					t.Fatalf("root-only Result candidates: %+v %v", got, err)
				}
				expected := layout.Type{Kind: layout.KindStruct, Fields: []layout.Field{
					{Name: "tag", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u32"}},
					{Name: "p_0", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: scalar.String()}},
					{Name: "p_1", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u32"}},
				}}
				want, err := taggedCandidateEngine(t).Of(expected)
				if err != nil || got[0].layout.SizeBytes != want.SizeBytes || got[0].layout.AlignmentBytes != want.AlignmentBytes || len(got[0].layout.Fields) != 3 {
					t.Fatalf("Result layout: %+v want %+v: %v", got[0].layout, want, err)
				}
				for i := range want.Fields {
					if got[0].layout.Fields[i].OffsetBytes != want.Fields[i].OffsetBytes || got[0].layout.Fields[i].SizeBytes != want.Fields[i].SizeBytes {
						t.Fatalf("payload %d layout: %+v want %+v", i, got[0].layout, want)
					}
				}
			}
			for _, value := range []hir.Type{scalar, flat} {
				if supportedTaggedPayload(value, true, nil, nil, nil, nil) {
					t.Fatalf("shared payload admission widened: %s", value)
				}
				if _, ok := copyScalarTaggedPayloadLayout(value); ok {
					t.Fatalf("shared payload layout widened: %s", value)
				}
				if _, ok := copyScalarTaggedAggregateLayout(value); ok {
					t.Fatalf("aggregate layout widened: %s", value)
				}
				analysis := emissionAnalysis{}
				for _, result := range []bool{false, true} {
					if _, err := analysis.foreignCType(value, result); err == nil {
						t.Fatalf("new FFI admission: %s", value)
					}
				}
			}
			analysis := emissionAnalysis{}
			for _, value := range []hir.Type{hir.RawPointer(flat, false), hir.RawPointer(flat, true), hir.SharedBorrow(flat), hir.MutableBorrow(flat)} {
				for _, result := range []bool{false, true} {
					if _, err := analysis.foreignCType(value, result); err == nil {
						t.Fatalf("Result pointer/reference FFI admission widened: %s", value)
					}
				}
			}
		})
	}
}

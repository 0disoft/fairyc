package cbackend

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestCompositeOptionCandidateBoundaries(t *testing.T) {
	for _, scalar := range []hir.Type{hir.TypeI8, hir.TypeI16, hir.TypeISize, hir.TypeU16, hir.TypeU64, hir.TypeF32, hir.TypeF64} {
		t.Run(scalar.String(), func(t *testing.T) {
			flat := hir.Option(scalar)
			blocked := []hir.Type{hir.Option(hir.Option(flat)), hir.Result(scalar, hir.TypeI64), hir.Result(hir.TypeI32, scalar), hir.Result(flat, hir.TypeI32), hir.Option(hir.Result(scalar, hir.TypeI32)), hir.Option(hir.SharedBorrow(scalar))}
			// Include the allowed sibling first: rejected roots must not inherit
			// permission from a previously retained candidate.
			params := append([]hir.Type{flat}, blocked...)
			got, byType, err := prepareTaggedTypes([]mir.Function{{ParameterTypes: params}}, nil, nil, nil, nil, nil, nil, nil, taggedCandidateEngine(t))
			if err != nil || len(got) != 1 || len(byType) != 1 || got[0].typ != flat || got[0].needsDrop {
				t.Fatalf("flat Option candidates: %+v %v", got, err)
			}
			if len(got[0].layout.Fields) != 2 {
				t.Fatal("Option lost tag/payload layout")
			}
			payload, err := taggedCandidateEngine(t).Of(layout.Type{Kind: layout.KindLanguageScalar, Name: scalar.String()})
			if err != nil || got[0].layout.Fields[1].SizeBytes != payload.SizeBytes {
				t.Fatalf("payload layout: %+v %v", got[0].layout, err)
			}
			if supportedTaggedPayload(scalar, true, nil, nil, nil, nil) {
				t.Fatal("shared Result payload admission widened")
			}
			if _, ok := copyScalarTaggedPayloadLayout(scalar); ok {
				t.Fatal("shared aggregate payload layout widened")
			}
			if _, ok := copyScalarTaggedAggregateLayout(flat); ok {
				t.Fatal("aggregate layout admission widened")
			}
			analysis := emissionAnalysis{}
			for _, typ := range []hir.Type{scalar, flat} {
				for _, result := range []bool{false, true} {
					if _, err := analysis.foreignCType(typ, result); err == nil {
						t.Fatalf("new FFI admission %s", typ)
					}
				}
			}
		})
	}
}

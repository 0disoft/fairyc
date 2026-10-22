package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func floatingScalarsForTarget(spec target.Spec) map[hir.Type]target.Scalar {
	result := make(map[hir.Type]target.Scalar, 2)
	for _, scalar := range spec.LanguageScalars {
		typ := hir.ParseType(scalar.Name)
		if hir.IsFloatScalar(typ) {
			result[typ] = scalar
		}
	}
	return result
}

func (e *emitter) emitFloatingTransportAssertions(output *strings.Builder) error {
	for _, typ := range []hir.Type{hir.TypeF32, hir.TypeF64} {
		if _, used := e.analysis.features["fairy-"+typ.String()+"-transport"]; !used {
			continue
		}
		scalar, ok := e.analysis.floatingScalars[typ]
		if !ok {
			return CapabilityError{Feature: fmt.Sprintf("floating type %q on target %q", typ, e.analysis.targetID)}
		}
		cType, prefix, mantissa, minimum, maximum := "float", "FLT", 24, -125, 128
		if typ == hir.TypeF64 {
			cType, prefix, mantissa, minimum, maximum = "double", "DBL", 53, -1021, 1024
		}
		fmt.Fprintf(output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc target %s size\");\n", cType, scalar.Layout.SizeBytes, typ)
		fmt.Fprintf(output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc target %s alignment\");\n", cType, scalar.Layout.AlignmentBytes, typ)
		fmt.Fprintf(output, "_Static_assert(FLT_RADIX == 2 && %s_MANT_DIG == %d && %s_MIN_EXP == %d && %s_MAX_EXP == %d, \"fairyc %s binary format\");\n\n", prefix, mantissa, prefix, minimum, prefix, maximum, typ)
	}
	return nil
}

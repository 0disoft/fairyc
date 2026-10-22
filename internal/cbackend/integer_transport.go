package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func integerTransportCType(value hir.Type) (string, bool) {
	switch value {
	case hir.TypeI8:
		return "int8_t", true
	case hir.TypeI16:
		return "int16_t", true
	case hir.TypeU16:
		return "uint16_t", true
	case hir.TypeU64:
		return "uint64_t", true
	case hir.TypeISize:
		return "intptr_t", true
	}
	return "", false
}

func (e *emitter) emitIntegerTransportAssertions(output *strings.Builder) error {
	spec, ok := target.Lookup(e.analysis.targetID)
	if !ok {
		return CapabilityError{Feature: fmt.Sprintf("integer transport target %q", e.analysis.targetID)}
	}
	for _, scalar := range spec.LanguageScalars {
		typ := hir.ParseType(scalar.Name)
		cType, supported := integerTransportCType(typ)
		if !supported {
			continue
		}
		if _, used := e.analysis.features["fairy-"+typ.String()+"-transport"]; !used {
			continue
		}
		fmt.Fprintf(output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc target %s size\");\n", cType, scalar.Layout.SizeBytes, typ)
		fmt.Fprintf(output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc target %s alignment\");\n", cType, scalar.Layout.AlignmentBytes, typ)
		comparison := "<"
		if scalar.Signedness == "unsigned" {
			comparison = ">"
		}
		fmt.Fprintf(output, "_Static_assert(((%s)-1) %s (%s)0, \"fairyc target %s signedness\");\n\n", cType, comparison, cType, typ)
	}
	return nil
}

package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

type cScalarTransport struct {
	sourceType   hir.Type
	targetName   string
	cType        string
	feature      string
	header       string
	availability string
}

var cScalarTransports = []cScalarTransport{
	{sourceType: hir.TypeCBool, targetName: "c.bool", cType: "_Bool", feature: "c-bool"},
	{sourceType: hir.TypeCChar, targetName: "c.char", cType: "char", feature: "c-char"},
	{sourceType: hir.TypeCDouble, targetName: "c.double", cType: "double", feature: "c-double"},
	{sourceType: hir.TypeCFloat, targetName: "c.float", cType: "float", feature: "c-float"},
	{sourceType: hir.TypeCInt, targetName: "c.int", cType: "int", feature: "c-int"},
	{sourceType: hir.TypeCLong, targetName: "c.long", cType: "long", feature: "c-long"},
	{sourceType: hir.TypeCLongLong, targetName: "c.longlong", cType: "long long", feature: "c-longlong"},
	{sourceType: hir.TypeCSChar, targetName: "c.schar", cType: "signed char", feature: "c-schar"},
	{sourceType: hir.TypeCShort, targetName: "c.short", cType: "short", feature: "c-short"},
	{sourceType: hir.TypeCSizeT, targetName: "c.size_t", cType: "size_t", feature: "c-size-t", header: "stddef.h", availability: "iso-c11"},
	{sourceType: hir.TypeCSSizeT, targetName: "c.ssize_t", cType: "ssize_t", feature: "c-ssize-t", header: "sys/types.h", availability: "posix.1-2008"},
	{sourceType: hir.TypeCUChar, targetName: "c.uchar", cType: "unsigned char", feature: "c-uchar"},
	{sourceType: hir.TypeCUInt, targetName: "c.uint", cType: "unsigned int", feature: "c-uint"},
	{sourceType: hir.TypeCULong, targetName: "c.ulong", cType: "unsigned long", feature: "c-ulong"},
	{sourceType: hir.TypeCULongLong, targetName: "c.ulonglong", cType: "unsigned long long", feature: "c-ulonglong"},
	{sourceType: hir.TypeCUShort, targetName: "c.ushort", cType: "unsigned short", feature: "c-ushort"},
}

func cScalarTransportForType(value hir.Type) (cScalarTransport, bool) {
	for _, transport := range cScalarTransports {
		if transport.sourceType == value {
			return transport, true
		}
	}
	return cScalarTransport{}, false
}

func validateCScalarTransports() error {
	expected := hir.CScalarTypes()
	if len(cScalarTransports) != len(expected) {
		return fmt.Errorf("C scalar transport table has %d entries, want %d", len(cScalarTransports), len(expected))
	}
	seenTarget := make(map[string]struct{}, len(cScalarTransports))
	seenFeature := make(map[string]struct{}, len(cScalarTransports))
	for index, transport := range cScalarTransports {
		if transport.sourceType != expected[index] || transport.targetName == "" || transport.cType == "" || transport.feature == "" {
			return fmt.Errorf("C scalar transport entry %d is incomplete or out of canonical order", index)
		}
		if _, duplicate := seenTarget[transport.targetName]; duplicate {
			return fmt.Errorf("C scalar transport target %q is duplicated", transport.targetName)
		}
		if _, duplicate := seenFeature[transport.feature]; duplicate {
			return fmt.Errorf("C scalar transport feature %q is duplicated", transport.feature)
		}
		seenTarget[transport.targetName] = struct{}{}
		seenFeature[transport.feature] = struct{}{}
	}
	return nil
}

func targetCScalar(spec target.Spec, name string) (target.Scalar, bool) {
	for _, scalar := range spec.CScalars {
		if scalar.Name == name {
			return scalar, true
		}
	}
	return target.Scalar{}, false
}

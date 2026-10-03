package cbackend

import (
	"strings"

	"github.com/0disoft/fairyc/internal/cstdlibcontract"
	"github.com/0disoft/fairyc/internal/target"
)

type cIdentifierScope uint8

const (
	cIdentifierFileScope cIdentifierScope = iota
	cIdentifierMemberScope
)

func validCABIRecordIdentifier(value string) bool {
	return validCIdentifier(value, cIdentifierFileScope) && !reservedBackendCRecordName(value)
}

func validCABIMemberIdentifier(value string) bool {
	return validCIdentifier(value, cIdentifierMemberScope)
}

func validExternalCIdentifier(value string, targetID target.ID) bool {
	if value == "main" || !validCIdentifier(value, cIdentifierFileScope) {
		return false
	}
	if strings.HasPrefix(value, "fy_") && !cstdlibcontract.AdmittedForTarget(value, targetID) {
		return false
	}
	return true
}

func validCIdentifier(value string, scope cIdentifierScope) bool {
	if value == "" || reservedCImplementationIdentifier(value, scope) || strings.HasPrefix(value, "FY_") {
		return false
	}
	if _, reserved := generatedCNames[value]; reserved {
		return false
	}
	if _, reserved := cHeaderMacroNames[value]; reserved {
		return false
	}
	for index, character := range value {
		if character > 0x7f || !(character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || index > 0 && character >= '0' && character <= '9') {
			return false
		}
	}
	_, keyword := c11Keywords[value]
	return !keyword
}

func reservedCImplementationIdentifier(value string, scope cIdentifierScope) bool {
	if !strings.HasPrefix(value, "_") {
		return false
	}
	if scope == cIdentifierFileScope || strings.HasPrefix(value, "__") {
		return true
	}
	return len(value) > 1 && value[1] >= 'A' && value[1] <= 'Z'
}

func reservedBackendCRecordName(value string) bool {
	switch value {
	case "fy_region", "fy_byte_slice", "fy_mut_byte_slice", "fy_i64_slice", "fy_mut_i64_slice", "fy_i32_slice", "fy_mut_i32_slice", "fy_u32_slice", "fy_mut_u32_slice", "fy_usize_slice", "fy_mut_usize_slice", "fy_struct_slice", "fy_str":
		return true
	default:
		return false
	}
}

var generatedCNames = map[string]struct{}{
	"uint8_t": {}, "int32_t": {}, "uint32_t": {}, "uint64_t": {}, "uintptr_t": {}, "size_t": {}, "ssize_t": {},
	"UINT8_C": {}, "INT32_C": {}, "UINT32_C": {}, "UINT64_C": {},
	"INT32_MIN": {}, "INT32_MAX": {}, "UINT8_MAX": {}, "UINT32_MAX": {}, "UINTPTR_MAX": {},
}

var cHeaderMacroNames = map[string]struct{}{
	"NULL": {}, "offsetof": {},
	"INT8_C": {}, "INT16_C": {}, "INT64_C": {},
	"UINT16_C": {}, "UINT32_C": {}, "UINT64_C": {},
	"INT8_MIN": {}, "INT16_MIN": {}, "INT64_MIN": {},
	"INT8_MAX": {}, "INT16_MAX": {}, "INT64_MAX": {},
	"UINT16_MAX": {}, "UINT64_MAX": {},
	"INT_LEAST8_MIN": {}, "INT_LEAST16_MIN": {}, "INT_LEAST32_MIN": {}, "INT_LEAST64_MIN": {},
	"INT_LEAST8_MAX": {}, "INT_LEAST16_MAX": {}, "INT_LEAST32_MAX": {}, "INT_LEAST64_MAX": {},
	"UINT_LEAST8_MAX": {}, "UINT_LEAST16_MAX": {}, "UINT_LEAST32_MAX": {}, "UINT_LEAST64_MAX": {},
	"INT_FAST8_MIN": {}, "INT_FAST16_MIN": {}, "INT_FAST32_MIN": {}, "INT_FAST64_MIN": {},
	"INT_FAST8_MAX": {}, "INT_FAST16_MAX": {}, "INT_FAST32_MAX": {}, "INT_FAST64_MAX": {},
	"UINT_FAST8_MAX": {}, "UINT_FAST16_MAX": {}, "UINT_FAST32_MAX": {}, "UINT_FAST64_MAX": {},
	"INTPTR_MIN": {}, "INTPTR_MAX": {}, "UINTPTR_MAX": {},
	"INTMAX_MIN": {}, "INTMAX_MAX": {}, "UINTMAX_MAX": {},
	"PTRDIFF_MIN": {}, "PTRDIFF_MAX": {}, "SIG_ATOMIC_MIN": {}, "SIG_ATOMIC_MAX": {},
	"SIZE_MAX": {}, "WCHAR_MIN": {}, "WCHAR_MAX": {}, "WINT_MIN": {}, "WINT_MAX": {},
}

var c11Keywords = map[string]struct{}{
	"auto": {}, "break": {}, "case": {}, "char": {}, "const": {}, "continue": {}, "default": {},
	"do": {}, "double": {}, "else": {}, "enum": {}, "extern": {}, "float": {}, "for": {}, "goto": {},
	"if": {}, "inline": {}, "int": {}, "long": {}, "register": {}, "restrict": {}, "return": {},
	"short": {}, "signed": {}, "sizeof": {}, "static": {}, "struct": {}, "switch": {}, "typedef": {},
	"union": {}, "unsigned": {}, "void": {}, "volatile": {}, "while": {}, "_Alignas": {}, "_Alignof": {},
	"_Atomic": {}, "_Bool": {}, "_Complex": {}, "_Generic": {}, "_Imaginary": {}, "_Noreturn": {},
	"_Static_assert": {}, "_Thread_local": {},
}

package hir

import "strings"

// TypeCCallbackI32UserData is the first bounded callback signature. It is only
// source-admitted as the penultimate foreign parameter, before explicit user data.
var TypeCCallbackI32UserData = builtin("unsafe extern C fn i32(i32, *mut void)")

const MaxCallbackValueParameters = 8
const callbackPrefix = "unsafe extern C fn "

// Callbacks remain a closed builtin signature family: only package-independent
// foreign scalars/pointers occur in their spelling. Nominal aggregates, captures
// and general first-class function values are deliberately not represented here.
func CCallback(result Type, parameters []Type) Type {
	if !ValidCallbackSignature(result, parameters) {
		return TypeInvalid
	}
	parts := make([]string, len(parameters))
	for i, typ := range parameters {
		parts[i] = typ.String()
	}
	return builtin(callbackPrefix + result.String() + "(" + strings.Join(parts, ", ") + ")")
}

func CallbackSignature(value Type) (Type, []Type, bool) {
	if value.Descriptor().Kind != TypeKindBuiltin {
		return TypeInvalid, nil, false
	}
	return parseCallbackSignature(value.String(), 0)
}

func IsCCallback(value Type) bool { _, _, ok := CallbackSignature(value); return ok }

func parseCallbackSignature(text string, depth int) (Type, []Type, bool) {
	if !strings.HasPrefix(text, callbackPrefix) || len(text) > 2048 || !strings.HasSuffix(text, ")") {
		return TypeInvalid, nil, false
	}
	body := text[len(callbackPrefix):]
	open := strings.IndexByte(body, '(')
	if open < 1 {
		return TypeInvalid, nil, false
	}
	parts := strings.Split(body[open+1:len(body)-1], ", ")
	if len(parts) < 1 || len(parts) > MaxCallbackValueParameters+1 {
		return TypeInvalid, nil, false
	}
	result := parseType(body[:open], depth+1)
	parameters := make([]Type, len(parts))
	for i, part := range parts {
		if strings.ContainsAny(part, "(),") {
			return TypeInvalid, nil, false
		}
		parameters[i] = parseType(part, depth+1)
		if parameters[i].String() != part {
			return TypeInvalid, nil, false
		}
	}
	if !ValidCallbackSignature(result, parameters) {
		return TypeInvalid, nil, false
	}
	return result, parameters, true
}

func callbackScalar(value Type) bool {
	return value == TypeU8 || value == TypeI32 || value == TypeI64 || value == TypeU32 || value == TypeUSize || IsCScalar(value) || IsIntegerTransportScalar(value) || IsFloatScalar(value)
}

func callbackPointer(value Type) bool {
	element, mutable, pointer := RawPointerElement(value)
	if !pointer {
		return false
	}
	if callbackDataElement(element) {
		return true
	}
	leaf, innerMutable, nested := RawPointerElement(element)
	return mutable && nested && innerMutable && (leaf == TypeVoid || IsCScalar(leaf))
}

func ValidCallbackSignature(result Type, parameters []Type) bool {
	if result != TypeVoid && !callbackScalar(result) && !callbackPointer(result) {
		return false
	}
	if len(parameters) < 1 || len(parameters) > MaxCallbackValueParameters+1 {
		return false
	}
	// The last parameter is explicit borrowed user data, with one pointer layer.
	data, _, pointer := RawPointerElement(parameters[len(parameters)-1])
	if !pointer || !callbackDataElement(data) {
		return false
	}
	for _, typ := range parameters[:len(parameters)-1] {
		if !callbackScalar(typ) && !callbackPointer(typ) {
			return false
		}
	}
	return true
}

func callbackDataElement(value Type) bool {
	return value == TypeVoid || value == TypeU8 || value == TypeI32 || value == TypeI64 || value == TypeU32 || value == TypeUSize || IsCScalar(value)
}

// ValidBoundedCallbackParameters checks the closed callback/user-data pair.
// A declaration without a callback retains its existing parameter contract.
func ValidBoundedCallbackParameters(parameters []Type) bool {
	for index, typ := range parameters {
		if _, signature, callback := CallbackSignature(typ); callback {
			if index != len(parameters)-2 || parameters[index+1] != signature[len(signature)-1] {
				return false
			}
		}
	}
	return true
}

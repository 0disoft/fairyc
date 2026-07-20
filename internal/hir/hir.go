// Package hir defines the typed High-level Intermediate Representation used after semantic analysis.
package hir

import (
	"slices"
	"strconv"
	"strings"
)

// Type is one canonical Fairy C type in the implemented Stage 0 subset.
type Type string

const (
	TypeInvalid    Type = "<invalid>"
	TypeVoid       Type = "void"
	TypeBool       Type = "bool"
	TypeU8         Type = "u8"
	TypeI32        Type = "i32"
	TypeU32        Type = "u32"
	TypeUSize      Type = "usize"
	TypeByteSlice  Type = "[u8]"
	TypeStr        Type = "str"
	TypeCStr       Type = "CStr"
	TypeCBool      Type = "c::bool"
	TypeCChar      Type = "c::char"
	TypeCDouble    Type = "c::double"
	TypeCFloat     Type = "c::float"
	TypeCInt       Type = "c::int"
	TypeCLong      Type = "c::long"
	TypeCLongLong  Type = "c::longlong"
	TypeCSChar     Type = "c::schar"
	TypeCShort     Type = "c::short"
	TypeCSizeT     Type = "c::size_t"
	TypeCSSizeT    Type = "c::ssize_t"
	TypeCUChar     Type = "c::uchar"
	TypeCUInt      Type = "c::uint"
	TypeCULong     Type = "c::ulong"
	TypeCULongLong Type = "c::ulonglong"
	TypeCUShort    Type = "c::ushort"
	TypeRegion     Type = "Region"
)

var cScalarTypes = []Type{
	TypeCBool, TypeCChar, TypeCDouble, TypeCFloat, TypeCInt, TypeCLong,
	TypeCLongLong, TypeCSChar, TypeCShort, TypeCSizeT, TypeCSSizeT,
	TypeCUChar, TypeCUInt, TypeCULong, TypeCULongLong, TypeCUShort,
}

// CScalarTypes returns the closed canonical C scalar vocabulary in target-key order.
func CScalarTypes() []Type {
	return slices.Clone(cScalarTypes)
}

// IsCScalar reports whether value is one exact target-owned C scalar identity.
func IsCScalar(value Type) bool {
	return slices.Contains(cScalarTypes, value)
}

// SharedBorrow returns the canonical non-null shared-borrow type for element.
func SharedBorrow(element Type) Type {
	return Type("&" + string(element))
}

// MutableBorrow returns the canonical non-null exclusive-borrow type for element.
func MutableBorrow(element Type) Type {
	return Type("&mut " + string(element))
}

// FixedArray returns the canonical sized-array type for one positive element count.
func FixedArray(element Type, length uint64) Type {
	return Type("[" + string(element) + "; " + strconv.FormatUint(length, 10) + "]")
}

// MaxExecutableFixedArrayElements is the provisional Stage 0 implementation guard for arrays
// transported by value. It is not a stable language or ABI limit and may change when array storage
// and calling conventions stop using the first private value representation.
const MaxExecutableFixedArrayElements uint64 = 1 << 12

// FixedArrayParts decomposes one canonical positive fixed-array type.
func FixedArrayParts(value Type) (element Type, length uint64, ok bool) {
	text := string(value)
	if len(text) < 5 || text[0] != '[' || text[len(text)-1] != ']' {
		return TypeInvalid, 0, false
	}
	inside := text[1 : len(text)-1]
	depth := 0
	separator := -1
	for index := 0; index < len(inside); index++ {
		switch inside[index] {
		case '[', '<':
			depth++
		case ']', '>':
			depth--
			if depth < 0 {
				return TypeInvalid, 0, false
			}
		case ';':
			if depth == 0 {
				if separator >= 0 {
					return TypeInvalid, 0, false
				}
				separator = index
			}
		}
	}
	if depth != 0 || separator <= 0 || separator == len(inside)-1 {
		return TypeInvalid, 0, false
	}
	elementText := strings.TrimSpace(inside[:separator])
	lengthText := strings.TrimSpace(inside[separator+1:])
	parsed, err := strconv.ParseUint(lengthText, 10, 64)
	if err != nil || parsed == 0 || elementText == "" {
		return TypeInvalid, 0, false
	}
	return Type(elementText), parsed, true
}

// FixedArrayScalarElement reports the closed copy-scalar element set admitted by the first
// executable fixed-array slice.
func FixedArrayScalarElement(value Type) bool {
	return value == TypeBool || value == TypeU8 || value == TypeI32 || value == TypeU32 || value == TypeUSize
}

// ExecutableFixedArrayShape reports whether one array shape belongs to the bounded first
// executable subset shared by semantic analysis, MIR verification, interpretation, and C emission.
func ExecutableFixedArrayShape(element Type, length uint64) bool {
	return FixedArrayScalarElement(element) && length > 0 && length <= MaxExecutableFixedArrayElements
}

// RawPointer returns the canonical nullable raw-pointer type for element.
func RawPointer(element Type, mutable bool) Type {
	if mutable {
		return Type("*mut " + string(element))
	}
	return Type("*" + string(element))
}

// RawPointerElement decomposes one canonical raw-pointer type.
func RawPointerElement(value Type) (element Type, mutable bool, ok bool) {
	text := string(value)
	if len(text) > len("*mut ") && strings.HasPrefix(text, "*mut ") {
		return Type(text[len("*mut "):]), true, true
	}
	if len(text) > 1 && text[0] == '*' {
		return Type(text[1:]), false, true
	}
	return TypeInvalid, false, false
}

// Option returns the canonical compiler-known Option type for one payload type.
func Option(element Type) Type {
	return Type("Option<" + string(element) + ">")
}

// ValuePreservingIntegerCast reports the closed Stage 0 widening conversion subset.
func ValuePreservingIntegerCast(sourceType, targetType Type) bool {
	switch sourceType {
	case TypeU8:
		return targetType == TypeI32 || targetType == TypeU32 || targetType == TypeUSize
	case TypeU32:
		return targetType == TypeUSize
	default:
		return false
	}
}

// ScalarEqualityType reports the closed Stage 0 equality-comparison subset.
func ScalarEqualityType(value Type) bool {
	return value == TypeBool || ScalarOrderedType(value)
}

// ScalarOrderedType reports the closed Stage 0 ordered-comparison subset.
func ScalarOrderedType(value Type) bool {
	return value == TypeI32 || value == TypeU8 || value == TypeU32 || value == TypeUSize
}

// CheckedArithmeticType reports the closed Stage 0 ordinary +, -, and * subset.
func CheckedArithmeticType(value Type) bool {
	return value == TypeI32 || value == TypeU8 || value == TypeU32 || value == TypeUSize
}

// CheckedDivisionType reports the closed Stage 0 ordinary / and % subset.
func CheckedDivisionType(value Type) bool {
	return CheckedArithmeticType(value)
}

// BitwiseIntegerType reports the closed Stage 0 binary &, ^, and | subset.
func BitwiseIntegerType(value Type) bool {
	return CheckedArithmeticType(value)
}

// BitwiseNotType reports the closed Stage 0 unary ~ subset.
func BitwiseNotType(value Type) bool {
	return BitwiseIntegerType(value)
}

// CheckedShiftType reports the closed Stage 0 checked right-shift subset.
func CheckedShiftType(value Type) bool {
	return CheckedArithmeticType(value)
}

// OptionElement decomposes one canonical compiler-known Option type.
func OptionElement(value Type) (element Type, ok bool) {
	text := string(value)
	const prefix = "Option<"
	if len(text) <= len(prefix)+1 || text[:len(prefix)] != prefix || text[len(text)-1] != '>' {
		return TypeInvalid, false
	}
	inside := strings.TrimSpace(text[len(prefix) : len(text)-1])
	if inside == "" {
		return TypeInvalid, false
	}
	depth := 0
	for index := 0; index < len(inside); index++ {
		switch inside[index] {
		case '<':
			depth++
		case '>':
			depth--
			if depth < 0 {
				return TypeInvalid, false
			}
		case ',':
			if depth == 0 {
				return TypeInvalid, false
			}
		}
	}
	if depth != 0 {
		return TypeInvalid, false
	}
	return Type(inside), true
}

// Result returns the canonical compiler-known Result type for two payload types.
func Result(success, failure Type) Type {
	return Type("Result<" + string(success) + ", " + string(failure) + ">")
}

// ResultElements decomposes one canonical compiler-known Result type.
func ResultElements(value Type) (success Type, failure Type, ok bool) {
	text := string(value)
	const prefix = "Result<"
	if len(text) <= len(prefix)+1 || text[:len(prefix)] != prefix || text[len(text)-1] != '>' {
		return TypeInvalid, TypeInvalid, false
	}
	inside := text[len(prefix) : len(text)-1]
	depth := 0
	for index := 0; index < len(inside); index++ {
		switch inside[index] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				left := strings.TrimSpace(inside[:index])
				right := strings.TrimSpace(inside[index+1:])
				if left == "" || right == "" {
					return TypeInvalid, TypeInvalid, false
				}
				return Type(left), Type(right), true
			}
		}
	}
	return TypeInvalid, TypeInvalid, false
}

// BorrowElement decomposes one canonical safe-borrow type.
func BorrowElement(value Type) (element Type, mutable bool, ok bool) {
	text := string(value)
	if len(text) > len("&mut ") && text[:len("&mut ")] == "&mut " {
		return Type(text[len("&mut "):]), true, true
	}
	if len(text) > 1 && text[0] == '&' {
		return Type(text[1:]), false, true
	}
	return TypeInvalid, false, false
}

// SymbolID is build-local and must not be serialized as a language compatibility fact.
type SymbolID int

// NoSymbol represents an expression without a resolved declaration.
const NoSymbol SymbolID = -1

// Span identifies original source bytes without storing source contents.
type Span struct {
	Source    string
	StartByte int
	EndByte   int
}

// Package contains typed modules in deterministic source-path order.
type Package struct {
	Modules []Module
}

// Module is one typed source module.
type Module struct {
	Path             string
	SourcePath       string
	Structs          []Struct
	Enums            []Enum
	Resources        []Resource
	ForeignFunctions []ForeignFunction
	Functions        []Function
}

// Enum is one resolved restricted inert tagged definition in declaration order.
type Enum struct {
	Name     string
	Type     Type
	Span     Span
	Variants []EnumVariant
}

// EnumVariant is one resolved tag with ordered named payload fields.
type EnumVariant struct {
	Name   string
	Span   Span
	Fields []EnumField
}

// EnumField is one resolved inert copyable payload field.
type EnumField struct {
	Name string
	Type Type
	Span Span
}

// Struct is one inert aggregate definition in declaration order.
type Struct struct {
	Name   string
	Type   Type
	Span   Span
	ReprC  bool
	Fields []StructField
}

// StructField is one resolved inert aggregate field.
type StructField struct {
	Name string
	Type Type
	Span Span
}

// ForeignFunction is one typed raw C declaration with an exact external symbol name.
type ForeignFunction struct {
	Symbol     SymbolID
	Name       string
	LinkName   string
	ABI        string
	Span       Span
	Effects    []Effect
	Parameters []ForeignParameter
	ResultType Type
}

// ForeignParameter is one named C ABI parameter in declaration order.
type ForeignParameter struct {
	Name string
	Type Type
	Span Span
}

// ResourceDropKind classifies how much user drop behavior survived semantic lowering.
type ResourceDropKind string

const (
	ResourceDropMissing  ResourceDropKind = "missing"
	ResourceDropEmpty    ResourceDropKind = "empty"
	ResourceDropNonEmpty ResourceDropKind = "non-empty"
)

// Resource is one resolved move-only aggregate definition in declaration order.
type Resource struct {
	Name       string
	Type       Type
	Span       Span
	Fields     []ResourceField
	DropKind   ResourceDropKind
	DropSpan   Span
	DropSymbol SymbolID
}

// ResourceField is one resolved resource field with source identity.
type ResourceField struct {
	Name string
	Type Type
	Span Span
}

// Function is one typed function body.
type Function struct {
	Symbol      SymbolID
	Name        string
	Unsafe      bool
	Span        Span
	Effects     []Effect
	Parameters  []Parameter
	Locals      []Local
	ResultType  Type
	Body        Block
	DropGlueFor Type
}

// Parameter is one resolved function parameter.
type Parameter struct {
	Symbol SymbolID
	Name   string
	Type   Type
	Span   Span
}

// Local is one resolved lexical binding.
type Local struct {
	Symbol  SymbolID
	Name    string
	Type    Type
	Mutable bool
	Span    Span
}

// Block is one typed lexical sequence and optional tail value.
type Block struct {
	Span       Span
	Statements []Statement
	Tail       *Expression
}

// Statement is one typed source statement.
type Statement struct {
	Kind       string
	Span       Span
	Binding    SymbolID
	Target     *Expression
	Expression *Expression
}

// Expression is one typed HIR expression.
type Expression struct {
	Kind        string
	Type        Type
	Span        Span
	Value       string
	Cooked      string
	Variant     string
	Operator    string
	TargetType  Type
	Symbol      SymbolID
	ForeignABI  string
	Effects     []Effect
	UnsafeScope *Span
	Children    []Expression
	Fields      []string
	Block       *Block
	Arms        []MatchArm
}

// MatchArm is one already-resolved shallow tagged branch.
type MatchArm struct {
	Variant     string
	Binding     SymbolID
	BindingSpan Span
	Bindings    []MatchBinding
	Expression  Expression
}

// MatchBinding maps one named user-enum payload field to its arm-local symbol.
type MatchBinding struct {
	Field  string
	Symbol SymbolID
	Type   Type
	Span   Span
}

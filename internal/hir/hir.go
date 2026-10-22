// Package hir defines the typed High-level Intermediate Representation used after semantic analysis.
package hir

import (
	"slices"
	"strconv"
	"strings"
)

// Builtin identities share the same interner as constructed compound types.
var (
	TypeInvalid    Type = builtin("<invalid>")
	TypeVoid       Type = builtin("void")
	TypeBool       Type = builtin("bool")
	TypeU8         Type = builtin("u8")
	TypeI32        Type = builtin("i32")
	TypeI64        Type = builtin("i64")
	TypeU32        Type = builtin("u32")
	TypeUSize      Type = builtin("usize")
	TypeByteSlice  Type = Slice(TypeU8)
	TypeBoolSlice  Type = Slice(TypeBool)
	TypeI64Slice   Type = Slice(TypeI64)
	TypeI32Slice   Type = Slice(TypeI32)
	TypeU32Slice   Type = Slice(TypeU32)
	TypeUsizeSlice Type = Slice(TypeUSize)
	TypeStr        Type = builtin("str")
	TypeCStr       Type = builtin("CStr")
	TypeCBool      Type = builtin("c::bool")
	TypeCChar      Type = builtin("c::char")
	TypeCDouble    Type = builtin("c::double")
	TypeCFloat     Type = builtin("c::float")
	TypeCInt       Type = builtin("c::int")
	TypeCLong      Type = builtin("c::long")
	TypeCLongLong  Type = builtin("c::longlong")
	TypeCSChar     Type = builtin("c::schar")
	TypeCShort     Type = builtin("c::short")
	TypeCSizeT     Type = builtin("c::size_t")
	TypeCSSizeT    Type = builtin("c::ssize_t")
	TypeCUChar     Type = builtin("c::uchar")
	TypeCUInt      Type = builtin("c::uint")
	TypeCULong     Type = builtin("c::ulong")
	TypeCULongLong Type = builtin("c::ulonglong")
	TypeCUShort    Type = builtin("c::ushort")
	TypeRegion     Type = builtin("Region")
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
	return internType(TypeDescriptor{Kind: TypeKindBorrow, Element: element})
}

// MutableBorrow returns the canonical non-null exclusive-borrow type for element.
func MutableBorrow(element Type) Type {
	return internType(TypeDescriptor{Kind: TypeKindBorrow, Element: element, Mutable: true})
}

// FixedArray returns the canonical sized-array type for one positive element count.
func FixedArray(element Type, length uint64) Type {
	return internType(TypeDescriptor{Kind: TypeKindFixedArray, Element: element, Length: length})
}

// MaxExecutableFixedArrayElements is the provisional Stage 0 implementation guard for arrays
// transported by value. It is not a stable language or ABI limit and may change when array storage
// and calling conventions stop using the first private value representation.
const MaxExecutableFixedArrayElements uint64 = 1 << 12

// FixedArrayParts decomposes one canonical positive fixed-array type.
func FixedArrayParts(value Type) (element Type, length uint64, ok bool) {
	d := value.node()
	if d.kind != TypeKindFixedArray || d.length == 0 || d.element == (Type{}) {
		return TypeInvalid, 0, false
	}
	return d.element, d.length, true
}

// FixedArrayScalarElement reports the closed copy-scalar element set admitted by the first
// executable fixed-array slice.
func FixedArrayScalarElement(value Type) bool {
	return value == TypeBool || value == TypeU8 || value == TypeI32 || value == TypeI64 || value == TypeU32 || value == TypeUSize
}

// ExecutableFixedArrayShape reports whether one array shape belongs to the bounded first
// array shape. Nominal elements still require an actual plain struct definition in
// semantic analysis and MIR verification before interpretation or C emission.
func ExecutableFixedArrayShape(element Type, length uint64) bool {
	return (FixedArrayScalarElement(element) || element.Descriptor().Kind == TypeKindNominal) && length > 0 && length <= MaxExecutableFixedArrayElements
}

// RawPointer returns the canonical nullable raw-pointer type for element.
func RawPointer(element Type, mutable bool) Type {
	return internType(TypeDescriptor{Kind: TypeKindRawPointer, Element: element, Mutable: mutable})
}

// RawPointerElement decomposes one canonical raw-pointer type.
func RawPointerElement(value Type) (element Type, mutable bool, ok bool) {
	d := value.node()
	if d.kind != TypeKindRawPointer || d.element == (Type{}) {
		return TypeInvalid, false, false
	}
	return d.element, d.mutable, true
}

// Option returns the canonical compiler-known Option type for one payload type.
func Option(element Type) Type {
	return internType(TypeDescriptor{Kind: TypeKindOption, Element: element})
}

// ValuePreservingIntegerCast reports the closed Stage 0 widening and exact-representation subset.
func ValuePreservingIntegerCast(sourceType, targetType Type) bool {
	switch sourceType {
	case TypeU8:
		return targetType == TypeI32 || targetType == TypeI64 || targetType == TypeU32 || targetType == TypeUSize
	case TypeI32:
		return targetType == TypeCInt || targetType == TypeI64
	case TypeU32:
		return targetType == TypeUSize || targetType == TypeI64
	case TypeCInt:
		return targetType == TypeI32
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
	return value == TypeI32 || value == TypeI64 || value == TypeU8 || value == TypeU32 || value == TypeUSize
}

// CheckedArithmeticType reports the closed Stage 0 ordinary +, -, and * subset.
func CheckedArithmeticType(value Type) bool {
	return value == TypeI32 || value == TypeI64 || value == TypeU8 || value == TypeU32 || value == TypeUSize
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

// CompoundAssignmentBinary maps one source compound assignment to the existing checked binary
// operation that computes its replacement value. Assignment itself remains a statement.
func CompoundAssignmentBinary(operator string) (string, bool) {
	switch operator {
	case "+=":
		return "+", true
	case "-=":
		return "-", true
	case "*=":
		return "*", true
	case "/=":
		return "/", true
	case "%=":
		return "%", true
	case "<<=":
		return "<<", true
	case ">>=":
		return ">>", true
	case "&=":
		return "&", true
	case "^=":
		return "^", true
	case "|=":
		return "|", true
	default:
		return "", false
	}
}

// CompoundAssignmentType reports whether the mapped binary operation is defined for value.
func CompoundAssignmentType(operator string, value Type) bool {
	binary, ok := CompoundAssignmentBinary(operator)
	if !ok {
		return false
	}
	switch binary {
	case "+", "-", "*":
		return CheckedArithmeticType(value)
	case "/", "%":
		return CheckedDivisionType(value)
	case "<<", ">>":
		return CheckedShiftType(value)
	case "&", "^", "|":
		return BitwiseIntegerType(value)
	default:
		return false
	}
}

// OptionElement decomposes one canonical compiler-known Option type.
func OptionElement(value Type) (element Type, ok bool) {
	d := value.node()
	if d.kind != TypeKindOption || d.element == (Type{}) {
		return TypeInvalid, false
	}
	return d.element, true
}

// Result returns the canonical compiler-known Result type for two payload types.
func Result(success, failure Type) Type {
	return internType(TypeDescriptor{Kind: TypeKindResult, Element: success, Failure: failure})
}

// ResultElements decomposes one canonical compiler-known Result type.
func ResultElements(value Type) (success Type, failure Type, ok bool) {
	d := value.node()
	if d.kind != TypeKindResult || d.element == (Type{}) || d.failure == (Type{}) {
		return TypeInvalid, TypeInvalid, false
	}
	return d.element, d.failure, true
}

// BorrowElement decomposes one canonical safe-borrow type.
func BorrowElement(value Type) (element Type, mutable bool, ok bool) {
	d := value.node()
	if d.kind != TypeKindBorrow || d.element == (Type{}) {
		return TypeInvalid, false, false
	}
	return d.element, d.mutable, true
}

// SymbolID is build-local and must not be serialized as a language compatibility fact.
type SymbolID int

// NoSymbol represents an expression without a resolved declaration.
const NoSymbol SymbolID = -1

// PackageID is a deterministic build-local resolved-instance identity. It must not be serialized
// as PackageInstanceId, a lockfile key, a language type spelling, or an ABI name.
type PackageID uint32

// DefaultPackage preserves the original single-package compiler path.
const DefaultPackage PackageID = 0

// QualifiedModulePath returns one compiler-private module identity. Source-visible module paths
// remain unchanged; only non-default resolved instances receive a build-local prefix.
func QualifiedModulePath(packageID PackageID, modulePath string) string {
	if packageID == DefaultPackage {
		return modulePath
	}
	return "$p" + strconv.FormatUint(uint64(packageID), 10) + "::" + modulePath
}

// QualifiedSourcePath returns one compiler-private source identity. The original path remains
// recoverable by the package input owner; this prevents equal package-local paths from collapsing
// build-local spans before a public diagnostic path-mapping contract exists.
func QualifiedSourcePath(packageID PackageID, sourcePath string) string {
	if packageID == DefaultPackage {
		return sourcePath
	}
	return "$p" + strconv.FormatUint(uint64(packageID), 10) + "/" + sourcePath
}

// SourcePathParts separates one compiler-private source identity into its build-local package and
// original source-root-relative path. Valid source paths cannot begin with '$', so malformed
// private spellings remain ordinary default-package paths instead of being guessed.
func SourcePathParts(sourceIdentity string) (PackageID, string) {
	if !strings.HasPrefix(sourceIdentity, "$p") {
		return DefaultPackage, sourceIdentity
	}
	separator := strings.IndexByte(sourceIdentity, '/')
	if separator <= 2 || separator == len(sourceIdentity)-1 {
		return DefaultPackage, sourceIdentity
	}
	value, err := strconv.ParseUint(sourceIdentity[2:separator], 10, 32)
	if err != nil || value == 0 {
		return DefaultPackage, sourceIdentity
	}
	return PackageID(value), sourceIdentity[separator+1:]
}

// Span identifies original source bytes without storing source contents. Package is build-local;
// Source remains the original relative display path and never carries private qualification.
type Span struct {
	Package   PackageID
	Source    string
	StartByte int
	EndByte   int
}

// Package contains typed modules in deterministic source-path order.
type Package struct {
	Modules           []Module
	StateLayouts      []StateLayout
	StateTransactions []StateTransaction
}

// Module is one typed source module.
type Module struct {
	Package          PackageID
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
	CBase             string `json:",omitempty"`
	Name              string
	Public            bool
	Type              Type
	Span              Span
	DocumentationSpan Span
	Variants          []EnumVariant
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
	Name              string
	Public            bool
	Type              Type
	Span              Span
	DocumentationSpan Span
	ReprC             bool
	Fields            []StructField
}

// StructField is one resolved inert aggregate field.
type StructField struct {
	Name string
	Type Type
	Span Span
}

// ForeignFunction is one typed raw C declaration with an exact external symbol name.
type ForeignFunction struct {
	Symbol            SymbolID
	Name              string
	Public            bool
	LinkName          string
	ABI               string
	Span              Span
	DocumentationSpan Span
	MustUse           bool
	Effects           []Effect
	Parameters        []ForeignParameter
	ResultType        Type
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

// ResourceAbility is one compiler-known property of a move-only resource. The
// slice stored on Resource is canonical and describes effective abilities, so
// a legacy declaration carries drop even without an explicit source attribute.
type ResourceAbility string

const (
	ResourceAbilityDrop  ResourceAbility = "drop"
	ResourceAbilityStore ResourceAbility = "store"
	ResourceAbilityKey   ResourceAbility = "key"
)

// Resource is one resolved move-only aggregate definition in declaration order.
type Resource struct {
	Name              string
	Public            bool
	Type              Type
	Span              Span
	DocumentationSpan Span
	Fields            []ResourceField
	DropKind          ResourceDropKind
	DropSpan          Span
	DropSymbol        SymbolID
	DropEffects       []Effect
	Abilities         []ResourceAbility
}

// ResourceField is one resolved resource field with source identity.
type ResourceField struct {
	Name string
	Type Type
	Span Span
}

// Function is one typed function body.
type Function struct {
	Symbol            SymbolID
	Name              string
	Public            bool
	OwnerType         Type
	Receiver          string
	Unsafe            bool
	Span              Span
	DocumentationSpan Span
	MustUse           bool
	Effects           []Effect
	Parameters        []Parameter
	Locals            []Local
	ResultType        Type
	Body              Block
	DropGlueFor       Type
}

// Parameter is one resolved function parameter.
type Parameter struct {
	Symbol  SymbolID
	Name    string
	Type    Type
	Mutable bool
	Span    Span
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
	Kind       StatementKind
	Span       Span
	Operator   string
	Binding    SymbolID
	Target     *Expression
	Expression *Expression
	For        *ForLoop
}

// ForLoop is the typed classic counter-loop boundary. The initializer owns a
// lexical scope that includes Condition, Post, and Body but ends at Span.
type ForLoop struct {
	Span        Span
	Initializer *Statement
	Condition   *Expression
	Post        *Statement
	Body        Block
}

// Expression is one typed HIR expression.
type Expression struct {
	Kind        ExpressionKind
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

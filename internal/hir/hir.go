// Package hir defines the typed High-level Intermediate Representation used after semantic analysis.
package hir

import "strings"

// Type is one canonical Fairy C type in the implemented Stage 0 subset.
type Type string

const (
	TypeInvalid Type = "<invalid>"
	TypeVoid    Type = "void"
	TypeBool    Type = "bool"
	TypeU8      Type = "u8"
	TypeI32     Type = "i32"
	TypeU32     Type = "u32"
	TypeUSize   Type = "usize"
	TypeRegion  Type = "Region"
)

// SharedBorrow returns the canonical non-null shared-borrow type for element.
func SharedBorrow(element Type) Type {
	return Type("&" + string(element))
}

// MutableBorrow returns the canonical non-null exclusive-borrow type for element.
func MutableBorrow(element Type) Type {
	return Type("&mut " + string(element))
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
	Resources        []Resource
	ForeignFunctions []ForeignFunction
	Functions        []Function
}

// ForeignFunction is one typed raw C declaration with an exact external symbol name.
type ForeignFunction struct {
	Symbol     SymbolID
	Name       string
	LinkName   string
	ABI        string
	Span       Span
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
	Name     string
	Type     Type
	Span     Span
	Fields   []ResourceField
	DropKind ResourceDropKind
	DropSpan Span
}

// ResourceField is one resolved resource field with source identity.
type ResourceField struct {
	Name string
	Type Type
	Span Span
}

// Function is one typed function body.
type Function struct {
	Symbol     SymbolID
	Name       string
	Unsafe     bool
	Span       Span
	Parameters []Parameter
	Locals     []Local
	ResultType Type
	Body       Block
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
	Variant     string
	Operator    string
	Symbol      SymbolID
	ForeignABI  string
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
	Expression  Expression
}

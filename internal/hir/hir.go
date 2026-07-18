// Package hir defines the typed High-level Intermediate Representation used after semantic analysis.
package hir

// Type is one canonical Fairy C type in the implemented Stage 0 subset.
type Type string

const (
	TypeInvalid Type = "<invalid>"
	TypeVoid    Type = "void"
	TypeBool    Type = "bool"
	TypeI32     Type = "i32"
)

// SharedBorrow returns the canonical non-null shared-borrow type for element.
func SharedBorrow(element Type) Type {
	return Type("&" + string(element))
}

// MutableBorrow returns the canonical non-null exclusive-borrow type for element.
func MutableBorrow(element Type) Type {
	return Type("&mut " + string(element))
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
	Path       string
	SourcePath string
	Functions  []Function
}

// Function is one typed function body.
type Function struct {
	Symbol     SymbolID
	Name       string
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
	Kind     string
	Type     Type
	Span     Span
	Operator string
	Symbol   SymbolID
	Children []Expression
	Block    *Block
}

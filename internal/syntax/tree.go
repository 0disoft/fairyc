// Package syntax owns the Stage 0 lossless CST and batch parser.
package syntax

import (
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
)

// NodeID is an arena index. It is stable only for one Tree instance.
type NodeID int

// NoNode represents an absent child.
const NoNode NodeID = -1

// Kind classifies compact CST nodes without exposing Go implementation types as a public AST.
type Kind string

// TypeForm records the parser-owned shape of a type node so AST lowering never has to reparse its
// raw source spelling. Text on the node remains the exact source slice used by diagnostics and
// source-aware tools.
type TypeForm string

const (
	TypeFormNamed           TypeForm = "named"
	TypeFormReference       TypeForm = "reference"
	TypeFormReferenceMut    TypeForm = "reference-mut"
	TypeFormPointer         TypeForm = "pointer"
	TypeFormPointerMut      TypeForm = "pointer-mut"
	TypeFormSlice           TypeForm = "slice"
	TypeFormArray           TypeForm = "array"
	TypeFormGrouped         TypeForm = "grouped"
	TypeFormForeignFunction TypeForm = "foreign-function"
)

const (
	KindSource     Kind = "source"
	KindModule     Kind = "module"
	KindImport     Kind = "import"
	KindAttribute  Kind = "attribute"
	KindConst      Kind = "const"
	KindStruct     Kind = "struct"
	KindResource   Kind = "resource"
	KindCleanup    Kind = "cleanup"
	KindEnum       Kind = "enum"
	KindImpl       Kind = "impl"
	KindFunction   Kind = "function"
	KindOwner      Kind = "owner"
	KindForeign    Kind = "foreign-function"
	KindField      Kind = "field"
	KindVariant    Kind = "variant"
	KindParameter  Kind = "parameter"
	KindType       Kind = "type"
	KindBlock      Kind = "block"
	KindBinding    Kind = "binding"
	KindAssignment Kind = "assignment"
	KindAssert     Kind = "assert"
	KindReturn     Kind = "return"
	KindDiscard    Kind = "discard"
	KindWhile      Kind = "while"
	KindBreak      Kind = "break"
	KindContinue   Kind = "continue"
	KindRegion     Kind = "region"
	KindExprStmt   Kind = "expression-statement"
	KindTail       Kind = "tail-expression"
	KindPath       Kind = "path"
	KindLiteral    Kind = "literal"
	KindGroup      Kind = "group"
	KindPrefix     Kind = "prefix"
	KindBinary     Kind = "binary"
	KindCast       Kind = "cast"
	KindCall       Kind = "call"
	KindIndex      Kind = "index"
	KindSlice      Kind = "slice"
	KindOmitBound  Kind = "omitted-range-bound"
	KindFieldExpr  Kind = "field-expression"
	KindPropagate  Kind = "propagate"
	KindArray      Kind = "array"
	KindRecord     Kind = "record"
	KindIf         Kind = "if"
	KindMatch      Kind = "match"
	KindSwitch     Kind = "switch"
	KindUnsafe     Kind = "unsafe"
	KindPattern    Kind = "pattern"
	KindError      Kind = "error"
)

// Node is one compact arena entry. Text and Cooked are populated only when the kind needs them.
type Node struct {
	Kind       Kind
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	TokenStart int
	TokenEnd   int
	Text       string
	Cooked     string
	Children   []NodeID
}

// Tree preserves every lexer token and indexes parsed structure over the same source snapshot.
type Tree struct {
	Source      []byte
	Tokens      []lexer.Token
	Nodes       []Node
	Root        NodeID
	Module      string
	Diagnostics []diagnostic.Diagnostic
}

// Node returns an arena entry when id belongs to the tree.
func (t *Tree) Node(id NodeID) (Node, bool) {
	if t == nil || id < 0 || int(id) >= len(t.Nodes) {
		return Node{}, false
	}
	return t.Nodes[id], true
}

// ErrorFree reports whether the CST may be lowered to AST/HIR-facing syntax.
func (t *Tree) ErrorFree() bool {
	return t != nil && len(t.Diagnostics) == 0
}

// Reconstruct proves that the lossless token stream still accounts for every original byte.
func (t *Tree) Reconstruct() ([]byte, bool) {
	if t == nil {
		return nil, false
	}
	return lexer.Result{Source: t.Source, Tokens: t.Tokens}.Reconstruct()
}

func sortAndDedupeDiagnostics(values []diagnostic.Diagnostic) []diagnostic.Diagnostic {
	slices.SortFunc(values, diagnostic.Compare)
	return slices.CompactFunc(values, func(a, b diagnostic.Diagnostic) bool {
		return diagnostic.Compare(a, b) == 0
	})
}

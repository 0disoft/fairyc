package ast

import (
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/syntax"
)

// TypeKind classifies written type syntax without resolving names or selecting semantic types.
type TypeKind string

const (
	TypeKindInvalid         TypeKind = "invalid"
	TypeKindNamed           TypeKind = "named"
	TypeKindReference       TypeKind = "reference"
	TypeKindRawPointer      TypeKind = "raw-pointer"
	TypeKindSlice           TypeKind = "slice"
	TypeKindArray           TypeKind = "array"
	TypeKindForeignFunction TypeKind = "foreign-function"
)

// TypeLength preserves one fixed-array length expression without folding it during syntax
// lowering. Stage 0 accepts an integer literal or path; semantic analysis owns admissibility and
// value selection.
type TypeLength struct {
	Text string
	Span diagnostic.Span
	Kind syntax.Kind
}

// TypeReference is the structural surface-AST representation of one written type. Text preserves
// the exact trimmed source spelling for diagnostics only; semantic analysis consumes the typed
// fields and must not reparse Text.
type TypeReference struct {
	Kind       TypeKind
	Text       string
	Span       diagnostic.Span
	Path       string
	Mutable    bool
	Element    *TypeReference
	Arguments  []TypeReference
	Parameters []TypeReference
	Result     *TypeReference
	Length     *TypeLength
	ABI        string
}

// NamedTypeReference constructs a synthetic named type with an explicit diagnostic origin.
func NamedTypeReference(path string, span diagnostic.Span) TypeReference {
	return TypeReference{Kind: TypeKindNamed, Text: path, Span: span, Path: path}
}

func lowerType(tree *syntax.Tree, node syntax.Node) TypeReference {
	result := TypeReference{Kind: TypeKindInvalid, Text: strings.TrimSpace(node.Text), Span: node.Span}
	switch syntax.TypeForm(node.Cooked) {
	case syntax.TypeFormNamed:
		result.Kind = TypeKindNamed
		if len(node.Children) == 0 {
			return result
		}
		if path, found := tree.Node(node.Children[0]); found && path.Kind == syntax.KindPath {
			result.Path = path.Text
		}
		for _, childID := range node.Children[1:] {
			if child, found := tree.Node(childID); found && child.Kind == syntax.KindType {
				result.Arguments = append(result.Arguments, lowerType(tree, child))
			}
		}
	case syntax.TypeFormReference, syntax.TypeFormReferenceMut:
		result.Kind = TypeKindReference
		result.Mutable = syntax.TypeForm(node.Cooked) == syntax.TypeFormReferenceMut
		result.Element = lowerTypeElement(tree, node)
	case syntax.TypeFormPointer, syntax.TypeFormPointerMut:
		result.Kind = TypeKindRawPointer
		result.Mutable = syntax.TypeForm(node.Cooked) == syntax.TypeFormPointerMut
		result.Element = lowerTypeElement(tree, node)
	case syntax.TypeFormSlice:
		result.Kind = TypeKindSlice
		result.Element = lowerTypeElement(tree, node)
	case syntax.TypeFormArray:
		result.Kind = TypeKindArray
		result.Element = lowerTypeElement(tree, node)
		if len(node.Children) > 1 {
			if length, found := tree.Node(node.Children[1]); found {
				result.Length = &TypeLength{Text: strings.TrimSpace(length.Text), Span: length.Span, Kind: length.Kind}
			}
		}
	case syntax.TypeFormForeignFunction:
		result.Kind = TypeKindForeignFunction
		result.ABI = "C"
		children := make([]TypeReference, 0, len(node.Children))
		for _, childID := range node.Children {
			if child, found := tree.Node(childID); found && child.Kind == syntax.KindType {
				children = append(children, lowerType(tree, child))
			}
		}
		if len(children) > 0 {
			result.Parameters = append(result.Parameters, children[:len(children)-1]...)
			functionResult := children[len(children)-1]
			result.Result = &functionResult
		}
	}
	return result
}

func lowerTypeElement(tree *syntax.Tree, node syntax.Node) *TypeReference {
	if len(node.Children) == 0 {
		return nil
	}
	child, found := tree.Node(node.Children[0])
	if !found || child.Kind != syntax.KindType {
		return nil
	}
	element := lowerType(tree, child)
	return &element
}

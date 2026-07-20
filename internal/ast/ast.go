// Package ast lowers error-free compact syntax into the first Stage 0 AST boundary.
package ast

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Module is one source file after syntax-only lowering.
type Module struct {
	Path             string
	Span             diagnostic.Span
	Imports          []Import
	Structs          []Struct
	Resources        []Resource
	Enums            []Enum
	ForeignFunctions []ForeignFunction
	Functions        []Function
	Bindings         []Binding
	Literals         []Literal
}

// Enum is one restricted inert tagged type declaration.
type Enum struct {
	Name       string
	Public     bool
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Attributes []Attribute
	Variants   []EnumVariant
}

// EnumVariant preserves one declared tag and its ordered named payload fields.
type EnumVariant struct {
	Name     string
	Span     diagnostic.Span
	NameSpan diagnostic.Span
	Fields   []FieldDeclaration
}

// Struct is one inert named aggregate without compiler-owned cleanup.
type Struct struct {
	Name       string
	Public     bool
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Attributes []Attribute
	Fields     []FieldDeclaration
}

// Attribute preserves one item attribute until semantic ownership selects its meaning.
type Attribute struct {
	Name      string
	Span      diagnostic.Span
	Arguments []AttributeArgument
}

// AttributeArgument preserves source spelling, lexer-cooked content, and its exact source span.
type AttributeArgument struct {
	Raw    string
	Cooked string
	Span   diagnostic.Span
}

// Resource is one move-only named aggregate with compiler-owned cleanup.
type Resource struct {
	Name       string
	Public     bool
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Attributes []Attribute
	Fields     []FieldDeclaration
	Drop       *ResourceDrop
}

// ResourceDrop preserves the source declaration and body until semantic support is selected.
type ResourceDrop struct {
	Span diagnostic.Span
	Body *Block
}

// FieldDeclaration records one named aggregate field and its written type.
type FieldDeclaration struct {
	Name     string
	Span     diagnostic.Span
	NameSpan diagnostic.Span
	Type     TypeReference
}

// Import records one explicit module alias.
type Import struct {
	Path      string
	Alias     string
	Span      diagnostic.Span
	PathSpan  diagnostic.Span
	AliasSpan diagnostic.Span
}

// Function records a source function before name resolution and typing.
type Function struct {
	Name       string
	Public     bool
	Unsafe     bool
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Attributes []Attribute
	Parameters []Parameter
	ResultType TypeReference
	Body       *Block
}

// ForeignFunction records one raw C declaration before name resolution and ABI validation.
type ForeignFunction struct {
	Name       string
	Public     bool
	ABI        string
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Attributes []Attribute
	Parameters []Parameter
	ResultType TypeReference
}

// Parameter records one function parameter and its written type.
type Parameter struct {
	Name     string
	Span     diagnostic.Span
	NameSpan diagnostic.Span
	Type     TypeReference
}

// TypeReference preserves the canonicalizable source spelling and exact source span.
type TypeReference struct {
	Text string
	Span diagnostic.Span
}

// Block owns one lexical statement scope and an optional tail expression.
type Block struct {
	Span       diagnostic.Span
	Statements []Statement
	Tail       *Expression
}

// Statement is the syntax-level statement shape used by semantic analysis.
type Statement struct {
	Kind       syntax.Kind
	Span       diagnostic.Span
	Binding    *Binding
	Target     *Expression
	Expression *Expression
}

// Binding records a source declaration and its expression tree.
type Binding struct {
	Name       string
	Mutable    bool
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Type       *TypeReference
	Expression *Expression
}

// Literal records raw and cooked values without selecting a target type.
type Literal struct {
	Raw    string
	Kind   string
	Cooked string
}

// Expression is a structural AST expression used before name resolution and typing.
type Expression struct {
	Kind     syntax.Kind
	Span     diagnostic.Span
	Text     string
	Cooked   string
	Type     *TypeReference
	Children []*Expression
	Block    *Block
}

// Lower rejects diagnostic CSTs and converts source concepts without semantic analysis.
func Lower(tree *syntax.Tree) (*Module, error) {
	if tree == nil {
		return nil, errors.New("syntax tree is nil")
	}
	if !tree.ErrorFree() {
		return nil, errors.New("diagnostic syntax tree cannot lower to AST")
	}
	root, found := tree.Node(tree.Root)
	if !found || root.Kind != syntax.KindSource {
		return nil, errors.New("syntax tree has no source root")
	}
	module := &Module{Path: tree.Module, Span: root.Span}
	for _, childID := range root.Children {
		child, ok := tree.Node(childID)
		if !ok {
			continue
		}
		switch child.Kind {
		case syntax.KindImport:
			module.Imports = append(module.Imports, lowerImport(tree, child))
		case syntax.KindStruct:
			module.Structs = append(module.Structs, lowerStruct(tree, child))
		case syntax.KindResource:
			module.Resources = append(module.Resources, lowerResource(tree, child))
		case syntax.KindEnum:
			module.Enums = append(module.Enums, lowerEnum(tree, child))
		case syntax.KindFunction:
			module.Functions = append(module.Functions, lowerFunction(tree, child))
		case syntax.KindForeign:
			module.ForeignFunctions = append(module.ForeignFunctions, lowerForeignFunction(tree, child))
		}
	}

	visited := make(map[syntax.NodeID]struct{}, len(tree.Nodes))
	var walk func(syntax.NodeID)
	walk = func(id syntax.NodeID) {
		if id == syntax.NoNode {
			return
		}
		if _, seen := visited[id]; seen {
			return
		}
		visited[id] = struct{}{}
		node, ok := tree.Node(id)
		if !ok {
			return
		}
		if node.Kind == syntax.KindBinding {
			module.Bindings = append(module.Bindings, lowerBinding(tree, node))
		}
		if node.Kind == syntax.KindLiteral {
			module.Literals = append(module.Literals, Literal{
				Raw:    node.Text,
				Kind:   classifyLiteral(node.Text),
				Cooked: node.Cooked,
			})
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(tree.Root)
	return module, nil
}

func lowerEnum(tree *syntax.Tree, node syntax.Node) Enum {
	value := Enum{
		Name:       node.Text,
		Public:     containsKeywordBeforeName(tree, node, "pub", node.Text),
		Span:       node.Span,
		NameSpan:   findIdentifierSpan(tree, node, node.Text, "enum"),
		Attributes: lowerAttributes(tree, node),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != syntax.KindVariant {
			continue
		}
		value.Variants = append(value.Variants, EnumVariant{
			Name:     child.Text,
			Span:     child.Span,
			NameSpan: findIdentifierSpan(tree, child, child.Text, ""),
			Fields:   lowerFields(tree, child),
		})
	}
	return value
}

func lowerStruct(tree *syntax.Tree, node syntax.Node) Struct {
	value := Struct{
		Name:       node.Text,
		Public:     containsKeywordBeforeName(tree, node, "pub", node.Text),
		Span:       node.Span,
		NameSpan:   findIdentifierSpan(tree, node, node.Text, "struct"),
		Attributes: lowerAttributes(tree, node),
	}
	value.Fields = lowerFields(tree, node)
	return value
}

func lowerAttributes(tree *syntax.Tree, node syntax.Node) []Attribute {
	var result []Attribute
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != syntax.KindAttribute {
			continue
		}
		attribute := Attribute{Name: child.Text, Span: child.Span}
		for index, argumentID := range child.Children {
			argument, ok := tree.Node(argumentID)
			if !ok || index == 0 && argument.Kind == syntax.KindPath && argument.Text == child.Text {
				continue
			}
			attribute.Arguments = append(attribute.Arguments, AttributeArgument{
				Raw: argument.Text, Cooked: decodedAttributeArgument(argument), Span: argument.Span,
			})
		}
		result = append(result, attribute)
	}
	return result
}

func decodedAttributeArgument(argument syntax.Node) string {
	if !strings.HasPrefix(argument.Text, "\"") || argument.Cooked == "" {
		return ""
	}
	decoded, err := hex.DecodeString(argument.Cooked)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func lowerResource(tree *syntax.Tree, node syntax.Node) Resource {
	value := Resource{
		Name:       node.Text,
		Public:     containsKeywordBeforeName(tree, node, "pub", node.Text),
		Span:       node.Span,
		NameSpan:   findIdentifierSpan(tree, node, node.Text, "resource"),
		Attributes: lowerAttributes(tree, node),
	}
	value.Fields = lowerFields(tree, node)
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != syntax.KindFunction || child.Text != "drop" {
			continue
		}
		value.Drop = &ResourceDrop{Span: child.Span}
		for _, dropChildID := range child.Children {
			dropChild, ok := tree.Node(dropChildID)
			if ok && dropChild.Kind == syntax.KindBlock {
				value.Drop.Body = lowerBlock(tree, dropChild)
				break
			}
		}
		break
	}
	return value
}

func lowerFields(tree *syntax.Tree, node syntax.Node) []FieldDeclaration {
	var fields []FieldDeclaration
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != syntax.KindField {
			continue
		}
		field := FieldDeclaration{
			Name:     child.Text,
			Span:     child.Span,
			NameSpan: findIdentifierSpan(tree, child, child.Text, ""),
		}
		if len(child.Children) > 0 {
			if typeNode, ok := tree.Node(child.Children[0]); ok && typeNode.Kind == syntax.KindType {
				field.Type = lowerType(typeNode)
			}
		}
		fields = append(fields, field)
	}
	return fields
}

func lowerImport(tree *syntax.Tree, node syntax.Node) Import {
	value := Import{
		Path:  node.Text,
		Alias: node.Cooked,
		Span:  node.Span,
	}
	if len(node.Children) > 0 {
		if pathNode, found := tree.Node(node.Children[0]); found {
			value.PathSpan = pathNode.Span
		}
	}
	value.AliasSpan = findIdentifierSpan(tree, node, node.Cooked, "as")
	return value
}

func lowerFunction(tree *syntax.Tree, node syntax.Node) Function {
	value := Function{
		Name:       node.Text,
		Public:     containsKeywordBeforeName(tree, node, "pub", node.Text),
		Unsafe:     containsKeywordBeforeName(tree, node, "unsafe", node.Text),
		Span:       node.Span,
		NameSpan:   findIdentifierSpan(tree, node, node.Text, "fn"),
		Attributes: lowerAttributes(tree, node),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found {
			continue
		}
		switch child.Kind {
		case syntax.KindParameter:
			value.Parameters = append(value.Parameters, lowerParameter(tree, child))
		case syntax.KindType:
			value.ResultType = lowerType(child)
		case syntax.KindBlock:
			value.Body = lowerBlock(tree, child)
		}
	}
	return value
}

func lowerForeignFunction(tree *syntax.Tree, node syntax.Node) ForeignFunction {
	value := ForeignFunction{
		Name:       node.Text,
		Public:     containsKeywordBeforeName(tree, node, "pub", node.Text),
		ABI:        "C",
		Span:       node.Span,
		NameSpan:   findIdentifierSpan(tree, node, node.Text, "fn"),
		Attributes: lowerAttributes(tree, node),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found {
			continue
		}
		switch child.Kind {
		case syntax.KindParameter:
			value.Parameters = append(value.Parameters, lowerParameter(tree, child))
		case syntax.KindType:
			value.ResultType = lowerType(child)
		}
	}
	return value
}

func lowerParameter(tree *syntax.Tree, node syntax.Node) Parameter {
	value := Parameter{
		Name:     node.Text,
		Span:     node.Span,
		NameSpan: findIdentifierSpan(tree, node, node.Text, ""),
	}
	if len(node.Children) > 0 {
		if typeNode, found := tree.Node(node.Children[0]); found && typeNode.Kind == syntax.KindType {
			value.Type = lowerType(typeNode)
		}
	}
	return value
}

func lowerType(node syntax.Node) TypeReference {
	return TypeReference{Text: strings.TrimSpace(node.Text), Span: node.Span}
}

func lowerBlock(tree *syntax.Tree, node syntax.Node) *Block {
	block := &Block{Span: node.Span}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found {
			continue
		}
		if child.Kind == syntax.KindTail {
			if len(child.Children) > 0 {
				block.Tail = lowerExpression(tree, child.Children[0])
			}
			continue
		}
		block.Statements = append(block.Statements, lowerStatement(tree, child))
	}
	return block
}

func lowerStatement(tree *syntax.Tree, node syntax.Node) Statement {
	statement := Statement{Kind: node.Kind, Span: node.Span}
	switch node.Kind {
	case syntax.KindBinding:
		binding := lowerBinding(tree, node)
		statement.Binding = &binding
	case syntax.KindAssignment:
		if len(node.Children) > 0 {
			statement.Target = lowerExpression(tree, node.Children[0])
		}
		if len(node.Children) > 1 {
			statement.Expression = lowerExpression(tree, node.Children[1])
		}
	case syntax.KindAssert, syntax.KindReturn, syntax.KindDiscard, syntax.KindExprStmt:
		if len(node.Children) > 0 {
			statement.Expression = lowerExpression(tree, node.Children[0])
		}
	default:
		statement.Expression = lowerExpressionNode(tree, node)
	}
	return statement
}

func lowerBinding(tree *syntax.Tree, node syntax.Node) Binding {
	binding := Binding{
		Name:     node.Text,
		Mutable:  node.Cooked == "var",
		Span:     node.Span,
		NameSpan: findIdentifierSpan(tree, node, node.Text, node.Cooked),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found {
			continue
		}
		if child.Kind == syntax.KindType {
			typeReference := lowerType(child)
			binding.Type = &typeReference
			continue
		}
		binding.Expression = lowerExpression(tree, childID)
	}
	return binding
}

func lowerExpression(tree *syntax.Tree, id syntax.NodeID) *Expression {
	node, found := tree.Node(id)
	if !found {
		return nil
	}
	return lowerExpressionNode(tree, node)
}

func lowerExpressionNode(tree *syntax.Tree, node syntax.Node) *Expression {
	expression := &Expression{
		Kind:   node.Kind,
		Span:   node.Span,
		Text:   node.Text,
		Cooked: node.Cooked,
	}
	if node.Kind == syntax.KindBlock {
		expression.Block = lowerBlock(tree, node)
		return expression
	}
	for _, child := range node.Children {
		if node.Kind == syntax.KindCast {
			if typeNode, found := tree.Node(child); found && typeNode.Kind == syntax.KindType {
				target := lowerType(typeNode)
				expression.Type = &target
				continue
			}
		}
		if lowered := lowerExpression(tree, child); lowered != nil {
			expression.Children = append(expression.Children, lowered)
		}
	}
	return expression
}

func findIdentifierSpan(tree *syntax.Tree, node syntax.Node, name, after string) diagnostic.Span {
	if name == "" {
		return diagnostic.Span{}
	}
	seenAfter := after == ""
	start := max(node.TokenStart, 0)
	end := min(node.TokenEnd, len(tree.Tokens))
	for index := start; index < end; index++ {
		token := tree.Tokens[index]
		text := token.Text(tree.Source)
		if !seenAfter {
			seenAfter = text == after
			continue
		}
		if token.Kind == lexer.Identifier && text == name {
			return token.Span
		}
	}
	return node.Span
}

func containsKeywordBeforeName(tree *syntax.Tree, node syntax.Node, keyword, name string) bool {
	start := max(node.TokenStart, 0)
	end := min(node.TokenEnd, len(tree.Tokens))
	for index := start; index < end; index++ {
		text := tree.Tokens[index].Text(tree.Source)
		if tree.Tokens[index].Kind == lexer.Identifier && text == name {
			return false
		}
		if text == keyword {
			return true
		}
	}
	return false
}

func classifyLiteral(raw string) string {
	switch {
	case raw == "true" || raw == "false":
		return "boolean"
	case raw == "null":
		return "null"
	case strings.HasPrefix(raw, "b\""):
		return "byte-string"
	case strings.HasPrefix(raw, "c\""):
		return "c-string"
	case strings.HasPrefix(raw, "\""):
		return "string"
	case strings.ContainsAny(raw, ".eE") && !strings.HasPrefix(raw, "0x"):
		return "float"
	default:
		return "integer"
	}
}

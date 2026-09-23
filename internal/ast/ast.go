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
	SourceSize       int
	SourceLineStarts []int
	Imports          []Import
	Constants        []Constant
	StaticAssertions []StaticAssertion
	States           []StateDeclaration
	Structs          []Struct
	Resources        []Resource
	Enums            []Enum
	ForeignFunctions []ForeignFunction
	Functions        []Function
	Bindings         []Binding
	Literals         []Literal
}

// StateDeclaration is the single package-owned persistent-state schema admitted by source.
type StateDeclaration struct {
	Public     bool
	Span       diagnostic.Span
	Attributes []Attribute
	Fields     []StateFieldDeclaration
}

// StateFieldDeclaration preserves one ordered cell or map codec declaration.
type StateFieldKind string

const (
	StateFieldCell StateFieldKind = "cell"
	StateFieldMap  StateFieldKind = "map"
)

type StateFieldDeclaration struct {
	Kind      StateFieldKind
	Name      string
	Span      diagnostic.Span
	NameSpan  diagnostic.Span
	KeyType   *TypeReference
	ValueType TypeReference
}

// Constant preserves one module constant before bounded compile-time evaluation.
type Constant struct {
	Name       string
	Public     bool
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Attributes []Attribute
	Type       TypeReference
	Expression *Expression
}

// StaticAssertion preserves one module-level compile-time assertion.
type StaticAssertion struct {
	Public     bool
	Span       diagnostic.Span
	Attributes []Attribute
	Expression *Expression
}

// Enum is one restricted inert tagged type declaration.
type Enum struct {
	Name              string
	Public            bool
	Span              diagnostic.Span
	NameSpan          diagnostic.Span
	DocumentationSpan diagnostic.Span
	Attributes        []Attribute
	Variants          []EnumVariant
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
	TypeParameters    []string
	Name              string
	Public            bool
	Span              diagnostic.Span
	NameSpan          diagnostic.Span
	DocumentationSpan diagnostic.Span
	Attributes        []Attribute
	Fields            []FieldDeclaration
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
	TypeParameters    []string
	Name              string
	Public            bool
	Span              diagnostic.Span
	NameSpan          diagnostic.Span
	DocumentationSpan diagnostic.Span
	Attributes        []Attribute
	Fields            []FieldDeclaration
	Cleanup           *ResourceCleanup
}

// ResourceCleanup preserves the non-callable cleanup declaration until semantic lowering creates
// compiler-private drop glue.
type ResourceCleanup struct {
	Span         diagnostic.Span
	ReceiverName string
	Attributes   []Attribute
	Body         *Block
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
	TypeParameters    []string
	Name              string
	Owner             string
	Public            bool
	Unsafe            bool
	Span              diagnostic.Span
	NameSpan          diagnostic.Span
	OwnerSpan         diagnostic.Span
	DocumentationSpan diagnostic.Span
	Attributes        []Attribute
	Parameters        []Parameter
	ResultType        TypeReference
	Body              *Block
}

// ForeignFunction records one raw C declaration before name resolution and ABI validation.
type ForeignFunction struct {
	Name              string
	Public            bool
	ABI               string
	Span              diagnostic.Span
	NameSpan          diagnostic.Span
	DocumentationSpan diagnostic.Span
	Attributes        []Attribute
	Parameters        []Parameter
	ResultType        TypeReference
}

// Parameter records one function parameter and its written type.
type Parameter struct {
	Name     string
	Mutable  bool
	Span     diagnostic.Span
	NameSpan diagnostic.Span
	Type     TypeReference
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
	Operator   string
	Binding    *Binding
	Target     *Expression
	Expression *Expression
	For        *ForLoop
	ByteFor    *ByteForLoop
}

// ByteForLoop retains the proposed read-only byte-view form until semantic
// admission; the parser must not silently turn it into a classic counter loop.
type ByteForLoop struct {
	Span     diagnostic.Span
	Name     string
	NameSpan diagnostic.Span
	Type     TypeReference
	View     *Expression
	Body     *Block
}

// ForLoop preserves the four source regions of one classic C-shaped loop.
// Header clauses stay distinct so semantic lowering can keep initializer scope
// and route continue through Post before reevaluating Condition.
type ForLoop struct {
	Span        diagnostic.Span
	Initializer *Statement
	Condition   *Expression
	Post        *Statement
	Body        *Block
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
	CallLabels    *CallArgumentLabels `json:",omitempty"`
	TypeArguments []TypeReference
	Kind          syntax.Kind
	Span          diagnostic.Span
	Text          string
	Cooked        string
	Type          *TypeReference
	Children      []*Expression
	Block         *Block
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
	lineStarts := []int{0}
	for index, value := range tree.Source {
		if value == '\n' {
			lineStarts = append(lineStarts, index+1)
		}
	}
	module := &Module{
		Path:             tree.Module,
		Span:             root.Span,
		SourceSize:       len(tree.Source),
		SourceLineStarts: lineStarts,
	}
	for _, childID := range root.Children {
		child, ok := tree.Node(childID)
		if !ok {
			continue
		}
		switch child.Kind {
		case syntax.KindImport:
			module.Imports = append(module.Imports, lowerImport(tree, child))
		case syntax.KindConst:
			module.Constants = append(module.Constants, lowerConstant(tree, child))
		case syntax.KindStaticAssert:
			module.StaticAssertions = append(module.StaticAssertions, lowerStaticAssertion(tree, child))
		case syntax.KindState:
			module.States = append(module.States, lowerState(tree, child))
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

func lowerState(tree *syntax.Tree, node syntax.Node) StateDeclaration {
	value := StateDeclaration{
		Public:     containsKeywordBeforeName(tree, node, "pub", "state"),
		Span:       node.Span,
		Attributes: lowerAttributes(tree, node),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != syntax.KindStateCell && child.Kind != syntax.KindStateMap {
			continue
		}
		field := StateFieldDeclaration{
			Name:     child.Text,
			Span:     child.Span,
			NameSpan: child.NameSpan,
		}
		var types []TypeReference
		for _, typeID := range child.Children {
			if typeNode, ok := tree.Node(typeID); ok && typeNode.Kind == syntax.KindType {
				types = append(types, lowerType(tree, typeNode))
			}
		}
		if child.Kind == syntax.KindStateCell && len(types) == 1 {
			field.Kind = StateFieldCell
			field.ValueType = types[0]
		} else if child.Kind == syntax.KindStateMap && len(types) == 2 {
			field.Kind = StateFieldMap
			field.KeyType = &types[0]
			field.ValueType = types[1]
		}
		value.Fields = append(value.Fields, field)
	}
	return value
}

func lowerConstant(tree *syntax.Tree, node syntax.Node) Constant {
	value := Constant{
		Name:       node.Text,
		Public:     containsKeywordBeforeName(tree, node, "pub", node.Text),
		Span:       node.Span,
		NameSpan:   node.NameSpan,
		Attributes: lowerAttributes(tree, node),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found {
			continue
		}
		switch child.Kind {
		case syntax.KindType:
			value.Type = lowerType(tree, child)
		default:
			if child.Kind != syntax.KindAttribute {
				value.Expression = lowerExpression(tree, childID)
			}
		}
	}
	return value
}

func lowerStaticAssertion(tree *syntax.Tree, node syntax.Node) StaticAssertion {
	value := StaticAssertion{
		Public:     containsKeywordBeforeName(tree, node, "pub", "static_assert"),
		Span:       node.Span,
		Attributes: lowerAttributes(tree, node),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if found && child.Kind != syntax.KindAttribute {
			value.Expression = lowerExpression(tree, childID)
			break
		}
	}
	return value
}

func lowerEnum(tree *syntax.Tree, node syntax.Node) Enum {
	value := Enum{
		Name:              node.Text,
		Public:            containsKeywordBeforeName(tree, node, "pub", node.Text),
		Span:              node.Span,
		NameSpan:          findIdentifierSpan(tree, node, node.Text, "enum"),
		DocumentationSpan: node.DocumentationSpan,
		Attributes:        lowerAttributes(tree, node),
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
		TypeParameters:    lowerTypeParameters(tree, node),
		Name:              node.Text,
		Public:            containsKeywordBeforeName(tree, node, "pub", node.Text),
		Span:              node.Span,
		NameSpan:          findIdentifierSpan(tree, node, node.Text, "struct"),
		DocumentationSpan: node.DocumentationSpan,
		Attributes:        lowerAttributes(tree, node),
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
		TypeParameters:    lowerTypeParameters(tree, node),
		Name:              node.Text,
		Public:            containsKeywordBeforeName(tree, node, "pub", node.Text),
		Span:              node.Span,
		NameSpan:          findIdentifierSpan(tree, node, node.Text, "resource"),
		DocumentationSpan: node.DocumentationSpan,
		Attributes:        lowerAttributes(tree, node),
	}
	value.Fields = lowerFields(tree, node)
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != syntax.KindCleanup {
			continue
		}
		value.Cleanup = &ResourceCleanup{
			Span: child.Span, ReceiverName: "this", Attributes: lowerAttributes(tree, child),
		}
		for _, dropChildID := range child.Children {
			dropChild, ok := tree.Node(dropChildID)
			if ok && dropChild.Kind == syntax.KindBlock {
				value.Cleanup.Body = lowerBlock(tree, dropChild)
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
				field.Type = lowerType(tree, typeNode)
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
	public, unsafe := functionModifiers(tree, node)
	value := Function{
		Name:              node.Text,
		Public:            public,
		Unsafe:            unsafe,
		Span:              node.Span,
		NameSpan:          findIdentifierSpan(tree, node, node.Text, "fn"),
		DocumentationSpan: node.DocumentationSpan,
		Attributes:        lowerAttributes(tree, node),
	}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found {
			continue
		}
		switch child.Kind {
		case syntax.KindOwner:
			value.Owner = child.Text
			value.OwnerSpan = child.NameSpan
		case syntax.KindTypeParameters:
			for _, parameterID := range child.Children {
				if parameter, found := tree.Node(parameterID); found {
					value.TypeParameters = append(value.TypeParameters, parameter.Text)
				}
			}
		case syntax.KindParameter:
			value.Parameters = append(value.Parameters, lowerParameter(tree, child))
		case syntax.KindType:
			value.ResultType = lowerType(tree, child)
		case syntax.KindBlock:
			value.Body = lowerBlock(tree, child)
		}
	}
	return value
}

func lowerForeignFunction(tree *syntax.Tree, node syntax.Node) ForeignFunction {
	value := ForeignFunction{
		Name:              node.Text,
		Public:            containsKeywordBeforeName(tree, node, "pub", node.Text),
		ABI:               "C",
		Span:              node.Span,
		NameSpan:          findIdentifierSpan(tree, node, node.Text, "fn"),
		DocumentationSpan: node.DocumentationSpan,
		Attributes:        lowerAttributes(tree, node),
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
			value.ResultType = lowerType(tree, child)
		}
	}
	return value
}

func lowerParameter(tree *syntax.Tree, node syntax.Node) Parameter {
	value := Parameter{
		Name:     node.Text,
		Mutable:  node.Cooked == "mut",
		Span:     node.Span,
		NameSpan: findIdentifierSpan(tree, node, node.Text, node.Cooked),
	}
	if len(node.Children) > 0 {
		if typeNode, found := tree.Node(node.Children[0]); found && typeNode.Kind == syntax.KindType {
			value.Type = lowerType(tree, typeNode)
		}
	}
	return value
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
		statement.Operator = node.Text
		if statement.Operator == "" {
			statement.Operator = "="
		}
		if len(node.Children) > 0 {
			statement.Target = lowerExpression(tree, node.Children[0])
		}
		if len(node.Children) > 1 {
			statement.Expression = lowerExpression(tree, node.Children[1])
		}
	case syntax.KindAssert, syntax.KindDefer, syntax.KindReturn, syntax.KindDiscard, syntax.KindBurn, syntax.KindExprStmt:
		if len(node.Children) > 0 {
			statement.Expression = lowerExpression(tree, node.Children[0])
		}
	case syntax.KindFor:
		statement.For = lowerForLoop(tree, node)
	case syntax.KindByteFor:
		statement.ByteFor = lowerByteForLoop(tree, node)
		statement.Expression = byteForAsBlock(statement.ByteFor)
	default:
		statement.Expression = lowerExpressionNode(tree, node)
	}
	return statement
}

func lowerByteForLoop(tree *syntax.Tree, node syntax.Node) *ByteForLoop {
	loop := &ByteForLoop{Span: node.Span, Name: node.Text, NameSpan: node.NameSpan}
	if len(node.Children) != 3 {
		return loop
	}
	if itemType, found := tree.Node(node.Children[0]); found {
		loop.Type = lowerType(tree, itemType)
	}
	loop.View = lowerExpression(tree, node.Children[1])
	if body, found := tree.Node(node.Children[2]); found && body.Kind == syntax.KindBlock {
		loop.Body = lowerBlock(tree, body)
	}
	return loop
}

func lowerForLoop(tree *syntax.Tree, node syntax.Node) *ForLoop {
	loop := &ForLoop{Span: node.Span}
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found {
			continue
		}
		switch child.Kind {
		case syntax.KindForInit:
			if len(child.Children) == 1 {
				value, ok := tree.Node(child.Children[0])
				if ok {
					statement := lowerStatement(tree, value)
					loop.Initializer = &statement
				}
			}
		case syntax.KindForCond:
			if len(child.Children) == 1 {
				loop.Condition = lowerExpression(tree, child.Children[0])
			}
		case syntax.KindForPost:
			if len(child.Children) == 1 {
				value, ok := tree.Node(child.Children[0])
				if ok {
					statement := lowerStatement(tree, value)
					loop.Post = &statement
				}
			}
		case syntax.KindBlock:
			loop.Body = lowerBlock(tree, child)
		}
	}
	return loop
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
			typeReference := lowerType(tree, child)
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
		if node.Kind == syntax.KindCall {
			if label, found := tree.Node(child); found && label.Kind == syntax.KindArgumentLabel && len(label.Children) == 1 {
				if expression.CallLabels == nil {
					expression.CallLabels = &CallArgumentLabels{}
				}
				expression.CallLabels.Labels = append(expression.CallLabels.Labels, ArgumentLabel{Index: len(expression.Children) - 1, Name: label.Text, Span: label.Span})
				child = label.Children[0]
			}
		}
		if node.Kind == syntax.KindSpecialize || node.Kind == syntax.KindRecord {
			if typeNode, found := tree.Node(child); found && typeNode.Kind == syntax.KindType {
				expression.TypeArguments = append(expression.TypeArguments, lowerType(tree, typeNode))
				continue
			}
		}
		if node.Kind == syntax.KindCast {
			if typeNode, found := tree.Node(child); found && typeNode.Kind == syntax.KindType {
				target := lowerType(tree, typeNode)
				expression.Type = &target
				continue
			}
		}
		if lowered := lowerExpression(tree, child); lowered != nil {
			expression.Children = append(expression.Children, lowered)
		}
	}
	if expression.CallLabels != nil && len(expression.Children) > 0 {
		expression.CallLabels.ExplicitTypeArguments = expression.Children[0].Kind == syntax.KindSpecialize
	}
	return expression
}

func findIdentifierSpan(tree *syntax.Tree, node syntax.Node, name, after string) diagnostic.Span {
	if name == "" {
		return diagnostic.Span{}
	}
	if node.NameSpan.EndByte > node.NameSpan.StartByte {
		return node.NameSpan
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

package ast

import "github.com/0disoft/fairyc/internal/syntax"

// Vector loops are prepared only in the authenticated SDK expansion lane.
// Source ByteForLoop nodes remain intact, and semantic analysis still decides
// whether to admit them.
const vectorForOwnerName = "\x00fairyc_vector_for_owner"
const vectorForBoundName = "\x00fairyc_vector_for_bound"
const vectorForIndexName = "\x00fairyc_vector_for_index"

// PrepareSDKVectorLoopBody attaches candidate lowering to a detached function
// body after its std::vec provider has been authenticated. Ordinary AST
// lowering never builds these unused blocks.
func PrepareSDKVectorLoopBody(block *Block, alias string) {
	if alias != "" {
		prepareVectorForBlock(block, alias)
	}
}

func prepareVectorForBlock(block *Block, alias string) {
	if block == nil {
		return
	}
	for index := range block.Statements {
		prepareVectorForStatement(&block.Statements[index], alias)
	}
	prepareVectorForExpression(block.Tail, alias)
}

func prepareVectorForStatement(statement *Statement, alias string) {
	if statement == nil {
		return
	}
	if statement.ByteFor != nil {
		prepareVectorForBlock(statement.ByteFor.Body, alias)
		if statement.ByteFor.Type.Kind == TypeKindReference && !statement.ByteFor.Type.Mutable {
			statement.ByteFor.PreparedVector = vectorForAsBlock(statement.ByteFor, alias)
		}
		return
	}
	if statement.For != nil {
		prepareVectorForStatement(statement.For.Initializer, alias)
		prepareVectorForExpression(statement.For.Condition, alias)
		prepareVectorForStatement(statement.For.Post, alias)
		prepareVectorForBlock(statement.For.Body, alias)
	}
	if statement.Binding != nil {
		prepareVectorForExpression(statement.Binding.Expression, alias)
	}
	prepareVectorForExpression(statement.Target, alias)
	prepareVectorForExpression(statement.Expression, alias)
}

func prepareVectorForExpression(expression *Expression, alias string) {
	if expression == nil {
		return
	}
	prepareVectorForBlock(expression.Block, alias)
	for _, child := range expression.Children {
		prepareVectorForExpression(child, alias)
	}
}

func vectorForAsBlock(loop *ByteForLoop, alias string) *Expression {
	if loop == nil || loop.Body == nil || loop.View == nil || loop.Type.Kind != TypeKindReference ||
		loop.Type.Mutable || loop.Type.Element == nil || alias == "" || !VectorForEligibleView(loop.View) {
		return nil
	}
	itemType := *loop.Type.Element
	vecType := TypeReference{
		Kind: TypeKindNamed, Path: alias + "::Vec", Text: alias + "::Vec", Span: loop.NameSpan,
		Arguments: []TypeReference{itemType},
	}
	ownerType := TypeReference{Kind: TypeKindReference, Element: &vecType, Span: loop.NameSpan}
	path := func(name string) *Expression {
		return &Expression{Kind: syntax.KindPath, Span: loop.NameSpan, Text: name}
	}
	call := func(name string, arguments ...*Expression) *Expression {
		callee := &Expression{Kind: syntax.KindSpecialize, Span: loop.NameSpan,
			TypeArguments: []TypeReference{itemType},
			Children:      []*Expression{path(alias + "::" + name)}}
		return &Expression{Kind: syntax.KindCall, Span: loop.NameSpan,
			Children: append([]*Expression{callee}, arguments...)}
	}
	literal := func(text string) *Expression {
		return &Expression{Kind: syntax.KindLiteral, Span: loop.NameSpan, Text: text, Cooked: text}
	}
	owner := &Binding{Name: vectorForOwnerName, Span: loop.Span, NameSpan: loop.NameSpan,
		Type: &ownerType, Expression: loop.View}
	boundType := NamedTypeReference("usize", loop.NameSpan)
	bound := &Binding{Name: vectorForBoundName, Span: loop.Span, NameSpan: loop.NameSpan,
		Type: &boundType, Expression: call("len", path(vectorForOwnerName))}
	indexType := NamedTypeReference("usize", loop.NameSpan)
	index := &Binding{Name: vectorForIndexName, Mutable: true, Span: loop.Span, NameSpan: loop.NameSpan,
		Type: &indexType, Expression: literal("0")}
	item := &Binding{Name: loop.Name, Span: loop.NameSpan, NameSpan: loop.NameSpan,
		Type: &loop.Type, Expression: call("at", path(vectorForOwnerName), path(vectorForIndexName))}
	body := *loop.Body
	body.Statements = make([]Statement, 0, len(loop.Body.Statements)+1)
	body.Statements = append(body.Statements, Statement{Kind: syntax.KindBinding, Span: loop.NameSpan, Binding: item})
	body.Statements = append(body.Statements, loop.Body.Statements...)
	classic := &ForLoop{Span: loop.Span,
		Initializer: &Statement{Kind: syntax.KindBinding, Span: loop.NameSpan, Binding: index},
		Condition: &Expression{Kind: syntax.KindBinary, Span: loop.View.Span, Text: "<",
			Children: []*Expression{path(vectorForIndexName), path(vectorForBoundName)}},
		Post: &Statement{Kind: syntax.KindAssignment, Span: loop.NameSpan, Operator: "+=",
			Target: path(vectorForIndexName), Expression: literal("1")},
		Body: &body,
	}
	block := &Block{Span: loop.Span, Statements: []Statement{
		{Kind: syntax.KindBinding, Span: loop.NameSpan, Binding: owner},
		{Kind: syntax.KindBinding, Span: loop.NameSpan, Binding: bound},
		{Kind: syntax.KindFor, Span: loop.Span, For: classic},
	}}
	return &Expression{Kind: syntax.KindBlock, Span: loop.Span, Block: block}
}

// VectorForEligibleView is the shared source-shape check used by preparation
// and the semantic candidate gate.
func VectorForEligibleView(view *Expression) bool {
	if view == nil {
		return false
	}
	if view.Kind == syntax.KindPath && len(view.Children) == 0 {
		return true
	}
	if view.Kind != syntax.KindPrefix || view.Text != "&" || len(view.Children) != 1 {
		return false
	}
	var place func(*Expression) bool
	place = func(expression *Expression) bool {
		if expression == nil {
			return false
		}
		if expression.Kind == syntax.KindPath && len(expression.Children) == 0 {
			return true
		}
		return expression.Kind == syntax.KindFieldExpr && len(expression.Children) == 1 &&
			place(expression.Children[0])
	}
	return place(view.Children[0])
}

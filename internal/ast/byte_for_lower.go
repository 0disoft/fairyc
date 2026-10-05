package ast

import "github.com/0disoft/fairyc/internal/syntax"

// These names contain a byte that source identifiers cannot contain. The
// synthetic bindings have one private lexical block per written byte loop.
const byteForBoundName = "\x00fairyc_byte_for_bound"
const byteForIndexName = "\x00fairyc_byte_for_index"

// byteForAsBlock prepares the ordinary scalar-loop path without replacing the
// written AST node. Semantic admission still checks the source form and may
// reject it before the generated expression reaches executable MIR.
func byteForAsBlock(loop *ByteForLoop) *Expression {
	return prepareViewFor(loop, false)
}

// PrepareReferenceViewLoop is used after SDK vector candidates have been
// selected. Actual integer/struct view and array admission remains semantic-owned.
func PrepareReferenceViewLoop(loop *ByteForLoop) *Expression {
	return prepareViewFor(loop, true)
}

func prepareViewFor(loop *ByteForLoop, sharedItems bool) *Expression {
	if loop == nil || loop.Body == nil || loop.View == nil || !viewForSource(loop.View) {
		return nil
	}
	itemType := loop.Type
	mutableItem := itemType.Kind == TypeKindReference && itemType.Mutable && itemType.Element != nil
	referenceItem := itemType.Kind == TypeKindReference && itemType.Element != nil && (itemType.Mutable || sharedItems)
	element := itemType
	if referenceItem {
		element = *itemType.Element
	}
	if (element.Kind != TypeKindNamed && element.Kind != TypeKindBoundNominal) || len(element.Arguments) != 0 {
		return nil
	}
	view := func() *Expression {
		if loop.View.Kind != syntax.KindPath {
			return loop.View // Semantic preparation replaces this with a single descriptor.
		}
		return &Expression{Kind: syntax.KindPath, Span: loop.View.Span, Text: loop.View.Text}
	}
	path := func(name string) *Expression {
		return &Expression{Kind: syntax.KindPath, Span: loop.NameSpan, Text: name}
	}
	literal := func(text string) *Expression {
		return &Expression{Kind: syntax.KindLiteral, Span: loop.NameSpan, Text: text, Cooked: text}
	}
	boundType := NamedTypeReference("usize", loop.NameSpan)
	bound := &Binding{
		Name: byteForBoundName, Span: loop.Span, NameSpan: loop.NameSpan,
		Type:       &boundType,
		Expression: &Expression{Kind: syntax.KindFieldExpr, Span: loop.View.Span, Text: "len", Children: []*Expression{loop.View}},
	}
	indexType := NamedTypeReference("usize", loop.NameSpan)
	index := &Binding{
		Name: byteForIndexName, Mutable: true, Span: loop.Span, NameSpan: loop.NameSpan,
		Type: &indexType, Expression: literal("0"),
	}
	item := &Binding{
		Name: loop.Name, Span: loop.NameSpan, NameSpan: loop.NameSpan,
		Type: &itemType,
		Expression: &Expression{Kind: syntax.KindIndex, Span: loop.View.Span,
			Children: []*Expression{view(), path(byteForIndexName)}},
	}
	if referenceItem {
		operator := "&"
		if mutableItem {
			operator = "&mut"
		}
		item.Expression = &Expression{Kind: syntax.KindPrefix, Span: loop.NameSpan, Text: operator, Children: []*Expression{item.Expression}}
	}
	body := *loop.Body
	body.Statements = make([]Statement, 0, len(loop.Body.Statements)+1)
	body.Statements = append(body.Statements, Statement{Kind: syntax.KindBinding, Span: loop.NameSpan, Binding: item})
	body.Statements = append(body.Statements, loop.Body.Statements...)
	classic := &ForLoop{
		Span:        loop.Span,
		Initializer: &Statement{Kind: syntax.KindBinding, Span: loop.NameSpan, Binding: index},
		Condition: &Expression{Kind: syntax.KindBinary, Span: loop.View.Span, Text: "<",
			Children: []*Expression{path(byteForIndexName), path(byteForBoundName)}},
		Post: &Statement{Kind: syntax.KindAssignment, Span: loop.NameSpan, Operator: "+=",
			Target: path(byteForIndexName), Expression: literal("1")},
		Body: &body,
	}
	block := &Block{Span: loop.Span, Statements: []Statement{
		{Kind: syntax.KindBinding, Span: loop.NameSpan, Binding: bound},
		{Kind: syntax.KindFor, Span: loop.Span, For: classic},
	}}
	return &Expression{Kind: syntax.KindBlock, Span: loop.Span, Block: block}
}

// Named receivers and field/group paths retain stable storage identity. Bounds retain the ordinary
// expression grammar; semantic analysis materializes the subview exactly once.
func viewForSource(view *Expression) bool {
	if view.Kind == syntax.KindSlice && len(view.Children) == 3 {
		view = view.Children[0]
	}
	if view != nil && view.Kind == syntax.KindPath && len(view.Children) == 0 {
		return true
	}
	for view != nil && view.Kind == syntax.KindGroup && len(view.Children) == 1 {
		view = view.Children[0]
	}
	if view == nil || view.Kind != syntax.KindFieldExpr {
		return false
	}
	for view != nil {
		switch view.Kind {
		case syntax.KindFieldExpr, syntax.KindGroup:
			if len(view.Children) != 1 {
				return false
			}
			view = view.Children[0]
		case syntax.KindPath:
			return len(view.Children) == 0
		default:
			return false
		}
	}
	return false
}

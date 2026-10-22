package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerCall(expression *hir.Expression) (Operand, error) {
	arguments := expression.Children
	if len(arguments) > 0 {
		arguments = arguments[1:]
	}
	argumentSpans := make([]hir.Span, 0, len(arguments))
	argumentValues := make([]Operand, 0, len(arguments))
	hasNestedSideEffect := false
	for index := range arguments {
		argument := &arguments[index]
		hasNestedSideEffect = hasNestedSideEffect || containsCall(argument)
		if argument.Kind == hir.ExpressionCCallback {
			argumentSpans = append(argumentSpans, argument.Span)
			argumentValues = append(argumentValues, Operand{})
			continue
		}
		value, err := l.lowerExpression(argument)
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		argumentSpans = append(argumentSpans, argument.Span)
		_, _, fixedArray := hir.FixedArrayParts(argument.Type)
		if path := l.liveResourcePath(argument); path != nil {
			value = l.materializeMove(path, value, "call-argument", true)
		} else if index+1 < len(arguments) && value.Kind == OperandLocal &&
			(hir.FixedArrayScalarElement(argument.Type) || (fixedArray || l.properties.IsStruct(argument.Type) || l.properties.Enum(argument.Type) != nil) && !l.properties.NeedsDrop(argument.Type)) {
			// Capture copyable by-value arrays, structs and enums as well as scalars before later
			// arguments can overwrite their source locals or nested fields.
			if _, sourceLocal := l.bindings[hir.SymbolID(value.Local)]; sourceLocal {
				value = l.freezeScalarOperand(value, argument.Span)
			}
		}
		argumentValues = append(argumentValues, value)
	}
	// Forming a named static callback address has no source side effect. Materialize
	// it after all user-data evaluation so it belongs to the final foreign-call block.
	for index := range arguments {
		if arguments[index].Kind == hir.ExpressionCCallback {
			value, err := l.lowerExpression(&arguments[index])
			if err != nil {
				return Operand{}, err
			}
			argumentValues[index] = value
		}
	}
	if len(argumentSpans) >= 2 {
		l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
			Expression: expression.Span, Sequence: slices.Clone(argumentSpans), MaterializedBeforeParent: true,
		})
	}
	if hasNestedSideEffect {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "nested-side-effect", RemainingCount: 0})
	}
	callFact := CallFact{
		Call: expression.Span, ArgumentOrder: slices.Clone(argumentSpans), NormalSuccessor: true,
		ForeignABI: expression.ForeignABI,
	}
	if expression.UnsafeScope != nil {
		callFact.UnsafeScope = *expression.UnsafeScope
	}
	l.result.Calls = append(l.result.Calls, callFact)
	destination := NoLocal
	if expression.Type != hir.TypeVoid {
		destination = l.newTemporary()
	}
	continuation := l.newBlock()
	terminator := Terminator{
		Kind: TerminatorCall, Span: expression.Span, Successors: []BlockID{continuation}, UnwindSuccessor: NoBlock,
		Callee: expression.Symbol, Arguments: argumentValues, DestinationLocal: destination,
		ForeignABI: expression.ForeignABI,
	}
	if expression.UnsafeScope != nil {
		terminator.UnsafeScope = *expression.UnsafeScope
		terminator.Entity = sourceEntity("call", expression.Span)
		l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
			Entity:           terminator.Entity,
			SourceSpanEntity: sourceEntity("span", expression.Span),
			ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
		})
	}
	l.terminate(terminator)
	l.current = continuation
	if destination == NoLocal {
		return Operand{}, nil
	}
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func sourceEntity(kind string, span hir.Span) string {
	return fmt.Sprintf("%s:%s:%d:%d", kind, hir.QualifiedSourcePath(span.Package, span.Source), span.StartByte, span.EndByte)
}

package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerMatch(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) == 1 && l.properties.Enum(expression.Children[0].Type) != nil {
		return l.lowerEnumMatch(expression)
	}
	return l.lowerBuiltinMatch(expression)
}

func (l *functionLowerer) lowerBuiltinMatch(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || len(expression.Arms) != 2 {
		return Operand{}, fmt.Errorf("built-in match has %d scrutinees and %d arms", len(expression.Children), len(expression.Arms))
	}
	family := ""
	firstVariant := ""
	secondVariant := ""
	payloadOperation := OperationKind("")
	if _, _, result := hir.ResultElements(expression.Children[0].Type); result {
		family, firstVariant, secondVariant = "result", "ok", "err"
		payloadOperation = OperationResultPayload
	} else if _, option := hir.OptionElement(expression.Children[0].Type); option {
		family, firstVariant, secondVariant = "option", "some", "none"
		payloadOperation = OperationOptionPayload
	} else {
		return Operand{}, fmt.Errorf("match scrutinee type %q is not an implemented built-in tag", expression.Children[0].Type)
	}
	scrutinee, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if scrutinee.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("%s match scrutinee has no value", family)
	}
	if path := l.liveResourcePath(&expression.Children[0]); path != nil {
		scrutinee = l.materializeMove(path, scrutinee, family+"-match", true)
	}
	branchLive := cloneLive(l.live)
	firstBlock := l.newBlock()
	secondBlock := l.newBlock()
	joinBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{firstBlock, secondBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: family, SwitchOperands: []Operand{scrutinee},
	})

	destination := NoLocal
	if expression.Type != hir.TypeVoid {
		destination = l.newTemporary()
	}
	blocks := map[string]BlockID{firstVariant: firstBlock, secondVariant: secondBlock}
	continuing := 0
	var mergedLive map[hir.SymbolID]bool
	seen := make(map[string]struct{}, 2)
	for index := range expression.Arms {
		arm := &expression.Arms[index]
		block, found := blocks[arm.Variant]
		if !found {
			return Operand{}, fmt.Errorf("unsupported %s match variant %q", family, arm.Variant)
		}
		if _, duplicate := seen[arm.Variant]; duplicate {
			return Operand{}, fmt.Errorf("duplicate %s match variant %q", family, arm.Variant)
		}
		seen[arm.Variant] = struct{}{}
		l.current = block
		l.live = cloneLive(branchLive)
		if arm.Binding != hir.NoSymbol {
			binding := l.bindings[arm.Binding]
			if l.properties.NeedsDrop(binding.typ) {
				payload := l.newTemporary()
				l.appendOperation(Operation{
					Kind: payloadOperation, Span: arm.BindingSpan, DestinationLocal: payload,
					Operands: []Operand{scrutinee}, Variant: arm.Variant, MovePayload: true, Type: binding.typ,
				})
				l.appendOperation(Operation{
					Kind: OperationInitialize, Span: arm.BindingSpan, Symbol: arm.Binding,
					DestinationLocal: LocalID(arm.Binding), Operands: []Operand{{Kind: OperandLocal, Local: payload}},
				})
				l.live[arm.Binding] = true
			} else {
				l.appendOperation(Operation{
					Kind: payloadOperation, Span: arm.BindingSpan, DestinationLocal: LocalID(arm.Binding),
					Operands: []Operand{scrutinee}, Variant: arm.Variant, Type: binding.typ,
				})
			}
		} else if family != "option" || arm.Variant != "none" {
			return Operand{}, fmt.Errorf("%s match variant %q is missing its payload binding", family, arm.Variant)
		}
		armValue, lowerErr := l.lowerExpression(&arm.Expression)
		if lowerErr != nil {
			return Operand{}, lowerErr
		}
		if l.blockTerminated(l.current) {
			continue
		}
		if destination != NoLocal {
			if armValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing %s match arm %q has no value", family, arm.Variant)
			}
			if path := l.liveResourcePath(&arm.Expression); l.properties.NeedsDrop(expression.Type) && path != nil {
				armValue = l.materializeMove(path, armValue, family+"-match-result", true)
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: arm.Expression.Span, DestinationLocal: destination, Operands: []Operand{armValue}})
		}
		if arm.Binding != hir.NoSymbol {
			if l.live[arm.Binding] {
				l.appendOperation(Operation{Kind: OperationDrop, Span: arm.BindingSpan, Symbol: arm.Binding, FlagClearedFirst: true})
			}
			delete(l.live, arm.Binding)
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: arm.Expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
		if continuing == 0 {
			mergedLive = cloneLive(l.live)
		} else if !sameLiveState(mergedLive, l.live) {
			return Operand{}, fmt.Errorf("%s match arms disagree on resource state after Flow validation", family)
		}
		continuing++
	}
	if len(seen) != 2 {
		return Operand{}, fmt.Errorf("non-exhaustive %s match", family)
	}
	l.current = joinBlock
	if continuing == 0 {
		l.live = cloneLive(branchLive)
		l.terminate(Terminator{Kind: TerminatorUnreachable, Span: expression.Span, UnwindSuccessor: NoBlock})
		return Operand{}, nil
	}
	l.live = mergedLive
	if destination == NoLocal {
		return Operand{}, nil
	}
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerEnumMatch(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 {
		return Operand{}, fmt.Errorf("enum match has %d scrutinees", len(expression.Children))
	}
	definition := l.properties.Enum(expression.Children[0].Type)
	if definition == nil || len(expression.Arms) != len(definition.Variants) || len(definition.Variants) == 0 {
		return Operand{}, fmt.Errorf("enum match for %q has %d arms, want %d", expression.Children[0].Type, len(expression.Arms), len(definition.Variants))
	}
	scrutinee, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if scrutinee.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("enum match scrutinee has no value")
	}
	branchLive := cloneLive(l.live)
	blocks := make(map[string]BlockID, len(definition.Variants))
	successors := make([]BlockID, 0, len(definition.Variants))
	variants := make([]string, 0, len(definition.Variants))
	for _, variant := range definition.Variants {
		block := l.newBlock()
		blocks[variant.Name] = block
		successors = append(successors, block)
		variants = append(variants, variant.Name)
	}
	joinBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span, Successors: successors, UnwindSuccessor: NoBlock,
		SwitchKind: "enum", SwitchType: definition.Type, SwitchVariants: variants, SwitchOperands: []Operand{scrutinee},
	})
	destination := NoLocal
	if expression.Type != hir.TypeVoid {
		destination = l.newTemporary()
	}
	continuing := 0
	var mergedLive map[hir.SymbolID]bool
	seen := make(map[string]struct{}, len(definition.Variants))
	for index := range expression.Arms {
		arm := &expression.Arms[index]
		block, found := blocks[arm.Variant]
		if !found {
			return Operand{}, fmt.Errorf("enum match has unknown variant %q", arm.Variant)
		}
		if _, duplicate := seen[arm.Variant]; duplicate {
			return Operand{}, fmt.Errorf("enum match has duplicate variant %q", arm.Variant)
		}
		seen[arm.Variant] = struct{}{}
		variant, _ := enumVariantDefinition(definition, arm.Variant)
		l.current = block
		l.live = cloneLive(branchLive)
		bound := make(map[string]struct{}, len(arm.Bindings))
		for _, binding := range arm.Bindings {
			field, declared := enumFieldDefinition(variant, binding.Field)
			if !declared || binding.Symbol == hir.NoSymbol || field.Type != binding.Type {
				return Operand{}, fmt.Errorf("enum match binding %q does not match variant %q", binding.Field, arm.Variant)
			}
			if _, duplicate := bound[binding.Field]; duplicate {
				return Operand{}, fmt.Errorf("enum match binding %q is duplicated", binding.Field)
			}
			bound[binding.Field] = struct{}{}
			l.appendOperation(Operation{
				Kind: OperationEnumPayload, Span: binding.Span, DestinationLocal: LocalID(binding.Symbol),
				Operands: []Operand{scrutinee}, OwnerType: definition.Type, Variant: arm.Variant,
				Field: binding.Field, Type: binding.Type,
			})
		}
		if len(bound) != len(variant.Fields) {
			return Operand{}, fmt.Errorf("enum match variant %q has incomplete payload bindings", arm.Variant)
		}
		armValue, lowerErr := l.lowerExpression(&arm.Expression)
		if lowerErr != nil {
			return Operand{}, lowerErr
		}
		if l.blockTerminated(l.current) {
			continue
		}
		if destination != NoLocal {
			if armValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing enum match arm %q has no value", arm.Variant)
			}
			if path := l.liveResourcePath(&arm.Expression); l.properties.NeedsDrop(expression.Type) && path != nil {
				armValue = l.materializeMove(path, armValue, "enum-match-result", true)
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: arm.Expression.Span, DestinationLocal: destination, Operands: []Operand{armValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: arm.Expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
		if continuing == 0 {
			mergedLive = cloneLive(l.live)
		} else if !sameLiveState(mergedLive, l.live) {
			return Operand{}, fmt.Errorf("enum match arms disagree on resource state after Flow validation")
		}
		continuing++
	}
	if len(seen) != len(definition.Variants) {
		return Operand{}, fmt.Errorf("non-exhaustive enum match")
	}
	l.current = joinBlock
	if continuing == 0 {
		l.live = cloneLive(branchLive)
		l.terminate(Terminator{Kind: TerminatorUnreachable, Span: expression.Span, UnwindSuccessor: NoBlock})
		return Operand{}, nil
	}
	l.live = mergedLive
	if destination == NoLocal {
		return Operand{}, nil
	}
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func enumVariantDefinition(definition *EnumType, name string) (EnumVariant, bool) {
	if definition == nil {
		return EnumVariant{}, false
	}
	for _, variant := range definition.Variants {
		if variant.Name == name {
			return variant, true
		}
	}
	return EnumVariant{}, false
}

func enumFieldDefinition(variant EnumVariant, name string) (EnumField, bool) {
	for _, field := range variant.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return EnumField{}, false
}

func enumFieldNames(fields []EnumField) []string {
	result := make([]string, len(fields))
	for index := range fields {
		result[index] = fields[index].Name
	}
	return result
}

func sameNames(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, name := range left {
		if name == "" {
			return false
		}
		seen[name] = struct{}{}
	}
	if len(seen) != len(left) {
		return false
	}
	for _, name := range right {
		if _, found := seen[name]; !found {
			return false
		}
	}
	return true
}

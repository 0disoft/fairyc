package mir

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/hir"
)

// Lower normalizes typed HIR into executable mutable-local MIR.
func Lower(input hir.Package) (Program, error) {
	structTypes, err := lowerStructTypes(input)
	if err != nil {
		return Program{}, err
	}
	enumTypes, err := lowerEnumTypes(input)
	if err != nil {
		return Program{}, err
	}
	resourceTypes, err := lowerResourceTypes(input)
	if err != nil {
		return Program{}, err
	}
	foreignFunctions, err := lowerForeignFunctions(input)
	if err != nil {
		return Program{}, err
	}
	program := Program{StructTypes: structTypes, EnumTypes: enumTypes, ResourceTypes: resourceTypes, ForeignFunctions: foreignFunctions}
	properties := newTypeProperties(resourceTypes, structTypes, enumTypes)
	for _, module := range input.Modules {
		moduleIdentity := hir.QualifiedModulePath(module.Package, module.Path)
		for index := range module.Functions {
			function := &module.Functions[index]
			lowerer := newFunctionLowerer(moduleIdentity, function, properties)
			lowered, err := lowerer.lower()
			if err != nil {
				return Program{}, fmt.Errorf("lower %s::%s: %w", moduleIdentity, function.Name, err)
			}
			program.Functions = append(program.Functions, lowered)
		}
	}
	slices.SortFunc(program.Functions, func(left, right Function) int {
		if left.Name < right.Name {
			return -1
		}
		if left.Name > right.Name {
			return 1
		}
		return 0
	})
	return program, nil
}

func lowerEnumTypes(input hir.Package) ([]EnumType, error) {
	var result []EnumType
	seen := make(map[hir.Type]struct{})
	for _, module := range input.Modules {
		for _, enumeration := range module.Enums {
			if enumeration.Type == "" || enumeration.Type == hir.TypeInvalid || enumeration.Name == "" {
				return nil, fmt.Errorf("enum %s::%s has no concrete identity", module.Path, enumeration.Name)
			}
			if _, duplicate := seen[enumeration.Type]; duplicate {
				return nil, fmt.Errorf("duplicate enum type %q", enumeration.Type)
			}
			seen[enumeration.Type] = struct{}{}
			definition := EnumType{
				Type: enumeration.Type, Name: enumeration.Name, Span: enumeration.Span,
				Entity: enumTypeEntity(enumeration.Type), Variants: make([]EnumVariant, 0, len(enumeration.Variants)),
			}
			variantNames := make(map[string]struct{}, len(enumeration.Variants))
			for _, variant := range enumeration.Variants {
				if variant.Name == "" {
					return nil, fmt.Errorf("enum %q has an empty variant name", enumeration.Type)
				}
				if _, duplicate := variantNames[variant.Name]; duplicate {
					return nil, fmt.Errorf("enum %q has duplicate variant %q", enumeration.Type, variant.Name)
				}
				variantNames[variant.Name] = struct{}{}
				lowered := EnumVariant{Name: variant.Name, Span: variant.Span, Fields: make([]EnumField, 0, len(variant.Fields))}
				fieldNames := make(map[string]struct{}, len(variant.Fields))
				for _, field := range variant.Fields {
					if field.Name == "" || field.Type == "" || field.Type == hir.TypeInvalid {
						return nil, fmt.Errorf("enum %q variant %q has an invalid field", enumeration.Type, variant.Name)
					}
					if _, duplicate := fieldNames[field.Name]; duplicate {
						return nil, fmt.Errorf("enum %q variant %q has duplicate field %q", enumeration.Type, variant.Name, field.Name)
					}
					fieldNames[field.Name] = struct{}{}
					lowered.Fields = append(lowered.Fields, EnumField{Name: field.Name, Type: field.Type, Span: field.Span})
				}
				definition.Variants = append(definition.Variants, lowered)
			}
			result = append(result, definition)
		}
	}
	slices.SortFunc(result, func(left, right EnumType) int {
		return strings.Compare(string(left.Type), string(right.Type))
	})
	return result, nil
}

func lowerStructTypes(input hir.Package) ([]StructType, error) {
	var result []StructType
	seen := make(map[hir.Type]struct{})
	for _, module := range input.Modules {
		for _, structure := range module.Structs {
			if structure.Type == "" || structure.Type == hir.TypeInvalid || structure.Name == "" {
				return nil, fmt.Errorf("struct %s::%s has no concrete identity", module.Path, structure.Name)
			}
			if _, duplicate := seen[structure.Type]; duplicate {
				return nil, fmt.Errorf("duplicate struct type %q", structure.Type)
			}
			seen[structure.Type] = struct{}{}
			definition := StructType{
				Type: structure.Type, Name: structure.Name, Span: structure.Span,
				Entity: structTypeEntity(structure.Type), ReprC: structure.ReprC,
				Fields: make([]StructField, 0, len(structure.Fields)),
			}
			fieldNames := make(map[string]struct{}, len(structure.Fields))
			for _, field := range structure.Fields {
				if field.Name == "" || field.Type == "" || field.Type == hir.TypeInvalid {
					return nil, fmt.Errorf("struct %q has an invalid field definition", structure.Type)
				}
				if _, duplicate := fieldNames[field.Name]; duplicate {
					return nil, fmt.Errorf("struct %q has duplicate field %q", structure.Type, field.Name)
				}
				fieldNames[field.Name] = struct{}{}
				definition.Fields = append(definition.Fields, StructField{Name: field.Name, Type: field.Type, Span: field.Span})
			}
			result = append(result, definition)
		}
	}
	slices.SortFunc(result, func(left, right StructType) int {
		return strings.Compare(string(left.Type), string(right.Type))
	})
	return result, nil
}

func lowerForeignFunctions(input hir.Package) ([]ForeignFunction, error) {
	var result []ForeignFunction
	seen := make(map[hir.SymbolID]struct{})
	for _, module := range input.Modules {
		moduleIdentity := hir.QualifiedModulePath(module.Package, module.Path)
		for _, function := range module.ForeignFunctions {
			if function.Symbol == hir.NoSymbol || function.Name == "" || function.LinkName == "" || function.ABI == "" {
				return nil, fmt.Errorf("foreign function %s::%s has an incomplete declaration", module.Path, function.Name)
			}
			if _, duplicate := seen[function.Symbol]; duplicate {
				return nil, fmt.Errorf("duplicate foreign function symbol %d", function.Symbol)
			}
			seen[function.Symbol] = struct{}{}
			lowered := ForeignFunction{
				Symbol: function.Symbol, Name: moduleIdentity + "::" + function.Name,
				LinkName: function.LinkName, ABI: function.ABI, Span: function.Span,
				ResultType: function.ResultType,
				Entity:     sourceEntity("foreign", function.Span),
			}
			for _, parameter := range function.Parameters {
				lowered.ParameterTypes = append(lowered.ParameterTypes, parameter.Type)
			}
			result = append(result, lowered)
		}
	}
	slices.SortFunc(result, func(left, right ForeignFunction) int {
		if value := strings.Compare(left.LinkName, right.LinkName); value != 0 {
			return value
		}
		return int(left.Symbol - right.Symbol)
	})
	return result, nil
}

func lowerResourceTypes(input hir.Package) ([]ResourceType, error) {
	var result []ResourceType
	seen := make(map[hir.Type]struct{})
	for _, module := range input.Modules {
		for _, resource := range module.Resources {
			if resource.Type == "" || resource.Type == hir.TypeInvalid {
				return nil, fmt.Errorf("resource %s::%s has no concrete type", module.Path, resource.Name)
			}
			if _, duplicate := seen[resource.Type]; duplicate {
				return nil, fmt.Errorf("duplicate resource type %q", resource.Type)
			}
			seen[resource.Type] = struct{}{}
			if resource.DropKind != hir.ResourceDropEmpty && resource.DropKind != hir.ResourceDropNonEmpty {
				return nil, fmt.Errorf("resource %q has unsupported drop kind %q", resource.Type, resource.DropKind)
			}
			if resource.DropKind == hir.ResourceDropEmpty && resource.DropSymbol != hir.NoSymbol ||
				resource.DropKind == hir.ResourceDropNonEmpty && resource.DropSymbol == hir.NoSymbol {
				return nil, fmt.Errorf("resource %q has inconsistent drop symbol %d for kind %q", resource.Type, resource.DropSymbol, resource.DropKind)
			}
			definition := ResourceType{
				Type: resource.Type, Span: resource.Span, Entity: resourceTypeEntity(resource.Type),
				DropKind: resource.DropKind, DropSpan: resource.DropSpan, DropSymbol: resource.DropSymbol,
				DropEffects: slices.Clone(resource.DropEffects),
				Fields:      make([]ResourceField, 0, len(resource.Fields)),
			}
			fieldNames := make(map[string]struct{}, len(resource.Fields))
			for _, field := range resource.Fields {
				if field.Name == "" || field.Type == "" || field.Type == hir.TypeInvalid {
					return nil, fmt.Errorf("resource %q has an invalid field definition", resource.Type)
				}
				if _, duplicate := fieldNames[field.Name]; duplicate {
					return nil, fmt.Errorf("resource %q has duplicate field %q", resource.Type, field.Name)
				}
				fieldNames[field.Name] = struct{}{}
				definition.Fields = append(definition.Fields, ResourceField{Name: field.Name, Type: field.Type, Span: field.Span})
			}
			result = append(result, definition)
		}
	}
	slices.SortFunc(result, func(left, right ResourceType) int {
		return strings.Compare(string(left.Type), string(right.Type))
	})
	return result, nil
}

type bindingInfo struct {
	typ  hir.Type
	span hir.Span
}

type activeRegion struct {
	symbol    hir.SymbolID
	span      hir.Span
	factIndex int
}

type activeLoop struct {
	header         BlockID
	continueTarget BlockID
	exit           BlockID
	body           hir.Span
	regionDepth    int
	headerLive     map[hir.SymbolID]bool
}

type deferredCall struct {
	scope         hir.Span
	span          hir.Span
	callee        hir.SymbolID
	arguments     []Operand
	argumentSpans []hir.Span
}

type functionLowerer struct {
	module       string
	source       *hir.Function
	properties   typeProperties
	result       Function
	current      BlockID
	bindings     map[hir.SymbolID]bindingInfo
	live         map[hir.SymbolID]bool
	regions      map[hir.SymbolID]int
	regionStack  []activeRegion
	loopStack    []activeLoop
	currentScope hir.Span
	deferred     []deferredCall
	nextLocal    LocalID
}

func newFunctionLowerer(module string, function *hir.Function, properties typeProperties) *functionLowerer {
	result := Function{
		Symbol:      function.Symbol,
		Name:        module + "::" + function.Name,
		Span:        function.Span,
		ResultType:  function.ResultType,
		Effects:     slices.Clone(function.Effects),
		Entry:       0,
		DropGlueFor: function.DropGlueFor,
	}
	return &functionLowerer{
		module:     module,
		source:     function,
		properties: properties,
		result:     result,
		bindings:   make(map[hir.SymbolID]bindingInfo),
		live:       make(map[hir.SymbolID]bool),
		regions:    make(map[hir.SymbolID]int),
	}
}

func (l *functionLowerer) lower() (Function, error) {
	l.newBlock()
	maximumSymbol := hir.SymbolID(0)
	for index, parameter := range l.source.Parameters {
		maximumSymbol = max(maximumSymbol, parameter.Symbol)
		l.result.Parameters = append(l.result.Parameters, LocalID(parameter.Symbol))
		l.result.ParameterTypes = append(l.result.ParameterTypes, parameter.Type)
		l.bindings[parameter.Symbol] = bindingInfo{typ: parameter.Type, span: parameter.Span}
		isDropReceiver := l.source.DropGlueFor != "" && index == 0 && parameter.Type == l.source.DropGlueFor
		if l.properties.NeedsDrop(parameter.Type) && !isDropReceiver {
			l.live[parameter.Symbol] = true
			l.result.Resources = append(l.result.Resources, ResourceLocal{
				Symbol: parameter.Symbol, Type: parameter.Type, Binding: parameter.Span, Parameter: true,
			})
		}
	}
	for _, local := range l.source.Locals {
		maximumSymbol = max(maximumSymbol, local.Symbol)
		l.bindings[local.Symbol] = bindingInfo{typ: local.Type, span: local.Span}
		if l.properties.NeedsDrop(local.Type) {
			l.result.Resources = append(l.result.Resources, ResourceLocal{
				Symbol: local.Symbol, Type: local.Type, Binding: local.Span,
			})
		}
	}
	l.nextLocal = LocalID(maximumSymbol + 1)

	terminated, err := l.lowerBlock(l.source.Body)
	if err != nil {
		return Function{}, err
	}
	if !terminated && l.source.ResultType == hir.TypeVoid {
		actions := l.emitDeferredCallsForScope(l.source.Body.Span)
		actions = append(actions, l.emitFunctionExitCleanup(l.live, l.source.Body.Span, "return")...)
		l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: l.source.Body.Span, ExitKind: "return", Actions: actions})
		if cleanupContainsDrop(actions) {
			l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
		}
		l.terminate(Terminator{Kind: TerminatorReturn, Span: l.source.Body.Span, UnwindSuccessor: NoBlock})
		terminated = true
	}
	if !terminated {
		return Function{}, fmt.Errorf("function has no lowered return")
	}
	return l.result, nil
}

func (l *functionLowerer) lowerBlock(block hir.Block) (bool, error) {
	previousScope := l.currentScope
	l.currentScope = block.Span
	defer func() { l.currentScope = previousScope }()
	terminated, err := l.lowerStatements(block.Statements)
	if err != nil || terminated {
		return terminated, err
	}
	if block.Tail != nil {
		if err := l.lowerReturn(block.Tail, block.Tail.Span); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (l *functionLowerer) lowerStatements(statements []hir.Statement) (bool, error) {
	for _, statement := range statements {
		if l.blockTerminated(l.current) {
			return true, nil
		}
		switch statement.Kind {
		case "binding":
			value, err := l.lowerExpression(statement.Expression)
			if err != nil {
				return false, err
			}
			binding, found := l.bindings[statement.Binding]
			if found && l.properties.NeedsDrop(binding.typ) {
				if value.Kind == OperandInvalid {
					return false, fmt.Errorf("resource binding %d has no value", statement.Binding)
				}
				if path := l.liveResourcePath(statement.Expression); path != nil {
					value = l.materializeMove(path, value, "binding-value", true)
				}
				l.appendOperation(Operation{
					Kind: OperationInitialize, Span: binding.span, Symbol: statement.Binding,
					DestinationLocal: LocalID(statement.Binding), Operands: []Operand{value},
				})
				l.live[statement.Binding] = true
			} else if value.Kind != OperandInvalid {
				l.appendOperation(Operation{Kind: OperationAssign, Span: statement.Span, DestinationLocal: LocalID(statement.Binding), Operands: []Operand{value}})
			}
		case "return":
			if err := l.lowerReturn(statement.Expression, expressionSpan(statement.Expression, statement.Span)); err != nil {
				return false, err
			}
			return true, nil
		case "discard", "expression-statement":
			if err := l.lowerIgnoredExpression(statement.Expression); err != nil {
				return false, err
			}
		case "burn":
			if statement.Expression == nil || statement.Expression.Kind != "path" {
				return false, fmt.Errorf("burn requires one direct resource path")
			}
			binding, found := l.bindings[statement.Expression.Symbol]
			if !found || !l.properties.NeedsDrop(binding.typ) {
				return false, fmt.Errorf("burn names non-resource local %d", statement.Expression.Symbol)
			}
			if !l.live[statement.Expression.Symbol] {
				return false, fmt.Errorf("burn names discharged resource local %d", statement.Expression.Symbol)
			}
			l.appendOperation(Operation{
				Kind: OperationBurn, Span: statement.Span, Symbol: statement.Expression.Symbol,
				DestinationLocal: NoLocal, Type: binding.typ,
			})
			l.live[statement.Expression.Symbol] = false
		case "assignment":
			if statement.Target != nil && statement.Target.Kind == "prefix" && statement.Target.Operator == "*" {
				if err := l.lowerDerefStore(statement); err != nil {
					return false, err
				}
				continue
			}
			if statement.Target != nil && statement.Target.Kind == "field-expression" {
				if err := l.lowerFieldStore(statement); err != nil {
					return false, err
				}
				continue
			}
			if statement.Target != nil && (statement.Target.Kind == "array-index" || statement.Target.Kind == "view-index") {
				if err := l.lowerIndexedAssignment(statement); err != nil {
					return false, err
				}
				continue
			}
			operator := statement.Operator
			if operator == "" {
				operator = "="
			}
			var value Operand
			var err error
			if operator != "=" && statement.Target != nil && statement.Expression != nil {
				binary, ok := hir.CompoundAssignmentBinary(operator)
				if !ok || statement.Target.Kind != "path" || !hir.CompoundAssignmentType(operator, statement.Target.Type) {
					return false, fmt.Errorf("malformed compound assignment %q", operator)
				}
				prior, lowerErr := l.lowerExpression(statement.Target)
				if lowerErr != nil {
					return false, lowerErr
				}
				prior = l.freezeScalarOperand(prior, statement.Target.Span)
				right, lowerErr := l.lowerExpression(statement.Expression)
				if lowerErr != nil {
					return false, lowerErr
				}
				combined := &hir.Expression{
					Kind: "binary", Type: statement.Target.Type, Span: statement.Span, Operator: binary,
					Children: []hir.Expression{*statement.Target, *statement.Expression},
				}
				value, err = l.lowerBinaryOperands(combined, prior, right)
			} else {
				value, err = l.lowerExpression(statement.Expression)
			}
			if err != nil {
				return false, err
			}
			if statement.Target == nil || statement.Target.Kind != "path" || value.Kind == OperandInvalid {
				continue
			}
			binding, found := l.bindings[statement.Target.Symbol]
			if found && l.properties.NeedsDrop(binding.typ) {
				if path := l.liveResourcePath(statement.Expression); path != nil {
					value = l.materializeMove(path, value, "reassignment-value", true)
				}
				droppedOld := l.live[statement.Target.Symbol]
				if droppedOld {
					l.appendOperation(Operation{Kind: OperationDrop, Span: binding.span, Symbol: statement.Target.Symbol, FlagClearedFirst: true})
					l.live[statement.Target.Symbol] = false
				}
				l.appendOperation(Operation{
					Kind: OperationInitialize, Span: statement.Target.Span, Symbol: statement.Target.Symbol,
					DestinationLocal: LocalID(statement.Target.Symbol), Operands: []Operand{value},
				})
				l.live[statement.Target.Symbol] = true
				l.result.Reassignments = append(l.result.Reassignments, ReassignmentFact{
					Binding: binding.span, Target: statement.Target.Span, Value: statement.Expression.Span,
					ValueBeforeDrop: true, DroppedOld: droppedOld, InstalledNew: true,
				})
			} else {
				l.appendOperation(Operation{Kind: OperationAssign, Span: statement.Span, DestinationLocal: LocalID(statement.Target.Symbol), Operands: []Operand{value}})
			}
		case "assert":
			if err := l.lowerAssert(statement.Expression, statement.Span); err != nil {
				return false, err
			}
		case "defer":
			if err := l.lowerDefer(statement); err != nil {
				return false, err
			}
		case "while":
			if _, err := l.lowerWhile(statement.Expression); err != nil {
				return false, err
			}
		case "for":
			if err := l.lowerFor(statement.For); err != nil {
				return false, err
			}
		case "break", "continue":
			if err := l.lowerLoopControl(statement.Kind, statement.Span); err != nil {
				return false, err
			}
			return true, nil
		case "region":
			if _, err := l.lowerExpression(statement.Expression); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("unsupported HIR statement %q", statement.Kind)
		}
	}
	return l.blockTerminated(l.current), nil
}

func (l *functionLowerer) lowerIndexedAssignment(statement hir.Statement) error {
	target := statement.Target
	operator := statement.Operator
	if operator == "" {
		operator = "="
	}
	if target == nil || statement.Expression == nil || len(target.Children) != 2 || target.Operator != "[]" ||
		target.Children[1].Type != hir.TypeUSize || statement.Expression.Type != target.Type {
		return fmt.Errorf("malformed indexed assignment")
	}
	if operator != "=" && !hir.CompoundAssignmentType(operator, target.Type) {
		return fmt.Errorf("malformed indexed compound assignment %q", operator)
	}
	receiver, err := l.lowerExpression(&target.Children[0])
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	index, err := l.lowerExpression(&target.Children[1])
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	if receiver.Kind == OperandInvalid || index.Kind == OperandInvalid {
		return fmt.Errorf("indexed assignment has an invalid place operand")
	}
	checkKind := OperationArrayIndexCheck
	storeKind := OperationArrayIndexStore
	switchKind := "checked-array-index"
	if target.Kind == "view-index" {
		element, indexed := hir.IndexedViewElement(target.Children[0].Type)
		if !indexed || !hir.MutableIndexedView(target.Children[0].Type) || target.Type != element {
			return fmt.Errorf("mutable byte-view assignment has incompatible type metadata")
		}
		checkKind = OperationViewIndexCheck
		storeKind = OperationViewIndexStore
		switchKind = "checked-view-index"
		receiver = l.freezeScalarOperand(receiver, target.Children[0].Span)
	} else {
		elementType, _, array := hir.FixedArrayParts(target.Children[0].Type)
		if target.Kind != "array-index" || !array || elementType != target.Type {
			return fmt.Errorf("fixed-array assignment has incompatible type metadata")
		}
	}
	index = l.freezeScalarOperand(index, target.Children[1].Span)
	placeOperands := []Operand{receiver, index}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression:               statement.Span,
		Sequence:                 []hir.Span{target.Children[0].Span, target.Children[1].Span, statement.Expression.Span},
		MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: checkKind, Span: target.Span, DestinationLocal: NoLocal, Operands: slices.Clone(placeOperands),
		OwnerType: target.Children[0].Type, IndexType: hir.TypeUSize, Type: target.Type, Operator: "[]",
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: target.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: switchKind, SwitchOperands: slices.Clone(placeOperands), Operator: "[]",
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: target.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: target.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	var prior Operand
	if operator != "=" {
		loadKind := OperationArrayIndex
		if target.Kind == "view-index" {
			loadKind = OperationViewIndex
		}
		destination := l.newTemporary()
		l.appendOperation(Operation{
			Kind: loadKind, Span: target.Span, DestinationLocal: destination, Operands: slices.Clone(placeOperands),
			OwnerType: target.Children[0].Type, IndexType: hir.TypeUSize, Type: target.Type, Operator: "[]",
		})
		prior = Operand{Kind: OperandLocal, Local: destination}
	}
	value, err := l.lowerExpression(statement.Expression)
	if err != nil {
		return err
	}
	// A terminating RHS has no stored value and no normal continuation.
	if l.blockTerminated(l.current) {
		return nil
	}
	if value.Kind == OperandInvalid {
		return fmt.Errorf("indexed assignment has an invalid value operand")
	}
	if operator != "=" {
		binary, _ := hir.CompoundAssignmentBinary(operator)
		combined := &hir.Expression{
			Kind: "binary", Type: target.Type, Span: statement.Span, Operator: binary,
			Children: []hir.Expression{{Type: target.Type, Span: target.Span}, *statement.Expression},
		}
		value, err = l.lowerBinaryOperands(combined, prior, value)
		if err != nil {
			return err
		}
	}
	l.appendOperation(Operation{
		Kind: storeKind, Span: statement.Span, DestinationLocal: NoLocal,
		Operands: append(slices.Clone(placeOperands), value), OwnerType: target.Children[0].Type,
		IndexType: hir.TypeUSize, Type: target.Type, Operator: "[]=",
	})
	return nil
}

func (l *functionLowerer) freezeScalarOperand(value Operand, at hir.Span) Operand {
	if value.Kind == OperandInvalid {
		return value
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: at, DestinationLocal: destination, Operands: []Operand{value}})
	return Operand{Kind: OperandLocal, Local: destination}
}

func (l *functionLowerer) lowerDefer(statement hir.Statement) error {
	expression := statement.Expression
	if expression == nil || expression.Kind != "call" || expression.Type != hir.TypeVoid ||
		expression.Symbol == hir.NoSymbol || expression.ForeignABI != "" || len(expression.Children) == 0 || l.currentScope.Source == "" {
		return fmt.Errorf("malformed defer statement")
	}
	arguments := make([]Operand, 0, len(expression.Children)-1)
	argumentSpans := make([]hir.Span, 0, len(expression.Children)-1)
	for index := 1; index < len(expression.Children); index++ {
		argument := &expression.Children[index]
		if !hir.FixedArrayScalarElement(argument.Type) {
			return fmt.Errorf("defer argument %d has unsupported type %q", index-1, argument.Type)
		}
		value, err := l.lowerExpression(argument)
		if err != nil {
			return err
		}
		if l.blockTerminated(l.current) {
			return nil
		}
		if value.Kind == OperandInvalid {
			return fmt.Errorf("defer argument %d has no executable value", index-1)
		}
		arguments = append(arguments, l.freezeScalarOperand(value, argument.Span))
		argumentSpans = append(argumentSpans, argument.Span)
	}
	l.deferred = append(l.deferred, deferredCall{
		scope: l.currentScope, span: expression.Span, callee: expression.Symbol,
		arguments: arguments, argumentSpans: argumentSpans,
	})
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "lexical-defer", RemainingCount: 0})
	return nil
}

func (l *functionLowerer) lowerAssert(expression *hir.Expression, statementSpan hir.Span) error {
	if expression == nil || expression.Type != hir.TypeBool {
		return fmt.Errorf("malformed assert statement")
	}
	if expression.Kind == "literal" && expression.Value == "false" {
		l.terminate(Terminator{Kind: TerminatorTrap, Span: statementSpan, UnwindSuccessor: NoBlock, TrapCategories: []string{"assertion-failed"}})
		l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
			Operation: statementSpan, Categories: []string{"assertion-failed"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
		})
		return nil
	}
	condition, err := l.lowerExpression(expression)
	if err != nil {
		return err
	}
	if condition.Kind == OperandInvalid {
		return fmt.Errorf("assert statement has no condition value")
	}
	condition.Type = hir.TypeBool
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: statementSpan, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "assert", SwitchType: hir.TypeBool, SwitchOperands: []Operand{condition}, Operator: "assert",
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: statementSpan, UnwindSuccessor: NoBlock, TrapCategories: []string{"assertion-failed"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: statementSpan, Categories: []string{"assertion-failed"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	return nil
}

func (l *functionLowerer) lowerNestedBlock(block hir.Block, ownsCleanup bool) (Operand, bool, error) {
	previousScope := l.currentScope
	l.currentScope = block.Span
	defer func() { l.currentScope = previousScope }()
	terminated, err := l.lowerStatements(block.Statements)
	if err != nil || terminated {
		return Operand{}, terminated, err
	}
	if block.Tail == nil {
		l.emitDeferredCallsForScope(block.Span)
		if ownsCleanup {
			l.finishValueBlockCleanup(block.Span)
		}
		return Operand{}, false, nil
	}
	value, err := l.lowerExpression(block.Tail)
	if err == nil && !l.blockTerminated(l.current) {
		if ownsCleanup && l.hasValueBlockCleanup(block.Span) {
			if path := l.liveResourcePath(block.Tail); path != nil {
				value = l.materializeMove(path, value, "block-result", true)
			} else {
				value = l.freezeScalarOperand(value, block.Tail.Span)
			}
		}
		l.emitDeferredCallsForScope(block.Span)
		if ownsCleanup {
			l.finishValueBlockCleanup(block.Span)
		}
	}
	return value, l.blockTerminated(l.current), err
}

func (l *functionLowerer) lowerExpression(expression *hir.Expression) (Operand, error) {
	if expression == nil {
		return Operand{}, nil
	}
	switch expression.Kind {
	case "literal":
		if expression.Type == hir.SharedBorrow(hir.TypeStr) && strings.HasPrefix(expression.Value, "\"") {
			decoded, err := hex.DecodeString(expression.Cooked)
			if err != nil || !utf8.Valid(decoded) {
				return Operand{}, fmt.Errorf("invalid cooked UTF-8 string literal %q", expression.Value)
			}
			return Operand{Kind: OperandString, Type: expression.Type, StaticData: string(decoded)}, nil
		}
		if expression.Type == hir.SharedBorrow(hir.TypeByteSlice) && strings.HasPrefix(expression.Value, "b\"") {
			decoded, err := hex.DecodeString(expression.Cooked)
			if err != nil {
				return Operand{}, fmt.Errorf("invalid cooked byte-string literal %q", expression.Value)
			}
			return Operand{Kind: OperandByteString, Type: expression.Type, StaticData: string(decoded)}, nil
		}
		if expression.Type == hir.SharedBorrow(hir.TypeCStr) && strings.HasPrefix(expression.Value, "c\"") {
			decoded, err := hex.DecodeString(expression.Cooked)
			if err != nil || !validCStringStaticData(string(decoded)) {
				return Operand{}, fmt.Errorf("invalid cooked C string literal %q", expression.Value)
			}
			return Operand{Kind: OperandCString, Type: expression.Type, StaticData: string(decoded)}, nil
		}
		if expression.Type == hir.TypeBool {
			switch expression.Value {
			case "true":
				return Operand{Kind: OperandI32, I32: 1, Type: hir.TypeBool}, nil
			case "false":
				return Operand{Kind: OperandI32, I32: 0, Type: hir.TypeBool}, nil
			default:
				return Operand{}, fmt.Errorf("invalid bool literal %q", expression.Value)
			}
		}
		switch expression.Type {
		case hir.TypeI64:
			value, err := parseSigned(expression.Value, 64)
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandI64, I64: value}, nil
		case hir.TypeI32:
			value, err := parseI32(expression.Value)
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandI32, I32: value}, nil
		case hir.TypeU8:
			value, err := parseUnsigned(expression.Value, 8, "u8")
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandU8, U64: value}, nil
		case hir.TypeU32:
			value, err := parseUnsigned(expression.Value, 32, "u32")
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandU32, U64: value}, nil
		case hir.TypeUSize:
			value, err := parseUnsigned(expression.Value, 64, "usize")
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandUSize, U64: value}, nil
		default:
			if _, _, pointer := hir.RawPointerElement(expression.Type); pointer && expression.Value == "null" {
				return Operand{Kind: OperandNull, Type: expression.Type}, nil
			}
			return Operand{}, fmt.Errorf("unsupported literal %q of type %q", expression.Value, expression.Type)
		}
	case "path":
		return Operand{Kind: OperandLocal, Local: LocalID(expression.Symbol)}, nil
	case "group":
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("group expression has %d children", len(expression.Children))
		}
		return l.lowerExpression(&expression.Children[0])
	case "record":
		return l.lowerAggregateConstruction(expression)
	case "array":
		return l.lowerArrayConstruction(expression)
	case "array-view-borrow":
		return l.lowerArrayBorrowView(expression)
	case "array-length":
		return l.lowerArrayLength(expression)
	case "storage-layout":
		return l.lowerStorageLayout(expression)
	case "storage-take":
		return l.lowerStorageTake(expression)
	case "storage-put":
		return l.lowerStoragePut(expression)
	case "storage-borrow", "storage-borrow-mut":
		return l.lowerStorageBorrow(expression)
	case "storage-element-address":
		return l.lowerStorageElementAddress(expression)
	case "field-expression":
		return l.lowerFieldRead(expression)
	case "field-borrow":
		return l.lowerFieldBorrow(expression)
	case "view-length":
		return l.lowerViewLength(expression)
	case "view-empty":
		return l.lowerViewEmpty(expression)
	case "view-index", "view-element-borrow":
		return l.lowerViewIndex(expression)
	case "array-index":
		return l.lowerArrayIndex(expression)
	case "view-slice":
		return l.lowerViewSlice(expression)
	case "scalar-cast":
		return l.lowerScalarCast(expression)
	case "shared-reborrow":
		return l.lowerSharedReborrow(expression)
	case "cast":
		return l.lowerRawAddressCast(expression)
	case "borrowed-view-from-raw-parts":
		return l.lowerBorrowedViewFromRawParts(expression)
	case "prefix":
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("prefix expression has %d children", len(expression.Children))
		}
		operand, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		if expression.Operator == "-" {
			l.lowerCheckedArithmetic(expression, []Operand{operand}, "checked-unary", []string{"integer-overflow"})
		}
		destination := l.newTemporary()
		l.appendOperation(Operation{
			Kind: OperationUnary, Span: expression.Span, DestinationLocal: destination,
			Operands: []Operand{operand}, OwnerType: expression.Children[0].Type,
			Type: expression.Type, Operator: expression.Operator,
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	case "call":
		if expression.Operator == "place" && len(expression.Children) == 2 && expression.Children[0].Type == hir.TypeRegion {
			return l.lowerRegionPlace(expression)
		}
		return l.lowerCall(expression)
	case "unsafe":
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("unsafe expression has %d children", len(expression.Children))
		}
		return l.lowerExpression(&expression.Children[0])
	case "result-variant":
		return l.lowerResultVariant(expression)
	case "option-variant":
		return l.lowerOptionVariant(expression)
	case "enum-variant":
		return l.lowerEnumVariant(expression)
	case "binary":
		if len(expression.Children) != 2 {
			return Operand{}, fmt.Errorf("binary expression has %d children", len(expression.Children))
		}
		if expression.Operator == "&&" || expression.Operator == "||" {
			return l.lowerShortCircuit(expression)
		}
		left, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		// Capture the read before a later operand can mutate its source local.
		if _, sourceLocal := l.bindings[hir.SymbolID(left.Local)]; left.Kind == OperandLocal && sourceLocal {
			left = l.freezeScalarOperand(left, expression.Children[0].Span)
		}
		right, err := l.lowerExpression(&expression.Children[1])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		return l.lowerBinaryOperands(expression, left, right)
	case "propagate":
		return l.lowerPropagate(expression)
	case "if":
		return l.lowerIf(expression)
	case "match":
		return l.lowerMatch(expression)
	case "while":
		return l.lowerWhile(expression)
	case "block":
		if expression.Block == nil {
			return Operand{}, fmt.Errorf("block expression has no block")
		}
		value, _, err := l.lowerNestedBlock(*expression.Block, true)
		return value, err
	case "region":
		return l.lowerRegion(expression)
	default:
		return Operand{}, fmt.Errorf("unsupported HIR expression %q", expression.Kind)
	}
}

func (l *functionLowerer) lowerBinaryOperands(expression *hir.Expression, left, right Operand) (Operand, error) {
	if expression == nil || len(expression.Children) != 2 || expression.Type == hir.TypeInvalid {
		return Operand{}, fmt.Errorf("malformed binary operation")
	}
	if expression.Operator == "/" || expression.Operator == "%" {
		if err := l.lowerCheckedDivision(expression, left, right); err != nil {
			return Operand{}, err
		}
	} else {
		switch expression.Operator {
		case "+", "-", "*":
			l.lowerCheckedArithmetic(expression, []Operand{left, right}, "checked-arithmetic", []string{"integer-overflow"})
		case ">>":
			l.lowerCheckedArithmetic(expression, []Operand{left, right}, "checked-shift", []string{"invalid-shift"})
		case "<<":
			l.lowerCheckedArithmetic(expression, []Operand{left, right}, "checked-shift", []string{"invalid-shift", "integer-overflow"})
		}
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationBinary, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{left, right}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerArrayConstruction(expression *hir.Expression) (Operand, error) {
	elementType, length, array := hir.FixedArrayParts(expression.Type)
	validForm := expression.Operator == "list" && uint64(len(expression.Children)) == length ||
		expression.Operator == "repeat" && len(expression.Children) == 1
	if !array || !validForm || !hir.ExecutableFixedArrayShape(elementType, length) {
		return Operand{}, fmt.Errorf("malformed fixed-array construction")
	}
	operands := make([]Operand, 0, len(expression.Children))
	sequence := make([]hir.Span, 0, len(expression.Children))
	for index := range expression.Children {
		if expression.Children[index].Type != elementType {
			return Operand{}, fmt.Errorf("fixed-array element %d has type %q, want %q", index, expression.Children[index].Type, elementType)
		}
		operand, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		if operand.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("fixed-array element %d has no executable value", index)
		}
		if operand.Kind == OperandLocal {
			operand = l.freezeScalarOperand(operand, expression.Children[index].Span)
		}
		operands = append(operands, operand)
		sequence = append(sequence, expression.Children[index].Span)
	}
	if expression.Operator == "repeat" {
		value := operands[0]
		operands = make([]Operand, int(length))
		for index := range operands {
			operands[index] = value
		}
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Type: expression.Type, OwnerType: elementType, Operator: "list",
	})
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: sequence, MaterializedBeforeParent: true,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerArrayBorrowView(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || !hir.ArrayViewBorrow(expression.Children[0].Type, expression.Type, expression.Operator) {
		return Operand{}, fmt.Errorf("malformed fixed-array view borrow")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("fixed-array byte-view borrow requires one local owner")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayBorrowView, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerArrayLength(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator != "len" || expression.Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed fixed-array length")
	}
	elementType, length, array := hir.FixedArrayParts(expression.Children[0].Type)
	if !array || !hir.ExecutableFixedArrayShape(elementType, length) {
		return Operand{}, fmt.Errorf("fixed-array length has unsupported owner type %q", expression.Children[0].Type)
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("fixed-array length has no receiver value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayLength, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerArrayIndex(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 2 || expression.Operator != "[]" || expression.Children[1].Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed fixed-array index")
	}
	elementType, _, array := hir.FixedArrayParts(expression.Children[0].Type)
	if !array || expression.Type != elementType {
		return Operand{}, fmt.Errorf("fixed-array index result type %q does not match owner %q", expression.Type, expression.Children[0].Type)
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	index, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid || index.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("fixed-array index has an invalid operand")
	}
	operands := []Operand{receiver, index}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: []hir.Span{expression.Children[0].Span, expression.Children[1].Span}, MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: OperationArrayIndexCheck, Span: expression.Span, DestinationLocal: NoLocal, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, IndexType: hir.TypeUSize, Type: elementType, Operator: expression.Operator,
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-array-index", SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayIndex, Span: expression.Span, DestinationLocal: destination, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, IndexType: hir.TypeUSize, Type: elementType, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerScalarCast(expression *hir.Expression) (Operand, error) {
	if expression.Operator != "as" || expression.TargetType != expression.Type || len(expression.Children) != 1 ||
		!hir.ValuePreservingIntegerCast(expression.Children[0].Type, expression.Type) {
		return Operand{}, fmt.Errorf("malformed value-preserving integer cast")
	}
	operand, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if operand.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("value-preserving integer cast has no operand value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationScalarCast, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{operand}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerRawAddressCast(expression *hir.Expression) (Operand, error) {
	if expression.Operator != "as" || expression.TargetType != expression.Type || expression.UnsafeScope == nil || len(expression.Children) != 1 {
		return Operand{}, fmt.Errorf("malformed raw-address cast")
	}
	borrow := &expression.Children[0]
	if borrow.Type == hir.SharedBorrow(hir.TypeStr) && expression.Type == hir.RawPointer(hir.TypeU8, false) {
		value, err := l.lowerExpression(borrow)
		if err != nil {
			return Operand{}, err
		}
		if value.Kind != OperandString && value.Kind != OperandLocal {
			return Operand{}, fmt.Errorf("string data address source has invalid operand kind %q", value.Kind)
		}
		destination := l.newTemporary()
		entity := sourceEntity("str-data-address", expression.Span)
		l.appendOperation(Operation{
			Kind: OperationStrDataAddress, Entity: entity, Span: expression.Span,
			DestinationLocal: destination,
			Operands:         []Operand{value},
			Type:             expression.Type,
			OwnerType:        borrow.Type,
			Operator:         expression.Operator,
			UnsafeScope:      *expression.UnsafeScope,
		})
		l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
			Entity:           entity,
			SourceSpanEntity: sourceEntity("span", expression.Span),
			ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	}
	byteSliceElement, byteSliceMutable, byteSliceBorrow := hir.BorrowElement(borrow.Type)
	bytePointerElement, bytePointerMutable, bytePointer := hir.RawPointerElement(expression.Type)
	if byteSliceBorrow && byteSliceElement == hir.TypeByteSlice &&
		bytePointer && bytePointerElement == hir.TypeU8 && (byteSliceMutable || !bytePointerMutable) {
		value, err := l.lowerExpression(borrow)
		if err != nil {
			return Operand{}, err
		}
		if value.Kind != OperandByteString && value.Kind != OperandLocal {
			return Operand{}, fmt.Errorf("byte-slice data address source has invalid operand kind %q", value.Kind)
		}
		destination := l.newTemporary()
		entity := sourceEntity("byte-slice-data-address", expression.Span)
		l.appendOperation(Operation{
			Kind: OperationByteSliceDataAddress, Entity: entity, Span: expression.Span,
			DestinationLocal: destination,
			Operands:         []Operand{value},
			Type:             expression.Type,
			OwnerType:        borrow.Type,
			Operator:         expression.Operator,
			UnsafeScope:      *expression.UnsafeScope,
		})
		l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
			Entity:           entity,
			SourceSpanEntity: sourceEntity("span", expression.Span),
			ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	}
	if borrow.Type == hir.SharedBorrow(hir.TypeCStr) && expression.Type == hir.RawPointer(hir.TypeU8, false) {
		value, err := l.lowerExpression(borrow)
		if err != nil {
			return Operand{}, err
		}
		if value.Kind != OperandCString && value.Kind != OperandLocal {
			return Operand{}, fmt.Errorf("C string data address source has invalid operand kind %q", value.Kind)
		}
		destination := l.newTemporary()
		entity := sourceEntity("cstr-data-address", expression.Span)
		l.appendOperation(Operation{
			Kind: OperationCStrDataAddress, Entity: entity, Span: expression.Span,
			DestinationLocal: destination,
			Operands:         []Operand{value},
			Type:             expression.Type,
			OwnerType:        borrow.Type,
			Operator:         expression.Operator,
			UnsafeScope:      *expression.UnsafeScope,
		})
		l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
			Entity:           entity,
			SourceSpanEntity: sourceEntity("span", expression.Span),
			ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	}
	if borrow.Kind == "path" && borrow.Symbol != hir.NoSymbol {
		sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(borrow.Type)
		targetElement, targetMutable, targetPointer := hir.RawPointerElement(expression.Type)
		if sourceBorrow && targetPointer && (targetElement == sourceElement || targetElement == hir.TypeVoid) &&
			(sourceMutable || !targetMutable) {
			value, err := l.lowerExpression(borrow)
			if err != nil {
				return Operand{}, err
			}
			if value.Kind != OperandLocal {
				return Operand{}, fmt.Errorf("raw borrow pointer source has invalid operand kind %q", value.Kind)
			}
			destination := l.newTemporary()
			entity := sourceEntity("raw-borrow-pointer", expression.Span)
			l.appendOperation(Operation{
				Kind: OperationRawBorrowPointer, Entity: entity, Span: expression.Span,
				DestinationLocal: destination,
				Operands:         []Operand{value},
				Type:             expression.Type,
				OwnerType:        borrow.Type,
				Operator:         expression.Operator,
				UnsafeScope:      *expression.UnsafeScope,
			})
			l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
				Entity:           entity,
				SourceSpanEntity: sourceEntity("span", expression.Span),
				ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
			})
			return Operand{Kind: OperandLocal, Local: destination}, nil
		}
	}
	if borrow.Kind != "prefix" || (borrow.Operator != "&" && borrow.Operator != "&mut") || len(borrow.Children) != 1 {
		return Operand{}, fmt.Errorf("raw-address cast source is not a direct borrow")
	}
	place := &borrow.Children[0]
	if place.Kind != "path" || place.Symbol == hir.NoSymbol {
		return Operand{}, fmt.Errorf("raw-address cast source is not a direct local place")
	}
	element, _, pointer := hir.RawPointerElement(expression.Type)
	if !pointer || element != place.Type && element != hir.TypeVoid {
		return Operand{}, fmt.Errorf("raw-address cast type %q does not match place type %q", expression.Type, place.Type)
	}
	destination := l.newTemporary()
	entity := sourceEntity("raw-address", expression.Span)
	l.appendOperation(Operation{
		Kind: OperationRawAddress, Entity: entity, Span: expression.Span,
		DestinationLocal: destination,
		Operands:         []Operand{{Kind: OperandLocal, Local: LocalID(place.Symbol)}},
		Type:             expression.Type,
		OwnerType:        place.Type,
		Operator:         borrow.Operator,
		UnsafeScope:      *expression.UnsafeScope,
	})
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
		Entity:           entity,
		SourceSpanEntity: sourceEntity("span", expression.Span),
		ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerBorrowedViewFromRawParts(expression *hir.Expression) (Operand, error) {
	if expression.UnsafeScope == nil || len(expression.Children) != 3 {
		return Operand{}, fmt.Errorf("malformed borrowed-view raw-parts construction")
	}
	operationKind := OperationKind("")
	entityKind := ""
	switch {
	case expression.Operator == "str::from_raw_parts" && expression.Type == hir.SharedBorrow(hir.TypeStr):
		operationKind = OperationStrFromRawParts
		entityKind = "str-from-raw-parts"
	case expression.Operator == "CStr::from_raw_parts" && expression.Type == hir.SharedBorrow(hir.TypeCStr):
		operationKind = OperationCStrFromRawParts
		entityKind = "cstr-from-raw-parts"
	case expression.Operator == "byte_slice::from_raw_parts" && expression.Type == hir.SharedBorrow(hir.TypeByteSlice):
		operationKind = OperationByteSliceFromRawParts
		entityKind = "byte-slice-from-raw-parts"
	case expression.Operator == "byte_slice::from_raw_parts_mut" && expression.Type == hir.MutableBorrow(hir.TypeByteSlice):
		operationKind = OperationMutByteSliceFromRawParts
		entityKind = "mut-byte-slice-from-raw-parts"
	default:
		return Operand{}, fmt.Errorf("unsupported borrowed-view raw-parts constructor %q with result %q", expression.Operator, expression.Type)
	}

	operands := make([]Operand, 0, len(expression.Children))
	for index := range expression.Children {
		operand, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if operand.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("borrowed-view raw-parts argument %d has no value", index)
		}
		operands = append(operands, operand)
	}

	destination := l.newTemporary()
	entity := sourceEntity(entityKind, expression.Span)
	l.appendOperation(Operation{
		Kind:             operationKind,
		Entity:           entity,
		Span:             expression.Span,
		DestinationLocal: destination,
		Operands:         operands,
		Type:             expression.Type,
		OwnerType:        expression.Children[0].Type,
		Operator:         expression.Operator,
		UnsafeScope:      *expression.UnsafeScope,
	})
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
		Entity:           entity,
		SourceSpanEntity: sourceEntity("span", expression.Span),
		ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerRegion(expression *hir.Expression) (Operand, error) {
	if expression.Symbol == hir.NoSymbol || len(expression.Children) != 1 || expression.Block == nil {
		return Operand{}, fmt.Errorf("malformed lexical region")
	}
	arena := &expression.Children[0]
	if arena.Kind != "prefix" || arena.Operator != "&mut" || len(arena.Children) != 1 || arena.Children[0].Kind != "path" {
		return Operand{}, fmt.Errorf("executable lexical region source must be a direct &mut local")
	}
	if _, active := l.regions[expression.Symbol]; active {
		return Operand{}, fmt.Errorf("region %d is entered twice", expression.Symbol)
	}
	arenaOwner, err := l.lowerExpression(&arena.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if arenaOwner.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("executable lexical region source must name one Arena local")
	}
	factIndex := len(l.result.RegionScopes)
	l.result.RegionScopes = append(l.result.RegionScopes, RegionScopeFact{Region: expression.Span})
	l.regions[expression.Symbol] = factIndex
	l.regionStack = append(l.regionStack, activeRegion{symbol: expression.Symbol, span: expression.Span, factIndex: factIndex})
	l.appendOperation(Operation{
		Kind: OperationRegionEnter, Span: expression.Span, Symbol: expression.Symbol,
		DestinationLocal: NoLocal, Operands: []Operand{arenaOwner},
		OwnerType: arena.Children[0].Type, Type: arena.Type, Operator: "&mut",
	})
	_, terminated, err := l.lowerNestedBlock(*expression.Block, false)
	if err != nil {
		return Operand{}, err
	}
	if err := l.leaveLexicalRegion(expression.Symbol); err != nil {
		return Operand{}, err
	}
	if terminated {
		return Operand{}, nil
	}
	cleanup := l.emitCleanupDeclaredWithin(l.live, expression.Span)
	l.appendOperation(Operation{Kind: OperationRegionReset, Span: expression.Span, Symbol: expression.Symbol, DestinationLocal: NoLocal, ExitKind: "fallthrough"})
	fact := &l.result.RegionScopes[factIndex]
	fact.CleanupBeforeReset = cleanup
	fact.Reset = expression.Span
	return Operand{}, nil
}

func (l *functionLowerer) lowerRegionPlace(expression *hir.Expression) (Operand, error) {
	receiver := &expression.Children[0]
	if receiver.Kind != "path" || receiver.Symbol == hir.NoSymbol {
		return Operand{}, fmt.Errorf("Region.place receiver is not one region capability")
	}
	factIndex, active := l.regions[receiver.Symbol]
	if !active {
		return Operand{}, fmt.Errorf("Region.place uses inactive region %d", receiver.Symbol)
	}
	valueExpression := &expression.Children[1]
	value, err := l.lowerExpression(valueExpression)
	if err != nil {
		return Operand{}, err
	}
	if value.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("Region.place value has no executable operand")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationRegionPlace, Span: expression.Span, Symbol: receiver.Symbol,
		DestinationLocal: destination, Operands: []Operand{value}, Type: expression.Type, OwnerType: valueExpression.Type,
	})
	l.result.RegionScopes[factIndex].Placements = append(l.result.RegionScopes[factIndex].Placements, expression.Span)
	regionEntity := fmt.Sprintf("region:%d", receiver.Symbol)
	operationEntity := fmt.Sprintf("operation:region-place:%d:%d", expression.Span.StartByte, expression.Span.EndByte)
	l.result.RegionValues = append(l.result.RegionValues, RegionValue{
		Entity: fmt.Sprintf("local:region-value:%d", destination), OperationEntity: operationEntity,
		RegionEntity: regionEntity, NeedsDrop: l.properties.NeedsDrop(valueExpression.Type),
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerAggregateConstruction(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != len(expression.Fields) || expression.Type == hir.TypeInvalid {
		return Operand{}, fmt.Errorf("malformed aggregate construction with %d values and %d fields", len(expression.Children), len(expression.Fields))
	}
	isResource := l.properties.IsResource(expression.Type)
	isStruct := l.properties.IsStruct(expression.Type)
	if isResource && len(expression.Children) == 0 {
		return Operand{}, fmt.Errorf("empty resource construction has no MIR support")
	}
	if !isResource && !isStruct {
		return Operand{}, fmt.Errorf("aggregate construction type %q has no MIR definition", expression.Type)
	}
	fields := l.properties.constructionFields[expression.Type]
	if len(fields) != len(expression.Fields) {
		return Operand{}, fmt.Errorf("aggregate construction fields do not match definition %q", expression.Type)
	}
	positions := make(map[string]int, len(fields))
	for index, name := range fields {
		positions[name] = index
	}
	operands := make([]Operand, len(fields))
	for index := range expression.Children {
		value, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		if value.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("aggregate field %q has no value", expression.Fields[index])
		}
		if path := l.liveResourcePath(&expression.Children[index]); isResource && path != nil {
			value = l.materializeMove(path, value, "resource-field", true)
		}
		// Evaluate written initializers in order; snapshot copyable values before
		// later initializers can mutate their source locals. Only the final
		// construction operand list follows declaration order.
		if _, sourceLocal := l.bindings[hir.SymbolID(value.Local)]; value.Kind == OperandLocal && sourceLocal && !l.properties.NeedsDrop(expression.Children[index].Type) {
			value = l.freezeScalarOperand(value, expression.Children[index].Span)
		}
		position, found := positions[expression.Fields[index]]
		if !found || operands[position].Kind != OperandInvalid {
			return Operand{}, fmt.Errorf("aggregate construction has unknown or duplicate field %q", expression.Fields[index])
		}
		operands[position] = value
	}
	destination := l.newTemporary()
	kind := OperationStructConstruct
	if isResource {
		kind = OperationResourceConstruct
	}
	l.appendOperation(Operation{
		Kind: kind, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Fields: slices.Clone(fields), Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerFieldRead(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator == "" {
		return Operand{}, fmt.Errorf("malformed field read %q with %d receivers", expression.Operator, len(expression.Children))
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("field read %q has no receiver value", expression.Operator)
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationFieldRead, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerViewLength(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator != "len" || expression.Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed borrowed-view length")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("borrowed-view length has no receiver value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationViewLength, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerViewEmpty(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator != "is_empty" || expression.Type != hir.TypeBool {
		return Operand{}, fmt.Errorf("malformed borrowed-view emptiness")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("borrowed-view emptiness has no receiver value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationViewEmpty, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerViewIndex(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 2 {
		return Operand{}, fmt.Errorf("malformed indexed-view operands")
	}
	element, indexed := hir.IndexedViewElement(expression.Children[0].Type)
	resultType := element
	kind := OperationViewIndex
	if expression.Kind == "view-element-borrow" {
		referent, mutable, borrowed := hir.BorrowElement(expression.Type)
		if !borrowed || referent != element || mutable && !hir.MutableIndexedView(expression.Children[0].Type) {
			return Operand{}, fmt.Errorf("invalid indexed element reference")
		}
		resultType = expression.Type
		kind = OperationViewElementBorrow
	}
	if !indexed || expression.Operator != "[]" || expression.Type != resultType || expression.Children[1].Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed indexed-view read")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	// Select the descriptor before index-side effects can reassign its binding.
	// This copies the descriptor, not the underlying borrowed storage.
	if kind == OperationViewElementBorrow || receiver.Kind == OperandLocal {
		receiver = l.freezeScalarOperand(receiver, expression.Children[0].Span)
	}
	index, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	if receiver.Kind == OperandInvalid || index.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("byte-view index has an invalid operand")
	}
	operands := []Operand{receiver, index}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression:               expression.Span,
		Sequence:                 []hir.Span{expression.Children[0].Span, expression.Children[1].Span},
		MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: OperationViewIndexCheck, Span: expression.Span, DestinationLocal: NoLocal, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, IndexType: expression.Children[1].Type, Type: element, Operator: expression.Operator,
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-view-index", SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: kind, Symbol: hir.NoSymbol, Span: expression.Span, DestinationLocal: destination,
		Operands: slices.Clone(operands), OwnerType: expression.Children[0].Type,
		IndexType: expression.Children[1].Type, Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func isIndexedViewType(value hir.Type) bool {
	_, indexed := hir.IndexedViewElement(value)
	return indexed
}

func (l *functionLowerer) lowerViewSlice(expression *hir.Expression) (Operand, error) {
	wantView := expression.Type
	if len(expression.Children) != 3 || expression.Operator != ".." ||
		!isIndexedViewType(wantView) || expression.Children[0].Type != wantView || expression.Children[1].Type != hir.TypeUSize || expression.Children[2].Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed immutable byte-view slice")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	// Bounds may reassign a descriptor binding; keep the receiver selected first.
	if receiver.Kind == OperandLocal {
		receiver = l.freezeScalarOperand(receiver, expression.Children[0].Span)
	}
	start := Operand{Kind: OperandUSize, U64: 0}
	if expression.Children[1].Kind == "omitted-range-bound" {
		if expression.Children[1].Operator != "start" {
			return Operand{}, fmt.Errorf("malformed omitted byte-view slice start")
		}
	} else {
		start, err = l.lowerExpression(&expression.Children[1])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		if start.Kind == OperandLocal {
			start = l.freezeScalarOperand(start, expression.Children[1].Span)
		}
	}
	var end Operand
	if expression.Children[2].Kind == "omitted-range-bound" {
		if expression.Children[2].Operator != "end" {
			return Operand{}, fmt.Errorf("malformed omitted byte-view slice end")
		}
		endLocal := l.newTemporary()
		l.appendOperation(Operation{
			Kind: OperationViewLength, Span: expression.Children[2].Span, DestinationLocal: endLocal,
			Operands: []Operand{receiver}, OwnerType: wantView, Type: hir.TypeUSize, Field: "len",
		})
		end = Operand{Kind: OperandLocal, Local: endLocal}
	} else {
		end, err = l.lowerExpression(&expression.Children[2])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
	}
	if receiver.Kind == OperandInvalid || start.Kind == OperandInvalid || end.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("immutable byte-view slice has an invalid operand")
	}
	operands := []Operand{receiver, start, end}
	sequence := []hir.Span{expression.Children[0].Span}
	if expression.Children[1].Kind != "omitted-range-bound" {
		sequence = append(sequence, expression.Children[1].Span)
	}
	if expression.Children[2].Kind != "omitted-range-bound" {
		sequence = append(sequence, expression.Children[2].Span)
	}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression:               expression.Span,
		Sequence:                 sequence,
		MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: OperationViewSliceCheck, Span: expression.Span, DestinationLocal: NoLocal, Operands: slices.Clone(operands),
		OwnerType: wantView, IndexType: hir.TypeUSize, Type: wantView, Operator: expression.Operator,
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-view-slice", SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationViewSlice, Span: expression.Span, DestinationLocal: destination,
		Operands: slices.Clone(operands), OwnerType: wantView,
		IndexType: hir.TypeUSize, Type: wantView, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerResultVariant(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || (expression.Variant != "ok" && expression.Variant != "err") {
		return Operand{}, fmt.Errorf("malformed Result variant %q with %d payloads", expression.Variant, len(expression.Children))
	}
	payload, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if payload.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("Result::%s payload has no value", expression.Variant)
	}
	if path := l.liveResourcePath(&expression.Children[0]); path != nil {
		payload = l.materializeMove(path, payload, "result-payload", true)
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{payload}, Variant: expression.Variant, Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerOptionVariant(expression *hir.Expression) (Operand, error) {
	if expression.Variant != "some" && expression.Variant != "none" {
		return Operand{}, fmt.Errorf("malformed Option variant %q", expression.Variant)
	}
	operands := make([]Operand, 0, 1)
	if expression.Variant == "some" {
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("Option::Some has %d payloads", len(expression.Children))
		}
		payload, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		if payload.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("Option::Some payload has no value")
		}
		if path := l.liveResourcePath(&expression.Children[0]); path != nil {
			payload = l.materializeMove(path, payload, "option-payload", true)
		}
		operands = append(operands, payload)
	} else if len(expression.Children) != 0 {
		return Operand{}, fmt.Errorf("Option::None has %d payloads", len(expression.Children))
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationOptionConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Variant: expression.Variant, Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerEnumVariant(expression *hir.Expression) (Operand, error) {
	definition := l.properties.Enum(expression.Type)
	if definition == nil || expression.Variant == "" || len(expression.Children) != len(expression.Fields) {
		return Operand{}, fmt.Errorf("malformed enum variant %q with %d values and %d fields", expression.Variant, len(expression.Children), len(expression.Fields))
	}
	variant, found := enumVariantDefinition(definition, expression.Variant)
	if !found || !sameNames(expression.Fields, enumFieldNames(variant.Fields)) {
		return Operand{}, fmt.Errorf("enum variant %q fields do not match type %q", expression.Variant, expression.Type)
	}
	operands := make([]Operand, 0, len(expression.Children))
	for index := range expression.Children {
		value, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if value.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("enum payload field %q has no value", expression.Fields[index])
		}
		operands = append(operands, value)
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationEnumConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Fields: slices.Clone(expression.Fields), Variant: expression.Variant, Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

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

func (l *functionLowerer) lowerWhile(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 2 {
		return Operand{}, fmt.Errorf("while expression has %d children", len(expression.Children))
	}
	headerLive := cloneLive(l.live)
	headerBlock := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})

	l.current = headerBlock
	l.live = cloneLive(headerLive)
	condition, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	if !sameLiveState(headerLive, l.live) {
		return Operand{}, fmt.Errorf("while condition changes resource state after Flow validation")
	}
	bodyBlock := l.newBlock()
	exitBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{bodyBlock, exitBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{condition},
	})

	l.current = bodyBlock
	l.live = cloneLive(headerLive)
	l.loopStack = append(l.loopStack, activeLoop{
		header: headerBlock, continueTarget: headerBlock, exit: exitBlock, body: expression.Children[1].Span,
		regionDepth: len(l.regionStack), headerLive: cloneLive(headerLive),
	})
	if _, err := l.lowerExpression(&expression.Children[1]); err != nil {
		l.loopStack = l.loopStack[:len(l.loopStack)-1]
		return Operand{}, err
	}
	l.loopStack = l.loopStack[:len(l.loopStack)-1]
	if !l.blockTerminated(l.current) {
		cleanup := l.emitCleanupDeclaredWithin(l.live, expression.Children[1].Span)
		if cleanupContainsDrop(cleanup) {
			l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
		}
		if !matchesLoopHeader(headerLive, l.live) {
			return Operand{}, fmt.Errorf("while back-edge changes resource state after Flow validation")
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Children[1].Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = exitBlock
	l.live = cloneLive(headerLive)
	return Operand{}, nil
}

func (l *functionLowerer) lowerLoopControl(kind string, exit hir.Span) error {
	if len(l.loopStack) == 0 {
		return fmt.Errorf("%s has no enclosing loop after type validation", kind)
	}
	loop := l.loopStack[len(l.loopStack)-1]
	actions := l.emitDeferredCallsForLoopExit(loop, exit)
	actions = append(actions, l.emitLoopControlCleanup(l.live, loop, exit, kind)...)
	if !matchesLoopHeader(loop.headerLive, l.live) {
		return fmt.Errorf("%s edge changes resource state after Flow validation", kind)
	}
	l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: exit, ExitKind: kind, Actions: actions})
	if cleanupContainsDrop(actions) {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	target := loop.continueTarget
	if kind == "break" {
		target = loop.exit
	}
	l.terminate(Terminator{Kind: TerminatorGoto, Span: exit, Successors: []BlockID{target}, UnwindSuccessor: NoBlock})
	return nil
}

func (l *functionLowerer) lowerFor(loop *hir.ForLoop) error {
	if loop == nil {
		return fmt.Errorf("for statement has no typed loop")
	}
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "classic-for", RemainingCount: 0})
	if loop.Initializer != nil {
		terminated, err := l.lowerStatements([]hir.Statement{*loop.Initializer})
		if err != nil {
			return err
		}
		if terminated {
			return fmt.Errorf("for initializer terminates control flow")
		}
	}

	headerLive := cloneLive(l.live)
	headerBlock := l.newBlock()
	bodyBlock := l.newBlock()
	postBlock := l.newBlock()
	exitBlock := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorGoto, Span: loop.Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})

	l.current = headerBlock
	l.live = cloneLive(headerLive)
	conditionExpression := loop.Condition
	if conditionExpression == nil {
		conditionExpression = &hir.Expression{Kind: "literal", Type: hir.TypeBool, Span: loop.Span, Value: "true", Cooked: "true", Symbol: hir.NoSymbol}
	}
	condition, err := l.lowerExpression(conditionExpression)
	if err != nil {
		return err
	}
	if !sameLiveState(headerLive, l.live) {
		return fmt.Errorf("for condition changes resource state after Flow validation")
	}
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: conditionExpression.Span,
		Successors: []BlockID{bodyBlock, exitBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchType: hir.TypeBool, SwitchOperands: []Operand{condition},
	})

	l.current = bodyBlock
	l.live = cloneLive(headerLive)
	l.loopStack = append(l.loopStack, activeLoop{
		header: headerBlock, continueTarget: postBlock, exit: exitBlock, body: loop.Body.Span,
		regionDepth: len(l.regionStack), headerLive: cloneLive(headerLive),
	})
	_, bodyTerminated, err := l.lowerNestedBlock(loop.Body, false)
	l.loopStack = l.loopStack[:len(l.loopStack)-1]
	if err != nil {
		return err
	}
	if !bodyTerminated && !l.blockTerminated(l.current) {
		cleanup := l.emitCleanupDeclaredWithin(l.live, loop.Body.Span)
		if cleanupContainsDrop(cleanup) {
			l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
		}
		if !matchesLoopHeader(headerLive, l.live) {
			return fmt.Errorf("for body back-edge changes resource state after Flow validation")
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: loop.Body.Span, Successors: []BlockID{postBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = postBlock
	l.live = cloneLive(headerLive)
	if loop.Post != nil {
		terminated, lowerErr := l.lowerStatements([]hir.Statement{*loop.Post})
		if lowerErr != nil {
			return lowerErr
		}
		if terminated {
			return fmt.Errorf("for post clause terminates control flow")
		}
	}
	if !sameLiveState(headerLive, l.live) {
		return fmt.Errorf("for post clause changes resource state after Flow validation")
	}
	l.terminate(Terminator{Kind: TerminatorGoto, Span: loop.Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})

	l.current = exitBlock
	l.live = cloneLive(headerLive)
	cleanup := l.emitCleanupDeclaredWithin(l.live, loop.Span)
	if cleanupContainsDrop(cleanup) {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	return nil
}

func (l *functionLowerer) lowerIf(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) < 2 || len(expression.Children) > 3 {
		return Operand{}, fmt.Errorf("if expression has %d children", len(expression.Children))
	}
	condition, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	branchLive := cloneLive(l.live)
	thenBlock := l.newBlock()
	elseBlock := l.newBlock()
	joinBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{thenBlock, elseBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{condition},
	})

	destination := NoLocal
	if expression.Type != hir.TypeVoid {
		destination = l.newTemporary()
	}

	l.current = thenBlock
	l.live = cloneLive(branchLive)
	thenValue, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	thenContinues := !l.blockTerminated(l.current)
	thenLive := cloneLive(l.live)
	if thenContinues {
		if destination != NoLocal {
			if thenValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing if has an empty then branch")
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span, DestinationLocal: destination, Operands: []Operand{thenValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Children[1].Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = elseBlock
	l.live = cloneLive(branchLive)
	elseValue := Operand{}
	if len(expression.Children) == 3 {
		elseValue, err = l.lowerExpression(&expression.Children[2])
		if err != nil {
			return Operand{}, err
		}
	}
	elseContinues := !l.blockTerminated(l.current)
	elseLive := cloneLive(l.live)
	if elseContinues {
		if destination != NoLocal {
			if elseValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing if has an empty else branch")
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: expressionSpan(&expression.Children[len(expression.Children)-1], expression.Span), DestinationLocal: destination, Operands: []Operand{elseValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = joinBlock
	switch {
	case thenContinues && elseContinues:
		if !sameLiveState(thenLive, elseLive) {
			return Operand{}, fmt.Errorf("if branches disagree on resource state after Flow validation")
		}
		l.live = thenLive
	case thenContinues:
		l.live = thenLive
	case elseContinues:
		l.live = elseLive
	default:
		l.live = cloneLive(branchLive)
		l.terminate(Terminator{Kind: TerminatorUnreachable, Span: expression.Span, UnwindSuccessor: NoBlock})
		return Operand{}, nil
	}
	if destination == NoLocal {
		return Operand{}, nil
	}
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerShortCircuit(expression *hir.Expression) (Operand, error) {
	left, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	branchLive := cloneLive(l.live)
	rightBlock := l.newBlock()
	shortBlock := l.newBlock()
	joinBlock := l.newBlock()
	successors := []BlockID{rightBlock, shortBlock}
	shortValue := int32(0)
	if expression.Operator == "||" {
		successors = []BlockID{shortBlock, rightBlock}
		shortValue = 1
	}
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: successors, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{left},
	})

	destination := l.newTemporary()
	l.current = shortBlock
	l.live = cloneLive(branchLive)
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[0].Span, DestinationLocal: destination, Operands: []Operand{{Kind: OperandI32, I32: shortValue, Type: hir.TypeBool}}})
	l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	shortLive := cloneLive(l.live)

	l.current = rightBlock
	l.live = cloneLive(branchLive)
	right, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	rightContinues := !l.blockTerminated(l.current)
	rightLive := cloneLive(l.live)
	if rightContinues {
		if right.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("short-circuit right operand has no value")
		}
		l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span, DestinationLocal: destination, Operands: []Operand{right}})
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = joinBlock
	if rightContinues {
		if !sameLiveState(shortLive, rightLive) {
			return Operand{}, fmt.Errorf("short-circuit branches disagree on resource state after Flow validation")
		}
		l.live = rightLive
	} else {
		l.live = shortLive
	}
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "source-short-circuit", RemainingCount: 0})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

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

func (l *functionLowerer) lowerCheckedDivision(expression *hir.Expression, left, right Operand) error {
	categories := []string{"division-by-zero"}
	if expression.Type == hir.TypeI32 || expression.Type == hir.TypeI64 {
		categories = append(categories, "signed-division-overflow")
	}
	l.appendOperation(Operation{
		Kind: OperationCheckedDivision, Span: expression.Span, Operands: []Operand{left, right},
		OwnerType: expression.Children[0].Type, Type: expression.Type,
		Operator: expression.Operator, TrapCategories: slices.Clone(categories),
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-division", SwitchType: expression.Children[0].Type,
		SwitchOperands: []Operand{left, right}, Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: slices.Clone(categories)})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: slices.Clone(categories), CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	return nil
}

func (l *functionLowerer) lowerCheckedArithmetic(expression *hir.Expression, operands []Operand, switchKind string, categories []string) {
	l.appendOperation(Operation{
		Kind: OperationArithmeticCheck, Span: expression.Span, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, Type: expression.Type,
		Operator: expression.Operator, TrapCategories: slices.Clone(categories),
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: switchKind, SwitchType: expression.Children[0].Type,
		SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: slices.Clone(categories)})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: slices.Clone(categories), CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
}

func (l *functionLowerer) lowerPropagate(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 {
		return Operand{}, fmt.Errorf("propagate expression has %d children", len(expression.Children))
	}
	operand := &expression.Children[0]
	resultValue, err := l.lowerExpression(operand)
	if err != nil {
		return Operand{}, err
	}
	if resultValue.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("propagate operand has no value")
	}
	if path := l.liveResourcePath(operand); path != nil {
		resultValue = l.materializeMove(path, resultValue, "result-branch", false)
	}
	l.appendOperation(Operation{Kind: OperationResultSwitch, Span: expression.Span, Operands: []Operand{resultValue}})
	success := l.newBlock()
	errorBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{success, errorBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "result", SwitchOperands: []Operand{resultValue},
	})

	errorLive := cloneLive(l.live)
	l.current = errorBlock
	_, errorType, operandIsResult := hir.ResultElements(operand.Type)
	_, returnErrorType, returnIsResult := hir.ResultElements(l.source.ResultType)
	if !operandIsResult || !returnIsResult || errorType != returnErrorType {
		return Operand{}, fmt.Errorf("propagate error type %q does not match function result %q", errorType, l.source.ResultType)
	}
	errorPayload := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultPayload, Span: expression.Span, DestinationLocal: errorPayload,
		Operands: []Operand{resultValue}, Variant: "err", MovePayload: l.properties.NeedsDrop(errorType), Type: errorType,
	})
	propagatedResult := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultConstruct, Span: expression.Span, DestinationLocal: propagatedResult,
		Operands: []Operand{{Kind: OperandLocal, Local: errorPayload}}, Variant: "err", Type: l.source.ResultType,
	})
	errorActions := l.emitDeferredCallsForFunctionExit(expression.Span)
	errorActions = append(errorActions, l.emitFunctionExitCleanup(errorLive, expression.Span, "propagate")...)
	l.terminate(Terminator{
		Kind: TerminatorReturn, Span: expression.Span, UnwindSuccessor: NoBlock,
		ReturnValue: Operand{Kind: OperandLocal, Local: propagatedResult},
	})
	question := hir.Span{Package: expression.Span.Package, Source: expression.Span.Source, StartByte: max(expression.Span.StartByte, expression.Span.EndByte-1), EndByte: expression.Span.EndByte}
	l.result.ResultBranches = append(l.result.ResultBranches, ResultBranchFact{
		Question: question, ResultExpression: operand.Span, EvaluationCount: 1,
		SuccessPayload: "Move", ErrorPayload: "Move", ResultTemporaryDeadOn: []string{"success", "error"},
		ErrorCleanupActions: errorActions,
	})
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "question", RemainingCount: 0})
	l.current = success
	payload := l.newTemporary()
	movePayload := false
	successType, _, resultType := hir.ResultElements(operand.Type)
	if resultType {
		movePayload = l.properties.NeedsDrop(successType)
	}
	l.appendOperation(Operation{
		Kind: OperationResultPayload, Span: expression.Span, DestinationLocal: payload,
		Operands: []Operand{resultValue}, Variant: "ok", MovePayload: movePayload, Type: successType,
	})
	return Operand{Kind: OperandLocal, Local: payload}, nil
}

func (l *functionLowerer) lowerReturn(expression *hir.Expression, exit hir.Span) error {
	value, err := l.lowerExpression(expression)
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	if path := l.liveResourcePath(expression); path != nil {
		value = l.materializeMove(path, value, "return-value", true)
	}
	actions := l.emitDeferredCallsForFunctionExit(exit)
	actions = append(actions, l.emitFunctionExitCleanup(l.live, exit, "return")...)
	l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: exit, ExitKind: "return", Actions: actions})
	if cleanupContainsDrop(actions) {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	l.terminate(Terminator{Kind: TerminatorReturn, Span: exit, UnwindSuccessor: NoBlock, ReturnValue: value})
	return nil
}

func (l *functionLowerer) movePath(expression *hir.Expression, destination string, record bool) {
	binding := l.bindings[expression.Symbol]
	l.appendOperation(Operation{Kind: OperationMove, Span: expression.Span, Symbol: expression.Symbol, Destination: destination})
	l.live[expression.Symbol] = false
	if record {
		l.result.Moves = append(l.result.Moves, MoveFact{
			Binding: binding.span, Operand: expression.Span, Destination: destination, CleanupObligationTransferred: true,
		})
	}
}

func (l *functionLowerer) materializeMove(expression *hir.Expression, value Operand, destination string, record bool) Operand {
	temporary := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Span, DestinationLocal: temporary, Operands: []Operand{value}})
	l.movePath(expression, destination, record)
	return Operand{Kind: OperandLocal, Local: temporary}
}

func (l *functionLowerer) emitDeferredCallsForScope(scope hir.Span) []CleanupAction {
	return l.emitDeferredCalls(func(call deferredCall) bool {
		return sameHIRSpan(call.scope, scope)
	})
}

func (l *functionLowerer) emitDeferredCallsForFunctionExit(exit hir.Span) []CleanupAction {
	return l.emitDeferredCalls(func(call deferredCall) bool {
		return spanContains(call.scope, exit) && call.span.StartByte < exit.EndByte
	})
}

func (l *functionLowerer) emitDeferredCallsForLoopExit(loop activeLoop, exit hir.Span) []CleanupAction {
	return l.emitDeferredCalls(func(call deferredCall) bool {
		return spanContains(loop.body, call.scope) && spanContains(call.scope, exit) && call.span.StartByte < exit.EndByte
	})
}

func (l *functionLowerer) emitDeferredCalls(selectCall func(deferredCall) bool) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.deferred) - 1; index >= 0; index-- {
		call := l.deferred[index]
		if !selectCall(call) {
			continue
		}
		l.result.Calls = append(l.result.Calls, CallFact{
			Call: call.span, ArgumentOrder: slices.Clone(call.argumentSpans), NormalSuccessor: true,
		})
		continuation := l.newBlock()
		l.terminate(Terminator{
			Kind: TerminatorCall, Span: call.span, Successors: []BlockID{continuation}, UnwindSuccessor: NoBlock,
			Callee: call.callee, Arguments: slices.Clone(call.arguments), DestinationLocal: NoLocal,
		})
		l.current = continuation
		actions = append(actions, CleanupAction{Kind: "defer-call", Binding: call.span})
	}
	return actions
}

func sameHIRSpan(left, right hir.Span) bool {
	return left.Package == right.Package && left.Source == right.Source &&
		left.StartByte == right.StartByte && left.EndByte == right.EndByte
}

func spanContains(container, nested hir.Span) bool {
	return container.Package == nested.Package && container.Source == nested.Source &&
		container.StartByte <= nested.StartByte && container.EndByte >= nested.EndByte
}

func (l *functionLowerer) emitCleanup(live map[hir.SymbolID]bool) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.result.Resources) - 1; index >= 0; index-- {
		resource := l.result.Resources[index]
		if !live[resource.Symbol] {
			continue
		}
		l.appendOperation(Operation{Kind: OperationDrop, Span: resource.Binding, Symbol: resource.Symbol, FlagClearedFirst: true})
		live[resource.Symbol] = false
		actions = append(actions, CleanupAction{Kind: "drop", Binding: resource.Binding})
	}
	return actions
}

func (l *functionLowerer) emitCleanupDeclaredWithin(live map[hir.SymbolID]bool, scope hir.Span) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.result.Resources) - 1; index >= 0; index-- {
		resource := l.result.Resources[index]
		if !sameSpanSource(resource.Binding, scope) || resource.Binding.StartByte < scope.StartByte || resource.Binding.EndByte > scope.EndByte || !live[resource.Symbol] {
			continue
		}
		l.appendOperation(Operation{Kind: OperationDrop, Span: resource.Binding, Symbol: resource.Symbol, FlagClearedFirst: true})
		live[resource.Symbol] = false
		actions = append(actions, CleanupAction{Kind: "drop", Binding: resource.Binding})
	}
	return actions
}

func (l *functionLowerer) emitFunctionExitCleanup(live map[hir.SymbolID]bool, exit hir.Span, exitKind string) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.regionStack) - 1; index >= 0; index-- {
		region := l.regionStack[index]
		cleanup := l.emitCleanupDeclaredWithin(live, region.span)
		actions = append(actions, cleanup...)
		l.appendOperation(Operation{Kind: OperationRegionReset, Span: region.span, Symbol: region.symbol, DestinationLocal: NoLocal, ExitKind: exitKind})
		actions = append(actions, CleanupAction{Kind: "region-reset", Binding: region.span})
		fact := &l.result.RegionScopes[region.factIndex]
		fact.EarlyResets = append(fact.EarlyResets, RegionResetFact{
			Exit: exit, ExitKind: exitKind, CleanupBeforeReset: cleanup, Reset: region.span,
		})
	}
	actions = append(actions, l.emitCleanup(live)...)
	return actions
}

func (l *functionLowerer) emitLoopControlCleanup(live map[hir.SymbolID]bool, loop activeLoop, exit hir.Span, exitKind string) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.regionStack) - 1; index >= loop.regionDepth; index-- {
		region := l.regionStack[index]
		cleanup := l.emitCleanupDeclaredWithin(live, region.span)
		actions = append(actions, cleanup...)
		l.appendOperation(Operation{Kind: OperationRegionReset, Span: region.span, Symbol: region.symbol, DestinationLocal: NoLocal, ExitKind: exitKind})
		actions = append(actions, CleanupAction{Kind: "region-reset", Binding: region.span})
		fact := &l.result.RegionScopes[region.factIndex]
		fact.EarlyResets = append(fact.EarlyResets, RegionResetFact{
			Exit: exit, ExitKind: exitKind, CleanupBeforeReset: cleanup, Reset: region.span,
		})
	}
	actions = append(actions, l.emitCleanupDeclaredWithin(live, loop.body)...)
	return actions
}

func cleanupContainsDrop(actions []CleanupAction) bool {
	for _, action := range actions {
		if action.Kind == "drop" {
			return true
		}
	}
	return false
}

func (l *functionLowerer) leaveLexicalRegion(symbol hir.SymbolID) error {
	if len(l.regionStack) == 0 || l.regionStack[len(l.regionStack)-1].symbol != symbol {
		return fmt.Errorf("region %d does not close in lexical stack order", symbol)
	}
	l.regionStack = l.regionStack[:len(l.regionStack)-1]
	delete(l.regions, symbol)
	return nil
}

func (l *functionLowerer) liveResourcePath(expression *hir.Expression) *hir.Expression {
	if expression == nil {
		return nil
	}
	switch expression.Kind {
	case "group", "unsafe":
		if len(expression.Children) == 1 {
			return l.liveResourcePath(&expression.Children[0])
		}
		return nil
	case "block":
		if expression.Block != nil && expression.Block.Tail != nil {
			return l.liveResourcePath(expression.Block.Tail)
		}
		return nil
	case "path":
		if expression.Symbol == hir.NoSymbol || !l.live[expression.Symbol] {
			return nil
		}
		binding, found := l.bindings[expression.Symbol]
		if found && l.properties.NeedsDrop(binding.typ) {
			return expression
		}
	}
	return nil
}

func (l *functionLowerer) newBlock() BlockID {
	id := BlockID(len(l.result.Blocks))
	l.result.Blocks = append(l.result.Blocks, Block{ID: id, Terminator: Terminator{UnwindSuccessor: NoBlock}})
	return id
}

func (l *functionLowerer) newTemporary() LocalID {
	result := l.nextLocal
	l.nextLocal++
	return result
}

func (l *functionLowerer) appendOperation(operation Operation) {
	l.result.Blocks[l.current].Operations = append(l.result.Blocks[l.current].Operations, operation)
}

func (l *functionLowerer) terminate(terminator Terminator) {
	l.result.Blocks[l.current].Terminator = terminator
}

func (l *functionLowerer) blockTerminated(id BlockID) bool {
	return l.result.Blocks[id].Terminator.Kind != ""
}

func expressionSpan(expression *hir.Expression, fallback hir.Span) hir.Span {
	if expression == nil {
		return fallback
	}
	return expression.Span
}

func cloneLive(input map[hir.SymbolID]bool) map[hir.SymbolID]bool {
	result := make(map[hir.SymbolID]bool, len(input))
	for symbol, live := range input {
		result[symbol] = live
	}
	return result
}

func matchesLoopHeader(header, actual map[hir.SymbolID]bool) bool {
	for symbol, live := range header {
		if actual[symbol] != live {
			return false
		}
	}
	for symbol, live := range actual {
		if _, existed := header[symbol]; !existed && live {
			return false
		}
	}
	return true
}

func containsCall(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if expression.Kind == "call" {
		return true
	}
	for index := range expression.Children {
		if containsCall(&expression.Children[index]) {
			return true
		}
	}
	for index := range expression.Arms {
		if containsCall(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func parseI32(text string) (int32, error) {
	value, err := parseSigned(text, 32)
	return int32(value), err
}

func parseSigned(text string, width int) (int64, error) {
	clean := strings.ReplaceAll(text, "_", "")
	base := 10
	digits := clean
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		base, digits = 16, clean[2:]
	} else if strings.HasPrefix(clean, "0b") || strings.HasPrefix(clean, "0B") {
		base, digits = 2, clean[2:]
	}
	value, err := strconv.ParseInt(digits, base, width)
	if err != nil {
		return 0, fmt.Errorf("lower i%d literal %q: %w", width, text, err)
	}
	return value, nil
}

func parseUnsigned(text string, bitSize int, typeName string) (uint64, error) {
	clean := strings.ReplaceAll(text, "_", "")
	base := 10
	digits := clean
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		base, digits = 16, clean[2:]
	} else if strings.HasPrefix(clean, "0b") || strings.HasPrefix(clean, "0B") {
		base, digits = 2, clean[2:]
	}
	value, err := strconv.ParseUint(digits, base, bitSize)
	if err != nil {
		return 0, fmt.Errorf("lower %s literal %q: %w", typeName, text, err)
	}
	return value, nil
}

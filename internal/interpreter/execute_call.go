package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

type callCompletion struct {
	value        runtimeValue
	trapCategory string
}

type runtimeRegion struct {
	span       hir.Span
	owner      *resourceState
	placements []runtimeValue
}

func (r *executionState) call(function *mir.Function, arguments []runtimeValue, depth int) (callCompletion, error) {
	if err := r.checkCancellation(); err != nil {
		return callCompletion{}, err
	}
	if depth >= r.callDepthLimit {
		return callCompletion{}, capabilityErrorKind(LimitCallDepth, "interpreter call-depth limit exceeded")
	}
	if len(arguments) != len(function.Parameters) {
		return callCompletion{}, fmt.Errorf("call %q received %d argument(s), want %d", function.Name, len(arguments), len(function.Parameters))
	}
	if definition := r.arenaConstructors[function.Symbol]; definition != nil {
		if err := r.consumeStep(); err != nil {
			return callCompletion{}, err
		}
		value, err := r.createAbstractArena(definition)
		return callCompletion{value: value}, err
	}
	locals := make(map[mir.LocalID]runtimeValue)
	regions := make(map[hir.SymbolID]*runtimeRegion)
	for index, parameter := range function.Parameters {
		locals[parameter] = arguments[index].cloneForValueCopy()
	}
	current := function.Entry
	for {
		if err := r.consumeStep(); err != nil {
			return callCompletion{}, err
		}
		if int(current) < 0 || int(current) >= len(function.Blocks) {
			return callCompletion{}, fmt.Errorf("function %q jumps to missing block %d", function.Name, current)
		}
		block := function.Blocks[current]
		for _, operation := range block.Operations {
			if err := r.consumeStep(); err != nil {
				return callCompletion{}, err
			}
			switch operation.Kind {
			case mir.OperationUse, mir.OperationDropFlagInit, mir.OperationDropFlagClear,
				mir.OperationArithmeticCheck, mir.OperationCheckedDivision, mir.OperationResultSwitch:
				// Verification and the explicit control-flow edges own these markers.
			case mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts,
				mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts,
				mir.OperationStrDataAddress, mir.OperationCStrDataAddress, mir.OperationByteSliceDataAddress,
				mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStoragePut,
				mir.OperationStorageBorrow, mir.OperationStorageBorrowMut, mir.OperationStorageElementAddress:
				return callCompletion{}, capabilityErrorKind(operation.Kind.String(), "%s in %q requires a native backend", operation.Kind, function.Name)
			case mir.OperationInitialize:
				if len(operation.Operands) != 1 {
					return callCompletion{}, capabilityError("resource initialization in %q has no executable value", function.Name)
				}
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("resource initialization in %q: %w", function.Name, err)
				}
				locals[mir.LocalID(operation.Symbol)] = value
			case mir.OperationAssign:
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("assign in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = value.cloneForValueCopy()
			case mir.OperationUnary:
				value, trap, err := evaluateUnary(locals, operation)
				if err != nil {
					return callCompletion{}, fmt.Errorf("unary in %q: %w", function.Name, err)
				}
				if trap != "" {
					return callCompletion{trapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationBinary:
				result, trap, err := evaluateBinary(locals, operation)
				if err != nil {
					return callCompletion{}, fmt.Errorf("binary in %q: %w", function.Name, err)
				}
				if trap != "" {
					return callCompletion{trapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = result
			case mir.OperationSharedReborrow:
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, err
				}
				value.typ = operation.Type
				if value.view() != nil {
					shared := *value.view()
					shared.mutable = false
					value.payload = &shared
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationScalarCast:
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("value-preserving integer cast in %q: %w", function.Name, err)
				}
				converted, err := valuePreservingIntegerCast(value, operation.OwnerType, operation.Type)
				if err != nil {
					return callCompletion{}, fmt.Errorf("value-preserving integer cast in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = converted
			case mir.OperationArrayConstruct:
				elementType, length, array := hir.FixedArrayParts(operation.Type)
				if !array || !hir.ExecutableFixedArrayShape(elementType, length) || operation.OwnerType != elementType || operation.Operator != "list" || uint64(len(operation.Operands)) != length {
					return callCompletion{}, fmt.Errorf("fixed-array construction in %q has incompatible type metadata", function.Name)
				}
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("fixed-array construction element %d in %q: %w", index, function.Name, err)
					}
					if !indexedElementRuntimeMatches(value, elementType) {
						return callCompletion{}, capabilityError("fixed-array construction element %d in %q has incompatible runtime type", index, function.Name)
					}
					values[index] = value.cloneForValueCopy()
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueArray, typ: operation.Type, payload: &arrayState{elements: values}}
			case mir.OperationArrayBorrowView:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("fixed-array byte-view borrow in %q: %w", function.Name, err)
				}
				arrayType := operation.OwnerType
				if element, _, borrowed := hir.BorrowElement(arrayType); borrowed {
					arrayType = element
				}
				elementType, length, _ := hir.FixedArrayParts(arrayType)
				wantView := operation.Type
				if !hir.ArrayViewBorrow(operation.OwnerType, wantView, operation.Operator) ||
					receiver.kind != valueArray || receiver.typ != operation.OwnerType || receiver.array() == nil || uint64(len(receiver.array().elements)) != length ||
					operation.Type != wantView || (operation.Operator != "&" && operation.Operator != "&mut") {
					return callCompletion{}, fmt.Errorf("fixed-array byte-view borrow in %q has incompatible runtime value", function.Name)
				}
				for index, element := range receiver.array().elements {
					if !indexedElementRuntimeMatches(element, elementType) {
						return callCompletion{}, fmt.Errorf("fixed-array view borrow element %d in %q has incompatible type", index, function.Name)
					}
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueView, typ: wantView, payload: &indexedViewState{
					array: receiver.array(), length: length, mutable: operation.Operator == "&mut",
				}}
			case mir.OperationArrayLength:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("fixed-array length in %q: %w", function.Name, err)
				}
				elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
				if !array || !hir.ExecutableFixedArrayShape(elementType, length) || receiver.kind != valueArray || receiver.typ != operation.OwnerType ||
					receiver.array() == nil || uint64(len(receiver.array().elements)) != length || operation.Type != hir.TypeUSize || operation.Field != "len" {
					return callCompletion{}, fmt.Errorf("fixed-array length in %q has incompatible runtime value", function.Name)
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueUSize, scalar: length, typ: hir.TypeUSize}
			case mir.OperationArrayIndex:
				receiver, index, err := evaluateArrayIndexOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("fixed-array index in %q: %w", function.Name, err)
				}
				elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
				if !array || !hir.ExecutableFixedArrayShape(elementType, length) || receiver.typ != operation.OwnerType || operation.Type != elementType || operation.IndexType != hir.TypeUSize || operation.Operator != "[]" {
					return callCompletion{}, fmt.Errorf("fixed-array index in %q has incompatible runtime type metadata", function.Name)
				}
				if receiver.array() == nil || index.u64() >= length || index.u64() >= uint64(len(receiver.array().elements)) {
					return callCompletion{}, fmt.Errorf("fixed-array index in %q executed without a dominating bounds check", function.Name)
				}
				locals[operation.DestinationLocal] = receiver.array().elements[index.u64()].cloneForValueCopy()
			case mir.OperationArrayIndexCheck:
			case mir.OperationArrayIndexStore:
				receiver, index, value, err := evaluateArrayIndexStoreOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("fixed-array index store in %q: %w", function.Name, err)
				}
				elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
				if !array || operation.Type != elementType || operation.IndexType != hir.TypeUSize || operation.Operator != "[]=" || receiver.typ != operation.OwnerType ||
					receiver.array() == nil || index.u64() >= length || index.u64() >= uint64(len(receiver.array().elements)) || !indexedElementRuntimeMatches(value, elementType) {
					return callCompletion{}, fmt.Errorf("fixed-array index store in %q has incompatible runtime value", function.Name)
				}
				receiver.array().elements[index.u64()] = value.cloneForValueCopy()
			case mir.OperationViewLength:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("borrowed-view length in %q: %w", function.Name, err)
				}
				if receiver.kind != valueView || receiver.view() == nil || receiver.typ != operation.OwnerType || operation.Type != hir.TypeUSize || operation.Field != "len" {
					return callCompletion{}, fmt.Errorf("borrowed-view length in %q has incompatible runtime value", function.Name)
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueUSize, scalar: receiver.view().length, typ: hir.TypeUSize}
			case mir.OperationViewEmpty:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("borrowed-view emptiness in %q: %w", function.Name, err)
				}
				if receiver.kind != valueView || receiver.view() == nil || receiver.typ != operation.OwnerType || operation.Type != hir.TypeBool || operation.Field != "is_empty" {
					return callCompletion{}, fmt.Errorf("borrowed-view emptiness in %q has incompatible runtime value", function.Name)
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueI32, scalar: i32Bits(boolValue(receiver.view().length == 0)), typ: hir.TypeBool}
			case mir.OperationViewIndex:
				receiver, index, err := evaluateViewIndexOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("byte-view index in %q: %w", function.Name, err)
				}
				element, indexed := hir.IndexedViewElement(operation.OwnerType)
				if receiver.typ != operation.OwnerType || !indexed ||
					operation.IndexType != hir.TypeUSize || operation.Type != element || operation.Operator != "[]" {
					return callCompletion{}, fmt.Errorf("byte-view index in %q has incompatible runtime value", function.Name)
				}
				value, err := receiver.indexedAt(index.u64())
				if err != nil {
					return callCompletion{}, fmt.Errorf("byte-view index in %q executed without a dominating bounds check: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationViewElementBorrow:
				receiver, index, err := evaluateViewIndexOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, err
				}
				value, err := receiver.borrowViewElement(index.u64(), operation.Type)
				if err != nil {
					return callCompletion{}, err
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationViewIndexCheck:
			case mir.OperationViewIndexStore:
				receiver, index, value, err := evaluateViewIndexStoreOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("mutable byte-view index store in %q: %w", function.Name, err)
				}
				element, indexed := hir.IndexedViewElement(operation.OwnerType)
				if !indexed || receiver.typ != operation.OwnerType || !hir.MutableIndexedView(operation.OwnerType) || operation.IndexType != hir.TypeUSize ||
					operation.Type != element || operation.Operator != "[]=" {
					return callCompletion{}, fmt.Errorf("mutable byte-view index store in %q has incompatible runtime metadata", function.Name)
				}
				if err := receiver.storeIndexed(index.u64(), value); err != nil {
					return callCompletion{}, fmt.Errorf("mutable byte-view index store in %q executed without a dominating bounds check: %w", function.Name, err)
				}
			case mir.OperationViewSlice:
				receiver, start, end, err := evaluateViewSliceOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("immutable byte-view slice in %q: %w", function.Name, err)
				}
				wantView := operation.OwnerType
				if receiver.typ != operation.OwnerType || operation.OwnerType != wantView ||
					!isRuntimeIndexedViewType(wantView) || operation.IndexType != hir.TypeUSize || operation.Type != wantView || operation.Operator != ".." {
					return callCompletion{}, fmt.Errorf("immutable byte-view slice in %q has incompatible runtime value", function.Name)
				}
				sliced, err := receiver.sliceIndexedView(start.u64(), end.u64())
				if err != nil {
					return callCompletion{}, fmt.Errorf("immutable byte-view slice in %q executed without a dominating bounds check: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = sliced
			case mir.OperationViewSliceCheck:
			case mir.OperationResultConstruct:
				payload, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Result construction in %q: %w", function.Name, err)
				}
				constructed, err := resultValue(operation.Variant, payload)
				if err != nil {
					return callCompletion{}, err
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationResultPayload:
				result, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Result payload in %q: %w", function.Name, err)
				}
				payload, err := result.resultPayload(operation.Variant, operation.MovePayload)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Result payload in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = payload
			case mir.OperationOptionConstruct:
				var payload *runtimeValue
				if operation.Variant == "some" {
					value, err := evaluateOne(locals, operation.Operands)
					if err != nil {
						return callCompletion{}, fmt.Errorf("Option construction in %q: %w", function.Name, err)
					}
					payload = &value
				} else if len(operation.Operands) != 0 {
					return callCompletion{}, fmt.Errorf("Option::None construction in %q has a payload", function.Name)
				}
				constructed, err := optionValue(operation.Variant, payload)
				if err != nil {
					return callCompletion{}, err
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationOptionPayload:
				option, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Option payload in %q: %w", function.Name, err)
				}
				payload, err := option.optionPayload(operation.MovePayload)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Option payload in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = payload
			case mir.OperationEnumConstruct:
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("enum construction in %q: %w", function.Name, err)
					}
					values[index] = value
				}
				constructed, err := enumValue(r.enumTypes[operation.Type], operation.Variant, operation.Fields, values)
				if err != nil {
					return callCompletion{}, fmt.Errorf("enum construction in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationEnumPayload:
				container, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("enum payload in %q: %w", function.Name, err)
				}
				payload, err := container.enumPayload(r.enumTypes[operation.OwnerType], operation.Variant, operation.Field)
				if err != nil {
					return callCompletion{}, fmt.Errorf("enum payload in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = payload
			case mir.OperationRawAddress, mir.OperationRawBorrowPointer:
				return callCompletion{}, capabilityErrorKind("raw-address", "raw address in %q requires a native backend", function.Name)
			case mir.OperationStructConstruct:
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("struct construction in %q: %w", function.Name, err)
					}
					values[index] = value
				}
				constructed, err := structValue(r.structTypes[operation.Type], values)
				if err != nil {
					return callCompletion{}, fmt.Errorf("struct construction in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationResourceConstruct:
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("resource construction in %q: %w", function.Name, err)
					}
					values[index] = value
				}
				constructed, err := resourceValue(r.resourceTypes[operation.Type], values)
				if err != nil {
					return callCompletion{}, fmt.Errorf("resource construction in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationFieldStore:
				if err := storeField(locals, operation, r.structTypes); err != nil {
					return callCompletion{}, fmt.Errorf("field store in %q: %w", function.Name, err)
				}
			case mir.OperationDerefStore:
				if err := storeDeref(locals, operation, r.structTypes); err != nil {
					return callCompletion{}, fmt.Errorf("dereference store in %q: %w", function.Name, err)
				}
			case mir.OperationFieldRead, mir.OperationFieldBorrow:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("field read in %q: %w", function.Name, err)
				}
				var value runtimeValue
				aggregateType := fieldAggregateType(operation.OwnerType)
				if definition := r.structTypes[aggregateType]; definition != nil {
					value, err = receiver.structField(definition, operation.Field)
				} else {
					value, err = receiver.field(r.resourceTypes[aggregateType], operation.Field)
				}
				if err != nil {
					return callCompletion{}, fmt.Errorf("field read in %q: %w", function.Name, err)
				}
				if operation.Kind == mir.OperationFieldBorrow {
					if element, _, _ := hir.BorrowElement(operation.Type); hir.FixedArrayScalarElement(element) {
						var fields *[]runtimeValue
						fieldIndex := -1
						if receiver.structure() != nil {
							fields = &receiver.structure().fields
							fieldIndex = structFieldIndex(receiver.structure().definition, operation.Field)
						} else if receiver.resource() != nil {
							fields = &receiver.resource().fields
							fieldIndex = resourceFieldIndex(receiver.resource().definition, operation.Field)
						}
						value.payload = &scalarReference{fields: fields, field: fieldIndex}
					}
					value.typ = operation.Type
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationRegionEnter:
				if _, active := regions[operation.Symbol]; active {
					return callCompletion{}, fmt.Errorf("region enter in %q reuses active region %d", function.Name, operation.Symbol)
				}
				owner, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("region enter in %q: %w", function.Name, err)
				}
				if owner.kind != valueResource || owner.resource() == nil || owner.resource().definition == nil ||
					owner.resource().definition.Type != operation.OwnerType || operation.Type != hir.MutableBorrow(operation.OwnerType) {
					return callCompletion{}, fmt.Errorf("region enter in %q has incompatible Arena owner", function.Name)
				}
				regions[operation.Symbol] = &runtimeRegion{span: operation.Span, owner: owner.resource()}
				if r.abstractArenas[owner.resource()] {
					if owner.resource().fields[2].u64() != 0 || owner.resource().fields[1].u64() == 0 {
						return callCompletion{trapCategory: "unreachable"}, nil
					}
					owner.resource().fields[2].scalar = 1
				}
			case mir.OperationRegionPlace:
				region := regions[operation.Symbol]
				if region == nil {
					return callCompletion{}, fmt.Errorf("region place in %q uses inactive region %d", function.Name, operation.Symbol)
				}
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("region place in %q: %w", function.Name, err)
				}
				stored := value.cloneForValueCopy()
				if operation.Operator == "alloc_zeroed_bytes" {
					stored, err = r.regionBytes(value)
					if err != nil {
						return callCompletion{}, err
					}
				}
				region.placements = append(region.placements, stored)
				borrow, _, result := hir.ResultElements(operation.Type)
				expectedBorrow := hir.SharedBorrow(operation.OwnerType)
				if operation.Operator == "alloc_zeroed_bytes" {
					expectedBorrow = hir.MutableBorrow(hir.TypeByteSlice)
				}
				if !result || borrow != expectedBorrow {
					return callCompletion{}, fmt.Errorf("region placement has incompatible result type")
				}
				stored.typ = borrow
				placed, err := resultValue("ok", stored)
				if err != nil {
					return callCompletion{}, fmt.Errorf("region place result in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = placed
			case mir.OperationRegionReset:
				region := regions[operation.Symbol]
				if region == nil {
					return callCompletion{}, fmt.Errorf("region reset in %q uses inactive region %d", function.Name, operation.Symbol)
				}
				r.regionTrace = append(r.regionTrace, RegionEvent{
					Region: region.span, ExitKind: operation.ExitKind,
					Placements: len(region.placements), DropCountBeforeReset: len(r.dropTrace),
				})
				if r.abstractArenas[region.owner] {
					region.owner.fields[2].scalar = 0
					generation := region.owner.fields[1].u64()
					if generation == ^uint64(0) {
						generation = 0
					}
					region.owner.fields[1].scalar = generation + 1
				}
				delete(regions, operation.Symbol)
			case mir.OperationMove:
				delete(locals, mir.LocalID(operation.Symbol))
			case mir.OperationDrop:
				local := mir.LocalID(operation.Symbol)
				value, found := locals[local]
				if !found {
					return callCompletion{}, fmt.Errorf("drop in %q names missing local %d", function.Name, local)
				}
				trap, err := r.dropValue(value, operation.Span, depth)
				if err != nil {
					return callCompletion{}, fmt.Errorf("drop in %q: %w", function.Name, err)
				}
				if trap != "" {
					return callCompletion{trapCategory: trap}, nil
				}
				delete(locals, local)
			case mir.OperationBurn:
				local := mir.LocalID(operation.Symbol)
				value, found := locals[local]
				if !found {
					return callCompletion{}, fmt.Errorf("burn in %q names missing local %d", function.Name, local)
				}
				if value.kind != valueResource || value.resource() == nil || value.resource().definition == nil || value.resource().definition.Type != operation.Type {
					return callCompletion{}, fmt.Errorf("burn in %q names an incompatible resource local %d", function.Name, local)
				}
				delete(locals, local)
			default:
				return callCompletion{}, capabilityError("unsupported MIR operation %q", operation.Kind)
			}
		}

		terminator := block.Terminator
		switch terminator.Kind {
		case mir.TerminatorGoto:
			next, err := oneSuccessor(terminator)
			if err != nil {
				return callCompletion{}, err
			}
			current = next
		case mir.TerminatorCall:
			if terminator.ForeignABI != "" {
				return callCompletion{}, capabilityErrorKind("foreign-call", "interpreter cannot execute foreign ABI %q call symbol %d", terminator.ForeignABI, terminator.Callee)
			}
			callee := r.bySymbol[terminator.Callee]
			if callee == nil {
				return callCompletion{}, fmt.Errorf("call in %q names missing symbol %d", function.Name, terminator.Callee)
			}
			values := make([]runtimeValue, len(terminator.Arguments))
			for index, operand := range terminator.Arguments {
				value, err := evaluate(locals, operand)
				if err != nil {
					return callCompletion{}, fmt.Errorf("call argument in %q: %w", function.Name, err)
				}
				values[index] = value
			}
			completion := callCompletion{}
			var callErr error
			if intrinsic, configured := r.testIntrinsics[callee.Symbol]; configured {
				value, err := r.callTestIntrinsic(intrinsic, values)
				if err != nil {
					return callCompletion{}, err
				}
				completion.value = value
			} else {
				completion, callErr = r.call(callee, values, depth+1)
			}
			if callErr != nil || completion.trapCategory != "" {
				return completion, callErr
			}
			if terminator.DestinationLocal != mir.NoLocal {
				locals[terminator.DestinationLocal] = completion.value.cloneForValueCopy()
			}
			next, err := oneSuccessor(terminator)
			if err != nil {
				return callCompletion{}, err
			}
			current = next
		case mir.TerminatorSwitch:
			next, trap, err := evaluateSwitch(locals, terminator)
			if err != nil {
				return callCompletion{}, err
			}
			if trap != "" {
				return callCompletion{trapCategory: trap}, nil
			}
			current = next
		case mir.TerminatorReturn:
			if len(regions) != 0 {
				return callCompletion{}, fmt.Errorf("return from %q leaves %d active region(s)", function.Name, len(regions))
			}
			if function.ResultType == hir.TypeVoid {
				return callCompletion{value: runtimeValue{kind: valueVoid, typ: hir.TypeVoid}}, nil
			}
			value, err := evaluate(locals, terminator.ReturnValue)
			if err != nil {
				return callCompletion{}, fmt.Errorf("return from %q: %w", function.Name, err)
			}
			return callCompletion{value: value}, nil
		case mir.TerminatorTrap:
			if len(terminator.TrapCategories) == 0 {
				return callCompletion{}, fmt.Errorf("trap in %q has no category", function.Name)
			}
			return callCompletion{trapCategory: terminator.TrapCategories[0]}, nil
		case mir.TerminatorUnreachable:
			return callCompletion{}, fmt.Errorf("function %q reached an unreachable MIR block", function.Name)
		default:
			return callCompletion{}, fmt.Errorf("function %q has unsupported terminator %q", function.Name, terminator.Kind)
		}
	}
}

package mir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

// operationVerifier holds the definition context for one function. Worklist state
// is passed separately; operation checks update its existing maps in evaluation order.
type operationVerifier struct {
	function       *Function
	structTypes    map[hir.Type]*StructType
	enumTypes      map[hir.Type]*EnumType
	resourceTypes  map[hir.Type]*ResourceType
	resources      map[hir.SymbolID]ResourceLocal
	dropEffects    map[hir.Type][]hir.Effect
	flags          map[string]DropFlag
	unsafeByEntity map[string]UnsafeOperation
	storeTypes     map[LocalID]hir.Type
}

func (v *operationVerifier) verifyOperation(state verificationState, operation Operation, entity string) []VerificationError {
	function := v.function
	structTypes := v.structTypes
	enumTypes := v.enumTypes
	resourceTypes := v.resourceTypes
	resources := v.resources
	dropEffects := v.dropEffects
	flags := v.flags
	storeTypes := v.storeTypes
	var failures []VerificationError
	if !operation.Kind.Valid() {
		failures = append(failures, VerificationError{Code: CodeOperationKind, Entity: entity, Detail: fmt.Sprintf("unknown MIR operation kind %q", operation.Kind)})
		return failures
	}
	if detail := verifyIndexedViewOperands(operation, storeTypes); detail != "" {
		failures = append(failures, VerificationError{Code: CodeStorageOperands, Entity: entity, Detail: detail})
	}
	if operation.SourceForm != "" {
		failures = append(failures, VerificationError{Code: "V1601", Entity: entity, Detail: "source-only form remains: " + operation.SourceForm})
	}
	if operation.MovePayload && operation.Kind != OperationResultPayload && operation.Kind != OperationOptionPayload {
		failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "payload move is attached to a non-payload operation"})
	}
	switch operation.Kind {
	case OperationInitialize:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if len(operation.Operands) > 1 || len(operation.Operands) == 1 && operation.DestinationLocal != LocalID(operation.Symbol) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "resource initialization shape is invalid"})
		}
		if state.live[operation.Symbol] {
			failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: "resource initialized twice"})
		}
		state.live[operation.Symbol] = true
	case OperationUse:
		if !state.live[operation.Symbol] {
			failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: "value used before initialization"})
		}
	case OperationMove:
		if operation.ProjectionEntity != "" {
			failures = append(failures, VerificationError{Code: "V1103", Entity: operation.ProjectionEntity, Detail: "resource move names a projected place"})
		}
		if resource, found := resources[operation.Symbol]; found && !state.live[operation.Symbol] {
			failures = append(failures, VerificationError{Code: "V1101", Entity: resourceEntity(resource, entity), Detail: "resource moved after discharge"})
		}
		state.live[operation.Symbol] = false
	case OperationDropFlagInit:
		flag := state.flags[operation.DropFlagEntity]
		flag.initialized = true
		flag.active = true
		state.flags[operation.DropFlagEntity] = flag
	case OperationDropFlagClear:
		flag := state.flags[operation.DropFlagEntity]
		if _, declared := flags[operation.DropFlagEntity]; !declared || !flag.initialized {
			failures = append(failures, VerificationError{Code: "V1201", Entity: operation.DropFlagEntity, Detail: "drop flag used before initialization"})
		}
		flag.active = false
		state.flags[operation.DropFlagEntity] = flag
	case OperationDrop:
		resource, found := resources[operation.Symbol]
		if found {
			required, known := dropEffects[resource.Type]
			if !known {
				required = resourceCleanupEffects(resource.Type, resourceTypes)
				dropEffects[resource.Type] = required
			}
			if len(hir.MissingEffects(function.Effects, required)) != 0 {
				failures = append(failures, VerificationError{Code: CodeEffectAuthority, Entity: entity, Detail: "resource cleanup exceeds function effect authority"})
			}
		}
		if found && !state.live[operation.Symbol] {
			failures = append(failures, VerificationError{Code: "V1101", Entity: resourceEntity(resource, entity), Detail: "resource dropped after discharge"})
		}
		if operation.DropFlagEntity != "" {
			if state.flags[operation.DropFlagEntity].active {
				failures = append(failures, VerificationError{Code: "V1202", Entity: entity, Detail: "drop runs before its active flag is cleared"})
			}
		} else if !operation.FlagClearedFirst {
			failures = append(failures, VerificationError{Code: "V1202", Entity: entity, Detail: "drop does not clear its active flag first"})
		}
		state.live[operation.Symbol] = false
	case OperationBurn:
		resource, found := resources[operation.Symbol]
		if !found || operation.Type == hir.ParseType("") || operation.Type == hir.TypeInvalid || found && operation.Type != resource.Type ||
			operation.ProjectionEntity != "" || operation.Destination != "" || operation.DestinationLocal != NoLocal ||
			len(operation.Operands) != 0 || operation.DropFlagEntity != "" || operation.FlagClearedFirst {
			failures = append(failures, VerificationError{Code: "V1104", Entity: entity, Detail: "burn does not name one direct declared resource root"})
		}
		if found && !state.live[operation.Symbol] {
			failures = append(failures, VerificationError{Code: "V1101", Entity: resourceEntity(resource, entity), Detail: "resource burned after discharge"})
		}
		state.live[operation.Symbol] = false
	case OperationCheckedDivision:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		categories, valid := divisionTrapCategories(operation.OwnerType)
		if !validCheckedDivisionShape(operation) || !valid || !slices.Equal(operation.TrapCategories, categories) {
			failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "division is missing required trap checks"})
		}
	case OperationArithmeticCheck:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		_, categories, valid := checkedArithmeticSpec(operation.Operator, len(operation.Operands))
		if !valid || !validCheckedArithmeticShape(operation) || !slices.Equal(operation.TrapCategories, categories) {
			failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "arithmetic operation is missing required trap checks"})
		}
	case OperationViewIndexCheck:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validViewIndexShape(operation, false) || !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
			failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "byte-view index is missing its bounds check"})
		}
	case OperationViewSliceCheck:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validViewSliceShape(operation, false) || !slices.Equal(operation.TrapCategories, []string{"bounds"}) ||
			hir.MutableIndexedView(operation.OwnerType) && fieldStoreOperandType(operation.Operands[0], storeTypes) != operation.OwnerType {
			failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "immutable byte-view slice is missing its bounds check"})
		}
	case OperationAssign:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationUnary:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validUnaryShape(operation) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "typed scalar unary operation shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationBinary:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validBinaryShape(operation) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "typed scalar binary operation shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationSharedReborrow:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !ValidSharedReborrowShape(operation) || fieldStoreOperandType(operation.Operands[0], storeTypes) != operation.OwnerType {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "shared reborrow must shorten the same exclusive referent"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationScalarCast:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if operation.DestinationLocal == NoLocal || len(operation.Operands) != 1 || operation.Operator != "as" ||
			!hir.ValuePreservingIntegerCast(operation.OwnerType, operation.Type) ||
			!scalarCastOperandMatches(operation.Operands[0], operation.OwnerType) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "value-preserving integer cast shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationArrayConstruct:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validArrayConstructShape(operation) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "fixed-array construction shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationArrayBorrowView:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validArrayBorrowViewShape(operation) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "fixed-array byte-view borrow shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationArrayLength:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validArrayLengthShape(operation) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "fixed-array length shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationStorageLayout:
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationStorageTake, OperationStorageBorrow, OperationStorageBorrowMut:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if (operation.Kind == OperationStorageBorrow || operation.Kind == OperationStorageBorrowMut) && isIndexedStorageView(operation.Type) &&
			(len(operation.Operands) != 3 || fieldStoreOperandType(operation.Operands[0], storeTypes) != operation.OwnerType ||
				fieldStoreOperandType(operation.Operands[1], storeTypes) != operation.IndexType ||
				fieldStoreOperandType(operation.Operands[2], storeTypes) != hir.TypeUSize) {
			failures = append(failures, VerificationError{Code: CodeStorageOperands, Entity: entity, Detail: "private storage view operand types are invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationStorageElementAddress:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if len(operation.Operands) != 2 ||
			fieldStoreOperandType(operation.Operands[0], storeTypes) != hir.RawPointer(hir.TypeVoid, true) ||
			fieldStoreOperandType(operation.Operands[1], storeTypes) != hir.TypeUSize {
			failures = append(failures, VerificationError{Code: CodeStorageOperands, Entity: entity, Detail: "private storage element address operand types are invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationStoragePut:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if len(operation.Operands) == 2 && operation.Operands[1].Kind == OperandLocal {
			state.live[hir.SymbolID(operation.Operands[1].Local)] = false
		}
	case OperationArrayIndexCheck:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validArrayIndexShape(operation, false) || !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
			failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "fixed-array index is missing its bounds check"})
		}
	case OperationArrayIndex:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validArrayIndexShape(operation, true) || len(operation.TrapCategories) != 0 {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "fixed-array index shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationArrayIndexStore:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validArrayIndexStoreShape(operation) || len(operation.TrapCategories) != 0 {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "fixed-array index store shape is invalid"})
		}
	case OperationResultConstruct:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if len(operation.Operands) != 1 || operation.DestinationLocal == NoLocal {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "Result operation must have one operand and one destination"})
		}
		if operation.Variant != "ok" && operation.Variant != "err" {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "Result operation has an invalid variant"})
		}
		if _, _, ok := hir.ResultElements(operation.Type); !ok {
			failures = append(failures, VerificationError{Code: CodeOperationType, Entity: entity, Detail: "Result construction has no concrete Result type"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationResultPayload:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if len(operation.Operands) != 1 || operation.DestinationLocal == NoLocal {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "Result operation must have one operand and one destination"})
		}
		if operation.Variant != "ok" && operation.Variant != "err" {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "Result operation has an invalid variant"})
		}
		if operation.Type == hir.TypeInvalid || operation.Type == hir.ParseType("") {
			failures = append(failures, VerificationError{Code: CodeOperationType, Entity: entity, Detail: "Result payload has no concrete payload type"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationOptionConstruct:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		validShape := operation.DestinationLocal != NoLocal &&
			(operation.Variant == "some" && len(operation.Operands) == 1 || operation.Variant == "none" && len(operation.Operands) == 0)
		if !validShape {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "Option construction shape does not match its variant"})
		}
		if _, ok := hir.OptionElement(operation.Type); !ok {
			failures = append(failures, VerificationError{Code: CodeOperationType, Entity: entity, Detail: "Option construction has no concrete Option type"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationOptionPayload:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if operation.DestinationLocal == NoLocal || operation.Variant != "some" || len(operation.Operands) != 1 {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "Option payload extraction must target Some"})
		}
		if operation.Type == hir.TypeInvalid || operation.Type == hir.ParseType("") {
			failures = append(failures, VerificationError{Code: CodeOperationType, Entity: entity, Detail: "Option payload has no concrete payload type"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationEnumConstruct:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		definition := enumTypes[operation.Type]
		variant, found := enumVariantDefinition(definition, operation.Variant)
		if definition == nil || !found {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: enumTypeEntity(operation.Type), Detail: "enum construction type or variant has no definition"})
		}
		if operation.DestinationLocal == NoLocal || len(operation.Fields) != len(operation.Operands) || !sameNames(operation.Fields, enumFieldNames(variant.Fields)) {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "enum construction fields do not match the variant definition"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationEnumPayload:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		definition := enumTypes[operation.OwnerType]
		variant, variantFound := enumVariantDefinition(definition, operation.Variant)
		field, fieldFound := enumFieldDefinition(variant, operation.Field)
		if definition == nil || !variantFound || !fieldFound || field.Type != operation.Type || operation.DestinationLocal == NoLocal || len(operation.Operands) != 1 {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "enum payload extraction does not match the variant definition"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationRawAddress, OperationRawBorrowPointer, OperationStrFromRawParts, OperationCStrFromRawParts, OperationByteSliceFromRawParts, OperationMutByteSliceFromRawParts, OperationStrDataAddress, OperationByteSliceDataAddress, OperationCStrDataAddress:
		failures = append(failures, v.verifyPointerOperation(state, operation, entity)...)
	case OperationStructConstruct:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		definition := structTypes[operation.Type]
		if definition == nil {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: structTypeEntity(operation.Type), Detail: "struct construction type has no definition"})
		}
		if operation.DestinationLocal == NoLocal || operation.Type == hir.ParseType("") || operation.Type == hir.TypeInvalid || len(operation.Fields) != len(operation.Operands) || !uniqueNonEmpty(operation.Fields) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "struct construction shape is invalid"})
		}
		if definition != nil && !slices.Equal(operation.Fields, structFieldNames(definition.Fields)) {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "struct construction fields do not match the definition"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationResourceConstruct:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		definition := resourceTypes[operation.Type]
		if definition == nil {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: resourceTypeEntity(operation.Type), Detail: "resource construction type has no definition"})
		}
		if operation.DestinationLocal == NoLocal || operation.Type == hir.ParseType("") || operation.Type == hir.TypeInvalid || len(operation.Fields) == 0 || len(operation.Fields) != len(operation.Operands) || !uniqueNonEmpty(operation.Fields) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "resource construction shape is invalid"})
		}
		if definition != nil && !slices.Equal(operation.Fields, resourceFieldNames(definition.Fields)) {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "resource construction fields do not match the definition"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationFieldStore:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		valid := ValidFieldStoreShape(operation) ||
			ValidOneWordStructFieldStoreShape(operation, structTypes[operation.Type]) ||
			ValidPlainStructFieldStoreShape(operation, func(typ hir.Type) *StructType { return structTypes[typ] })
		if valid {
			valid = fieldStoreOperandType(operation.Operands[0], storeTypes) == operation.OwnerType &&
				fieldStoreOperandType(operation.Operands[1], storeTypes) == operation.Type
		}
		aggregate := fieldAggregateType(operation.OwnerType)
		if len(operation.Fields) > 0 {
			_, found := ResolveNestedFieldStore(operation, func(typ hir.Type) *StructType { return structTypes[typ] })
			valid = valid && found
		} else if definition := structTypes[aggregate]; definition != nil {
			field, found := structField(definition.Fields, operation.Field)
			valid = valid && found && field.Type == operation.Type
		} else if definition := resourceTypes[aggregate]; definition != nil {
			field, found := resourceField(definition.Fields, operation.Field)
			valid = valid && found && field.Type == operation.Type
		} else {
			valid = false
		}
		if !valid {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "scalar field store does not match its writable owner and declared field"})
		}
	case OperationDerefStore:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if (!ValidDerefStoreShape(operation) && !ValidPlainStructDerefStoreShape(operation, func(typ hir.Type) *StructType { return structTypes[typ] })) ||
			fieldStoreOperandType(operation.Operands[0], storeTypes) != operation.OwnerType ||
			fieldStoreOperandType(operation.Operands[1], storeTypes) != operation.Type {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "dereference store does not match its mutable reference and admitted value"})
		}
	case OperationFieldRead, OperationFieldBorrow:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		fieldType := operation.Type
		if operation.Kind == OperationFieldBorrow {
			element, _, _ := hir.BorrowElement(operation.Type)
			if !validFieldBorrowShape(operation) ||
				!validMutableFieldBorrow(operation, func(typ hir.Type) *StructType { return structTypes[typ] }) ||
				fieldStoreOperandType(operation.Operands[0], storeTypes) != operation.OwnerType {
				failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "field borrow requires matching reference kinds, actual owner type and an admitted field"})
			}
			fieldType = element
		}
		aggregateType := fieldAggregateType(operation.OwnerType)
		resourceDefinition := resourceTypes[aggregateType]
		structDefinition := structTypes[aggregateType]
		if resourceDefinition == nil && structDefinition == nil {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: structTypeEntity(aggregateType), Detail: "field-read owner type has no aggregate definition"})
		}
		if operation.DestinationLocal == NoLocal || operation.OwnerType == hir.ParseType("") || operation.OwnerType == hir.TypeInvalid || operation.Type == hir.ParseType("") || operation.Type == hir.TypeInvalid || operation.Field == "" || len(operation.Operands) != 1 {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "field read shape is invalid"})
		}
		if resourceDefinition != nil {
			field, found := resourceField(resourceDefinition.Fields, operation.Field)
			if !found || field.Type != fieldType {
				failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "field read does not match the resource definition"})
			}
		}
		if structDefinition != nil {
			field, found := structField(structDefinition.Fields, operation.Field)
			if !found || field.Type != fieldType {
				failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: entity, Detail: "field read does not match the struct definition"})
			}
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationViewLength:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		ownerElement, _, ownerBorrow := hir.BorrowElement(operation.OwnerType)
		ownerValid := ownerBorrow && (ownerElement == hir.TypeStr || ownerElement == hir.TypeByteSlice || hir.NonByteIndexedView(operation.OwnerType))
		operandValid := len(operation.Operands) == 1 && (operation.Operands[0].Kind == OperandLocal ||
			ownerElement == hir.TypeStr && operation.Operands[0].Kind == OperandString ||
			ownerElement == hir.TypeByteSlice && operation.Operands[0].Kind == OperandByteString)
		if operation.DestinationLocal == NoLocal || !ownerValid || !operandValid ||
			operation.Type != hir.TypeUSize || operation.Field != "len" {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "borrowed-view length shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationViewEmpty:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		ownerElement, _, ownerBorrow := hir.BorrowElement(operation.OwnerType)
		ownerValid := ownerBorrow && (ownerElement == hir.TypeStr || ownerElement == hir.TypeByteSlice || hir.NonByteIndexedView(operation.OwnerType))
		operandValid := len(operation.Operands) == 1 && (operation.Operands[0].Kind == OperandLocal ||
			ownerElement == hir.TypeStr && operation.Operands[0].Kind == OperandString ||
			ownerElement == hir.TypeByteSlice && operation.Operands[0].Kind == OperandByteString)
		if operation.DestinationLocal == NoLocal || !ownerValid || !operandValid ||
			operation.Type != hir.TypeBool || operation.Field != "is_empty" {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "borrowed-view emptiness shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationViewIndex:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validViewIndexShape(operation, true) || len(operation.TrapCategories) != 0 {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "byte-view index shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationViewElementBorrow:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !ValidViewElementBorrowShape(operation) {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "indexed element borrow shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationViewIndexStore:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validViewIndexStoreShape(operation) || len(operation.TrapCategories) != 0 {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "mutable byte-view index store shape is invalid"})
		}
	case OperationViewSlice:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if !validViewSliceShape(operation, true) || len(operation.TrapCategories) != 0 ||
			hir.MutableIndexedView(operation.OwnerType) && fieldStoreOperandType(operation.Operands[0], storeTypes) != operation.OwnerType {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "immutable byte-view slice shape is invalid"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationRegionEnter:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		arenaType, mutable, borrow := hir.BorrowElement(operation.Type)
		shapeValid := operation.Symbol != hir.NoSymbol && len(operation.Operands) == 1 &&
			operation.Operands[0].Kind == OperandLocal && operation.DestinationLocal == NoLocal &&
			borrow && mutable && operation.Operator == "&mut" && operation.OwnerType == arenaType &&
			resourceTypes[arenaType] != nil
		if !shapeValid {
			failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "region enter shape is invalid"})
		}
		if operation.Symbol == hir.NoSymbol {
			break
		}
		if _, active := state.regions[operation.Symbol]; active {
			failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "region is entered while already active"})
		} else {
			state.regions[operation.Symbol] = entity
		}
	case OperationRegionPlace:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		successType, failureType, result := hir.ResultElements(operation.Type)
		placedType, mutable, borrow := hir.BorrowElement(successType)
		payloadValid := borrow && !mutable && placedType == operation.OwnerType && operation.Operator == ""
		if operation.Operator == "alloc_zeroed_bytes" {
			payloadValid = successType == hir.MutableBorrow(hir.TypeByteSlice) && operation.OwnerType == hir.TypeUSize &&
				len(operation.Operands) == 1 && fieldStoreOperandType(operation.Operands[0], storeTypes) == hir.TypeUSize
		}
		shapeValid := result && payloadValid &&
			failureType != hir.TypeInvalid && failureType != hir.ParseType("") && len(operation.Operands) == 1 &&
			operation.DestinationLocal != NoLocal && operation.OwnerType != hir.TypeInvalid && operation.OwnerType != hir.ParseType("")
		if _, active := state.regions[operation.Symbol]; !active || !shapeValid {
			failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "region placement does not name one active region and value"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationRegionReset:
		_, active := state.regions[operation.Symbol]
		if !active || len(operation.Operands) != 0 || operation.DestinationLocal != NoLocal || !validRegionExitKind(operation.ExitKind) {
			failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "region reset does not close one active region"})
		}
		if active {
			delete(state.regions, operation.Symbol)
		}
	case OperationResultSwitch:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		if len(operation.Operands) != 1 {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "Result switch operation must have one operand"})
		}
	}
	return failures
}

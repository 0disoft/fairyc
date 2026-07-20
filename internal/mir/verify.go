package mir

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

// VerificationError identifies one internal MIR invariant failure.
type VerificationError struct {
	Code   string
	Entity string
	Detail string
}

func (e VerificationError) Error() string {
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Entity, e.Detail)
}

type flagState struct {
	initialized bool
	active      bool
}

type verificationState struct {
	live    map[hir.SymbolID]bool
	flags   map[string]flagState
	regions map[hir.SymbolID]string
}

// Verify checks the executable invariants required before MIR handoff.
func Verify(program Program) []VerificationError {
	structTypes, failures := verifyStructTypes(program.StructTypes)
	enumTypes, enumFailures := verifyEnumTypes(program.EnumTypes)
	failures = append(failures, enumFailures...)
	resourceTypes, resourceFailures := verifyResourceTypes(program.ResourceTypes)
	failures = append(failures, resourceFailures...)
	for value := range structTypes {
		if _, collision := resourceTypes[value]; collision {
			failures = append(failures, VerificationError{Code: "V1001", Entity: structTypeEntity(value), Detail: "aggregate type is both struct and resource"})
		}
		if _, collision := enumTypes[value]; collision {
			failures = append(failures, VerificationError{Code: "V1001", Entity: structTypeEntity(value), Detail: "aggregate type is both struct and enum"})
		}
	}
	for value := range enumTypes {
		if _, collision := resourceTypes[value]; collision {
			failures = append(failures, VerificationError{Code: "V1001", Entity: enumTypeEntity(value), Detail: "aggregate type is both enum and resource"})
		}
	}
	foreignFunctions, foreignFailures := verifyForeignFunctions(program.ForeignFunctions)
	failures = append(failures, foreignFailures...)
	for index := range program.Functions {
		failures = append(failures, verifyFunction(&program.Functions[index], structTypes, enumTypes, resourceTypes, foreignFunctions)...)
	}
	slices.SortFunc(failures, func(left, right VerificationError) int {
		if value := cmp.Compare(left.Code, right.Code); value != 0 {
			return value
		}
		return cmp.Compare(left.Entity, right.Entity)
	})
	return slices.CompactFunc(failures, func(left, right VerificationError) bool {
		return left.Code == right.Code && left.Entity == right.Entity
	})
}

func verifyEnumTypes(definitions []EnumType) (map[hir.Type]*EnumType, []VerificationError) {
	enums := make(map[hir.Type]*EnumType, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, enumTypeEntity(definition.Type))
		if definition.Type == "" || definition.Type == hir.TypeInvalid || definition.Name == "" {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum definition has no concrete identity"})
			continue
		}
		if _, duplicate := enums[definition.Type]; duplicate {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum definition is duplicated"})
			continue
		}
		enums[definition.Type] = definition
		variantNames := make(map[string]struct{}, len(definition.Variants))
		for _, variant := range definition.Variants {
			if variant.Name == "" {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum definition has an empty variant"})
				continue
			}
			if _, duplicate := variantNames[variant.Name]; duplicate {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum definition has a duplicate variant"})
				continue
			}
			variantNames[variant.Name] = struct{}{}
			fieldNames := make(map[string]struct{}, len(variant.Fields))
			for _, field := range variant.Fields {
				if field.Name == "" || field.Type == "" || field.Type == hir.TypeInvalid {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum definition has an invalid payload field"})
					continue
				}
				if _, duplicate := fieldNames[field.Name]; duplicate {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum definition has a duplicate payload field"})
					continue
				}
				fieldNames[field.Name] = struct{}{}
			}
		}
	}
	return enums, failures
}

func verifyStructTypes(definitions []StructType) (map[hir.Type]*StructType, []VerificationError) {
	structs := make(map[hir.Type]*StructType, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, structTypeEntity(definition.Type))
		if definition.Type == "" || definition.Type == hir.TypeInvalid || definition.Name == "" {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "struct definition has no concrete identity"})
			continue
		}
		if _, duplicate := structs[definition.Type]; duplicate {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "struct definition is duplicated"})
			continue
		}
		structs[definition.Type] = definition
		if definition.ReprC && len(definition.Fields) == 0 {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "repr(C) struct definition is empty"})
		}
		fieldNames := make(map[string]struct{}, len(definition.Fields))
		for _, field := range definition.Fields {
			if field.Name == "" || field.Type == "" || field.Type == hir.TypeInvalid {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "struct definition has an invalid field"})
				continue
			}
			if _, duplicate := fieldNames[field.Name]; duplicate {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "struct definition has a duplicate field"})
				continue
			}
			fieldNames[field.Name] = struct{}{}
		}
	}
	return structs, failures
}

func verifyForeignFunctions(definitions []ForeignFunction) (map[hir.SymbolID]*ForeignFunction, []VerificationError) {
	result := make(map[hir.SymbolID]*ForeignFunction, len(definitions))
	byLinkName := make(map[string]*ForeignFunction, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, fmt.Sprintf("foreign:%d", definition.Symbol))
		if definition.Symbol == hir.NoSymbol || definition.Name == "" || definition.LinkName == "" || definition.ABI != "C" || definition.ResultType == "" || definition.ResultType == hir.TypeInvalid {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "foreign declaration is incomplete or has an unsupported ABI"})
			continue
		}
		if _, duplicate := result[definition.Symbol]; duplicate {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "foreign symbol is duplicated"})
			continue
		}
		result[definition.Symbol] = definition
		if previous, duplicate := byLinkName[definition.LinkName]; duplicate {
			if !CompatibleForeignSignature(*previous, *definition) {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "foreign declarations for one link name have conflicting signatures"})
			}
		} else {
			byLinkName[definition.LinkName] = definition
		}
		if _, _, borrow := hir.BorrowElement(definition.ResultType); borrow {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "foreign result exposes a safe borrow"})
		}
		for _, parameterType := range definition.ParameterTypes {
			if parameterType == "" || parameterType == hir.TypeInvalid {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "foreign parameter has no concrete type"})
			}
			if _, _, borrow := hir.BorrowElement(parameterType); borrow {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "foreign parameter exposes a safe borrow"})
			}
		}
	}
	return result, failures
}

func verifyResourceTypes(definitions []ResourceType) (map[hir.Type]*ResourceType, []VerificationError) {
	resources := make(map[hir.Type]*ResourceType, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, resourceTypeEntity(definition.Type))
		if definition.Type == "" || definition.Type == hir.TypeInvalid {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource definition has no concrete type"})
			continue
		}
		if _, duplicate := resources[definition.Type]; duplicate {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource definition is duplicated"})
			continue
		}
		resources[definition.Type] = definition
		if definition.DropKind != hir.ResourceDropEmpty {
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource definition has unsupported drop behavior"})
		}
		fieldNames := make(map[string]struct{}, len(definition.Fields))
		for _, field := range definition.Fields {
			if field.Name == "" || field.Type == "" || field.Type == hir.TypeInvalid {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource definition has an invalid field"})
				continue
			}
			if _, duplicate := fieldNames[field.Name]; duplicate {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource definition has a duplicate field"})
				continue
			}
			fieldNames[field.Name] = struct{}{}
		}
	}
	return resources, failures
}

func verifyFunction(function *Function, structTypes map[hir.Type]*StructType, enumTypes map[hir.Type]*EnumType, resourceTypes map[hir.Type]*ResourceType, foreignFunctions map[hir.SymbolID]*ForeignFunction) []VerificationError {
	if int(function.Entry) < 0 || int(function.Entry) >= len(function.Blocks) {
		return []VerificationError{{Code: "V1001", Entity: function.Name, Detail: "entry block is missing"}}
	}
	var failures []VerificationError
	resources := make(map[hir.SymbolID]ResourceLocal, len(function.Resources))
	initial := verificationState{
		live:    make(map[hir.SymbolID]bool, len(function.Resources)),
		flags:   make(map[string]flagState, len(function.DropFlags)),
		regions: make(map[hir.SymbolID]string),
	}
	for _, resource := range function.Resources {
		resources[resource.Symbol] = resource
		initial.live[resource.Symbol] = resource.Parameter
		if !containsDefinedResource(resource.Type, resourceTypes) {
			failures = append(failures, VerificationError{Code: "V1001", Entity: resourceTypeEntity(resource.Type), Detail: "resource local type has no definition"})
		}
	}
	for _, parameter := range function.Parameters {
		initial.live[hir.SymbolID(parameter)] = true
	}
	flags := make(map[string]DropFlag, len(function.DropFlags))
	for _, flag := range function.DropFlags {
		flags[flag.Entity] = flag
		initial.flags[flag.Entity] = flagState{}
	}
	unsafeByEntity := make(map[string]UnsafeOperation, len(function.UnsafeOperations))
	for _, operation := range function.UnsafeOperations {
		if operation.Entity == "" {
			failures = append(failures, VerificationError{Code: "V1701", Entity: function.Name, Detail: "unsafe instruction has no entity identity"})
			continue
		}
		if _, duplicate := unsafeByEntity[operation.Entity]; duplicate {
			failures = append(failures, VerificationError{Code: "V1701", Entity: operation.Entity, Detail: "unsafe instruction entity is duplicated"})
			continue
		}
		unsafeByEntity[operation.Entity] = operation
	}
	type pendingBlock struct {
		id    BlockID
		state verificationState
	}
	queue := []pendingBlock{{id: function.Entry, state: initial}}
	seen := make(map[BlockID]verificationState)
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if previous, found := seen[item.id]; found {
			merged, changed, incompatible := mergeVerificationState(previous, item.state, resources)
			if incompatible {
				failures = append(failures, VerificationError{Code: "V1002", Entity: blockEntityFor(function, item.id), Detail: "incoming initialization states disagree"})
				continue
			}
			if !changed {
				continue
			}
			item.state = merged
		}
		seen[item.id] = cloneVerificationState(item.state)
		if int(item.id) < 0 || int(item.id) >= len(function.Blocks) {
			failures = append(failures, VerificationError{Code: "V1001", Entity: blockEntityFor(function, item.id), Detail: "successor block is missing"})
			continue
		}
		block := function.Blocks[item.id]
		state := cloneVerificationState(item.state)
		for operationIndex, operation := range block.Operations {
			entity := operationEntity(item.id, operationIndex, operation)
			if operation.SourceForm != "" {
				failures = append(failures, VerificationError{Code: "V1601", Entity: entity, Detail: "source-only form remains: " + operation.SourceForm})
			}
			if operation.MovePayload && operation.Kind != OperationResultPayload && operation.Kind != OperationOptionPayload {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "payload move is attached to a non-payload operation"})
			}
			switch operation.Kind {
			case OperationInitialize:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if len(operation.Operands) > 1 || len(operation.Operands) == 1 && operation.DestinationLocal != LocalID(operation.Symbol) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource initialization shape is invalid"})
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
					failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "immutable byte-view index is missing its bounds check"})
				}
			case OperationAssign:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationUnary:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if !validUnaryShape(operation) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "typed scalar unary operation shape is invalid"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationBinary:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if !validBinaryShape(operation) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "typed scalar binary operation shape is invalid"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationScalarCast:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if operation.DestinationLocal == NoLocal || len(operation.Operands) != 1 || operation.Operator != "as" ||
					!hir.ValuePreservingIntegerCast(operation.OwnerType, operation.Type) ||
					!scalarCastOperandMatches(operation.Operands[0], operation.OwnerType) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "value-preserving integer cast shape is invalid"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationResultConstruct:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if len(operation.Operands) != 1 || operation.DestinationLocal == NoLocal {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result operation must have one operand and one destination"})
				}
				if operation.Variant != "ok" && operation.Variant != "err" {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result operation has an invalid variant"})
				}
				if _, _, ok := hir.ResultElements(operation.Type); !ok {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result construction has no concrete Result type"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationResultPayload:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if len(operation.Operands) != 1 || operation.DestinationLocal == NoLocal {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result operation must have one operand and one destination"})
				}
				if operation.Variant != "ok" && operation.Variant != "err" {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result operation has an invalid variant"})
				}
				if operation.Type == hir.TypeInvalid || operation.Type == "" {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result payload has no concrete payload type"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationOptionConstruct:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				validShape := operation.DestinationLocal != NoLocal &&
					(operation.Variant == "some" && len(operation.Operands) == 1 || operation.Variant == "none" && len(operation.Operands) == 0)
				if !validShape {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Option construction shape does not match its variant"})
				}
				if _, ok := hir.OptionElement(operation.Type); !ok {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Option construction has no concrete Option type"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationOptionPayload:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if operation.DestinationLocal == NoLocal || operation.Variant != "some" || len(operation.Operands) != 1 {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Option payload extraction must target Some"})
				}
				if operation.Type == hir.TypeInvalid || operation.Type == "" {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Option payload has no concrete payload type"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationEnumConstruct:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				definition := enumTypes[operation.Type]
				variant, found := enumVariantDefinition(definition, operation.Variant)
				if definition == nil || !found {
					failures = append(failures, VerificationError{Code: "V1001", Entity: enumTypeEntity(operation.Type), Detail: "enum construction type or variant has no definition"})
				}
				if operation.DestinationLocal == NoLocal || len(operation.Fields) != len(operation.Operands) || !sameNames(operation.Fields, enumFieldNames(variant.Fields)) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum construction fields do not match the variant definition"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationEnumPayload:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				definition := enumTypes[operation.OwnerType]
				variant, variantFound := enumVariantDefinition(definition, operation.Variant)
				field, fieldFound := enumFieldDefinition(variant, operation.Field)
				if definition == nil || !variantFound || !fieldFound || field.Type != operation.Type || operation.DestinationLocal == NoLocal || len(operation.Operands) != 1 {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "enum payload extraction does not match the variant definition"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationRawAddress:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				element, mutable, pointer := hir.RawPointerElement(operation.Type)
				shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 &&
					operation.Operands[0].Kind == OperandLocal && operation.OwnerType != "" && operation.OwnerType != hir.TypeInvalid &&
					pointer && element == operation.OwnerType &&
					(operation.Operator == "&" && !mutable || operation.Operator == "&mut")
				if !shapeValid {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "raw address shape is invalid"})
				}
				provenance, hasProvenance := unsafeByEntity[operation.Entity]
				expectedEntity := sourceEntity("raw-address", operation.Span)
				sourceSpanEntity := sourceEntity("span", operation.Span)
				scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
				scopeContainsOperation := operation.UnsafeScope.Source == operation.Span.Source &&
					operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
				if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
					provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
					failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "raw address lacks matching unsafe provenance"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationStrDataAddress:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				element, mutable, pointer := hir.RawPointerElement(operation.Type)
				shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 &&
					(operation.Operands[0].Kind == OperandString || operation.Operands[0].Kind == OperandLocal) &&
					operation.OwnerType == hir.SharedBorrow(hir.TypeStr) &&
					pointer && element == hir.TypeU8 && !mutable && operation.Operator == "as"
				if !shapeValid {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "string data address shape is invalid"})
				}
				provenance, hasProvenance := unsafeByEntity[operation.Entity]
				expectedEntity := sourceEntity("str-data-address", operation.Span)
				sourceSpanEntity := sourceEntity("span", operation.Span)
				scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
				scopeContainsOperation := operation.UnsafeScope.Source == operation.Span.Source &&
					operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
				if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
					provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
					failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "string data address lacks matching unsafe provenance"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationByteSliceDataAddress:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				element, mutable, pointer := hir.RawPointerElement(operation.Type)
				shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 &&
					(operation.Operands[0].Kind == OperandByteString || operation.Operands[0].Kind == OperandLocal) &&
					operation.OwnerType == hir.SharedBorrow(hir.TypeByteSlice) &&
					pointer && element == hir.TypeU8 && !mutable && operation.Operator == "as"
				if !shapeValid {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "byte-slice data address shape is invalid"})
				}
				provenance, hasProvenance := unsafeByEntity[operation.Entity]
				expectedEntity := sourceEntity("byte-slice-data-address", operation.Span)
				sourceSpanEntity := sourceEntity("span", operation.Span)
				scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
				scopeContainsOperation := operation.UnsafeScope.Source == operation.Span.Source &&
					operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
				if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
					provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
					failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "byte-slice data address lacks matching unsafe provenance"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationCStrDataAddress:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				element, mutable, pointer := hir.RawPointerElement(operation.Type)
				shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 &&
					operation.Operands[0].Kind == OperandCString && operation.OwnerType == hir.SharedBorrow(hir.TypeCStr) &&
					pointer && element == hir.TypeU8 && !mutable && operation.Operator == "as"
				if !shapeValid {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "C string data address shape is invalid"})
				}
				provenance, hasProvenance := unsafeByEntity[operation.Entity]
				expectedEntity := sourceEntity("cstr-data-address", operation.Span)
				sourceSpanEntity := sourceEntity("span", operation.Span)
				scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
				scopeContainsOperation := operation.UnsafeScope.Source == operation.Span.Source &&
					operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
				if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
					provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
					failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "C string data address lacks matching unsafe provenance"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationStructConstruct:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				definition := structTypes[operation.Type]
				if definition == nil {
					failures = append(failures, VerificationError{Code: "V1001", Entity: structTypeEntity(operation.Type), Detail: "struct construction type has no definition"})
				}
				if operation.DestinationLocal == NoLocal || operation.Type == "" || operation.Type == hir.TypeInvalid || len(operation.Fields) == 0 || len(operation.Fields) != len(operation.Operands) || !uniqueNonEmpty(operation.Fields) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "struct construction shape is invalid"})
				}
				if definition != nil && !slices.Equal(operation.Fields, structFieldNames(definition.Fields)) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "struct construction fields do not match the definition"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationResourceConstruct:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				definition := resourceTypes[operation.Type]
				if definition == nil {
					failures = append(failures, VerificationError{Code: "V1001", Entity: resourceTypeEntity(operation.Type), Detail: "resource construction type has no definition"})
				}
				if operation.DestinationLocal == NoLocal || operation.Type == "" || operation.Type == hir.TypeInvalid || len(operation.Fields) == 0 || len(operation.Fields) != len(operation.Operands) || !uniqueNonEmpty(operation.Fields) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource construction shape is invalid"})
				}
				if definition != nil && !slices.Equal(operation.Fields, resourceFieldNames(definition.Fields)) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource construction fields do not match the definition"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationFieldRead:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				resourceDefinition := resourceTypes[operation.OwnerType]
				structDefinition := structTypes[operation.OwnerType]
				if resourceDefinition == nil && structDefinition == nil {
					failures = append(failures, VerificationError{Code: "V1001", Entity: structTypeEntity(operation.OwnerType), Detail: "field-read owner type has no aggregate definition"})
				}
				if operation.DestinationLocal == NoLocal || operation.OwnerType == "" || operation.OwnerType == hir.TypeInvalid || operation.Type == "" || operation.Type == hir.TypeInvalid || operation.Field == "" || len(operation.Operands) != 1 {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "field read shape is invalid"})
				}
				if resourceDefinition != nil {
					field, found := resourceField(resourceDefinition.Fields, operation.Field)
					if !found || field.Type != operation.Type {
						failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "field read does not match the resource definition"})
					}
				}
				if structDefinition != nil {
					field, found := structField(structDefinition.Fields, operation.Field)
					if !found || field.Type != operation.Type {
						failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "field read does not match the struct definition"})
					}
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationViewLength:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				ownerElement, _, ownerBorrow := hir.BorrowElement(operation.OwnerType)
				ownerValid := ownerBorrow && (ownerElement == hir.TypeStr || ownerElement == hir.TypeByteSlice)
				operandValid := len(operation.Operands) == 1 && (operation.Operands[0].Kind == OperandLocal ||
					ownerElement == hir.TypeStr && operation.Operands[0].Kind == OperandString ||
					ownerElement == hir.TypeByteSlice && operation.Operands[0].Kind == OperandByteString)
				if operation.DestinationLocal == NoLocal || !ownerValid || !operandValid ||
					operation.Type != hir.TypeUSize || operation.Field != "len" {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "borrowed-view length shape is invalid"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationViewIndex:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if !validViewIndexShape(operation, true) || len(operation.TrapCategories) != 0 {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "immutable byte-view index shape is invalid"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationRegionEnter:
				if operation.Symbol == hir.NoSymbol || len(operation.Operands) != 0 || operation.DestinationLocal != NoLocal {
					failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "region enter shape is invalid"})
				} else if _, active := state.regions[operation.Symbol]; active {
					failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "region is entered while already active"})
				} else {
					state.regions[operation.Symbol] = entity
				}
			case OperationRegionPlace:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if _, active := state.regions[operation.Symbol]; !active || len(operation.Operands) != 1 || operation.DestinationLocal == NoLocal || operation.Type == hir.TypeInvalid {
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
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result switch operation must have one operand"})
				}
			}
		}

		terminator := block.Terminator
		if terminator.Kind == "" {
			failures = append(failures, VerificationError{Code: "V1001", Entity: blockEntityFor(function, item.id), Detail: "reachable block has no terminator"})
			continue
		}
		if terminator.Kind == TerminatorCall && terminator.UnwindSuccessor != NoBlock {
			failures = append(failures, VerificationError{Code: "V1502", Entity: terminatorEntity(item.id, terminator), Detail: "Stage 0 call has an unwind successor"})
		}
		if terminator.Kind == TerminatorCall {
			failures = append(failures, verifyOperands(state, terminator.Arguments, terminatorEntity(item.id, terminator))...)
			foreign := foreignFunctions[terminator.Callee]
			if terminator.ForeignABI != "" {
				provenance, hasProvenance := unsafeByEntity[terminatorEntity(item.id, terminator)]
				callEntity := sourceEntity("call", terminator.Span)
				sourceSpanEntity := sourceEntity("span", terminator.Span)
				scopeEntity := sourceEntity("unsafe-scope", terminator.UnsafeScope)
				scopeContainsCall := terminator.UnsafeScope.Source == terminator.Span.Source &&
					terminator.UnsafeScope.StartByte <= terminator.Span.StartByte && terminator.UnsafeScope.EndByte >= terminator.Span.EndByte
				if foreign == nil || terminator.ForeignABI != "C" || terminator.Entity != callEntity || !scopeContainsCall ||
					!hasProvenance || provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
					failures = append(failures, VerificationError{Code: "V1701", Entity: terminatorEntity(item.id, terminator), Detail: "foreign call lacks a matching declaration or unsafe provenance"})
				}
			} else if foreign != nil {
				failures = append(failures, VerificationError{Code: "V1701", Entity: terminatorEntity(item.id, terminator), Detail: "foreign call is missing its ABI and unsafe provenance"})
			}
			if terminator.DestinationLocal != NoLocal {
				state.live[hir.SymbolID(terminator.DestinationLocal)] = true
			}
		}
		if terminator.Kind == TerminatorSwitch {
			failures = append(failures, verifyOperands(state, terminator.SwitchOperands, terminatorEntity(item.id, terminator))...)
			if !validSwitchShape(terminator) {
				failures = append(failures, VerificationError{Code: "V1001", Entity: terminatorEntity(item.id, terminator), Detail: "switch shape does not match its kind"})
			}
			if terminator.SwitchKind == "enum" {
				definition := enumTypes[terminator.SwitchType]
				if definition == nil || !slices.Equal(terminator.SwitchVariants, enumVariantNames(definition.Variants)) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: terminatorEntity(item.id, terminator), Detail: "enum switch variants do not match the enum definition"})
				}
			}
		}
		if terminator.Kind == TerminatorReturn {
			failures = append(failures, verifyOperands(state, []Operand{terminator.ReturnValue}, terminatorEntity(item.id, terminator))...)
			for symbol, resource := range resources {
				if state.live[symbol] {
					failures = append(failures, VerificationError{Code: "V1102", Entity: blockEntityFor(function, item.id), Detail: "resource " + resourceEntity(resource, fmt.Sprintf("local:%d", symbol)) + " exits without cleanup"})
				}
			}
			for _, entity := range state.regions {
				failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "normal return exits an active region without reset"})
			}
			continue
		}
		if terminator.Kind == TerminatorTrap || terminator.Kind == TerminatorUnreachable {
			continue
		}
		for _, successor := range terminator.Successors {
			queue = append(queue, pendingBlock{id: successor, state: cloneVerificationState(state)})
		}
	}
	failures = append(failures, verifyTrapCheckEdges(function)...)

	for _, loan := range function.Loans {
		if loan.OutlivesOrigin {
			failures = append(failures, VerificationError{Code: "V1301", Entity: loan.Entity, Detail: "safe borrow outlives its origin"})
		}
	}
	if len(function.ReturnOrigins) > 0 {
		returned := 0
		for _, origin := range function.ReturnOrigins {
			if origin.Returned {
				returned++
			}
		}
		if returned != 1 {
			failures = append(failures, VerificationError{Code: "V1302", Entity: fallbackEntity(function.ReturnEntity, "terminator:return"), Detail: "safe-borrow return does not have exactly one origin"})
		}
	}
	for _, value := range function.RegionValues {
		if value.Escapes {
			failures = append(failures, VerificationError{Code: "V1401", Entity: value.Entity, Detail: "region-derived value escapes its region"})
		}
		if value.NeedsDrop {
			failures = append(failures, VerificationError{Code: "V1402", Entity: value.OperationEntity, Detail: "plain region receives a NeedsDrop value"})
		}
	}
	for _, operation := range function.UnsafeOperations {
		if operation.SourceSpanEntity == "" || operation.ScopeEntity == "" {
			failures = append(failures, VerificationError{Code: "V1701", Entity: operation.Entity, Detail: "unsafe instruction lacks source provenance"})
		}
	}
	return failures
}

func validRegionExitKind(value string) bool {
	return value == "fallthrough" || value == "return" || value == "propagate" || value == "break" || value == "continue"
}

func validSwitchShape(terminator Terminator) bool {
	switch terminator.SwitchKind {
	case "":
		return len(terminator.SwitchOperands) == 0 && len(terminator.Successors) == 2
	case "bool":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "result":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "option":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "enum":
		return terminator.SwitchType != "" && terminator.SwitchType != hir.TypeInvalid && len(terminator.SwitchOperands) == 1 && len(terminator.Successors) > 0 && len(terminator.Successors) == len(terminator.SwitchVariants) && uniqueNonEmpty(terminator.SwitchVariants)
	case "checked-division":
		return len(terminator.SwitchOperands) == 2 && len(terminator.Successors) == 2 &&
			(terminator.Operator == "/" || terminator.Operator == "%") && hir.CheckedDivisionType(terminator.SwitchType)
	case "checked-view-index":
		return len(terminator.SwitchOperands) == 2 && len(terminator.Successors) == 2 && terminator.Operator == "[]"
	case "checked-unary", "checked-arithmetic", "checked-shift":
		switchKind, _, valid := checkedArithmeticSpec(terminator.Operator, len(terminator.SwitchOperands))
		if !valid || switchKind != terminator.SwitchKind || len(terminator.Successors) != 2 {
			return false
		}
		if terminator.SwitchKind == "checked-arithmetic" {
			return hir.CheckedArithmeticType(terminator.SwitchType)
		}
		if terminator.SwitchKind == "checked-shift" {
			return hir.CheckedShiftType(terminator.SwitchType)
		}
		return terminator.SwitchType == hir.TypeI32
	default:
		return false
	}
}

func enumVariantNames(variants []EnumVariant) []string {
	result := make([]string, len(variants))
	for index := range variants {
		result[index] = variants[index].Name
	}
	return result
}

func checkedArithmeticSpec(operator string, operandCount int) (string, []string, bool) {
	switch {
	case operator == "-" && operandCount == 1:
		return "checked-unary", []string{"integer-overflow"}, true
	case (operator == "+" || operator == "-" || operator == "*") && operandCount == 2:
		return "checked-arithmetic", []string{"integer-overflow"}, true
	case operator == ">>" && operandCount == 2:
		return "checked-shift", []string{"invalid-shift"}, true
	case operator == "<<" && operandCount == 2:
		return "checked-shift", []string{"invalid-shift", "integer-overflow"}, true
	default:
		return "", nil, false
	}
}

func validCheckedArithmeticShape(operation Operation) bool {
	if len(operation.Operands) == 0 {
		return false
	}
	if operation.Operator == "+" || operation.Operator == "-" || operation.Operator == "*" {
		if len(operation.Operands) == 2 {
			return hir.CheckedArithmeticType(operation.OwnerType) && operation.Type == operation.OwnerType &&
				scalarOperandMatches(operation.Operands[0], operation.OwnerType) &&
				scalarOperandMatches(operation.Operands[1], operation.OwnerType)
		}
	}
	if len(operation.Operands) == 1 && operation.Operator == "-" {
		if operation.OwnerType != hir.TypeI32 || operation.Type != hir.TypeI32 {
			return false
		}
		return scalarOperandMatches(operation.Operands[0], hir.TypeI32)
	}
	if len(operation.Operands) == 2 && (operation.Operator == ">>" || operation.Operator == "<<") {
		return hir.CheckedShiftType(operation.OwnerType) && operation.Type == operation.OwnerType &&
			scalarOperandMatches(operation.Operands[0], operation.OwnerType) &&
			scalarOperandMatches(operation.Operands[1], operation.OwnerType)
	}
	return false
}

func validCheckedDivisionShape(operation Operation) bool {
	return len(operation.Operands) == 2 && (operation.Operator == "/" || operation.Operator == "%") &&
		hir.CheckedDivisionType(operation.OwnerType) && operation.Type == operation.OwnerType &&
		scalarOperandMatches(operation.Operands[0], operation.OwnerType) &&
		scalarOperandMatches(operation.Operands[1], operation.OwnerType)
}

func divisionTrapCategories(ownerType hir.Type) ([]string, bool) {
	if !hir.CheckedDivisionType(ownerType) {
		return nil, false
	}
	categories := []string{"division-by-zero"}
	if ownerType == hir.TypeI32 {
		categories = append(categories, "signed-division-overflow")
	}
	return categories, true
}

func validViewIndexShape(operation Operation, destination bool) bool {
	if len(operation.Operands) != 2 || operation.OwnerType != hir.SharedBorrow(hir.TypeByteSlice) ||
		operation.IndexType != hir.TypeUSize || operation.Type != hir.TypeU8 || operation.Operator != "[]" {
		return false
	}
	if destination != (operation.DestinationLocal != NoLocal) {
		return false
	}
	receiver := operation.Operands[0]
	index := operation.Operands[1]
	if receiver.Kind != OperandLocal && (receiver.Kind != OperandByteString || receiver.Type != operation.OwnerType) {
		return false
	}
	return index.Kind == OperandLocal || index.Kind == OperandUSize
}

func scalarCastOperandMatches(operand Operand, sourceType hir.Type) bool {
	if operand.Kind == OperandLocal {
		return true
	}
	return sourceType == hir.TypeU8 && operand.Kind == OperandU8 || sourceType == hir.TypeU32 && operand.Kind == OperandU32
}

func validBinaryShape(operation Operation) bool {
	if operation.DestinationLocal == NoLocal || len(operation.Operands) != 2 ||
		!scalarOperandMatches(operation.Operands[0], operation.OwnerType) ||
		!scalarOperandMatches(operation.Operands[1], operation.OwnerType) {
		return false
	}
	switch operation.Operator {
	case "==", "!=":
		_, _, pointer := hir.RawPointerElement(operation.OwnerType)
		return (hir.ScalarEqualityType(operation.OwnerType) || pointer) && operation.Type == hir.TypeBool
	case "<", "<=", ">", ">=":
		return hir.ScalarOrderedType(operation.OwnerType) && operation.Type == hir.TypeBool
	case "+", "-", "*":
		return hir.CheckedArithmeticType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "/", "%":
		return hir.CheckedDivisionType(operation.OwnerType) && operation.Type == operation.OwnerType
	case ">>", "<<":
		return hir.CheckedShiftType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "&", "^", "|":
		return hir.BitwiseIntegerType(operation.OwnerType) && operation.Type == operation.OwnerType
	default:
		return false
	}
}

func validUnaryShape(operation Operation) bool {
	if operation.DestinationLocal == NoLocal || len(operation.Operands) != 1 ||
		!scalarOperandMatches(operation.Operands[0], operation.OwnerType) {
		return false
	}
	switch operation.Operator {
	case "!":
		return operation.OwnerType == hir.TypeBool && operation.Type == hir.TypeBool
	case "+", "-":
		return operation.OwnerType == hir.TypeI32 && operation.Type == hir.TypeI32
	case "~":
		return hir.BitwiseNotType(operation.OwnerType) && operation.Type == operation.OwnerType
	default:
		return false
	}
}

func scalarOperandMatches(operand Operand, ownerType hir.Type) bool {
	if operand.Kind == OperandLocal {
		return true
	}
	if _, _, pointer := hir.RawPointerElement(ownerType); pointer {
		return operand.Kind == OperandNull && operand.Type == ownerType
	}
	switch ownerType {
	case hir.TypeBool, hir.TypeI32:
		return operand.Kind == OperandI32
	case hir.TypeU8:
		return operand.Kind == OperandU8
	case hir.TypeU32:
		return operand.Kind == OperandU32
	case hir.TypeUSize:
		return operand.Kind == OperandUSize
	default:
		return false
	}
}

func verifyTrapCheckEdges(function *Function) []VerificationError {
	blocks := make(map[BlockID]*Block, len(function.Blocks))
	predecessors := make(map[BlockID][]BlockID, len(function.Blocks))
	for index := range function.Blocks {
		block := &function.Blocks[index]
		blocks[block.ID] = block
		for _, successor := range block.Terminator.Successors {
			predecessors[successor] = append(predecessors[successor], block.ID)
		}
	}
	var failures []VerificationError
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex, operation := range block.Operations {
			switchKind, markerKind, categories, trapping := trappingOperationSpec(operation)
			if !trapping {
				continue
			}
			entity := operationEntity(block.ID, operationIndex, operation)
			incoming := predecessors[block.ID]
			if len(incoming) != 1 {
				failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "trapping operation does not have one checked predecessor"})
				continue
			}
			checkBlock := blocks[incoming[0]]
			if checkBlock == nil {
				failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "trapping operation is not dominated by its complete checked edge"})
				continue
			}
			markerEntity, markerValid := checkedMarkerStatus(checkBlock, markerKind, operation.Operator, operation.Operands, operation.OwnerType, operation.Type, categories)
			if !markerValid {
				if markerEntity == "" {
					markerEntity = entity
				}
				failures = append(failures, VerificationError{Code: "V1501", Entity: markerEntity, Detail: "trapping operation is missing its complete check marker"})
				continue
			}
			if !checkedTerminatorMatches(checkBlock.Terminator, block.ID, switchKind, operation.Operator, operation.Operands, operation.OwnerType) ||
				!checkedTrapSuccessorMatches(blocks, checkBlock.Terminator, categories) {
				failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "trapping operation is not dominated by its complete checked edge"})
			}
		}
	}
	return failures
}

func trappingOperationSpec(operation Operation) (string, OperationKind, []string, bool) {
	if operation.Kind == OperationViewIndex {
		return "checked-view-index", OperationViewIndexCheck, []string{"bounds"}, validViewIndexShape(operation, true)
	}
	if operation.Kind == OperationUnary {
		switchKind, categories, valid := checkedArithmeticSpec(operation.Operator, len(operation.Operands))
		return switchKind, OperationArithmeticCheck, categories, valid && operation.OwnerType == hir.TypeI32 && operation.Type == hir.TypeI32
	}
	if operation.Kind != OperationBinary {
		return "", "", nil, false
	}
	if operation.Operator == "/" || operation.Operator == "%" {
		categories, valid := divisionTrapCategories(operation.OwnerType)
		return "checked-division", OperationCheckedDivision, categories, valid && validBinaryShape(operation)
	}
	switchKind, categories, valid := checkedArithmeticSpec(operation.Operator, len(operation.Operands))
	return switchKind, OperationArithmeticCheck, categories, valid && validBinaryShape(operation)
}

func checkedTerminatorMatches(terminator Terminator, continuation BlockID, switchKind, operator string, operands []Operand, ownerType hir.Type) bool {
	if terminator.Kind != TerminatorSwitch || terminator.SwitchKind != switchKind || terminator.Operator != operator ||
		len(terminator.Successors) != 2 || terminator.Successors[0] != continuation || !slices.Equal(terminator.SwitchOperands, operands) {
		return false
	}
	if switchKind == "checked-unary" || switchKind == "checked-arithmetic" || switchKind == "checked-division" || switchKind == "checked-shift" {
		return terminator.SwitchType == ownerType
	}
	return true
}

func checkedMarkerStatus(block *Block, markerKind OperationKind, operator string, operands []Operand, ownerType, resultType hir.Type, categories []string) (string, bool) {
	for index := len(block.Operations) - 1; index >= 0; index-- {
		operation := block.Operations[index]
		if operation.Kind == markerKind && operation.Operator == operator && slices.Equal(operation.Operands, operands) {
			typeMatches := (markerKind != OperationArithmeticCheck && markerKind != OperationCheckedDivision) ||
				operation.OwnerType == ownerType && operation.Type == resultType
			return operationEntity(block.ID, index, operation), typeMatches && slices.Equal(operation.TrapCategories, categories)
		}
	}
	return "", false
}

func checkedTrapSuccessorMatches(blocks map[BlockID]*Block, terminator Terminator, categories []string) bool {
	if len(terminator.Successors) != 2 {
		return false
	}
	trap := blocks[terminator.Successors[1]]
	return trap != nil && trap.Terminator.Kind == TerminatorTrap && slices.Equal(trap.Terminator.TrapCategories, categories)
}

func verifyOperands(state verificationState, operands []Operand, entity string) []VerificationError {
	var failures []VerificationError
	for _, operand := range operands {
		switch operand.Kind {
		case OperandInvalid:
		case OperandLocal:
			if operand.Local < 0 || !state.live[hir.SymbolID(operand.Local)] {
				failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: fmt.Sprintf("local %d used before initialization", operand.Local)})
			}
		case OperandI32:
		case OperandU8:
			if operand.U64 > 255 {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "u8 operand is outside its value range"})
			}
		case OperandU32:
			if operand.U64 > uint64(^uint32(0)) {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "u32 operand is outside its value range"})
			}
		case OperandUSize:
		case OperandNull:
			if _, _, pointer := hir.RawPointerElement(operand.Type); !pointer {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "null operand has no raw-pointer type"})
			}
		case OperandByteString:
			if operand.Type != hir.SharedBorrow(hir.TypeByteSlice) {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "byte-string operand has invalid static storage"})
			}
		case OperandString:
			if operand.Type != hir.SharedBorrow(hir.TypeStr) || !validUTF8StringStaticData(operand.StaticData) {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "string operand has invalid static storage"})
			}
		case OperandCString:
			if operand.Type != hir.SharedBorrow(hir.TypeCStr) || !validCStringStaticData(operand.StaticData) {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "C string operand has invalid static storage"})
			}
		default:
			failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: fmt.Sprintf("operand has unknown kind %q", operand.Kind)})
		}
	}
	return failures
}

func uniqueNonEmpty(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func containsDefinedResource(value hir.Type, definitions map[hir.Type]*ResourceType) bool {
	if definitions[value] != nil {
		return true
	}
	if element, option := hir.OptionElement(value); option {
		return containsDefinedResource(element, definitions)
	}
	success, failure, result := hir.ResultElements(value)
	return result && (containsDefinedResource(success, definitions) || containsDefinedResource(failure, definitions))
}

func resourceFieldNames(fields []ResourceField) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func structFieldNames(fields []StructField) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func resourceField(fields []ResourceField, name string) (ResourceField, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return ResourceField{}, false
}

func structField(fields []StructField, name string) (StructField, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return StructField{}, false
}

func cloneVerificationState(input verificationState) verificationState {
	result := verificationState{
		live:    cloneLive(input.live),
		flags:   make(map[string]flagState, len(input.flags)),
		regions: make(map[hir.SymbolID]string, len(input.regions)),
	}
	for entity, state := range input.flags {
		result.flags[entity] = state
	}
	for symbol, entity := range input.regions {
		result.regions[symbol] = entity
	}
	return result
}

func mergeVerificationState(previous, incoming verificationState, resources map[hir.SymbolID]ResourceLocal) (verificationState, bool, bool) {
	if len(previous.flags) != len(incoming.flags) || len(previous.regions) != len(incoming.regions) {
		return verificationState{}, false, true
	}
	for entity, state := range previous.flags {
		if incoming.flags[entity] != state {
			return verificationState{}, false, true
		}
	}
	for symbol, entity := range previous.regions {
		if incoming.regions[symbol] != entity {
			return verificationState{}, false, true
		}
	}
	merged := cloneVerificationState(previous)
	changed := false
	keys := make(map[hir.SymbolID]struct{}, len(previous.live)+len(incoming.live))
	for symbol := range previous.live {
		keys[symbol] = struct{}{}
	}
	for symbol := range incoming.live {
		keys[symbol] = struct{}{}
	}
	for symbol := range keys {
		left := previous.live[symbol]
		right := incoming.live[symbol]
		if _, resource := resources[symbol]; resource {
			if left != right {
				return verificationState{}, false, true
			}
			continue
		}
		value := left && right
		if merged.live[symbol] != value {
			merged.live[symbol] = value
			changed = true
		}
	}
	return merged, changed, false
}

func sameLiveState(left, right map[hir.SymbolID]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for symbol, live := range left {
		if right[symbol] != live {
			return false
		}
	}
	return true
}

func blockEntityFor(function *Function, id BlockID) string {
	if int(id) >= 0 && int(id) < len(function.Blocks) && function.Blocks[id].Entity != "" {
		return function.Blocks[id].Entity
	}
	return fmt.Sprintf("block:%d", id)
}

func operationEntity(block BlockID, index int, operation Operation) string {
	return fallbackEntity(operation.Entity, fmt.Sprintf("block:%d.operation:%d", block, index))
}

func terminatorEntity(block BlockID, terminator Terminator) string {
	return fallbackEntity(terminator.Entity, fmt.Sprintf("block:%d.terminator", block))
}

func resourceEntity(resource ResourceLocal, fallback string) string {
	return fallbackEntity(resource.Entity, fallback)
}

func fallbackEntity(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

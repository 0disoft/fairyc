package mir

import "github.com/0disoft/fairyc/internal/hir"

func (v *operationVerifier) verifyPointerOperation(state verificationState, operation Operation, entity string) []VerificationError {
	unsafeByEntity := v.unsafeByEntity
	var failures []VerificationError
	switch operation.Kind {
	case OperationRawAddress:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		element, mutable, pointer := hir.RawPointerElement(operation.Type)
		shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 &&
			operation.Operands[0].Kind == OperandLocal && operation.OwnerType != hir.ParseType("") && operation.OwnerType != hir.TypeInvalid &&
			pointer && (element == operation.OwnerType || element == hir.TypeVoid) &&
			(operation.Operator == "&" && !mutable || operation.Operator == "&mut")
		if !shapeValid {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "raw address shape is invalid"})
		}
		provenance, hasProvenance := unsafeByEntity[operation.Entity]
		expectedEntity := sourceEntity("raw-address", operation.Span)
		sourceSpanEntity := sourceEntity("span", operation.Span)
		scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
		scopeContainsOperation := sameSpanSource(operation.UnsafeScope, operation.Span) &&
			operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
		if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
			provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
			failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "raw address lacks matching unsafe provenance"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationRawBorrowPointer:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(operation.OwnerType)
		targetElement, targetMutable, targetPointer := hir.RawPointerElement(operation.Type)
		shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 &&
			operation.Operands[0].Kind == OperandLocal && sourceBorrow && targetPointer &&
			(targetElement == sourceElement || targetElement == hir.TypeVoid) &&
			(sourceMutable || !targetMutable) && operation.Operator == "as"
		if !shapeValid {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "raw borrow pointer shape is invalid"})
		}
		provenance, hasProvenance := unsafeByEntity[operation.Entity]
		expectedEntity := sourceEntity("raw-borrow-pointer", operation.Span)
		sourceSpanEntity := sourceEntity("span", operation.Span)
		scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
		scopeContainsOperation := sameSpanSource(operation.UnsafeScope, operation.Span) &&
			operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
		if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
			provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
			failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "raw borrow pointer lacks matching unsafe provenance"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationStrFromRawParts, OperationCStrFromRawParts, OperationByteSliceFromRawParts, OperationMutByteSliceFromRawParts:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		_, ownerMutable, ownerBorrow := hir.BorrowElement(operation.OwnerType)
		wantType := hir.SharedBorrow(hir.TypeStr)
		wantOperator := "str::from_raw_parts"
		entityKind := "str-from-raw-parts"
		wantOwnerMutable := false
		if operation.Kind == OperationCStrFromRawParts {
			wantType = hir.SharedBorrow(hir.TypeCStr)
			wantOperator = "CStr::from_raw_parts"
			entityKind = "cstr-from-raw-parts"
		} else if operation.Kind == OperationByteSliceFromRawParts {
			wantType = hir.SharedBorrow(hir.TypeByteSlice)
			wantOperator = "byte_slice::from_raw_parts"
			entityKind = "byte-slice-from-raw-parts"
		} else if operation.Kind == OperationMutByteSliceFromRawParts {
			wantType = hir.MutableBorrow(hir.TypeByteSlice)
			wantOperator = "byte_slice::from_raw_parts_mut"
			entityKind = "mut-byte-slice-from-raw-parts"
			wantOwnerMutable = true
		}
		shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 3 &&
			operation.Operands[0].Kind == OperandLocal && ownerBorrow && ownerMutable == wantOwnerMutable &&
			operation.Type == wantType && operation.Operator == wantOperator
		if !shapeValid {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "borrowed-view raw-parts shape is invalid"})
		}
		provenance, hasProvenance := unsafeByEntity[operation.Entity]
		expectedEntity := sourceEntity(entityKind, operation.Span)
		sourceSpanEntity := sourceEntity("span", operation.Span)
		scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
		scopeContainsOperation := sameSpanSource(operation.UnsafeScope, operation.Span) &&
			operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
		if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
			provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
			failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "borrowed-view raw-parts construction lacks matching unsafe provenance"})
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
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "string data address shape is invalid"})
		}
		provenance, hasProvenance := unsafeByEntity[operation.Entity]
		expectedEntity := sourceEntity("str-data-address", operation.Span)
		sourceSpanEntity := sourceEntity("span", operation.Span)
		scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
		scopeContainsOperation := sameSpanSource(operation.UnsafeScope, operation.Span) &&
			operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
		if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
			provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
			failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "string data address lacks matching unsafe provenance"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	case OperationByteSliceDataAddress:
		failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
		element, mutable, pointer := hir.RawPointerElement(operation.Type)
		ownerElement, ownerMutable, ownerBorrow := hir.BorrowElement(operation.OwnerType)
		staticStorageValid := len(operation.Operands) == 1 &&
			(operation.Operands[0].Kind != OperandByteString || !ownerMutable)
		shapeValid := operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 &&
			(operation.Operands[0].Kind == OperandByteString || operation.Operands[0].Kind == OperandLocal) &&
			ownerBorrow && ownerElement == hir.TypeByteSlice && staticStorageValid &&
			pointer && element == hir.TypeU8 && (ownerMutable || !mutable) && operation.Operator == "as"
		if !shapeValid {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "byte-slice data address shape is invalid"})
		}
		provenance, hasProvenance := unsafeByEntity[operation.Entity]
		expectedEntity := sourceEntity("byte-slice-data-address", operation.Span)
		sourceSpanEntity := sourceEntity("span", operation.Span)
		scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
		scopeContainsOperation := sameSpanSource(operation.UnsafeScope, operation.Span) &&
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
			(operation.Operands[0].Kind == OperandCString || operation.Operands[0].Kind == OperandLocal) &&
			operation.OwnerType == hir.SharedBorrow(hir.TypeCStr) &&
			pointer && element == hir.TypeU8 && !mutable && operation.Operator == "as"
		if !shapeValid {
			failures = append(failures, VerificationError{Code: CodeOperationShape, Entity: entity, Detail: "C string data address shape is invalid"})
		}
		provenance, hasProvenance := unsafeByEntity[operation.Entity]
		expectedEntity := sourceEntity("cstr-data-address", operation.Span)
		sourceSpanEntity := sourceEntity("span", operation.Span)
		scopeEntity := sourceEntity("unsafe-scope", operation.UnsafeScope)
		scopeContainsOperation := sameSpanSource(operation.UnsafeScope, operation.Span) &&
			operation.UnsafeScope.StartByte <= operation.Span.StartByte && operation.UnsafeScope.EndByte >= operation.Span.EndByte
		if operation.Entity != expectedEntity || !scopeContainsOperation || !hasProvenance ||
			provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
			failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "C string data address lacks matching unsafe provenance"})
		}
		state.live[hir.SymbolID(operation.DestinationLocal)] = true
	}
	return failures
}

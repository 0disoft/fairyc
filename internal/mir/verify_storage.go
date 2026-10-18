package mir

import "github.com/0disoft/fairyc/internal/admission"

func verifyStorageOperations(function *Function, properties typeProperties, capabilities admission.Capabilities) []VerificationError {
	var failures []VerificationError
	for _, block := range function.Blocks {
		for index, operation := range block.Operations {
			var admitted, valid, transfer bool
			switch operation.Kind {
			case OperationStorageLayout:
				admitted = capabilities.AllowsStorageQueries()
				valid = validStorageQueryShape(operation) && storageQueryType(operation.OwnerType, properties)
			case OperationStorageTake:
				transfer = true
				valid = validStorageTakeShape(operation) && storageQueryType(operation.Type, properties)
			case OperationStoragePut:
				transfer = true
				valid = validStoragePutShape(operation) && storageQueryType(operation.OwnerType, properties)
			case OperationStorageBorrow, OperationStorageBorrowMut:
				transfer = true
				valid = validStorageBorrowShape(operation, properties)
			case OperationStorageElementAddress:
				transfer = true
				valid = validStorageElementAddressShape(operation, properties)
			default:
				continue
			}
			if transfer {
				admitted = capabilities.AllowsStorageTransfers()
			}
			entity := operationEntity(block.ID, index, operation)
			if !admitted {
				failures = append(failures, VerificationError{Code: CodeStorageAdmission, Entity: entity, Detail: "private " + string(operation.Kind) + " is not admitted"})
			}
			if !valid {
				failures = append(failures, VerificationError{Code: CodeStorageShape, Entity: entity, Detail: "private " + string(operation.Kind) + " has an invalid shape or element type"})
			}
			if transfer && !validStorageTransferProvenance(function, operation) {
				failures = append(failures, VerificationError{Code: CodeStorageProvenance, Entity: entity, Detail: "private " + string(operation.Kind) + " lacks matching unsafe provenance"})
			}
		}
	}
	return failures
}

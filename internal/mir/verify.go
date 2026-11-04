package mir

import (
	"context"
	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/hir"
)

// Verify checks the executable invariants required before MIR handoff.
func Verify(program Program) []VerificationError {
	return VerifyWithCapabilities(program, admission.Capabilities{})
}

// VerifyStorageQueries additionally admits compiler-private storage queries.
func VerifyStorageQueries(program Program) []VerificationError {
	return VerifyWithCapabilities(program, admission.Capabilities{Storage: admission.StorageQueries})
}

// VerifyStorageTransfers additionally admits unsafe storage ownership transfers.
func VerifyStorageTransfers(program Program) []VerificationError {
	return VerifyWithCapabilities(program, admission.Capabilities{Storage: admission.StorageTransfers})
}

// VerifyWithCapabilities uses the source-preparation admission policy. Invalid
// capabilities fail before executable IR is accepted.
func VerifyWithCapabilities(program Program, capabilities admission.Capabilities) []VerificationError {
	failures, _ := VerifyWithCapabilitiesContext(context.Background(), program, capabilities)
	return failures
}

func VerifyWithCapabilitiesContext(ctx context.Context, program Program, capabilities admission.Capabilities) (failures []VerificationError, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			failures, err = nil, canceled
		}
	}()
	if err := capabilities.Validate(); err != nil {
		return []VerificationError{{Code: CodeAdmission, Entity: "program", Detail: err.Error()}}, nil
	}
	enumTypes, enumFailures := verifyEnumTypes(program.EnumTypes)
	if verificationReportsFull(enumFailures) {
		return canonicalVerificationErrors(enumFailures), nil
	}
	structTypes, failures := verifyStructTypes(program.StructTypes, enumTypes)
	failures = appendVerificationErrors(failures, enumFailures...)
	if verificationReportsFull(failures) {
		return canonicalVerificationErrors(failures), nil
	}
	resourceTypes, resourceFailures := verifyResourceTypes(program.ResourceTypes)
	failures = appendVerificationErrors(failures, resourceFailures...)
	if verificationReportsFull(failures) {
		return canonicalVerificationErrors(failures), nil
	}
	// Declaration order keeps truncated diagnostics independent of map iteration.
	for index := range program.StructTypes {
		definition := &program.StructTypes[index]
		value := definition.Type
		if structTypes[value] != definition {
			continue
		}
		if _, collision := resourceTypes[value]; collision {
			failures = appendVerificationErrors(failures, VerificationError{Code: CodeAggregateIdentity, Entity: structTypeEntity(value), Detail: "aggregate type is both struct and resource"})
			if verificationReportsFull(failures) {
				return canonicalVerificationErrors(failures), nil
			}
		}
		if _, collision := enumTypes[value]; collision {
			failures = appendVerificationErrors(failures, VerificationError{Code: CodeAggregateIdentity, Entity: structTypeEntity(value), Detail: "aggregate type is both struct and enum"})
			if verificationReportsFull(failures) {
				return canonicalVerificationErrors(failures), nil
			}
		}
	}
	for index := range program.EnumTypes {
		definition := &program.EnumTypes[index]
		value := definition.Type
		if enumTypes[value] != definition {
			continue
		}
		if _, collision := resourceTypes[value]; collision {
			failures = appendVerificationErrors(failures, VerificationError{Code: CodeAggregateIdentity, Entity: enumTypeEntity(value), Detail: "aggregate type is both enum and resource"})
			if verificationReportsFull(failures) {
				return canonicalVerificationErrors(failures), nil
			}
		}
	}
	foreignFunctions, foreignFailures := verifyForeignFunctions(program.ForeignFunctions)
	failures = appendVerificationErrors(failures, foreignFailures...)
	if verificationReportsFull(failures) {
		return canonicalVerificationErrors(failures), nil
	}
	functions, functionFailures := verifyFunctionDefinitions(program.Functions, foreignFunctions)
	failures = appendVerificationErrors(failures, functionFailures...)
	if verificationReportsFull(failures) {
		return canonicalVerificationErrors(failures), nil
	}
	failures = appendVerificationErrors(failures, verifyDropGlueLinks(program.ResourceTypes, program.Functions, functions)...)
	if verificationReportsFull(failures) {
		return canonicalVerificationErrors(failures), nil
	}
	failures = appendVerificationErrors(failures, verifyBoundedCallbacks(program, functions, foreignFunctions)...)
	if verificationReportsFull(failures) {
		return canonicalVerificationErrors(failures), nil
	}
	properties := newTypeProperties(program.ResourceTypes, program.StructTypes, program.EnumTypes)
	// Resource definitions are immutable within this verification. Reuse required
	// cleanup effects across functions; each caller still checks its own authority.
	dropEffects := make(map[hir.Type][]hir.Effect)
	var fields aggregateFieldIndex
	for index := range program.Functions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		failures = appendVerificationErrors(failures, verifyInlineFunction(&program.Functions[index])...)
		if verificationReportsFull(failures) {
			return canonicalVerificationErrors(failures), nil
		}
		failures = appendVerificationErrors(failures, verifyStorageOperations(&program.Functions[index], properties, capabilities)...)
		if verificationReportsFull(failures) {
			return canonicalVerificationErrors(failures), nil
		}
		failures = appendVerificationErrors(failures, verifyStructViewTypes(&program.Functions[index], properties)...)
		if verificationReportsFull(failures) {
			return canonicalVerificationErrors(failures), nil
		}
		failures = appendVerificationErrors(failures, verifyFunctionContext(ctx, &program.Functions[index], structTypes, enumTypes, resourceTypes, foreignFunctions, functions, dropEffects, &fields)...)
		if verificationReportsFull(failures) {
			return canonicalVerificationErrors(failures), nil
		}
	}
	return canonicalVerificationErrors(failures), nil
}

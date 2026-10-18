package mir

import "github.com/0disoft/fairyc/internal/admission"

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
	if err := capabilities.Validate(); err != nil {
		return []VerificationError{{Code: CodeAdmission, Entity: "program", Detail: err.Error()}}
	}
	structTypes, failures := verifyStructTypes(program.StructTypes)
	enumTypes, enumFailures := verifyEnumTypes(program.EnumTypes)
	failures = append(failures, enumFailures...)
	resourceTypes, resourceFailures := verifyResourceTypes(program.ResourceTypes)
	failures = append(failures, resourceFailures...)
	for value := range structTypes {
		if _, collision := resourceTypes[value]; collision {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: structTypeEntity(value), Detail: "aggregate type is both struct and resource"})
		}
		if _, collision := enumTypes[value]; collision {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: structTypeEntity(value), Detail: "aggregate type is both struct and enum"})
		}
	}
	for value := range enumTypes {
		if _, collision := resourceTypes[value]; collision {
			failures = append(failures, VerificationError{Code: CodeAggregateIdentity, Entity: enumTypeEntity(value), Detail: "aggregate type is both enum and resource"})
		}
	}
	foreignFunctions, foreignFailures := verifyForeignFunctions(program.ForeignFunctions)
	failures = append(failures, foreignFailures...)
	functions, functionFailures := verifyFunctionDefinitions(program.Functions, foreignFunctions)
	failures = append(failures, functionFailures...)
	failures = append(failures, verifyDropGlueLinks(program.ResourceTypes, program.Functions, functions)...)
	properties := newTypeProperties(program.ResourceTypes, program.StructTypes, program.EnumTypes)
	for index := range program.Functions {
		failures = append(failures, verifyStorageOperations(&program.Functions[index], properties, capabilities)...)
		failures = append(failures, verifyStructViewTypes(&program.Functions[index], properties)...)
		failures = append(failures, verifyFunction(&program.Functions[index], structTypes, enumTypes, resourceTypes, foreignFunctions, functions)...)
	}
	return canonicalVerificationErrors(failures)
}

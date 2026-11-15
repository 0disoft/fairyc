package contractprofile

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

type profileTypeFindings uint8

const (
	profileRawType profileTypeFindings = 1 << iota
	profileFloatType
	profileCScalarType
	profileUSizeType

	profileTargetTypes = profileCScalarType | profileUSizeType
	profileAllTypes    = profileRawType | profileFloatType | profileTargetTypes
)

func (findings profileTypeFindings) withType(value hir.Type, checked profileTypeFindings) profileTypeFindings {
	remaining := checked &^ findings
	if remaining == 0 || value.String() == "" {
		return findings
	}
	if remaining&profileRawType != 0 && containsRawType(value) {
		findings |= profileRawType
	}
	if remaining&profileFloatType != 0 && containsFloatType(value) {
		findings |= profileFloatType
	}
	if remaining&profileCScalarType != 0 && containsCScalarType(value) {
		findings |= profileCScalarType
	}
	if remaining&profileUSizeType != 0 && containsTypeToken(value.String(), hir.TypeUSize.String()) {
		findings |= profileUSizeType
	}
	return findings
}

func operationTypeFindings(operation mir.Operation) profileTypeFindings {
	var findings profileTypeFindings
	if rawPointerOperation(operation) {
		findings |= profileRawType
	}
	for _, value := range [...]hir.Type{operation.Type, operation.OwnerType, operation.IndexType} {
		findings = findings.withType(value, profileAllTypes)
	}
	for _, operand := range operation.Operands {
		findings = findings.withType(operand.Type, profileAllTypes)
	}
	return findings
}

func terminatorTypeFindings(terminator mir.Terminator) profileTypeFindings {
	var findings profileTypeFindings
	for _, operand := range terminator.Arguments {
		findings = findings.withType(operand.Type, profileAllTypes)
	}
	findings = findings.withType(terminator.ReturnValue.Type, profileAllTypes)
	// Switch metadata is not call/return transport. Preserve the existing policy:
	// its type participates in float/target checks, its operands only in target checks.
	findings = findings.withType(terminator.SwitchType, profileFloatType|profileTargetTypes)
	for _, operand := range terminator.SwitchOperands {
		findings = findings.withType(operand.Type, profileTargetTypes)
	}
	return findings
}

func appendOperationTypeViolations(report *Report, owner string, operation mir.Operation) {
	appendMIRTypeViolations(report, owner, operation.Span, operationTypeFindings(operation), "MIR operation",
		"raw-pointer MIR operation is unavailable in the deterministic contract profile",
		"floating-point MIR operation is unavailable until every backend shares exact semantics")
}

func appendTerminatorTypeViolations(report *Report, owner string, terminator mir.Terminator) {
	appendMIRTypeViolations(report, owner, terminator.Span, terminatorTypeFindings(terminator), "MIR terminator",
		"raw-pointer call or return transport is unavailable in the deterministic contract profile",
		"floating-point call or return transport is unavailable until every backend shares exact semantics")
}

func appendMIRTypeViolations(report *Report, owner string, span hir.Span, findings profileTypeFindings, context, rawMessage, floatMessage string) {
	if findings == 0 {
		return
	}
	if findings&profileRawType != 0 {
		report.Violations = append(report.Violations, violationAt("CP1003", owner, span, rawMessage))
	}
	if findings&profileFloatType != 0 {
		report.Violations = append(report.Violations, violationAt("CP1013", owner, span, floatMessage))
	}
	if findings&profileCScalarType != 0 {
		report.Violations = append(report.Violations, violationAt("CP1014", owner, span, context+" contains a target-dependent c::* scalar"))
	}
	if findings&profileUSizeType != 0 {
		report.Violations = append(report.Violations, violationAt("CP1015", owner, span, context+" contains target-width usize"))
	}
}

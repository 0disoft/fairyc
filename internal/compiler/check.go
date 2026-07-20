// Package compiler coordinates the shared Stage 0 semantic-to-verified-MIR boundary.
package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// Stage identifies the compiler boundary that produced an internal pipeline error.
type Stage string

const (
	StagePackageInput Stage = "package-input"
	StageMIRLower     Stage = "mir-lower"
	StageMIRVerify    Stage = "mir-verify"
)

// Error reports an internal pipeline failure separately from ordinary source diagnostics and
// deliberately unsupported semantic forms, which remain in Result.Semantic.
type Error struct {
	Stage        Stage
	Cause        error
	Verification []mir.VerificationError
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Stage, e.Cause)
	}
	if len(e.Verification) != 0 {
		return fmt.Sprintf("%s: %v", e.Stage, e.Verification[0])
	}
	return string(e.Stage)
}

// Unwrap exposes a lowering or package-input cause when one exists.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Result contains the semantic result and, only after successful verification, executable MIR.
type Result struct {
	Semantic semantic.Result
	MIR      mir.Program
	Verified bool
}

// Check analyzes one explicit single-package source closure and verifies its lowered MIR when
// semantic analysis completes successfully.
func Check(files []semantic.File) (Result, error) {
	return lowerVerified(semantic.Analyze(files))
}

// CheckPackages analyzes one already-resolved package input and verifies its lowered MIR without
// choosing a manifest, solver, source-acquisition, or stable package-identity format.
func CheckPackages(input semantic.PackageInput) (Result, error) {
	semanticResult, err := semantic.AnalyzePackages(input)
	if err != nil {
		return Result{}, &Error{Stage: StagePackageInput, Cause: err}
	}
	return lowerVerified(semanticResult)
}

func lowerVerified(semanticResult semantic.Result) (Result, error) {
	result := Result{Semantic: semanticResult}
	if semanticResult.CompletedThrough != semantic.PhaseFlow || semanticResult.FailedPhase != semantic.PhaseNone ||
		len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return result, nil
	}
	program, err := mir.Lower(semanticResult.HIR)
	if err != nil {
		return result, &Error{Stage: StageMIRLower, Cause: err}
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return result, &Error{Stage: StageMIRVerify, Verification: failures}
	}
	result.MIR = program
	result.Verified = true
	return result, nil
}

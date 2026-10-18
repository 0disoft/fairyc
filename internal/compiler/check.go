// Package compiler coordinates the shared Stage 0 semantic-to-verified-MIR boundary.
package compiler

import (
	"context"
	"errors"
	"fmt"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// Stage identifies the compiler boundary that produced an internal pipeline error.
type Stage string

const (
	StageSemantic      Stage = "semantic"
	StagePackageInput  Stage = "package-input"
	StageMIRLower      Stage = "mir-lower"
	StageMIRVerify     Stage = "mir-verify"
	StageStateMIRLower Stage = "contract-state-mir-lower"
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

// Result contains the semantic result and, only after every executable IR
// boundary verifies, ordinary MIR plus compiler-private contract-state MIR.
type Result struct {
	GenericProvenance GenericProvenance `json:"-"`
	Semantic          semantic.Result
	MIR               mir.Program
	StateTransactions []contractstatemir.LoweredTransaction
	Verified          bool
}

// Analysis is the completed semantic half of one compiler request. It deliberately contains no
// executable MIR, so process-local cache coordinators can inspect typed package interfaces before
// deciding whether ordinary MIR lowering is necessary.
type Analysis struct {
	GenericProvenance GenericProvenance `json:"-"`
	Semantic          semantic.Result
}

// Check analyzes one explicit single-package source closure and verifies its lowered MIR when
// semantic analysis completes successfully.
func Check(files []semantic.File) (Result, error) { return CheckContext(context.Background(), files) }

func CheckContext(ctx context.Context, files []semantic.File) (Result, error) {
	analyzed, err := semantic.AnalyzeContext(ctx, files)
	if err != nil {
		return Result{}, &Error{Stage: StageSemantic, Cause: err}
	}
	return lowerWithCapabilitiesContext(ctx, analyzed, admission.Capabilities{})
}

// CheckPackages analyzes one already-resolved package input and verifies its lowered MIR without
// choosing a manifest, solver, source-acquisition, or stable package-identity format.
func CheckPackages(input semantic.PackageInput) (Result, error) {
	return CheckPackagesContext(context.Background(), input)
}

func CheckPackagesContext(ctx context.Context, input semantic.PackageInput) (Result, error) {
	analysis, err := AnalyzePackagesContext(ctx, input)
	if err != nil {
		return Result{}, err
	}
	return LowerAnalyzedContext(ctx, analysis)
}

// AnalyzePackages runs the package-aware semantic pipeline without lowering MIR. Package input
// failures remain internal pipeline errors; ordinary source diagnostics remain in Analysis.
func AnalyzePackages(input semantic.PackageInput) (Analysis, error) {
	return AnalyzePackagesContext(context.Background(), input)
}

func AnalyzePackagesContext(ctx context.Context, input semantic.PackageInput) (Analysis, error) {
	semanticResult, err := semantic.AnalyzePackagesContext(ctx, input)
	if err != nil {
		return Analysis{}, &Error{Stage: StagePackageInput, Cause: err}
	}
	return Analysis{Semantic: semanticResult}, nil
}

// LowerAnalyzed lowers one semantic result and publishes MIR only after the ordinary closed-world
// verifier accepts it. Cache coordinators must call this exact boundary on every fallback path.
func LowerAnalyzed(analysis Analysis) (Result, error) {
	return LowerAnalyzedContext(context.Background(), analysis)
}

func LowerAnalyzedContext(ctx context.Context, analysis Analysis) (Result, error) {
	return LowerAnalyzedWithCapabilitiesContext(ctx, analysis, admission.Capabilities{})
}

// LowerAnalyzedWithCapabilities applies the source admission at MIR handoff.
func LowerAnalyzedWithCapabilities(analysis Analysis, capabilities admission.Capabilities) (Result, error) {
	return LowerAnalyzedWithCapabilitiesContext(context.Background(), analysis, capabilities)
}

func LowerAnalyzedWithCapabilitiesContext(ctx context.Context, analysis Analysis, capabilities admission.Capabilities) (Result, error) {
	result, err := lowerWithCapabilitiesContext(ctx, analysis.Semantic, capabilities)
	if err == nil && result.Verified {
		result.GenericProvenance = analysis.GenericProvenance
	}
	return result, err
}

func lowerVerified(semanticResult semantic.Result) (Result, error) {
	return lowerWithCapabilities(semanticResult, admission.Capabilities{})
}

func lowerWithCapabilities(semanticResult semantic.Result, capabilities admission.Capabilities) (Result, error) {
	return lowerWithCapabilitiesContext(context.Background(), semanticResult, capabilities)
}

func lowerWithCapabilitiesContext(ctx context.Context, semanticResult semantic.Result, capabilities admission.Capabilities) (result Result, err error) {
	if err := ctx.Err(); err != nil {
		return Result{}, &Error{Stage: StageMIRLower, Cause: err}
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result = Result{}
			if err == nil {
				err = &Error{Stage: StageMIRLower, Cause: canceled}
			}
		}
	}()
	if err := capabilities.Validate(); err != nil {
		return Result{}, &Error{Stage: StageMIRVerify, Cause: err}
	}
	result = Result{Semantic: semanticResult}
	if semanticResult.CompletedThrough != semantic.PhaseFlow || semanticResult.FailedPhase != semantic.PhaseNone ||
		len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return result, nil
	}
	program, err := mir.LowerContext(ctx, semanticResult.HIR)
	if err != nil {
		var unsupported *mir.Unsupported
		if errors.As(err, &unsupported) {
			result.Semantic.Unsupported = []semantic.Unsupported{{Feature: unsupported.Feature, Span: unsupported.Span}}
			return result, nil
		}
		return result, &Error{Stage: StageMIRLower, Cause: err}
	}
	failures, err := mir.VerifyWithCapabilitiesContext(ctx, program, capabilities)
	if err != nil {
		return Result{}, &Error{Stage: StageMIRVerify, Cause: err}
	}
	if len(failures) != 0 {
		return result, &Error{Stage: StageMIRVerify, Verification: failures}
	}
	stateTransactions, err := contractstatemir.LowerHIRTransactions(semanticResult.HIR)
	if err != nil {
		return result, &Error{Stage: StageStateMIRLower, Cause: err}
	}
	result.MIR = program
	result.StateTransactions = stateTransactions
	result.Verified = true
	return result, nil
}

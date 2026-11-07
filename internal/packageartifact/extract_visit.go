package packageartifact

import (
	"context"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// VisitAllContext extracts and verifies one independently owned package at a
// time, in package-ID order. It retains no delivered artifact. The callback
// must finish publishing or consuming each artifact before returning to bound
// residency. Earlier callbacks may have completed if a later package fails.
func VisitAllContext(ctx context.Context, program mir.Program, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID, visit func(Artifact) error) error {
	return visitAllWithCapabilities(ctx, program, identities, coordinates, admission.Capabilities{}, visit)
}

// VisitAllStorageTransfersContext selects the same explicit private verifier
// contract as ExtractAllStorageTransfers, without retaining a complete slice.
func VisitAllStorageTransfersContext(ctx context.Context, program mir.Program, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID, visit func(Artifact) error) error {
	return visitAllWithCapabilities(ctx, program, identities, coordinates, admission.Capabilities{Storage: admission.StorageTransfers}, visit)
}

func visitAllWithCapabilities(ctx context.Context, program mir.Program, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID, capabilities admission.Capabilities, visit func(Artifact) error) error {
	verify := func(program mir.Program) []mir.VerificationError {
		failures, _ := mir.VerifyWithCapabilitiesContext(ctx, program, capabilities)
		return failures
	}
	return visitAll(ctx, program, identities, coordinates, verify, visit)
}

func visitAll(ctx context.Context, program mir.Program, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID, verifyMIR func(mir.Program) []mir.VerificationError, visit func(Artifact) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if visit == nil {
		return fmt.Errorf("package artifact visitor is nil")
	}
	if failures := verifyMIR(program); len(failures) != 0 {
		return fmt.Errorf("verify whole program before package extraction: %w", failures[0])
	}
	if err := rejectLegacyClosureValues(program); err != nil {
		return err
	}
	packages := make(map[hir.PackageID]struct{})
	for _, function := range program.Functions {
		packages[function.Span.Package] = struct{}{}
	}
	packageIDs := make([]hir.PackageID, 0, len(packages))
	for packageID := range packages {
		packageIDs = append(packageIDs, packageID)
	}
	slices.Sort(packageIDs)
	index := &extractionIndex{}
	for _, packageID := range packageIDs {
		if err := ctx.Err(); err != nil {
			return err
		}
		artifact, err := extractVerified(program, packageID, identities, coordinates, verifyMIR, index)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(artifact); err != nil {
			return err
		}
	}
	return ctx.Err()
}

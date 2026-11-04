package workspacecompiler

import (
	"context"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// Keep only owner membership, not the complete set of cloned package contexts.
// As before, additional lowered helper owners are checked but not published.
func visitOrdinaryArtifacts(ctx context.Context, program mir.Program, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID, owners []hir.PackageID, publish func(packageartifact.Artifact) error) error {
	pending := make(map[hir.PackageID]struct{}, len(owners))
	for _, id := range owners {
		pending[id] = struct{}{}
	}
	err := packageartifact.VisitAllContext(ctx, program, identities, coordinates, func(artifact packageartifact.Artifact) error {
		if _, wanted := pending[artifact.PackageID]; !wanted {
			return nil
		}
		delete(pending, artifact.PackageID)
		return publish(artifact)
	})
	if err != nil {
		return err
	}
	for _, id := range owners {
		if _, missing := pending[id]; missing {
			return fmt.Errorf("lowered MIR has no artifact for function-owning package %d", id)
		}
	}
	return nil
}

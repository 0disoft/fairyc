package workspacecompiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Reuse the identities already validated and computed by SemanticPackages.
// The detached maps belong to this check, so edits never reuse stale coordinates.
func packageIdentityMaps(metadata []workspaceloader.SemanticPackage) (map[hir.PackageID]packagegraph.InstanceID, map[hir.PackageID]packagegraph.CoordinateID, error) {
	identities := make(map[hir.PackageID]packagegraph.InstanceID, len(metadata))
	coordinates := make(map[hir.PackageID]packagegraph.CoordinateID, len(metadata))
	for _, loaded := range metadata {
		if loaded.ID == hir.DefaultPackage {
			return nil, nil, fmt.Errorf("workspace package id overflow")
		}
		identities[loaded.ID] = loaded.Instance
		coordinates[loaded.ID] = loaded.Coordinate
	}
	return identities, coordinates, nil
}

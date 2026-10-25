package packageartifact

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

type packageCoordinateBinding struct {
	id        hir.PackageID
	ambiguous bool
}

type packageResolver func(packagegraph.CoordinateID) (packageCoordinateBinding, bool)

func scanPackageCoordinate(coordinate packagegraph.CoordinateID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (binding packageCoordinateBinding, found bool) {
	for candidate, candidateCoordinate := range coordinates {
		if candidateCoordinate != coordinate {
			continue
		}
		if found && candidate != binding.id {
			binding.ambiguous = true
			return binding, true
		}
		binding.id, found = candidate, true
	}
	return binding, found
}

// indexedPackageResolver belongs to one rebind call. Build only when a nominal
// type or foreign owner needs lookup; unused ambiguous coordinates remain inert.
func indexedPackageResolver(coordinates map[hir.PackageID]packagegraph.CoordinateID) packageResolver {
	var index map[packagegraph.CoordinateID]packageCoordinateBinding
	return func(coordinate packagegraph.CoordinateID) (packageCoordinateBinding, bool) {
		if index == nil {
			index = make(map[packagegraph.CoordinateID]packageCoordinateBinding, len(coordinates))
			for id, coordinate := range coordinates {
				binding, found := index[coordinate]
				if !found {
					binding.id = id
				} else if binding.id != id {
					binding.ambiguous = true
				}
				index[coordinate] = binding
			}
		}
		binding, found := index[coordinate]
		return binding, found
	}
}

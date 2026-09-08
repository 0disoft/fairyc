package workspaceloader

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

// VectorCandidateStorageOwners verifies exact embedded source authority before
// selecting disposable compiler owner IDs. A claimed SDK name/digest is not proof.
func (workspace Workspace) VectorCandidateStorageOwners() ([]hir.PackageID, error) {
	if !workspace.vectorCandidate {
		return nil, fmt.Errorf("vector candidate source selection is not active")
	}
	return workspace.SDKStorageOwners()
}

// SDKStorageOwners authenticates the selected core-plus-vector source bundle.
// Module presence requests verification; it never grants authority by itself.
func (workspace Workspace) SDKStorageOwners() ([]hir.PackageID, error) {
	load, identity := sdkstdlib.LoadVector, sdkstdlib.SDKIdentity
	if workspace.vectorCandidate {
		load, identity = sdkstdlib.LoadVectorCandidate, sdkstdlib.VectorCandidateIdentity
	} else {
		present := false
		for _, pkg := range workspace.Packages {
			if pkg.Path != sdkstdlib.PackagePath {
				continue
			}
			for _, module := range pkg.Modules {
				present = present || module.Path == "std::vec"
			}
		}
		if !present {
			return nil, nil
		}
	}
	definition, err := load()
	if err != nil {
		return nil, err
	}
	count := 0
	for _, loaded := range workspace.Packages {
		if loaded.Path != sdkstdlib.PackagePath {
			if loaded.SourceKind == SourceKindSDK || loaded.SDKIdentity != "" {
				return nil, fmt.Errorf("SDK authority claimed by a non-SDK package")
			}
			continue
		}
		count++
		if count != 1 || loaded.SourceKind != SourceKindSDK || loaded.SDKIdentity != identity ||
			loaded.RuntimeABI != sdkstdlib.RuntimeABI || loaded.Digest != definition.Digest ||
			len(loaded.Sources) != len(definition.Sources) || !reflect.DeepEqual(loaded.Modules, definition.Modules) {
			return nil, fmt.Errorf("vector SDK catalog does not match embedded source")
		}
		for i, expected := range definition.Sources {
			actual := loaded.Sources[i]
			if actual.Test || actual.Path != expected.Path || !bytes.Equal(actual.Bytes, expected.Bytes) || !reflect.DeepEqual(actual.Module, expected.Module) {
				return nil, fmt.Errorf("vector SDK source or AST differs from embedded source")
			}
		}
	}
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		return nil, err
	}
	var owners []hir.PackageID
	for _, loaded := range metadata {
		if loaded.Path == sdkstdlib.PackagePath {
			if count != 1 || loaded.Role != packagegraph.RoleStandardLibrary || loaded.Source != definition.Digest || loaded.RuntimeABI != sdkstdlib.RuntimeABI {
				return nil, fmt.Errorf("resolved SDK owner has no exact source catalog")
			}
			owners = append(owners, loaded.ID)
		}
	}
	return owners, nil
}

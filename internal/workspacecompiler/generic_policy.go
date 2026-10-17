package workspacecompiler

import (
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func sdkStorageCapabilities() admission.Capabilities {
	return admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments, Templates: admission.SDKOwnerTemplates}
}

// Bind SDK authority anew for every check, after the loader authenticates the
// current source. A session's configured profile is never itself source authority.
func workspaceAdmission(workspace workspaceloader.Workspace, capabilities admission.Capabilities) (admission.Policy[hir.PackageID], error) {
	if err := capabilities.Validate(); err != nil {
		return admission.Policy[hir.PackageID]{}, err
	}
	policy := admission.Policy[hir.PackageID]{Capabilities: capabilities}
	if capabilities.SDKTemplates() {
		owners, err := workspace.SDKStorageOwners()
		if err != nil {
			return admission.Policy[hir.PackageID]{}, err
		}
		if len(owners) == 0 {
			return admission.Policy[hir.PackageID]{}, nil
		}
		policy.StorageOwners = owners
	}
	return policy, policy.Validate()
}

// genericArtifactPolicy selects the matching codec, verifier and identity family
// together. These remain distinct formats; a capability change cannot mix them.
type genericArtifactPolicy struct {
	newMemory     func(packagecache.Limits, uint64) (*packageartifact.GenericCache, error)
	newDisk       func(string, packagecache.DiskLimits, uint64) (*packageartifact.GenericStableCache, error)
	newDiskInRoot func(*os.Root, string, packagecache.DiskLimits, uint64) (*packageartifact.GenericStableCache, error)
	extract       func(mir.Program, map[hir.PackageID]packagegraph.InstanceID, map[hir.PackageID]packagegraph.CoordinateID) ([]packageartifact.Artifact, error)
	reassemble    func([]packageartifact.Artifact) (mir.Program, error)
	rebind        func([]packageartifact.StableArtifact, map[hir.PackageID]packagegraph.CoordinateID, hir.SymbolID, hir.SymbolID, mir.LocalID) (mir.Program, error)
	processSuffix string
	family        string
	version       string
}

func genericArtifactsFor(capabilities admission.Capabilities) (genericArtifactPolicy, error) {
	if err := capabilities.Validate(); err != nil {
		return genericArtifactPolicy{}, err
	}
	if capabilities.VectorCandidate() || capabilities.Storage == admission.StorageQueries ||
		capabilities.AllowsNominalArguments() != capabilities.AllowsStorageTransfers() {
		return genericArtifactPolicy{}, fmt.Errorf("workspace generic cache requires ordinary scalar or released nominal storage-transfer capabilities")
	}
	policy := genericArtifactPolicy{
		newMemory:     packageartifact.NewGenericCache,
		newDisk:       packageartifact.NewGenericStableCache,
		newDiskInRoot: packageartifact.NewGenericStableCacheInRoot,
		extract:       packageartifact.ExtractAll,
		reassemble:    packageartifact.Reassemble,
		rebind:        packageartifact.RebindStableArtifacts,
		processSuffix: ":generic-v0",
		family:        "fairyc.compiler-generic-stable-cache/0", version: packageartifact.StableArtifactVersion,
	}
	if capabilities.AllowsStorageTransfers() {
		policy = genericArtifactPolicy{
			newMemory:     packageartifact.NewStorageTransferCache,
			newDisk:       packageartifact.NewStorageTransferStableCache,
			newDiskInRoot: packageartifact.NewStorageTransferStableCacheInRoot,
			extract:       packageartifact.ExtractAllStorageTransfers,
			reassemble:    packageartifact.ReassembleStorageTransfers,
			rebind:        packageartifact.RebindStorageTransferStableArtifacts,
			processSuffix: ":generic-v0:storage-transfer-v0",
			family:        "fairyc.compiler-storage-stable-cache/0", version: packageartifact.StorageStableArtifactVersion,
		}
	}
	return policy, nil
}

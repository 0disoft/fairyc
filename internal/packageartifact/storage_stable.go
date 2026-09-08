package packageartifact

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const (
	StorageStableArtifactSchema  = "fairyc.stable-package-storage-mir/v0"
	StorageStableArtifactVersion = "fairyc-stable-package-storage-mir/0"
)

func stableArtifactContract(transfers bool) (string, string) {
	if transfers {
		return StorageStableArtifactSchema, StorageStableArtifactVersion
	}
	return StableArtifactSchema, StableArtifactVersion
}

// DescribeStorageTransferStableArtifact projects a verified private package into
// coordinate-bound context and bodies, without process-local package/symbol IDs.
func DescribeStorageTransferStableArtifact(artifact Artifact, maxBytes uint64) (StableArtifact, error) {
	return describeStableArtifact(artifact, maxBytes, true)
}

// EncodeStorageTransferStableArtifact emits the exact private envelope. This
// structural transport is not a source-compatibility or executable-publication
// decision; complete rebind and the transfer MIR verifier still own publication.
func EncodeStorageTransferStableArtifact(artifact StableArtifact, maxBytes uint64) ([]byte, error) {
	return encodeStableArtifact(artifact, maxBytes, true)
}

// DecodeStorageTransferStableArtifact checks canonical bytes, ownership, context,
// body identities and exact imports. It accepts no ordinary artifact envelope.
func DecodeStorageTransferStableArtifact(encoded []byte, maxBytes uint64) (StableArtifact, error) {
	return decodeStableArtifact(encoded, maxBytes, true)
}

// RebindStorageTransferStableArtifacts resolves the complete package closure into
// fresh namespaces and verifies transfer ownership before returning any MIR.
func RebindStorageTransferStableArtifacts(artifacts []StableArtifact, coordinates map[hir.PackageID]packagegraph.CoordinateID, firstFunctionSymbol, firstForeignSymbol hir.SymbolID, firstLocal mir.LocalID) (mir.Program, error) {
	result, err := rebindStableArtifactsWithBindings(artifacts, coordinates, firstFunctionSymbol, firstForeignSymbol, firstLocal, true)
	return result.MIR, err
}

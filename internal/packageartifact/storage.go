package packageartifact

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// ExtractAllStorageTransfers is the explicit private package-slicing lane for
// storage MIR. Source identities and coordinates must describe the input closure.
func ExtractAllStorageTransfers(program mir.Program, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID) ([]Artifact, error) {
	return extractAll(program, identities, coordinates, mir.VerifyStorageTransfers)
}

// VerifyStorageTransfers retains ordinary owner/import/context checks while
// allowing the private transfer operation set. It grants no stable cache use.
func VerifyStorageTransfers(artifact Artifact) error {
	return verifyArtifact(artifact, mir.VerifyStorageTransfers)
}

// ReassembleStorageTransfers requires a complete compatible package closure,
// rebinds calls and destructors to fresh symbols, and verifies the result. No
// import stub can escape and no caller-owned artifact is modified.
func ReassembleStorageTransfers(artifacts []Artifact) (mir.Program, error) {
	return reassemble(artifacts, mir.VerifyStorageTransfers)
}

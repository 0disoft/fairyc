package workspaceloader

// LoadVectorCandidate preserves ordinary manifest, path and source validation
// while explicitly selecting the compiler-private SDK vector bundle. It does
// not enable a manifest flag, stable compilation or private runtime linkage.
// Test overlays retain the selection, including when only tests import std.
func LoadVectorCandidate(root string) (Workspace, error) {
	return loadWorkspace(root, true)
}

// UsesVectorCandidate reports loader selection, not source authority. Consumers
// must verify VectorCandidateStorageOwners before admitting private operations.
func (workspace Workspace) UsesVectorCandidate() bool {
	return workspace.vectorCandidate
}

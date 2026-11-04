package attributedtext

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type exactSourceResolver func(string, string) (workspaceloader.ExactSource, error)

// EncodeWorkspaceSourceAndDocumentation encodes both selections for one subject,
// sharing only this call's detached, revalidated source. Nothing is cached across
// requests, and the workspace must not be concurrently mutated during the call.
func EncodeWorkspaceSourceAndDocumentation(workspace workspaceloader.Workspace, index *atlasmap.QueryIndex, snapshot semanticsnapshot.Snapshot, symbolID string, budget Budget, maxBytes uint64) ([]byte, []byte, error) {
	return encodeWorkspacePair(workspace.ResolveExactSource, index, snapshot, symbolID, budget, maxBytes)
}

func encodeWorkspacePair(resolve exactSourceResolver, index *atlasmap.QueryIndex, snapshot semanticsnapshot.Snapshot, symbolID string, budget Budget, maxBytes uint64) ([]byte, []byte, error) {
	if maxBytes == 0 {
		return nil, nil, fmt.Errorf("Atlas attributed-text document byte limit must be positive")
	}
	var cached workspaceloader.ExactSource
	var instance, path string
	resolved := false
	resolveOnce := func(packageInstance, sourcePath string) (workspaceloader.ExactSource, error) {
		if resolved && instance == packageInstance && path == sourcePath {
			return cached, nil
		}
		exact, err := resolve(packageInstance, sourcePath)
		if err == nil {
			cached, instance, path, resolved = exact, packageInstance, sourcePath, true
		}
		return exact, err
	}
	request := Request{SymbolID: symbolID, Kinds: []Kind{KindSourceExcerpt}, Origins: []Origin{OriginWorkspaceAuthored}, Budget: budget}
	source, err := extractWorkspaceSource(resolveOnce, index, request)
	if err != nil {
		return nil, nil, err
	}
	sourceDocument, err := encodeResult(source, maxBytes)
	if err != nil {
		return nil, nil, err
	}
	request.Kinds = []Kind{KindDeclarationDocumentation}
	documentation, err := extractWorkspaceDocumentation(resolveOnce, index, snapshot, request)
	if err != nil {
		return nil, nil, err
	}
	documentationDocument, err := encodeResult(documentation, maxBytes)
	if err != nil {
		return nil, nil, err
	}
	return sourceDocument, documentationDocument, nil
}

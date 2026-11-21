package cli

import (
	"context"
	"io"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// atlasImpactSession belongs to one sequential MCP session, with a fixed root
// and running compiler image. It retains no process-global or persisted cache.
type atlasImpactSession struct {
	root     string
	prepared *preparedAtlasImpact
	files    *workspaceloader.FileSnapshot
}

func (session *atlasImpactSession) document(o impactOptions, stderr io.Writer) ([]byte, int) {
	current := false
	if session.prepared != nil && session.files != nil {
		matches, err := session.files.Matches(context.Background(), nil)
		current = err == nil && matches
	}
	if !current {
		// Release the old graph even when replacement inputs cannot be loaded.
		session.prepared, session.files = nil, nil
		workspace, code := prepareAtlasWorkspaceSchema(session.root, true, stderr, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			return nil, code
		}
		session.prepared = &preparedAtlasImpact{workspace: workspace}
		session.files = workspaceloader.NewLockedFileSnapshot(workspace.workspace, workspace.verifiedLock)
	}
	return session.prepared.document(o, stderr)
}

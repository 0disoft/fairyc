package cli

import (
	"context"
	"io"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/lockfile"
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
		if err == nil && matches {
			// Capture may race a lock edit after preparation. Validate its meaning
			// as well as its exact bytes before reusing a locked result.
			lock, _, err := lockfile.Read(filepath.Join(session.root, lockfile.FileName))
			current = err == nil && lockfile.MatchesWorkspace(lock, session.prepared.workspace.workspace) == nil
		}
	}
	if !current {
		// Release the old graph even when replacement inputs cannot be loaded.
		session.prepared, session.files = nil, nil
		workspace, code := prepareAtlasWorkspaceSchema(session.root, true, stderr, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			return nil, code
		}
		session.prepared = &preparedAtlasImpact{workspace: workspace}
		// Snapshot failures disable reuse, without changing this fresh query's
		// established preparation or diagnostic behavior.
		session.files, _ = workspaceloader.CaptureFileSnapshot(context.Background(), workspace.workspace, nil)
	}
	return session.prepared.document(o, stderr)
}

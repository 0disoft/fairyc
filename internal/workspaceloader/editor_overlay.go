package workspaceloader

import (
	"context"
	"fmt"
	"github.com/0disoft/fairyc/internal/source"
	"path/filepath"
	"slices"
	"strings"
)

// LoadEditorOverlays applies immutable unsaved .fy contents to the ordinary local
// loader. Manifests, SDK ownership, graph resolution and disk contents remain unchanged.
func LoadEditorOverlays(root string, overlays map[string][]byte) (Workspace, error) {
	return LoadEditorOverlaysContext(context.Background(), root, overlays)
}

func LoadEditorOverlaysContext(ctx context.Context, root string, overlays map[string][]byte) (Workspace, error) {
	if err := ctx.Err(); err != nil {
		return Workspace{}, err
	}
	canonical, err := resolveRoot(root)
	if err != nil {
		return Workspace{}, err
	}
	if len(overlays) > MaxSourceFiles {
		return Workspace{}, fmt.Errorf("too many editor overlays")
	}
	copied := make(map[string][]byte, len(overlays))
	total := 0
	for filename, data := range overlays {
		absolute, err := filepath.Abs(filename)
		if err != nil {
			return Workspace{}, err
		}
		relative, err := filepath.Rel(canonical, absolute)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) || filepath.Ext(absolute) != ".fy" {
			return Workspace{}, fmt.Errorf("editor overlay escapes workspace or is not Fairy source")
		}
		if len(data) > source.DefaultLimits().MaxBytes || total > MaxSourceBytes-len(data) {
			return Workspace{}, fmt.Errorf("editor overlay byte limit")
		}
		total += len(data)
		copied[absolute] = slices.Clone(data)
	}
	return loadWorkspaceOverlaysContext(ctx, canonical, false, copied)
}

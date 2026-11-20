package sourceedit

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const MaxBatchEdits = 16

type BatchReport struct {
	Schema   string   `json:"schema"`
	Edits    []Report `json:"edits"`
	Verified bool     `json:"verified"`
}

// PreviewBodies checks the combined overlay, preserving the caller's edit
// order in the report. All hashes refer to whole files.
func PreviewBodies(w workspaceloader.Workspace, edits []BodyEdit) (BatchReport, error) {
	if len(edits) == 0 || len(edits) > MaxBatchEdits {
		return BatchReport{}, fmt.Errorf("body batch requires 1 to %d edits", MaxBatchEdits)
	}
	ctx := context.Background()
	baseline, err := workspaceloader.CaptureFileSnapshot(ctx, w, nil)
	if err != nil {
		return BatchReport{}, err
	}
	type indexedBody struct {
		preparedBody
		index int
	}
	groups := make(map[string][]indexedBody)
	paths := make([]string, 0, len(edits))
	for i, edit := range edits {
		prepared, err := prepareBody(w, edit)
		if err != nil {
			return BatchReport{}, fmt.Errorf("edit %d: %w", i+1, err)
		}
		if len(groups[prepared.absolute]) == 0 {
			paths = append(paths, prepared.absolute)
		}
		groups[prepared.absolute] = append(groups[prepared.absolute], indexedBody{prepared, i})
	}
	overlays := make(map[string][]byte, len(groups))
	reports := make([]Report, len(edits))
	for _, path := range paths {
		group := groups[path]
		slices.SortFunc(group, func(a, b indexedBody) int { return cmp.Compare(a.start, b.start) })
		bodies := make([]preparedBody, len(group))
		for i, edit := range group {
			bodies[i] = edit.preparedBody
		}
		after, err := composeBodies(bodies)
		if err != nil {
			return BatchReport{}, fmt.Errorf("%s: %w", bodies[0].relative, err)
		}
		overlays[path] = after
		for _, edit := range group {
			reports[edit.index] = edit.report(after)
		}
	}
	if err := validateOverlays(ctx, w.Root, overlays); err != nil {
		return BatchReport{}, err
	}
	current, err := baseline.Matches(ctx, nil)
	if err != nil {
		return BatchReport{}, err
	}
	if !current {
		return BatchReport{}, fmt.Errorf("workspace changed during edit batch preview")
	}
	return BatchReport{Schema: "fairyc.structural-body-edit-batch/v1", Edits: reports, Verified: true}, nil
}

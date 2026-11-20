package sourceedit

import (
	"bytes"
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

type batchFile struct {
	path          string
	before, after []byte
}

type BatchPlan struct {
	Report   BatchReport
	root     string
	baseline *workspaceloader.FileSnapshot
	files    []batchFile
}

// PlanBodies checks one combined overlay. The plan owns the verified bytes
// for publication, while the report keeps the caller's edit order.
func PlanBodies(w workspaceloader.Workspace, edits []BodyEdit) (BatchPlan, error) {
	if len(edits) == 0 || len(edits) > MaxBatchEdits {
		return BatchPlan{}, fmt.Errorf("body batch requires 1 to %d edits", MaxBatchEdits)
	}
	ctx := context.Background()
	baseline, err := workspaceloader.CaptureFileSnapshot(ctx, w, nil)
	if err != nil {
		return BatchPlan{}, err
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
			return BatchPlan{}, fmt.Errorf("edit %d: %w", i+1, err)
		}
		if len(groups[prepared.absolute]) == 0 {
			paths = append(paths, prepared.absolute)
		}
		groups[prepared.absolute] = append(groups[prepared.absolute], indexedBody{prepared, i})
	}
	overlays := make(map[string][]byte, len(groups))
	reports := make([]Report, len(edits))
	files := make([]batchFile, 0, len(paths))
	for _, path := range paths {
		group := groups[path]
		slices.SortFunc(group, func(a, b indexedBody) int { return cmp.Compare(a.start, b.start) })
		bodies := make([]preparedBody, len(group))
		for i, edit := range group {
			bodies[i] = edit.preparedBody
		}
		after, err := composeBodies(bodies)
		if err != nil {
			return BatchPlan{}, fmt.Errorf("%s: %w", bodies[0].relative, err)
		}
		overlays[path] = after
		files = append(files, batchFile{bodies[0].relative, bytes.Clone(bodies[0].file.Bytes), after})
		for _, edit := range group {
			reports[edit.index] = edit.report(after)
		}
	}
	if err := validateOverlays(ctx, w.Root, overlays); err != nil {
		return BatchPlan{}, err
	}
	current, err := baseline.Matches(ctx, nil)
	if err != nil {
		return BatchPlan{}, err
	}
	if !current {
		return BatchPlan{}, fmt.Errorf("workspace changed during edit batch preview")
	}
	slices.SortFunc(files, func(a, b batchFile) int { return cmp.Compare(a.path, b.path) })
	return BatchPlan{Report: BatchReport{Schema: "fairyc.structural-body-edit-batch/v1", Edits: reports, Verified: true},
		root: w.Root, baseline: baseline, files: files}, nil
}

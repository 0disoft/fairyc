package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

type atlasEditBatchRequest struct {
	Edits []struct {
		Symbol         string `json:"symbol"`
		BodyFile       string `json:"bodyFile"`
		ExpectedSHA256 string `json:"expectedSourceSha256"`
	} `json:"edits"`
}

func runAtlasEditBatch(action string, args []string, root string, stdout, stderr io.Writer) int {
	var uid, filename string
	_, err := parseCommandFlags(args, flagSchema{
		Command: "atlas-edit " + action, Experimental: true,
		Required: []string{"--package-uid", "--edit-file"},
		Flags:    []flagDefinition{{Name: "--package-uid", Kind: flagScalar}, {Name: "--edit-file", Kind: flagScalar}},
	}, func(flag, value string) error {
		if flag == "--package-uid" {
			uid = value
		} else {
			filename = value
		}
		return nil
	})
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	prepared, code := prepareAtlasWorkspaceSchema(root, true, stderr, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		return code
	}
	partition, code := atlasEditPartition(prepared, uid, stderr)
	if code != ExitSuccess {
		return code
	}
	opened, err := os.OpenRoot(prepared.workspace.Root)
	if err != nil {
		return packageFailure(stderr, err)
	}
	defer opened.Close()
	manifest, err := sourceedit.ReadBodyFile(prepared.workspace, opened, filename)
	if err != nil {
		return packageFailure(stderr, err)
	}
	request, err := decodeAtlasEditBatch(manifest)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	edits := make([]sourceedit.BodyEdit, 0, len(request.Edits))
	for i, edit := range request.Edits {
		key, err := atlasEditFunction(partition, edit.Symbol)
		if err != nil {
			return invocationError(stderr, fmt.Sprintf("edit %d: %v", i+1, err))
		}
		body, err := sourceedit.ReadBodyFile(prepared.workspace, opened, edit.BodyFile)
		if err != nil {
			return packageFailure(stderr, fmt.Errorf("edit %d: %w", i+1, err))
		}
		edits = append(edits, sourceedit.BodyEdit{Module: key.Module, Function: key.Name, Expected: edit.ExpectedSHA256, Body: body})
	}
	plan, err := sourceedit.PlanBodies(prepared.workspace, edits)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	data, err := json.Marshal(plan.Report)
	if err != nil || len(data) > 32<<10 {
		return invocationError(stderr, "edit batch report exceeds output byte limit")
	}
	if action == "apply-batch" {
		if err := plan.ApplyWithOptions(); err != nil {
			return packageFailure(stderr, err)
		}
		// Applied states encode as true, which is shorter than false.
		data, err = json.Marshal(plan.Report)
		if err != nil {
			return internalError(stderr)
		}
	}
	if written, err := stdout.Write(data); err != nil || written != len(data) {
		return outputError(stderr)
	}
	return ExitSuccess
}

func decodeAtlasEditBatch(manifest []byte) (atlasEditBatchRequest, error) {
	var request atlasEditBatchRequest
	decoder := json.NewDecoder(bytes.NewReader(manifest))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return atlasEditBatchRequest{}, fmt.Errorf("invalid edit batch: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return atlasEditBatchRequest{}, fmt.Errorf("edit batch requires a single JSON object")
	}
	if len(request.Edits) == 0 || len(request.Edits) > sourceedit.MaxBatchEdits {
		return atlasEditBatchRequest{}, fmt.Errorf("body batch requires 1 to %d edits", sourceedit.MaxBatchEdits)
	}
	for i, edit := range request.Edits {
		if edit.Symbol == "" || edit.BodyFile == "" || len(edit.ExpectedSHA256) != 64 {
			return atlasEditBatchRequest{}, fmt.Errorf("edit %d requires symbol, bodyFile and expectedSourceSha256", i+1)
		}
	}
	return request, nil
}

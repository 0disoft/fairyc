// Package sourceedit prepares narrow AST-owned transforms and validates the complete workspace.
package sourceedit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const MaxBody = 64 << 10

type Report struct {
	Schema        string `json:"schema"`
	Path          string `json:"path"`
	Module        string `json:"module"`
	Function      string `json:"function"`
	BodyStartByte int    `json:"bodyStartByte"`
	BodyEndByte   int    `json:"bodyEndByte"`
	BeforeSHA256  string `json:"beforeSha256"`
	AfterSHA256   string `json:"afterSha256"`
	Verified      bool   `json:"verified"`
	Applied       bool   `json:"applied"`
}
type Plan struct {
	Report                  Report
	root, path, fingerprint string
	before, after           []byte
	required                genericCoverage
}

func Digest(data []byte) string { s := sha256.Sum256(data); return hex.EncodeToString(s[:]) }

func PlanBody(w workspaceloader.Workspace, module, name, expected string, body []byte) (Plan, error) {
	edit, err := prepareBody(w, BodyEdit{Module: module, Function: name, Expected: expected, Body: body})
	if err != nil {
		return Plan{}, err
	}
	after, err := composeBodies([]preparedBody{edit})
	if err != nil {
		return Plan{}, err
	}
	ctx := context.Background()
	required, err := requireGenericCoverage(ctx, w, []preparedBody{edit})
	if err != nil {
		return Plan{}, err
	}
	if err := validateOverlays(ctx, w.Root, map[string][]byte{edit.absolute: after}, required); err != nil {
		return Plan{}, err
	}
	fingerprint, err := snapshot(w)
	if err != nil {
		return Plan{}, err
	}
	report := edit.report(after)
	return Plan{Report: report, root: w.Root, path: edit.relative, fingerprint: fingerprint, before: bytes.Clone(edit.file.Bytes), after: after, required: required}, nil
}

type BodyEdit struct {
	Module, Function, Expected string
	Body                       []byte
}

type preparedBody struct {
	file               workspaceloader.SourceFile
	absolute, relative string
	module, function   string
	start, end, name   int
	body               []byte
	generic            bool
}

func prepareBody(w workspaceloader.Workspace, edit BodyEdit) (preparedBody, error) {
	module, name, expected, body := edit.Module, edit.Function, edit.Expected, edit.Body
	if len(body) > MaxBody {
		return preparedBody{}, fmt.Errorf("body byte limit exceeded")
	}
	body = bytes.TrimSpace(body)
	if len(body) < 2 || body[0] != '{' || body[len(body)-1] != '}' {
		return preparedBody{}, fmt.Errorf("replacement must be exactly one braced body")
	}
	pkg, ok := w.Package(".")
	if !ok {
		return preparedBody{}, fmt.Errorf("root package unavailable")
	}
	var selected *ast.Function
	var file workspaceloader.SourceFile
	for _, s := range pkg.Sources {
		if s.Module != nil && s.Module.Path == module {
			for i := range s.Module.Functions {
				f := &s.Module.Functions[i]
				if f.Owner == "" && f.Name == name {
					if selected != nil {
						return preparedBody{}, fmt.Errorf("ambiguous source function")
					}
					selected = f
					file = s
				}
			}
		}
	}
	if selected == nil || selected.Public || selected.Unsafe || selected.Owner != "" || selected.Name == "main" || len(selected.Attributes) != 0 || selected.Body == nil {
		return preparedBody{}, fmt.Errorf("only private safe non-entry functions without owners or attributes are editable")
	}
	if len(expected) != 64 || Digest(file.Bytes) != expected {
		return preparedBody{}, fmt.Errorf("expected source SHA-256 is stale or invalid")
	}
	span := selected.Body.Span
	if !span.Valid(len(file.Bytes)) {
		return preparedBody{}, fmt.Errorf("invalid AST body span")
	}
	absolute := filepath.Join(pkg.SourceRoot, filepath.FromSlash(file.Path))
	relative, err := filepath.Rel(w.Root, absolute)
	if err != nil {
		return preparedBody{}, err
	}
	relative = filepath.ToSlash(relative)
	if strings.HasPrefix(strings.ToLower(relative), "vendor/") {
		return preparedBody{}, fmt.Errorf("vendor source is not editable")
	}
	return preparedBody{file: file, absolute: absolute, relative: relative, module: module, function: name,
		start: span.StartByte, end: span.EndByte, name: selected.NameSpan.StartByte, body: body, generic: len(selected.TypeParameters) != 0}, nil
}

func composeBodies(edits []preparedBody) ([]byte, error) {
	file := edits[0].file
	type extent struct{ name, start, end int }
	expected := make(map[extent]string, len(edits))
	after := make([]byte, 0, len(file.Bytes))
	previousEnd := 0
	for _, edit := range edits {
		if edit.start < previousEnd {
			return nil, fmt.Errorf("duplicate or overlapping edit bodies")
		}
		shift := len(after) - previousEnd
		start := edit.start + shift
		expected[extent{edit.name + shift, start, start + len(edit.body)}] = edit.function
		after = append(after, file.Bytes[previousEnd:edit.start]...)
		after = append(after, edit.body...)
		previousEnd = edit.end
	}
	after = append(after, file.Bytes[previousEnd:]...)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile(file.Path, after), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return nil, fmt.Errorf("replacement does not parse")
	}
	for _, f := range parsed.AST.Functions {
		if f.Body == nil {
			continue
		}
		key := extent{f.NameSpan.StartByte, f.Body.Span.StartByte, f.Body.Span.EndByte}
		if expected[key] == f.Name {
			delete(expected, key)
		}
	}
	if len(expected) != 0 {
		return nil, fmt.Errorf("replacement injected or escaped the selected body")
	}
	return after, nil
}

func (edit preparedBody) report(after []byte) Report {
	return Report{Schema: "fairyc.structural-body-edit/v1", Path: edit.relative, Module: edit.module,
		Function: edit.function, BodyStartByte: edit.start, BodyEndByte: edit.end,
		BeforeSHA256: Digest(edit.file.Bytes), AfterSHA256: Digest(after), Verified: true}
}

func validateOverlays(ctx context.Context, root string, overlays map[string][]byte, required genericCoverage) error {
	w, err := workspaceloader.LoadEditorOverlaysContext(ctx, root, overlays)
	if err != nil {
		return fmt.Errorf("edited workspace load failed: %w", errors.Join(err, ctx.Err(), context.Cause(ctx)))
	}
	// Derive private storage admission from the overlay's authenticated SDK,
	// exactly as current-source Atlas preparation does. Client code gains none.
	r, err := workspacecompiler.CheckGenericContext(ctx, w)
	if err != nil || !r.Verified || len(r.Semantic.Diagnostics) != 0 || len(r.Semantic.Unsupported) != 0 {
		return errors.Join(fmt.Errorf("edited workspace failed semantic/MIR validation"), err, ctx.Err(), context.Cause(ctx))
	}
	return required.check(r.GenericProvenance)
}

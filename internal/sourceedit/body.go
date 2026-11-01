// Package sourceedit prepares narrow AST-owned transforms and validates the complete workspace.
package sourceedit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
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
}

func Digest(data []byte) string { s := sha256.Sum256(data); return hex.EncodeToString(s[:]) }

func PlanBody(w workspaceloader.Workspace, module, name, expected string, body []byte) (Plan, error) {
	if len(body) > MaxBody {
		return Plan{}, fmt.Errorf("body byte limit exceeded")
	}
	body = bytes.TrimSpace(body)
	if len(body) < 2 || body[0] != '{' || body[len(body)-1] != '}' {
		return Plan{}, fmt.Errorf("replacement must be exactly one braced body")
	}
	pkg, ok := w.Package(".")
	if !ok {
		return Plan{}, fmt.Errorf("root package unavailable")
	}
	var selected *ast.Function
	var file workspaceloader.SourceFile
	for _, s := range pkg.Sources {
		if s.Module != nil && s.Module.Path == module {
			for i := range s.Module.Functions {
				f := &s.Module.Functions[i]
				if f.Name == name {
					if selected != nil {
						return Plan{}, fmt.Errorf("ambiguous source function")
					}
					selected = f
					file = s
				}
			}
		}
	}
	if selected == nil || selected.Public || selected.Unsafe || selected.Owner != "" || selected.Name == "main" || len(selected.TypeParameters) != 0 || len(selected.Attributes) != 0 || selected.Body == nil {
		return Plan{}, fmt.Errorf("only private ordinary safe non-entry functions without attributes are editable")
	}
	if len(expected) != 64 || Digest(file.Bytes) != expected {
		return Plan{}, fmt.Errorf("expected source SHA-256 is stale or invalid")
	}
	span := selected.Body.Span
	if !span.Valid(len(file.Bytes)) {
		return Plan{}, fmt.Errorf("invalid AST body span")
	}
	after := make([]byte, 0, len(file.Bytes)+len(body))
	after = append(after, file.Bytes[:span.StartByte]...)
	after = append(after, body...)
	after = append(after, file.Bytes[span.EndByte:]...)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile(file.Path, after), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return Plan{}, fmt.Errorf("replacement does not parse")
	}
	matched := false
	for _, f := range parsed.AST.Functions {
		if f.NameSpan.StartByte == selected.NameSpan.StartByte && f.Name == name && f.Body != nil {
			matched = f.Body.Span.StartByte == span.StartByte && f.Body.Span.EndByte == span.StartByte+len(body)
		}
	}
	if !matched {
		return Plan{}, fmt.Errorf("replacement injected or escaped the selected body")
	}
	absolute := filepath.Join(pkg.SourceRoot, filepath.FromSlash(file.Path))
	relative, err := filepath.Rel(w.Root, absolute)
	if err != nil {
		return Plan{}, err
	}
	relative = filepath.ToSlash(relative)
	if strings.HasPrefix(strings.ToLower(relative), "vendor/") {
		return Plan{}, fmt.Errorf("vendor source is not editable")
	}
	if err := validateOverlay(w.Root, absolute, after); err != nil {
		return Plan{}, err
	}
	fingerprint, err := snapshot(w)
	if err != nil {
		return Plan{}, err
	}
	return Plan{Report: Report{Schema: "fairyc.structural-body-edit/v1", Path: relative, Module: module, Function: name, BodyStartByte: span.StartByte, BodyEndByte: span.EndByte, BeforeSHA256: Digest(file.Bytes), AfterSHA256: Digest(after), Verified: true}, root: w.Root, path: relative, fingerprint: fingerprint, before: bytes.Clone(file.Bytes), after: after}, nil
}

func validateOverlay(root, absolute string, after []byte) error {
	w, err := workspaceloader.LoadEditorOverlays(root, map[string][]byte{absolute: after})
	if err != nil {
		return fmt.Errorf("edited workspace load failed: %w", err)
	}
	input, err := w.SemanticInput()
	if err != nil {
		return err
	}
	coordinates, err := w.PackageCoordinates()
	if err != nil {
		return err
	}
	r, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !r.Verified || len(r.Semantic.Diagnostics) != 0 || len(r.Semantic.Unsupported) != 0 {
		return fmt.Errorf("edited workspace failed semantic/MIR validation")
	}
	return nil
}

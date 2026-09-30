package languageserver

import (
	"errors"
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"os"
	"path/filepath"
	"strings"
)

type sourceKey struct {
	Package hir.PackageID
	Path    string
}
type sourceView struct {
	Path   string
	Text   []byte
	Module *ast.Module
}
type lspDiagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     string `json:"code"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}
type analysis struct {
	result      semantic.Result
	files       map[sourceKey]sourceView
	imports     map[sourceKey]hir.PackageID
	diagnostics map[string][]lspDiagnostic
	clean       bool
}

func (s *server) projectRoot(filename string) string {
	// Parent lookup stays within the one explicitly initialized workspace folder.
	if s.root == "" {
		return ""
	}
	for directory := filepath.Dir(filename); contained(s.root, directory); directory = filepath.Dir(directory) {
		info, err := os.Lstat(filepath.Join(directory, "fairy.toml"))
		if err == nil && info.Mode().IsRegular() {
			return directory
		}
		if directory == s.root || directory == filepath.Dir(directory) {
			break
		}
	}
	return ""
}
func (s *server) analyze(d *document, root string) *analysis {
	a := &analysis{files: map[sourceKey]sourceView{}, imports: map[sourceKey]hir.PackageID{}, diagnostics: map[string][]lspDiagnostic{}}
	if root != "" {
		overlays := map[string][]byte{}
		for _, opened := range s.docs {
			if contained(root, opened.Path) {
				overlays[opened.Path] = opened.Text
			}
		}
		workspace, err := workspaceloader.LoadEditorOverlays(root, overlays)
		if err != nil {
			var load *workspaceloader.LoadError
			if errors.As(err, &load) && len(load.Diagnostics) > 0 {
				filename := load.AbsoluteSourcePath
				if filename == "" {
					filename = filepath.Join(root, filepath.FromSlash(load.Path))
				}
				for _, diag := range load.Diagnostics {
					a.add(filename, load.Source, diag.Span.StartByte, diag.Span.EndByte, diag.Code, diag.Code)
				}
			}
			// Every open file receives an explicit unavailable diagnostic; no stale clean results.
			for filename, text := range overlays {
				if len(a.diagnostics[filename]) == 0 {
					a.add(filename, text, 0, 0, "workspace-analysis-unavailable", internalDiagnostic(err))
				}
			}
			return a
		}
		input, err := workspace.SemanticInput()
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
			return a
		}
		for _, imported := range input.Imports {
			a.imports[sourceKey{imported.FromPackage, imported.ImportPath}] = imported.ToPackage
		}
		packages, err := workspace.SemanticPackages()
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
			return a
		}
		for _, p := range packages {
			loaded, ok := workspace.Package(p.Path)
			if !ok {
				continue
			}
			for _, file := range loaded.Sources {
				physical := filepath.Join(loaded.SourceRoot, filepath.FromSlash(file.Path))
				if loaded.SourceKind == workspaceloader.SourceKindSDK {
					physical = ""
				}
				a.files[sourceKey{p.ID, file.Path}] = sourceView{physical, file.Bytes, file.Module}
			}
		}
		result, err := workspacecompiler.Check(workspace)
		a.result = result.Semantic
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
		}
		for filename, text := range overlays {
			included := false
			for _, file := range a.files {
				if file.Path == filename {
					included = true
					break
				}
			}
			if !included {
				a.add(filename, text, 0, 0, "editor-source-not-in-project", "Source is outside the project's resolved source roots or reachable package closure")
			}
		}
	} else {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(filepath.Base(d.Path), d.Text), source.DefaultLimits())
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
			return a
		}
		for _, diag := range parsed.Tree.Diagnostics {
			a.add(d.Path, d.Text, diag.Span.StartByte, diag.Span.EndByte, diag.Code, diag.Code)
		}
		a.files[sourceKey{0, filepath.Base(d.Path)}] = sourceView{d.Path, d.Text, parsed.AST}
		if parsed.AST == nil {
			return a
		}
		result, err := compiler.Check([]semantic.File{{Path: filepath.Base(d.Path), Module: parsed.AST}})
		a.result = result.Semantic
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
		}
	}
	for _, diag := range a.result.Diagnostics {
		f, ok := a.file(diag.Primary)
		if !ok || f.Path == "" {
			continue
		}
		message := diag.Code
		if diag.ExpectedType != "" || diag.ActualType != "" {
			message += ": expected " + string(diag.ExpectedType) + ", got " + string(diag.ActualType)
		}
		a.add(f.Path, f.Text, diag.Primary.StartByte, diag.Primary.EndByte, diag.Code, message)
	}
	for _, u := range a.result.Unsupported {
		if f, ok := a.file(u.Span); ok && f.Path != "" {
			a.add(f.Path, f.Text, u.Span.StartByte, u.Span.EndByte, "unsupported-source", u.Feature)
		}
	}
	a.clean = true
	for _, diags := range a.diagnostics {
		if len(diags) > 0 {
			a.clean = false
			break
		}
	}
	return a
}
func (a *analysis) file(span hir.Span) (sourceView, bool) {
	key := sourceKey{span.Package, span.Source}
	if key.Package == 0 {
		if pkg, relative := hir.SourcePathParts(span.Source); pkg != 0 {
			key = sourceKey{pkg, relative}
		}
	}
	f, ok := a.files[key]
	return f, ok
}
func (a *analysis) add(filename string, text []byte, start, end int, code, message string) {
	if strings.TrimSpace(message) == "" {
		message = code
	}
	a.diagnostics[filename] = append(a.diagnostics[filename], lspDiagnostic{textRange(text, start, end), 1, code, "fairyc", message})
}

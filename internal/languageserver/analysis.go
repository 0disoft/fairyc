package languageserver

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/durablefile"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const sourceNotInProjectDiagnostic = "editor-source-not-in-project"

type sourceKey struct {
	Package hir.PackageID
	Path    string
}
type sourceView struct {
	Path   string
	Text   []byte
	Module *ast.Module
	SDKURI string
}
type lspDiagnostic struct {
	Range              Range                   `json:"range"`
	Severity           int                     `json:"severity"`
	Code               string                  `json:"code"`
	Source             string                  `json:"source"`
	Message            string                  `json:"message"`
	RelatedInformation []diagnosticRelatedInfo `json:"relatedInformation,omitempty"`
}

type diagnosticRelatedInfo struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}
type analysis struct {
	result          semantic.Result
	files           map[sourceKey]sourceView
	imports         map[sourceKey]hir.PackageID
	diagnostics     map[string][]lspDiagnostic
	positions       map[string]*positionIndex
	definitions     map[string][]definitionSpan
	completion      completionSourceIndex
	clean           bool
	root            string
	snapshot        *workspaceloader.FileSnapshot
	syntaxSnapshot  *workspaceloader.EditorSyntaxSnapshot
	standalone      [sha256.Size]byte
	standaloneValid bool
	lastUsed        uint64
	budgetExceeded  bool
	unsynchronized  bool
	repair          *syntaxRepair
	sourceCharge    int64
	sourceCharged   bool
}

func projectRoot(boundary, startDirectory string) string {
	if boundary == "" || !projectDirectoryPath(boundary, boundary) {
		return ""
	}
	packageDirectory := ""
	for directory := startDirectory; contained(boundary, directory); directory = filepath.Dir(directory) {
		info, err := os.Lstat(filepath.Join(directory, "fairy.toml"))
		if err == nil && info.Mode().IsRegular() {
			if !projectDirectoryPath(boundary, directory) {
				return ""
			}
			packageDirectory = directory
			break
		}
		if sameDocumentPath(directory, boundary) || directory == filepath.Dir(directory) {
			break
		}
	}
	if packageDirectory == "" {
		return ""
	}
	// A member manifest cannot carry the outer catalog. Prefer the nearest explicit
	// workspace whose members name this package directory exactly.
	for directory := filepath.Dir(packageDirectory); contained(boundary, directory); directory = filepath.Dir(directory) {
		if workspaceContainsPackage(directory, packageDirectory) {
			return directory
		}
		if sameDocumentPath(directory, boundary) || directory == filepath.Dir(directory) {
			break
		}
	}
	return packageDirectory
}

func projectDirectoryPath(root, directory string) bool {
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false
	}
	relative, err := filepath.Rel(root, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return false
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err = os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false
		}
	}
	return true
}

func workspaceContainsPackage(directory, packageDirectory string) bool {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return false
	}
	defer root.Close()
	file, err := durablefile.OpenRegular(root, projectmanifest.FileName)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() > projectmanifest.MaxBytes {
		return false
	}
	contents, err := io.ReadAll(io.LimitReader(file, projectmanifest.MaxBytes+1))
	if err != nil || len(contents) > projectmanifest.MaxBytes {
		return false
	}
	manifest, err := projectmanifest.Parse(contents)
	if err != nil || !manifest.Workspace.Present {
		return false
	}
	packageDirectory = filepath.Clean(packageDirectory)
	for _, member := range manifest.Workspace.Members {
		if filepath.Clean(filepath.Join(directory, filepath.FromSlash(member))) == packageDirectory {
			return true
		}
	}
	return false
}
func (s *server) analyze(d *document, root string) *analysis {
	result, _ := s.analyzeContext(context.Background(), d, root)
	return result
}

type analysisOptions struct {
	sourceLimit           int64
	reserveSource         func(int64) (bool, error)
	retainedSyntaxWeight  func() int64
	sdkSourceNavigation   bool
	diagnosticRelatedInfo bool
}

func (options analysisOptions) admitSource(required int64) (bool, error) {
	if options.reserveSource != nil {
		return options.reserveSource(required)
	}
	return required >= 0 && required <= options.sourceLimit, nil
}

func (options analysisOptions) syntaxRoom() int64 {
	if options.retainedSyntaxWeight == nil {
		return maxSessionSyntaxWeight
	}
	return maxSessionSyntaxWeight - options.retainedSyntaxWeight()
}

func (s *server) analyzeContext(ctx context.Context, d *document, root string) (*analysis, error) {
	return analyzeDocument(ctx, d, root, s.docs, analysisOptions{
		sourceLimit:           s.analysisSourceLimit(),
		reserveSource:         s.reserveAnalysisSource,
		retainedSyntaxWeight:  s.retainedSyntaxWeight,
		sdkSourceNavigation:   s.sdkSourceNavigation,
		diagnosticRelatedInfo: s.diagnosticRelatedInfo,
	})
}

func analyzeDocument(ctx context.Context, d *document, root string, documents map[string]*document, options analysisOptions) (result *analysis, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result, err = nil, canceled
		}
	}()
	if previous := d.View; !d.needsFullText && root == "" && previous != nil && previous.root == "" && previous.standaloneValid && previous.standalone == sha256.Sum256(d.Text) {
		return previous, nil
	}
	a := &analysis{root: root, files: map[sourceKey]sourceView{}, imports: map[sourceKey]hir.PackageID{}, diagnostics: map[string][]lspDiagnostic{}}
	if d.needsFullText && root == "" {
		a.unsynchronized = true
		a.add(d.Path, d.Text, 0, 0, "editor-document-unsynchronized", "Document needs full synchronization")
		return a, nil
	}
	if root != "" {
		overlays := map[string][]byte{}
		for _, opened := range documents {
			if contained(root, opened.Path) {
				overlays[opened.Path] = opened.Text
				a.unsynchronized = a.unsynchronized || opened.needsFullText
			}
		}
		if a.unsynchronized {
			for filename, text := range overlays {
				a.add(filename, text, 0, 0, "editor-document-unsynchronized", "An open document in this project needs full synchronization")
			}
			return a, nil
		}
		if previous := d.View; previous != nil && previous.root == root && previous.snapshot != nil {
			current, err := previous.snapshot.Matches(ctx, overlays)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if err == nil && current {
				return previous, nil
			}
		}
		var previousSyntax *workspaceloader.EditorSyntaxSnapshot
		if d.View != nil && d.View.root == root {
			previousSyntax = d.View.syntaxSnapshot
		}
		if previousSyntax == nil && d.LastClean != nil && d.LastClean.root == root {
			previousSyntax = d.LastClean.syntaxSnapshot
		}
		workspace, parsedSyntax, err := workspaceloader.LoadEditorSyntaxContext(ctx, root, overlays, previousSyntax)
		if err != nil {
			var load *workspaceloader.LoadError
			if errors.As(err, &load) && len(load.Diagnostics) > 0 {
				filename := load.AbsoluteSourcePath
				if filename == "" {
					filename = filepath.Join(root, filepath.FromSlash(load.Path))
				}
				a.repair = singleSyntaxRepair(filename, load.RelativeSourcePath, load.Diagnostics)
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
			a.sourceCharge, a.sourceCharged = analysisSourceBytes(a), true
			if retained, err := options.admitSource(a.sourceCharge); err != nil {
				return nil, err
			} else if !retained {
				return analysisBudgetUnavailable(d, root, overlays, options.sourceLimit), nil
			}
			return a, nil
		}
		a.sourceCharge, a.sourceCharged = workspaceSourceBytes(workspace), true
		if retained, err := options.admitSource(a.sourceCharge); err != nil {
			return nil, err
		} else if !retained {
			return analysisBudgetUnavailable(d, root, overlays, options.sourceLimit), nil
		}
		if parsedSyntax.Weight() > 0 && parsedSyntax.Weight() <= options.syntaxRoom() {
			a.syntaxSnapshot = parsedSyntax
		}
		input, err := workspace.SemanticInput()
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
			return a, nil
		}
		for _, imported := range input.Imports {
			a.imports[sourceKey{imported.FromPackage, imported.ImportPath}] = imported.ToPackage
		}
		packages, err := workspace.SemanticPackages()
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
			return a, nil
		}
		for _, p := range packages {
			loaded, ok := workspace.Package(p.Path)
			if !ok {
				continue
			}
			for _, file := range loaded.Sources {
				view := sourceView{Text: file.Bytes, Module: file.Module}
				if loaded.SourceKind == workspaceloader.SourceKindSDK {
					if options.sdkSourceNavigation {
						view.SDKURI = sdkSourceURI(file.Path, file.Bytes)
					}
				} else {
					view.Path = filepath.Join(loaded.SourceRoot, filepath.FromSlash(file.Path))
				}
				a.files[sourceKey{p.ID, file.Path}] = view
			}
		}
		result, err := workspacecompiler.CheckContext(ctx, workspace)
		a.result = result.Semantic
		if err == nil {
			a.snapshot, _ = workspaceloader.CaptureFileSnapshot(ctx, workspace, overlays)
		}
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
				a.add(filename, text, 0, 0, sourceNotInProjectDiagnostic, "Source is outside the project's resolved source roots or reachable package closure")
			}
		}
	} else {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(filepath.Base(d.Path), d.Text), source.DefaultLimits())
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
			return a, nil
		}
		for _, diag := range parsed.Tree.Diagnostics {
			a.add(d.Path, d.Text, diag.Span.StartByte, diag.Span.EndByte, diag.Code, diag.Code)
		}
		a.repair = singleSyntaxRepair(d.Path, filepath.Base(d.Path), parsed.Tree.Diagnostics)
		a.files[sourceKey{0, filepath.Base(d.Path)}] = sourceView{Path: d.Path, Text: d.Text, Module: parsed.AST}
		a.sourceCharge, a.sourceCharged = int64(len(d.Text)), true
		if retained, err := options.admitSource(a.sourceCharge); err != nil {
			return nil, err
		} else if !retained {
			return analysisBudgetUnavailable(d, root, nil, options.sourceLimit), nil
		}
		if parsed.AST == nil {
			return a, nil
		}
		result, err := compiler.CheckContext(ctx, []semantic.File{{Path: filepath.Base(d.Path), Module: parsed.AST}})
		a.result = result.Semantic
		if err == nil {
			a.standalone, a.standaloneValid = sha256.Sum256(d.Text), true
		}
		if err != nil {
			a.add(d.Path, d.Text, 0, 0, "editor-analysis-failed", internalDiagnostic(err))
		}
	}
	a.result.Resolutions = append(a.result.Resolutions, a.result.SelectorResolutions...)
	for _, diag := range a.result.Diagnostics {
		f, ok := a.file(diag.Primary)
		if !ok || f.Path == "" {
			continue
		}
		message := diag.Code
		if diag.ExpectedType != hir.ParseType("") || diag.ActualType != hir.ParseType("") {
			message += ": expected " + diag.ExpectedType.String() + ", got " + diag.ActualType.String()
		}
		var related []diagnosticRelatedInfo
		if options.diagnosticRelatedInfo && diag.Code == "E2105" {
			related = a.borrowMoveRelatedInfo(diag)
		}
		a.add(f.Path, f.Text, diag.Primary.StartByte, diag.Primary.EndByte, diag.Code, message, related...)
	}
	for _, u := range a.result.Unsupported {
		if f, ok := a.file(u.Span); ok && f.Path != "" {
			a.add(f.Path, f.Text, u.Span.StartByte, u.Span.EndByte, "unsupported-source", u.Feature)
		}
	}
	for _, diags := range a.diagnostics {
		// An excluded overlay has its own diagnostic and contributes no
		// compiler facts to the shared project analysis.
		if slices.ContainsFunc(diags, func(diag lspDiagnostic) bool { return diag.Code != sourceNotInProjectDiagnostic }) {
			return a, nil
		}
	}
	a.clean = true
	return a, nil
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
func (a *analysis) borrowMoveRelatedInfo(diag semantic.Diagnostic) []diagnosticRelatedInfo {
	var locations []diagnosticRelatedInfo
	for _, related := range diag.Related {
		var message string
		switch related.Role {
		case "owner":
			message = "Owner declared here"
		case "loan":
			message = "Borrow created here"
		case "last-use":
			// Flow uses the move span as a fallback for loans held by a loop or call.
			if related.Span == diag.Primary {
				continue
			}
			message = "Borrow last used here"
		default:
			continue
		}
		if location, ok := a.sourceLocation(related.Span); ok {
			locations = append(locations, diagnosticRelatedInfo{Location: location, Message: message})
		}
	}
	return locations
}

func (a *analysis) add(filename string, text []byte, start, end int, code, message string, related ...diagnosticRelatedInfo) {
	if strings.TrimSpace(message) == "" {
		message = code
	}
	var location Range
	if start != 0 || end != 0 {
		location = a.sourceRange(filename, text, start, end)
	}
	a.diagnostics[filename] = append(a.diagnostics[filename], lspDiagnostic{
		Range: location, Severity: 1, Code: code, Source: "fairyc", Message: message,
		RelatedInformation: related,
	})
}

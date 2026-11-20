package languageserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0disoft/fairyc/internal/source"
	"io"
	"path/filepath"
	"sort"
	"unicode/utf8"
)

type document struct {
	URI, Path       string
	Text            []byte
	Version         int
	needsFullText   bool
	View            *analysis
	LastClean       *analysis
	analysisEvicted bool
	completionIndex *completionContextIndex
	positionIndex   *positionIndex
}
type server struct {
	ctx                   context.Context
	out                   io.Writer
	workspaceRoots        []string
	initialized, shutdown bool
	versionedEdits        bool
	renameSupported       bool
	sdkSourceNavigation   bool
	analysisPending       bool
	analysisBudget        int64
	analysisClock         uint64
	analysisProtect       map[*analysis]struct{}
	docs                  map[string]*document
}
type documentID struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}
type documentParams struct {
	TextDocument documentID `json:"textDocument"`
	Position     Position   `json:"position"`
}

func (s *server) handle(m message) error {
	bad := func(reason string) error {
		if len(m.ID) == 0 {
			return s.notify("window/logMessage", map[string]any{"type": 2, "message": reason})
		}
		return s.response(m.ID, nil, -32602, reason)
	}
	if m.Method == "initialize" {
		if s.initialized {
			return s.response(m.ID, nil, -32600, "already initialized")
		}
		var p initializeParams
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid initialize parameters")
		}
		roots, err := p.roots()
		if err != nil {
			return bad(err.Error())
		}
		s.workspaceRoots = roots
		s.initialized = true
		s.sdkSourceNavigation = p.InitializationOptions.SDKSourceNavigation
		s.renameSupported = p.Capabilities.Workspace.WorkspaceEdit.DocumentChanges
		s.versionedEdits = s.renameSupported && p.Capabilities.TextDocument.CodeAction.LiteralSupport != nil
		var codeActionProvider any = false
		var renameProvider any = false
		if s.versionedEdits {
			codeActionProvider = map[string]any{"codeActionKinds": []string{"quickfix"}}
		}
		if s.renameSupported {
			renameProvider = map[string]bool{"prepareProvider": true}
		}
		return s.response(m.ID, map[string]any{
			"serverInfo": map[string]any{"name": "fairyc-lsp", "version": "0.1.0"},
			"capabilities": map[string]any{
				"positionEncoding":   "utf-16",
				"textDocumentSync":   map[string]any{"openClose": true, "change": 2, "save": map[string]any{"includeText": false}},
				"definitionProvider": true, "hoverProvider": true, "referencesProvider": true, "documentFormattingProvider": true,
				"documentRangeFormattingProvider": true,
				"codeActionProvider":              codeActionProvider,
				"renameProvider":                  renameProvider,
				"experimental":                    map[string]bool{"fairycSdkSourceNavigation": s.sdkSourceNavigation},
				"semanticTokensProvider": map[string]any{
					"legend": map[string]any{"tokenTypes": semanticTokenTypes, "tokenModifiers": semanticTokenModifiers},
					"full":   true,
				},
				"completionProvider": map[string]any{"resolveProvider": false, "triggerCharacters": []string{":", "."}},
				"workspace":          map[string]any{"workspaceFolders": map[string]bool{"supported": true, "changeNotifications": true}},
			},
		}, 0, "")
	}
	if !s.initialized {
		return s.response(m.ID, nil, -32002, "server not initialized")
	}
	if s.shutdown {
		return s.response(m.ID, nil, -32600, "server is shutting down")
	}
	switch m.Method {
	case "initialized", "$/cancelRequest", "$/setTrace":
		return nil
	case "shutdown":
		s.shutdown = true
		return s.response(m.ID, nil, 0, "")
	case "fairyc/sdkSource":
		if !s.sdkSourceNavigation {
			return s.response(m.ID, nil, -32601, "method not supported")
		}
		var p documentID
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid SDK source parameters")
		}
		text, ok := readSDKSource(p.URI)
		if !ok {
			return bad("unknown embedded SDK source URI")
		}
		return s.response(m.ID, text, 0, "")
	case "textDocument/didOpen":
		var p struct {
			TextDocument struct {
				URI        string `json:"uri"`
				Text       string `json:"text"`
				Version    int    `json:"version"`
				LanguageID string `json:"languageId"`
			} `json:"textDocument"`
		}
		if json.Unmarshal(m.Params, &p) != nil || p.TextDocument.LanguageID != "fairyc" {
			return bad("invalid Fairy C document")
		}
		d := p.TextDocument
		filename, err := filePath(d.URI)
		if err != nil || filepath.Ext(filename) != ".fy" || (len(s.workspaceRoots) > 0 && s.workspaceRoot(filename) == "") {
			return bad("document is outside the local workspace or is not .fy")
		}
		if _, exists := s.docs[d.URI]; exists {
			return bad("document already open")
		}
		for _, opened := range s.docs {
			if sameDocumentPath(opened.Path, filename) {
				return bad("document already open")
			}
		}
		if len(s.docs) >= 32 || !s.fits(d.URI, d.Text) {
			return bad("editor document limit exceeded")
		}
		s.docs[d.URI] = &document{URI: d.URI, Path: filename, Text: []byte(d.Text), Version: d.Version}
		return s.recheck()
	case "textDocument/didChange":
		var p struct {
			TextDocument   documentID      `json:"textDocument"`
			ContentChanges json.RawMessage `json:"contentChanges"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid change parameters")
		}
		d := s.docs[p.TextDocument.URI]
		if d == nil {
			return bad("document is not open")
		}
		if p.TextDocument.Version <= d.Version {
			return nil
		}
		text, err := applyDocumentChanges(d.Text, p.ContentChanges, s.documentByteLimit(d.URI), d.needsFullText)
		if err != nil {
			d.needsFullText = true
			s.analysisPending = true
			return bad(err.Error())
		}
		d.Text = text
		d.Version = p.TextDocument.Version
		d.needsFullText = false
		d.completionIndex = nil
		d.positionIndex = nil
		return s.recheck()
	case "textDocument/didSave":
		return s.recheck()
	case "workspace/didChangeWatchedFiles":
		return s.recheck()
	case "workspace/didChangeWorkspaceFolders":
		var p struct {
			Event *workspaceFolderChange `json:"event"`
		}
		if json.Unmarshal(m.Params, &p) != nil || p.Event == nil {
			return bad("invalid workspace folder change")
		}
		roots, err := p.Event.apply(s.workspaceRoots)
		if err != nil {
			return bad(err.Error())
		}
		s.workspaceRoots = roots
		return s.recheck()
	case "textDocument/didClose":
		var p documentParams
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid close parameters")
		}
		delete(s.docs, p.TextDocument.URI)
		if err := s.notify("textDocument/publishDiagnostics", map[string]any{"uri": p.TextDocument.URI, "diagnostics": []any{}}); err != nil {
			return err
		}
		return s.recheck()
	case "textDocument/prepareRename", "textDocument/rename":
		if !s.renameSupported {
			return s.response(m.ID, nil, -32601, "method not supported")
		}
		var p struct {
			documentParams
			NewName string `json:"newName"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid rename parameters")
		}
		d := s.docs[p.TextDocument.URI]
		if d == nil {
			return bad("document is not open")
		}
		if d.needsFullText {
			return s.response(m.ID, nil, -32801, "document needs full synchronization")
		}
		offset, valid := d.positionByte(p.Position)
		if !valid {
			return bad("invalid UTF-16 rename position")
		}
		if err := s.ensureAnalysis(d); err != nil {
			return err
		}
		plan, err := s.planRename(d, offset)
		if errors.Is(err, errRenameUnavailable) && m.Method == "textDocument/prepareRename" {
			return s.response(m.ID, nil, 0, "")
		}
		if err == nil && m.Method == "textDocument/prepareRename" {
			return s.response(m.ID, map[string]any{
				"range": textRange(d.Text, plan.cursor.start, plan.cursor.end), "placeholder": plan.name,
			}, 0, "")
		}
		if err == nil {
			var edits []documentEdit
			edits, err = s.rename(d, plan, p.NewName)
			if err == nil {
				return s.response(m.ID, map[string]any{"documentChanges": edits}, 0, "")
			}
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return s.response(m.ID, nil, -32803, err.Error())
	case "textDocument/codeAction":
		var p struct {
			documentParams
			Range   json.RawMessage   `json:"range"`
			Context codeActionContext `json:"context"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid code action parameters")
		}
		d := s.docs[p.TextDocument.URI]
		if d == nil {
			return bad("document is not open")
		}
		if d.needsFullText {
			return s.response(m.ID, nil, -32801, "document needs full synchronization")
		}
		start, end, valid := rangeOffsets(d.Text, p.Range)
		if !valid {
			return bad("invalid UTF-16 code action range")
		}
		actions, err := s.codeActions(d, start, end, p.Context)
		if err != nil {
			return err
		}
		return s.response(m.ID, actions, 0, "")
	case "textDocument/formatting", "textDocument/rangeFormatting":
		var p struct {
			documentParams
			Range json.RawMessage `json:"range"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid formatting parameters")
		}
		d := s.docs[p.TextDocument.URI]
		if d == nil {
			return bad("document is not open")
		}
		if d.needsFullText {
			return s.response(m.ID, nil, -32801, "document needs full synchronization")
		}
		start, end := 0, len(d.Text)
		if m.Method == "textDocument/rangeFormatting" {
			var valid bool
			start, end, valid = rangeOffsets(d.Text, p.Range)
			if !valid {
				return bad("invalid UTF-16 formatting range")
			}
		}
		edits, err := s.formatDocument(d, start, end)
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return err
		case errors.Is(err, errFormattingLimit):
			return s.response(m.ID, nil, -32803, err.Error())
		case err != nil:
			return s.response(m.ID, nil, -32803, "document could not be formatted")
		default:
			return s.response(m.ID, edits, 0, "")
		}
	case "textDocument/definition", "textDocument/completion", "textDocument/hover", "textDocument/references", "textDocument/semanticTokens/full":
		var p struct {
			documentParams
			Context struct {
				IncludeDeclaration *bool `json:"includeDeclaration"`
			} `json:"context"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid position parameters")
		}
		d := s.docs[p.TextDocument.URI]
		if d == nil {
			return bad("document is not open")
		}
		if d.needsFullText {
			return s.response(m.ID, nil, -32801, "document needs full synchronization")
		}
		var offset int
		if m.Method != "textDocument/semanticTokens/full" {
			var ok bool
			offset, ok = d.positionByte(p.Position)
			if !ok {
				return bad("invalid UTF-16 position")
			}
		}
		if err := s.ensureAnalysis(d); err != nil {
			return err
		}
		switch m.Method {
		case "textDocument/definition":
			return s.response(m.ID, s.definition(d, offset), 0, "")
		case "textDocument/hover":
			return s.response(m.ID, s.hover(d, offset), 0, "")
		case "textDocument/semanticTokens/full":
			tokens, err := s.semanticTokens(d)
			if errors.Is(err, errSemanticTokenLimit) {
				return s.response(m.ID, nil, -32803, err.Error())
			}
			if err != nil {
				return err
			}
			return s.response(m.ID, tokens, 0, "")
		case "textDocument/references":
			if p.Context.IncludeDeclaration == nil {
				return bad("reference context must include includeDeclaration")
			}
			locations, err := s.references(d, offset, *p.Context.IncludeDeclaration)
			if errors.Is(err, errReferenceLimit) {
				return s.response(m.ID, nil, -32803, err.Error())
			}
			if err != nil {
				return err
			}
			return s.response(m.ID, locations, 0, "")
		default:
			items := s.completion(d, offset)
			if s.ctx != nil && s.ctx.Err() != nil {
				return s.ctx.Err()
			}
			return s.response(m.ID, items, 0, "")
		}
	default:
		return s.response(m.ID, nil, -32601, "method not supported")
	}
}

func (s *server) ensureAnalysis(d *document) error {
	if s.analysisPending {
		return s.recheckFocused(d.URI, false)
	}
	if d.analysisEvicted || d.View == nil {
		return s.recheckFocused(d.URI, true)
	}
	return nil
}
func (s *server) fits(uri, text string) bool {
	return len(text) <= s.documentByteLimit(uri) && utf8.ValidString(text)
}

const maxOpenDocumentBytes = 32 << 20

func (s *server) documentByteLimit(uri string) int {
	remaining := maxOpenDocumentBytes
	for key, d := range s.docs {
		if key != uri {
			remaining -= len(d.Text)
		}
	}
	return min(remaining, source.DefaultLimits().MaxBytes)
}
func (s *server) recheck() error {
	return s.recheckFocused("", false)
}

func (s *server) recheckFocused(focusURI string, onlyFocus bool) error {
	// A newer notification can cancel this analysis and then fail validation.
	// Queries must recover it before reading views of older document contents.
	s.analysisPending = true
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	keys := make([]string, 0, len(s.docs))
	for uri := range s.docs {
		keys = append(keys, uri)
	}
	sort.Strings(keys)
	type analysisGroup struct {
		key, root string
		documents []*document
	}
	groups := make(map[string]*analysisGroup)
	groupKeys := make([]string, 0)
	focusGroup := ""
	for _, uri := range keys {
		d := s.docs[uri]
		root := s.projectRoot(d.Path)
		key := root
		if key == "" {
			key = d.Path
		}
		group := groups[key]
		if group == nil {
			group = &analysisGroup{key: key, root: root}
			groups[key] = group
			groupKeys = append(groupKeys, key)
		}
		group.documents = append(group.documents, d)
		if uri == focusURI {
			focusGroup = key
		}
	}
	sort.Strings(groupKeys)
	if focusGroup != "" {
		for index, key := range groupKeys {
			if key == focusGroup {
				copy(groupKeys[1:index+1], groupKeys[:index])
				groupKeys[0] = focusGroup
				break
			}
		}
	}
	if onlyFocus && focusGroup != "" {
		groupKeys = groupKeys[:1]
	}
	s.analysisProtect = map[*analysis]struct{}{}
	defer func() { s.analysisProtect = nil }()
	for _, key := range groupKeys {
		group := groups[key]
		view, err := s.analyzeContext(ctx, group.documents[0], group.root)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		s.touchAnalysis(view)
		if key == focusGroup {
			s.analysisProtect[view] = struct{}{}
		}
		for _, d := range group.documents {
			s.setDocumentView(d, view)
			diagnostics := view.diagnostics[d.Path]
			if diagnostics == nil {
				diagnostics = []lspDiagnostic{}
			}
			if err := s.notify("textDocument/publishDiagnostics", map[string]any{"uri": d.URI, "version": d.Version, "diagnostics": diagnostics}); err != nil {
				return err
			}
		}
	}
	if retained, err := s.reserveAnalysisSource(0); err != nil {
		return err
	} else if !retained {
		return fmt.Errorf("editor analysis source budget cannot retire enough analyses")
	}
	s.analysisPending = false
	return nil
}

func (s *server) setDocumentView(d *document, view *analysis) {
	if d.LastClean != nil && (d.LastClean.root != view.root || view.unsynchronized) {
		d.LastClean = nil
	}
	d.View = view
	if view.clean {
		d.LastClean = view
	}
	d.analysisEvicted = view.budgetExceeded
}
func internalDiagnostic(err error) string { return fmt.Sprintf("editor analysis: %v", err) }

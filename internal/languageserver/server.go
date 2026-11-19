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
		return s.response(m.ID, map[string]any{
			"serverInfo": map[string]any{"name": "fairyc-lsp", "version": "0.1.0"},
			"capabilities": map[string]any{
				"positionEncoding":   "utf-16",
				"textDocumentSync":   map[string]any{"openClose": true, "change": 2, "save": map[string]any{"includeText": false}},
				"definitionProvider": true, "hoverProvider": true, "referencesProvider": true, "documentFormattingProvider": true,
				"documentRangeFormattingProvider": true,
				"completionProvider":              map[string]any{"resolveProvider": false, "triggerCharacters": []string{":"}},
				"workspace":                       map[string]any{"workspaceFolders": map[string]bool{"supported": true, "changeNotifications": true}},
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
	case "textDocument/definition", "textDocument/completion", "textDocument/hover", "textDocument/references":
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
		offset, ok := d.positionByte(p.Position)
		if !ok {
			return bad("invalid UTF-16 position")
		}
		if s.analysisPending {
			if err := s.recheckFocused(d.URI, false); err != nil {
				return err
			}
		} else if d.analysisEvicted || d.View == nil {
			if err := s.recheckFocused(d.URI, true); err != nil {
				return err
			}
		}
		switch m.Method {
		case "textDocument/definition":
			return s.response(m.ID, s.definition(d, offset), 0, "")
		case "textDocument/hover":
			return s.response(m.ID, s.hover(d, offset), 0, "")
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
			return s.response(m.ID, s.completion(d, offset), 0, "")
		}
	default:
		return s.response(m.ID, nil, -32601, "method not supported")
	}
}
func (s *server) fits(uri, text string) bool {
	return len(text) <= s.documentByteLimit(uri) && utf8.ValidString(text)
}

func (s *server) documentByteLimit(uri string) int {
	remaining := 32 << 20
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

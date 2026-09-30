package languageserver

import (
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/source"
	"io"
	"path/filepath"
	"sort"
	"unicode/utf8"
)

type document struct {
	URI, Path string
	Text      []byte
	Version   int
	View      *analysis
	LastClean *analysis
}
type server struct {
	out                   io.Writer
	root                  string
	initialized, shutdown bool
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
		var p struct {
			RootURI          string `json:"rootUri"`
			RootPath         string `json:"rootPath"`
			WorkspaceFolders []struct {
				URI string `json:"uri"`
			} `json:"workspaceFolders"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid initialize parameters")
		}
		if len(p.WorkspaceFolders) > 1 {
			return bad("preview supports one workspace folder")
		}
		if p.RootURI == "" && len(p.WorkspaceFolders) == 1 {
			p.RootURI = p.WorkspaceFolders[0].URI
		}
		if p.RootURI != "" {
			var err error
			s.root, err = filePath(p.RootURI)
			if err != nil {
				return bad(err.Error())
			}
		} else if p.RootPath != "" {
			if !filepath.IsAbs(p.RootPath) {
				return bad("rootPath must be absolute")
			}
			s.root = filepath.Clean(p.RootPath)
		}
		s.initialized = true
		return s.response(m.ID, map[string]any{"serverInfo": map[string]any{"name": "fairyc-lsp", "version": "0.1.0"}, "capabilities": map[string]any{"positionEncoding": "utf-16", "textDocumentSync": map[string]any{"openClose": true, "change": 1, "save": map[string]any{"includeText": false}}, "definitionProvider": true, "completionProvider": map[string]any{"resolveProvider": false, "triggerCharacters": []string{":"}}}}, 0, "")
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
		if err != nil || filepath.Ext(filename) != ".fy" || (s.root != "" && !contained(s.root, filename)) {
			return bad("document is outside the local workspace or is not .fy")
		}
		if _, exists := s.docs[d.URI]; exists {
			return bad("document already open")
		}
		if len(s.docs) >= 32 || !s.fits(d.URI, d.Text) {
			return bad("editor document limit exceeded")
		}
		s.docs[d.URI] = &document{URI: d.URI, Path: filename, Text: []byte(d.Text), Version: d.Version}
		return s.recheck()
	case "textDocument/didChange":
		var p struct {
			TextDocument   documentID `json:"textDocument"`
			ContentChanges []struct {
				Text  string `json:"text"`
				Range *Range `json:"range"`
			} `json:"contentChanges"`
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
		if len(p.ContentChanges) != 1 || p.ContentChanges[0].Range != nil {
			return bad("full document synchronization required")
		}
		if !s.fits(d.URI, p.ContentChanges[0].Text) {
			return bad("editor document limit exceeded")
		}
		d.Text = []byte(p.ContentChanges[0].Text)
		d.Version = p.TextDocument.Version
		return s.recheck()
	case "textDocument/didSave":
		return s.recheck()
	case "workspace/didChangeWatchedFiles":
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
	case "textDocument/definition", "textDocument/completion":
		var p documentParams
		if json.Unmarshal(m.Params, &p) != nil {
			return bad("invalid position parameters")
		}
		d := s.docs[p.TextDocument.URI]
		if d == nil {
			return bad("document is not open")
		}
		offset, ok := positionByte(d.Text, p.Position)
		if !ok {
			return bad("invalid UTF-16 position")
		}
		if m.Method == "textDocument/definition" {
			return s.response(m.ID, s.definition(d, offset), 0, "")
		}
		return s.response(m.ID, s.completion(d, offset), 0, "")
	default:
		return s.response(m.ID, nil, -32601, "method not supported")
	}
}
func (s *server) fits(uri, text string) bool {
	if !utf8.ValidString(text) || len(text) > source.DefaultLimits().MaxBytes {
		return false
	}
	total := len(text)
	for key, d := range s.docs {
		if key != uri {
			total += len(d.Text)
		}
	}
	return total <= 32<<20
}
func (s *server) recheck() error {
	keys := make([]string, 0, len(s.docs))
	for uri := range s.docs {
		keys = append(keys, uri)
	}
	sort.Strings(keys)
	views := map[string]*analysis{}
	for _, uri := range keys {
		d := s.docs[uri]
		root := s.projectRoot(d.Path)
		key := root
		if key == "" {
			key = d.Path
		}
		view := views[key]
		if view == nil {
			view = s.analyze(d, root)
			views[key] = view
		}
		d.View = view
		if view.clean {
			d.LastClean = view
		}
		diagnostics := view.diagnostics[d.Path]
		if diagnostics == nil {
			diagnostics = []lspDiagnostic{}
		}
		if err := s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "version": d.Version, "diagnostics": diagnostics}); err != nil {
			return err
		}
	}
	return nil
}
func internalDiagnostic(err error) string { return fmt.Sprintf("editor analysis: %v", err) }

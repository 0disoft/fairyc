package languageserver

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalysisSourceBudgetExactBoundaries(t *testing.T) {
	text := "module main; fn i32 main() { return 0; }"
	sourceBytes := int64(len(text))
	for _, test := range []struct {
		name    string
		budget  int64
		clean   bool
		charged bool
	}{
		{"below", sourceBytes - 1, false, false},
		{"exact", sourceBytes, true, true},
		{"above", sourceBytes + 1, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			filename := filepath.Join(root, "main.fy")
			uri := fileURI(filename)
			var output bytes.Buffer
			s := server{out: &output, docs: map[string]*document{}, analysisBudget: test.budget}
			send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
			send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
				"uri": uri, "languageId": "fairyc", "version": 1, "text": text,
			}})
			d := s.docs[uri]
			if d.View.clean != test.clean || d.View.budgetExceeded == test.clean {
				t.Fatalf("clean=%v budgetExceeded=%v", d.View.clean, d.View.budgetExceeded)
			}
			if test.charged && analysisSourceBytes(d.View) != sourceBytes {
				t.Fatalf("source charge = %d, want %d", analysisSourceBytes(d.View), sourceBytes)
			}
			if !test.clean && (len(d.View.diagnostics[filename]) != 1 || d.View.diagnostics[filename][0].Code != analysisBudgetDiagnostic) {
				t.Fatalf("budget diagnostics = %+v", d.View.diagnostics[filename])
			}
		})
	}
}

func TestAnalysisSourceBudgetBoundsSyntaxFailedStandalone(t *testing.T) {
	text := "module main; fn i32 main() {"
	root := t.TempDir()
	filename := filepath.Join(root, "main.fy")
	uri := fileURI(filename)
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}, analysisBudget: int64(len(text) - 1)}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "languageId": "fairyc", "version": 1, "text": text,
	}})
	d := s.docs[uri]
	if !d.View.budgetExceeded || d.LastClean != nil || len(d.View.diagnostics[filename]) != 1 || d.View.diagnostics[filename][0].Code != analysisBudgetDiagnostic {
		t.Fatalf("syntax-failed budget result = %+v", d.View)
	}
}

func writeBudgetWorkspace(t *testing.T) (root, mainPath, answerPath string) {
	t.Helper()
	root = t.TempDir()
	writeProjectRootFile(t, filepath.Join(root, "fairy.toml"), projectRootManifest(
		"test.lsp-budget.root", "budget-root", "src", "",
	))
	mainPath = filepath.Join(root, "src", "main.fy")
	answerPath = filepath.Join(root, "src", "answer.fy")
	writeProjectRootFile(t, mainPath, "module main; import answer as answer; fn i32 main() { return answer::value(); }")
	writeProjectRootFile(t, answerPath, "module answer; pub fn i32 value() { return 42; }")
	return root, mainPath, answerPath
}

func openBudgetWorkspace(t *testing.T, budget int64) (*server, *document, *document) {
	t.Helper()
	root, mainPath, answerPath := writeBudgetWorkspace(t)
	mainURI, answerURI := fileURI(mainPath), fileURI(answerPath)
	var output bytes.Buffer
	s := &server{out: &output, docs: map[string]*document{}, analysisBudget: budget}
	send(t, s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	answerText, err := os.ReadFile(answerPath)
	if err != nil {
		t.Fatal(err)
	}
	send(t, s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": answerURI, "languageId": "fairyc", "version": 1, "text": string(answerText),
	}})
	send(t, s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": mainURI, "languageId": "fairyc", "version": 1,
		"text": "module main; import answer as answer; fn i32 main() { return answer::value(); }",
	}})
	return s, s.docs[mainURI], s.docs[answerURI]
}

func TestAnalysisSourceBudgetCountsSharedViewsAndLastClean(t *testing.T) {
	for _, test := range []struct {
		name            string
		budgetSlack     int64
		wantLastClean   bool
		wantAnalysisLen int
	}{
		{"exact old and new", 0, true, 2},
		{"old evicted for new", -1, false, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, mainDocument, answerDocument := openBudgetWorkspace(t, 0)
			clean := mainDocument.View
			if !clean.clean || mainDocument.LastClean != clean || answerDocument.View != clean || answerDocument.LastClean != clean {
				t.Fatal("project documents do not share the clean analysis")
			}
			if len(clean.files) != 2 || analysisSourceBytes(clean) <= int64(len(mainDocument.Text)) {
				t.Fatalf("source closure not charged: files=%d charge=%d open=%d", len(clean.files), analysisSourceBytes(clean), len(mainDocument.Text))
			}
			if retained := s.retainedAnalyses(); len(retained) != 1 || s.retainedAnalysisSourceBytes() != analysisSourceBytes(clean) {
				t.Fatalf("shared clean accounting = %d analyses / %d bytes", len(retained), s.retainedAnalysisSourceBytes())
			}
			s.analysisBudget = 2*analysisSourceBytes(clean) + test.budgetSlack
			dirtyMain := "module main; import answer as answer; fn i32 main() { return false; }"
			if padding := len(mainDocument.Text) - len(dirtyMain); padding > 0 {
				dirtyMain = dirtyMain[:len(dirtyMain)-1] + strings.Repeat(" ", padding) + "}"
			}
			send(t, s, 0, "textDocument/didChange", map[string]any{
				"textDocument":   map[string]any{"uri": mainDocument.URI, "version": 2},
				"contentChanges": []any{map[string]any{"text": dirtyMain}},
			})
			if mainDocument.View == clean || answerDocument.View != mainDocument.View || mainDocument.View.clean || mainDocument.View.budgetExceeded {
				t.Fatal("dirty project analysis was not shared")
			}
			if test.wantLastClean {
				if mainDocument.LastClean != clean || answerDocument.LastClean != clean {
					t.Fatal("exact budget did not retain the shared clean fallback")
				}
			} else if mainDocument.LastClean != nil || answerDocument.LastClean != nil {
				t.Fatal("old clean analysis survived after safe release")
			}
			if retained := s.retainedAnalyses(); len(retained) != test.wantAnalysisLen {
				t.Fatalf("retained analyses = %d, want %d", len(retained), test.wantAnalysisLen)
			}
			if s.retainedAnalysisSourceBytes() > s.analysisSourceLimit() {
				t.Fatalf("retained source bytes exceed cap: %d > %d", s.retainedAnalysisSourceBytes(), s.analysisSourceLimit())
			}
			send(t, s, 0, "textDocument/didClose", map[string]any{"textDocument": documentID{URI: answerDocument.URI}})
			send(t, s, 0, "textDocument/didClose", map[string]any{"textDocument": documentID{URI: mainDocument.URI}})
			if s.retainedAnalysisSourceBytes() != 0 {
				t.Fatalf("close retained %d analysis bytes", s.retainedAnalysisSourceBytes())
			}
		})
	}
}

func TestAnalysisSourceBudgetChargesUnreachableWorkspaceMembers(t *testing.T) {
	root := t.TempDir()
	writeProjectRootFile(t, filepath.Join(root, "fairy.toml"), projectRootManifest(
		"test.lsp-budget.catalog", "budget-catalog", "src",
		"[workspace]\nmembers = [\"a\", \"b\"]\n\n[dependencies.a]\npath = \"a\"\nversion = \"1.0.0\"\nfeatures = []\n",
	))
	mainText := "module main; import a as a; fn i32 main() { return a::value(); }"
	aText := "module a; pub fn i32 value() { return 1; }"
	bText := "module b; pub fn i32 unused() { return 2; }"
	writeProjectRootFile(t, filepath.Join(root, "src", "main.fy"), mainText)
	writeProjectRootFile(t, filepath.Join(root, "a", "fairy.toml"), projectRootManifest(
		"test.lsp-budget.a", "a", "src", "",
	))
	writeProjectRootFile(t, filepath.Join(root, "a", "src", "a.fy"), aText)
	writeProjectRootFile(t, filepath.Join(root, "b", "fairy.toml"), projectRootManifest(
		"test.lsp-budget.b", "b", "src", "",
	))
	writeProjectRootFile(t, filepath.Join(root, "b", "src", "b.fy"), bText)
	filename := filepath.Join(root, "src", "main.fy")
	uri := fileURI(filename)
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "languageId": "fairyc", "version": 1, "text": mainText,
	}})
	d := s.docs[uri]
	if !d.View.clean || len(d.View.files) != 2 {
		t.Fatalf("reachable closure = %+v", d.View)
	}
	wantCharge := int64(len(mainText) + len(aText) + len(bText))
	if got := analysisSourceBytes(d.View); got != wantCharge {
		t.Fatalf("catalog charge = %d, want %d including unreachable member", got, wantCharge)
	}
	s.analysisBudget = wantCharge - 1
	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": uri, "version": 2},
		"contentChanges": []any{map[string]any{"text": mainText + "\n"}},
	})
	if !d.View.budgetExceeded || d.LastClean != nil || s.retainedAnalysisSourceBytes() > s.analysisSourceLimit() {
		t.Fatalf("unreachable-member budget result: exceeded=%v clean=%v retained=%d cap=%d",
			d.View.budgetExceeded, d.LastClean != nil, s.retainedAnalysisSourceBytes(), s.analysisSourceLimit())
	}
}

func TestAnalysisBudgetFocusedRecoveryKeepsUnrelatedView(t *testing.T) {
	root := t.TempDir()
	text := "module main; fn i32 helper() { return 7; } fn i32 main() { return helper(); }"
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}, analysisBudget: int64(2 * len(text))}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	var documents []*document
	for index := 1; index <= 3; index++ {
		project := filepath.Join(root, "p"+string(rune('0'+index)))
		writeProjectRootFile(t, filepath.Join(project, "fairy.toml"), projectRootManifest(
			"test.lsp-budget.p"+string(rune('0'+index)), "p"+string(rune('0'+index)), "src", "",
		))
		filename := filepath.Join(project, "src", "main.fy")
		writeProjectRootFile(t, filename, text)
		uri := fileURI(filename)
		send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": uri, "languageId": "fairyc", "version": 1, "text": text,
		}})
		documents = append(documents, s.docs[uri])
	}
	if documents[0].View != nil || documents[1].View == nil || documents[2].View == nil {
		t.Fatal("three-project cap did not evict the oldest retained root")
	}
	thirdView := documents[2].View
	offset := strings.LastIndex(text, "helper()") + 3
	if got := s.definition(documents[0], offset); len(got) != 0 {
		t.Fatalf("evicted document returned stale definition: %+v", got)
	}
	position := bytePosition([]byte(text), offset)
	send(t, &s, 2, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": documents[0].URI}, "position": position,
	})
	if documents[0].View == nil || documents[2].View != thirdView || documents[1].View != nil {
		t.Fatal("focused recovery replaced an unrelated retained project")
	}
	definition := s.definition(documents[0], offset)
	if len(definition) != 1 || definition[0].URI != documents[0].URI || definition[0].Range.Start.Line != 0 {
		t.Fatalf("reloaded definition = %+v, clean=%v diagnostics=%+v resolutions=%d", definition,
			documents[0].View.clean, documents[0].View.diagnostics, len(documents[0].View.result.Resolutions))
	}
	send(t, &s, 3, "textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": documents[0].URI}, "position": bytePosition([]byte(text), offset),
	})
	found := false
	for _, item := range s.completion(documents[0], offset) {
		if item.Label == "helper" {
			found = true
		}
	}
	if !found {
		t.Fatal("reloaded completion missing helper")
	}
}

func TestAnalysisBudgetCountsClosedSourceOnFailedWorkspaceLoad(t *testing.T) {
	root, mainPath, answerPath := writeBudgetWorkspace(t)
	broken := "module answer;" + strings.Repeat(" ", 1024) + "fn i32 unfinished("
	writeProjectRootFile(t, answerPath, broken)
	mainText, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	uri := fileURI(mainPath)
	for _, budget := range []int64{int64(len(broken) - 1), int64(len(broken))} {
		var output bytes.Buffer
		s := server{out: &output, docs: map[string]*document{}, analysisBudget: budget}
		send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
		send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": uri, "languageId": "fairyc", "version": 1, "text": string(mainText),
		}})
		view := s.docs[uri].View
		if view.clean || view.budgetExceeded != (budget < int64(len(broken))) {
			t.Fatalf("budget=%d exceeded=%v clean=%v", budget, view.budgetExceeded, view.clean)
		}
		if view.budgetExceeded {
			if len(view.positions) != 0 || analysisSourceBytes(view) != 0 {
				t.Fatal("unavailable diagnostic retained source through a position index")
			}
		} else if got := analysisSourceBytes(view); got != int64(len(broken)) {
			t.Fatalf("failed-load source charge=%d want=%d", got, len(broken))
		}
	}
}

func TestNewAnalysisClearsLastCleanFromAnotherRoot(t *testing.T) {
	d := &document{
		View:      &analysis{root: "old", files: map[sourceKey]sourceView{{Path: "main.fy"}: {Path: "main.fy", Text: []byte("old")}}},
		LastClean: &analysis{root: "unrelated", files: map[sourceKey]sourceView{{Path: "main.fy"}: {Path: "main.fy", Text: []byte("clean")}}},
	}
	s := server{}
	view := &analysis{root: "old", files: map[sourceKey]sourceView{{Path: "main.fy"}: {Path: "main.fy", Text: []byte("dirty!")}}}
	s.setDocumentView(d, view)
	if d.View != view || d.LastClean != nil {
		t.Fatalf("cross-root LastClean survived: %+v", d.LastClean)
	}
}

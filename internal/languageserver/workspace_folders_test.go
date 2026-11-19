package languageserver

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
)

func TestWorkspaceFoldersIsolateProjectsAndHandleChanges(t *testing.T) {
	first, firstPath := editorFixture(t)
	second, secondPath := editorFixture(t)
	outside, outsidePath := editorFixture(t)
	firstURI, secondURI := fileURI(firstPath), fileURI(secondPath)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":          fileURI(outside),
		"workspaceFolders": []workspaceFolder{{URI: fileURI(first)}, {URI: fileURI(second)}},
	})
	var init struct {
		Capabilities struct {
			Workspace struct {
				WorkspaceFolders map[string]bool `json:"workspaceFolders"`
			} `json:"workspace"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(editorResult(t, &out), &init); err != nil ||
		!init.Capabilities.Workspace.WorkspaceFolders["supported"] ||
		!init.Capabilities.Workspace.WorkspaceFolders["changeNotifications"] {
		t.Fatalf("workspace capabilities missing: %s (%v)", out.String(), err)
	}
	open := func(uri, text string) {
		t.Helper()
		send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
		}})
	}
	good := "module main; fn i32 main(){ return 1; }"
	bad := "module main; fn i32 main(){ return false; }"
	open(firstURI, good)
	open(secondURI, bad)
	open(fileURI(outsidePath), good)
	a, b := s.docs[firstURI], s.docs[secondURI]
	if len(s.docs) != 2 || a == nil || b == nil || a.View == nil || b.View == nil ||
		!a.View.clean || b.View.clean || a.View.root != first || b.View.root != second {
		t.Fatalf("projects not isolated: %s", out.String())
	}

	update := func(added, removed string) {
		t.Helper()
		send(t, &s, 0, "workspace/didChangeWorkspaceFolders", map[string]any{
			"event": workspaceFolderChange{Added: []workspaceFolder{{URI: added}}, Removed: []workspaceFolder{{URI: removed}}},
		})
	}
	update("https://example.org/remote", fileURI(second))
	if !slices.Equal(s.workspaceRoots, []string{first, second}) || b.View.root != second {
		t.Fatal("invalid folder update partially changed the workspace")
	}
	update(fileURI(outside), fileURI(second))
	if b.View.root != "" || b.LastClean != nil || string(b.Text) != bad || b.Version != 1 {
		t.Fatal("removing a folder did not retain the document as a single-file overlay")
	}
	open(fileURI(outsidePath), good)
	if d := s.docs[fileURI(outsidePath)]; d == nil || d.View == nil || !d.View.clean || d.View.root != outside {
		t.Fatal("newly added folder did not admit its project")
	}
}

func TestNearestWorkspaceFolderBoundsProjectLookup(t *testing.T) {
	outer, nested, _ := siblingWorkspaceFixture(t)
	filename := filepath.Join(nested, "src", "a.fy")
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"workspaceFolders": []workspaceFolder{{URI: fileURI(outer)}, {URI: fileURI(nested)}},
	})
	if got := s.projectRoot(filename); got != nested {
		t.Fatalf("nested lookup selected %q, want %q", got, nested)
	}
	send(t, &s, 0, "workspace/didChangeWorkspaceFolders", map[string]any{
		"event": workspaceFolderChange{Removed: []workspaceFolder{{URI: fileURI(nested)}}},
	})
	if got := s.projectRoot(filename); got != outer {
		t.Fatalf("project lookup after removal selected %q, want %q", got, outer)
	}
}

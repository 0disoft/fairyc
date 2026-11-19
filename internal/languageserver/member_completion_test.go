package languageserver

import (
	"bytes"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestMemberCompletionChecksIncompleteOverlayAndVisibility(t *testing.T) {
	root, path := editorFixture(t)
	writeProjectRootFile(t, filepath.Join(root, "src", "geometry.fy"), `module geometry;
pub struct Point { i32 column; i32 row; }
pub fn i32 Point::read(ref Point this) { return this.column; }
fn i32 Point::hidden(ref Point this) { return this.row; }
pub fn i32 Point::factory() { return 0; }
pub resource Owned { i32 secret; cleanup {} }
pub fn i32 Owned::read(ref Owned this) { return this.secret; }
`)
	uri := fileURI(path)
	var notifications bytes.Buffer
	s := server{out: &notifications, workspaceRoots: []string{root}, initialized: true, docs: map[string]*document{}}
	prefix := "module main; import geometry as geo; "
	openText := prefix + "fn i32 read(ref geo::Point point) { return point.column; }"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": openText,
	}})
	doc := s.docs[uri]
	if doc == nil || doc.View == nil || !doc.View.clean {
		t.Fatalf("initial source is not clean: %s", &notifications)
	}
	labels := func(text string) []string {
		t.Helper()
		view, lastClean, wire := doc.View, doc.LastClean, notifications.Len()
		items := s.completion(doc, strings.Index(text, "; }"))
		if doc.View != view || doc.LastClean != lastClean || string(doc.Text) != text || notifications.Len() != wire {
			t.Fatal("speculative completion published or mutated live state")
		}
		got := make([]string, 0, len(items))
		for _, item := range items {
			got = append(got, item.Label)
		}
		return got
	}
	for i, tc := range []struct {
		body string
		want []string
	}{
		{"fn i32 read(ref geo::Point point) { return point.; }", []string{"column", "read", "row"}},
		{"fn i32 read(ref geo::Point point) { return point.ro; }", []string{"row"}},
		{"fn i32 read(ref geo::Owned point) { return point.; }", []string{"read"}},
		{"fn i32 read(i32 point) { return point.; }", nil},
	} {
		text := prefix + tc.body
		send(t, &s, 0, "textDocument/didChange", map[string]any{
			"textDocument":   documentID{URI: uri, Version: i + 2},
			"contentChanges": []map[string]string{{"text": text}},
		})
		if got := labels(text); !slices.Equal(got, tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.body, got, tc.want)
		}
	}
}

func TestMemberCompletionUsesExactLocalReceiver(t *testing.T) {
	const text = "module main; struct Inner { i32 field; } struct Outer { Inner nested; } fn i32 read(ref Outer value) { return value.nested.; }"
	uri := fileURI(filepath.Join(t.TempDir(), "main.fy"))
	s := server{out: io.Discard, initialized: true, docs: map[string]*document{}}
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
	}})
	if got := s.completion(s.docs[uri], strings.LastIndex(text, "; }")); !slices.Equal(got, []Completion{{"field", 5, "field"}}) {
		t.Fatalf("nested member completion = %+v", got)
	}
}

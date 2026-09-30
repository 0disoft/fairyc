package languageserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUTF16PositionsAndLocalURIs(t *testing.T) {
	text := []byte("한😀x\r\nsecond")
	for _, offset := range []int{0, 3, 7, 8, 10, 16} {
		p := bytePosition(text, offset)
		got, ok := positionByte(text, p)
		if !ok || got != offset {
			t.Fatalf("%d -> %+v -> %d %v", offset, p, got, ok)
		}
	}
	if _, ok := positionByte(text, Position{0, 2}); ok {
		t.Fatal("inside surrogate pair accepted")
	}
	if _, ok := positionByte(text, Position{0, 99}); ok {
		t.Fatal("outside line accepted")
	}
	path := filepath.Join(t.TempDir(), "한 space.fy")
	got, err := filePath(fileURI(path))
	if err != nil || got != path {
		t.Fatal(got, err)
	}
	for _, uri := range []string{"https://example.com/a.fy", "file://remote/a.fy", "file:///a.fy?token=bad"} {
		if _, err := filePath(uri); err == nil {
			t.Fatal("nonlocal URI accepted", uri)
		}
	}
}
func editorFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"lsp.test\"\nname = \"lsp-test\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	if err := os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, "src", "main.fy")
	if err := os.WriteFile(filename, []byte("module main; fn i32 main() { return 0; }"), 0644); err != nil {
		t.Fatal(err)
	}
	return root, filename
}
func send(t *testing.T, s *server, id int, method string, p any) {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var rid json.RawMessage
	if id != 0 {
		rid = json.RawMessage(fmt.Sprint(id))
	}
	if err := s.handle(message{JSONRPC: "2.0", ID: rid, Method: method, Params: raw}); err != nil {
		t.Fatal(err)
	}
}
func TestUnsavedDiagnosticsDefinitionCompletionAndVersions(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	var output bytes.Buffer
	s := server{out: &output, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	good := "module main;\nfn i32 helper(i32 value) { return value; }\nfn i32 main() { let i32 local = 2; return helper(local); }\n"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": good}})
	d := s.docs[uri]
	if d == nil || !d.View.clean {
		t.Fatalf("analysis: %+v", d.View.diagnostics)
	}
	use := strings.LastIndex(good, "helper")
	def := s.definition(d, use+1)
	if len(def) != 1 || def[0].URI != uri || def[0].Range.Start.Line != 1 {
		t.Fatalf("definition: %+v", def)
	}
	found := false
	for _, c := range s.completion(d, use+3) {
		if c.Label == "helper" {
			found = true
		}
	}
	if !found {
		t.Fatal("function completion missing")
	}
	found = false
	for _, c := range s.completion(d, strings.LastIndex(good, "local")+2) {
		if c.Label == "local" {
			found = true
		}
	}
	if !found {
		t.Fatal("local completion missing")
	}
	bad := strings.Replace(good, "helper(local)", "helper(false)", 1)
	send(t, &s, 0, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []any{map[string]any{"text": bad}}})
	if len(d.View.diagnostics[filename]) == 0 {
		t.Fatal("unsaved type mismatch not diagnosed")
	}
	send(t, &s, 0, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1}, "contentChanges": []any{map[string]any{"text": good}}})
	if string(d.Text) != bad {
		t.Fatal("stale version applied")
	}
	send(t, &s, 0, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 3}, "contentChanges": []any{map[string]any{"text": good}}})
	if len(d.View.diagnostics[filename]) != 0 {
		t.Fatal("diagnostics not cleared")
	}
	send(t, &s, 0, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 4}, "contentChanges": []any{map[string]any{"text": "module main; fn i32 main() { return"}}})
	diagnostics := d.View.diagnostics[filename]
	if len(diagnostics) == 0 || diagnostics[0].Code == "workspace-analysis-unavailable" {
		t.Fatalf("parser diagnostic location lost: %+v", diagnostics)
	}
	if len(s.definition(d, 10)) != 0 {
		t.Fatal("stale definition published for invalid source")
	}
	disk, _ := os.ReadFile(filename)
	if string(disk) == good || string(disk) == bad {
		t.Fatal("LSP wrote source")
	}
	send(t, &s, 0, "textDocument/didClose", map[string]any{"textDocument": map[string]any{"uri": uri}})
	if len(s.docs) != 0 {
		t.Fatal("document retained")
	}
}
func TestImportedDefinitionAndUnsavedNewModule(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{"rootUri": fileURI(root)})
	other := filepath.Join(root, "src", "answer.fy")
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": fileURI(other), "languageId": "fairyc", "version": 1, "text": "module answer; pub fn i32 value() { return 42; }"}})
	text := "module main; import answer as answer; fn i32 main() { return answer::value(); }"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": text}})
	d := s.docs[uri]
	if !d.View.clean {
		t.Fatalf("diagnostics: %+v", d.View.diagnostics)
	}
	def := s.definition(d, strings.Index(text, "::value")+3)
	if len(def) != 1 || def[0].URI != fileURI(other) {
		t.Fatalf("import definition: %+v", def)
	}
	found := false
	for _, c := range s.completion(d, strings.Index(text, "::value")+5) {
		if c.Label == "value" {
			found = true
		}
	}
	if !found {
		t.Fatal("import completion missing")
	}
}
func TestFramingLifecycleAndBounds(t *testing.T) {
	var in, out, errs bytes.Buffer
	_ = writeMessage(&in, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	_ = writeMessage(&in, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown", "params": nil})
	_ = writeMessage(&in, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	if code := Serve(&in, &out, &errs); code != 0 {
		t.Fatal(code, errs.String())
	}
	r := bufio.NewReader(&out)
	for i := 0; i < 2; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		var length int
		_, _ = fmt.Sscanf(line, "Content-Length: %d", &length)
		_, _ = r.ReadString('\n')
		raw := make([]byte, length)
		if _, err = io.ReadFull(r, raw); err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil || m["jsonrpc"] != "2.0" || m["error"] != nil {
			t.Fatal(string(raw))
		}
	}
	for _, frame := range []string{"Content-Length: 999999999\r\n\r\n", "Content-Length: 2\r\nContent-Length: 2\r\n\r\n{}", "X:" + strings.Repeat("a", 9000) + "\r\n\r\n"} {
		if _, err := readMessage(bufio.NewReaderSize(strings.NewReader(frame), 8192)); err == nil {
			t.Fatal("unbounded frame accepted")
		}
	}
}

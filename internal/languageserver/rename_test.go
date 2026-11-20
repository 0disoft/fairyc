package languageserver

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameVersionedEditsKeepExactBindings(t *testing.T) {
	root, path := editorFixture(t)
	uri := fileURI(path)
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":      fileURI(root),
		"capabilities": map[string]any{"workspace": map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}}},
	})
	if !bytes.Contains(editorResult(t, &out), []byte(`"renameProvider":{"prepareProvider":true}`)) {
		t.Fatal("rename capability missing")
	}
	const text = "module main;\nfn i32 helper(i32 base) { let i32 chosen = 1; if (true) { let i32 nested = 2; /* 😀 */ return chosen + nested; } return chosen + base; }\nfn i32 main() { return helper(. /* label */ base = 3); }"
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 7, "languageId": "fairyc", "text": text,
	}})
	doc := s.docs[uri]
	if doc == nil || doc.View == nil || !doc.View.clean {
		t.Fatalf("fixture did not check cleanly: %s", &out)
	}
	call := func(method, needle, newName string) {
		t.Helper()
		out.Reset()
		send(t, &s, 2, method, map[string]any{
			"textDocument": documentID{URI: uri},
			"position":     bytePosition(doc.Text, strings.Index(text, needle)),
			"newName":      newName,
		})
	}
	call("textDocument/prepareRename", "chosen + nested", "")
	var prep struct {
		Range       Range
		Placeholder string
	}
	if err := json.Unmarshal(editorResult(t, &out), &prep); err != nil || prep.Placeholder != "chosen" ||
		prep.Range.Start != bytePosition(doc.Text, strings.Index(text, "chosen + nested")) {
		t.Fatalf("prepareRename result = %+v (%v)", prep, err)
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		needle, oldName, newName string
		count                    int
	}{
		{"chosen + nested", "chosen", "answer", 3},
		{"helper(. ", "helper", "compute_value", 2},
		{"base = 3", "base", "input", 3},
		{"base) {", "base", "input", 3},
	} {
		call("textDocument/rename", scenario.needle, scenario.newName)
		var result struct{ DocumentChanges []documentEdit }
		if err := json.Unmarshal(editorResult(t, &out), &result); err != nil || len(result.DocumentChanges) != 1 {
			t.Fatalf("rename result = %+v (%v)", result, err)
		}
		edit := result.DocumentChanges[0]
		if edit.TextDocument.URI != uri || edit.TextDocument.Version == nil || *edit.TextDocument.Version != 7 || len(edit.Edits) != scenario.count {
			t.Fatalf("wrong versioned edit = %+v", edit)
		}
		for _, item := range edit.Edits {
			start, ok := positionByte(doc.Text, item.Range.Start)
			end, valid := positionByte(doc.Text, item.Range.End)
			if !ok || !valid || string(doc.Text[start:end]) != scenario.oldName || item.NewText != scenario.newName {
				t.Fatalf("unexpected rename edit %+v", item)
			}
		}
	}
	// This candidate still typechecks, but its first return would resolve
	// to nested's inner declaration instead of the renamed outer binding.
	call("textDocument/rename", "chosen + nested", "nested")
	if !strings.Contains(out.String(), errRenameChangesResolution.Error()) {
		t.Fatalf("capture-changing rename was accepted: %s", &out)
	}
	call("textDocument/prepareRename", "main()", "")
	if got := string(editorResult(t, &out)); got != "null" {
		t.Fatalf("prepareRename allowed entry point: %s", got)
	}
	if current, err := os.ReadFile(path); err != nil || !bytes.Equal(current, disk) || string(doc.Text) != text || doc.Version != 7 {
		t.Fatal("rename changed files or open document state")
	}
}

func TestRenameWorkspaceFunctionAcrossPackages(t *testing.T) {
	root, app, library := siblingWorkspaceFixture(t)
	mainPath := filepath.Join(root, "src", "main.fy")
	appPath := filepath.Join(app, "src", "a.fy")
	libraryPath := filepath.Join(library, "src", "b.fy")
	const appText = "module a; import b as b; pub fn i32 value() { /* 😀 */ return b::value() + 1; }"
	const libraryText = "module b; pub fn i32 value() { return 42; }"
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":      fileURI(root),
		"capabilities": map[string]any{"workspace": map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}}},
	})
	appURI := fileURI(appPath)
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": appURI, "version": 9, "languageId": "fairyc", "text": appText,
	}})
	d := s.docs[appURI]
	if d.View == nil || !d.View.clean {
		t.Fatal("fixture did not check cleanly")
	}
	offset := strings.Index(appText, "b::value()") + len("b::")
	request := map[string]any{
		"textDocument": documentID{URI: appURI},
		"position":     bytePosition(d.Text, offset),
		"newName":      "calculate",
	}
	out.Reset()
	send(t, &s, 2, "textDocument/prepareRename", request)
	var prep struct {
		Range       Range
		Placeholder string
	}
	if err := json.Unmarshal(editorResult(t, &out), &prep); err != nil || prep.Placeholder != "value" ||
		prep.Range != textRange(d.Text, offset, offset+len("value")) {
		t.Fatalf("qualified prepareRename = %+v (%v)", prep, err)
	}

	out.Reset()
	send(t, &s, 3, "textDocument/rename", request)
	raw := editorResult(t, &out)
	if !bytes.Contains(raw, []byte(`"version":null`)) {
		t.Fatalf("closed document needs an explicit null version: %s", raw)
	}
	var result struct{ DocumentChanges []documentEdit }
	if err := json.Unmarshal(raw, &result); err != nil || len(result.DocumentChanges) != 2 {
		t.Fatalf("rename result = %+v (%v)", result, err)
	}
	want := map[string]string{
		appURI:               strings.Replace(appText, "b::value", "b::calculate", 1),
		fileURI(libraryPath): strings.Replace(libraryText, "value", "calculate", 1),
	}
	for _, edit := range result.DocumentChanges {
		version := edit.TextDocument.Version
		text := []byte(libraryText)
		if edit.TextDocument.URI == appURI {
			text = d.Text
			if version == nil || *version != 9 {
				t.Fatalf("wrong open version: %v", version)
			}
		} else if version != nil {
			t.Fatalf("wrong closed version: %v", version)
		}
		if len(edit.Edits) != 1 {
			t.Fatalf("wrong number of edits for %s: %v", edit.TextDocument.URI, edit.Edits)
		}
		item := edit.Edits[0]
		start, ok := positionByte(text, item.Range.Start)
		end, valid := positionByte(text, item.Range.End)
		if !ok || !valid || string(text[start:end]) != "value" ||
			string(text[:start])+item.NewText+string(text[end:]) != want[edit.TextDocument.URI] {
			t.Fatalf("wrong edit for %s: %+v", edit.TextDocument.URI, item)
		}
		delete(want, edit.TextDocument.URI)
	}
	if len(want) != 0 || string(d.Text) != appText || d.Version != 9 || len(s.docs) != 1 {
		t.Fatal("rename missed edits or changed open document state")
	}
	for path, expected := range map[string]string{
		appPath:     "module a; import b as b; pub fn i32 value() { return b::value(); }",
		libraryPath: libraryText,
		mainPath:    "module main; import a as a; fn i32 main() { return a::value(); }",
	} {
		if current, err := os.ReadFile(path); err != nil || string(current) != expected {
			t.Fatalf("rename changed %s on disk: %q (%v)", path, current, err)
		}
	}

	// A disk edit without a watcher event invalidates the original ranges.
	writeProjectRootFile(t, libraryPath, "\n"+libraryText)
	out.Reset()
	send(t, &s, 4, "textDocument/rename", request)
	if !strings.Contains(out.String(), errRenameChangesResolution.Error()) {
		t.Fatalf("rename accepted changed disk inputs: %s", &out)
	}
}

func TestRenameMethodKeepsOwnerAndUpdatesBothCallForms(t *testing.T) {
	root, path := editorFixture(t)
	uri := fileURI(path)
	geometry := filepath.Join(root, "src", "geometry.fy")
	const library = "module geometry;\n" +
		"pub struct Point { i32 value; }\npub struct Other { i32 value; }\n" +
		"pub fn i32 Point::read(ref Point this) { return this.value; }\n" +
		"pub fn i32 Point::taken(ref Point this) { return 1; }\n" +
		"pub fn i32 Other::read(ref Other this) { return this.value; }\n" +
		"pub fn i32 direct(ref Point point) { return Point::read(point); }\n"
	writeProjectRootFile(t, geometry, library)
	const text = "module main; import geometry as geo;\n" +
		"fn i32 inspect(ref geo::Point point, ref geo::Other other) {\n" +
		"return /* 😀 */ point. /* method */ read() + other.read(); }\n"
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":      fileURI(root),
		"capabilities": map[string]any{"workspace": map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}}},
	})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 12, "languageId": "fairyc", "text": text,
	}})
	d := s.docs[uri]
	if d.View == nil || !d.View.clean {
		t.Fatalf("fixture did not check cleanly: %s", &out)
	}
	offset := strings.Index(text, "read()")
	request := map[string]any{
		"textDocument": documentID{URI: uri},
		"position":     bytePosition(d.Text, offset),
		"newName":      "measure",
	}
	out.Reset()
	send(t, &s, 2, "textDocument/prepareRename", request)
	var prep struct {
		Range       Range
		Placeholder string
	}
	if err := json.Unmarshal(editorResult(t, &out), &prep); err != nil || prep.Placeholder != "read" ||
		prep.Range != textRange(d.Text, offset, offset+len("read")) {
		t.Fatalf("method prepareRename = %+v (%v)", prep, err)
	}
	out.Reset()
	send(t, &s, 3, "textDocument/rename", request)
	var result struct{ DocumentChanges []documentEdit }
	if err := json.Unmarshal(editorResult(t, &out), &result); err != nil || len(result.DocumentChanges) != 2 {
		t.Fatalf("rename result = %+v (%v)", result, err)
	}
	want := map[string][]int{
		uri: {offset},
		fileURI(geometry): {
			strings.Index(library, "Point::read") + len("Point::"),
			strings.LastIndex(library, "Point::read") + len("Point::"),
		},
	}
	for _, changes := range result.DocumentChanges {
		expected := want[changes.TextDocument.URI]
		if len(expected) == 0 || len(changes.Edits) != len(expected) {
			t.Fatalf("unexpected edits: %+v", changes)
		}
		data := []byte(library)
		if changes.TextDocument.URI == uri {
			data = d.Text
			if changes.TextDocument.Version == nil || *changes.TextDocument.Version != 12 {
				t.Fatalf("open version: %+v", changes.TextDocument)
			}
		} else if changes.TextDocument.Version != nil {
			t.Fatalf("closed version: %+v", changes.TextDocument)
		}
		for i, edit := range changes.Edits {
			start := expected[i]
			if edit.Range != textRange(data, start, start+len("read")) || edit.NewText != "measure" {
				t.Fatalf("wrong method edit %+v at byte %d", edit, start)
			}
		}
		delete(want, changes.TextDocument.URI)
	}
	out.Reset()
	request["newName"] = "taken"
	send(t, &s, 4, "textDocument/rename", request)
	if !strings.Contains(out.String(), errRenameChangesResolution.Error()) {
		t.Fatalf("accepted conflicting method name: %s", &out)
	}
}

func TestRenameFieldChecksWritesInitializersAndOwner(t *testing.T) {
	root, path := editorFixture(t)
	uri := fileURI(path)
	const prefix = "module main; struct Point { i32 value; i32 other; } struct Other { i32 value; }\n" +
		"fn i32 main() { let i32 value = 40; var Point point = Point{. /* field */ "
	const suffix = ", .other = 0}; let Other other = Other{.value = 2}; " +
		"point.value = point.value + other.value; return point.value; }"
	const text = prefix + "value = value" + suffix
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":      fileURI(root),
		"capabilities": map[string]any{"workspace": map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}}},
	})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
	}})
	d := s.docs[uri]
	if d.View == nil || !d.View.clean {
		t.Fatalf("fixture did not check cleanly: %s", &out)
	}
	request := func(method string, offset int, name string) {
		t.Helper()
		out.Reset()
		send(t, &s, 2, method, map[string]any{
			"textDocument": documentID{URI: uri},
			"position":     bytePosition(d.Text, offset),
			"newName":      name,
		})
	}
	request("textDocument/rename", len(prefix), "coordinate")
	var result struct{ DocumentChanges []documentEdit }
	if err := json.Unmarshal(editorResult(t, &out), &result); err != nil || len(result.DocumentChanges) != 1 {
		t.Fatalf("field rename = %+v (%v)", result, err)
	}
	changes := result.DocumentChanges[0]
	want := []int{
		strings.Index(text, "value; i32"),
		len(prefix),
		strings.Index(text, "point.value =") + len("point."),
		strings.Index(text, "point.value +") + len("point."),
		strings.Index(text, "return point.value") + len("return point."),
	}
	if changes.TextDocument.URI != uri || len(changes.Edits) != len(want) {
		t.Fatalf("field changes = %+v", changes)
	}
	for i, edit := range changes.Edits {
		if edit.NewText != "coordinate" || edit.Range != textRange(d.Text, want[i], want[i]+len("value")) {
			t.Fatalf("unexpected field edit at %d: %+v", want[i], edit)
		}
	}
	request("textDocument/rename", want[0], "other")
	if !strings.Contains(out.String(), errRenameChangesResolution.Error()) {
		t.Fatalf("accepted duplicate field name: %s", &out)
	}

	send(t, &s, 0, "textDocument/didChange", map[string]any{
		"textDocument":   documentID{URI: uri, Version: 2},
		"contentChanges": []map[string]string{{"text": prefix + "value" + suffix}},
	})
	if !d.View.clean {
		t.Fatalf("shorthand fixture did not check cleanly: %s", &out)
	}
	for _, scenario := range []struct {
		from, count     int
		name, shorthand string
	}{
		{want[0], 5, "coordinate", "coordinate = value"},
		{strings.Index(prefix, "value = 40"), 2, "input", "value = input"},
	} {
		request("textDocument/rename", scenario.from, scenario.name)
		var result struct{ DocumentChanges []documentEdit }
		if err := json.Unmarshal(editorResult(t, &out), &result); err != nil || len(result.DocumentChanges) != 1 {
			t.Fatalf("expanded rename = %+v (%v)", result, err)
		}
		edits := result.DocumentChanges[0].Edits
		shared := textRange(d.Text, len(prefix), len(prefix)+len("value"))
		if len(edits) != scenario.count || edits[1] != (textEdit{Range: shared, NewText: scenario.shorthand}) {
			t.Fatalf("wrong shorthand expansion: %+v", edits)
		}
	}
	request("textDocument/prepareRename", len(prefix), "")
	if got := string(editorResult(t, &out)); got != "null" {
		t.Fatalf("picked one of two bindings at the shared token: %s", got)
	}
}

func TestRenameEnumMembersUsesCheckedVariantAndPayload(t *testing.T) {
	root, path := editorFixture(t)
	uri := fileURI(path)
	const text = "module main;\n" +
		"enum Event { Empty; Pair { i32 value; }; Other { i32 value; }; }\n" +
		"fn Event make() { return .Pair{.value = 42}; }\n" +
		"fn Event empty() { return .Empty; }\n" +
		"fn i32 main() { return switch (make()) {\n" +
		"case .Empty: { 0 }\ncase .Pair{. /* field */ value = item}: { item }\n" +
		"case .Other{.value = item}: { item }\n}; }\n"
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":      fileURI(root),
		"capabilities": map[string]any{"workspace": map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}}},
	})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
	}})
	d := s.docs[uri]
	if d.View == nil || !d.View.clean {
		t.Fatalf("fixture did not check cleanly: %s", &out)
	}
	for _, scenario := range []struct {
		name, replacement string
		want              []int
	}{
		{"Pair", "Payload", []int{strings.Index(text, "Pair"), strings.Index(text, ".Pair") + 1, strings.LastIndex(text, ".Pair") + 1}},
		{"Empty", "None", []int{strings.Index(text, "Empty"), strings.Index(text, ".Empty") + 1, strings.LastIndex(text, ".Empty") + 1}},
		{"value", "amount", []int{strings.Index(text, "value"), strings.Index(text, ".value") + 1, strings.Index(text, "value = item")}},
	} {
		out.Reset()
		send(t, &s, 2, "textDocument/rename", map[string]any{
			"textDocument": documentID{URI: uri},
			"position":     bytePosition(d.Text, scenario.want[2]),
			"newName":      scenario.replacement,
		})
		var result struct{ DocumentChanges []documentEdit }
		if err := json.Unmarshal(editorResult(t, &out), &result); err != nil || len(result.DocumentChanges) != 1 {
			t.Fatalf("%s rename = %+v (%v)", scenario.name, result, err)
		}
		edits := result.DocumentChanges[0].Edits
		if len(edits) != len(scenario.want) {
			t.Fatalf("%s edits = %+v", scenario.name, edits)
		}
		for i, edit := range edits {
			start := scenario.want[i]
			if edit.NewText != scenario.replacement || edit.Range != textRange(d.Text, start, start+len(scenario.name)) {
				t.Fatalf("%s edit %d = %+v", scenario.name, i, edit)
			}
		}
	}
}

func TestRenameMethodParameterPreservesReceiver(t *testing.T) {
	root, path := editorFixture(t)
	uri := fileURI(path)
	const text = "module main; struct Point { i32 value; }\n" +
		"fn i32 Point::add(ref Point this, i32 input) { return this.value + input; }\n" +
		"fn i32 Point::fixed(i32 factor) { return factor; }\n"
	var out bytes.Buffer
	s := server{out: &out, docs: map[string]*document{}}
	send(t, &s, 1, "initialize", map[string]any{
		"rootUri":      fileURI(root),
		"capabilities": map[string]any{"workspace": map[string]any{"workspaceEdit": map[string]bool{"documentChanges": true}}},
	})
	send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "version": 1, "languageId": "fairyc", "text": text,
	}})
	d := s.docs[uri]
	if d.View == nil || !d.View.clean {
		t.Fatalf("fixture did not check cleanly: %s", &out)
	}
	request := func(method, needle, name string) {
		t.Helper()
		out.Reset()
		send(t, &s, 2, method, map[string]any{
			"textDocument": documentID{URI: uri},
			"position":     bytePosition(d.Text, strings.Index(text, needle)),
			"newName":      name,
		})
	}
	for _, old := range []string{"input", "factor"} {
		request("textDocument/rename", old+";", "amount")
		var result struct{ DocumentChanges []documentEdit }
		if err := json.Unmarshal(editorResult(t, &out), &result); err != nil || len(result.DocumentChanges) != 1 {
			t.Fatalf("%s rename = %+v (%v)", old, result, err)
		}
		edits := result.DocumentChanges[0].Edits
		if len(edits) != 2 {
			t.Fatalf("%s edits = %+v", old, edits)
		}
		for i, start := range []int{strings.Index(text, old), strings.LastIndex(text, old)} {
			if want := (textEdit{Range: textRange(d.Text, start, start+len(old)), NewText: "amount"}); edits[i] != want {
				t.Fatalf("%s edit %d = %+v, want %+v", old, i, edits[i], want)
			}
		}
	}
	request("textDocument/prepareRename", "this.value", "")
	if got := string(editorResult(t, &out)); got != "null" {
		t.Fatalf("receiver was renameable: %s", got)
	}
	request("textDocument/rename", "factor;", "this")
	if !strings.Contains(out.String(), "this is reserved for the method receiver") {
		t.Fatalf("accepted a new receiver name: %s", &out)
	}
}

package languageserver

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func TestEmbeddedSDKDefinitionAndSourceRetrieval(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		root, filename := editorFixture(t)
		uri := fileURI(filename)
		var out bytes.Buffer
		s := server{out: &out, docs: map[string]*document{}}
		send(t, &s, 1, "initialize", map[string]any{
			"rootUri":               fileURI(root),
			"initializationOptions": map[string]bool{"fairycSdkSourceNavigation": enabled},
		})
		const input = `module main; import std::bytes as bytes;
fn bool same() { return bytes::equal(b"x", b"x"); }
`
		send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": uri, "version": 1, "languageId": "fairyc", "text": input,
		}})
		if d := s.docs[uri]; d == nil || d.View == nil || !d.View.clean {
			t.Fatalf("SDK fixture is not clean: %s", &out)
		}
		out.Reset()
		send(t, &s, 2, "textDocument/definition", map[string]any{
			"textDocument": documentID{URI: uri},
			"position":     bytePosition([]byte(input), strings.Index(input, "bytes::equal")+len("bytes::")),
		})
		var locations []Location
		targetCount := 0
		if enabled {
			targetCount = 1
		}
		if err := json.Unmarshal(editorResult(t, &out), &locations); err != nil || len(locations) != targetCount {
			t.Fatalf("SDK locations with opt-in %v: %+v (%v)", enabled, locations, err)
		}
		if !enabled {
			continue
		}
		loc := locations[0]
		exact, err := sdkstdlib.ReadSource("std/bytes.fy")
		if err != nil || loc.URI != sdkSourceURI("std/bytes.fy", exact) {
			t.Fatalf("incorrect SDK URI %+v (%v)", loc, err)
		}
		out.Reset()
		send(t, &s, 3, "fairyc/sdkSource", map[string]string{"uri": loc.URI})
		var received string
		if err := json.Unmarshal(editorResult(t, &out), &received); err != nil || received != string(exact) {
			t.Fatalf("source request did not return the compiler's exact bytes (%v)", err)
		}
		start, a := positionByte(exact, loc.Range.Start)
		end, b := positionByte(exact, loc.Range.End)
		if !a || !b || start > end || string(exact[start:end]) != "equal" {
			t.Fatalf("SDK definition range does not identify equal: %+v", loc.Range)
		}
		for _, rejected := range []string{
			sdkSourcePrefix + strings.Repeat("0", 64) + "/std/bytes.fy",
			loc.URI + "?extra=1",
			strings.Replace(loc.URI, "/std/", "/std/../std/", 1),
			uri,
		} {
			out.Reset()
			send(t, &s, 4, "fairyc/sdkSource", map[string]string{"uri": rejected})
			if !strings.Contains(out.String(), `"code":-32602`) {
				t.Fatalf("invalid SDK URI accepted: %q", rejected)
			}
		}
	}
}

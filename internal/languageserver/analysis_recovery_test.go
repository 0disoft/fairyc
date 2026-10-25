package languageserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestQueryRecoversAnalysisAfterRejectedSupersedingChange(t *testing.T) {
	for _, rejected := range []string{"stale-version", "invalid-parameters"} {
		for _, method := range []string{"textDocument/definition", "textDocument/completion"} {
			t.Run(rejected+"/"+method, func(t *testing.T) {
				root, filename := editorFixture(t)
				uri := fileURI(filename)
				var output bytes.Buffer
				s := server{out: &output, root: root, initialized: true, docs: map[string]*document{}}
				before := "module main;\nfn i32 before() { return 1; }\nfn i32 main() { return before(); }\n"
				after := "// shifted\n" + strings.ReplaceAll(before, "before", "after_value")
				send(t, &s, 0, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": before}})
				old := s.docs[uri].View
				if old == nil || !old.clean {
					t.Fatal("initial analysis is not clean")
				}

				control := newRequestControl()
				defer control.abort()
				accepted, first, err := control.start(context.Background(), message{Method: "textDocument/didChange"})
				if err != nil {
					t.Fatal(err)
				}
				// Admission cancels the older analysis before the handler validates
				// the superseding notification's version or parameters.
				latest, second, err := control.start(context.Background(), message{Method: "textDocument/didChange"})
				if err != nil {
					t.Fatal(err)
				}
				change, err := json.Marshal(map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []any{map[string]any{"text": after}}})
				if err != nil {
					t.Fatal(err)
				}
				s.ctx = accepted
				if err := s.handle(message{Method: "textDocument/didChange", Params: change}); !errors.Is(err, context.Canceled) {
					t.Fatalf("superseded recheck: %v", err)
				}
				control.finish(first)
				s.ctx = latest
				if rejected == "stale-version" {
					send(t, &s, 0, "textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1}, "contentChanges": []any{map[string]any{"text": before}}})
				} else if err := s.handle(message{Method: "textDocument/didChange", Params: json.RawMessage(`[]`)}); err != nil {
					t.Fatal(err)
				}
				control.finish(second)
				if string(s.docs[uri].Text) != after || s.docs[uri].Version != 2 || s.docs[uri].View != old {
					t.Fatal("rejected change replaced the accepted text or published a canceled analysis")
				}

				position := bytePosition([]byte(after), strings.LastIndex(after, "after_value")+3)
				query := map[string]any{"textDocument": map[string]any{"uri": uri}, "position": position}
				raw, err := json.Marshal(query)
				if err != nil {
					t.Fatal(err)
				}
				canceled, stop := context.WithCancel(context.Background())
				stop()
				s.ctx = canceled
				if err := s.handle(message{ID: json.RawMessage(`1`), Method: method, Params: raw}); !errors.Is(err, context.Canceled) {
					t.Fatalf("canceled recovery query: %v", err)
				}
				s.ctx = context.Background()
				send(t, &s, 1, method, query)
				wire := output.Bytes()
				var response struct {
					Result json.RawMessage `json:"result"`
					Error  any             `json:"error"`
				}
				if err := json.Unmarshal(wire[bytes.LastIndex(wire, []byte("\r\n\r\n"))+4:], &response); err != nil || response.Error != nil {
					t.Fatalf("query response: %s (%v)", wire, err)
				}
				if method == "textDocument/definition" {
					var locations []Location
					if err := json.Unmarshal(response.Result, &locations); err != nil || len(locations) != 1 || locations[0].URI != uri || locations[0].Range.Start.Line != 2 {
						t.Fatalf("definition did not use version 2: %s (%v)", response.Result, err)
					}
				} else {
					var items []Completion
					if err := json.Unmarshal(response.Result, &items); err != nil || len(items) != 1 || items[0].Label != "after_value" {
						t.Fatalf("completion did not use version 2: %s (%v)", response.Result, err)
					}
				}
				view := s.docs[uri].View
				if view == nil || !view.clean || view == old {
					t.Fatal("query did not recover current analysis")
				}
				send(t, &s, 2, method, query)
				if s.docs[uri].View != view {
					t.Fatal("unchanged query repeated completed analysis")
				}
			})
		}
	}
}

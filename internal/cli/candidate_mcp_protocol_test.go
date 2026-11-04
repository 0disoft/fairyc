package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmcp"
)

const mcpInitialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
const mcpReady = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
const mcpCall = `{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"atlas_impact","arguments":{"packageUid":"example.app","symbol":"selected"}}}`

// This probe executes only the pure authored protocol core. A test-only host
// supplies a fixed impact document; production current-source wiring is separate.
const mcpProtocolMain = `module main;import project_mcp_protocol as mcp;` + atlasReadInput + `
@effects(console) fn i32 main(){let data=input();let source=bytes::view(ref data);var state=mcp::empty();var usize start=0;var usize at=0;let usize maximum=if(process::arg_count()>1){2048}else{4096};
while(at<source.len){if(source[at]!=10 && at+1<source.len){at+=1;continue;}let end=if(source[at]==10){at}else{at+1};let stop=if(end>start && source[end-1]==13){end-1}else{end};var bytes::Buffer frame=bytes::empty();if(!bytes::append(ref mut frame,source[start..stop]).is_ok){return 2;}let p=switch(mcp::step(ref mut state,frame,maximum)){case .ok(p):{p}case .err(_):{return 2;}};
if(mcp::pending(ref p)){let invalid=bytes::equal(mcp::uid(ref p),b"invalid.host");let available=invalid || bytes::equal(mcp::uid(ref p),b"example.app") && bytes::equal(mcp::symbol(ref p),b"selected");let result=switch(mcp::finish(ref p,if(invalid){b"{"}else{b"{\"state\":\"bounded\",\"omissions\":[\"external\"]}"},available,maximum)){case .ok(p):{p}case .err(_):{return 2;}};discard io::write_bytes(bytes::view(ref result));}else{discard io::write_bytes(mcp::packet(ref p));}at+=1;start=at;}return 0;}`

func TestCandidateMCPProtocol(t *testing.T) {
	join := func(frames ...string) string { return strings.Join(frames, "\n") + "\n" }
	ping := `{"jsonrpc":"2.0","id":2,"method":"ping"}`
	type fixture struct{ name, input string }
	cases := []fixture{
		{"empty", ""}, {"empty-frame", "\n"}, {"crlf-and-final-eof", mcpInitialize + "\r\n" + mcpReady + "\r\n" + mcpCall},
		{"lifecycle", join(`{"jsonrpc":"2.0","id":0,"method":"tools/list"}`, ping, mcpInitialize, mcpReady, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, mcpCall, mcpInitialize)},
		{"notifications", join(mcpReady, `{"jsonrpc":"2.0","method":"tools/call"}`, mcpInitialize, `{"jsonrpc":"2.0","method":"notifications/initialized","params":null}`, mcpCall, mcpReady, mcpCall)},
		{"selection-failure", join(mcpInitialize, mcpReady, strings.Replace(mcpCall, "example.app", "other.app", 1))},
		{"invalid-host-document", join(mcpInitialize, mcpReady, strings.Replace(mcpCall, "example.app", "invalid.host", 1))},
		{"method-and-policy", join(mcpInitialize, mcpReady, strings.Replace(mcpCall, "atlas_impact", "atlas_edit", 1), strings.Replace(mcpCall, `"symbol":"selected"`, `"symbol":"selected","root":"elsewhere"`, 1), `{"jsonrpc":"2.0","id":3,"method":"unknown"}`, strings.Replace(mcpCall, `"selected"`, `1`, 1))},
		{"invalid-json", join("{", "[]", "null", `{ "jsonrpc":"2.0", "id":1, "method":"ping", "method":"tools/call" }`, `{"jsonrpc":"2.0","id":1,"method":"ping","\u0069d":2}`, ping+ping)},
		{"invalid-id", join(`{"jsonrpc":"2.0","id":null,"method":"ping"}`, `{"jsonrpc":"2.0","id":false,"method":"ping"}`, `{"jsonrpc":"2.0","id":{},"method":"ping"}`, `{"jsonrpc":"2.0","id":1.0,"method":"ping"}`, `{"jsonrpc":"2.0","id":1e0,"method":"ping"}`, `{"jsonrpc":"2.0","id":-0,"method":"ping"}`)},
		{"id-byte-limit", join(strings.Replace(ping, `"id":2`, `"id":"`+strings.Repeat("x", 510)+`"`, 1), strings.Replace(ping, `"id":2`, `"id":"`+strings.Repeat("x", 511)+`"`, 1), strings.Replace(ping, `"id":2`, `"id":12345678901234567890`, 1), strings.Replace(ping, `"id":2`, `"id":123456789012345678901`, 1))},
		{"html-id-output-limit", join(strings.Replace(ping, `"id":2`, `"id":"`+strings.Repeat("<", 510)+`"`, 1))},
		{"nested-boundary", join(strings.TrimSuffix(ping, "}")+`,"extra":`+strings.Repeat("[", 15)+"0"+strings.Repeat("]", 15)+"}", strings.TrimSuffix(ping, "}")+`,"extra":`+strings.Repeat("[", 16)+"0"+strings.Repeat("]", 16)+"}")},
		{"case-insensitive-struct-fields", join(strings.ReplaceAll(mcpInitialize, `"jsonrpc"`, `"JSONRPC"`), mcpReady, strings.ReplaceAll(mcpCall, `"name"`, `"NAME"`), strings.ReplaceAll(mcpCall, `"packageUid"`, `"PACKAGEUID"`))},
		{"initialization-and-params", join(strings.Replace(mcpInitialize, `"capabilities":{}`, `"capabilities":null`, 1), strings.Replace(mcpInitialize, "2025-11-25", "2099-01-01", 1), mcpReady, `{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{"_meta":{}}}`, `{"jsonrpc":"2.0","id":5,"method":"tools/list","params":null}`, `{"jsonrpc":"2.0","id":6,"method":"tools/list","params":{"cursor":"no"}}`)},
	}
	const prefix = `{"jsonrpc":"2.0","id":9,"method":"ping","extra":"`
	for _, size := range []int{65535, 65536, 65537} {
		cases = append(cases, fixture{fmt.Sprintf("frame-%d", size), prefix + strings.Repeat("x", size-len(prefix)-2) + `"}` + "\n"})
	}
	candidateAtlasProbeSize(t, "project_mcp_protocol", mcpProtocolMain, 128<<10, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var want bytes.Buffer
				err := atlasmcp.Serve(strings.NewReader(c.input), &want, 1024, 2048, func(uid, symbol string) (json.RawMessage, error) {
					if uid == "invalid.host" {
						return json.RawMessage("{"), nil
					}
					if uid != "example.app" || symbol != "selected" {
						return nil, fmt.Errorf("unavailable")
					}
					return json.RawMessage(`{"state":"bounded","omissions":["external"]}`), nil
				})
				wantCode := 0
				if err != nil {
					wantCode = 2
				}
				code, got := candidateAtlasProcess(t, image, []byte(c.input), "tight")
				if code != wantCode || !reflect.DeepEqual(mcpDocuments(t, got), mcpDocuments(t, want.String())) {
					t.Fatalf("code=%d want=%d\nGot: %s\nWant: %s", code, wantCode, got, &want)
				}
				for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
					if len(line) > 2048 {
						t.Fatalf("response grew past allowance: %d", len(line))
					}
				}
			})
		}
	})
}

func mcpDocuments(t *testing.T, data string) []any {
	t.Helper()
	var values []any
	for _, line := range strings.Split(strings.TrimSuffix(data, "\n"), "\n") {
		if line == "" {
			continue
		}
		var value any
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("invalid response %q: %v", line, err)
		}
		values = append(values, value)
	}
	return values
}

package atlasmcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMCPSessionTransitionAndErrorOrder(t *testing.T) {
	const call = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"atlas_impact","arguments":{"packageUid":"example.app","symbol":"selected"}}}`
	const badCall = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{}}`
	const badList = `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"cursor":"next"}}`
	const badInit = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	const badReady = `{"jsonrpc":"2.0","method":"notifications/initialized","params":{"unexpected":true}}`
	notReady := &rpcError{-32002, "session not initialized"}
	invalidInit := &rpcError{-32602, "invalid or unsupported initialization"}
	invalidTool := &rpcError{-32602, "unknown tool or invalid arguments"}
	invalidStrings := &rpcError{-32602, "exact packageUid and symbol strings required"}
	type replyExpectation struct {
		id      string
		failure *rpcError
	}
	for _, test := range []struct {
		name     string
		messages []string
		replies  []replyExpectation
		calls    int
	}{
		{"early ready notification", []string{ready, badCall}, []replyExpectation{{"2", notReady}}, 0},
		{"initialize notification grants no authority", []string{strings.Replace(initialize, `"id":1,`, "", 1), ready, call}, []replyExpectation{{"2", notReady}}, 0},
		{"failed initialization grants no authority", []string{badInit, ready, badCall}, []replyExpectation{{"1", invalidInit}, {"2", notReady}}, 0},
		{"readiness before call parameters", []string{initialize, badCall}, []replyExpectation{{"1", nil}, {"2", notReady}}, 0},
		{"readiness before list parameters", []string{initialize, badList}, []replyExpectation{{"1", nil}, {"2", notReady}}, 0},
		{"invalid ready then recovery", []string{initialize, badReady, badCall, ready, call}, []replyExpectation{{"1", nil}, {"2", notReady}, {"2", nil}}, 1},
		{"reinitialization retains readiness", []string{initialize, ready, initialize, call}, []replyExpectation{{"1", nil}, {"1", invalidInit}, {"2", nil}}, 1},
		{"tool notification is silent", []string{initialize, ready, strings.Replace(call, `"id":2,`, "", 1), call}, []replyExpectation{{"1", nil}, {"2", nil}}, 1},
		{"list validation after readiness", []string{initialize, ready, badList}, []replyExpectation{{"1", nil}, {"2", &rpcError{-32602, "pagination and other list parameters are unsupported"}}}, 0},
		{"tool envelope then scalar arguments", []string{initialize, ready, badCall, strings.Replace(call, `"selected"`, "7", 1), call}, []replyExpectation{{"1", nil}, {"2", invalidTool}, {"2", invalidStrings}, {"2", nil}}, 1},
		{"initialization retry", []string{badInit, initialize, ready, call}, []replyExpectation{{"1", invalidInit}, {"1", nil}, {"2", nil}}, 1},
		{"notification metadata", []string{initialize, `{"jsonrpc":"2.0","method":"notifications/initialized","params":{"_meta":{"trace":"id"}}}`, call}, []replyExpectation{{"1", nil}, {"2", nil}}, 1},
		{"invalid notification retains readiness", []string{initialize, ready, badReady, call}, []replyExpectation{{"1", nil}, {"2", nil}}, 1},
		{"duplicate JSON key before authority", []string{`{"jsonrpc":"2.0","id":2,"method":"tools/call","method":"ping"}`}, []replyExpectation{{"null", &rpcError{-32700, "invalid JSON message"}}}, 0},
		{"invalid envelope before authority", []string{`{"jsonrpc":"1.0","id":2,"method":"tools/call"}`}, []replyExpectation{{"null", &rpcError{-32600, "invalid JSON-RPC request"}}}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(strings.Join(test.messages, "\n"))
			before := bytes.Clone(input)
			hostResult := json.RawMessage(`{"state":"bounded"}`)
			var output bytes.Buffer
			calls := 0
			err := Serve(bytes.NewReader(input), &output, len(test.messages), 4096, func(uid, symbol string) (json.RawMessage, error) {
				calls++
				if uid != "example.app" || symbol != "selected" {
					t.Fatalf("changed arguments: %q %q", uid, symbol)
				}
				return hostResult, nil
			})
			if err != nil || calls != test.calls {
				t.Fatalf("err=%v calls=%d want=%d", err, calls, test.calls)
			}
			if !bytes.Equal(input, before) || string(hostResult) != `{"state":"bounded"}` {
				t.Fatal("caller-owned request or host result was modified")
			}
			lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
			if len(lines) != len(test.replies) {
				t.Fatalf("replies=%d want=%d: %s", len(lines), len(test.replies), output.String())
			}
			for i, want := range test.replies {
				var got response
				if err := json.Unmarshal(lines[i], &got); err != nil {
					t.Fatal(err)
				}
				if got.Version != "2.0" || string(got.ID) != want.id || !reflect.DeepEqual(got.Error, want.failure) || (got.Error == nil) != (got.Result != nil) {
					t.Fatalf("reply[%d]=%s, want id=%s error=%+v", i, lines[i], want.id, want.failure)
				}
			}
		})
	}
}

func TestMCPSessionHostFailureSemantics(t *testing.T) {
	const call = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"atlas_impact","arguments":{"packageUid":"example.app","symbol":"selected"}}}`
	for _, invalidJSON := range []bool{false, true} {
		var output bytes.Buffer
		err := Serve(strings.NewReader(strings.Join([]string{initialize, ready, call, `{"jsonrpc":"2.0","id":3,"method":"ping"}`}, "\n")), &output, 4, 4096, func(string, string) (json.RawMessage, error) {
			if invalidJSON {
				return json.RawMessage("{"), nil
			}
			return nil, errors.New("private host error")
		})
		if invalidJSON {
			if err == nil || err.Error() != "MCP host returned invalid JSON" || bytes.Count(output.Bytes(), []byte("\n")) != 1 {
				t.Fatalf("invalid host JSON: err=%v output=%s", err, output.String())
			}
			continue
		}
		if err != nil || bytes.Count(output.Bytes(), []byte("\n")) != 3 || !strings.Contains(output.String(), `"isError":true`) || strings.Contains(output.String(), "private host error") {
			t.Fatalf("host query failure: err=%v output=%s", err, output.String())
		}
	}
}

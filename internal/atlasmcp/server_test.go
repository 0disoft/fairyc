package atlasmcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test-client","version":"1"}}}`
const ready = `{"jsonrpc":"2.0","method":"notifications/initialized"}`

func TestMCPReadOnlyLifecycleAndBounds(t *testing.T) {
	count := 0
	handler := func(uid, symbol string) (json.RawMessage, error) {
		count++
		if uid != "example.app" || symbol != "selected" {
			t.Fatal("unexpected arguments")
		}
		return json.RawMessage(`{"state":"bounded","omissions":["external"]}`), nil
	}
	messages := []string{
		`{"jsonrpc":"2.0","id":0,"method":"tools/list"}`,
		initialize, ready,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"atlas_impact","arguments":{"packageUid":"example.app","symbol":"selected"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"atlas_edit","arguments":{"packageUid":"example.app","symbol":"selected"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"atlas_impact","arguments":{"packageUid":"example.app","symbol":"selected","root":"outside"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/list","method":"ping"}`,
	}
	var out bytes.Buffer
	if err := Serve(strings.NewReader(strings.Join(messages, "\n")), &out, 16, 4096, handler); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if count != 1 || len(lines) != 7 {
		t.Fatalf("calls=%d responses=%d", count, len(lines))
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatal(line)
		}
	}
	if !strings.Contains(lines[0], "not initialized") || !strings.Contains(lines[3], `"structuredContent"`) || !strings.Contains(lines[6], `"code":-32700`) {
		t.Fatal(out.String())
	}
	if strings.Contains(lines[2], "atlas_edit") || strings.Contains(lines[2], "shell") {
		t.Fatal("write capability advertised")
	}
	if err := Serve(strings.NewReader(strings.Repeat("x", MaxInput+2)), &bytes.Buffer{}, 1, 4096, handler); err == nil {
		t.Fatal("oversized frame accepted")
	}
	if err := Serve(strings.NewReader(initialize+"\n"+ready), &bytes.Buffer{}, 1, 4096, handler); err == nil {
		t.Fatal("message budget ignored")
	}
}

func TestMCPVersionNegotiationAndOversizedID(t *testing.T) {
	var out bytes.Buffer
	input := strings.Replace(initialize, "2025-11-25", "2099-01-01", 1) + "\n" + `{"jsonrpc":"2.0","id":"` + strings.Repeat("a", 600) + `","method":"ping"}`
	if err := Serve(strings.NewReader(input), &out, 2, 2048, func(string, string) (json.RawMessage, error) { t.Fatal("unexpected tool call"); return nil, nil }); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if !strings.Contains(lines[0], ProtocolVersion) || !strings.Contains(lines[1], `"id":null`) || len(lines[1]) > 2048 {
		t.Fatal(out.String())
	}
}

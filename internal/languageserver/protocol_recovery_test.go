package languageserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

func TestServeRecoversAfterMalformedMessageBodies(t *testing.T) {
	var wire, output, stderr bytes.Buffer
	bodies := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0",`,
		"{\"method\":\"\xff\"}",
		`[]`,
		`{"jsonrpc":"2.0","id":1.5,"method":"exit"}`,
		`{"jsonrpc":"2.0","id":2,"method":23}`,
		`{"jsonrpc":"2.0","id":"last","method":"shutdown"}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
	}
	for _, body := range bodies {
		fmt.Fprintf(&wire, "Content-Length: %d\r\n\r\n%s", len(body), body)
	}
	if code := Serve(&wire, &output, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("session exit %d, stderr %q", code, &stderr)
	}
	wireOut := output.Bytes()
	for i, wantCode := range []int{0, -32700, -32700, -32600, -32600, -32600, 0} {
		header, remaining, found := bytes.Cut(wireOut, []byte("\r\n\r\n"))
		if !found || !bytes.HasPrefix(header, []byte("Content-Length: ")) {
			t.Fatalf("missing response frame %d", i)
		}
		size, err := strconv.Atoi(string(header[len("Content-Length: "):]))
		if err != nil || size < 0 || size > len(remaining) {
			t.Fatalf("invalid response length %q", header)
		}
		var response struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Result  json.RawMessage `json:"result"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		body := remaining[:size]
		if err := json.Unmarshal(body, &response); err != nil || response.JSONRPC != "2.0" {
			t.Fatalf("invalid response %s (%v)", body, err)
		}
		switch {
		case i == 0:
			if string(response.ID) != "1" || response.Error != nil || len(response.Result) == 0 {
				t.Fatalf("initialize response: %s", body)
			}
		case wantCode != 0:
			if string(response.ID) != "null" || response.Error == nil || response.Error.Code != wantCode ||
				response.Error.Message == "" || response.Result != nil {
				t.Fatalf("expected fault %d with null ID: %s", wantCode, body)
			}
		default:
			if string(response.ID) != `"last"` || response.Error != nil || string(response.Result) != "null" {
				t.Fatalf("shutdown response: %s", body)
			}
		}
		wireOut = remaining[size:]
	}
	if len(wireOut) != 0 {
		t.Fatalf("unexpected remaining frames: %q", wireOut)
	}
}

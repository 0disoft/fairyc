package languageserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestProtocolCancellationPassesQueuedRequest(t *testing.T) {
	var wire bytes.Buffer
	for _, value := range []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "textDocument/definition"},
		map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/completion"},
		map[string]any{"jsonrpc": "2.0", "method": "$/cancelRequest", "params": map[string]any{"id": 1}},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
	} {
		if err := writeMessage(&wire, value); err != nil {
			t.Fatal(err)
		}
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	control := newRequestControl()
	incoming := make(chan incomingMessage)
	finished := make(chan struct{})
	go func() {
		readMessages(ctx, bufio.NewReader(&wire), control, incoming)
		close(finished)
	}()
	defer func() {
		stop()
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Error("reader did not stop")
		}
	}()
	first := <-incoming
	select {
	case <-first.ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("queued completion prevented cancellation of the active request")
	}
	control.finish(first.token)
	for _, method := range []string{"textDocument/completion", "exit"} {
		next := <-incoming
		if next.message.Method != method {
			t.Fatalf("request order changed: got %s, want %s", next.message.Method, method)
		}
		if method == "textDocument/completion" && next.ctx.Err() != nil {
			t.Fatal("unrelated queued completion was canceled")
		}
		control.finish(next.token)
	}
}

func TestPendingRequestsEnforceCountAndReleaseCapacity(t *testing.T) {
	control := newRequestControl()
	defer control.abort()
	var first uint64
	for n := range maxPendingRequests {
		_, token, err := control.start(context.Background(), message{Method: "textDocument/completion"})
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			first = token
		}
	}
	if _, _, err := control.start(context.Background(), message{Method: "textDocument/completion"}); !errors.Is(err, errPendingRequestLimit) {
		t.Fatalf("request count limit: %v", err)
	}
	control.finish(first)
	if _, _, err := control.start(context.Background(), message{Method: "textDocument/completion"}); err != nil {
		t.Fatalf("completed request did not release capacity: %v", err)
	}
	control.abort()
	if len(control.pending) != 0 || control.bytes != 0 {
		t.Fatal("abort retained pending requests or byte charges")
	}
}

func TestPendingRequestsEnforceByteLimit(t *testing.T) {
	control := newRequestControl()
	defer control.abort()
	m := message{Method: "textDocument/didChange", Params: json.RawMessage(`"` + strings.Repeat("x", maxPendingRequestBytes/4) + `"`)}
	var first uint64
	for n := range 3 {
		_, token, err := control.start(context.Background(), m)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			first = token
		}
	}
	if _, _, err := control.start(context.Background(), m); !errors.Is(err, errPendingRequestLimit) {
		t.Fatalf("pending byte limit: %v", err)
	}
	control.finish(first)
	if _, _, err := control.start(context.Background(), m); err != nil {
		t.Fatalf("completed request did not release byte capacity: %v", err)
	}
}

func TestQueuedQueryPreservesItsPriorAnalysis(t *testing.T) {
	control := newRequestControl()
	defer control.abort()
	prior, priorToken, err := control.start(context.Background(), message{Method: "textDocument/didOpen"})
	if err != nil {
		t.Fatal(err)
	}
	_, queryToken, err := control.start(context.Background(), message{ID: json.RawMessage("1"), Method: "textDocument/definition"})
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := control.start(context.Background(), message{Method: "textDocument/didChange"})
	if err != nil || prior.Err() != nil {
		t.Fatalf("queued query lost its prior analysis: %v %v", err, prior.Err())
	}
	control.finish(priorToken)
	control.finish(queryToken)
	if _, _, err := control.start(context.Background(), message{Method: "textDocument/didChange"}); err != nil || next.Err() == nil {
		t.Fatalf("obsolete analysis was not superseded after the query: %v", err)
	}
}

func TestProtocolQueueOverflowCancelsWorkAndReportsFailure(t *testing.T) {
	var wire bytes.Buffer
	for n := range maxPendingRequests + 1 {
		if err := writeMessage(&wire, map[string]any{"jsonrpc": "2.0", "id": n, "method": "textDocument/completion"}); err != nil {
			t.Fatal(err)
		}
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	control := newRequestControl()
	incoming := make(chan incomingMessage)
	go readMessages(ctx, bufio.NewReader(&wire), control, incoming)
	first := <-incoming
	select {
	case <-first.ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("queue overflow did not stop active work")
	}
	for {
		select {
		case next, open := <-incoming:
			if !open {
				t.Fatal("queue overflow was silently discarded")
			}
			if next.err != nil {
				if !errors.Is(next.err, errPendingRequestLimit) {
					t.Fatalf("overflow error: %v", next.err)
				}
				return
			}
			control.finish(next.token)
		case <-time.After(2 * time.Second):
			t.Fatal("queue did not report overflow")
		}
	}
}

func TestProtocolQueryBeforeEditKeepsItsAnalysis(t *testing.T) {
	root, filename := editorFixture(t)
	uri := fileURI(filename)
	text := "module main; fn i32 helper(){return 1;} fn i32 main(){return helper();}"
	var wire bytes.Buffer
	for _, value := range []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"rootUri": fileURI(root)}},
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "fairyc", "version": 1, "text": text}}},
		map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": bytePosition([]byte(text), strings.LastIndex(text, "helper"))}},
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []any{map[string]any{"text": "module main; fn i32 main(){return 0;}"}}}},
		map[string]any{"jsonrpc": "2.0", "id": 3, "method": "shutdown"},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
	} {
		if err := writeMessage(&wire, value); err != nil {
			t.Fatal(err)
		}
	}
	var output, diagnostics bytes.Buffer
	if code := Serve(&wire, &output, &diagnostics); code != 0 {
		t.Fatalf("protocol session: %d %s", code, diagnostics.String())
	}
	reader := bufio.NewReader(&output)
	for {
		header, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("definition response missing: %v", err)
		}
		var size int
		if _, err := fmt.Sscanf(header, "Content-Length: %d", &size); err != nil {
			t.Fatal(err)
		}
		if _, err := reader.Discard(2); err != nil {
			t.Fatal(err)
		}
		body := make([]byte, size)
		if _, err := io.ReadFull(reader, body); err != nil {
			t.Fatal(err)
		}
		var response struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatal(err)
		}
		if response.ID != 2 {
			continue
		}
		var locations []Location
		if err := json.Unmarshal(response.Result, &locations); err != nil || len(locations) != 1 || locations[0].URI != uri {
			t.Fatalf("query lost the analysis before its queued edit: %s %v", body, err)
		}
		return
	}
}

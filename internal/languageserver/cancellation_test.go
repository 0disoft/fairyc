package languageserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
)

func TestRequestControlCancelsOnlyMatchingRequestAndSupersededAnalysis(t *testing.T) {
	control := newRequestControl()
	first, one := control.start(context.Background(), message{ID: json.RawMessage(`"alpha"`), Method: "textDocument/definition"})
	second, two := control.start(context.Background(), message{ID: json.RawMessage(`1`), Method: "textDocument/completion"})
	control.cancelRequest(json.RawMessage(`"\u0061lpha"`))
	if !errors.Is(first.Err(), context.Canceled) || second.Err() != nil {
		t.Fatalf("request selection: %v %v", first.Err(), second.Err())
	}
	analysis, three := control.start(context.Background(), message{Method: "textDocument/didOpen"})
	latest, four := control.start(context.Background(), message{Method: "textDocument/didChange"})
	if !errors.Is(analysis.Err(), context.Canceled) || latest.Err() != nil || second.Err() != nil {
		t.Fatal("supersession canceled an unrelated request")
	}
	for _, token := range []uint64{one, two, three, four} {
		control.finish(token)
	}
	if len(control.pending) != 0 {
		t.Fatal("completed requests retained")
	}
}

func TestProtocolReaderReceivesCancellationWhileHandlerIsBusy(t *testing.T) {
	var wire bytes.Buffer
	for _, value := range []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "textDocument/definition", "params": map[string]any{}},
		map[string]any{"jsonrpc": "2.0", "method": "$/cancelRequest", "params": map[string]any{"id": 1}},
		map[string]any{"jsonrpc": "2.0", "method": "exit", "params": map[string]any{}},
	} {
		if err := writeMessage(&wire, value); err != nil {
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
		t.Fatal("cancellation waited for handler completion")
	}
	control.finish(first.token)
	next := <-incoming
	if next.message.Method != "exit" {
		t.Fatalf("next message: %+v", next)
	}
	control.finish(next.token)
	if _, open := <-incoming; open {
		t.Fatal("reader did not stop at exit")
	}
}

func TestCanceledRecheckPreservesPublishedView(t *testing.T) {
	root, filename := editorFixture(t)
	ctx, stop := context.WithCancel(context.Background())
	stop()
	old := &analysis{clean: true}
	d := &document{URI: fileURI(filename), Path: filename, Text: []byte("module main; fn i32 main(){return 0;}"), Version: 2, View: old, LastClean: old}
	var out bytes.Buffer
	s := server{ctx: ctx, root: root, out: &out, docs: map[string]*document{d.URI: d}}
	if err := s.recheck(); !errors.Is(err, context.Canceled) || out.Len() != 0 || d.View != old || d.LastClean != old {
		t.Fatalf("canceled recheck: %v output=%q", err, out.String())
	}
	if view, err := s.analyzeContext(ctx, d, root); !errors.Is(err, context.Canceled) || view != nil {
		t.Fatalf("canceled analysis: %+v %v", view, err)
	}
}

func TestServeContextStopsBlockedInput(t *testing.T) {
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	finished := make(chan int, 1)
	var output, diagnostics bytes.Buffer
	go func() { finished <- ServeContext(ctx, input, &output, &diagnostics) }()
	stop()
	select {
	case code := <-finished:
		if code != 0 || output.Len() != 0 || diagnostics.Len() != 0 {
			t.Fatalf("stop: %d %q %q", code, output.String(), diagnostics.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocked input prevented cancellation")
	}
}

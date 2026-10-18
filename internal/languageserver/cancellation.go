package languageserver

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"
	"sync"
)

type pendingRequest struct {
	cancel   context.CancelFunc
	id       string
	analysis bool
}

// Only cancellation crosses the protocol reader/handler boundary. Documents,
// compiler results and writes remain owned by the ordered handler goroutine.
type requestControl struct {
	mu      sync.Mutex
	next    uint64
	pending map[uint64]pendingRequest
}

func newRequestControl() *requestControl {
	return &requestControl{pending: make(map[uint64]pendingRequest)}
}

func requestID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return "string:" + text
	}
	return "number:" + strings.TrimSpace(string(raw))
}

func rechecks(method string) bool {
	switch method {
	case "textDocument/didOpen", "textDocument/didChange", "textDocument/didSave", "textDocument/didClose", "workspace/didChangeWatchedFiles":
		return true
	default:
		return false
	}
}

func (control *requestControl) start(parent context.Context, m message) (context.Context, uint64) {
	control.mu.Lock()
	defer control.mu.Unlock()
	analysis := rechecks(m.Method)
	if analysis || m.Method == "shutdown" || m.Method == "exit" {
		for _, pending := range control.pending {
			if pending.analysis {
				pending.cancel()
			}
		}
	}
	ctx, cancel := context.WithCancel(parent)
	control.next++
	control.pending[control.next] = pendingRequest{cancel: cancel, id: requestID(m.ID), analysis: analysis}
	return ctx, control.next
}

func (control *requestControl) cancelRequest(id json.RawMessage) {
	key := requestID(id)
	if key == "" {
		return
	}
	control.mu.Lock()
	defer control.mu.Unlock()
	for _, pending := range control.pending {
		if pending.id == key {
			pending.cancel()
		}
	}
}

func (control *requestControl) finish(token uint64) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if pending, found := control.pending[token]; found {
		pending.cancel()
		delete(control.pending, token)
	}
}

type incomingMessage struct {
	message message
	ctx     context.Context
	token   uint64
	err     error
}

func readMessages(ctx context.Context, reader *bufio.Reader, control *requestControl, incoming chan<- incomingMessage) {
	defer close(incoming)
	for {
		m, err := readMessage(reader)
		if err != nil {
			select {
			case incoming <- incomingMessage{err: err}:
			case <-ctx.Done():
			}
			return
		}
		if m.Method == "$/cancelRequest" {
			var params struct {
				ID json.RawMessage `json:"id"`
			}
			if json.Unmarshal(m.Params, &params) == nil {
				control.cancelRequest(params.ID)
			}
			continue
		}
		requestCtx, token := control.start(ctx, m)
		select {
		case incoming <- incomingMessage{message: m, ctx: requestCtx, token: token}:
		case <-ctx.Done():
			control.finish(token)
			return
		}
		if m.Method == "exit" {
			return
		}
	}
}

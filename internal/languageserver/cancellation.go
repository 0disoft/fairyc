package languageserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

const (
	maxPendingRequests     = 64
	maxPendingRequestBytes = 32 << 20
)

var errPendingRequestLimit = errors.New("LSP pending request limit exceeded")

type pendingRequest struct {
	cancel   context.CancelFunc
	id       string
	analysis bool
	size     int
}

// Only cancellation crosses the protocol reader/handler boundary. Documents,
// compiler results and writes remain owned by the ordered handler goroutine.
type requestControl struct {
	mu      sync.Mutex
	next    uint64
	pending map[uint64]pendingRequest
	bytes   int
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

func (control *requestControl) start(parent context.Context, m message) (context.Context, uint64, error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if err := parent.Err(); err != nil {
		return nil, 0, err
	}
	size := len(m.JSONRPC) + len(m.ID) + len(m.Method) + len(m.Params)
	if len(control.pending) >= maxPendingRequests || size > maxPendingRequestBytes-control.bytes {
		return nil, 0, errPendingRequestLimit
	}
	analysis := rechecks(m.Method)
	if analysis || m.Method == "shutdown" || m.Method == "exit" {
		// A queued query must still observe the analysis before it. Only
		// supersede analyses newer than every outstanding query barrier.
		var barrier uint64
		for token, pending := range control.pending {
			if pending.id != "" && token > barrier {
				barrier = token
			}
		}
		for token, pending := range control.pending {
			if pending.analysis && token > barrier {
				pending.cancel()
			}
		}
	}
	ctx, cancel := context.WithCancel(parent)
	control.next++
	control.pending[control.next] = pendingRequest{cancel: cancel, id: requestID(m.ID), analysis: analysis, size: size}
	control.bytes += size
	return ctx, control.next, nil
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
		control.bytes -= pending.size
		delete(control.pending, token)
	}
}

func (control *requestControl) abort() {
	control.mu.Lock()
	defer control.mu.Unlock()
	for _, pending := range control.pending {
		pending.cancel()
	}
	clear(control.pending)
	control.bytes = 0
}

type incomingMessage struct {
	message message
	ctx     context.Context
	token   uint64
	err     error
}

func readMessages(ctx context.Context, reader *bufio.Reader, control *requestControl, incoming chan<- incomingMessage) {
	defer close(incoming)
	ctx, stop := context.WithCancel(ctx)
	defer func() {
		stop()
		control.abort()
	}()
	// Reading must continue while the ordered handler is busy. The admission
	// limits cover active plus queued payloads; one extra slot carries an error.
	queued := make(chan incomingMessage, maxPendingRequests+1)
	go queueMessages(ctx, reader, control, queued)
	for {
		select {
		case <-ctx.Done():
			return
		case next, open := <-queued:
			if !open {
				return
			}
			select {
			case incoming <- next:
			case <-ctx.Done():
				return
			}
		}
	}
}

func queueMessages(ctx context.Context, reader *bufio.Reader, control *requestControl, queued chan<- incomingMessage) {
	defer close(queued)
	defer func() {
		if ctx.Err() != nil {
			control.abort()
		}
	}()
	for {
		if ctx.Err() != nil {
			return
		}
		m, err := readMessage(reader)
		if err != nil {
			select {
			case queued <- incomingMessage{err: err}:
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
		requestCtx, token, err := control.start(ctx, m)
		if err != nil {
			control.abort()
			select {
			case queued <- incomingMessage{err: err}:
			case <-ctx.Done():
			}
			return
		}
		select {
		case queued <- incomingMessage{message: m, ctx: requestCtx, token: token}:
		case <-ctx.Done():
			control.finish(token)
			return
		}
		if m.Method == "exit" {
			return
		}
	}
}

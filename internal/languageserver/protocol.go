// Package languageserver implements a local stdio LSP shell over compiler-owned analysis.
package languageserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxMessage = 9 << 20

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func readMessage(in *bufio.Reader) (message, error) {
	length := -1
	headerBytes := 0
	for {
		rawLine, err := in.ReadSlice('\n')
		if err != nil {
			return message{}, err
		}
		line := string(rawLine)
		headerBytes += len(line)
		if headerBytes > 8192 {
			return message{}, errors.New("LSP header too large")
		}
		if line == "\r\n" {
			break
		}
		key, value, ok := strings.Cut(strings.TrimSuffix(line, "\r\n"), ":")
		if !ok {
			return message{}, errors.New("invalid LSP header")
		}
		if strings.EqualFold(key, "Content-Length") {
			if length != -1 {
				return message{}, errors.New("duplicate Content-Length")
			}
			length, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return message{}, err
			}
		}
	}
	if length < 0 || length > maxMessage {
		return message{}, errors.New("invalid LSP message length")
	}
	bytes := make([]byte, length)
	if _, err := io.ReadFull(in, bytes); err != nil {
		return message{}, err
	}
	var m message
	if err := json.Unmarshal(bytes, &m); err != nil {
		return message{}, err
	}
	if m.JSONRPC != "2.0" || m.Method == "" {
		return message{}, errors.New("invalid JSON-RPC request")
	}
	return m, nil
}
func writeMessage(out io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}
func (s *server) notify(method string, params any) error {
	return writeMessage(s.out, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}
func (s *server) response(id json.RawMessage, result any, code int, problem string) error {
	if len(id) == 0 {
		return nil
	}
	m := map[string]any{"jsonrpc": "2.0", "id": id}
	if code == 0 {
		m["result"] = result
	} else {
		m["error"] = map[string]any{"code": code, "message": problem}
	}
	return writeMessage(s.out, m)
}

// Serve runs one foreground stdio session. It never builds, executes, downloads or writes files.
func Serve(in io.Reader, out, stderr io.Writer) int {
	return ServeContext(context.Background(), in, out, stderr)
}

// ServeContext cancels compilation when its parent is canceled or a newer
// document update supersedes an in-flight recheck. It owns the session input;
// a blocking input must implement io.Closer to unblock reads on shutdown.
func ServeContext(ctx context.Context, in io.Reader, out, stderr io.Writer) int {
	ctx, stop := context.WithCancel(ctx)
	defer func() {
		stop()
		if closer, ok := in.(io.Closer); ok {
			_ = closer.Close()
		}
	}()
	s := server{out: out, docs: map[string]*document{}}
	requests := newRequestControl()
	incoming := make(chan incomingMessage)
	go readMessages(ctx, bufio.NewReaderSize(in, 8192), requests, incoming)
	fail := func(err error) int { fmt.Fprintln(stderr, err); return 1 }
	for {
		select {
		case <-ctx.Done():
			return 0
		case next, ok := <-incoming:
			if !ok {
				return 0
			}
			if next.err != nil {
				if errors.Is(next.err, io.EOF) {
					return 0
				}
				return fail(next.err)
			}
			m := next.message
			if m.Method == "exit" {
				requests.finish(next.token)
				if s.shutdown {
					return 0
				}
				return 1
			}
			s.ctx = next.ctx
			var err error
			if next.ctx.Err() != nil && len(m.ID) != 0 {
				err = s.response(m.ID, nil, -32800, "Request cancelled")
			} else {
				err = s.handle(m)
			}
			requests.finish(next.token)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if len(m.ID) != 0 {
					if err = s.response(m.ID, nil, -32800, "Request cancelled"); err != nil {
						return fail(err)
					}
				}
				continue
			}
			if err != nil {
				return fail(err)
			}
		}
	}
}

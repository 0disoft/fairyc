// Package atlasmcp implements a bounded, read-only MCP stdio session, without host authority.
package atlasmcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const (
	ProtocolVersion = "2025-11-25"
	MaxInput        = 64 << 10

	maxSessionMessages = 1024
	minSessionOutput   = 2048
	maxSessionOutput   = 4 << 20
	initialInputBuffer = 4096
	maxLineEndingBytes = 2
	maxRequestIDBytes  = 512
	maxIntegerIDBytes  = 20
	maxJSONNesting     = 16
)

var ErrOutput = errors.New("MCP output failed")

type Impact func(uid, symbol string) (json.RawMessage, error)
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type response struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type request struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

var integerID = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// Serve reads newline-delimited JSON-RPC. Limits and the only capability are supplied by the host.
func Serve(input io.Reader, output io.Writer, maxMessages, maxOutput int, impact Impact) error {
	return ServeWithOptions(input, output, maxMessages, maxOutput, impact)
}

// ServeOptions controls one session. A nil Context uses context.Background.
// Input, output and impact remain caller-supplied I/O ports. Cancellation is
// cooperative between reads and dispatch/write boundaries; it does not close
// caller-owned streams or interrupt an in-flight host callback or syscall.
type ServeOptions struct {
	Context context.Context
}

// ServeWithOptions preserves Serve's required inputs and result shape.
func ServeWithOptions(input io.Reader, output io.Writer, maxMessages, maxOutput int, impact Impact, options ...ServeOptions) error {
	if len(options) > 1 {
		return fmt.Errorf("at most one MCP session options value is supported")
	}
	ctx := context.Background()
	if len(options) == 1 && options[0].Context != nil {
		ctx = options[0].Context
	}
	if maxMessages < 1 || maxMessages > maxSessionMessages || maxOutput < minSessionOutput || maxOutput > maxSessionOutput || impact == nil {
		return fmt.Errorf("invalid MCP session policy")
	}
	scanner := bufio.NewScanner(mcpInput{ctx: ctx, reader: input})
	// ScanLines strips CRLF as well as LF. Reserve both delimiter bytes while
	// retaining the independent payload bound, including the final EOF line.
	scanner.Buffer(make([]byte, initialInputBuffer), MaxInput+maxLineEndingBytes)
	var session mcpSession
	for count := 0; scanner.Scan(); count++ {
		if err := mcpContextError(ctx); err != nil {
			return err
		}
		if count >= maxMessages {
			return fmt.Errorf("MCP message budget exhausted")
		}
		if err := session.serveMessage(ctx, scanner.Bytes(), output, maxOutput, impact); err != nil {
			return err
		}
	}
	if stop := mcpContextError(ctx); stop != nil {
		return errors.Join(scanner.Err(), stop)
	}
	return scanner.Err()
}

func mcpContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	return nil
}

type mcpInput struct {
	ctx    context.Context
	reader io.Reader
}

func (input mcpInput) Read(buffer []byte) (int, error) {
	if err := mcpContextError(input.ctx); err != nil {
		return 0, err
	}
	n, err := input.reader.Read(buffer)
	if stop := mcpContextError(input.ctx); stop != nil {
		return n, errors.Join(err, stop)
	}
	return n, err
}

func validID(id json.RawMessage) bool {
	if len(id) > maxRequestIDBytes {
		return false
	}
	if len(id) == 0 {
		return true
	}
	var s string
	return (json.Unmarshal(id, &s) == nil && len(id) > 0 && id[0] == '"') || (len(id) <= maxIntegerIDBytes && integerID.Match(id))
}
func emptyParams(p json.RawMessage) bool {
	if len(p) == 0 {
		return true
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(p, &fields) != nil || fields == nil {
		return false
	}
	if len(fields) == 0 {
		return true
	}
	var meta map[string]json.RawMessage
	return len(fields) == 1 && json.Unmarshal(fields["_meta"], &meta) == nil && meta != nil
}

// Reject duplicate keys and excessive nesting before any capability dispatch.
func strictJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > maxJSONNesting {
			return fmt.Errorf("JSON nesting limit")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		if delim != '{' && delim != '[' {
			return fmt.Errorf("unexpected delimiter")
		}
		seen := map[string]bool{}
		for d.More() {
			if delim == '{' {
				key, err := d.Token()
				if err != nil {
					return err
				}
				s, ok := key.(string)
				if !ok || seen[s] {
					return fmt.Errorf("duplicate key")
				}
				seen[s] = true
			}
			if err := walk(depth + 1); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}

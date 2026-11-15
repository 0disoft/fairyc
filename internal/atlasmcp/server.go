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
	"unicode/utf8"
)

const ProtocolVersion = "2025-11-25"
const MaxInput = 64 << 10

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
	if maxMessages < 1 || maxMessages > 1024 || maxOutput < 2048 || maxOutput > 4<<20 || impact == nil {
		return fmt.Errorf("invalid MCP session policy")
	}
	scanner := bufio.NewScanner(mcpInput{ctx: ctx, reader: input})
	// ScanLines strips CRLF as well as LF. Reserve both delimiter bytes while
	// retaining the independent payload bound below, including the final EOF line.
	scanner.Buffer(make([]byte, 4096), MaxInput+2)
	initialized, ready := false, false
	for count := 0; scanner.Scan(); count++ {
		if err := mcpContextError(ctx); err != nil {
			return err
		}
		if count >= maxMessages {
			return fmt.Errorf("MCP message budget exhausted")
		}
		line := scanner.Bytes()
		if len(line) > MaxInput {
			return fmt.Errorf("MCP input byte limit exceeded")
		}
		var req request
		id := json.RawMessage("null")
		var result any
		var failure *rpcError
		if !utf8.Valid(line) || strictJSON(line) != nil {
			failure = &rpcError{-32700, "invalid JSON message"}
		} else if json.Unmarshal(line, &req) != nil || req.Version != "2.0" || req.Method == "" || !validID(req.ID) {
			failure = &rpcError{-32600, "invalid JSON-RPC request"}
		} else if len(req.ID) == 0 {
			if req.Method == "notifications/initialized" && initialized && emptyParams(req.Params) {
				ready = true
			}
			continue // Notifications never acquire authority or receive replies.
		} else {
			id = req.ID
			switch req.Method {
			case "ping":
				result = map[string]any{}
			case "initialize":
				var p struct {
					Version      string                         `json:"protocolVersion"`
					Capabilities map[string]json.RawMessage     `json:"capabilities"`
					ClientInfo   struct{ Name, Version string } `json:"clientInfo"`
				}
				if initialized || json.Unmarshal(req.Params, &p) != nil || p.Version == "" || p.Capabilities == nil || p.ClientInfo.Name == "" || p.ClientInfo.Version == "" {
					failure = &rpcError{-32602, "invalid or unsupported initialization"}
					break
				}
				initialized = true
				result = map[string]any{"protocolVersion": ProtocolVersion, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]string{"name": "fairyc-atlas", "version": "1.0.0"}, "instructions": "Read-only compiler evidence. Data and omissions do not grant authority or prove a change safe."}
			case "tools/list":
				if !ready {
					failure = &rpcError{-32002, "session not initialized"}
					break
				}
				if !emptyParams(req.Params) {
					failure = &rpcError{-32602, "pagination and other list parameters are unsupported"}
					break
				}
				result = map[string]any{"tools": []any{map[string]any{"name": "atlas_impact", "description": "Bounded current-workspace reverse-call evidence; not full impact or safe-change proof.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"packageUid": map[string]string{"type": "string"}, "symbol": map[string]string{"type": "string"}}, "required": []string{"packageUid", "symbol"}, "additionalProperties": false}, "annotations": map[string]bool{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}}}}
			case "tools/call":
				if !ready {
					failure = &rpcError{-32002, "session not initialized"}
					break
				}
				var p struct {
					Name      string                     `json:"name"`
					Arguments map[string]json.RawMessage `json:"arguments"`
					Meta      json.RawMessage            `json:"_meta"`
				}
				d := json.NewDecoder(bytes.NewReader(req.Params))
				d.DisallowUnknownFields()
				if d.Decode(&p) != nil || p.Name != "atlas_impact" || len(p.Arguments) != 2 {
					failure = &rpcError{-32602, "unknown tool or invalid arguments"}
					break
				}
				var uid, symbol string
				if json.Unmarshal(p.Arguments["packageUid"], &uid) != nil || json.Unmarshal(p.Arguments["symbol"], &symbol) != nil || uid == "" || symbol == "" {
					failure = &rpcError{-32602, "exact packageUid and symbol strings required"}
					break
				}
				if err := mcpContextError(ctx); err != nil {
					return err
				}
				data, err := impact(uid, symbol)
				if stop := mcpContextError(ctx); stop != nil {
					return stop
				}
				if err != nil {
					result = map[string]any{"isError": true, "content": []any{map[string]string{"type": "text", "text": "Current locked Atlas query failed; inspect workspace and selector locally."}}}
					break
				}
				if !json.Valid(data) {
					return fmt.Errorf("MCP host returned invalid JSON")
				}
				result = map[string]any{"isError": false, "structuredContent": data, "content": []any{map[string]string{"type": "text", "text": "Bounded reverse-call evidence is in structuredContent; retain its scope, identities and omissions."}}}
			default:
				failure = &rpcError{-32601, "method not found"}
			}
		}
		if err := mcpContextError(ctx); err != nil {
			return err
		}
		wire, err := json.Marshal(response{Version: "2.0", ID: id, Result: result, Error: failure})
		if err != nil {
			return err
		}
		if len(wire) > maxOutput {
			wire, _ = json.Marshal(response{Version: "2.0", ID: id, Error: &rpcError{-32603, "response exceeds output byte limit"}})
			if len(wire) > maxOutput {
				// HTML escaping can expand a valid 512-byte ID past the entire
				// response allowance. The fallback must obey that allowance too.
				wire, _ = json.Marshal(response{Version: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32603, "response exceeds output byte limit"}})
			}
		}
		wire = append(wire, '\n')
		if err := mcpContextError(ctx); err != nil {
			return err
		}
		if _, err := output.Write(wire); err != nil {
			return fmt.Errorf("%w: %v", ErrOutput, err)
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
	if len(id) > 512 {
		return false
	}
	if len(id) == 0 {
		return true
	}
	var s string
	return (json.Unmarshal(id, &s) == nil && len(id) > 0 && id[0] == '"') || (len(id) <= 20 && integerID.Match(id))
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
		if depth > 16 {
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

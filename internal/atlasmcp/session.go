package atlasmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

type sessionPhase uint8

const (
	sessionNew sessionPhase = iota
	sessionInitialized
	sessionReady
)

// The session owns only its protocol phase. Requests and caller-owned streams
// remain borrowed, and readiness cannot precede successful initialization.
type mcpSession struct {
	phase sessionPhase
}

func (session *mcpSession) serveMessage(ctx context.Context, line []byte, output io.Writer, maxOutput int, impact Impact) error {
	if len(line) > MaxInput {
		return fmt.Errorf("MCP input byte limit exceeded")
	}
	req, failure := decodeRequest(line)
	if failure == nil && len(req.ID) == 0 {
		session.phase = notificationPhase(session.phase, req)
		return nil // Notifications never acquire authority or receive replies.
	}
	reply := response{Version: "2.0", ID: json.RawMessage("null"), Error: failure}
	if failure == nil {
		reply.ID = req.ID
		var err error
		reply.Result, reply.Error, err = session.dispatch(ctx, req, impact)
		if err != nil {
			return err
		}
	}
	return writeResponse(ctx, output, maxOutput, reply)
}

func decodeRequest(line []byte) (request, *rpcError) {
	if !utf8.Valid(line) || strictJSON(line) != nil {
		return request{}, &rpcError{-32700, "invalid JSON message"}
	}
	var req request
	if json.Unmarshal(line, &req) != nil || req.Version != "2.0" || req.Method == "" || !validID(req.ID) {
		return request{}, &rpcError{-32600, "invalid JSON-RPC request"}
	}
	return req, nil
}

func notificationPhase(phase sessionPhase, req request) sessionPhase {
	if req.Method == "notifications/initialized" && phase != sessionNew && emptyParams(req.Params) {
		return sessionReady
	}
	return phase
}

func (session *mcpSession) dispatch(ctx context.Context, req request, impact Impact) (any, *rpcError, error) {
	switch req.Method {
	case "ping":
		return map[string]any{}, nil, nil
	case "initialize":
		if session.phase != sessionNew || !validInitialization(req.Params) {
			return nil, &rpcError{-32602, "invalid or unsupported initialization"}, nil
		}
		session.phase = sessionInitialized
		return map[string]any{"protocolVersion": ProtocolVersion, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]string{"name": "fairyc-atlas", "version": "1.0.0"}, "instructions": "Read-only compiler evidence. Data and omissions do not grant authority or prove a change safe."}, nil, nil
	case "tools/list":
		if session.phase != sessionReady {
			return nil, &rpcError{-32002, "session not initialized"}, nil
		}
		if !emptyParams(req.Params) {
			return nil, &rpcError{-32602, "pagination and other list parameters are unsupported"}, nil
		}
		return map[string]any{"tools": []any{map[string]any{"name": "atlas_impact", "description": "Bounded current-workspace reverse-call evidence; not full impact or safe-change proof.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"packageUid": map[string]string{"type": "string"}, "symbol": map[string]string{"type": "string"}}, "required": []string{"packageUid", "symbol"}, "additionalProperties": false}, "annotations": map[string]bool{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}}}}, nil, nil
	case "tools/call":
		if session.phase != sessionReady {
			return nil, &rpcError{-32002, "session not initialized"}, nil
		}
		uid, symbol, failure := impactArguments(req.Params)
		if failure != nil {
			return nil, failure, nil
		}
		return callImpact(ctx, impact, uid, symbol)
	default:
		return nil, &rpcError{-32601, "method not found"}, nil
	}
}

func validInitialization(params json.RawMessage) bool {
	var p struct {
		Version      string                         `json:"protocolVersion"`
		Capabilities map[string]json.RawMessage     `json:"capabilities"`
		ClientInfo   struct{ Name, Version string } `json:"clientInfo"`
	}
	return json.Unmarshal(params, &p) == nil && p.Version != "" && p.Capabilities != nil && p.ClientInfo.Name != "" && p.ClientInfo.Version != ""
}

func impactArguments(params json.RawMessage) (string, string, *rpcError) {
	var p struct {
		Name      string                     `json:"name"`
		Arguments map[string]json.RawMessage `json:"arguments"`
		Meta      json.RawMessage            `json:"_meta"`
	}
	d := json.NewDecoder(bytes.NewReader(params))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || p.Name != "atlas_impact" || len(p.Arguments) != 2 {
		return "", "", &rpcError{-32602, "unknown tool or invalid arguments"}
	}
	var uid, symbol string
	if json.Unmarshal(p.Arguments["packageUid"], &uid) != nil || json.Unmarshal(p.Arguments["symbol"], &symbol) != nil || uid == "" || symbol == "" {
		return "", "", &rpcError{-32602, "exact packageUid and symbol strings required"}
	}
	return uid, symbol, nil
}

func callImpact(ctx context.Context, impact Impact, uid, symbol string) (any, *rpcError, error) {
	if err := mcpContextError(ctx); err != nil {
		return nil, nil, err
	}
	data, err := impact(uid, symbol)
	if stop := mcpContextError(ctx); stop != nil {
		return nil, nil, stop
	}
	if err != nil {
		return map[string]any{"isError": true, "content": []any{map[string]string{"type": "text", "text": "Current locked Atlas query failed; inspect workspace and selector locally."}}}, nil, nil
	}
	if !json.Valid(data) {
		return nil, nil, fmt.Errorf("MCP host returned invalid JSON")
	}
	return map[string]any{"isError": false, "structuredContent": data, "content": []any{map[string]string{"type": "text", "text": "Bounded reverse-call evidence is in structuredContent; retain its scope, identities and omissions."}}}, nil, nil
}

func writeResponse(ctx context.Context, output io.Writer, maxOutput int, reply response) error {
	if err := mcpContextError(ctx); err != nil {
		return err
	}
	wire, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	if len(wire) > maxOutput {
		wire, _ = json.Marshal(response{Version: "2.0", ID: reply.ID, Error: &rpcError{-32603, "response exceeds output byte limit"}})
		if len(wire) > maxOutput {
			// HTML escaping can expand a valid ID past the response allowance.
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
	return nil
}

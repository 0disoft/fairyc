package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAtlasMCPCurrentLockedPermissionAndNoSource(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	s := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")
	args := []string{"experimental", "atlas-mcp", "--allow-impact", "--max-packages", "8", "--max-calls", "128", "--max-depth", "8", "--max-symbols", "64", "--max-output-bytes", "100000", "--max-requests", "10"}
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"cli-test","version":"1"}}}`
	call, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "atlas_impact", "arguments": map[string]string{"packageUid": "example.session.answer", "symbol": s.Visible}}})
	input := init + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" + string(call) + "\n"
	var out, err bytes.Buffer
	code := RunWithInput(args, root, strings.NewReader(input), &out, &err)
	if code != 0 || err.Len() != 0 || strings.Contains(out.String(), root) || !strings.Contains(out.String(), `"isError":false`) {
		t.Fatalf("MCP %d %s %s", code, &out, &err)
	}
	if strings.Contains(out.String(), workspaceSessionRootSource) {
		t.Fatal("source leaked")
	}
	out.Reset()
	err.Reset()
	code = RunWithInput(append(args[:2], args[3:]...), root, strings.NewReader(input), &out, &err)
	if code != ExitInvalidInvocation || out.Len() != 0 {
		t.Fatal("implicit capability accepted")
	}
	writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource+"\n")
	out.Reset()
	err.Reset()
	// Recreate policy rather than reuse the append-mutated argument backing array.
	args = []string{"experimental", "atlas-mcp", "--allow-impact", "--max-packages", "8", "--max-calls", "128", "--max-depth", "8", "--max-symbols", "64", "--max-output-bytes", "100000", "--max-requests", "10"}
	if code := RunWithInput(args, root, strings.NewReader(input), &out, &err); code != 0 || !strings.Contains(out.String(), `"isError":true`) || strings.Contains(out.String(), root) {
		t.Fatal("stale workspace not safely rejected", &out, &err)
	}
}

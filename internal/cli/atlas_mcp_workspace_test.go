package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestAtlasMCPSessionReusesPreparationAndIndexWithoutBypassingQueryPolicy(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	member := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")
	main := atlasFunctionSymbolForPackage(t, root, "example.session.app", "main")
	o := impactOptions{uid: "example.session.answer", symbol: member.Visible, packages: 8, calls: 128, depth: 8, symbols: 64, output: 100000}
	session := atlasImpactSession{root: root}
	first, code := session.document(o, io.Discard)
	if code != ExitSuccess || session.files == nil || session.prepared.index == nil {
		t.Fatalf("cold query = %d, %s", code, first)
	}
	prepared, index := session.prepared, session.prepared.index
	for range 3 {
		data, code := session.document(o, io.Discard)
		if code != ExitSuccess || !bytes.Equal(data, first) || session.prepared != prepared || session.prepared.index != index {
			t.Fatalf("warm query did not reuse its exact preparation: %d, %s", code, data)
		}
	}
	selected := o
	selected.uid, selected.symbol = "example.session.app", main.Full
	data, code := session.document(selected, io.Discard)
	if code != ExitSuccess || bytes.Equal(data, first) || session.prepared != prepared || session.prepared.index != index {
		t.Fatalf("selector was not evaluated independently: %d, %s", code, data)
	}
	selected = o
	selected.depth = 0
	data, code = session.document(selected, io.Discard)
	var impact atlasmap.ImpactResult
	if code != ExitSuccess || json.Unmarshal(data, &impact) != nil || len(impact.Nodes) != 1 {
		t.Fatalf("depth limit bypassed: %d, %s", code, data)
	}
	for _, policy := range []string{"packages", "calls", "output", "package", "symbol"} {
		t.Run(policy, func(t *testing.T) {
			limited := o
			switch policy {
			case "packages":
				limited.packages = 1
			case "calls":
				limited.calls = 0
			case "output":
				limited.output = 1
			case "package":
				limited.uid = "example.missing"
			case "symbol":
				limited.symbol = strings.Repeat("1", 64)
			}
			if data, code := session.document(limited, io.Discard); code != ExitInvalidInvocation || len(data) != 0 || session.prepared != prepared {
				t.Fatalf("warm policy %s bypassed: %d, %s", policy, code, data)
			}
		})
	}
}

func TestAtlasMCPSessionInvalidatesInputsAndRecovers(t *testing.T) {
	for _, change := range []string{"closed source", "added source", "removed source", "manifest", "lock", "missing lock"} {
		t.Run(change, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
			writeWorkspaceFile(t, root, "src/spare.fy", "module spare; fn i32 spare() { return 1; }\n")
			symbol := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")
			o := impactOptions{uid: "example.session.answer", symbol: symbol.Visible, packages: 8, calls: 128, depth: 8, symbols: 64, output: 100000}
			session := atlasImpactSession{root: root}
			if _, code := session.document(o, io.Discard); code != ExitSuccess {
				t.Fatalf("cold query = %d", code)
			}
			original := session.prepared
			switch change {
			case "closed source":
				name := filepath.Join(root, "libs/answer/src/answer.fy")
				info, err := os.Stat(name)
				if err != nil {
					t.Fatal(err)
				}
				writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", strings.Replace(workspaceSessionMemberSource, "42", "43", 1))
				if err := os.Chtimes(name, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
			case "added source":
				writeWorkspaceFile(t, root, "src/extra.fy", "module extra; fn i32 extra() { return 2; }\n")
			case "removed source":
				if err := os.Remove(filepath.Join(root, "src/spare.fy")); err != nil {
					t.Fatal(err)
				}
			case "manifest":
				writeWorkspaceFile(t, root, "fairy.toml", strings.Replace(workspaceSessionRootManifest, `version = "1.0.0"`, `version = "1.0.1"`, 1))
			case "lock":
				writeWorkspaceFile(t, root, "fairy.lock", "invalid lock\n")
			case "missing lock":
				if err := os.Remove(filepath.Join(root, "fairy.lock")); err != nil {
					t.Fatal(err)
				}
			}
			if data, code := session.document(o, io.Discard); code != ExitPackageFailure || len(data) != 0 || session.prepared != nil || session.files != nil {
				t.Fatalf("stale %s query = %d, %s, retained = %t", change, code, data, session.prepared != nil)
			}
			// Only ordinary workspace loading updates the lock; the MCP query
			// must neither repair it nor become permanently poisoned by failure.
			if _, _, err := loadWorkspaceWithLock(root, false); err != nil {
				t.Fatal(err)
			}
			if data, code := session.document(o, io.Discard); code != ExitSuccess || session.prepared == original {
				t.Fatalf("recovery = %d, %s", code, data)
			}
			current := session.prepared
			if _, code := session.document(o, io.Discard); code != ExitSuccess || session.prepared != current {
				t.Fatal("replacement preparation not reused")
			}
		})
	}
}

func TestAtlasMCPSessionRejectsInvalidLockCapturedAfterPreparation(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	symbol := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")
	o := impactOptions{uid: "example.session.answer", symbol: symbol.Visible, packages: 8, calls: 128, depth: 8, symbols: 64, output: 100000}
	session := atlasImpactSession{root: root}
	if _, code := session.document(o, io.Discard); code != ExitSuccess {
		t.Fatal(code)
	}
	writeWorkspaceFile(t, root, "fairy.lock", "invalid lock\n")
	// Reproduce a lock edit between locked preparation and snapshot capture.
	var err error
	session.files, err = workspaceloader.CaptureFileSnapshot(context.Background(), session.prepared.workspace.workspace, nil)
	if err != nil {
		t.Fatal(err)
	}
	if data, code := session.document(o, io.Discard); code != ExitPackageFailure || len(data) != 0 {
		t.Fatalf("unvalidated lock reused = %d, %s", code, data)
	}
}

func TestAtlasMCPSeesChangesBetweenProtocolRequests(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	symbol := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")
	call, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "atlas_impact", "arguments": map[string]string{"packageUid": "example.session.answer", "symbol": symbol.Visible}}})
	call = append(call, '\n')
	reader := &workspaceSessionStagedReader{stages: []workspaceSessionReaderStage{
		{data: []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"cli-test","version":"1"}}}` + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")},
		{data: call},
		{data: call},
		{before: func() { writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource+"\n") }, data: call},
		{before: func() {
			if _, _, err := loadWorkspaceWithLock(root, false); err != nil {
				t.Fatal(err)
			}
		}, data: call},
	}}
	args := []string{"experimental", "atlas-mcp", "--allow-impact", "--max-packages", "8", "--max-calls", "128", "--max-depth", "8", "--max-symbols", "64", "--max-output-bytes", "100000", "--max-requests", "8"}
	var out, stderr bytes.Buffer
	if code := RunWithInput(args, root, reader, &out, &stderr); code != ExitSuccess || stderr.Len() != 0 || strings.Count(out.String(), `"isError":false`) != 3 || strings.Count(out.String(), `"isError":true`) != 1 || strings.Contains(out.String(), root) {
		t.Fatalf("MCP recovery = %d, %s, %s", code, &out, &stderr)
	}
}

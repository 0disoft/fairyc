package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

type candidateMCPSDKBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (buffer *candidateMCPSDKBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(data)
}

func (buffer *candidateMCPSDKBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}

type candidateMCPSDKReply struct {
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Error   json.RawMessage `json:"error"`
	Result  json.RawMessage `json:"result"`
}

func TestCandidateMCPSDKCommand(t *testing.T) {
	main := `module main;import project_mcp_cli as mcp;@effects(filesystem,environment,console) fn i32 main(){return mcp::arguments();}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, verifyCandidateMCPSDKCommand)
}

func verifyCandidateMCPSDKCommand(t *testing.T, image string) {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, atlasSDKCommandSource)
	prepare := func() (preparedAtlasWorkspace, *atlasmap.WorkspaceCallIndex) {
		t.Helper()
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("MCP SDK oracle: %d %s", code, &diagnostic)
		}
		index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 8192})
		if err != nil {
			t.Fatal(err)
		}
		return prepared, index
	}
	selectSDK := func(prepared preparedAtlasWorkspace) (atlasmap.Partition, string) {
		t.Helper()
		for _, partition := range prepared.partitions {
			for _, symbol := range partition.Symbols() {
				if symbol.Declaration.Module == "std::num" && symbol.Declaration.Name == "parse_i64_decimal" {
					return partition, symbol.Visible
				}
			}
		}
		t.Fatal("missing authenticated std::num::parse_i64_decimal")
		return atlasmap.Partition{}, ""
	}
	prepared, index := prepare()
	sdk, symbol := selectSDK(prepared)
	resolved := len(index.Calls())
	if resolved == 0 {
		t.Fatal("missing SDK resolved-call oracle")
	}
	t.Logf("MCP SDK resolved calls=%d", resolved)

	imageData, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(imageData))
	t.Setenv("PATH", "")
	arguments := replaceWorkspaceSessionArgument(candidateMCPArguments(), "--max-calls", strconv.Itoa(resolved))
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, image, arguments...)
	command.Dir = root
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := &candidateMCPSDKBuffer{}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = input.Close()
		_ = output.Close()
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()

	reader := bufio.NewReader(output)
	read := func(id int) candidateMCPSDKReply {
		t.Helper()
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("MCP SDK response %d: %v; stderr=%s", id, err, stderr.String())
		}
		if len(line) > 100001 || !strings.HasSuffix(line, "\n") || strings.Contains(line, root) ||
			strings.Contains(line, atlasSDKCommandSource) || strings.Contains(line, workspaceSessionMemberSource) {
			t.Fatalf("MCP SDK protocol/leak response %d: %q", id, line)
		}
		var reply candidateMCPSDKReply
		if json.Unmarshal([]byte(line), &reply) != nil || reply.Version != "2.0" || reply.ID != id || len(reply.Error) != 0 || len(reply.Result) == 0 {
			t.Fatalf("MCP SDK JSON-RPC response %d: %s", id, line)
		}
		return reply
	}
	send := func(message string, id int) candidateMCPSDKReply {
		t.Helper()
		if _, err := io.WriteString(input, message+"\n"); err != nil {
			t.Fatalf("MCP SDK request %d: %v; stderr=%s", id, err, stderr.String())
		}
		return read(id)
	}

	initialized := send(mcpInitialize, 1)
	var lifecycle struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools struct {
				ListChanged bool `json:"listChanged"`
			} `json:"tools"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if json.Unmarshal(initialized.Result, &lifecycle) != nil || lifecycle.ProtocolVersion != "2025-11-25" ||
		lifecycle.Capabilities.Tools.ListChanged || lifecycle.ServerInfo.Name != "fairyc-atlas" || lifecycle.ServerInfo.Version != "1.0.0" {
		t.Fatalf("MCP SDK initialize: %s", initialized.Result)
	}
	if _, err := io.WriteString(input, mcpReady+"\n"); err != nil {
		t.Fatalf("MCP SDK initialized notification: %v; stderr=%s", err, stderr.String())
	}
	listed := send(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, 2)
	var tools struct {
		Tools []struct {
			Name        string          `json:"name"`
			Annotations map[string]bool `json:"annotations"`
		} `json:"tools"`
	}
	if json.Unmarshal(listed.Result, &tools) != nil || len(tools.Tools) != 1 || tools.Tools[0].Name != "atlas_impact" ||
		!tools.Tools[0].Annotations["readOnlyHint"] || tools.Tools[0].Annotations["destructiveHint"] {
		t.Fatalf("MCP SDK tools/list: %s", listed.Result)
	}

	type toolResult struct {
		IsError           bool                  `json:"isError"`
		StructuredContent atlasmap.ImpactResult `json:"structuredContent"`
	}
	toolCall := func(partition atlasmap.Partition, selector string, current *atlasmap.WorkspaceCallIndex, id int, failure bool) toolResult {
		t.Helper()
		reply := send(candidateMCPCall(partition.Header().PackageUID, selector, id), id)
		var result toolResult
		if json.Unmarshal(reply.Result, &result) != nil || result.IsError != failure {
			t.Fatalf("MCP SDK tool result %d: %s", id, reply.Result)
		}
		if failure {
			return result
		}
		want, err := current.Impact(partition.Header().PackageInstance, selector, 8, 64)
		if err != nil {
			t.Fatal(err)
		}
		compareCandidateAtlasAggregateImpact(t, result.StructuredContent, want, compiler)
		return result
	}

	before := remoteTree(t, root)
	memberBefore := before["libs/answer/src/answer.fy"]
	toolCall(sdk, symbol, index, 3, false)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("MCP SDK query changed workspace")
	}

	writeWorkspaceFile(t, root, "src/main.fy", atlasSDKCommandSource+"\n")
	stale := remoteTree(t, root)
	if stale["libs/answer/src/answer.fy"] != memberBefore {
		t.Fatal("stale setup changed the workspace member")
	}
	toolCall(sdk, symbol, index, 4, true)
	if !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatal("stale MCP SDK request repaired or otherwise changed workspace")
	}

	if _, _, err := loadWorkspaceWithLock(root, false); err != nil {
		t.Fatalf("relock SDK root source: %v", err)
	}
	currentPrepared, current := prepare()
	currentSDK, currentSymbol := selectSDK(currentPrepared)
	recovered := remoteTree(t, root)
	toolCall(currentSDK, currentSymbol, current, 5, false)
	after := remoteTree(t, root)
	if !reflect.DeepEqual(recovered, after) {
		t.Fatal("recovered MCP SDK query changed workspace")
	}
	if after["libs/answer/src/answer.fy"] != memberBefore {
		t.Fatal("MCP SDK queries changed the workspace member")
	}

	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	remaining, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("MCP SDK final stdout: %v; stderr=%s", err, stderr.String())
	}
	if len(remaining) != 0 {
		t.Fatalf("MCP SDK wrote non-response stdout after completion: %q", remaining)
	}
	if err := command.Wait(); err != nil || stderr.String() != "" {
		t.Fatalf("MCP SDK completion: %v (%v); stderr=%s", err, ctx.Err(), stderr.String())
	}
}

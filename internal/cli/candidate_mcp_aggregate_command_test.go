package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestCandidateMCPAggregateCommand(t *testing.T) {
	main := `module main;import project_mcp_cli as mcp;@effects(filesystem,environment,console)fn i32 main(){return mcp::arguments();}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, verifyCandidateMCPAggregateCommand)
}

func verifyCandidateMCPAggregateCommand(t *testing.T, image string) {
	t.Helper()
	root := candidateAggregateCacheWorkspace(t, candidateAggregateLayoutFixture{root: atlasAggregateCommandRoot, maker: atlasAggregateCommandMaker})
	prepared, index := candidateAtlasAggregateCommandOracle(t, root)
	var selected atlasmap.Partition
	var symbol string
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if p.Header().PackageUID == "example.aggregate.helper" && s.Declaration.Name == "finish" {
				selected, symbol = p, s.Visible
			}
		}
	}
	if symbol == "" {
		t.Fatal("missing aggregate cleanup helper")
	}
	want, err := index.Impact(selected.Header().PackageInstance, symbol, 8, 64)
	if err != nil {
		t.Fatal(err)
	}
	imageBytes, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(imageBytes))
	t.Setenv("PATH", "")
	args := replaceWorkspaceSessionArgument(candidateMCPArguments(), "--max-calls", strconv.Itoa(len(index.Calls())))
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Dir = root
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = in.Close()
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	type reply struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Error   json.RawMessage `json:"error"`
		Result  struct {
			IsError bool                  `json:"isError"`
			Data    atlasmap.ImpactResult `json:"structuredContent"`
		} `json:"result"`
	}
	reader := bufio.NewReader(out)
	send := func(message string, id int, failure bool) reply {
		t.Helper()
		if _, err := in.Write([]byte(message + "\n")); err != nil {
			t.Fatal(err)
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("aggregate MCP response: %v", err)
		}
		var got reply
		if len(line) > 100001 || strings.Contains(line, root) || strings.Contains(line, atlasAggregateCommandRoot) || json.Unmarshal([]byte(line), &got) != nil || got.JSONRPC != "2.0" || got.ID != id || len(got.Error) != 0 || got.Result.IsError != failure {
			t.Fatalf("aggregate MCP response: %s", line)
		}
		return got
	}
	send(mcpInitialize, 1, false)
	before := remoteTree(t, root)
	got := send(mcpReady+"\n"+candidateMCPCall(selected.Header().PackageUID, symbol, 2), 2, false)
	compareCandidateAtlasAggregateImpact(t, got.Result.Data, want, compiler)
	send(candidateMCPCall("example.unknown", symbol, 3), 3, true)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("aggregate MCP queries changed workspace")
	}
	writeWorkspaceFile(t, root, "libs/maker/src/maker.fy", strings.Replace(atlasAggregateCommandMaker, "pub struct Unused<T>{T value;}", "pub struct Unused<T>{T value;T other;}", 1))
	stale := remoteTree(t, root)
	send(candidateMCPCall(selected.Header().PackageUID, symbol, 4), 4, true)
	if !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatal("stale aggregate MCP request changed workspace")
	}
	_, current := candidateAtlasAggregateCommandOracle(t, root)
	after := remoteTree(t, root)
	want, err = current.Impact(selected.Header().PackageInstance, symbol, 8, 64)
	if err != nil {
		t.Fatal(err)
	}
	got = send(candidateMCPCall(selected.Header().PackageUID, symbol, 5), 5, false)
	compareCandidateAtlasAggregateImpact(t, got.Result.Data, want, compiler)
	if !reflect.DeepEqual(after, remoteTree(t, root)) {
		t.Fatal("fresh aggregate MCP request changed workspace")
	}
	if err := in.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil || diagnostic.Len() != 0 {
		t.Fatalf("aggregate MCP completion: %v %s", err, &diagnostic)
	}
}

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
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func candidateMCPArguments() []string {
	return []string{"experimental", "atlas-mcp", "--allow-impact", "--max-packages", "8", "--max-calls", "128", "--max-depth", "8", "--max-symbols", "64", "--max-output-bytes", "100000", "--max-requests", "10"}
}
func candidateMCPCall(uid, symbol string, id int) string {
	data, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": "atlas_impact", "arguments": map[string]string{"packageUid": uid, "symbol": symbol}}})
	return string(data)
}
func candidateMCPProcess(t *testing.T, image, root string, args []string, input string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(input)
	var out, diagnostic bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diagnostic
	err := cmd.Run()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, out.String(), diagnostic.String()
}

func TestCandidateMCPCommand(t *testing.T) {
	main := `module main;import project_mcp_cli as mcp;@effects(filesystem,environment,console) fn i32 main(){return mcp::arguments();}`
	candidateAtlasProbeSize(t, "project_mcp_cli", main, 2<<20, func(t *testing.T, image string) { verifyCandidateMCPCommand(t, image) })
}

func verifyCandidateMCPCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateMCPCommandWithHelp(t, image, candidateUsage())
}

func verifyCandidateMCPCommandWithHelp(t *testing.T, image, help string) {
	t.Helper()
	const source = `module main;import answer as answer;fn i32 leaf(){return answer::value();}fn i32 main(){return leaf();}`
	root := writeWorkspaceSessionFixture(t, source)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatal(code, &diagnostic)
	}
	index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	var selected atlasmap.Partition
	for _, p := range prepared.partitions {
		if p.Header().PackageUID == "example.session.answer" {
			selected = p
		}
	}
	symbol := selected.Symbols()[0].Visible
	want, err := index.Impact(selected.Header().PackageInstance, symbol, 8, 64)
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(binary))
	before := remoteTree(t, root)
	t.Setenv("PATH", "") // No Go child or build companion can supply the query.
	call := candidateMCPCall("example.session.answer", symbol, 2)
	input := mcpInitialize + "\n" + mcpReady + "\n" + call + "\n" + candidateMCPCall("example.unknown", symbol, 3) + "\n" + call + "\n"
	code, out, stderr := candidateMCPProcess(t, image, root, candidateMCPArguments(), input)
	if code != 0 || stderr != "" || strings.Contains(out, root) || strings.Contains(out, source) {
		t.Fatalf("MCP command: %d %s %s", code, out, stderr)
	}
	var replies []struct {
		Result struct {
			IsError bool                  `json:"isError"`
			Data    atlasmap.ImpactResult `json:"structuredContent"`
		} `json:"result"`
	}
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		var r struct {
			Result struct {
				IsError bool                  `json:"isError"`
				Data    atlasmap.ImpactResult `json:"structuredContent"`
			} `json:"result"`
		}
		if json.Unmarshal([]byte(line), &r) != nil {
			t.Fatal(line)
		}
		replies = append(replies, r)
	}
	if len(replies) != 4 || replies[1].Result.IsError || !replies[2].Result.IsError || replies[3].Result.IsError {
		t.Fatalf("request isolation/recovery: %s", out)
	}
	for _, r := range []atlasmap.ImpactResult{replies[1].Result.Data, replies[3].Result.Data} {
		if r.Schema != want.Schema || r.State != "partial" || !reflect.DeepEqual(r.Subject, want.Subject) || !reflect.DeepEqual(r.Nodes, want.Nodes) || !r.TraversalComplete || !reflect.DeepEqual(r.Omissions, want.Omissions) || len(r.Contexts) != len(want.Contexts) {
			t.Fatalf("impact mismatch: %+v / %+v", r, want)
		}
		for i, c := range r.Contexts {
			w := want.Contexts[i]
			if c.CompilerBuild != compiler || c.PackageSource != w.PackageSource || c.InterfaceHash != w.InterfaceHash || c.PackageInstance != w.PackageInstance {
				t.Fatal("independent source/image identities lost")
			}
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("MCP changed workspace tree")
	}
	args := candidateMCPArguments()
	args[len(args)-1] = "2"
	code, out, stderr = candidateMCPProcess(t, image, root, args, mcpInitialize+"\n"+mcpReady+"\n"+call+"\n")
	if code != 2 || len(mcpDocuments(t, out)) != 1 || !strings.Contains(stderr, "resource policy") {
		t.Fatalf("notification/message allowance: %d %s %s", code, out, stderr)
	}
	for _, args := range [][]string{{"experimental", "atlas-mcp", "--help"}, candidateMCPArguments()[0:3]} {
		code, out, _ := candidateMCPProcess(t, image, t.TempDir(), args, "")
		if len(args) == 3 && args[2] == "--help" {
			if code != 0 || out != help {
				t.Fatal("help read workspace")
			}
		} else if code != 2 || out != "" {
			t.Fatal("missing permission accepted")
		}
	}
	verifyCandidateMCPLive(t, image, root, source, call)
	verifyCandidateMCPBrokenOutput(t, image, root)
	t.Run("concrete", func(t *testing.T) { verifyCandidateMCPConcrete(t, image) })
}

func verifyCandidateMCPConcrete(t *testing.T, image string) {
	t.Helper()
	const uid = "example.session.app"
	root := candidateAtlasGenericProductionWorkspace(t, atlasGenericProductionUsedSource)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("concrete MCP oracle: %d %s", code, &diagnostic)
	}
	index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	var instance, symbol, goodID, foreignID string
	for _, p := range prepared.partitions {
		if p.Header().PackageUID != uid {
			continue
		}
		instance = p.Header().PackageInstance
		for _, s := range p.Symbols() {
			if s.Declaration.Name == "wrap" {
				symbol = s.Full
			}
		}
		for _, impl := range p.Specializations() {
			switch impl.Declaration.Name {
			case "wrap":
				goodID = impl.ID
			case "choose":
				foreignID = impl.ID
			}
		}
	}
	if instance == "" || symbol == "" || goodID == "" || foreignID == "" {
		t.Fatal("missing concrete MCP fixture")
	}
	good := symbol + "#" + goodID
	want, err := index.Impact(instance, good, 8, 64)
	if err != nil {
		t.Fatal(err)
	}

	input := strings.Join([]string{
		mcpInitialize, mcpReady,
		candidateMCPCall(uid, symbol+"#"+foreignID, 2),
		candidateMCPCall(uid, symbol+"#", 3),
		candidateMCPCall(uid, good, 4),
		"",
	}, "\n")
	before := remoteTree(t, root)
	code, out, stderr := candidateMCPProcess(t, image, root, candidateMCPArguments(), input)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if code != 0 || stderr != "" || len(lines) != 4 {
		t.Fatalf("concrete MCP: %d %s %s", code, out, stderr)
	}
	for i, line := range lines[1:] {
		var reply struct {
			ID     int `json:"id"`
			Result struct {
				IsError bool                  `json:"isError"`
				Data    atlasmap.ImpactResult `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &reply); err != nil || reply.ID != i+2 || reply.Result.IsError != (i < 2) {
			t.Fatalf("unexpected concrete MCP response: %s", line)
		}
		if i == 2 {
			got := reply.Result.Data
			if got.SubjectSpecialization != goodID || got.Subject != want.Subject || !reflect.DeepEqual(got.Nodes, want.Nodes) || got.TraversalComplete != want.TraversalComplete {
				t.Fatalf("concrete MCP impact differs: %s", line)
			}
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("concrete MCP changed workspace")
	}
}

func verifyCandidateMCPLive(t *testing.T, image, root, source, call string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, candidateMCPArguments()...)
	cmd.Dir = root
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = in.Close()
		_ = out.Close()
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	reader := bufio.NewReader(out)
	read := func(wantFailure bool) {
		t.Helper()
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("live MCP: %v %s", err, &diagnostic)
		}
		var response struct {
			Result struct {
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		if json.Unmarshal([]byte(line), &response) != nil || response.Result.IsError != wantFailure {
			t.Fatalf("live response: %s", line)
		}
	}
	if _, err = in.Write([]byte(mcpInitialize + "\n")); err != nil {
		t.Fatal(err)
	}
	read(false)
	if _, err = in.Write([]byte(mcpReady + "\n" + call + "\n")); err != nil {
		t.Fatal(err)
	}
	read(false)
	path := filepath.Join(root, "src", "main.fy")
	if err = os.WriteFile(path, []byte(source+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = in.Write([]byte(call + "\n")); err != nil {
		t.Fatal(err)
	}
	read(true)
	if err = os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = in.Write([]byte(call + "\n")); err != nil {
		t.Fatal(err)
	}
	read(false)
	_ = in.Close()
	if err = cmd.Wait(); err != nil || diagnostic.Len() != 0 {
		t.Fatalf("live completion: %v %s", err, &diagnostic)
	}
}

func verifyCandidateMCPBrokenOutput(t *testing.T, image, root string) {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = read.Close()
	defer write.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, candidateMCPArguments()...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(mcpInitialize + "\n")
	cmd.Stdout = write
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	err = cmd.Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 4 || !strings.Contains(diagnostic.String(), "write output failed") {
		t.Fatalf("broken output: %v %s", err, &diagnostic)
	}
}

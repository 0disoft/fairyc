package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const atlasSDKCommandSource = `module main;import std::num as n;import std::bytes as b;
fn i32 leaf(){let value=switch(b::copy(b"42")){case .ok(p):{p}case .err(_):{return 1;}};
return switch(n::parse_i64_decimal(b::view(ref value))){case .ok(p):{if(p==42){42}else{1}}case .err(_):{1}};}
fn i32 main(){return leaf();}`

func candidateAtlasSDKCLIProcess(t *testing.T, image, root string, args []string) (int, string, string) {
	t.Helper()
	return candidateAtlasSDKCLIProcessLimit(t, image, root, args, 3*time.Minute)
}

func candidateAtlasSDKCLIProcessLimit(t *testing.T, image, root string, args []string, limit time.Duration) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), limit)
	defer cancel()
	command := exec.CommandContext(ctx, image, args...)
	command.Dir = root
	var stdout, diagnostic bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &diagnostic
	err := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("bounded SDK CLI execution after %s (%v): %v", limit, args, ctx.Err())
	}
	code := 0
	if err != nil {
		if exited, ok := err.(*exec.ExitError); ok {
			code = exited.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, stdout.String(), diagnostic.String()
}

func TestCandidateAtlasSDKCommand(t *testing.T) {
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, verifyCandidateAtlasSDKCommand)
}

func verifyCandidateAtlasSDKCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateAtlasSDKCommandSource(t, image, atlasSDKCommandSource, "std::num", "parse_i64_decimal")
}

func verifyCandidateAtlasSDKCommandSource(t *testing.T, image, source, module, name string, subjects ...string) {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, source)
	prepare := func() (preparedAtlasWorkspace, *atlasmap.WorkspaceCallIndex) {
		t.Helper()
		var diagnostic bytes.Buffer
		p, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("SDK command oracle: %d %s", code, &diagnostic)
		}
		index, err := atlasmap.NewWorkspaceCallIndexV3(p.partitions, p.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 8192})
		if err != nil {
			t.Fatal(err)
		}
		return p, index
	}
	prepared, index := prepare()
	data, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(data))
	resolved := len(index.Calls())
	if resolved < 2 {
		t.Fatalf("missing SDK resolved-call oracle: %d", resolved)
	}
	t.Logf("SDK command resolved calls=%d", resolved)
	var sdk atlasmap.Partition
	var selected string
	query := func(p atlasmap.Partition, symbol string, current *atlasmap.WorkspaceCallIndex) atlasmap.ImpactResult {
		t.Helper()
		args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, symbol)...)
		args = replaceWorkspaceSessionArgument(args, "--max-calls", strconv.Itoa(len(current.Calls())))
		code, out, diagnostic := candidateAtlasSDKCLIProcess(t, image, root, args)
		if code != 0 || diagnostic != "" {
			t.Fatalf("SDK impact command: %d %s %s", code, out, diagnostic)
		}
		var got atlasmap.ImpactResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("SDK impact JSON: %v %s", err, out)
		}
		want, err := current.Impact(p.Header().PackageInstance, symbol, 8, 64)
		if err != nil {
			t.Fatal(err)
		}
		compareCandidateAtlasAggregateImpact(t, got, want, compiler)
		return got
	}
	t.Setenv("PATH", "")
	before := remoteTree(t, root)
	matched := map[string]bool{}
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if s.Declaration.Module == module && s.Declaration.Name == name {
				if selected != "" {
					t.Fatal("duplicate SDK subject")
				}
				sdk, selected = p, s.Visible
				query(p, s.Visible, index)
			} else if p.Header().PackageUID == "example.session.app" && (s.Declaration.Name == "leaf" || slices.Contains(subjects, s.Declaration.Name)) {
				matched[s.Declaration.Name] = true
				query(p, s.Full, index)
			}
		}
	}
	if selected == "" || len(matched) != len(subjects)+1 || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("SDK/root selection or read-only query guarantee", matched)
	}
	base := append([]string{"experimental", "atlas-impact"}, impactArgs(sdk.Header().PackageUID, selected)...)
	base = replaceWorkspaceSessionArgument(base, "--max-calls", strconv.Itoa(resolved))
	for _, limit := range [][2]string{{"--max-packages", "2"}, {"--max-calls", strconv.Itoa(resolved - 1)}} {
		code, out, diagnostic := candidateAtlasSDKCLIProcess(t, image, root, replaceWorkspaceSessionArgument(base, limit[0], limit[1]))
		if code != 2 || out != "" || diagnostic == "" {
			t.Fatalf("SDK %s cutoff: %d %q %q", limit[0], code, out, diagnostic)
		}
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("SDK cutoff changed workspace")
	}
	writeWorkspaceFile(t, root, "src/main.fy", source+"\n")
	stale := remoteTree(t, root)
	code, out, diagnostic := candidateAtlasSDKCLIProcess(t, image, root, base)
	if code != 7 || out != "" || diagnostic == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatalf("SDK stale source: %d %q %q", code, out, diagnostic)
	}
	_, current := prepare()
	before = remoteTree(t, root)
	query(sdk, selected, current)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("SDK relocked query changed workspace")
	}
}

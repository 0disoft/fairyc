package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/atlasmap"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCandidateAtlasCommand(t *testing.T) {
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment) fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, func(t *testing.T, image string) { verifyCandidateAtlasCommand(t, image) })
}
func verifyCandidateAtlasCommand(t *testing.T, image string) {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, `module main;import answer as answer;fn i32 leaf(){return answer::value();}fn i32 left(){return right()+leaf();}fn i32 right(){return left()+leaf()+leaf();}fn i32 add(i32 a,i32 b){return a+b;}fn i32 main(){return add(left(),right());}`)

	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("prepare: %d %s", code, &diagnostic)
	}
	index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	compilerBytes, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	compiler := fmt.Sprintf("fairyc.compiler-executable-sha256/v1:%x", sha256.Sum256(compilerBytes))
	lockBefore, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	treeBefore := remoteTree(t, root)
	for _, p := range prepared.partitions {
		for _, s := range p.Symbols() {
			if s.Declaration.Name != "value" && s.Declaration.Name != "add" {
				continue
			}
			for _, symbol := range []string{s.Visible, s.Full} {
				args := append([]string{"experimental", "atlas-impact"}, impactArgs(p.Header().PackageUID, symbol)...)
				if symbol == s.Full {
					args = append(args, "--locked")
				}
				code, out, stderr := fetchCLIProcess(t, image, root, args, "")
				if code != 0 || stderr != "" {
					t.Fatalf("impact: %d %q %q", code, out, stderr)
				}
				var got atlasmap.ImpactResult
				if err = json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("decode: %v %s", err, out)
				}
				want, err := index.Impact(p.Header().PackageInstance, symbol, 8, 64)
				if err != nil {
					t.Fatal(err)
				}
				if got.Schema != want.Schema || got.Scope != want.Scope || got.State != "partial" || !reflect.DeepEqual(got.Subject, want.Subject) || !reflect.DeepEqual(got.Nodes, want.Nodes) || got.TraversalComplete != want.TraversalComplete || !reflect.DeepEqual(got.Omissions, want.Omissions) {
					t.Fatalf("impact differs\ngot %s\nwant %+v", out, want)
				}
				if len(got.Contexts) != len(want.Contexts) {
					t.Fatal("context count")
				}
				for i, c := range got.Contexts {
					w := want.Contexts[i]
					if c.CompilerBuild != compiler || c.PackageInstance != w.PackageInstance || c.PackageSource != w.PackageSource || c.InterfaceHash != w.InterfaceHash || c.GenericInterfaceHash != w.GenericInterfaceHash {
						t.Fatalf("context differs: %+v / %+v", c, w)
					}
				}
			}
		}
	}
	if !reflect.DeepEqual(treeBefore, remoteTree(t, root)) {
		t.Fatal("Atlas changed workspace tree")
	}
	var selected atlasmap.Partition
	for _, p := range prepared.partitions {
		if p.Header().PackageUID == "example.session.answer" {
			selected = p
		}
	}
	symbol := selected.Symbols()[0].Visible
	base := append([]string{"experimental", "atlas-impact"}, impactArgs(selected.Header().PackageUID, symbol)...)
	for _, limits := range [][2]int{{0, 64}, {8, 1}} {
		args := append([]string{}, base...)
		for i := range args {
			if args[i] == "--max-depth" {
				args[i+1] = fmt.Sprint(limits[0])
			}
			if args[i] == "--max-symbols" {
				args[i+1] = fmt.Sprint(limits[1])
			}
		}
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if code != 0 || stderr != "" {
			t.Fatalf("cutoff %d %s %s", code, out, stderr)
		}
		var got atlasmap.ImpactResult
		if json.Unmarshal([]byte(out), &got) != nil {
			t.Fatal(out)
		}
		want, err := index.Impact(selected.Header().PackageInstance, symbol, limits[0], limits[1])
		if err != nil || !reflect.DeepEqual(got.Nodes, want.Nodes) || !reflect.DeepEqual(got.Omissions, want.Omissions) || got.TraversalComplete != want.TraversalComplete {
			t.Fatalf("cutoff differs %+v %+v %v", got, want, err)
		}
	}
	for _, change := range [][2]string{{"--package-uid", "example.unknown"}, {"--symbol", "fysym1-sha256-128-" + strings.Repeat("0", 32)}, {"--max-packages", "1"}, {"--max-calls", "1"}, {"--max-output-bytes", "1"}, {"--max-depth", "65"}} {
		args := append([]string{}, base...)
		for i := range args {
			if args[i] == change[0] {
				args[i+1] = change[1]
			}
		}
		code, out, stderr := fetchCLIProcess(t, image, root, args, "")
		if code != 2 || out != "" || stderr == "" {
			t.Fatalf("reject %v: %d %q %q", change, code, out, stderr)
		}
	}
	code, out, stderr := fetchCLIProcess(t, image, t.TempDir(), base, "")
	if code != 7 || out != "" || stderr == "" {
		t.Fatalf("missing lock: %d %q %q", code, out, stderr)
	}
	for _, args := range [][]string{{"experimental", "atlas-impact"}, {"experimental", "atlas-impact", "--bad"}, append(append([]string{}, base...), "--locked", "--locked")} {
		empty := t.TempDir()
		before := remoteTree(t, empty)
		code, out, stderr := fetchCLIProcess(t, image, empty, args, "")
		if code != 2 || out != "" || stderr == "" || !reflect.DeepEqual(before, remoteTree(t, empty)) {
			t.Fatalf("syntax: %d %q %q", code, out, stderr)
		}
	}
	code, _, stderr = fetchCLIProcess(t, image, root, base, "stdout")
	if code != 4 || stderr == "" {
		t.Fatalf("stdout failure: %d %q", code, stderr)
	}
	originalSource := mustReadAtlasFile(t, filepath.Join(root, "src/main.fy"))
	writeWorkspaceFile(t, root, "src/main.fy", string(originalSource)+"\n")
	stale := remoteTree(t, root)
	code, out, stderr = fetchCLIProcess(t, image, root, base, "")
	if code != 7 || out != "" || stderr == "" || !reflect.DeepEqual(stale, remoteTree(t, root)) {
		t.Fatalf("stale lock %d %q %q", code, out, stderr)
	}
	// Keep the initial authored bytes when restoring the fixture after a drift check.
	if err := os.WriteFile(filepath.Join(root, "src/main.fy"), originalSource, 0600); err != nil {
		t.Fatal(err)
	}

	lockAfter, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("query changed lock", err)
	}

}

func mustReadAtlasFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

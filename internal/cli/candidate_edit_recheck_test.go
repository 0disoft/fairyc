package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const candidateEditRecheckMain = `module main;import project_edit_preview as preview;
` + atlasReadInput + `
@effects(filesystem,environment,console) fn i32 main(){
 var u8[4096] a=[0;4096];var u8[4096] b=[0;4096];var u8[65] c=[0;65];
 let x=switch(process::arg(1,ref mut a)){case .ok(p):{p}case .err(_):{return 2;}};
 let y=switch(process::arg(2,ref mut b)){case .ok(p):{p}case .err(_):{return 2;}};
 let z=switch(process::arg(3,ref mut c)){case .ok(p):{p}case .err(_):{return 2;}};
 let ref u8[] uid=ref a;let ref u8[] symbol=ref b;let ref u8[] expected=ref c;
 let result=switch(preview::prepare(b".",uid[..x],symbol[..y],expected[..z],b"{return 1;}")){case .ok(p):{p}case .err(_):{return 2;}};
 if(!io::write_bytes(preview::fingerprint(ref result)).is_ok || !io::write("\n").is_ok){return 4;}
 let ignored=input();
 switch(preview::recheck(b".",uid[..x],symbol[..y],expected[..z],ref result)){case .ok(_):{return if(io::write("ok").is_ok){0}else{4};}case .err(p):{discard io::write_usize(p.stage as usize);discard io::write("/");discard io::write_usize(p.code as usize);return 2;}};
 return 2;
}`

// The ready line is a process barrier after preview and before any recheck.
// All mutations run in Go while the actual Fairy program waits for stdin EOF.
func candidateEditBarrier(t *testing.T, image, root string, args []string, mutate func(), wantCode int) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Dir = root
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
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
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		cancel()
		if !waited {
			_ = cmd.Wait()
		}
	}()
	reader := bufio.NewReader(out)
	ready, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("preview barrier: %v", err)
	}
	if want := editSnapshotDigest(t, root) + "\n"; ready != want {
		t.Fatalf("preview fingerprint %q; want %q", ready, want)
	}
	mutate()
	before := remoteTree(t, root)
	if err := in.Close(); err != nil {
		t.Fatal(err)
	}
	rest, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	code := 0
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			code = failure.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if code != wantCode || diagnostic.Len() != 0 {
		t.Fatalf("recheck: %d %q %q; want %d", code, rest, &diagnostic, wantCode)
	}
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("recheck changed source/lock or created files")
	}
	return string(rest)
}

func TestCandidateEditRecheck(t *testing.T) {
	candidateAtlasProbeSize(t, "project_edit_preview", candidateEditRecheckMain, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, scenario := range []string{"unchanged", "source-stale", "dependency-stale", "root-manifest", "member-manifest", "graph-lock", "dependency-relocked", "body-file"} {
			t.Run(scenario, func(t *testing.T) {
				root, args := editFixture(t)
				want := 2
				if scenario == "unchanged" || scenario == "body-file" {
					want = 0
				}
				text := candidateEditBarrier(t, image, root, []string{args[4], args[6], args[10]}, func() {
					if scenario == "unchanged" {
						return
					}
					path := map[string]string{"source-stale": "src/main.fy", "dependency-stale": "libs/answer/src/answer.fy", "dependency-relocked": "libs/answer/src/answer.fy", "root-manifest": "fairy.toml", "member-manifest": "libs/answer/fairy.toml", "graph-lock": "fairy.lock", "body-file": "edits/body.txt"}[scenario]
					absolute := filepath.Join(root, filepath.FromSlash(path))
					data, err := os.ReadFile(absolute)
					if err != nil {
						t.Fatal(err)
					}
					data = append(data, '\n')
					if scenario == "source-stale" {
						data = bytes.Replace(data, []byte("return 0;"), []byte("return 3;"), 1)
					}
					if strings.HasPrefix(scenario, "dependency-") {
						data = bytes.Replace(data, []byte("42"), []byte("43"), 1)
					}
					if err := os.WriteFile(absolute, data, 0600); err != nil {
						t.Fatal(err)
					}
					if scenario == "dependency-relocked" {
						var diagnostic bytes.Buffer
						if _, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3); code != 0 {
							t.Fatal(code, &diagnostic)
						}
					}
				}, want)
				if want == 0 && text != "ok" {
					t.Fatal(text)
				}
				if want != 0 && (text == "" || !strings.Contains(text, "/")) {
					t.Fatalf("missing refusal stage: %q", text)
				}
			})
		}
	})
}

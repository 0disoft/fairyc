package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const candidateEditSnapshotMain = `module main;import project_edit_snapshot as snapshot;import project_module_visibility as visibility;import project_source_set as sources;import windows_rooted_file as host;import std::bytes as bytes;import std::io as io;import std::process as process;
@effects(filesystem,console) fn i32 main(){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
 let resolved=switch(visibility::load(b".",ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return 2;}};
 let root=switch(host::open(b".")){case .ok(p):{p}case .err(_):{return 2;}};
 let result=switch(if(process::arg_count()==2){snapshot::capture(ref root,b".")}else{snapshot::from_resolved(ref root,ref resolved)}){case .ok(p):{p}case .err(_):{return 2;}};
 return if(io::write_bytes(snapshot::digest(ref result)).is_ok){0}else{4};
}`

// Independent construction of the Go edit owner's exact fingerprint bytes.
func editSnapshotDigest(t *testing.T, root string) string {
	t.Helper()
	w, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{}
	read := func(path string) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		inputs[path] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	for _, p := range w.Packages {
		if p.SourceKind != workspaceloader.SourceKindLocal {
			continue
		}
		read(filepath.ToSlash(filepath.Join(p.Path, "fairy.toml")))
		for _, s := range p.Sources {
			relative, err := filepath.Rel(root, filepath.Join(p.SourceRoot, filepath.FromSlash(s.Path)))
			if err != nil {
				t.Fatal(err)
			}
			read(filepath.ToSlash(relative))
		}
	}
	read("fairy.lock")
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	hash := sha256.New()
	for _, key := range keys {
		hash.Write([]byte(key + "\x00" + inputs[key] + "\x00"))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func TestCandidateEditSnapshot(t *testing.T) {
	candidateAtlasProbeSize(t, "project_edit_snapshot", candidateEditSnapshotMain, 512<<10, func(t *testing.T, image string) {
		root, _ := editFixture(t)
		run := func() string {
			t.Helper()
			want := editSnapshotDigest(t, root)
			before := remoteTree(t, root)
			code, out, stderr := fetchCLIProcess(t, image, root, nil, "")
			if code != 0 || out != want || stderr != "" {
				t.Fatalf("snapshot: %d %q %q; want %s", code, out, stderr, want)
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("snapshot created or rewrote an input")
			}
			return out
		}
		first := run()
		code, out, diagnostic := fetchCLIProcess(t, image, root, []string{"locked"}, "")
		if code != 0 || out != first || diagnostic != "" {
			t.Fatalf("fresh locked snapshot: %d %q %q", code, out, diagnostic)
		}
		sourcePath := filepath.Join(root, "src", "main.fy")
		original, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(string(original), "return 0;", "return 3;", 1)
		if changed == string(original) {
			t.Fatal("stale-source control did not change the input")
		}
		writeWorkspaceFile(t, root, "src/main.fy", changed)
		before := remoteTree(t, root)
		code, out, diagnostic = fetchCLIProcess(t, image, root, []string{"locked"}, "")
		if code != 2 || out != "" || diagnostic != "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("fresh snapshot must reject stale lock: %d %q %q", code, out, diagnostic)
		}
		writeWorkspaceFile(t, root, "src/main.fy", string(original))
		code, out, diagnostic = fetchCLIProcess(t, image, root, []string{"locked"}, "")
		if code != 0 || out != first || diagnostic != "" {
			t.Fatalf("restored locked snapshot: %d %q %q", code, out, diagnostic)
		}
		writeWorkspaceFile(t, root, "edits/body.txt", "ignored outside production source")
		if next := run(); next != first {
			t.Fatal("unrelated body input changed workspace fingerprint")
		}
		for _, path := range []string{"fairy.toml", "libs/answer/fairy.toml", "fairy.lock"} {
			absolute := filepath.Join(root, filepath.FromSlash(path))
			data, err := os.ReadFile(absolute)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(absolute, append(data, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			if next := run(); next == first {
				t.Fatal("raw metadata edit was not fingerprinted:", path)
			} else {
				first = next
			}
		}
		// Exercise sorted records preceding the separately merged graph lock.
		if err := os.Rename(filepath.Join(root, "libs", "answer"), filepath.Join(root, "aaa")); err != nil {
			t.Fatal(err)
		}
		manifest, err := os.ReadFile(filepath.Join(root, "fairy.toml"))
		if err != nil {
			t.Fatal(err)
		}
		writeWorkspaceFile(t, root, "fairy.toml", strings.ReplaceAll(string(manifest), "libs/answer", "aaa"))
		if next := run(); next == first {
			t.Fatal("changed member path did not alter the fingerprint")
		}
	})
}

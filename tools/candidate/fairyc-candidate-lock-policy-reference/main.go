// Fixture/oracle only: invoke the owning stable CLI, not a second lock policy.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func write(root, leaf string, b []byte) {
	p := filepath.Join(root, filepath.FromSlash(leaf))
	must(os.MkdirAll(filepath.Dir(p), 0700))
	must(os.WriteFile(p, b, 0600))
}
func canonical(root string) []byte {
	workspace, e := workspaceloader.Load(root)
	must(e)
	doc, e := lockfile.FromWorkspace(workspace)
	must(e)
	b, e := lockfile.Marshal(doc)
	must(e)
	return b
}
func manifest(uid, name, extra string) []byte {
	return []byte(fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid='%s'\nname='%s'\nversion='1.0.0'\nlanguage='%s'\nsource='src'\n%s", uid, name, sdkstdlib.LanguageRevision, extra))
}

type row struct {
	Name      string `json:"name"`
	Locked    bool   `json:"locked"`
	GoExit    int    `json:"goExit"`
	State     uint64 `json:"state"`
	Phase     uint64 `json:"phase"`
	Stage     uint64 `json:"stage"`
	Kind      uint64 `json:"kind"`
	Before    []byte `json:"before"`
	After     []byte `json:"after"`
	Files     uint64 `json:"files"`
	Bytes     uint64 `json:"bytes"`
	Unchanged bool   `json:"unchanged"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: lock-policy-reference OUTPUT")
	}
	root, e := filepath.Abs(os.Args[1])
	must(e)
	workspaceRoot, e := filepath.Abs(".")
	must(e)
	must(os.MkdirAll(root, 0700))
	var rows []row
	for _, spec := range []struct {
		name, setup string
		locked      bool
	}{
		{"Current", "current", false}, {"LockedCurrent", "current", true}, {"Missing", "missing", false}, {"LockedMissing", "missing", true},
		{"Reordered", "reordered", false}, {"LockedReordered", "reordered", true}, {"Malformed", "malformed", false}, {"LockedMalformed", "malformed", true},
		{"Empty", "empty", false}, {"LockedEmpty", "empty", true}, {"Stale", "stale", false}, {"LockedStale", "stale", true},
		{"Directory", "directory", false}, {"LockedDirectory", "directory", true}, {"SDK", "sdk", false}, {"LockedSDK", "sdk-current", true},
		{"DependencyStale", "dependency-stale", false}, {"LockedDependencyStale", "dependency-stale", true},
		{"Manifest", "manifest", false}, {"LockedManifest", "manifest", true}, {"Syntax", "syntax", false}, {"LockedSyntax", "syntax", true},
		{"Semantic", "semantic", false}, {"Import", "import", false},
	} {
		dir := filepath.Join(root, spec.name)
		must(os.MkdirAll(dir, 0700))
		source := []byte("module main;\nfn i32 answer(){return 1;}\n")
		extra := ""
		var child []byte
		if strings.HasPrefix(spec.setup, "sdk") {
			source = []byte("module main;\nimport std::bytes as bytes;\nfn i32 answer(){return 1;}\n")
		}
		if spec.setup == "dependency-stale" {
			extra = "[workspace]\nmembers=['lib']\n[dependencies.net]\npath='lib'\nversion='1.0.0'\nfeatures=[]\n"
			source = []byte("module main;\nimport net as n;\nfn i32 answer(){return n::value();}\n")
			child = []byte("module net;\npub fn i32 value(){return 1;}\n")
			write(dir, "lib/fairy.toml", manifest("example.policy.lib", "net", ""))
			write(dir, "lib/src/net.fy", child)
		}
		write(dir, "fairy.toml", manifest("example.policy", "policy", extra))
		write(dir, "src/main.fy", source)
		before := canonical(dir)
		kind := uint64(1)
		phase, stage := uint64(0), uint64(0)
		switch spec.setup {
		case "missing", "sdk":
			kind = 0
			before = nil
		case "reordered":
			var fields map[string]any
			must(json.Unmarshal(before, &fields))
			before, e = json.MarshalIndent(fields, "", "    ")
			must(e)
		case "malformed":
			before = []byte("{")
		case "empty":
			before = []byte{}
		case "stale":
			source = append(source, []byte("fn i32 changed(){return 2;}\n")...)
			write(dir, "src/main.fy", source)
		case "dependency-stale":
			child = append(child, []byte("pub fn i32 changed(){return 2;}\n")...)
			write(dir, "lib/src/net.fy", child)
		case "directory":
			kind = 2
			before = nil
			write(dir, "fairy.lock/sentinel", []byte("DIRECTORY"))
		case "manifest":
			write(dir, "fairy.toml", []byte("schema='invalid'\n"))
			phase = 1
		case "syntax":
			source = []byte("module main;\nfn i32 broken(){\n")
			write(dir, "src/main.fy", source)
			phase = 1
		case "semantic":
			source = []byte("module main;\nfn i32 answer(){return true;}\n")
			write(dir, "src/main.fy", source)
		case "import":
			source = []byte("module main;\nimport missing as x;\nfn i32 answer(){return 1;}\n")
			write(dir, "src/main.fy", source)
			phase = 1
		}
		live := filepath.Join(dir, lockfile.FileName)
		if kind == 1 {
			write(dir, lockfile.FileName, before)
			must(os.Chtimes(live, time.Unix(1700000000, 0), time.Unix(1700000000, 0)))
		}
		args := []string{"check"}
		if spec.locked {
			args = append(args, "--locked")
		}
		var stdout, stderr bytes.Buffer
		code := cli.Run(args, dir, &stdout, &stderr)
		var after []byte
		info, e := os.Stat(live)
		if e == nil && info.Mode().IsRegular() {
			after, e = os.ReadFile(live)
			must(e)
		} else if kind != 2 && !os.IsNotExist(e) {
			panic(e)
		}
		state := uint64(0)
		if code == 0 {
			if strings.HasSuffix(stdout.String(), "lock updated\n") {
				state = 2
			} else if strings.HasSuffix(stdout.String(), "lock unchanged\n") {
				state = 1
			} else if strings.HasSuffix(stdout.String(), "lock verified\n") {
				state = 3
			} else {
				panic("unrecognized CLI lock state")
			}
		}
		if phase == 0 && state == 0 {
			if !spec.locked && kind != 2 && !bytes.Equal(before, after) {
				state = 2
			} else {
				phase = 2
				if spec.locked {
					stage = 3
					if spec.setup == "stale" || spec.setup == "dependency-stale" {
						stage = 4
					}
				} else {
					stage = 6
				}
			}
		}
		if state == 2 {
			doc, e := lockfile.Parse(after)
			must(e)
			expected, e := lockfile.Marshal(doc)
			must(e)
			if !bytes.Equal(expected, after) {
				panic("noncanonical CLI update")
			}
		}
		unchanged := state != 2
		if unchanged && !bytes.Equal(before, after) && kind == 1 {
			panic("failed/readonly Go invocation changed lock")
		}
		if kind == 1 {
			write(dir, lockfile.FileName, before)
			must(os.Chtimes(live, time.Unix(1700000000, 0), time.Unix(1700000000, 0)))
		} else if kind == 0 && e == nil {
			must(os.Remove(live))
		}
		files := uint64(1)
		if len(child) > 0 {
			files++
		}
		rows = append(rows, row{spec.name, spec.locked, code, state, phase, stage, kind, before, after, files, uint64(len(source) + len(child)), unchanged, stdout.String(), stderr.String()})
	}
	// Private SDK rejection is covered directly by the candidate API; this fixture
	// uses the same trusted source set with vector-candidate selection enabled.
	private := filepath.Join(root, "Private")
	write(private, "fairy.toml", manifest("example.private", "private", ""))
	write(private, "src/main.fy", []byte("module main;\nimport std::vec as v;\nfn i32 value(){return 1;}\n"))
	write(private, lockfile.FileName, []byte("OLD"))
	write(root, "root.bin", []byte(root))
	var packet bytes.Buffer
	packet.WriteString("FYLPOL01")
	number := func(v uint64) { must(binary.Write(&packet, binary.BigEndian, v)) }
	field := func(b []byte) { number(uint64(len(b))); packet.Write(b) }
	number(uint64(len(rows)))
	var paths []string
	for _, r := range rows {
		field([]byte(r.Name))
		if r.Locked {
			number(1)
		} else {
			number(0)
		}
		for _, v := range []uint64{r.State, r.Phase, r.Stage, r.Kind, r.Files, r.Bytes} {
			number(v)
		}
		field(r.Before)
		field(r.After)
		if r.Unchanged {
			p, e := filepath.Rel(workspaceRoot, filepath.Join(root, r.Name, lockfile.FileName))
			must(e)
			paths = append(paths, filepath.ToSlash(p))
		}
	}
	write(root, "policies.bin", packet.Bytes())
	b, e := json.MarshalIndent(rows, "", "  ")
	must(e)
	write(root, "observations.json", append(b, '\n'))
	b, e = json.MarshalIndent(paths, "", "  ")
	must(e)
	write(root, "unchanged-paths.json", append(b, '\n'))
	fmt.Printf("stable Go CLI observations passed: %d cases\n", len(rows))
}

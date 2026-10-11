// Selection observations from the full owning Go workspace loader.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type dep struct {
	binding, path string
	features      []string
}

func write(root, path, body string) {
	p := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		panic(err)
	}
}
func manifest(uid string, members []string, deps []dep) string {
	body := fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid=%q\nname='selection'\nversion='1.0.0'\nlanguage=%q\nsource='src'\n", uid, sdkstdlib.LanguageRevision)
	if members != nil {
		raw, _ := json.Marshal(members)
		body += "[workspace]\nmembers=" + string(raw) + "\n"
	}
	for _, d := range deps {
		raw, _ := json.Marshal(d.features)
		body += fmt.Sprintf("[dependencies.%s]\npath=%q\nversion='1.0.0'\nfeatures=%s\n", d.binding, d.path, raw)
	}
	return body
}
func main() {
	if len(os.Args) != 2 {
		panic("expected output directory")
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		panic(err)
	}
	type fixture struct {
		name, main, child string
		deps              []dep
		candidate         bool
	}
	shared := []dep{{"shared", "a", []string{}}}
	fixtures := []fixture{
		{"None", "", "", nil, false},
		{"CandidateUnused", "", "", nil, true},
		{"Core", "import std::io as io;", "", nil, false},
		{"AliasCore", "import std::io as first; import std::io as second;", "", nil, false},
		{"Vector", "import std::vec as vector;", "", nil, false},
		{"CandidateCore", "import std::io as io;", "", nil, true},
		{"ChildOnly", "import lib as lib;", "import std::io as io;", shared, false},
		{"Both", "import std::io as io; import lib as lib;", "import std::io as io;", shared, false},
		{"ChildVector", "import lib as lib;", "import std::vec as vector;", shared, false},
		{"UnreachableVector", "", "import std::vec as vector;", nil, false},
		{"FeatureSplit", "", "import std::io as io;", []dep{{"alpha", "a", []string{"alpha"}}, {"beta", "a", []string{"beta"}}}, false},
		{"InstanceAliases", "", "import std::io as io;", []dep{{"first", "a", []string{"alpha"}}, {"twin", "a", []string{"alpha"}}}, false},
		{"LongTrivia", "import std /*" + strings.Repeat(" ", 70000) + "*/ :: io as io;", "", nil, false},
		{"SimilarNamespaces", "import stdx as x; import foo::std as other;", "", nil, false},
	}
	var observations []map[string]any
	for _, f := range fixtures {
		dir := filepath.Join(root, f.name)
		members := []string(nil)
		if f.child != "" {
			members = []string{"a"}
		}
		write(dir, "fairy.toml", manifest("example.root", members, f.deps))
		write(dir, "src/main.fy", "module main;\n"+f.main+"\n")
		if f.child != "" {
			write(dir, "a/fairy.toml", manifest("example.a", nil, nil))
			write(dir, "a/src/lib.fy", "module lib;\n"+f.child+"\n")
		}
		if f.name == "SimilarNamespaces" {
			write(dir, "src/stdx.fy", "module stdx;\n")
			write(dir, "src/foo/std.fy", "module foo::std;\n")
		}
		load := workspaceloader.Load
		if f.candidate {
			load = workspaceloader.LoadVectorCandidate
		}
		loaded, err := load(dir)
		if err != nil {
			panic(err)
		}
		profile := 0
		for _, p := range loaded.Packages {
			if p.SourceKind == workspaceloader.SourceKindSDK {
				profile = 1
				if p.SDKIdentity == sdkstdlib.VectorCandidateIdentity {
					profile = 3
				} else if len(p.Sources) == 8 {
					profile = 2
				}
			}
		}
		var record strings.Builder
		fmt.Fprintln(&record, profile)
		var owners []string
		for _, node := range loaded.Graph.Nodes {
			for _, edge := range loaded.Graph.Dependencies {
				if edge.Owner == node.ID && edge.Binding == "stdlib" {
					id, err := packagegraph.ComputeInstanceID(node.Identity)
					if err != nil {
						panic(err)
					}
					owners = append(owners, id.String())
					fmt.Fprintln(&record, id.String())
				}
			}
		}
		write(dir, "selection.txt", record.String())
		write(dir, "candidate.txt", fmt.Sprint(f.candidate))
		observations = append(observations, map[string]any{"fixture": f.name, "profile": profile, "owners": owners, "candidate": f.candidate})
	}
	data, err := json.MarshalIndent(observations, "", "  ")
	if err != nil {
		panic(err)
	}
	write(root, "reference.json", string(data)+"\n")
	write(root, "root.bin", filepath.ToSlash(root))
	fmt.Println("PASS: 14 full Go-loader SDK selection fixtures")
}

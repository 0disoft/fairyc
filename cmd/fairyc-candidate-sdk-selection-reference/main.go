// Selection observations from the full owning Go workspace loader.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/lockfile"
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
func lockObservation(document lockfile.Document) []byte {
	var out bytes.Buffer
	writeNumber := func(value int) {
		var data [8]byte
		binary.BigEndian.PutUint64(data[:], uint64(value))
		out.Write(data[:])
	}
	writeText := func(value string) { writeNumber(len(value)); out.WriteString(value) }
	for _, value := range []string{document.Schema, document.PackageInstanceAlgorithm, document.SourceDigestAlgorithm, document.Language, document.Root} {
		writeText(value)
	}
	writeNumber(len(document.Packages))
	for _, p := range document.Packages {
		for _, value := range []string{p.ID, p.Role, p.PackageUID, p.SourceKind, p.LocalPath, p.SDKIdentity, p.RuntimeABI, p.SourceIdentity, p.SourceDigest, p.Version} {
			writeText(value)
		}
		writeNumber(len(p.Features))
		for _, feature := range p.Features {
			writeText(feature)
		}
	}
	writeNumber(len(document.Dependencies))
	for _, e := range document.Dependencies {
		writeText(e.Owner)
		writeText(e.Binding)
		writeText(e.Target)
	}
	return out.Bytes()
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
		ids := map[string]string{}
		var nodes, edges, lookups strings.Builder
		for _, n := range loaded.Graph.Nodes {
			id, err := packagegraph.ComputeInstanceID(n.Identity)
			if err != nil {
				panic(err)
			}
			ids[n.ID] = id.String()
			fmt.Fprintf(&nodes, "%s\n%s\n", id.String(), n.Role)
		}
		for _, n := range loaded.Graph.Nodes {
			for _, e := range loaded.Graph.Dependencies {
				if e.Owner == n.ID {
					fmt.Fprintf(&edges, "%s\n%s\n%s\n", ids[e.Owner], e.Binding, ids[e.Target])
				}
			}
		}
		ordinal := 0
		for _, n := range loaded.Graph.Nodes {
			for _, m := range n.Modules {
				for _, imported := range m.Imports {
					l := loaded.Resolution.Lookups[ordinal]
					if l.From != n.ID || l.ImportPath != imported || len(l.ProviderMembers) != 1 {
						panic("unexpected owning lookup order")
					}
					fmt.Fprintf(&lookups, "%s\n%s\n%s\n%s\n%s\n", ids[n.ID], m.Path, imported, ids[l.ProviderMembers[0]], l.ModulePath)
					ordinal++
				}
			}
		}
		fileCount, byteCount := 0, 0
		for _, p := range loaded.Packages {
			if p.SourceKind == workspaceloader.SourceKindLocal {
				fileCount += len(p.Sources)
				for _, s := range p.Sources {
					byteCount += len(s.Bytes)
				}
			} else {
				for _, s := range p.Sources {
					write(root, s.Path, string(s.Bytes))
				}
			}
		}
		var counts [40]byte
		for index, value := range []int{fileCount, byteCount, len(loaded.Graph.Nodes), len(loaded.Graph.Dependencies), len(loaded.Resolution.Lookups)} {
			binary.BigEndian.PutUint64(counts[index*8:], uint64(value))
		}
		write(dir, "counts.bin", string(counts[:]))
		write(dir, "nodes.txt", nodes.String())
		write(dir, "edges.txt", edges.String())
		write(dir, "lookups.txt", lookups.String())
		document, lockErr := lockfile.FromWorkspace(loaded)
		if lockErr != nil {
			if !f.candidate || profile != 3 || !strings.Contains(lockErr.Error(), "contradictory SDK source origin") {
				panic(lockErr)
			}
			write(dir, "lock-accepted.txt", "false")
		} else {
			canonical, err := lockfile.Marshal(document)
			if err != nil {
				panic(err)
			}
			write(dir, "lock-accepted.txt", "true")
			write(dir, "lock.json", string(canonical))
			write(dir, "projection.bin", string(lockObservation(document)))
		}
		observations = append(observations, map[string]any{"fixture": f.name, "profile": profile, "owners": owners, "candidate": f.candidate})
	}
	data, err := json.MarshalIndent(observations, "", "  ")
	if err != nil {
		panic(err)
	}
	write(root, "reference.json", string(data)+"\n")
	write(root, "root.bin", filepath.ToSlash(root))
	for _, name := range []string{"UnknownStd", "BareStd"} {
		dir := filepath.Join(root, name)
		imported := "std::missing"
		if name == "BareStd" {
			imported = "std"
		}
		write(dir, "fairy.toml", manifest("example.root", nil, nil))
		write(dir, "src/main.fy", "module main;\nimport "+imported+" as sdk;\n")
		_, err := workspaceloader.Load(dir)
		if err == nil {
			panic("unknown SDK import accepted")
		}
		rejection, ok := err.(*workspaceloader.LoadError)
		if !ok || rejection.Reason != workspaceloader.ReasonPackageGraph || !strings.Contains(rejection.Detail, packagegraph.ReasonMissingVisibleModule) {
			panic(err)
		}
		write(dir, "path.txt", imported)
		write(dir, "reason.txt", packagegraph.ReasonMissingVisibleModule)
	}
	fmt.Println("PASS: 14 full Go-loader SDK selection fixtures")
}

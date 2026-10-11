// Observe physical lock reads and exact-current matching through owning Go APIs.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func within(root, path string) {
	r, e := filepath.Rel(root, path)
	if e != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		panic("fixture path escaped root")
	}
}
func write(root, path string, data []byte) {
	within(root, path)
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		panic(e)
	}
	if e := os.WriteFile(path, data, 0600); e != nil {
		panic(e)
	}
}
func copyTree(root, from, to string) {
	within(root, from)
	within(root, to)
	e := filepath.WalkDir(from, func(path string, entry fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		relative, e := filepath.Rel(from, path)
		if e != nil {
			return e
		}
		target := filepath.Join(to, relative)
		within(root, target)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected nonregular fixture: %s", path)
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		return os.WriteFile(target, data, 0600)
	})
	if e != nil {
		panic(e)
	}
}
func replace(root, dir, leaf, old, next string) {
	path := filepath.Join(dir, filepath.FromSlash(leaf))
	data, e := os.ReadFile(path)
	if e != nil {
		panic(e)
	}
	if !bytes.Contains(data, []byte(old)) {
		panic("missing mutation target")
	}
	write(root, path, bytes.Replace(data, []byte(old), []byte(next), 1))
}

type record struct {
	Path      string `json:"path"`
	Candidate bool   `json:"candidate"`
	Matches   bool   `json:"matches"`
}

func reidentify(d *lockfile.Document) {
	ids := map[string]string{}
	for i := range d.Packages {
		p := &d.Packages[i]
		old := p.ID
		id, e := packagegraph.ComputeInstanceID(packagegraph.Identity{PackageUID: p.PackageUID, SourceIdentity: p.SourceIdentity, VersionIdentity: p.Version, Features: p.Features})
		if e != nil {
			panic(e)
		}
		p.ID = id.String()
		ids[old] = p.ID
	}
	if next, ok := ids[d.Root]; ok {
		d.Root = next
	}
	for i := range d.Dependencies {
		e := &d.Dependencies[i]
		if next, ok := ids[e.Owner]; ok {
			e.Owner = next
		}
		if next, ok := ids[e.Target]; ok {
			e.Target = next
		}
	}
	slices.SortFunc(d.Packages, func(a, b lockfile.PackageRecord) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(d.Dependencies, func(a, b lockfile.DependencyRecord) int {
		if v := strings.Compare(a.Owner, b.Owner); v != 0 {
			return v
		}
		if v := strings.Compare(a.Binding, b.Binding); v != 0 {
			return v
		}
		return strings.Compare(a.Target, b.Target)
	})
}
func main() {
	if len(os.Args) == 3 && os.Args[1] == "--read-reparse" {
		if _, _, err := lockfile.Read(os.Args[2]); err == nil {
			panic("owning Go read unexpectedly accepted reparse fixture")
		} else {
			fmt.Printf("Go rejected reparse lock: %v\n", err)
		}
		return
	}
	if len(os.Args) != 2 {
		panic("usage: fairyc-candidate-lock-match-reference GENERATED_SDK_FIXTURES")
	}
	root, e := filepath.Abs(os.Args[1])
	if e != nil {
		panic(e)
	}
	var records []record
	observe := func(name string, candidate bool) {
		dir := filepath.Join(root, filepath.FromSlash(name))
		loader := workspaceloader.Load
		if candidate {
			loader = workspaceloader.LoadVectorCandidate
		}
		workspace, e := loader(dir)
		if e != nil {
			panic(e)
		}
		document, _, e := lockfile.Read(filepath.Join(dir, lockfile.FileName))
		if e != nil {
			panic(e)
		}
		records = append(records, record{name, candidate, lockfile.MatchesWorkspace(document, workspace) == nil})
	}
	for _, name := range []string{"None", "CandidateUnused", "Core", "AliasCore", "Vector", "ChildOnly", "Both", "ChildVector", "UnreachableVector", "FeatureSplit", "InstanceAliases", "LongTrivia", "SimilarNamespaces"} {
		dir := filepath.Join(root, name)
		input, e := os.ReadFile(filepath.Join(dir, "lock.json"))
		if e != nil {
			panic(e)
		}
		write(root, filepath.Join(dir, lockfile.FileName), input)
		observe(name, name == "CandidateUnused")
	}
	for _, name := range []string{"None", "Core", "FeatureSplit"} {
		targetName := "reordered/" + name
		target := filepath.Join(root, filepath.FromSlash(targetName))
		copyTree(root, filepath.Join(root, name), target)
		input, e := os.ReadFile(filepath.Join(target, lockfile.FileName))
		if e != nil {
			panic(e)
		}
		var fields map[string]any
		if e := json.Unmarshal(input, &fields); e != nil {
			panic(e)
		}
		reordered, e := json.MarshalIndent(fields, "", "    ")
		if e != nil {
			panic(e)
		}
		write(root, filepath.Join(target, lockfile.FileName), reordered)
		observe(targetName, false)
	}
	for _, f := range []struct{ name, base string }{{"Source", "None"}, {"Version", "None"}, {"UID", "None"}, {"Name", "None"}, {"Profile", "Core"}, {"Unreachable", "UnreachableVector"}, {"Features", "FeatureSplit"}, {"Binding", "InstanceAliases"}, {"LocalPath", "Both"}} {
		name := "mutations/" + f.name
		dir := filepath.Join(root, filepath.FromSlash(name))
		copyTree(root, filepath.Join(root, f.base), dir)
		switch f.name {
		case "Source":
			path := filepath.Join(dir, "src", "main.fy")
			b, e := os.ReadFile(path)
			if e != nil {
				panic(e)
			}
			write(root, path, append(b, []byte("fn i32 changed(){return 1;}\n")...))
		case "Version":
			replace(root, dir, "fairy.toml", "version='1.0.0'", "version='2.0.0'")
		case "UID":
			replace(root, dir, "fairy.toml", "example.root", "example.changed")
		case "Name":
			replace(root, dir, "fairy.toml", "name='selection'", "name='cosmetic'")
		case "Profile":
			write(root, filepath.Join(dir, "src", "main.fy"), []byte("module main;\nimport std::vec as vector;\n"))
		case "Unreachable":
			path := filepath.Join(dir, "a", "src", "lib.fy")
			b, e := os.ReadFile(path)
			if e != nil {
				panic(e)
			}
			write(root, path, append(b, []byte("fn i32 changed(){return 1;}\n")...))
		case "Features":
			replace(root, dir, "fairy.toml", "features=[\"alpha\"]", "features=[\"alpha\",\"zz-extra\"]")
		case "Binding":
			replace(root, dir, "fairy.toml", "dependencies.first", "dependencies.changed")
		case "LocalPath":
			from := filepath.Join(dir, "a")
			to := filepath.Join(dir, "b")
			within(root, from)
			within(root, to)
			if e := os.Rename(from, to); e != nil {
				panic(e)
			}
			replace(root, dir, "fairy.toml", "members=[\"a\"]", "members=[\"b\"]")
			replace(root, dir, "fairy.toml", "path=\"a\"", "path=\"b\"")
		}
		observe(name, false)
	}
	// A coherent but changed SDK source digest is readable and must still be stale.
	name := "mutations/SDKDigest"
	dir := filepath.Join(root, filepath.FromSlash(name))
	copyTree(root, filepath.Join(root, "Core"), dir)
	input, e := os.ReadFile(filepath.Join(dir, lockfile.FileName))
	if e != nil {
		panic(e)
	}
	var doc lockfile.Document
	if e := json.Unmarshal(input, &doc); e != nil {
		panic(e)
	}
	for i := range doc.Packages {
		p := &doc.Packages[i]
		if p.SourceKind != "sdk" {
			continue
		}
		old := p.SourceDigest
		next := "fysrc1-sha256-" + strings.Repeat("0", 64)
		p.SourceDigest = next
		p.SourceIdentity = strings.Replace(p.SourceIdentity, old, next, 1)
	}
	// Reuse the owning encoder for ID refresh, rather than hand-hashing a tuple.
	reidentify(&doc)
	canonical, e := lockfile.Marshal(doc)
	if e != nil {
		panic(e)
	}
	write(root, filepath.Join(dir, lockfile.FileName), canonical)
	observe(name, false)
	for _, fault := range []string{"Missing", "Empty", "Malformed", "NullDependencies", "Directory", "Oversized", "Reparse"} {
		dir := filepath.Join(root, "faults", fault)
		if e := os.MkdirAll(dir, 0700); e != nil {
			panic(e)
		}
		path := filepath.Join(dir, lockfile.FileName)
		switch fault {
		case "Empty":
			write(root, path, nil)
		case "Malformed":
			write(root, path, []byte("{"))
		case "NullDependencies":
			b, e := os.ReadFile(filepath.Join(root, "None", lockfile.FileName))
			if e != nil {
				panic(e)
			}
			write(root, path, bytes.Replace(b, []byte(`"dependencies": []`), []byte(`"dependencies": null`), 1))
		case "Directory":
			if e := os.Mkdir(path, 0700); e != nil {
				panic(e)
			}
		case "Oversized":
			f, e := os.Create(path)
			if e != nil {
				panic(e)
			}
			if e := f.Truncate(lockfile.MaxBytes + 1); e != nil {
				panic(e)
			}
			if e := f.Close(); e != nil {
				panic(e)
			}
		}
		if fault != "Reparse" {
			if _, _, e := lockfile.Read(path); e == nil {
				panic("owning Go read unexpectedly accepted " + fault)
			}
		}
	}
	var out bytes.Buffer
	out.WriteString("FYLMATCH")
	number := func(v int) { var b [8]byte; binary.BigEndian.PutUint64(b[:], uint64(v)); out.Write(b[:]) }
	number(len(records))
	matched := 0
	for _, r := range records {
		number(len(r.Path))
		out.WriteString(r.Path)
		if r.Candidate {
			number(1)
		} else {
			number(0)
		}
		if r.Matches {
			number(1)
			matched++
		} else {
			number(0)
		}
	}
	write(root, filepath.Join(root, "matches.bin"), out.Bytes())
	report, e := json.MarshalIndent(records, "", "  ")
	if e != nil {
		panic(e)
	}
	write(root, filepath.Join(root, "matches.json"), append(report, '\n'))
	fmt.Printf("Current lock matches: %d cases, %d current, %d stale\n", len(records), matched, len(records)-matched)
}

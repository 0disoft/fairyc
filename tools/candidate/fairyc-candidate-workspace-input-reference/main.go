// Observe dependency declaration fixtures through the complete Go loader.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type dependency struct {
	Binding, Path, Version string
	Features               []string
}

func write(root, path, value string) error {
	p := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(value), 0600)
}
func manifest(uid, version string, members []string, deps []dependency) string {
	text := fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid=%q\nname='input'\nversion=%q\nlanguage='fairyc-language/20'\nsource='src'\n", uid, version)
	if members != nil {
		raw, _ := json.Marshal(members)
		text += "[workspace]\nmembers=" + string(raw) + "\n"
	}
	slices.SortFunc(deps, func(a, b dependency) int { return strings.Compare(a.Binding, b.Binding) })
	for _, d := range deps {
		raw, _ := json.Marshal(d.Features)
		text += fmt.Sprintf("[dependencies.%s]\npath=%q\nversion=%q\nfeatures=%s\n", d.Binding, d.Path, d.Version, raw)
	}
	return text
}
func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: workspace-input-reference <output>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type edge struct {
		Owner, Binding, Target string
		Features               []string
	}
	type row struct {
		Name         string
		Accepted     bool
		Reason       string
		Edges        []edge
		Files, Bytes int
	}
	rows := []row{}
	for _, name := range []string{"Simple", "Features", "UnknownRoot", "UnknownMember", "Version", "BuildVersion", "UnreachableVersion", "ReservedUID", "ReservedUIDMember", "ReservedModule", "ReservedNamespace", "ReservedModuleMember", "ReservedBinding", "NearSDKNames", "CatalogFailure"} {
		dir := filepath.Join(root, name)
		members := []string{"a"}
		rootUID := "example.root"
		aUID := "example.member"
		aVersion := "1.0.0"
		rootFile := "main.fy"
		rootSource := "module main;\n"
		aFile := "lib.fy"
		aSource := "module lib;\n"
		deps := []dependency{{"shared", "a", "1.0.0", []string{}}}
		aDeps := []dependency{}
		bVersion := "1.0.0"
		switch name {
		case "Features":
			members = []string{"a", "b"}
			bVersion = "2.0.0"
			deps = []dependency{{"a-lane", "a", "1.0.0", []string{"alpha"}}, {"b-lane", "a", "1.0.0", []string{"beta"}}}
			aDeps = []dependency{{"shared", "b", "2.0.0", []string{}}}
		case "UnknownRoot":
			deps[0].Path = "ghost"
		case "UnknownMember":
			deps = []dependency{}
			aDeps = []dependency{{"shared", "ghost", "1.0.0", []string{}}}
		case "Version":
			deps[0].Version = "2.0.0"
		case "BuildVersion":
			deps[0].Version = "1.0.0+build.2"
			aVersion = "1.0.0+build.1"
		case "UnreachableVersion":
			members = []string{"a", "b"}
			deps = []dependency{}
			aDeps = []dependency{{"shared", "b", "9.0.0", []string{}}}
		case "ReservedUID":
			rootUID = "fairyc.std"
		case "ReservedUIDMember":
			aUID = "fairyc.std"
		case "ReservedModule":
			rootFile = "std.fy"
			rootSource = "module std;\n"
		case "ReservedNamespace":
			rootFile = "std/x.fy"
			rootSource = "module std::x;\n"
		case "ReservedModuleMember":
			aFile = "std/x.fy"
			aSource = "module std::x;\n"
		case "ReservedBinding":
			deps[0].Binding = "stdlib"
		case "NearSDKNames":
			rootUID = "fairyc.stdtools"
			rootFile = "stdtools.fy"
			rootSource = "module stdtools;\n"
			deps[0].Binding = "stdlib-extra"
		}
		if err := write(dir, "fairy.toml", manifest(rootUID, "1.0.0", members, deps)); err != nil {
			return err
		}
		if err := write(dir, "src/"+rootFile, rootSource); err != nil {
			return err
		}
		if name == "CatalogFailure" {
			if err := os.MkdirAll(filepath.Join(dir, "a"), 0700); err != nil {
				return err
			}
		} else {
			if err := write(dir, "a/fairy.toml", manifest(aUID, aVersion, nil, aDeps)); err != nil {
				return err
			}
			if err := write(dir, "a/src/"+aFile, aSource); err != nil {
				return err
			}
		}
		if len(members) > 1 {
			if err := write(dir, "b/fairy.toml", manifest("example.other", bVersion, nil, nil)); err != nil {
				return err
			}
			if err := write(dir, "b/src/other.fy", "module other;\n"); err != nil {
				return err
			}
		}
		if name == "UnknownRoot" || name == "UnknownMember" {
			if err := write(dir, "ghost/fairy.toml", manifest("example.ghost", "1.0.0", nil, nil)); err != nil {
				return err
			}
			if err := write(dir, "ghost/src/lib.fy", "module lib;\n"); err != nil {
				return err
			}
		}
		workspace, err := workspaceloader.Load(dir)
		r := row{Name: name, Accepted: err == nil, Edges: []edge{}}
		if err != nil {
			e, ok := err.(*workspaceloader.LoadError)
			if !ok {
				return err
			}
			r.Reason = e.Reason
		} else {
			for _, p := range workspace.Packages {
				r.Files += len(p.Sources)
				for _, s := range p.Sources {
					r.Bytes += len(s.Bytes)
				}
				for _, d := range p.Manifest.Dependencies {
					r.Edges = append(r.Edges, edge{p.Path, d.Binding, d.Path, d.Features})
				}
			}
		}
		if err := write(dir, "oracle.txt", fmt.Sprintf("%t\n%s\n", r.Accepted, r.Reason)); err != nil {
			return err
		}
		var record strings.Builder
		for _, e := range r.Edges {
			fmt.Fprintf(&record, "%s\n%s\n%s\n%s\n", e.Owner, e.Binding, e.Target, strings.Join(e.Features, ","))
		}
		if err := write(dir, "edges.txt", record.String()); err != nil {
			return err
		}
		rows = append(rows, r)
	}
	encoded, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "reference.json"), append(encoded, '\n'), 0600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "root.bin"), []byte(root), 0600)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

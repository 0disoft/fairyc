// Local feature-instance graph observations from the complete Go loader.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type dependency struct {
	Binding, Path string
	Features      []string
}

func write(root, path, value string) error {
	p := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(value), 0600)
}
func manifest(uid string, members []string, deps []dependency, bad bool) string {
	result := fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid=%q\nname='graph'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n", uid)
	if members != nil {
		raw, _ := json.Marshal(members)
		result += "[workspace]\nmembers=" + string(raw) + "\n"
	}
	for _, d := range deps {
		raw, _ := json.Marshal(d.Features)
		version := "1.0.0"
		if bad {
			version = "9.0.0"
		}
		result += fmt.Sprintf("[dependencies.%s]\npath=%q\nversion=%q\nfeatures=%s\n", d.Binding, d.Path, version, raw)
	}
	return result
}
func cycleID(member string, features []string) (string, error) {
	path, body := "lib.fy", "module lib;\n"
	if member == "b" {
		path, body = "other.fy", "module other;\n"
	}
	digest, err := packagecontent.Compute([]packagecontent.File{{Path: path, Bytes: []byte(body)}})
	if err != nil {
		return "", err
	}
	id, err := packagegraph.ComputeInstanceID(packagegraph.Identity{PackageUID: "example." + member, SourceIdentity: digest.String(), VersionIdentity: "1.0.0", Features: features})
	return id.String(), err
}
func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: instance-graph-reference <output>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type node struct {
		ID, Path string
		Features []string
	}
	type edge struct{ Owner, Binding, Target string }
	type row struct {
		Name         string
		Accepted     bool
		Reason       string
		Nodes        []node
		Edges        []edge
		Cycle        []string
		Files, Bytes int
	}
	rows := []row{}
	for _, name := range []string{"Single", "Chain", "Aliases", "FeatureSplit", "FeatureDiamond", "SelfCycle", "TwoCycle", "FeatureLiftCycle", "UnreachableCycle", "VersionError", "LongChain"} {
		dir := filepath.Join(root, name)
		members := []string{"a"}
		deps := []dependency{{"shared", "a", []string{}}}
		aDeps := []dependency{}
		bDeps := []dependency{}
		switch name {
		case "Single":
			members = nil
			deps = nil
		case "Chain":
			members = []string{"a", "b"}
			aDeps = []dependency{{"shared", "b", []string{}}}
		case "Aliases":
			deps = []dependency{{"first", "a", []string{"alpha"}}, {"twin", "a", []string{"alpha"}}}
		case "FeatureSplit":
			deps = []dependency{{"alpha", "a", []string{"alpha"}}, {"beta", "a", []string{"beta"}}}
		case "FeatureDiamond":
			members = []string{"a", "b"}
			deps = []dependency{{"alpha", "a", []string{"alpha"}}, {"beta", "a", []string{"beta"}}}
			aDeps = []dependency{{"shared", "b", []string{"base"}}}
		case "SelfCycle":
			aDeps = []dependency{{"self", "a", []string{}}}
		case "TwoCycle":
			members = []string{"a", "b"}
			aDeps = []dependency{{"shared", "b", []string{}}}
			bDeps = []dependency{{"shared", "a", []string{}}}
		case "FeatureLiftCycle":
			deps = []dependency{{"shared", "a", []string{"alpha"}}}
			aDeps = []dependency{{"self", "a", []string{"beta"}}}
		case "UnreachableCycle":
			members = []string{"a", "b"}
			deps = nil
			aDeps = []dependency{{"shared", "b", []string{}}}
			bDeps = []dependency{{"shared", "a", []string{}}}
		case "LongChain":
			members = make([]string, 70)
			for i := range members {
				members[i] = fmt.Sprintf("p%03d", i)
			}
			deps = []dependency{{"shared", members[0], []string{}}}
		}
		if err := write(dir, "fairy.toml", manifest("example.root", members, deps, name == "VersionError")); err != nil {
			return err
		}
		if err := write(dir, "src/main.fy", "module main;\n"); err != nil {
			return err
		}
		for i, member := range members {
			memberDeps := aDeps
			path, body := "lib.fy", "module lib;\n"
			if member == "b" {
				memberDeps = bDeps
				path, body = "other.fy", "module other;\n"
			}
			if name == "LongChain" {
				memberDeps = []dependency{}
				if i+1 < len(members) {
					memberDeps = []dependency{{"shared", members[i+1], []string{}}}
				}
			}
			if err := write(dir, member+"/fairy.toml", manifest("example."+member, nil, memberDeps, false)); err != nil {
				return err
			}
			if err := write(dir, member+"/src/"+path, body); err != nil {
				return err
			}
		}
		loaded, err := workspaceloader.Load(dir)
		r := row{Name: name, Accepted: err == nil, Nodes: []node{}, Edges: []edge{}, Cycle: []string{}}
		var records strings.Builder
		if err != nil {
			e, ok := err.(*workspaceloader.LoadError)
			if !ok {
				return err
			}
			r.Reason = e.Reason
			if strings.Contains(e.Detail, packagegraph.ReasonDependencyCycle) {
				r.Reason = packagegraph.ReasonDependencyCycle
			}
			if r.Reason == packagegraph.ReasonDependencyCycle {
				features := []string{}
				if name == "FeatureLiftCycle" {
					features = []string{"beta"}
				}
				a, err := cycleID("a", features)
				if err != nil {
					return err
				}
				r.Cycle = []string{a, a}
				if name == "TwoCycle" {
					b, err := cycleID("b", []string{})
					if err != nil {
						return err
					}
					r.Cycle = []string{a, b, a}
				}
			}
		} else {
			ids := map[string]string{}
			for i, n := range loaded.Graph.Nodes {
				id, err := packagegraph.ComputeInstanceID(n.Identity)
				if err != nil {
					return err
				}
				ids[n.ID] = id.String()
				r.Nodes = append(r.Nodes, node{id.String(), loaded.Instances[i].PackagePath, n.Identity.Features})
				fmt.Fprintf(&records, "%s\n%s\n%s\n", id.String(), loaded.Instances[i].PackagePath, strings.Join(n.Identity.Features, ","))
			}
			for _, e := range loaded.Graph.Dependencies {
				r.Edges = append(r.Edges, edge{ids[e.Owner], e.Binding, ids[e.Target]})
				fmt.Fprintf(&records, "%s\n%s\n%s\n", ids[e.Owner], e.Binding, ids[e.Target])
			}
			for _, p := range loaded.Packages {
				r.Files += len(p.Sources)
				for _, s := range p.Sources {
					r.Bytes += len(s.Bytes)
				}
			}
		}
		if err := write(dir, "oracle.txt", fmt.Sprintf("%t\n%s\n", r.Accepted, r.Reason)); err != nil {
			return err
		}
		if err := write(dir, "records.txt", records.String()); err != nil {
			return err
		}
		if err := write(dir, "cycle.txt", strings.Join(r.Cycle, "\n")); err != nil {
			return err
		}
		rows = append(rows, r)
	}
	encoded, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err := write(root, "reference.json", string(encoded)+"\n"); err != nil {
		return err
	}
	return write(root, "root.bin", root)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Local import visibility observations from the owning Go loader/resolver.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type dependency struct {
	Binding, Path string
	Features      []string
}
type file struct{ Path, Body string }
type definition struct {
	Path         string
	Dependencies []dependency
	Files        []file
}

func write(root, path, value string) error {
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(value), 0600)
}
func manifest(p definition, members []string) string {
	uid := "example." + p.Path
	if p.Path == "." {
		uid = "example.root"
	}
	result := fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid=%q\nname='visibility'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n", uid)
	if p.Path == "." && len(members) > 0 {
		raw, _ := json.Marshal(members)
		result += "[workspace]\nmembers=" + string(raw) + "\n"
	}
	for _, d := range p.Dependencies {
		raw, _ := json.Marshal(d.Features)
		result += fmt.Sprintf("[dependencies.%s]\npath=%q\nversion='1.0.0'\nfeatures=%s\n", d.Binding, d.Path, raw)
	}
	return result
}
func observe(dir string, defs []definition) (packagegraph.Graph, packagegraph.Result, *packagegraph.Rejection, string, error) {
	loaded, err := workspaceloader.Load(dir)
	if err == nil {
		return loaded.Graph, loaded.Resolution, nil, "", nil
	}
	e, ok := err.(*workspaceloader.LoadError)
	if !ok {
		return packagegraph.Graph{}, packagegraph.Result{}, nil, "", err
	}
	if e.Reason != workspaceloader.ReasonPackageGraph || strings.Contains(e.Detail, packagegraph.ReasonDependencyCycle) {
		return packagegraph.Graph{}, packagegraph.Result{}, nil, e.Reason, nil
	}
	// The public loader returns no graph on rejection. Obtain its exact catalog
	// and feature graph with import-free module headers, restore all original
	// files, then attach their Go-parsed modules and exact Go source digests.
	modules := map[string][]packagegraph.Module{}
	digests := map[string]packagecontent.Digest{}
	for _, p := range defs {
		files := []packagecontent.File{}
		for _, f := range p.Files {
			analysis, err := frontend.AnalyzeSyntax(source.NewFile(f.Path, []byte(f.Body)), source.DefaultLimits())
			if err != nil {
				return packagegraph.Graph{}, packagegraph.Result{}, nil, "", err
			}
			if analysis.AST == nil || len(analysis.Tree.Diagnostics) > 0 {
				return packagegraph.Graph{}, packagegraph.Result{}, nil, "", fmt.Errorf("invalid reference source: %s", f.Path)
			}
			imports := []string{}
			for _, i := range analysis.AST.Imports {
				imports = append(imports, i.Path)
			}
			modules[p.Path] = append(modules[p.Path], packagegraph.Module{Path: analysis.AST.Path, Imports: imports})
			files = append(files, packagecontent.File{Path: f.Path, Bytes: []byte(f.Body)})
			if err := write(dir, p.Path+"/src/"+f.Path, "module "+analysis.AST.Path+";\n"); err != nil {
				return packagegraph.Graph{}, packagegraph.Result{}, nil, "", err
			}
		}
		digest, err := packagecontent.Compute(files)
		if err != nil {
			return packagegraph.Graph{}, packagegraph.Result{}, nil, "", err
		}
		digests[p.Path] = digest
		slices.SortFunc(modules[p.Path], func(a, b packagegraph.Module) int { return strings.Compare(a.Path, b.Path) })
	}
	neutral, loadErr := workspaceloader.Load(dir)
	for _, p := range defs {
		for _, f := range p.Files {
			if err := write(dir, p.Path+"/src/"+f.Path, f.Body); err != nil {
				return packagegraph.Graph{}, packagegraph.Result{}, nil, "", err
			}
		}
	}
	if loadErr != nil {
		return packagegraph.Graph{}, packagegraph.Result{}, nil, "", loadErr
	}
	for i := range neutral.Graph.Nodes {
		path := neutral.Instances[i].PackagePath
		neutral.Graph.Nodes[i].Modules = modules[path]
		neutral.Graph.Nodes[i].Identity.SourceIdentity = digests[path].String()
	}
	result, rejection, err := packagegraph.Resolve(neutral.Graph)
	if err != nil {
		return packagegraph.Graph{}, packagegraph.Result{}, nil, "", err
	}
	if rejection == nil || !strings.Contains(e.Detail, rejection.Reason) {
		return packagegraph.Graph{}, packagegraph.Result{}, nil, "", fmt.Errorf("loader/resolver rejection mismatch")
	}
	return neutral.Graph, result, rejection, "", nil
}
func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: module-visibility-reference <output>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type lookup struct{ Owner, Source, Path, Provider, Module string }
	type row struct {
		Name                string
		Accepted            bool
		Reason, ImportPath  string
		Lookups             []lookup
		Subjects            []string
		Files, Bytes, Nodes int
	}
	rows := []row{}
	for _, name := range []string{"Empty", "Own", "Direct", "Namespace", "LongTrivia", "LongMissingTrivia", "Aliases", "Transitive", "DirectTransitive", "Ambiguous", "OwnAmbiguous", "MemberOwnAmbiguous", "FeatureAmbiguous", "ByBinding", "CaseMismatch", "UnreachableImport", "PartialMissing", "ModuleOrder", "CycleFirst"} {
		r := definition{Path: ".", Files: []file{{"main.fy", "module main;\n"}}, Dependencies: []dependency{{"first", "a", []string{}}}}
		a := definition{Path: "a", Files: []file{{"lib.fy", "module lib;\n"}}}
		b := definition{Path: "b", Files: []file{{"lib.fy", "module lib;\n"}}}
		defs := []definition{}
		members := []string{"a"}
		r.Files[0].Body += "import lib as local;\n"
		switch name {
		case "Empty":
			members = nil
			r.Dependencies = nil
			r.Files[0].Body = "module main;\n"
		case "Own":
			members = nil
			r.Dependencies = nil
			r.Files[0].Body = "module main;\nimport main as self;\nimport util as local;\n"
			r.Files = append(r.Files, file{"util.fy", "module util;\n"})
		case "Namespace":
			r.Files[0].Body = "module main;\nimport ns::lib as local;\n"
			a.Files = []file{{"ns/lib.fy", "module ns::lib;\n"}}
		case "LongTrivia", "LongMissingTrivia":
			target := "lib"
			if name == "LongMissingTrivia" {
				target = "missing"
			}
			r.Files[0].Body = "module main;\nimport ns /*" + strings.Repeat("t", 70000) + "*/ :: " + target + " as local;\n"
			a.Files = []file{{"ns/lib.fy", "module ns::lib;\n"}}
		case "Aliases":
			r.Files[0].Body = "module main;\nimport lib as one;\nimport lib as two;\n"
			r.Dependencies = []dependency{{"first", "a", []string{"alpha"}}, {"twin", "a", []string{"alpha"}}}
		case "Transitive", "DirectTransitive":
			members = []string{"a", "b"}
			a.Files = []file{{"bridge.fy", "module bridge;\n"}}
			a.Dependencies = []dependency{{"shared", "b", []string{}}}
			if name == "DirectTransitive" {
				r.Dependencies = append(r.Dependencies, dependency{"second", "b", []string{}})
			}
		case "Ambiguous":
			members = []string{"a", "b"}
			r.Dependencies = append(r.Dependencies, dependency{"second", "b", []string{}})
		case "OwnAmbiguous":
			r.Files[0].Body = "module main;\nimport util as local;\n"
			r.Files = append(r.Files, file{"util.fy", "module util;\n"})
			a.Files = []file{{"util.fy", "module util;\n"}}
		case "MemberOwnAmbiguous":
			members = []string{"a", "b"}
			r.Dependencies = []dependency{{"early", "b", []string{}}, {"late", "a", []string{}}}
			r.Files[0].Body = "module main;\n"
			a.Files = []file{{"main.fy", "module main;\nimport util as local;\n"}, {"util.fy", "module util;\n"}}
			a.Dependencies = []dependency{{"shared", "b", []string{}}}
			b.Files = []file{{"util.fy", "module util;\n"}}
		case "FeatureAmbiguous":
			r.Dependencies = []dependency{{"alpha", "a", []string{"alpha"}}, {"beta", "a", []string{"beta"}}}
		case "ByBinding":
			r.Files[0].Body = "module main;\nimport first as local;\n"
		case "CaseMismatch":
			r.Files[0].Body = "module main;\nimport Lib as local;\n"
		case "UnreachableImport":
			r.Dependencies = nil
			r.Files[0].Body = "module main;\n"
			a.Files[0].Body = "module lib;\nimport ghost as absent;\n"
		case "PartialMissing":
			r.Files[0].Body = "module main;\nimport main as self;\nimport ghost as absent;\n"
		case "ModuleOrder":
			r.Files[0].Body = "module main;\n"
			r.Files = append(r.Files, file{"a/x.fy", "module a::x;\nimport left as local;\n"}, file{"a0.fy", "module a0;\nimport right as local;\n"})
			a.Files = []file{{"left.fy", "module left;\n"}, {"right.fy", "module right;\n"}}
		case "CycleFirst":
			r.Files[0].Body = "module main;\nimport ghost as absent;\n"
			a.Dependencies = []dependency{{"self", "a", []string{}}}
		}
		defs = append(defs, r)
		if len(members) > 0 {
			defs = append(defs, a)
		}
		if len(members) > 1 {
			defs = append(defs, b)
		}
		dir := filepath.Join(root, name)
		for _, p := range defs {
			if err := write(dir, p.Path+"/fairy.toml", manifest(p, members)); err != nil {
				return err
			}
			for _, f := range p.Files {
				if err := write(dir, p.Path+"/src/"+f.Path, f.Body); err != nil {
					return err
				}
			}
		}
		g, result, rejection, inputReason, err := observe(dir, defs)
		if err != nil {
			return err
		}
		out := row{Name: name, Accepted: inputReason == "" && rejection == nil, Lookups: []lookup{}, Subjects: []string{}, Nodes: len(g.Nodes)}
		ids := map[string]string{}
		for _, n := range g.Nodes {
			id, err := packagegraph.ComputeInstanceID(n.Identity)
			if err != nil {
				return err
			}
			ids[n.ID] = id.String()
		}
		if inputReason != "" {
			out.Reason = inputReason
		} else if rejection != nil {
			out.Reason = rejection.Reason
			out.ImportPath = rejection.ImportPath
			for _, handle := range rejection.Subjects {
				out.Subjects = append(out.Subjects, ids[handle])
			}
		}
		ordinal := 0
		var records strings.Builder
		for _, n := range g.Nodes {
			for _, m := range n.Modules {
				for _, path := range m.Imports {
					if ordinal < len(result.Lookups) {
						l := result.Lookups[ordinal]
						if l.From != n.ID || l.ImportPath != path || len(l.ProviderMembers) != 1 {
							return fmt.Errorf("unexpected Go lookup ordering")
						}
						entry := lookup{ids[n.ID], m.Path, path, ids[l.ProviderMembers[0]], l.ModulePath}
						out.Lookups = append(out.Lookups, entry)
						fmt.Fprintf(&records, "%s\n%s\n%s\n%s\n%s\n", entry.Owner, entry.Source, entry.Path, entry.Provider, entry.Module)
					}
					ordinal++
				}
			}
		}
		for _, p := range defs {
			out.Files += len(p.Files)
			for _, f := range p.Files {
				out.Bytes += len(f.Body)
			}
		}
		if err := write(dir, "oracle.txt", fmt.Sprintf("%t\n%s\n", out.Accepted, out.Reason)); err != nil {
			return err
		}
		if err := write(dir, "lookups.txt", records.String()); err != nil {
			return err
		}
		if err := write(dir, "subjects.txt", strings.Join(out.Subjects, "\n")); err != nil {
			return err
		}
		if err := write(dir, "path.txt", out.ImportPath); err != nil {
			return err
		}
		var counts [32]byte
		for i, n := range []int{out.Files, out.Bytes, len(out.Lookups), out.Nodes} {
			binary.BigEndian.PutUint64(counts[i*8:], uint64(n))
		}
		if err := os.WriteFile(filepath.Join(dir, "counts.bin"), counts[:], 0600); err != nil {
			return err
		}
		rows = append(rows, out)
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

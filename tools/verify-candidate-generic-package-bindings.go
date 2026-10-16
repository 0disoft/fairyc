//go:build ignore

// Build exact resolved-import and definition oracles from real Go workspaces.
// bun tools/run-go.ts run tools/verify-candidate-generic-package-bindings.go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
func number(out *bytes.Buffer, n int)       { binary.Write(out, binary.BigEndian, uint64(n)) }
func field(out *bytes.Buffer, value []byte) { number(out, len(value)); out.Write(value) }
func manifest(uid, extra string) string {
	return "schema='fairyc.manifest/v1'\n[package]\nuid='" + uid + "'\nname='binding'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n" + extra
}
func dependency(binding, path, feature string) string {
	features := "[]"
	if feature != "" {
		features = "['" + feature + "']"
	}
	return "[dependencies." + binding + "]\npath='" + path + "'\nversion='1.0.0'\nfeatures=" + features + "\n"
}

type definition struct {
	Name   string
	Digest string
}

func definitions(module *ast.Module, bindings []byte) ([]definition, error) {
	var out []definition
	add := func(name, domain string, value any) error {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		out = append(out, definition{name, genericplan.DigestText(domain, string(encoded)+"\n"+string(bindings)).String()})
		return nil
	}
	for _, f := range module.Functions {
		if len(f.TypeParameters) != 0 {
			if err := add(f.Name, "source-package-function-v0", f); err != nil {
				return nil, err
			}
		}
	}
	for _, s := range module.Structs {
		if len(s.TypeParameters) != 0 {
			if err := add(s.Name, "source-package-aggregate-v0", struct {
				Struct   *ast.Struct
				Resource *ast.Resource
			}{&s, nil}); err != nil {
				return nil, err
			}
		}
	}
	for _, r := range module.Resources {
		if len(r.TypeParameters) != 0 {
			if err := add(r.Name, "source-package-aggregate-v0", struct {
				Struct   *ast.Struct
				Resource *ast.Resource
			}{nil, &r}); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
func emitDefinitions(out *bytes.Buffer, defs []definition) {
	number(out, len(defs))
	for _, d := range defs {
		field(out, []byte(d.Name))
		field(out, []byte(d.Digest))
	}
}
func run() error {
	root := filepath.Join(".cache", "generic-package-bindings-20261013", "fixture")
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if err = write(filepath.Join(root, "root.bin"), []byte(absolute)); err != nil {
		return err
	}
	var reports []map[string]any
	var baseLeft string
	for mode, name := range []string{"Base", "Edited", "Reordered", "Provider"} {
		depA, depB := dependency("a", "left", ""), dependency("b", "right", "")
		if mode == 2 {
			depA, depB = depB, depA
		}
		secret, feature := "1", "fast"
		if mode == 1 {
			secret = "2"
		}
		if mode == 3 {
			feature = "tls"
		}
		inputs := map[string]string{
			"fairy.toml":           manifest("example.binding.root", "[workspace]\nmembers=['common','left','right']\n"+depA+depB),
			"src/main.fy":          "module main;import left as a;import left as a0;import right as b;import std::io as io;pub fn T id<T>(T value){return value;}fn i32 main(){return 0;}\n",
			"left/fairy.toml":      manifest("example.binding.left", dependency("c", "common", feature)),
			"left/src/left.fy":     "module left;import common as c;import helper as h;pub fn T id<T>(T value){return value;}\n",
			"right/fairy.toml":     manifest("example.binding.right", dependency("c", "common", "tls")),
			"right/src/right.fy":   "module right;import common as c;pub fn T id<T>(T value){return value;}\n",
			"common/fairy.toml":    manifest("example.binding.common", ""),
			"common/src/common.fy": "module common;pub struct Box<T>{T value;}fn i32 secret(){return " + secret + ";}\n",
			"common/src/helper.fy": "module helper;pub resource Hold<T>{T value;cleanup{}}\n",
		}
		dir := filepath.Join(root, name)
		for path, data := range inputs {
			if err := write(filepath.Join(dir, filepath.FromSlash(path)), []byte(data)); err != nil {
				return err
			}
		}
		workspace, err := workspaceloader.Load(dir)
		if err != nil {
			return err
		}
		packages, err := workspace.SemanticPackages()
		if err != nil {
			return err
		}
		input, err := workspace.SemanticInput()
		if err != nil {
			return err
		}
		byID := map[hir.PackageID]workspaceloader.SemanticPackage{}
		owner := map[hir.PackageID]string{}
		for _, p := range packages {
			byID[p.ID] = p
			owner[p.ID] = fmt.Sprintf("package_coordinate_v1_%x", p.Coordinate[:])
		}
		var encoded bytes.Buffer
		encoded.WriteString("FYGPBND1")
		number(&encoded, len(input.Files))
		defsCount, importCount := 0, 0
		for _, file := range input.Files {
			p := byID[file.Package]
			field(&encoded, []byte(p.Instance.String()))
			field(&encoded, []byte(file.Module.Path))
			field(&encoded, []byte(owner[file.Package]))
			var bindings []string
			for _, imported := range file.Module.Imports {
				var targetOwner string
				for _, b := range input.Imports {
					if b.FromPackage == file.Package && b.ImportPath == imported.Path {
						if targetOwner != "" && targetOwner != owner[b.ToPackage] {
							return fmt.Errorf("conflicting provider")
						}
						targetOwner = owner[b.ToPackage]
					}
				}
				if targetOwner == "" {
					return fmt.Errorf("missing provider")
				}
				bindings = append(bindings, imported.Alias+"="+targetOwner+"::"+imported.Path)
			}
			slices.Sort(bindings)
			wire, err := json.Marshal(bindings)
			if err != nil {
				return err
			}
			field(&encoded, wire)
			importCount += len(bindings)
			defs, err := definitions(file.Module, wire)
			if err != nil {
				return err
			}
			emitDefinitions(&encoded, defs)
			defsCount += len(defs)
			if file.Module.Path == "left" {
				if len(defs) != 1 {
					return fmt.Errorf("missing left template")
				}
				if mode == 0 {
					baseLeft = defs[0].Digest
				} else if (defs[0].Digest == baseLeft) != (mode != 3) {
					return fmt.Errorf("wrong provider invalidation for %s", name)
				}
			}
		}
		if err := write(filepath.Join(dir, "bindings.bin"), encoded.Bytes()); err != nil {
			return err
		}
		reports = append(reports, map[string]any{"case": name, "packages": len(packages), "modules": len(input.Files), "imports": importCount, "definitions": defsCount, "fixtureBytes": encoded.Len()})
	}
	for _, name := range []string{"alloc", "bytes", "fs", "io", "num", "process", "text"} {
		data, err := os.ReadFile(filepath.Join("internal", "sdkstdlib", "sources", "std", name+".fy"))
		if err != nil {
			return err
		}
		if err = write(filepath.Join(root, "std", name+".fy"), data); err != nil {
			return err
		}
	}
	vec, err := os.ReadFile("internal/sdkstdlib/sources/optional/std/vec.fy")
	if err != nil {
		return err
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("std/vec.fy", vec), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		return fmt.Errorf("Vec syntax: %v", err)
	}
	defs, err := definitions(parsed.AST, []byte("null"))
	if err != nil {
		return err
	}
	var encoded bytes.Buffer
	encoded.WriteString("FYGPVEC1")
	field(&encoded, vec)
	emitDefinitions(&encoded, defs)
	if err = write(filepath.Join(root, "vec.bin"), encoded.Bytes()); err != nil {
		return err
	}
	var unchanged []string
	var fixtures []map[string]any
	if err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		unchanged = append(unchanged, filepath.ToSlash(path))
		fixtures = append(fixtures, map[string]any{"path": filepath.ToSlash(path), "sha256": fmt.Sprintf("%x", sha256.Sum256(data))})
		return nil
	}); err != nil {
		return err
	}
	wire, err := json.Marshal(unchanged)
	if err != nil {
		return err
	}
	if err = write(filepath.Join(root, "..", "unchanged.json"), wire); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "fixture": root, "cases": reports, "fixtureFiles": fixtures, "vec": map[string]any{"definitions": len(defs), "sourceSHA256": fmt.Sprintf("%x", sha256.Sum256(vec))}})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

//go:build ignore

// Exact Go JSON and DigestChunks oracle for closed nominal package contexts.
// First generate the workspaces with verify-candidate-generic-package-bindings.go.
package main

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type entry struct {
	Owner   genericplan.PackageID
	Module  *ast.Module
	Imports map[string]genericplan.PackageID
}

func number(out *bytes.Buffer, n int)       { binary.Write(out, binary.BigEndian, uint64(n)) }
func field(out *bytes.Buffer, value []byte) { number(out, len(value)); out.Write(value) }
func write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
func closed(entries []entry) ([]byte, string, error) {
	entries = slices.Clone(entries)
	slices.SortFunc(entries, func(a, b entry) int {
		if order := cmp.Compare(a.Owner, b.Owner); order != 0 {
			return order
		}
		return cmp.Compare(a.Module.Path, b.Module.Path)
	})
	wire, err := json.Marshal(entries)
	if err != nil {
		return nil, "", err
	}
	return wire, genericplan.DigestChunks("source-nominal-closed-graph-v0", wire).String(), nil
}
func nominal(out *bytes.Buffer, module *ast.Module, context string) (int, error) {
	wire, err := json.Marshal(module)
	if err != nil {
		return 0, err
	}
	var names []string
	for _, s := range module.Structs {
		if len(s.TypeParameters) == 0 {
			names = append(names, s.Name)
		}
	}
	for _, r := range module.Resources {
		if len(r.TypeParameters) == 0 {
			names = append(names, r.Name)
		}
	}
	number(out, len(names))
	for _, name := range names {
		field(out, []byte(name))
		field(out, []byte(genericplan.DigestChunks("source-nominal-package-v0", []byte(context), []byte("\n"), wire, []byte("\n"), []byte(name)).String()))
	}
	return len(names), nil
}
func parse(data []byte) (*ast.Module, error) {
	path := regexp.MustCompile(`module\s+([A-Za-z_][A-Za-z_0-9:]*)\s*;`).FindSubmatch(data)
	if len(path) != 2 {
		return nil, fmt.Errorf("missing module path")
	}
	p, err := frontend.AnalyzeSyntax(source.NewFile(strings.ReplaceAll(string(path[1]), "::", "/")+".fy", data), source.DefaultLimits())
	if err != nil || p.AST == nil || !p.Tree.ErrorFree() {
		return nil, fmt.Errorf("syntax %q: %v", data, err)
	}
	return p.AST, nil
}
func run() error {
	base := filepath.Join(".cache", "generic-package-bindings-20261013", "fixture")
	outdir := filepath.Join(".cache", "generic-package-context-20261013")
	var reports []map[string]any
	var baseline string
	for mode, name := range []string{"Base", "Edited", "Reordered", "Provider"} {
		workspace, err := workspaceloader.Load(filepath.Join(base, name))
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
		owners := map[hir.PackageID]genericplan.PackageID{}
		instances := map[hir.PackageID]string{}
		for _, p := range packages {
			owners[p.ID] = genericplan.PackageID(fmt.Sprintf("package_coordinate_v1_%x", p.Coordinate[:]))
			instances[p.ID] = p.Instance.String()
		}
		var entries []entry
		for _, file := range input.Files {
			imports := map[string]genericplan.PackageID{}
			for _, sourceImport := range file.Module.Imports {
				for _, binding := range input.Imports {
					if binding.FromPackage == file.Package && binding.ImportPath == sourceImport.Path {
						imports[sourceImport.Alias] = owners[binding.ToPackage]
					}
				}
				if imports[sourceImport.Alias] == "" {
					return fmt.Errorf("missing import provider")
				}
			}
			entries = append(entries, entry{owners[file.Package], file.Module, imports})
		}
		wire, context, err := closed(entries)
		if err != nil {
			return err
		}
		if mode == 0 {
			baseline = context
		} else if (context == baseline) != (mode == 2) {
			return fmt.Errorf("unexpected context invalidation for %s", name)
		}
		var out bytes.Buffer
		out.WriteString("FYGPCTX1")
		number(&out, len(wire))
		field(&out, []byte(context))
		number(&out, len(input.Files))
		definitions := 0
		for _, file := range input.Files {
			field(&out, []byte(instances[file.Package]))
			field(&out, []byte(file.Module.Path))
			n, err := nominal(&out, file.Module, context)
			if err != nil {
				return err
			}
			definitions += n
		}
		if err = write(filepath.Join(base, name, "context.bin"), out.Bytes()); err != nil {
			return err
		}
		reports = append(reports, map[string]any{"case": name, "modules": len(entries), "jsonBytes": len(wire), "context": context, "nominalDefinitions": definitions})
	}
	authored, err := os.ReadFile("examples/lexer_candidate/tests/generic_package_context.fy")
	if err != nil {
		return err
	}
	matches := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`).FindAllSubmatch(authored, -1)
	if len(matches) != 3 {
		return fmt.Errorf("expected three authored sources")
	}
	var entries []entry
	var raw [][]byte
	for i, m := range matches {
		index, err := strconv.Atoi(string(m[1]))
		if err != nil || index != i {
			return fmt.Errorf("noncontiguous sources")
		}
		text, err := strconv.Unquote(string(m[2]))
		if err != nil {
			return err
		}
		data := []byte(text)
		module, err := parse(data)
		if err != nil {
			return err
		}
		raw = append(raw, data)
		owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat("2", 64))
		if i == 0 {
			owner = genericplan.PackageID("package_coordinate_v1_" + strings.Repeat("1", 64))
		}
		imports := map[string]genericplan.PackageID{}
		for _, imp := range module.Imports {
			imports[imp.Alias] = genericplan.PackageID("package_coordinate_v1_" + strings.Repeat("2", 64))
		}
		entries = append(entries, entry{owner, module, imports})
	}
	wire, context, err := closed(entries)
	if err != nil {
		return err
	}
	var small bytes.Buffer
	small.WriteString("FYGPUNIT")
	number(&small, len(wire))
	field(&small, []byte(context))
	number(&small, len(entries))
	for i, e := range entries {
		field(&small, raw[i])
		if _, err = nominal(&small, e.Module, context); err != nil {
			return err
		}
	}
	if err = write(filepath.Join(base, "unit-context.bin"), small.Bytes()); err != nil {
		return err
	}
	vec, err := os.ReadFile("internal/sdkstdlib/sources/optional/std/vec.fy")
	if err != nil {
		return err
	}
	module, err := parse(vec)
	if err != nil {
		return err
	}
	entries = nil
	for i := 0; i < 8; i++ {
		entries = append(entries, entry{genericplan.PackageID(fmt.Sprintf("package_coordinate_v1_%064x", i+1)), module, map[string]genericplan.PackageID{}})
	}
	large, largeContext, err := closed(entries)
	if err != nil {
		return err
	}
	if len(large) <= 1048576 {
		return fmt.Errorf("scale graph did not exceed 1MiB")
	}
	var scale bytes.Buffer
	scale.WriteString("FYGPSCL1")
	number(&scale, len(entries))
	number(&scale, len(large))
	field(&scale, []byte(largeContext))
	field(&scale, vec)
	if err = write(filepath.Join(base, "scale-context.bin"), scale.Bytes()); err != nil {
		return err
	}
	bindingSource, err := os.ReadFile("examples/std_fs_quickstart/src/main.fy")
	if err != nil {
		return err
	}
	// Identity encoding needs syntax only. Remove imports for a one-module oracle.
	bindingSource = regexp.MustCompile(`(?m)^import[^\n]*\n`).ReplaceAll(bindingSource, nil)
	bindingModule, err := parse(bindingSource)
	if err != nil {
		return err
	}
	bindingWire, bindingContext, err := closed([]entry{{genericplan.PackageID(fmt.Sprintf("package_coordinate_v1_%064x", 1)), bindingModule, map[string]genericplan.PackageID{}}})
	if err != nil {
		return err
	}
	var sidecar bytes.Buffer
	sidecar.WriteString("FYGPSID1")
	number(&sidecar, len(bindingWire))
	field(&sidecar, []byte(bindingContext))
	field(&sidecar, bindingSource)
	if err = write(filepath.Join(base, "sidecar-context.bin"), sidecar.Bytes()); err != nil {
		return err
	}
	var files []map[string]any
	var paths []string
	if err = filepath.WalkDir(base, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(path))
		files = append(files, map[string]any{"path": filepath.ToSlash(path), "sha256": fmt.Sprintf("%x", sha256.Sum256(data))})
		return nil
	}); err != nil {
		return err
	}
	encoded, err := json.Marshal(paths)
	if err != nil {
		return err
	}
	if err = write(filepath.Join(outdir, "unchanged.json"), encoded); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "cases": reports, "unit": map[string]any{"jsonBytes": len(wire), "context": context, "nominalDefinitions": 3}, "scale": map[string]any{"modules": 8, "jsonBytes": len(large), "context": largeContext, "sourceSHA256": fmt.Sprintf("%x", sha256.Sum256(vec))}, "fixtureFiles": files})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

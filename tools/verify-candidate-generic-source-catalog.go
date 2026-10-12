// Exact original declaration catalog from the real Go package expander and AST.
// bun tools/run-go.ts run tools/verify-candidate-generic-source-catalog.go
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/source"
)

func number(out *bytes.Buffer, n int) {
	if err := binary.Write(out, binary.BigEndian, uint64(n)); err != nil {
		panic(err)
	}
}
func field(out *bytes.Buffer, v []byte) { number(out, len(v)); out.Write(v) }
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	root := filepath.Join(".cache", "generic-source-catalog-20261013")
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	vec, err := os.ReadFile("internal/sdkstdlib/sources/optional/std/vec.fy")
	if err != nil {
		return err
	}
	raw := [][]byte{
		[]byte("module root;import helper as a;import helper as a0;import std::vec as vectors;pub fn T export<T>(T value){return a::id<T>(value);}fn T local<T>(T value){return value;}"),
		[]byte("module helper;pub fn U id<U>(U value){return value;}fn A pair<A,B>(A a,B b){return a;}fn T secret<T>(T value){return value;}"), vec,
	}
	owners := make([]genericplan.PackageID, len(raw))
	modules := make([]*ast.Module, len(raw))
	inputs := make([]genericsource.PackageModule, len(raw))
	var out bytes.Buffer
	out.WriteString("FYGSCAT1")
	number(&out, len(raw))
	for i, value := range raw {
		owners[i] = genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(i+1), 64))
		parsed, err := frontend.AnalyzeSyntax(source.NewFile([]string{"root.fy", "helper.fy", "std/vec.fy"}[i], value), source.DefaultLimits())
		if err != nil {
			return err
		}
		if parsed.AST == nil || !parsed.Tree.ErrorFree() {
			return fmt.Errorf("parse %d failed", i)
		}
		modules[i] = parsed.AST
		inputs[i] = genericsource.PackageModule{Module: parsed.AST, Owner: owners[i], Imports: map[string]genericplan.PackageID{}}
		field(&out, value)
	}
	inputs[0].Imports["helper"] = owners[1]
	inputs[0].Imports["std::vec"] = owners[2]
	expanded, err := genericsource.PrepareSDKStoragePackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{owners[2]})
	if err != nil {
		return err
	}
	number(&out, len(expanded.Declarations))
	var functions, aggregates, parameters int
	for _, origin := range expanded.Declarations {
		m := slices.Index(owners, origin.Owner)
		if m < 0 {
			return fmt.Errorf("missing owner")
		}
		var value any
		domain := "source-package-function-v0"
		kind := 3
		switch origin.Kind {
		case genericsource.OriginFunction:
			for _, f := range modules[m].Functions {
				if f.Name == origin.Name {
					value = f
					break
				}
			}
			functions++
		case genericsource.OriginStruct:
			kind = 4
			domain = "source-package-aggregate-v0"
			for i := range modules[m].Structs {
				if modules[m].Structs[i].Name == origin.Name {
					value = struct {
						Struct   *ast.Struct
						Resource *ast.Resource
					}{&modules[m].Structs[i], nil}
					break
				}
			}
			aggregates++
		case genericsource.OriginResource:
			kind = 6
			domain = "source-package-aggregate-v0"
			for i := range modules[m].Resources {
				if modules[m].Resources[i].Name == origin.Name {
					value = struct {
						Struct   *ast.Struct
						Resource *ast.Resource
					}{nil, &modules[m].Resources[i]}
					break
				}
			}
			aggregates++
		default:
			return fmt.Errorf("unknown origin kind")
		}
		if value == nil {
			return fmt.Errorf("missing source template %s", origin.ID)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var imports []string
		for _, imported := range modules[m].Imports {
			imports = append(imports, imported.Alias+"="+string(inputs[m].Imports[imported.Path])+"::"+imported.Path)
		}
		slices.Sort(imports)
		bound, err := json.Marshal(imports)
		if err != nil {
			return err
		}
		digest := genericplan.DigestText(domain, string(encoded)+"\n"+string(bound))
		number(&out, m)
		number(&out, kind)
		number(&out, int(origin.Span.StartByte))
		number(&out, int(origin.Span.EndByte))
		number(&out, int(origin.NameSpan.StartByte))
		number(&out, int(origin.NameSpan.EndByte))
		number(&out, 0)
		if origin.Public {
			out.Truncate(out.Len() - 8)
			number(&out, 1)
		}
		field(&out, []byte(origin.Name))
		field(&out, []byte(origin.ID))
		field(&out, []byte(digest.String()))
		number(&out, len(origin.Parameters))
		for _, name := range origin.Parameters {
			field(&out, []byte(name))
			scalar := 0
			if kind == 3 && m != 2 {
				scalar = 1
			}
			number(&out, scalar)
			parameters++
		}
	}
	if err := os.WriteFile(filepath.Join(root, "oracle.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	report := map[string]any{"modules": len(raw), "templates": len(expanded.Declarations), "functions": functions, "aggregates": aggregates, "parameters": parameters, "publicTemplates": len(expanded.Templates), "bytes": out.Len(), "fixture": filepath.Join(root, "oracle.bin"), "source": "genericsource.PrepareSDKStoragePackages.Declarations + actual AST package definitions"}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

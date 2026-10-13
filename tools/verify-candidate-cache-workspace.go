//go:build ignore

// Derive workspace cache facts through the actual Go workspace loader.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
func manifest(uid, extra string) string {
	return "schema='fairyc.manifest/v1'\n[package]\nuid='" + uid + "'\nname='cache-workspace'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n" + extra
}
func dependency(binding, path, feature string) string {
	features := "[]"
	if feature != "" {
		features = "['" + feature + "']"
	}
	return "[dependencies." + binding + "]\npath='" + path + "'\nversion='1.0.0'\nfeatures=" + features + "\n"
}
func run(root string) error {
	names := []string{"Base", "Edited", "Reordered", "NoSDK", "Vec", "Version", "Feature"}
	var reports []map[string]any
	for _, name := range names {
		a := dependency("a", "left", "")
		b := dependency("b", "right", "")
		if name == "Reordered" {
			a, b = b, a
		}
		rootManifest := manifest("example.cache.root", "[workspace]\nmembers=['common','left','right']\n"+a+b+dependency("same-left", "left", ""))
		if name == "Version" {
			rootManifest = strings.Replace(rootManifest, "version='1.0.0'", "version='1.1.0'", 1)
		}
		main := "module main;import left as a;import right as b;import std::io as io;fn i32 main(){return a::value()+b::value();}\n"
		if name == "NoSDK" {
			main = strings.ReplaceAll(main, "import std::io as io;", "")
		}
		if name == "Vec" {
			main = "module main;import left as a;import right as b;import std::vec as v;fn i32 main(){var v::Vec<i32> items=v::empty<i32>();discard v::push<i32>(ref mut items,a::value());return b::value();}\n"
		}
		secret, feature := "1", "fast"
		if name == "Edited" {
			secret = "2"
		}
		if name == "Feature" {
			feature = "turbo"
		}
		inputs := map[string]string{
			"fairy.toml": rootManifest, "src/main.fy": main,
			"left/fairy.toml":      manifest("example.cache.left", dependency("c", "common", feature)),
			"left/src/left.fy":     "module left;import common as c;pub fn i32 value(){return c::value();}\n",
			"right/fairy.toml":     manifest("example.cache.right", dependency("c", "common", "tls")),
			"right/src/right.fy":   "module right;import common as c;pub fn i32 value(){return c::value();}\n",
			"common/fairy.toml":    manifest("example.cache.common", ""),
			"common/src/common.fy": "module common;pub fn i32 value(){return secret();}fn i32 secret(){return " + secret + ";}\n",
		}
		dir := filepath.Join(root, name)
		for path, data := range inputs {
			if err := write(filepath.Join(dir, filepath.FromSlash(path)), []byte(data)); err != nil {
				return err
			}
		}
		loaded, err := workspaceloader.Load(dir)
		if err != nil {
			return err
		}
		packages, err := loaded.SemanticPackages()
		if err != nil {
			return err
		}
		byID := map[uint32]string{}
		for _, p := range packages {
			byID[uint32(p.ID)] = p.Instance.String()
		}
		var out bytes.Buffer
		out.WriteString("FYCWORK1")
		number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
		field := func(s string) { number(len(s)); out.WriteString(s) }
		number(len(packages))
		standardLibrary := ""
		for _, p := range packages {
			if p.PackageUID == "fairyc.std" {
				standardLibrary = p.Instance.String()
			}
			field(p.Instance.String())
			field(p.Coordinate.String())
			field(p.Source.String())
			number(len(p.Features))
			for _, f := range p.Features {
				field(f)
			}
			number(len(p.Dependencies))
			for _, d := range p.Dependencies {
				field(byID[uint32(d)])
			}
		}
		field(standardLibrary)
		if err := write(filepath.Join(dir, "facts.bin"), out.Bytes()); err != nil {
			return err
		}
		reports = append(reports, map[string]any{"case": name, "packages": packages, "standardLibrary": standardLibrary})
	}
	for _, name := range []string{"alloc", "bytes", "fs", "io", "num", "process", "text", "vec"} {
		path := filepath.Join("internal", "sdkstdlib", "sources", "std", name+".fy")
		if name == "vec" {
			path = filepath.Join("internal", "sdkstdlib", "sources", "optional", "std", "vec.fy")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := write(filepath.Join(root, "std", name+".fy"), data); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"fixture": root, "cases": reports, "passed": true})
}
func main() {
	root := flag.String("output", ".cache/cache-workspace-20261014/fixture", "contained fixture directory")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

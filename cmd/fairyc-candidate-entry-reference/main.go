// Go oracle and fixture generator only; entry selection is authored in Fairy.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func write(root, leaf, text string) {
	path := filepath.Join(root, filepath.FromSlash(leaf))
	must(os.MkdirAll(filepath.Dir(path), 0700))
	must(os.WriteFile(path, []byte(text), 0600))
}
func main() {
	if len(os.Args) != 2 {
		panic("usage: candidate-entry-reference OUTPUT")
	}
	root, err := filepath.Abs(os.Args[1])
	must(err)
	var rows []map[string]any
	for _, spec := range []struct{ name, source string }{
		{"Root", "module main;import aaa as h;fn i32 main(){return h::main()+41;}"},
		{"Missing", "module main;fn i32 answer(){return 1;}"},
		{"WrongModule", "module other;fn i32 main(){return 0;}"},
		{"WrongResult", "module main;fn bool main(){return true;}"},
		{"Parameter", "module main;fn i32 main(i32 n){return n;}"},
		{"Generic", "module main;fn T main<T>(T n){return n;}"},
	} {
		dir := filepath.Join(root, spec.name)
		write(dir, "fairy.toml", "schema='fairyc.manifest/v1'\n[package]\nuid='example.entry.root'\nname='entry'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n[workspace]\nmembers=['dep']\n[dependencies.d]\npath='dep'\nversion='1.0.0'\nfeatures=[]\n")
		write(dir, "src/aaa.fy", "module aaa;pub fn i32 main(){return 1;}")
		leaf := "src/main.fy"
		if spec.name == "WrongModule" {
			leaf = "src/other.fy"
		}
		write(dir, leaf, spec.source)
		write(dir, "dep/fairy.toml", "schema='fairyc.manifest/v1'\n[package]\nuid='example.entry.dep'\nname='dep'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n")
		write(dir, "dep/src/main.fy", "module main;pub fn i32 main(){return 99;}pub fn i32 answer(){return 41;}")
		workspace, err := workspaceloader.Load(dir)
		must(err)
		doc, err := lockfile.FromWorkspace(workspace)
		must(err)
		locked, err := lockfile.Marshal(doc)
		must(err)
		write(dir, "fairy.lock", string(locked))
		var stdout, stderr bytes.Buffer
		exit := cli.Run([]string{"experimental", "emit-workspace-c", "--target", "x86_64-windows-msvc", "--locked"}, dir, &stdout, &stderr)
		if (spec.name == "Root") != (exit == 0) {
			panic(fmt.Sprintf("%s unexpected entry result %d: %s", spec.name, exit, stderr.String()))
		}
		if spec.name != "Root" && !bytes.Contains(stderr.Bytes(), []byte("root entry")) {
			panic(spec.name + ": expected canonical root-entry rejection: " + stderr.String())
		}
		rows = append(rows, map[string]any{"name": spec.name, "exit": exit, "stderr": stderr.String(), "cBytes": stdout.Len()})
	}
	encoded, err := json.MarshalIndent(map[string]any{"schema": "fairyc.candidate-entry-reference/v1", "cases": rows}, "", "  ")
	must(err)
	write(root, "oracle.json", string(encoded)+"\n")
	write(root, "root.bin", root)
	fmt.Println("six entry fixtures match the Go root-main contract")
}

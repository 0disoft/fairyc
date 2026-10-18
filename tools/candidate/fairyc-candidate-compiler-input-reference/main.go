// Fixture/oracle only: production input projection remains in Fairy.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func write(root, leaf string, data []byte) {
	path := filepath.Join(root, filepath.FromSlash(leaf))
	must(os.MkdirAll(filepath.Dir(path), 0700))
	must(os.WriteFile(path, data, 0600))
}
func manifest(uid, extra string) []byte {
	return []byte(fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid='%s'\nname='input'\nversion='1.0.0'\nlanguage='%s'\nsource='src'\n%s", uid, sdkstdlib.LanguageRevision, extra))
}

type source struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Module string `json:"module"`
	Bytes  []byte `json:"bytes"`
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: compiler-input-reference OUTPUT")
	}
	root, err := filepath.Abs(os.Args[1])
	must(err)
	must(os.MkdirAll(root, 0700))
	write(root, "fairy.toml", manifest("example.input.root", "[workspace]\nmembers=['left','left-lib','right','right-lib']\n[dependencies.l]\npath='left'\nversion='1.0.0'\nfeatures=[]\n[dependencies.r]\npath='right'\nversion='1.0.0'\nfeatures=[]\n"))
	write(root, "src/main.fy", []byte("module main;\nimport common as c;\nimport left as l;\nimport right as r;\nfn i32 answer(){return c::value()+l::answer()+r::answer();}\n"))
	write(root, "src/common.fy", []byte("module common;\npub struct Box{i32 root;}\npub fn Box make(){return Box{.root=3};}\npub fn i32 value(){return 3;}\n"))
	for _, name := range []string{"left", "right"} {
		write(root, name+"/fairy.toml", manifest("example.input."+name, "[dependencies.c]\npath='"+name+"-lib'\nversion='1.0.0'\nfeatures=[]\n"))
		write(root, name+"/src/"+name+".fy", []byte(fmt.Sprintf("module %s;\nimport common as c;\npub fn c::Box make(){return c::make();}\npub fn i32 answer(){return c::value();}\nfn i32 hidden(){return 9;}\n", name)))
		write(root, name+"-lib/fairy.toml", manifest("example.input."+name+".lib", ""))
		field, value := "left", "11"
		if name == "right" {
			field, value = "right", "22"
		}
		write(root, name+"-lib/src/common.fy", []byte(fmt.Sprintf("module common;\npub struct Box{i32 %s;}\npub fn Box make(){return Box{.%s=%s};}\npub fn i32 value(){return %s;}\n", field, field, value, value)))
	}
	var stdout, stderr bytes.Buffer
	exit := cli.Run([]string{"check"}, root, &stdout, &stderr)
	if exit != 0 {
		panic(stderr.String())
	}
	ws, err := workspaceloader.Load(root)
	must(err)
	var rows []source
	for _, instance := range ws.Instances {
		pkg, ok := ws.Package(instance.PackagePath)
		if !ok {
			panic("package absent")
		}
		var id string
		for _, node := range ws.Graph.Nodes {
			if node.ID == instance.NodeID {
				value, e := packagegraph.ComputeInstanceID(node.Identity)
				must(e)
				id = value.String()
			}
		}
		if id == "" {
			panic("identity absent")
		}
		for _, file := range pkg.Sources {
			rows = append(rows, source{ID: id, Path: file.Path, Module: file.Module.Path, Bytes: file.Bytes})
		}
	}
	var packet bytes.Buffer
	packet.WriteString("FYINPUT1")
	must(binary.Write(&packet, binary.BigEndian, uint64(len(rows))))
	for _, row := range rows {
		for _, field := range [][]byte{[]byte(row.ID), []byte(row.Path), []byte(row.Module), row.Bytes} {
			must(binary.Write(&packet, binary.BigEndian, uint64(len(field))))
			packet.Write(field)
		}
	}
	write(root, "inputs.bin", packet.Bytes())
	write(root, "root.bin", []byte(filepath.ToSlash(root)))
	observation := struct {
		Exit    int      `json:"exit"`
		Stdout  string   `json:"stdout"`
		Stderr  string   `json:"stderr"`
		Sources []source `json:"sources"`
	}{exit, stdout.String(), stderr.String(), rows}
	data, err := json.MarshalIndent(observation, "", "  ")
	must(err)
	write(root, "observations.json", append(data, '\n'))
	fmt.Printf("Go CLI check passed; %d exact source snapshots\n", len(rows))
}

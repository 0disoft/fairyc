// Generate a local source-edit corpus using the real Go workspace loader.
// bun tools/run-go.ts run tools/verify-candidate-coordinate-graph.go
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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
	return "schema='fairyc.manifest/v1'\n[package]\nuid='" + uid + "'\nname='coordinate'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n" + extra
}
func dependency(binding, path, feature string) string {
	features := "[]"
	if feature != "" {
		features = "['" + feature + "']"
	}
	return "[dependencies." + binding + "]\npath='" + path + "'\nversion='1.0.0'\nfeatures=" + features + "\n"
}
func field(out *bytes.Buffer, value string) {
	binary.Write(out, binary.BigEndian, uint64(len(value)))
	out.WriteString(value)
}
func run() error {
	root := filepath.Join(".cache", "coordinate-graph-20261013", "fixture")
	baseID := map[string]string{}
	baseCoordinate := map[string]string{}
	var reports []map[string]any
	for mode, name := range []string{"Base", "Edited", "Reordered", "NoSDK"} {
		depA := dependency("a", "left", "")
		depB := dependency("b", "right", "")
		if mode == 2 {
			depA, depB = depB, depA
		}
		rootManifest := manifest("example.coordinate.root", "[workspace]\nmembers=['common','left','right']\n"+depA+depB)
		main := "module main;import left as a;import right as b;import std::io as io;fn i32 main(){return 0;}\n"
		if mode == 3 {
			main = strings.ReplaceAll(main, "import std::io as io;", "")
		}
		secret := "1"
		if mode == 1 {
			secret = "2"
		}
		inputs := map[string]string{
			"fairy.toml": rootManifest, "src/main.fy": main,
			"left/fairy.toml":      manifest("example.coordinate.left", dependency("c", "common", "fast")),
			"left/src/left.fy":     "module left;import common as c;pub fn i32 value(){return c::value();}\n",
			"right/fairy.toml":     manifest("example.coordinate.right", dependency("c", "common", "tls")),
			"right/src/right.fy":   "module right;import common as c;pub fn i32 value(){return c::value();}\n",
			"common/fairy.toml":    manifest("example.coordinate.common", ""),
			"common/src/common.fy": "module common;pub fn i32 value(){return secret();}fn i32 secret(){return " + secret + ";}\n",
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
		want := 6
		if mode == 3 {
			want = 5
		}
		if len(packages) != want {
			return fmt.Errorf("%s has %d packages, expected %d", name, len(packages), want)
		}
		var encoded bytes.Buffer
		encoded.WriteString("FYCOORD1")
		binary.Write(&encoded, binary.BigEndian, uint64(len(packages)))
		changed := 0
		for _, p := range packages {
			field(&encoded, p.Instance.String())
			field(&encoded, p.Coordinate.String())
			key := p.PackageUID + "[" + strings.Join(p.Features, ",") + "]"
			if mode == 0 {
				baseID[key] = p.Instance.String()
				baseCoordinate[key] = p.Coordinate.String()
			} else {
				if baseCoordinate[key] != p.Coordinate.String() {
					return fmt.Errorf("%s changed coordinate %s", name, key)
				}
				if baseID[key] != p.Instance.String() {
					changed++
				}
			}
		}
		if mode == 1 && changed != 2 {
			return fmt.Errorf("source edit changed %d instance IDs, expected 2", changed)
		}
		if err := write(filepath.Join(dir, "coordinates.bin"), encoded.Bytes()); err != nil {
			return err
		}
		reports = append(reports, map[string]any{"case": name, "packages": len(packages), "changedInstanceIDs": changed, "packagesObserved": packages})
	}
	for _, name := range []string{"alloc", "bytes", "fs", "io", "num", "process", "text"} {
		data, err := os.ReadFile(filepath.Join("internal", "sdkstdlib", "sources", "std", name+".fy"))
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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

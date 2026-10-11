// Go oracle/fixture only; the package compiler core is authored in Fairy.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/packagegraph"
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

type row struct {
	Name     string `json:"name"`
	Exit     int    `json:"exit"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	File     string `json:"file"`
	Excerpt  string `json:"excerpt"`
	Start    uint64 `json:"start"`
	End      uint64 `json:"end"`
	Phase    uint64 `json:"phase"`
	Stage    uint64 `json:"stage"`
	Frames   uint64 `json:"mirFunctions"`
	Instance string `json:"instance"`
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: compiler-core-reference BASE OUTPUT")
	}
	base, err := filepath.Abs(os.Args[1])
	must(err)
	root, err := filepath.Abs(os.Args[2])
	must(err)
	workspace, err := workspaceloader.Load(base)
	must(err)
	var rows []row
	for _, spec := range []struct {
		name, file, from, to, excerpt string
		phase, stage                  uint64
	}{
		{"Current", "", "", "", "", 0, 0},
		{"Signature", "right-lib/src/common.fy", "pub fn Box make()", "pub fn Missing make()", "Missing", 2, 0},
		{"RootBody", "src/main.fy", "return c::value()+l::answer()+r::answer();", "return true;", "true", 3, 4},
		{"DependencyBody", "right/src/right.fy", "return c::value();", "return true;", "true", 3, 4},
		{"Private", "src/main.fy", "l::answer()", "l::hidden()", "l::hidden", 3, 2},
		{"Nominal", "src/main.fy", "", "\nfn void mismatch(){var a=l::make();let b=r::make();a=b;}\n", "b", 3, 4},
		{"Moved", "src/main.fy", "", "\nresource Handle{i32 n;cleanup{}}\nfn Handle make(){return Handle{.n=1};}\nfn i32 moved(){let h=make();let other=h;return h.n;}\n", "h", 3, 5},
		{"SDK", "right/src/right.fy", "import common as c;", "import common as c;\nimport std::bytes as bytes;\nfn usize sdk_length(){let b=bytes::empty();return bytes::len(ref b);}", "", 0, 0},
	} {
		dir := filepath.Join(root, spec.name)
		for _, pkg := range workspace.Packages {
			manifest, err := os.ReadFile(filepath.Join(pkg.Directory, "fairy.toml"))
			must(err)
			write(dir, filepath.ToSlash(filepath.Join(pkg.Path, "fairy.toml")), manifest)
			for _, source := range pkg.Sources {
				write(dir, filepath.ToSlash(filepath.Join(pkg.Path, pkg.Manifest.Package.Source, source.Path)), source.Bytes)
			}
		}
		var start, end uint64
		if spec.file != "" {
			path := filepath.Join(dir, filepath.FromSlash(spec.file))
			data, err := os.ReadFile(path)
			must(err)
			text := string(data)
			if spec.from == "" {
				text += spec.to
			} else {
				if strings.Count(text, spec.from) != 1 {
					panic("non-unique replacement")
				}
				text = strings.Replace(text, spec.from, spec.to, 1)
			}
			write(dir, spec.file, []byte(text))
			if spec.excerpt != "" {
				index := strings.LastIndex(text, spec.excerpt)
				if spec.name == "Signature" {
					index = strings.Index(text, spec.excerpt)
				}
				if index < 0 {
					panic("missing span")
				}
				start, end = uint64(index), uint64(index+len(spec.excerpt))
			}
		}
		loaded, err := workspaceloader.Load(dir)
		must(err)
		doc, err := lockfile.FromWorkspace(loaded)
		must(err)
		locked, err := lockfile.Marshal(doc)
		must(err)
		write(dir, "fairy.lock", locked)
		var stdout, stderr bytes.Buffer
		exit := cli.Run([]string{"check", "--locked"}, dir, &stdout, &stderr)
		if (spec.phase == 0) != (exit == 0) {
			panic(spec.name + ": " + stderr.String())
		}
		var frames uint64
		if exit == 0 {
			m := regexp.MustCompile(`(\d+) MIR function`).FindStringSubmatch(stdout.String())
			if len(m) != 2 {
				panic("missing MIR count")
			}
			frames, err = strconv.ParseUint(m[1], 10, 64)
			must(err)
		} else {
			m := regexp.MustCompile(`\[bytes (\d+)\.\.(\d+)\]`).FindStringSubmatch(stderr.String())
			if len(m) != 3 {
				panic("missing diagnostic span")
			}
			actualStart, e := strconv.ParseUint(m[1], 10, 64)
			must(e)
			actualEnd, e := strconv.ParseUint(m[2], 10, 64)
			must(e)
			if actualStart != start || actualEnd != end {
				panic("fixture span differs from owning diagnostic")
			}
		}
		var instanceID string
		if spec.file != "" {
			packagePath := "."
			if cut := strings.Index(spec.file, "/src/"); cut >= 0 {
				packagePath = spec.file[:cut]
			}
			for index, mapping := range loaded.Instances {
				if mapping.PackagePath == packagePath {
					id, e := packagegraph.ComputeInstanceID(loaded.Graph.Nodes[index].Identity)
					must(e)
					instanceID = id.String()
					break
				}
			}
			if instanceID == "" {
				panic("missing diagnostic instance")
			}
		}
		rows = append(rows, row{spec.name, exit, stdout.String(), stderr.String(), spec.file, spec.excerpt, start, end, spec.phase, spec.stage, frames, instanceID})
	}
	var packet bytes.Buffer
	packet.WriteString("FYCORE01")
	must(binary.Write(&packet, binary.BigEndian, uint64(len(rows))))
	for _, r := range rows {
		for _, f := range []string{r.Name, filepath.Base(r.File), r.Excerpt, r.Instance} {
			must(binary.Write(&packet, binary.BigEndian, uint64(len(f))))
			packet.WriteString(f)
		}
		for _, n := range []uint64{uint64(r.Exit), r.Start, r.End, r.Phase, r.Stage, r.Frames} {
			must(binary.Write(&packet, binary.BigEndian, n))
		}
	}
	write(root, "cases.bin", packet.Bytes())
	write(root, "root.bin", []byte(filepath.ToSlash(root)))
	data, err := json.MarshalIndent(rows, "", "  ")
	must(err)
	write(root, "observations.json", append(data, '\n'))
	fmt.Printf("Go locked check oracle: %d cases\n", len(rows))
}

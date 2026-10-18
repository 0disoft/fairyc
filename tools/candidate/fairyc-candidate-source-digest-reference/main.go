// Generate exact source fixtures and digests with the owning Go implementation.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/0disoft/fairyc/internal/packagecontent"
)

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: source-digest-reference <output-directory>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type fixture struct {
		Name   string
		Files  []packagecontent.File
		Digest string
	}
	three := []packagecontent.File{{Path: "z.fy", Bytes: []byte("module z;\n")}, {Path: "nested/z.fy", Bytes: []byte("module nested::z;\n")}, {Path: "a.fy", Bytes: []byte("module a;\n")}}
	fixtures := []fixture{
		{Name: "One", Files: []packagecontent.File{{Path: "a.fy", Bytes: []byte("module a;\n")}}},
		{Name: "Three", Files: three},
		{Name: "OtherRoot", Files: slices.Clone(three)},
		{Name: "CRLF", Files: []packagecontent.File{{Path: "a.fy", Bytes: []byte("module a;\r\n")}}},
		{Name: "Binary", Files: []packagecontent.File{{Path: "a.fy", Bytes: []byte{0, 255, 128, 13, 10}}}},
		{Name: "Large", Files: []packagecontent.File{{Path: "a.fy", Bytes: bytes.Repeat([]byte{97}, 1052672)}}},
		{Name: "ModuleOrder", Files: []packagecontent.File{{Path: "a0.fy", Bytes: []byte("module a0;\n")}, {Path: "a/item.fy", Bytes: []byte("module a::item;\n")}}},
		{Name: "Renamed", Files: []packagecontent.File{{Path: "b.fy", Bytes: []byte("module a;\n")}}},
	}
	for i := range fixtures {
		f := &fixtures[i]
		digest, err := packagecontent.Compute(f.Files)
		if err != nil {
			return err
		}
		f.Digest = digest.String()
		reversed := slices.Clone(f.Files)
		slices.Reverse(reversed)
		again, err := packagecontent.Compute(reversed)
		if err != nil || digest != again {
			return fmt.Errorf("input-order mismatch: %s", f.Name)
		}
		for _, source := range f.Files {
			target := filepath.Join(root, f.Name, "src", filepath.FromSlash(source.Path))
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			if err := os.WriteFile(target, source.Bytes, 0600); err != nil {
				return err
			}
		}
		manifest := "schema='fairyc.manifest/v1'\n[package]\nuid='example.digest'\nname='digest'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n"
		if err := os.WriteFile(filepath.Join(root, f.Name, "fairy.toml"), []byte(manifest), 0600); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, f.Name, "digest.txt"), []byte(f.Digest), 0600); err != nil {
			return err
		}
	}
	if fixtures[1].Digest != fixtures[2].Digest || fixtures[0].Digest == fixtures[3].Digest || fixtures[0].Digest == fixtures[7].Digest {
		return fmt.Errorf("identity sensitivity mismatch")
	}
	encoded, err := json.MarshalIndent(fixtures, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "reference.json"), append(encoded, '\n'), 0600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "root.bin"), []byte(root), 0600)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

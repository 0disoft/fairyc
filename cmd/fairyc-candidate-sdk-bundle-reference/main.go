// Fixture bytes and identity observations from the owning compiler SDK bundle.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func write(root, name string, body []byte) {
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		panic(err)
	}
}
func main() {
	if len(os.Args) != 2 {
		panic("expected output directory")
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		panic(err)
	}
	profiles := []struct {
		name, identity string
		load           func() (sdkstdlib.Definition, error)
	}{
		{"core", sdkstdlib.SDKIdentity, sdkstdlib.Load},
		{"vector", sdkstdlib.SDKIdentity, sdkstdlib.LoadVector},
		{"candidate", sdkstdlib.VectorCandidateIdentity, sdkstdlib.LoadVectorCandidate},
	}
	var observations []map[string]any
	for _, profile := range profiles {
		d, err := profile.load()
		if err != nil {
			panic(err)
		}
		var record strings.Builder
		for _, value := range []string{sdkstdlib.PackageUID, sdkstdlib.PackageName, sdkstdlib.Version, sdkstdlib.LanguageRevision, sdkstdlib.RuntimeABI, sdkstdlib.PackagePath, profile.identity, "sdk:stdlib[]", d.SourceIdentity, d.Digest.String()} {
			fmt.Fprintln(&record, value)
		}
		for _, source := range d.Sources {
			write(root, source.Path, source.Bytes)
			fmt.Fprintln(&record, source.Path)
			fmt.Fprintln(&record, source.Module.Path)
			for _, imported := range source.Module.Imports {
				fmt.Fprintln(&record, imported.Path)
			}
			fmt.Fprintln(&record)
		}
		write(root, profile.name+".txt", []byte(record.String()))
		observations = append(observations, map[string]any{"profile": profile.name, "files": len(d.Sources), "digest": d.Digest.String(), "sourceIdentity": d.SourceIdentity})
	}
	data, err := json.MarshalIndent(observations, "", "  ")
	if err != nil {
		panic(err)
	}
	write(root, "reference.json", append(data, '\n'))
	write(root, "root.bin", []byte(filepath.ToSlash(root)))
	fmt.Println("PASS: exact core, vector and vector-candidate SDK source/identity fixtures")
}

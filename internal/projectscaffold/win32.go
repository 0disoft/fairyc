package projectscaffold

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// Canonical sources stay in templates/, tools/win32_app/ and the existing examples.
//
//go:generate python -B ../../tools/bundle_win32_starter.py
//go:embed assets/win32.zip
var win32Bundle []byte

func win32Files() ([]scaffoldFile, []string, error) {
	archive, err := zip.NewReader(bytes.NewReader(win32Bundle), int64(len(win32Bundle)))
	if err != nil {
		return nil, nil, err
	}
	if len(archive.File) > 128 {
		return nil, nil, fmt.Errorf("too many template files")
	}
	var files []scaffoldFile
	directories := map[string]bool{}
	seen := map[string]bool{}
	total := 0
	for _, entry := range archive.File {
		name := entry.Name
		if !fs.ValidPath(name) || strings.ContainsAny(name, `\:`) || !entry.Mode().IsRegular() || seen[name] {
			return nil, nil, fmt.Errorf("invalid template entry %q", name)
		}
		seen[name] = true
		input, err := entry.Open()
		if err != nil {
			return nil, nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(input, 1<<20+1))
		closeErr := input.Close()
		if readErr != nil {
			return nil, nil, readErr
		}
		if closeErr != nil {
			return nil, nil, closeErr
		}
		total += len(data)
		if len(data) > 1<<20 || total > 4<<20 {
			return nil, nil, fmt.Errorf("template exceeds size bound")
		}
		for directory := path.Dir(name); directory != "."; directory = path.Dir(directory) {
			directories[directory] = true
		}
		files = append(files, scaffoldFile{path: name, content: func(spec Spec) []byte {
			if name == "fairy.toml" {
				return manifestBytes(spec)
			}
			if name == "app.toml" || name == "native/app.rc" || name == "README.md" {
				return []byte(strings.NewReplacer("@@NAME@@", spec.Name, "@@UID@@", spec.UID, "@@TITLE@@", spec.Name).Replace(string(data)))
			}
			return data
		}})
	}
	for _, required := range []string{"fairy.toml", "app.toml", "native/app.rc", "native/app.ico", "src/main.fy", "win32_app/portable.py"} {
		if !seen[required] {
			return nil, nil, fmt.Errorf("missing template file %q", required)
		}
	}
	names := make([]string, 0, len(directories))
	for name := range directories {
		names = append(names, name)
	}
	slices.Sort(names)
	return files, names, nil
}

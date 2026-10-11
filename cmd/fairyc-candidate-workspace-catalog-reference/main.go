// Workspace-catalog fixtures observed through the owning Go workspace loader.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func write(root, path, value string) error {
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(value), 0600)
}
func manifest(uid, language string, members []string, nested bool) string {
	text := fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid=%q\nname='catalog'\nversion='1.0.0'\nlanguage=%q\nsource='src'\n", uid, language)
	if nested {
		raw, _ := json.Marshal(members)
		text += "[workspace]\nmembers=" + string(raw) + "\n"
	}
	return text
}
func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: workspace-catalog-reference <output>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type row struct {
		Name     string
		Accepted bool
		Reason   string
		Paths    []string
		UIDs     []string
		Digests  []string
		Files    int
		Bytes    int
	}
	rows := []row{}
	for _, name := range []string{"Single", "Valid", "NearNames", "InterveningOverlap", "CaseCollision", "Nested", "DuplicateUID", "MemberLanguage", "RootLanguage", "MissingManifest", "BadMemberSyntax", "WrongCase", "Undeclared", "TooMany"} {
		dir := filepath.Join(root, name)
		members := []string{}
		language := "fairyc-language/20"
		switch name {
		case "Valid":
			members = []string{"a", "b"}
		case "NearNames":
			members = []string{"a", "a-b"}
		case "InterveningOverlap":
			members = []string{"a", "a-b", "a/b"}
		case "CaseCollision":
			members = []string{"A", "a"}
		case "WrongCase":
			members = []string{"A"}
		case "TooMany":
			for i := 0; i < 1024; i++ {
				members = append(members, fmt.Sprintf("p%04d", i))
			}
		case "RootLanguage":
			language = "fairyc-language/19"
		case "Single", "Undeclared":
		default:
			members = []string{"a"}
		}
		if err := write(dir, "fairy.toml", manifest("example.root", language, members, len(members) > 0)); err != nil {
			return err
		}
		if err := write(dir, "src/main.fy", "module main;\n"); err != nil {
			return err
		}
		for i, member := range members {
			if name == "TooMany" {
				break
			}
			if name == "CaseCollision" && i > 0 {
				continue
			}
			if name == "WrongCase" {
				member = "a"
			}
			if name == "MissingManifest" {
				if err := os.MkdirAll(filepath.Join(dir, member), 0700); err != nil {
					return err
				}
				continue
			}
			uid := fmt.Sprintf("example.member-%d", i)
			if name == "Valid" {
				uid = ifUID(i)
			}
			if name == "DuplicateUID" {
				uid = "example.root"
			}
			lang := "fairyc-language/20"
			if name == "MemberLanguage" {
				lang = "fairyc-language/19"
			}
			if err := write(dir, member+"/fairy.toml", manifest(uid, lang, []string{}, name == "Nested")); err != nil {
				return err
			}
			source := "module lib;\n"
			if name == "BadMemberSyntax" {
				source = "module other;\n"
			}
			if err := write(dir, member+"/src/lib.fy", source); err != nil {
				return err
			}
		}
		if name == "Undeclared" {
			if err := write(dir, "a/fairy.toml", "invalid"); err != nil {
				return err
			}
		}
		workspace, err := workspaceloader.Load(dir)
		r := row{Name: name, Accepted: err == nil, Paths: []string{}, UIDs: []string{}, Digests: []string{}}
		if err != nil {
			load, ok := err.(*workspaceloader.LoadError)
			if !ok {
				return err
			}
			r.Reason = load.Reason
		} else {
			for _, p := range workspace.Packages {
				r.Paths = append(r.Paths, p.Path)
				r.UIDs = append(r.UIDs, p.Manifest.Package.UID)
				r.Digests = append(r.Digests, p.Digest.String())
				r.Files += len(p.Sources)
				for _, source := range p.Sources {
					r.Bytes += len(source.Bytes)
				}
			}
		}
		if err := write(dir, "oracle.txt", fmt.Sprintf("%t\n%s\n", r.Accepted, r.Reason)); err != nil {
			return err
		}
		var catalog strings.Builder
		for i := range r.Paths {
			fmt.Fprintf(&catalog, "%s\n%s\n%s\n", r.Paths[i], r.UIDs[i], r.Digests[i])
		}
		if err := write(dir, "catalog.txt", catalog.String()); err != nil {
			return err
		}
		rows = append(rows, r)
	}
	encoded, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "reference.json"), append(encoded, '\n'), 0600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "root.bin"), []byte(strings.TrimSuffix(root, string(filepath.Separator))), 0600)
}
func ifUID(i int) string {
	if i == 0 {
		return "example.zed"
	}
	return "example.aaa"
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

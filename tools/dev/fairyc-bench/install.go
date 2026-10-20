package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Installation trees are separate from per-case outputs. Shared native tools
// are inventoried once and list all their consumers.
func addInstallations(r *Receipt, a adapter, seen map[string]int) {
	add := func(name, scope, path string) {
		if i, ok := seen[path]; ok {
			for _, language := range r.Installations[i].UsedBy {
				if language == a.language {
					return
				}
			}
			r.Installations[i].UsedBy = append(r.Installations[i].UsedBy, a.language)
			return
		}
		i := Installation{Name: name, UsedBy: []string{a.language}, Scope: scope, Status: "measured"}
		err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				i.Symlinks++
				return nil
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				i.Bytes += info.Size()
				i.Files++
			}
			return nil
		})
		if err != nil {
			i.Status = "incomplete"
			i.Reason = err.Error()
		}
		if i.Symlinks > 0 {
			i.Status = "incomplete"
			i.Reason = "symlink targets excluded"
		}
		seen[path] = len(r.Installations)
		r.Installations = append(r.Installations, i)
	}
	switch a.language {
	case "fairy":
		add("Fairy C compiler", "standalone executable; embedded SDK included; Go build environment excluded", a.executable)
	case "zig":
		add("Zig distribution", "installed executable directory, including bundled libraries and documentation", filepath.Dir(a.executable))
	case "rust":
		if root := commandOutput("rustc", "--print", "sysroot"); root != "" {
			add("Rust active toolchain", "active rustc sysroot, including installed libraries, tools and documentation", root)
		}
	}
	if runtime.GOOS == "windows" && (a.language == "fairy" || a.language == "rust" || (a.language == "c" && strings.EqualFold(filepath.Base(a.executable), "cl.exe"))) {
		if tc, err := cdriver.ResolveWindowsHostMSVC(); err == nil {
			root := tc.Path
			for j := 0; j < 4; j++ {
				root = filepath.Dir(root)
			}
			add("MSVC "+tc.Version, "active MSVC version tree; includes installed architecture variants", root)
			for _, entry := range tc.Environment {
				name, value, _ := strings.Cut(entry, "=")
				if !strings.EqualFold(name, "LIB") {
					continue
				}
				for _, p := range strings.Split(value, ";") {
					if !strings.Contains(p, "Windows Kits") {
						continue
					}
					versionRoot := filepath.Dir(filepath.Dir(p))
					version := filepath.Base(versionRoot)
					sdkRoot := filepath.Dir(filepath.Dir(versionRoot))
					for _, part := range []string{"Include", "Lib", "bin"} {
						path := filepath.Join(sdkRoot, part, version)
						if _, err := os.Stat(path); err == nil {
							add(fmt.Sprintf("Windows SDK %s %s", version, part), "active SDK version subtree", path)
						}
					}
					break
				}
				break
			}
		}
	} else if runtime.GOOS == "linux" && (a.language == "fairy" || a.language == "c") {
		r.Installations = append(r.Installations, Installation{Name: "System C compiler and libc", UsedBy: []string{a.language}, Scope: "distributed OS packages", Status: "unmeasured", Reason: "no bounded installation root; executable identity is in toolchains"})
	}
}

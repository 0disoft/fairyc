package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func selectLinuxCache(t *testing.T, root string) {
	t.Helper()
	for name, changes := range map[string][][2]string{
		"project_cache_disk": {
			{"import windows_rooted_file as rooted;", "import linux_cache_file as rooted;"},
			{"import windows_rooted_file as handles;", "import linux_rooted_file as handles;"},
		},
		"project_cache_environment": {{"import windows_host_paths as paths;", "import linux_host_paths as paths;"}},
		"project_cache_profile": {
			{"x86_64-windows-msvc", "x86_64-linux-gnu"},
			{";llp64;microsoft-x64;msvc-x64-llp64;microsoft-x64;coff-x86-64;windows-x64-pdata-xdata", ";lp64;sysv-amd64;sysv-amd64-lp64;sysv-amd64;elf64;dwarf-cfi"},
		},
	} {
		path := filepath.Join(root, "src", name+".fy")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source := string(data)
		for _, change := range changes {
			want := 1
			if change[0] == "x86_64-windows-msvc" {
				want = 2
			}
			if strings.Count(source, change[0]) != want {
				t.Fatalf("Linux cache boundary changed: %s", name)
			}
			source = strings.ReplaceAll(source, change[0], change[1])
		}
		writeTestSource(t, root, "src/"+name+".fy", source)
	}
}

func checkLinuxCache(t *testing.T, tc cdriver.NativeToolchain, artifact cdriver.NativeArtifact) {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-cache"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return 0;}\n")
	cache := filepath.Join(root, "cache")
	if err := os.Mkdir(cache, 0700); err != nil {
		t.Fatal(err)
	}
	run := func(args []string, code int32, status string) string {
		t.Helper()
		result, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: args, WorkingDirectory: root, Timeout: 60 * time.Second})
		if err != nil || result.ExitCode != code || !strings.Contains(result.Stderr, status) {
			t.Fatalf("cache %v: %+v %v", args, result, err)
		}
		return result.Stdout
	}
	args := []string{"check", "--cache-root", "cache", "--cache-status"}
	cold := run(args, 0, "cache: cold stored\n")
	lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	locked := append(append([]string{}, args...), "--locked")
	warm := run(locked, 0, "cache: hit\n")
	if !strings.Contains(cold, "check ok:") || !strings.Contains(warm, "check ok:") {
		t.Fatal("cache changed successful report", cold, warm)
	}
	after, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || !bytes.Equal(lock, after) {
		t.Fatal("warm locked check changed lock", err)
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) == 0 {
		t.Fatal("no stored generations", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".stage") {
			t.Fatal("anonymous stage leaked into inventory")
		}
		if err := os.WriteFile(filepath.Join(cache, entry.Name()), []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run(locked, 0, "cache: cold stored\n")
	run(locked, 0, "cache: hit\n")
	run([]string{"check", "--cache-root", "missing-cache", "--cache-status"}, 0, "cache: cold uncached\n")
	if _, err := os.Stat(filepath.Join(root, "missing-cache")); !os.IsNotExist(err) {
		t.Fatal("cache directory must remain caller-owned", err)
	}
	if err := os.Symlink("cache", filepath.Join(root, "linked-cache")); err != nil {
		t.Fatal(err)
	}
	run([]string{"check", "--cache-root", "linked-cache", "--cache-status"}, 0, "cache: cold uncached\n")
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return 1;}\n")
	run(locked, 7, "cache: skipped\n")
	run(args, 0, "cache: cold stored\n")
	run(locked, 0, "cache: hit\n")
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return true;}\n")
	run(locked, 7, "cache: skipped\n")
	run(args, 3, "cache: cold uncached\n")
}

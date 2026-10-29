package cli

import (
	"bytes"
	"context"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func buildLinuxCompanion(t *testing.T) []byte {
	t.Helper()
	output := filepath.Join(t.TempDir(), "companion")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", output, "./tools/candidate/fairyc-candidate-linux-build")
	command.Dir = filepath.Join("..", "..")
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build Linux bootstrap companion: %s %v", data, err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func checkLinuxBuild(t *testing.T, artifact cdriver.NativeArtifact, backend []byte) {
	t.Helper()
	install := t.TempDir()
	image := filepath.Join(install, "fairyc")
	companion := filepath.Join(install, "fairyc-candidate-linux-build")
	if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(companion, backend, 0700); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	temporary := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-build"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	source := `module main;
import std::fs as fs;
@effects(filesystem) fn i32 main(){
    var file=switch(fs::open_read_utf8("input.txt")){case .ok(p):{p}case .err(_):{return 1;}};
    var u8[3] data=[0;3];let count=switch(fs::read(ref mut file,ref mut data)){case .ok(p):{p}case .err(_):{return 2;}};
    if(!fs::close(file).is_ok){return 3;}return if(count==3 && data[0]==97){0}else{4};
}
`
	writeTestSource(t, root, "src/main.fy", source)
	writeTestSource(t, root, "input.txt", "abc")
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	run := func(args []string, want int) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		command := exec.CommandContext(ctx, image, args...)
		command.Dir = root
		command.Env = append(os.Environ(), "TMPDIR="+temporary)
		command.WaitDelay = 2 * time.Second
		data, err := command.CombinedOutput()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("Linux build %v: code=%d %s", args, code, data)
		}
		remaining, err := os.ReadDir(temporary)
		if err != nil || len(remaining) != 0 {
			t.Fatalf("scratch leak after %v: %v %v", args, remaining, err)
		}
		return string(data)
	}
	if data := run([]string{"build", "--locked", "--cache-root", "cache", "--cache-status"}, 7); !strings.Contains(data, "cache: skipped\n") {
		t.Fatal(data)
	}
	for i, args := range [][]string{{"build", "--cache-root", "cache", "--cache-status"}, {"build", "--release", "--locked", "--cache-root", "cache", "--cache-status"}} {
		status := []string{"cache: cold stored\n", "cache: hit\n"}[i]
		if data := run(args, 0); !strings.Contains(data, "built: build/client (x86_64-linux-gnu,") || !strings.Contains(data, status) {
			t.Fatal(data)
		}
		path := filepath.Join(root, "build", "client")
		file, err := elf.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if file.Machine != elf.EM_X86_64 {
			t.Fatal(file.Machine)
		}
		file.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		command := exec.CommandContext(ctx, path)
		command.Dir = root
		data, err := command.CombinedOutput()
		cancel()
		if err != nil || len(data) != 0 {
			t.Fatalf("built SDK file reader: %s %v", data, err)
		}
	}
	previous, err := os.ReadFile(filepath.Join(root, "build", "client"))
	if err != nil {
		t.Fatal(err)
	}
	run([]string{"build", "--release", "--release"}, 2)
	run([]string{"build", "--cache-root"}, 2)
	run([]string{"build", "--cache-status"}, 2)
	run([]string{"build", "--cache-root", "cache", "--cache-status", "--cache-status"}, 2)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return true;}\n")
	if data := run([]string{"build", "--cache-root", "cache", "--cache-status"}, 3); !strings.Contains(data, "cache: cold uncached\n") || !strings.Contains(data, "E1009") {
		t.Fatal(data)
	}
	writeTestSource(t, root, "src/main.fy", source)
	if err := os.Remove(companion); err != nil {
		t.Fatal(err)
	}
	if data := run([]string{"build", "--cache-root", "cache", "--cache-status"}, 6); !strings.Contains(data, "cache: hit\n") {
		t.Fatal(data)
	}
	after, err := os.ReadFile(filepath.Join(root, "build", "client"))
	if err != nil || !bytes.Equal(previous, after) {
		t.Fatal("failed build replaced the previous executable", err)
	}
}

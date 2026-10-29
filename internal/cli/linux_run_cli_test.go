package cli

import (
	"bytes"
	"context"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func checkLinuxRun(t *testing.T, artifact cdriver.NativeArtifact, backend []byte) {
	t.Helper()
	install, root, temporary := t.TempDir(), t.TempDir(), t.TempDir()
	image, companion := filepath.Join(install, "fairyc"), filepath.Join(install, "fairyc-candidate-linux-build")
	for path, data := range map[string][]byte{image: artifact.Artifact.Bytes, companion: backend} {
		if err := os.WriteFile(path, data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-run"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return 0;}\n")
	writeTestSource(t, root, "build/client", "previous published image")
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	run := func(args []string, want int) (string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		command := exec.CommandContext(ctx, image, args...)
		command.Dir = root
		command.Env = append(os.Environ(), "TMPDIR="+temporary)
		command.WaitDelay = 2 * time.Second
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		err := command.Run()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("Linux run %v: code=%d want=%d stdout=%q stderr=%q", args, code, want, stdout.String(), stderr.String())
		}
		entries, err := os.ReadDir(temporary)
		if err != nil || len(entries) != 0 {
			t.Fatalf("run scratch leak %v %v", entries, err)
		}
		previous, err := os.ReadFile(filepath.Join(root, "build", "client"))
		if err != nil || string(previous) != "previous published image" {
			t.Fatal("run changed published image", err)
		}
		return stdout.String(), stderr.String()
	}
	run([]string{"run", "--locked"}, 7)
	for i, args := range [][]string{{"run", "--cache-root", "cache", "--cache-status"}, {"run", "--release", "--locked", "--cache-root", "cache", "--cache-status"}} {
		out, err := run(args, 0)
		if out != "" || err != []string{"cache: cold stored\n", "cache: hit\n"}[i] {
			t.Fatalf("silent cached run: %q %q", out, err)
		}
	}
	for _, test := range []struct {
		source string
		code   int
	}{{"return 133;", 133}, {"return -1;", 255}, {"return 256;", 0}} {
		writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){"+test.source+"}\n")
		out, err := run([]string{"run"}, test.code)
		if out != "" || err != "" {
			t.Fatalf("ordinary exit treated as trap: %q %q", out, err)
		}
	}
	writeTestSource(t, root, "src/main.fy", `module main;
import std::io as io;
import std::process as process;
import std::fs as fs;
@effects(console,environment,filesystem) fn i32 main(){
 if(process::arg_count()!=6){return 10;}
 var u8[32] raw=[0;32];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 11;}};
 if(n!=0){return 12;}
 let m=switch(process::arg(2,ref mut raw)){case .ok(p):{p}case .err(_):{return 13;}};
 let ref u8[] data=ref raw;if(!io::write_bytes(data[..m]).is_ok){return 14;}
 if(!io::write_bytes(b"\0\xff").is_ok || !io::write_stderr_bytes(b"err\0\xff").is_ok){return 15;}
 var file=switch(fs::open_read_utf8("input.txt")){case .ok(p):{p}case .err(_):{return 16;}};
 if(!fs::close(file).is_ok){return 17;}return 0;
}
`)
	writeTestSource(t, root, "input.txt", "workspace cwd")
	for i, profile := range [][]string{{"run", "--cache-root", "cache", "--cache-status", "--", "", "a b", "--release", "--cache-root", "--cache-status"}, {"run", "--release", "--locked", "--cache-root", "cache", "--cache-status", "--", "", "a b", "--release", "--cache-root", "--cache-status"}} {
		// The first call creates the lock for this exact SDK/source closure.
		out, err := run(profile, 0)
		if out != "a b\x00\xff" || err != []string{"cache: cold stored\n", "cache: hit\n"}[i]+"err\x00\xff" {
			t.Fatalf("run streams/args: %q %q", out, err)
		}
	}
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){assert(false);return 0;}\n")
	for i, args := range [][]string{{"run", "--cache-root", "cache", "--cache-status"}, {"run", "--locked", "--cache-root", "cache", "--cache-status"}} {
		out, err := run(args, 5)
		if out != "" || !strings.HasPrefix(err, []string{"cache: cold stored\n", "cache: hit\n"}[i]) || !strings.Contains(err, "trap: assertion-failed\n  at src/main.fy:1:") {
			t.Fatalf("cached trap receipt/location: %q %q", out, err)
		}
	}
	writeTestSource(t, root, "src/main.fy", `module main; import std::io as io;
@effects(console) fn i32 main(){var usize i=0;while(i<65537){if(!io::write("x").is_ok){return 1;}i+=1;}return 0;}
`)
	out, err := run([]string{"run"}, 6)
	if len(out) != 65536 || !strings.Contains(err, "program output exceeds 65536 bytes per stream") {
		t.Fatalf("run output limit: bytes=%d %q", len(out), err)
	}
	run([]string{"run", "--release", "--release"}, 2)
	run([]string{"run", "--cache-root"}, 2)
	run([]string{"run", "--cache-status"}, 2)
	run([]string{"run", "--cache-root", "cache", "--cache-root", "cache"}, 2)
	run([]string{"run", "program-arg-without-separator"}, 2)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return true;}\n")
	run([]string{"run"}, 3)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return 0;}\n")
	if err := os.Remove(companion); err != nil {
		t.Fatal(err)
	}
	_, detail := run([]string{"run", "--cache-root", "absent-cache", "--cache-status"}, 6)
	if !strings.HasPrefix(detail, "cache: cold uncached\n") {
		t.Fatal(detail)
	}
	if _, err := os.Stat(filepath.Join(root, "absent-cache")); !os.IsNotExist(err) {
		t.Fatal("cache directory created", err)
	}
}

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func checkLinuxTest(t *testing.T, artifact cdriver.NativeArtifact, backend []byte) {
	t.Helper()
	install, root, tmp := t.TempDir(), t.TempDir(), t.TempDir()
	image, companion := filepath.Join(install, "fairyc"), filepath.Join(install, "fairyc-candidate-linux-build")
	if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-test"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", "module main; fn i32 main(){return 0;}\n")
	writeTestSource(t, root, "tests/answer.fy", `module tests::answer;
@test fn void a_fails(){assert(false);}
@test fn void b_passes(){assert(true);}
`)
	writeTestSource(t, root, "tests/nested/empty.fy", "module tests::nested::empty;\n")
	writeTestSource(t, root, "build/client", "published sentinel")
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	run := func(args []string, input string, want int) (string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, image, args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "TMPDIR="+tmp)
		cmd.Stdin = strings.NewReader(input)
		cmd.WaitDelay = 2 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("Linux test %v code=%d want=%d stdout=%q stderr=%q", args, code, want, stdout.String(), stderr.String())
		}
		entries, err := os.ReadDir(tmp)
		if err != nil || len(entries) != 0 {
			t.Fatalf("test scratch leak: %v %v", entries, err)
		}
		old, err := os.ReadFile(filepath.Join(root, "build", "client"))
		if err != nil || string(old) != "published sentinel" {
			t.Fatal("test changed published image", err)
		}
		return stdout.String(), stderr.String()
	}
	// Listing and semantic execution must work without a native companion.
	run([]string{"test", "--locked"}, "", 7)
	run([]string{"test", "--list"}, "", 0)
	for _, args := range [][]string{
		{"test", "--list", "--json", "--locations", "--locked"},
		{"test", "--json", "--locked"},
		{"test", "--file", "tests/nested/empty.fy", "--locked"},
		{"test", "--dir", "tests/nested", "--list", "--locked"},
		{"test", "--exact", "tests::answer::b_passes", "--locked"},
		{"test", "--package", "test.linux-test", "--list", "--locked"},
		{"test", "--workspace", "--list", "--json", "--locked"},
	} {
		want := 0
		if len(args) == 3 && args[1] == "--json" {
			want = 5
		}
		out, err := run(args, "", want)
		code, refOut, refErr := runForTest(args, root)
		if code != want || out != refOut || err != refErr {
			t.Fatalf("semantic/list reference %v: (%d,%q,%q) vs (%d,%q,%q)", args, want, out, err, code, refOut, refErr)
		}
	}
	run([]string{"test", "--native", "--stdin", "--locked"}, "must not consume", 2)
	run([]string{"test", "--file", "tests/missing.fy", "--locked"}, "", 2)
	run([]string{"test", "--release"}, "", 2)
	run([]string{"test", "--list", "--native"}, "", 2)
	run([]string{"test", "--native", "--cache-root"}, "", 2)
	run([]string{"test", "--cache-status"}, "", 2)
	run([]string{"test", "--cache-root", "cache", "--cache-status", "--cache-status"}, "", 2)
	_, status := run([]string{"test", "--list", "--cache-root", "cache", "--cache-status"}, "", 0)
	if status != "cache: cold stored\n" {
		t.Fatal(status)
	}
	_, status = run([]string{"test", "--json", "--locked", "--cache-root", "cache", "--cache-status"}, "", 5)
	if status != "cache: units hit\n" {
		t.Fatal(status)
	}
	if err := os.WriteFile(companion, backend, 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"test", "--native", "--json", "--locked"}, {"test", "--native", "--release", "--json", "--locked"}} {
		cachedArgs := append(append([]string{}, args...), "--cache-root", "cache", "--cache-status")
		out, err := run(cachedArgs, "", 5)
		code, refOut, refErr := runForTest(args, root)
		if code != 5 || out != refOut || err != refErr+"cache: units hit\n" {
			t.Fatalf("native reference %v: %q %q vs (%d,%q,%q)", args, out, err, code, refOut, refErr)
		}
	}
	writeTestSource(t, root, "tests/answer.fy", `module tests::answer;
import std::io as io;
import std::process as process;
@effects(console,environment) fn void payload(){
 assert(process::arg_count()==3);var u8[16] arg=[0;16];
 let n=switch(process::arg(1,ref mut arg)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(n==0);
 var u8[4] data=[0;4];let count=switch(io::read_stdin(ref mut data)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let ref u8[] view=ref data;assert(io::write_bytes(view[..count]).is_ok && io::write_stderr_bytes(b"\xff").is_ok);
}
@test @effects(console,environment) fn void a_first(){payload();}
@test @effects(console,environment) fn void b_second(){payload();}
`)
	writeTestSource(t, root, "input.bin", string([]byte{0, 255, 7}))
	for _, args := range [][]string{
		{"test", "--native", "--json", "--capture-output", "--stdin-file", "input.bin", "--", "", "--flag"},
		{"test", "--native", "--release", "--json", "--capture-output", "--stdin-file", "input.bin", "--", "", "--flag"},
	} {
		out, err := run(args, "ignored", 0)
		if err != "" {
			t.Fatal(err)
		}
		var report stableTestExecutionReport
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		if report.Count != 2 || report.Passed != 2 || report.Target == nil || *report.Target != "x86_64-linux-gnu" {
			t.Fatal(out)
		}
		for _, test := range report.Tests {
			if test.StdoutBase64 == nil || *test.StdoutBase64 != "AP8H" || test.StderrBase64 == nil || *test.StderrBase64 != "/w==" {
				t.Fatal(out)
			}
		}
	}
	out, _ := run([]string{"test", "--native", "--json", "--capture-output", "--exact", "tests::answer::a_first", "--stdin", "--", "", "--flag"}, "xyz", 0)
	if !strings.Contains(out, `"stdoutBase64":"eHl6"`) {
		t.Fatal(out)
	}
	out, _ = run([]string{"test", "--native", "--json", "--capture-output", "--exact", "tests::answer::a_first", "--", "", "--flag"}, "must see EOF", 0)
	if !strings.Contains(out, `"stdoutBase64":""`) {
		t.Fatal(out)
	}
	run([]string{"test", "--native", "--exact", "tests::answer::a_first", "--", "", "--flag"}, "", 2)
	run([]string{"test", "--native", "--stdin-file", "absent.bin"}, "", 7)
	if err := os.Symlink("input.bin", filepath.Join(root, "link.bin")); err != nil {
		t.Fatal(err)
	}
	run([]string{"test", "--native", "--stdin-file", "link.bin"}, "", 7)
	writeTestSource(t, root, "oversize.bin", strings.Repeat("x", 65537))
	run([]string{"test", "--native", "--stdin-file", "oversize.bin"}, "", 7)
	writeTestSource(t, root, "tests/answer.fy", `module tests::answer; @test fn void times_out(){while(true){}}`)
	_, err := run([]string{"test", "--native", "--timeout-seconds", "1"}, "", 6)
	if !strings.Contains(err, "Linux GNU target executable exceeded 1s") {
		t.Fatal(err)
	}
	writeTestSource(t, root, "tests/answer.fy", `module tests::answer; import std::io as io; @test @effects(console) fn void too_much(){var usize i=0;while(i<65537){discard io::write("x");i+=1;}}`)
	run([]string{"test", "--native", "--json", "--capture-output"}, "", 6)
	writeTestSource(t, root, "tests/answer.fy", "module tests::answer; @test fn i32 invalid(){return 0;}\n")
	run([]string{"test", "--list"}, "", 3)
	writeTestSource(t, root, "tests/answer.fy", "module tests::answer; @test fn void passes(){assert(true);}\n")
	if err := os.Remove(companion); err != nil {
		t.Fatal(err)
	}
	run([]string{"test", "--native"}, "", 6)
	run([]string{"test", "--native", "--file", "tests/nested/empty.fy"}, "", 0)
}

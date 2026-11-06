package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// The actual authored command leaf, before paying for full compiler linking.
func TestCandidateFetchCommand(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows command leaf; Linux restoration remains batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_fetch_cli")
	writeTestSource(t, root, "src/main.fy", `module main;import project_fetch_cli as fetch;@effects(filesystem,console,network,clock) fn i32 main(){return fetch::arguments();}`)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 128<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "fetch-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			verifyCandidateFetchCLI(t, path)
		})
	}
}

func fetchCLIProcess(t *testing.T, image, root string, args []string, broken string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH=")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if broken != "" {
		file, err := os.Open(os.DevNull)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if broken == "stdout" {
			cmd.Stdout = file
		} else {
			cmd.Stderr = file
		}
	}
	err := cmd.Run()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, out.String(), stderr.String()
}

func verifyCandidateFetchCLI(t *testing.T, image string) {
	t.Helper()
	command := []string{"experimental", "fetch-packages", "--offline"}
	for _, args := range [][]string{{"experimental"}, {"experimental", "--help"}, {"experimental", "fetch-packages", "--help"}, {"experimental", "fetch-packages", "-h"}, {"experimental", "fetch-packages", "help"}, {"experimental", "unknown\ncommand"}, {"experimental", "unknown", "--help"}, {"experimental", "fetch-packages", "--bad"}, {"experimental", "fetch-packages", "--offline", "--offline"}, {"experimental", "fetch-packages", "--offline", "extra"}, {"experimental", "fetch-packages", "--"}, {"experimental", "fetch-packages", "--offline=true"}} {
		r := t.TempDir()
		want, stdout, stderr := runForTest(args, r)
		code, out, diagnostic := fetchCLIProcess(t, image, r, args, "")
		if code != want || out != stdout || diagnostic != stderr {
			t.Fatalf("%v: code=%d/%d stdout %s stderr %s", args, code, want, fetchCLITextDiff(out, stdout), fetchCLITextDiff(diagnostic, stderr))
		}
		entries, err := os.ReadDir(r)
		if err != nil || len(entries) != 0 {
			t.Fatal("syntax/help touched workspace", err)
		}
	}
	noOnline := t.TempDir()
	code, out, stderr := fetchCLIProcess(t, image, noOnline, []string{"experimental", "fetch-packages"}, "")
	if code != 7 || out != "" || !strings.HasPrefix(stderr, "package fetch failed: cannot read source plan or root manifest\n") {
		t.Fatalf("online: %d %q %q", code, out, stderr)
	}
	entries, err := os.ReadDir(noOnline)
	if err != nil || len(entries) != 0 {
		t.Fatal("online request touched workspace", err)
	}
	goRoot, nativeRoot := remoteCLIWorkspace(t), remoteCLIWorkspace(t)
	// Independent fixture ZIP bytes need not have the same map iteration order;
	// use one exact reviewed pin/cache snapshot for both implementations.
	for _, name := range []string{"fairy.toml", "fairy.sources.json"} {
		data, err := os.ReadFile(filepath.Join(goRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(nativeRoot, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	cache := filepath.Join(goRoot, "build/package-sources")
	items, err := os.ReadDir(cache)
	if err != nil {
		t.Fatal(err)
	}
	nativeCache := filepath.Join(nativeRoot, "build/package-sources")
	nativeItems, err := os.ReadDir(nativeCache)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range nativeItems {
		if err = os.Remove(filepath.Join(nativeCache, item.Name())); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range items {
		data, err := os.ReadFile(filepath.Join(cache, item.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(nativeCache, item.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []string{goRoot, nativeRoot} {
		if err = os.WriteFile(filepath.Join(r, "fairy.lock"), []byte("keep this graph lock\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		want, stdout, stderr := runForTest(command, goRoot)
		code, out, diagnostic := fetchCLIProcess(t, image, nativeRoot, command, "")
		if code != want || out != stdout || diagnostic != stderr {
			t.Fatalf("restore %d: %d/%d %q/%q %q/%q", i, code, want, out, stdout, diagnostic, stderr)
		}
		if !reflect.DeepEqual(remoteTree(t, goRoot), remoteTree(t, nativeRoot)) {
			t.Fatal("Go/Fairy tree mismatch")
		}
	}
	for _, item := range items {
		if err = os.WriteFile(filepath.Join(nativeCache, item.Name()), []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := remoteTree(t, nativeRoot)
	code, out, stderr = fetchCLIProcess(t, image, nativeRoot, command, "")
	if code != 7 || out != "" || stderr != "package fetch failed: source archive SHA-256 mismatch\n" || !reflect.DeepEqual(before, remoteTree(t, nativeRoot)) {
		t.Fatalf("corrupt warm cache: %d %q %q", code, out, stderr)
	}
	r := remoteCLIWorkspace(t)
	code, out, stderr = fetchCLIProcess(t, image, r, command, "stdout")
	if code != 4 || out != "" || stderr != "fairyc: write output failed\n" {
		t.Fatalf("stdout failure: %d %q %q", code, out, stderr)
	}
	if _, err = os.Stat(filepath.Join(r, "vendor/math/fairy.toml")); err != nil {
		t.Fatal("output failure lost completed restoration", err)
	}
	for _, args := range [][]string{command, {"experimental", "fetch-packages", "--bad"}} {
		code, out, stderr = fetchCLIProcess(t, image, t.TempDir(), args, "stderr")
		if code != 4 || out != "" || stderr != "" {
			t.Fatalf("stderr failure: %d %q %q", code, out, stderr)
		}
	}
}

func fetchCLITextDiff(actual, want string) string {
	at := 0
	for at < len(actual) && at < len(want) && actual[at] == want[at] {
		at++
	}
	return fmt.Sprintf("at byte %d: actual=%q expected=%q", at, actual[max(0, at-40):min(len(actual), at+160)], want[max(0, at-40):min(len(want), at+160)])
}

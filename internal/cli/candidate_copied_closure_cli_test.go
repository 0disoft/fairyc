package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func copiedClosureCLIProfiles(t *testing.T) []cdriver.Profile {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows full CLI replay; native Linux remains deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for full CLI and persistent cache replay")
	}
	profiles := []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile}
	if selected := os.Getenv("FAIRYC_CLOSURE_CLI_PROFILE"); selected != "" {
		for _, p := range profiles {
			if p.Name == selected {
				return []cdriver.Profile{p}
			}
		}
		t.Fatalf("unknown closure CLI profile %q", selected)
	}
	return profiles
}

type copiedClosureCommand struct {
	t           *testing.T
	image, root string
	calls       int
}

func (c *copiedClosureCommand) run(code int, args ...string) (string, string) {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(c.t.Context(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, c.image, args...)
	command.Dir = c.root
	var out, diagnostics bytes.Buffer
	command.Stdout = &out
	command.Stderr = &diagnostics
	err := command.Run()
	exit := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			exit = e.ExitCode()
		} else {
			c.t.Fatal(err)
		}
	}
	c.calls++
	c.t.Logf("CLI %v => %d", args, exit)
	if exit != code {
		c.t.Fatalf("%v: exit %d want %d\nstdout=%s\nstderr=%s", args, exit, code, &out, &diagnostics)
	}
	return out.String(), diagnostics.String()
}
func (c *copiedClosureCommand) cache(code int, status string, args ...string) string {
	c.t.Helper()
	args = append(args, "--cache-root", "cache", "--cache-status")
	out, diagnostics := c.run(code, args...)
	if !strings.HasPrefix(diagnostics, status+"\n") {
		c.t.Fatalf("%v: expected %q, stderr=%s", args, status, diagnostics)
	}
	return out
}
func copiedClosureCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.closure-cli"
name = "closure-cli"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", strings.Replace(candidateCopiedClosureProgram(), "module root;", "module main;", 1))
	writeTestSource(t, root, "tests/closures.fy", `module tests::closures;
@test fn void snapshots(){var i32 base=40;let add=fn [i32 base] i32(i32 x){return base+x;};base=100;assert(add(2)==42);}
`)
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}
func TestCandidateCopiedClosureCLI(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			command := copiedClosureCommand{t: t, image: image, root: copiedClosureCLIWorkspace(t)}
			if out := command.cache(0, "cache: cold stored", "check"); !strings.Contains(out, "check ok:") {
				t.Fatal(out)
			}
			if out := command.cache(0, "cache: hit", "check", "--locked"); !strings.Contains(out, "check ok:") {
				t.Fatal(out)
			}
			for _, flags := range [][]string{nil, {"--release"}} {
				args := append([]string{"build", "--locked"}, flags...)
				if out := command.cache(0, "cache: hit", args...); !strings.Contains(out, "built:") {
					t.Fatal(out)
				}
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				out, err := exec.CommandContext(ctx, filepath.Join(command.root, "build", "closure-cli.exe")).CombinedOutput()
				cancel()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 42 || len(out) != 0 {
					t.Fatalf("built closure: %v %s", err, out)
				}
				args = append([]string{"run", "--locked"}, flags...)
				if out := command.cache(42, "cache: hit", args...); out != "" {
					t.Fatal(out)
				}
			}
			command.run(0, "test", "--list", "--json", "--locked")
			for _, flags := range [][]string{nil, {"--native"}, {"--native", "--release"}} {
				args := append([]string{"test", "--json", "--locked"}, flags...)
				out, _ := command.run(0, args...)
				if !strings.Contains(out, `"passed":1`) {
					t.Fatal(out)
				}
			}
			t.Logf("%d independent CLI processes", command.calls)
		})
	}
}

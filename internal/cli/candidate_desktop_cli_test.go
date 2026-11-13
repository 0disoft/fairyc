package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestCandidateDesktopCLI(t *testing.T) {
	candidateResultNativeGate(t)
	artifacts, err := filepath.Abs(filepath.Join("..", "..", ".cache", "desktop-cli-20261114"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artifacts, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAIRYC_CLOSURE_CLI_ARTIFACTS", artifacts)
	image := copiedClosureCLIHost(t, cdriver.OptimizedProfile)
	candidateDesktopInvocationParity(t, image)
	python, err := exec.LookPath("python")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, python, "-B", "tools/verify_win32_package_cli.py", "--compiler", image, "--companion", filepath.Join(filepath.Dir(image), "fairyc-candidate-build.exe"))
	command.Dir = filepath.Join("..", "..")
	out, err := command.CombinedOutput()
	if writeErr := os.WriteFile(filepath.Join(artifacts, "acceptance.json"), out, 0600); writeErr != nil {
		t.Fatal(writeErr)
	}
	var report struct {
		OK            bool `json:"ok"`
		GUILaunched   bool `json:"guiLaunched"`
		InputInjected bool `json:"inputInjected"`
		Published     bool `json:"published"`
	}
	if err != nil || json.Unmarshal(out, &report) != nil || !report.OK || report.GUILaunched || report.InputInjected || report.Published {
		t.Fatalf("candidate desktop CLI acceptance: %v %s", err, out)
	}
	t.Logf("authored candidate desktop CLI: %s", out)
}

func candidateDesktopInvocationParity(t *testing.T, image string) {
	t.Helper()
	root := t.TempDir()
	command := copiedClosureCommand{t: t, image: image, root: root}
	for _, args := range [][]string{
		{"new", "valid", "--uid", "test.valid", "--template", ""},
		{"new", "Invalid", "--uid", "invalid", "--template", ""},
		{"new", "valid", "--template", "win32", "--template", "win32"},
		{"package", "--release"}, {"package", "--json", "--json"}, {"package"},
	} {
		code, out, diagnostic := runForTest(args, root)
		if code != 2 || out != "" {
			t.Fatalf("bad Go invocation control %v: %d %q %q", args, code, out, diagnostic)
		}
		actual, problem := command.run(code, args...)
		if actual != out || problem != diagnostic {
			t.Fatalf("candidate invocation differs from Go %v:\nactual=%q\nexpected=%q", args, problem, diagnostic)
		}
	}
}

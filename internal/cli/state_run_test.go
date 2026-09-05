package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestStateRunRequiresExplicitClosedArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"transaction.fy"}, {"transaction.fy", "--profile", "fast"}, {"transaction.fy", "--profile", "low", "--source", "other.fy"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(append([]string{"experimental", "run-state"}, args...), t.TempDir(), &stdout, &stderr); code == ExitSuccess || stdout.Len() != 0 {
			t.Fatalf("invalid arguments admitted: %v", args)
		}
	}
}

func TestStateRunExecutesSourceAtBothNativeProfiles(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 native command")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("MSVC unavailable: %v", err)
	}
	data, err := os.ReadFile("../../examples/contract_state_native/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	for _, exit := range []string{"commit", "reject", "trap"} {
		text := string(data)
		call := "storage::" + exit + "()"
		want := "state exit: " + exit + "\nstate value: absent\n"
		if exit == "commit" {
			want = "state exit: commit\nstate value: 000000000000002a\n"
		}
		if exit == "trap" {
			call = `storage::trap("overflow")`
			want = "state exit: trap\ntrap category: overflow\nstate value: absent\n"
		}
		text = strings.Replace(text, "storage::commit()", call, 1)
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "transaction.fy"), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		for _, profile := range []string{"low", "optimized"} {
			var stdout, stderr bytes.Buffer
			code := Run([]string{"experimental", "run-state", "transaction.fy", "--profile", profile}, root, &stdout, &stderr)
			if code != ExitSuccess || strings.ReplaceAll(stdout.String(), "\r\n", "\n") != want || stderr.Len() != 0 {
				t.Fatalf("%s/%s: code=%d stdout=%q stderr=%q", exit, profile, code, stdout.String(), stderr.String())
			}
		}
	}
}

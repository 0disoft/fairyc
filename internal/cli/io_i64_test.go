package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestI64ConsoleEmission(t *testing.T) {
	s, e := os.ReadFile("../../examples/io_i64_quickstart/src/main.fy")
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, string(s))
	w, e := workspaceloader.Load(root)
	if e != nil {
		t.Fatal(e)
	}
	c, e := workspacecompiler.Check(w)
	if e != nil || !c.Verified {
		t.Fatalf("check: %v %+v", e, c.Semantic)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		unit, e := cbackend.Emit(c.MIR, tokenScannerEntry(t, c), profile)
		if e != nil {
			t.Fatal(e)
		}
		if !slices.Equal(unit.ForeignSymbols, []string{"fy_std_stderr_write_all_v0", "fy_std_stdout_write_all_v0"}) {
			t.Fatalf("unexpected foreign symbols, including allocation or i64 C ABI: %v", unit.ForeignSymbols)
		}
		path := filepath.Join("..", "..", ".cache", "io-i64-emission")
		if e := os.MkdirAll(path, 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(path, string(profile.Target)+".c"), []byte(unit.Source), 0600); e != nil {
			t.Fatal(e)
		}
	}
}

func TestI64ConsoleSourceBoundary(t *testing.T) {
	for _, name := range []string{"write_i64", "write_stderr_i64"} {
		for _, tc := range []struct{ name, effect, value, diagnostic string }{
			{"missing-console-effect", "", "1", "E7102"},
			{"wrong-type", "@effects(console)", "true", "E1006"},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				root := t.TempDir()
				writeVecI64ViewWorkspace(t, root, "module main; import std::io as io; "+tc.effect+" fn i32 main(){discard io::"+name+"("+tc.value+");return 0;}")
				code, _, diag := runForTest([]string{"check"}, root)
				if code != ExitSourceDiagnostics || !strings.Contains(diag, tc.diagnostic) {
					t.Fatalf("expected source rejection: exit=%d %s", code, diag)
				}
			})
		}
	}
}

package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsByteViewQuotedPageCReference(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	pages, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "pages.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/pages.fy", string(pages))
	writeTestSource(t, root, "src/main.fy", `module main;
import pages as pages;
@effects(console) fn i32 main() { return pages::run(); }`)
	reference, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "quoted_page.c"))
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, input string
		previous    bool
		start       uint32
	}{
		{"empty", `""`, false, 0},
		{"short", `"abc123"`, false, 0},
		{"previous", `"Z9"`, true, 1},
		{"long-first-page", `"` + strings.Repeat("a", 10000) + `"`, false, 0},
		{"long-later-page", `"` + strings.Repeat("a", 10000) + `"`, false, 4096},
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var diagnostics strings.Builder
		fairy, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
		if code != ExitSuccess {
			t.Fatalf("%s Fairy compile: %d %s", profile.Name, code, diagnostics.String())
		}
		c, err := cdriver.BuildWindowsHost(toolchain, profile, string(reference), "int fairyc_quoted_page_support(void) { return 0; }\n")
		if err != nil {
			t.Fatalf("%s C compile: %v", profile.Name, err)
		}
		for _, test := range cases {
			t.Run(profile.Name+"/"+test.name, func(t *testing.T) {
				input := []byte(test.input)
				frame := streamPageFrame(input, test.previous, test.start)
				values := streamObservation(t, input, test.previous)
				var expected strings.Builder
				for _, value := range values[int(test.start):min(int(test.start)+4096, len(values))] {
					fmt.Fprintf(&expected, "%d ", value)
				}
				for _, lane := range []struct {
					name     string
					artifact cdriver.WindowsHostArtifact
				}{{"fairy", fairy.artifact}, {"c", c}} {
					outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(lane.artifact, bytes.NewReader(frame))
					if err != nil || outcome.ExitCode != 0 || outcome.Stdout != expected.String() || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("%s: exit=%d output_bytes=%d expected_bytes=%d stderr=%q trap=%q err=%v", lane.name, outcome.ExitCode, len(outcome.Stdout), expected.Len(), outcome.Stderr, outcome.TrapControl, err)
					}
				}
			})
		}
		t.Logf("QUOTED_PAGE_C_REFERENCE profile=%s msvc=%s fairy_exe_bytes=%d c_exe_bytes=%d", profile.Name, toolchain.Version, len(fairy.artifact.Bytes), len(c.Bytes))
	}
}

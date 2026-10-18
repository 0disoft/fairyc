package cli

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestNativeWorkspaceRunOutcomes(t *testing.T) {
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	native, _ := cdriver.LookupNativeTarget(id)
	for _, test := range []struct {
		name, source, stdout, stderr string
		code                         int
	}{
		{name: "reserved ordinary result", source: "module main; fn i32 main() { return 131; }", stdout: "native workspace " + native.ResultLabel + ": 131 (", code: ExitSuccess},
		{name: "private trap receipt", source: "module main; fn i32 main() { assert(false); return 0; }", stderr: "trap: assertion-failed\n", code: ExitProgramTrap},
		{name: "unexpected program output", source: `module main; import std::io as io;
@effects(console)
fn i32 main() { assert(switch (io::write_i32(42)) { case .ok(_) : { true } case .err(_) : { false } }); return 0; }`, stderr: native.Platform + " workspace executable produced unexpected output", code: ExitBuildFailure},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeNativeWorkspace(t, root, true)
			writeTestSource(t, root, "src/main.fy", test.source)
			arguments := []string{"experimental", "run-workspace", "--target", string(id), "--profile", "low"}
			if len(native.Compilers) != 0 {
				arguments = append(arguments, "--cc", native.Compilers[0])
			}
			code, stdout, stderr := runForTest(arguments, root)
			if code != test.code || (test.stdout == "" && stdout != "") || (test.stdout != "" && !strings.HasPrefix(stdout, test.stdout)) || (test.stderr == "" && stderr != "") || (test.stderr != "" && !strings.Contains(stderr, test.stderr)) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
		})
	}
}

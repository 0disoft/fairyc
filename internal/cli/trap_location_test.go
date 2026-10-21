package cli

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestFormatNativeTrap(t *testing.T) {
	const contents = "module main;\r\n한글 assert(false);\n"
	sources := map[string]trapSource{"main.fy": {display: "src/한글.fy", bytes: []byte(contents)}}
	start := uint64(strings.Index(contents, "assert"))
	for _, test := range []struct {
		name     string
		location *cdriver.TrapLocation
		want     string
	}{
		{"UTF-8 column and CRLF", &cdriver.TrapLocation{Source: "main.fy", StartByte: start, EndByte: start + 14}, "  at src/한글.fy:2:4\n"},
		{"first byte", &cdriver.TrapLocation{Source: "main.fy"}, "  at src/한글.fy:1:1\n"},
		{"EOF", &cdriver.TrapLocation{Source: "main.fy", StartByte: uint64(len(contents)), EndByte: uint64(len(contents))}, "  at src/한글.fy:3:1\n"},
		{"missing", nil, ""},
		{"unknown file", &cdriver.TrapLocation{Source: "missing.fy"}, ""},
		{"reversed span", &cdriver.TrapLocation{Source: "main.fy", StartByte: 2, EndByte: 1}, ""},
		{"past EOF", &cdriver.TrapLocation{Source: "main.fy", EndByte: ^uint64(0)}, ""},
		{"inside UTF-8 start", &cdriver.TrapLocation{Source: "main.fy", StartByte: 15, EndByte: start}, ""},
		{"inside UTF-8 end", &cdriver.TrapLocation{Source: "main.fy", EndByte: 15}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			outcome := cdriver.NativeOutcome{TrapCategory: "assertion-failed", TrapLocation: test.location}
			if got := formatNativeTrap(outcome, sources); got != "trap: assertion-failed\n"+test.want {
				t.Fatalf("trap = %q", got)
			}
		})
	}
	sources["main.fy"] = trapSource{display: "src/bad.fy", bytes: []byte{0xff}}
	if got := formatNativeTrap(cdriver.NativeOutcome{TrapCategory: "bounds", TrapLocation: &cdriver.TrapLocation{Source: "main.fy"}}, sources); got != "trap: bounds\n" {
		t.Fatalf("invalid UTF-8: %q", got)
	}
	if got := formatNativeTrap(cdriver.NativeOutcome{}, sources); got != "" {
		t.Fatalf("ordinary return: %q", got)
	}
}

func TestWorkspaceTrapSourceSnapshot(t *testing.T) {
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	workspace, _, err := loadWorkspaceWithLock(root, false)
	if err != nil {
		t.Fatal(err)
	}
	sources := workspaceTrapSources(workspace)
	memberID, err := workspace.PackageID("libs/answer")
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "libs/answer/src/answer.fy", "changed after compilation")
	outcome := cdriver.NativeOutcome{TrapCategory: "bounds", TrapLocation: &cdriver.TrapLocation{Source: cbackend.PackageTrapSource(memberID, "answer.fy"), StartByte: 16, EndByte: 16}}
	if got := formatNativeTrap(outcome, sources); got != "trap: bounds\n  at libs/answer/src/answer.fy:3:1\n" {
		t.Fatalf("snapshot = %q", got)
	}
	// Identical package-local names retain separate compiled snapshots.
	for i := range workspace.Packages {
		if workspace.Packages[i].Path == "libs/answer" {
			workspace.Packages[i].Sources[0].Path = "main.fy"
		}
	}
	outcome.TrapLocation.Source = cbackend.PackageTrapSource(memberID, "main.fy")
	if got := formatNativeTrap(outcome, workspaceTrapSources(workspace)); got != "trap: bounds\n  at libs/answer/src/main.fy:3:1\n" {
		t.Fatalf("duplicate member source = %q", got)
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	outcome.TrapLocation = &cdriver.TrapLocation{Source: cbackend.PackageTrapSource(rootID, "main.fy")}
	if got := formatNativeTrap(outcome, workspaceTrapSources(workspace)); got != "trap: bounds\n  at src/main.fy:1:1\n" {
		t.Fatalf("duplicate root source = %q", got)
	}
	outcome.TrapLocation.StartByte, outcome.TrapLocation.EndByte = 16, 16
	// Control characters in a source path cannot insert a second output line.
	if got := formatNativeTrap(outcome, map[string]trapSource{outcome.TrapLocation.Source: {display: "src/a\nb.fy", bytes: make([]byte, 16)}}); got != "trap: bounds\n  at \"src/a\\nb.fy\":1:17\n" {
		t.Fatalf("escaped path = %q", got)
	}
}

func TestNativeTrapLocationsAcrossPackages(t *testing.T) {
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	native, _ := cdriver.LookupNativeTarget(id)
	for _, scenario := range []string{"root", "dynamic member", "constant member"} {
		memberTrap := scenario != "root"
		root := t.TempDir()
		writeNativeWorkspace(t, root, true)
		writeTestSource(t, root, "libs/answer/src/answer.fy", `module answer;
import main as helper;
pub fn i32 value() { return helper::value(); }
`)
		member := "module main;\npub fn i32 value() { return 42; }\n"
		if memberTrap {
			member = `module main;
import std::process as process;
pub fn i32 value() {
    let ref u8[] data = b"a";
    return data[process::arg_count()] as i32;
}`
		}
		if scenario == "constant member" {
			member = strings.Replace(member, "process::arg_count()", "1", 1)
		}
		writeTestSource(t, root, "libs/answer/src/main.fy", member)
		writeTestSource(t, root, "src/main.fy", `module main;
import answer as answer;
fn i32 main() {
    let i32 value = answer::value();
    assert(value == 0);
    return 0;
}`)
		want := "trap: assertion-failed\n  at src/main.fy:5:5\n"
		if memberTrap {
			want = "trap: bounds\n  at libs/answer/src/main.fy:5:12\n"
		}
		for _, release := range []bool{false, true} {
			args := []string{"run"}
			profile := "low"
			if release {
				args = append(args, "--release", "--locked")
				profile = "optimized"
			}
			experimental := []string{"experimental", "run-workspace", "--target", string(id), "--profile", profile, "--locked"}
			if len(native.Compilers) != 0 {
				experimental = append(experimental, "--cc", native.Compilers[0])
			}
			for _, command := range [][]string{args, experimental} {
				code, stdout, stderr := runForTest(command, root)
				if code != ExitProgramTrap || stdout != "" || stderr != want {
					t.Fatalf("scenario=%s args=%v: code=%d stdout=%q stderr=%q", scenario, command, code, stdout, stderr)
				}
			}
		}
	}
}

func TestStableNativeTrapLocations(t *testing.T) {
	if _, err := cdriver.HostNativeTarget(); err != nil {
		t.Skip(err)
	}
	for _, test := range []struct{ name, source, want string }{
		{"assert with program stderr", `module main;
import std::io as io;
@effects(console)
fn i32 main() {
    assert(switch (io::write_stderr("before\n")) { case .ok(_) : { true } case .err(_) : { false } });
    let ref u8[] text = b"한글"; assert(false);
    return 0;
}`, "before\ntrap: assertion-failed\n  at src/main.fy:6:32\n"},
		{"bounds", `module main;
fn i32 main() {
    let ref u8[] data = b"a";
    return data[1] as i32;
}`, "trap: bounds\n  at src/main.fy:4:12\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeNativeWorkspace(t, root, true)
			writeTestSource(t, root, "src/main.fy", test.source)
			for _, release := range []bool{false, true} {
				args := []string{"run"}
				if release {
					args = append(args, "--release", "--locked")
				}
				code, stdout, stderr := runForTest(args, root)
				if code != ExitProgramTrap || stdout != "" || stderr != test.want {
					t.Fatalf("release=%t: code=%d stdout=%q stderr=%q", release, code, stdout, stderr)
				}
			}
		})
	}
}

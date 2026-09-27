package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

// Keep the experiment on the real record parser: only its private cursor index
// changes representation, while the exported parse boundary remains usize.
func parserCursorStructWorkspace(t *testing.T, wrapped bool) string {
	t.Helper()
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	writeTestSource(t, root, "src/main.fy", "module main; import records as records; fn i32 main() { return 0; }")
	if !wrapped {
		return root
	}
	file := filepath.Join(root, "src", "records.fy")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ReplaceAll(string(data), "\r\n", "\n")
	replace := func(old, new string, count int) {
		t.Helper()
		if strings.Count(source, old) != count {
			t.Fatalf("parser cursor source changed: %q count=%d", old, strings.Count(source, old))
		}
		source = strings.ReplaceAll(source, old, new)
	}
	replace("struct Cursor { usize index;", "struct TokenIndex { usize raw; }\nstruct ByteOffset { usize raw; }\nstruct Cursor { TokenIndex index;", 1)
	replace("cursor.index", "cursor.index.raw", 29)
	replace("cursor.index.raw = cursor.index.raw + 1;", "cursor.index = TokenIndex { .raw = cursor.index.raw + 1 };", 2)
	replace("cursor.index.raw = type.next;", "cursor.index = TokenIndex { .raw = type.next };", 1)
	replace("cursor.index.raw = value.next;", "cursor.index = TokenIndex { .raw = value.next };", 1)
	replace("Cursor { .index,", "Cursor { .index = TokenIndex { .raw = index },", 1)
	writeTestSource(t, root, "src/records.fy", source)
	return root
}

func TestParserCursorStructBoundary(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		root := parserCursorStructWorkspace(t, wrapped)
		code, stdout, stderr := runForTest([]string{"check"}, root)
		if code != ExitSuccess || !strings.HasPrefix(stdout, "check ok: ") || stderr != "" {
			t.Fatalf("valid wrapped=%v: %d %q %s", wrapped, code, stdout, stderr)
		}
		file := filepath.Join(root, "src", "records.fy")
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		source := string(data)
		needle := "fn void advance(ref scanner::Stream stream, ref mut Cursor cursor) {"
		if strings.Count(source, needle) != 1 {
			t.Fatal("advance boundary changed")
		}
		wrong := "cursor.index = 7;"
		if wrapped {
			wrong = "cursor.index = ByteOffset { .raw = 7 };"
		}
		writeTestSource(t, root, "src/records.fy", strings.Replace(source, needle, needle+"\n    "+wrong, 1))
		code, stdout, stderr = runForTest([]string{"check"}, root)
		if wrapped {
			if code != 3 || stdout != "" || !strings.Contains(stderr, "E1006") {
				t.Fatalf("wrong domain admitted or misdiagnosed: %d %q %s", code, stdout, stderr)
			}
		} else if code != ExitSuccess || !strings.HasPrefix(stdout, "check ok: ") || stderr != "" {
			t.Fatalf("raw index rejected: %d %q %s", code, stdout, stderr)
		}
	}
}

func TestWindowsParserCursorStructFieldStoreNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	type sizes struct{ exe, unit, runtime int }
	baseline := map[string]sizes{}
	for _, wrapped := range []bool{false, true} {
		root := parserCursorStructWorkspace(t, wrapped)
		for _, name := range []string{"syntax.fy", "integrity.fy", "file_syntax.fy"} {
			data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
			if err != nil {
				t.Fatal(err)
			}
			writeTestSource(t, root, "src/"+name, string(data))
		}
		writeTestSource(t, root, "src/main.fy", parserProbePrelude+`fn i32 main() {
    return switch (make(b"module main; struct R { u32 x; }")) {
        case .ok(parsed): {
            assert(syntax::problem_count(ref parsed) == 0);
            assert(syntax::field_count(ref parsed) == 1);
            0
        }
        case .err(_): { 1 }
    };
}`)
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			var diagnostics strings.Builder
			compiled, code := compileWindowsWorkspace(root,
				workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile},
				workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("parser cursor wrapped=%v profile=%s: %d %s", wrapped, profile.Name, code, diagnostics.String())
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(compiled.artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("parser cursor wrapped=%v profile=%s: %+v %v", wrapped, profile.Name, outcome, err)
			}
			current := sizes{exe: len(compiled.artifact.Bytes)}
			for _, object := range compiled.artifact.Objects {
				switch object.Name {
				case "unit.obj":
					current.unit = len(object.Bytes)
				case "runtime.obj":
					current.runtime = len(object.Bytes)
				}
			}
			if current.exe == 0 || current.unit == 0 || current.runtime == 0 {
				t.Fatalf("incomplete parser cursor artifacts: %+v", current)
			}
			if !wrapped {
				baseline[profile.Name] = current
			} else {
				previous := baseline[profile.Name]
				t.Logf("parser cursor %s: baseline=%+v wrapped=%+v delta=%+v", profile.Name, previous, current,
					sizes{exe: current.exe - previous.exe, unit: current.unit - previous.unit, runtime: current.runtime - previous.runtime})
			}
		}
	}
}

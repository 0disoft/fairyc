package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestCandidateTestCacheWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("real Windows cache transport requires Windows amd64")
	}
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	// This host regression executes interpreter/listing only. Exclude the
	// unrelated Windows native compiler from the optimized probe's reachable C.
	source, err := os.ReadFile(filepath.Join(root, "src", "project_test_cli.fy"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(source), "import native_cli as native;") != 1 {
		t.Fatal("test executor boundary changed")
	}
	writeTestSource(t, root, "src/project_test_cli.fy", strings.Replace(string(source), "import native_cli as native;", "import test_cache_native_probe as native;", 1))
	writeTestSource(t, root, "src/test_cache_native_probe.fy", `module test_cache_native_probe;
import project_compiler_core as core;
import project_compiler_input as input;
import project_test_options as options;
import std::vec as v;
pub fn i32 tests(core::Complete initial,ref input::Input projected,ref options::TestOptions selected,ref v::Vec<options::TestIdentity> tests,ref v::Vec<usize> chosen,ref u8[] root,u32 lock){discard initial;return 6;}
`)
	writeTestSource(t, root, "src/main.fy", candidateTestCacheProbe)
	unit, support, tc := windowsCandidateProbe(t, root)
	// Full compiler integration uses low: MSVC /O2 exceeds the one-minute
	// per-command budget for this unit even with native execution excluded.
	for _, profile := range []cdriver.Profile{cdriver.LowProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support})
			if err != nil {
				t.Fatal(err)
			}
			client := t.TempDir()
			writeTestSource(t, client, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.cache-client\"\nname = \"client\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n")
			writeTestSource(t, client, "src/main.fy", "module main; fn i32 main(){return 0;}\n")
			writeTestSource(t, client, "tests/answer.fy", "module tests::answer; @test fn void fails(){assert(false);}\n")
			cache := filepath.Join(client, "cache")
			if err := os.Mkdir(cache, 0700); err != nil {
				t.Fatal(err)
			}
			run := func(args []string, code int32, status string) string {
				t.Helper()
				result, err := tc.Execute(artifact, cdriver.NativeExecutionOptions{Arguments: args, Timeout: 60 * time.Second})
				if err != nil || result.ExitCode != code || !strings.Contains(result.Stderr, status) {
					t.Fatalf("%v: %+v %v", args, result, err)
				}
				return result.Stdout
			}
			run([]string{"--options"}, 0, "")
			args := []string{filepath.ToSlash(client), "--json", "--cache-root", filepath.ToSlash(cache), "--cache-status"}
			if out := run(args, 5, "cache: cold stored\n"); !strings.Contains(out, `"failed":1`) {
				t.Fatal(out)
			}
			if out := run(args, 5, "cache: units hit\n"); !strings.Contains(out, `"failed":1`) {
				t.Fatal("cached failure was not executed", out)
			}
			list := append(append([]string{}, args...), "--list", "--locked")
			if out := run(list, 0, "cache: units hit\n"); !strings.Contains(out, `"count":1`) {
				t.Fatal(out)
			}
			writeTestSource(t, client, "tests/answer.fy", "module tests::answer; @test fn void passes(){assert(true);}\n")
			if out := run(args, 0, "cache: cold stored\n"); !strings.Contains(out, `"passed":1`) {
				t.Fatal(out)
			}
			run(args, 0, "cache: units hit\n")
			entries, err := os.ReadDir(cache)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if err := os.WriteFile(filepath.Join(cache, entry.Name()), []byte("corrupt"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			run(args, 0, "cache: cold stored\n")
			writeTestSource(t, client, "tests/answer.fy", "module tests::answer; @test fn void invalid(){let i32 x=true;}\n")
			if out := run(args, 3, "cache: cold uncached\n"); out != "" {
				t.Fatal(out)
			}
			writeTestSource(t, client, "tests/answer.fy", "module tests::answer; @test fn void passes(){assert(true);}\n")
			fallback := []string{filepath.ToSlash(client), "--cache-root", filepath.ToSlash(filepath.Join(client, "missing")), "--cache-status"}
			run(fallback, 0, "cache: cold uncached\n")
			if _, err := os.Stat(filepath.Join(client, "missing")); !os.IsNotExist(err) {
				t.Fatal("created missing cache root", err)
			}
		})
	}
}

const candidateTestCacheProbe = `module main;
import project_test_cli as cli;
import project_test_options as options;
import native_cli_options as arguments;
import std::bytes as bytes;
import std::process as process;
import std::vec as v;
fn void probe_options(){
    var argv=v::empty<bytes::Buffer>();assert(options::test_add(ref mut argv,b"--cache-root") && options::test_add(ref mut argv,b"cache root"));
    assert(!options::parse_test_options(ref argv).is_ok);
    assert(options::test_add(ref mut argv,b"--cache-status") && options::test_add(ref mut argv,b"--native") && options::test_add(ref mut argv,b"--") && options::test_add(ref mut argv,b"--cache-status"));
    let p=switch(options::parse_test_options_cached(ref argv)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(options::cache_status(ref p) && bytes::equal(options::cache_root(ref p),b"cache root") && v::len<bytes::Buffer>(options::arguments(ref p))==1);
    assert(v::clear<bytes::Buffer>(ref mut argv)==6);assert(options::test_add(ref mut argv,b"--cache-status"));assert(!options::parse_test_options_cached(ref argv).is_ok);
    assert(v::clear<bytes::Buffer>(ref mut argv)==1);assert(options::test_add(ref mut argv,b"--cache-root"));assert(!options::parse_test_options_cached(ref argv).is_ok);
    assert(options::test_add(ref mut argv,b"bad\0path"));assert(!options::parse_test_options_cached(ref argv).is_ok);
    assert(v::clear<bytes::Buffer>(ref mut argv)==2);assert(options::test_add(ref mut argv,b"--cache-root") && options::test_add(ref mut argv,b"root") && options::test_add(ref mut argv,b"--cache-root") && options::test_add(ref mut argv,b"other"));assert(!options::parse_test_options_cached(ref argv).is_ok);
}
@effects(filesystem,console,environment,clock) fn i32 main(){
    var u8[4096] raw=[0;4096];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 2;}};let ref u8[] root=ref raw;
    if(bytes::equal(root[..n],b"--options")){probe_options();return 0;}
    let argv=switch(arguments::read(2)){case .ok(p):{p}case .err(_):{return 2;}};
    let selected=switch(options::parse_test_options_cached(ref argv)){case .ok(p):{p}case .err(_):{return 2;}};
    return cli::run_test_list(root[..n],ref selected);
}
`

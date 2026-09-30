package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsDirectoryNativeLifecycle(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	tree := filepath.Join(root, "한글 directory")
	empty := filepath.Join(root, "empty")
	for _, dir := range []string{filepath.Join(tree, "sub"), empty} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range map[string]string{"한글.txt": "abc", "plain.bin": "12345"} {
		if err := os.WriteFile(filepath.Join(tree, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "root-link")
	linkProof := ""
	childLinks := "0"
	if err := os.Symlink(tree, link); err == nil {
		linkProof = "assert(switch (fs::read_dir_utf8(" + strconv.Quote(filepath.ToSlash(link)) + ")) { case .ok(_) : { false } case .err(_) : { true } });\n" +
			"assert(switch (fs::read_dir_utf8(" + strconv.Quote(filepath.ToSlash(filepath.Join(link, "sub"))) + ")) { case .ok(_) : { false } case .err(_) : { true } });\n"
	} else {
		t.Logf("root/prefix symlink execution not exercised: %v", err)
	}
	if err := os.Symlink(empty, filepath.Join(tree, "external-link")); err == nil {
		childLinks = "1"
	} else {
		t.Logf("child reparse entry execution not exercised: %v", err)
	}
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.directory-native\"\nname = \"directory-native\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	program := `module main;
import std::fs as fs;
import std::bytes as bytes;
import std::text as text;
@effects(filesystem)
fn i32 inspect(fs::ReadDir input, bool empty) {
    var fs::ReadDir directory = input;
    var bytes::Buffer name = bytes::empty();
    assert(switch (bytes::resize_zeroed(ref mut name, 1)) { case .ok(_) : { true } case .err(_) : { false } });
    if (!empty) {
        assert(switch (fs::next_dir(ref mut directory, bytes::view_mut(ref mut name))) {
            case .ok(_) : { false }
            case .err(error) : { fs::error_os_code(error) == 122 && fs::error_completed(error) > 1 }
        });
    }
    assert(switch (bytes::resize_zeroed(ref mut name, 1024)) { case .ok(_) : { true } case .err(_) : { false } });
    var usize files = 0;
    var usize directories = 0;
    var usize links = 0;
    var usize total = 0;
    while (true) {
        let fs::DirEntry entry = switch (fs::next_dir(ref mut directory, bytes::view_mut(ref mut name))) {
            case .ok(entry) : { entry }
            case .err(_) : { return 1; }
        };
        if (!entry.has_entry) { break; }
        assert(entry.name_length > 0);
        assert(switch (text::validate_utf8(bytes::view(ref name)[..entry.name_length])) { case .ok(_) : { true } case .err(_) : { false } });
        if (entry.is_reparse) {
            assert(!fs::path_kind_is_file(entry.kind) && !fs::path_kind_is_directory(entry.kind));
            links += 1;
        } else if (fs::path_kind_is_file(entry.kind)) { files += 1; total += entry.size; }
        else if (fs::path_kind_is_directory(entry.kind)) { directories += 1; }
        else { return 2; }
    }
    assert(switch (fs::next_dir(ref mut directory, bytes::view_mut(ref mut name))) { case .ok(entry) : { !entry.has_entry && entry.name_length == 0 } case .err(_) : { false } });
    assert(switch (fs::close_dir(directory)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(if (empty) { files == 0 && directories == 0 && total == 0 && links == 0 } else { files == 2 && directories == 1 && total == 8 && links == CHILD_LINKS });
    return 0;
}
@effects(filesystem)
fn i32 early_exit(fs::ReadDir directory) { return 0; }
@effects(filesystem)
fn i32 main() {
    assert(switch (fs::read_dir_utf8(TREE)) { case .ok(directory) : { inspect(directory, false) == 0 } case .err(_) : { false } });
    assert(switch (fs::read_dir_utf8(EMPTY)) { case .ok(directory) : { inspect(directory, true) == 0 } case .err(_) : { false } });
    assert(switch (fs::read_dir_utf8(TREE)) { case .ok(directory) : { early_exit(directory) == 0 } case .err(_) : { false } });
    assert(switch (fs::read_dir_utf8(MISSING)) { case .ok(_) : { false } case .err(_) : { true } });
    assert(switch (fs::read_dir_utf8(FILE)) { case .ok(_) : { false } case .err(_) : { true } });
    assert(switch (fs::read_dir_utf8("")) { case .ok(_) : { false } case .err(_) : { true } });
    assert(switch (fs::read_dir_utf8("*")) { case .ok(_) : { false } case .err(_) : { true } });
    LINK_PROOF
    return 0;
}
`
	replace := strings.NewReplacer("TREE", strconv.Quote(filepath.ToSlash(tree)), "EMPTY", strconv.Quote(filepath.ToSlash(empty)), "MISSING", strconv.Quote(filepath.ToSlash(filepath.Join(root, "missing"))), "FILE", strconv.Quote(filepath.ToSlash(filepath.Join(tree, "plain.bin"))), "LINK_PROOF", linkProof, "CHILD_LINKS", childLinks)
	writeTestSource(t, root, "src/main.fy", replace.Replace(program))
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			var diagnostics strings.Builder
			compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("compile: %d %s", code, diagnostics.String())
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(compiled.artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("native directory lifecycle: %+v %v", outcome, err)
			}
		})
	}
}

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagecontent"
)

func TestStableWorkspaceTestRunsIsolatedVoidTestsInCanonicalOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/zeta.fy", `module tests::zeta;
import library as library;
@test
fn void second() { assert(library::answer() == 42); }
`)
	writeWorkspaceFile(t, root, "tests/alpha.fy", `module tests::alpha;
import library as library;
fn bool helper() { return library::answer() == 42; }
@test
fn void first() {
    let i32 value = library::answer();
    assert(value == 42);
}
`)

	code, stdout, stderr := runForTest([]string{"test"}, root)
	want := "test tests::alpha::first ... ok\n" +
		"test tests::zeta::second ... ok\n" +
		"test result: ok. 2 passed; 0 failed; lock updated\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	if matches, err := filepath.Glob(filepath.Join(root, "build", "*")); err != nil || len(matches) != 0 {
		t.Fatalf("test build artifacts = %v, %v", matches, err)
	}

	lockPath := filepath.Join(root, lockfile.FileName)
	lockBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	lockTimestamp := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(lockPath, lockTimestamp, lockTimestamp); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runForTest([]string{"test", "--locked"}, root)
	want = strings.Replace(want, "lock updated", "lock verified", 1)
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("locked test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	lockedBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	lockedInfo, err := os.Stat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(lockedBytes, lockBytes) || !lockedInfo.ModTime().Equal(lockTimestamp) {
		t.Fatalf("locked test modified fairy.lock: bytesEqual=%t modTime=%s", bytes.Equal(lockedBytes, lockBytes), lockedInfo.ModTime())
	}
}

func TestStableWorkspaceTestRunsRemainingTestsAfterATrap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/outcomes.fy", `module tests::outcomes;
@test
fn void first_fails() { assert(false); }
@test
fn void second_passes() { assert(true); }
`)

	code, stdout, stderr := runForTest([]string{"test"}, root)
	want := "test tests::outcomes::first_fails ... FAILED (assertion-failed)\n" +
		"test tests::outcomes::second_passes ... ok\n" +
		"test result: FAILED. 1 passed; 1 failed; lock updated\n"
	if code != ExitProgramTrap || stdout != want || stderr != "" {
		t.Fatalf("test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
}

func TestStableWorkspaceTestSelectsOneReachableMemberByPackageUID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.member-test.app"
name = "member-test-app"
version = "1.0.0"
language = "fairyc-language/10"
source = "src"
[workspace]
members = ["libs/answer", "libs/unused"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\nimport answer as answer;\nfn i32 value() { return answer::value(); }\n")
	writeWorkspaceFile(t, root, "libs/answer/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.member-test.answer"
name = "answer"
version = "1.0.0"
language = "fairyc-language/10"
source = "src"
`)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer;\npub fn i32 value() { return 42; }\n")
	writeWorkspaceFile(t, root, "libs/unused/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.member-test.unused"
name = "unused"
version = "1.0.0"
language = "fairyc-language/10"
source = "src"
`)
	writeWorkspaceFile(t, root, "libs/unused/src/unused.fy", "module unused;\npub fn i32 value() { return 7; }\n")
	writeWorkspaceFile(t, root, "libs/unused/tests/unused.fy", "module tests::unused;\n@test\nfn void unused_passes() { assert(true); }\n")
	writeWorkspaceFile(t, root, "tests/root.fy", "module tests::root;\n@test\nfn void root_passes() { assert(true); }\n")
	writeWorkspaceFile(t, root, "libs/answer/tests/member.fy", `module tests::member;
import answer as answer;
@test
fn void member_passes() { assert(answer::value() == 42); }
`)

	code, stdout, stderr := runForTest([]string{"test"}, root)
	if code != ExitSuccess || stdout != "test tests::root::root_passes ... ok\ntest result: ok. 1 passed; 0 failed; lock updated\n" || stderr != "" {
		t.Fatalf("default root test = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--package", "example.member-test.answer", "--json", "--locked"}, root)
	want := `{"schemaVersion":"fairyc.test-report/v1","engine":"interpreter","target":null,"profile":null,"tests":[{"identity":"tests::member::member_passes","outcome":"passed","trapCategory":null}],"count":1,"passed":1,"failed":0,"status":"ok","lock":"verified"}` + "\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("selected member test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	code, stdout, stderr = runForTest([]string{"test", "--list", "--package", "example.member-test.answer", "--locked"}, root)
	if code != ExitSuccess || stdout != "test tests::member::member_passes\ntest list: 1 discovered; lock verified\n" || stderr != "" {
		t.Fatalf("selected member list = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--workspace", "--json", "--locked"}, root)
	want = `{"schemaVersion":"fairyc.test-report/v2","engine":"interpreter","target":null,"profile":null,"tests":[{"identity":"example.member-test.answer::tests::member::member_passes","outcome":"passed","trapCategory":null},{"identity":"example.member-test.app::tests::root::root_passes","outcome":"passed","trapCategory":null}],"count":2,"passed":2,"failed":0,"status":"ok","lock":"verified"}` + "\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("workspace aggregate test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	code, stdout, stderr = runForTest([]string{"test", "--workspace", "--list", "--json", "--locked"}, root)
	want = `{"schemaVersion":"fairyc.test-list/v2","tests":["example.member-test.answer::tests::member::member_passes","example.member-test.app::tests::root::root_passes"],"count":2,"lock":"verified"}` + "\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("workspace aggregate list = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	code, stdout, stderr = runForTest([]string{"test", "--workspace", "--list", "--json", "--locations", "--locked"}, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("located workspace list = (%d, %q, %q)", code, stdout, stderr)
	}
	var locatedWorkspace struct {
		SchemaVersion string `json:"schemaVersion"`
		Locations     []stableTestLocation
	}
	if err := json.Unmarshal([]byte(stdout), &locatedWorkspace); err != nil {
		t.Fatal(err)
	}
	if locatedWorkspace.SchemaVersion != "fairyc.test-list/v3" || len(locatedWorkspace.Locations) != 2 ||
		locatedWorkspace.Locations[0].Identity != "example.member-test.answer::tests::member::member_passes" ||
		locatedWorkspace.Locations[0].Source.PackageUID != "example.member-test.answer" ||
		locatedWorkspace.Locations[0].Source.Path != "libs/answer/tests/member.fy" ||
		locatedWorkspace.Locations[1].Identity != "example.member-test.app::tests::root::root_passes" ||
		locatedWorkspace.Locations[1].Source.PackageUID != "example.member-test.app" ||
		locatedWorkspace.Locations[1].Source.Path != "tests/root.fy" {
		t.Fatalf("located workspace list = %+v", locatedWorkspace)
	}
	code, stdout, stderr = runForTest([]string{
		"test", "--package", "example.member-test.answer", "--file", "libs/answer/tests/member.fy", "--locked",
	}, root)
	if code != ExitSuccess || stdout != "test tests::member::member_passes ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" || stderr != "" {
		t.Fatalf("selected member file = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{
		"test", "--package", "example.member-test.answer", "--dir", "libs/answer/tests", "--locked",
	}, root)
	if code != ExitSuccess || stdout != "test tests::member::member_passes ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" || stderr != "" {
		t.Fatalf("selected member directory = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--workspace", "--file", "tests/root.fy", "--locked"}, root)
	if code != ExitSuccess || stdout != "test example.member-test.app::tests::root::root_passes ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" || stderr != "" {
		t.Fatalf("workspace root file = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--workspace", "--dir", "tests", "--locked"}, root)
	if code != ExitSuccess || stdout != "test example.member-test.app::tests::root::root_passes ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" || stderr != "" {
		t.Fatalf("workspace root directory = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{
		"test", "--package", "example.member-test.answer", "--file", "tests/member.fy", "--locked",
	}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, `test file selection was not found: "tests/member.fy"`) {
		t.Fatalf("package-relative selected member file = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{
		"test", "--workspace", "--exact", "example.member-test.answer::tests::member::member_passes", "--locked",
	}, root)
	want = "test example.member-test.answer::tests::member::member_passes ... ok\n" +
		"test result: ok. 1 passed; 0 failed; lock verified\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("workspace exact test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	code, stdout, stderr = runForTest([]string{"test", "--workspace", "--exact", "tests::member::member_passes", "--locked"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, `exact test selection was not found: "tests::member::member_passes"`) {
		t.Fatalf("unqualified workspace exact test = (%d, %q, %q)", code, stdout, stderr)
	}

	for _, arguments := range [][]string{
		{"test", "--package"},
		{"test", "--package", "missing.package"},
		{"test", "--package", "example.member-test.answer", "--package", "example.member-test.answer"},
	} {
		code, stdout, stderr = runForTest(arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "package") {
			t.Fatalf("invalid package selection %v = (%d, %q, %q)", arguments, code, stdout, stderr)
		}
	}
	for _, testCase := range []struct {
		arguments []string
		wantError string
	}{
		{arguments: []string{"test", "--workspace", "--workspace"}, wantError: "received --workspace more than once"},
		{arguments: []string{"test", "--workspace", "--package", "example.member-test.answer"}, wantError: "does not combine --workspace with --package"},
	} {
		code, stdout, stderr = runForTest(testCase.arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, testCase.wantError) {
			t.Fatalf("invalid workspace selection %v = (%d, %q, %q)", testCase.arguments, code, stdout, stderr)
		}
	}
	code, stdout, stderr = runForTest([]string{"test", "--package", "example.member-test.unused", "--locked"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "not root-reachable") {
		t.Fatalf("unreachable package selection = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableWorkspaceTestWritesCanonicalJSONExecutionReport(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/outcomes.fy", `module tests::outcomes;
@test
fn void first_fails() { assert(false); }
@test
fn void second_passes() { assert(true); }
`)

	code, stdout, stderr := runForTest([]string{"test", "--json"}, root)
	want := `{"schemaVersion":"fairyc.test-report/v1","engine":"interpreter","target":null,"profile":null,"tests":[{"identity":"tests::outcomes::first_fails","outcome":"failed","trapCategory":"assertion-failed"},{"identity":"tests::outcomes::second_passes","outcome":"passed","trapCategory":null}],"count":2,"passed":1,"failed":1,"status":"failed","lock":"updated"}` + "\n"
	if code != ExitProgramTrap || stdout != want || stderr != "" {
		t.Fatalf("JSON test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
}

func TestStableWorkspaceTestReportsExactSourceLocationsOnRequest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	source := `module tests::location;

@test
fn void passes() { assert(true); }
`
	writeWorkspaceFile(t, root, "tests/location.fy", source)
	wantFile, wantLineMap := packagecontent.ComputeFileIdentities([]byte(source))
	wantStart := strings.Index(source, "passes")

	code, stdout, stderr := runForTest([]string{"test", "--list", "--locations", "--json"}, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("located JSON list = (%d, %q, %q)", code, stdout, stderr)
	}
	var listed struct {
		SchemaVersion string `json:"schemaVersion"`
		Tests         []string
		Locations     []stableTestLocation `json:"locations"`
	}
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.SchemaVersion != "fairyc.test-list/v3" ||
		!slices.Equal(listed.Tests, []string{"tests::location::passes"}) || len(listed.Locations) != 1 {
		t.Fatalf("located list = %+v", listed)
	}
	location := listed.Locations[0]
	if location.Identity != "tests::location::passes" || location.Source.PackageUID != "example.stable-test.app" ||
		location.Source.Path != "tests/location.fy" || location.Source.Line != 4 || location.Source.ByteColumn != 9 ||
		location.Source.StartByte != wantStart || location.Source.EndByte != wantStart+len("passes") ||
		location.Source.FileIdentity != wantFile.String() || location.Source.LineMapIdentity != wantLineMap.String() {
		t.Fatalf("located list source = %+v", location)
	}

	code, stdout, stderr = runForTest([]string{"test", "--locations", "--json", "--locked"}, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("located JSON execution = (%d, %q, %q)", code, stdout, stderr)
	}
	var executed struct {
		SchemaVersion string `json:"schemaVersion"`
		Tests         []stableTestExecutionOutcome
	}
	if err := json.Unmarshal([]byte(stdout), &executed); err != nil {
		t.Fatal(err)
	}
	if executed.SchemaVersion != "fairyc.test-report/v3" || len(executed.Tests) != 1 ||
		executed.Tests[0].Identity != location.Identity || executed.Tests[0].Source == nil ||
		*executed.Tests[0].Source != location.Source {
		t.Fatalf("located execution = %+v", executed)
	}

	code, stdout, stderr = runForTest([]string{"test", "--list", "--locations", "--locked"}, root)
	wantText := "test tests::location::passes at tests/location.fy:4:9\n" +
		"test list: 1 discovered; lock verified\n"
	if code != ExitSuccess || stdout != wantText || stderr != "" {
		t.Fatalf("located text list = (%d, %q, %q), want stdout %q", code, stdout, stderr, wantText)
	}
}

func TestStableWorkspaceTestSelectsExactPortableFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/alpha.fy", `module tests::alpha;
@test
fn void first() { assert(true); }
@test
fn void second() { assert(true); }
`)
	writeWorkspaceFile(t, root, "tests/beta.fy", `module tests::beta;
@test
fn void excluded_failure() { assert(false); }
`)
	writeWorkspaceFile(t, root, "tests/helpers.fy", `module tests::helpers;
fn bool helper() { return true; }
`)

	code, stdout, stderr := runForTest([]string{"test", "--file", "tests/alpha.fy"}, root)
	want := "test tests::alpha::first ... ok\n" +
		"test tests::alpha::second ... ok\n" +
		"test result: ok. 2 passed; 0 failed; lock updated\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("file-selected test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}

	code, stdout, stderr = runForTest([]string{"test", "--list", "--json", "--locations", "--file", "tests/beta.fy", "--locked"}, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("file-selected located list = (%d, %q, %q)", code, stdout, stderr)
	}
	var listed struct {
		SchemaVersion string `json:"schemaVersion"`
		Tests         []string
		Locations     []stableTestLocation
	}
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.SchemaVersion != "fairyc.test-list/v3" ||
		!slices.Equal(listed.Tests, []string{"tests::beta::excluded_failure"}) ||
		len(listed.Locations) != 1 || listed.Locations[0].Source.Path != "tests/beta.fy" {
		t.Fatalf("file-selected located list = %+v", listed)
	}

	code, stdout, stderr = runForTest([]string{
		"test", "--list", "--file", "tests/beta.fy", "--file", "tests/alpha.fy", "--locked",
	}, root)
	want = "test tests::alpha::first\n" +
		"test tests::alpha::second\n" +
		"test tests::beta::excluded_failure\n" +
		"test list: 3 discovered; lock verified\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("multi-file selected list = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}

	code, stdout, stderr = runForTest([]string{"test", "--file", "tests/helpers.fy", "--locked"}, root)
	if code != ExitSuccess || stdout != "test result: ok. 0 passed; 0 failed; lock verified\n" || stderr != "" {
		t.Fatalf("empty admitted test file = (%d, %q, %q)", code, stdout, stderr)
	}

	for _, testCase := range []struct {
		arguments []string
		wantError string
	}{
		{arguments: []string{"test", "--file"}, wantError: "requires a test path after --file"},
		{arguments: []string{"test", "--file", "tests/missing.fy"}, wantError: `test file selection was not found: "tests/missing.fy"`},
		{arguments: []string{"test", "--file", `tests\\alpha.fy`}, wantError: `test file selection was not found: "tests\\\\alpha.fy"`},
		{arguments: []string{"test", "--file", "tests/alpha.fy", "--file", "tests/alpha.fy"}, wantError: `received test file "tests/alpha.fy" more than once`},
		{arguments: []string{"test", "--file", "tests/alpha.fy", "--exact", "tests::alpha::first"}, wantError: "does not combine --file with --exact"},
	} {
		code, stdout, stderr = runForTest(testCase.arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, testCase.wantError) {
			t.Fatalf("invalid file selection %v = (%d, %q, %q), want %q", testCase.arguments, code, stdout, stderr, testCase.wantError)
		}
	}

	writeWorkspaceFile(t, root, "tests/beta.fy", `module tests::beta;
@test
fn void invalid_unselected_file() { missing(); }
`)
	code, stdout, stderr = runForTest([]string{"test", "--file", "tests/alpha.fy", "--locked"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || stderr == "" {
		t.Fatalf("selected file with invalid unselected overlay = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableWorkspaceTestSelectsExactPortableDirectories(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/unit/alpha.fy", `module tests::unit::alpha;
@test
fn void first() { assert(true); }
`)
	writeWorkspaceFile(t, root, "tests/unit/nested/beta.fy", `module tests::unit::nested::beta;
@test
fn void second() { assert(true); }
`)
	writeWorkspaceFile(t, root, "tests/integration/failure.fy", `module tests::integration::failure;
@test
fn void excluded_failure() { assert(false); }
`)
	writeWorkspaceFile(t, root, "tests/empty/helpers.fy", `module tests::empty::helpers;
fn bool helper() { return true; }
`)

	code, stdout, stderr := runForTest([]string{"test", "--dir", "tests/unit"}, root)
	want := "test tests::unit::alpha::first ... ok\n" +
		"test tests::unit::nested::beta::second ... ok\n" +
		"test result: ok. 2 passed; 0 failed; lock updated\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("directory-selected test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}

	code, stdout, stderr = runForTest([]string{"test", "--list", "--json", "--locations", "--dir", "tests/unit/nested", "--locked"}, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("directory-selected located list = (%d, %q, %q)", code, stdout, stderr)
	}
	var listed struct {
		SchemaVersion string `json:"schemaVersion"`
		Tests         []string
		Locations     []stableTestLocation
	}
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.SchemaVersion != "fairyc.test-list/v3" ||
		!slices.Equal(listed.Tests, []string{"tests::unit::nested::beta::second"}) ||
		len(listed.Locations) != 1 || listed.Locations[0].Source.Path != "tests/unit/nested/beta.fy" {
		t.Fatalf("directory-selected located list = %+v", listed)
	}

	code, stdout, stderr = runForTest([]string{
		"test", "--list", "--dir", "tests/integration", "--dir", "tests/unit/nested", "--locked",
	}, root)
	want = "test tests::integration::failure::excluded_failure\n" +
		"test tests::unit::nested::beta::second\n" +
		"test list: 2 discovered; lock verified\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("multi-directory selected list = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}

	code, stdout, stderr = runForTest([]string{"test", "--dir", "tests/empty", "--locked"}, root)
	if code != ExitSuccess || stdout != "test result: ok. 0 passed; 0 failed; lock verified\n" || stderr != "" {
		t.Fatalf("empty admitted test directory = (%d, %q, %q)", code, stdout, stderr)
	}

	for _, testCase := range []struct {
		arguments []string
		wantError string
	}{
		{arguments: []string{"test", "--dir"}, wantError: "requires a test directory after --dir"},
		{arguments: []string{"test", "--dir", "tests/missing"}, wantError: `test directory selection was not found: "tests/missing"`},
		{arguments: []string{"test", "--dir", `tests\unit`}, wantError: `test directory selection was not found: "tests\\unit"`},
		{arguments: []string{"test", "--dir", "Tests/unit"}, wantError: `test directory selection was not found: "Tests/unit"`},
		{arguments: []string{"test", "--dir", "tests/../tests/unit"}, wantError: `test directory selection was not found: "tests/../tests/unit"`},
		{arguments: []string{"test", "--dir", "tests/unit/"}, wantError: `test directory selection was not found: "tests/unit/"`},
		{arguments: []string{"test", "--dir", "tests/unit", "--dir", "tests/unit"}, wantError: `received test directory "tests/unit" more than once`},
		{arguments: []string{"test", "--dir", "tests/unit", "--file", "tests/unit/alpha.fy"}, wantError: "does not combine --dir with --file or --exact"},
		{arguments: []string{"test", "--dir", "tests/unit", "--exact", "tests::unit::alpha::first"}, wantError: "does not combine --dir with --file or --exact"},
	} {
		code, stdout, stderr = runForTest(testCase.arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, testCase.wantError) {
			t.Fatalf("invalid directory selection %v = (%d, %q, %q), want %q", testCase.arguments, code, stdout, stderr, testCase.wantError)
		}
	}

	writeWorkspaceFile(t, root, "tests/integration/failure.fy", `module tests::integration::failure;
@test
fn void invalid_unselected_directory() { missing(); }
`)
	code, stdout, stderr = runForTest([]string{"test", "--dir", "tests/unit", "--locked"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || stderr == "" {
		t.Fatalf("selected directory with invalid unselected overlay = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableWorkspaceTestValidatesHarnessOwnedDeclarations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
	}{
		{name: "attribute arguments", source: "@test(\"extra\")\nfn void invalid() {}"},
		{name: "duplicate attribute", source: "@test\n@test\nfn void invalid() {}"},
		{name: "public", source: "@test\npub fn void invalid() {}"},
		{name: "unsafe", source: "@test\nunsafe fn void invalid() {}"},
		{name: "parameter", source: "@test\nfn void invalid(i32 value) {}"},
		{name: "result", source: "@test\nfn i32 invalid() { return 0; }"},
		{name: "non-function", source: "@test\nstruct invalid { i32 value; }"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeStableTestWorkspace(t, root)
			writeWorkspaceFile(t, root, "tests/invalid.fy", "module tests::invalid;\n"+test.source+"\n")
			code, stdout, stderr := runForTest([]string{"test"}, root)
			if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "tests/invalid.fy") || !strings.Contains(stderr, "E9001") || strings.Contains(stderr, "experimental") {
				t.Fatalf("test = (%d, %q, %q)", code, stdout, stderr)
			}
		})
	}
}

func TestStableWorkspaceTestKeepsProductionAndTestSourceFailuresSeparate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/broken.fy", "module tests::broken;\n@test\nfn void broken( {\n")

	code, stdout, stderr := runForTest([]string{"check"}, root)
	if code != ExitSuccess || stdout != "check ok: 1 instance(s), 1 source file(s), 1 MIR function(s), lock updated\n" || stderr != "" {
		t.Fatalf("stable check read tests/ = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--locked"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "tests/broken.fy") {
		t.Fatalf("stable test syntax failure = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableWorkspaceTestHandlesZeroTestsInvocationAndOutputFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)

	code, stdout, stderr := runForTest([]string{"test"}, root)
	if code != ExitSuccess || stdout != "test result: ok. 0 passed; 0 failed; lock updated\n" || stderr != "" {
		t.Fatalf("zero tests = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--list", "--locked"}, root)
	if code != ExitSuccess || stdout != "test list: 0 discovered; lock verified\n" || stderr != "" {
		t.Fatalf("zero test list = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--list", "--json", "--locked"}, root)
	if code != ExitSuccess || stdout != `{"schemaVersion":"fairyc.test-list/v1","tests":[],"count":0,"lock":"verified"}`+"\n" || stderr != "" {
		t.Fatalf("zero JSON test list = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--list", "--json", "--locations", "--locked"}, root)
	if code != ExitSuccess || stdout != `{"schemaVersion":"fairyc.test-list/v3","tests":[],"locations":[],"count":0,"lock":"verified"}`+"\n" || stderr != "" {
		t.Fatalf("zero located JSON test list = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--json", "--locked"}, root)
	if code != ExitSuccess || stdout != `{"schemaVersion":"fairyc.test-report/v1","engine":"interpreter","target":null,"profile":null,"tests":[],"count":0,"passed":0,"failed":0,"status":"ok","lock":"verified"}`+"\n" || stderr != "" {
		t.Fatalf("zero JSON test report = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--json", "--locations", "--locked"}, root)
	if code != ExitSuccess || stdout != `{"schemaVersion":"fairyc.test-report/v3","engine":"interpreter","target":null,"profile":null,"tests":[],"count":0,"passed":0,"failed":0,"status":"ok","lock":"verified"}`+"\n" || stderr != "" {
		t.Fatalf("zero located JSON test report = (%d, %q, %q)", code, stdout, stderr)
	}
	var outputError bytes.Buffer
	code = Run([]string{"test", "--list", "--json", "--locked"}, root, failingWriter{}, &outputError)
	if code != ExitInternalFailure || !strings.Contains(outputError.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, outputError.String())
	}
	outputError.Reset()
	code = Run([]string{"test", "--json", "--locked"}, root, failingWriter{}, &outputError)
	if code != ExitInternalFailure || !strings.Contains(outputError.String(), "write output failed") {
		t.Fatalf("JSON execution output failure = (%d, %q)", code, outputError.String())
	}
	for _, arguments := range [][]string{
		{"test", "--release"},
		{"test", "--locked", "--locked"},
		{"test", "--native", "--native"},
		{"test", "--list", "--list"},
		{"test", "--list", "--json", "--json"},
		{"test", "--locations", "--locations"},
		{"test", "--list", "--native"},
		{"test", "--list", "--release"},
		{"test", "--list", "--exact", "tests::missing::test"},
		{"test", "--exact"},
		{"test", "--exact", "tests::missing::test", "--exact", "tests::missing::test"},
	} {
		code, stdout, stderr = runForTest(arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, `command "test"`) {
			t.Fatalf("invalid invocation %v = (%d, %q, %q)", arguments, code, stdout, stderr)
		}
	}
}

func TestStableWorkspaceTestListsCanonicalIdentitiesWithoutExecution(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/list.fy", `module tests::list;
@test
fn void zeta_traps_if_run() { assert(false); }
@test
fn void alpha_passes() { assert(true); }
`)

	code, stdout, stderr := runForTest([]string{"test", "--list"}, root)
	want := "test tests::list::alpha_passes\n" +
		"test tests::list::zeta_traps_if_run\n" +
		"test list: 2 discovered; lock updated\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("test list = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	code, stdout, stderr = runForTest([]string{"test", "--json", "--list", "--locked"}, root)
	want = `{"schemaVersion":"fairyc.test-list/v1","tests":["tests::list::alpha_passes","tests::list::zeta_traps_if_run"],"count":2,"lock":"verified"}` + "\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("JSON test list = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	if matches, err := filepath.Glob(filepath.Join(root, "build", "*")); err != nil || len(matches) != 0 {
		t.Fatalf("test list build artifacts = %v, %v", matches, err)
	}

	writeWorkspaceFile(t, root, "tests/broken.fy", `module tests::broken;
@test
fn void broken() { missing(); }
`)
	code, stdout, stderr = runForTest([]string{"test", "--list", "--locked"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "tests/broken.fy") {
		t.Fatalf("test list semantic failure = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableWorkspaceTestNativeOptionsRequireExplicitNativeMode(t *testing.T) {
	t.Parallel()
	options, err := parseStableTestOptions([]string{"--release", "--exact", "tests::answer::is_42", "--json", "--locked", "--native"})
	if err != nil || !options.native || !options.release || !options.json || !options.locked ||
		len(options.exact) != 1 || options.exact[0] != "tests::answer::is_42" {
		t.Fatalf("native options = %+v, %v", options, err)
	}
	if _, err := parseStableTestOptions([]string{"--release"}); err == nil || !strings.Contains(err.Error(), "requires --native") {
		t.Fatalf("release-only options error = %v", err)
	}
}

func TestStableWorkspaceTestWritesNativeJSONExecutionIdentity(t *testing.T) {
	t.Parallel()
	category := "assertion-failed"
	outcomes := []workspaceTestOutcome{
		{display: "tests::native::fails", trapCategory: category},
		{display: "tests::native::passes"},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := writeStableTestReportJSON(
		outcomes,
		"verified",
		stableTestOptions{native: true, release: true, json: true},
		&stdout,
		&stderr,
	)
	want := `{"schemaVersion":"fairyc.test-report/v1","engine":"native","target":"x86_64-windows-msvc","profile":"release","tests":[{"identity":"tests::native::fails","outcome":"failed","trapCategory":"assertion-failed"},{"identity":"tests::native::passes","outcome":"passed","trapCategory":null}],"count":2,"passed":1,"failed":1,"status":"failed","lock":"verified"}` + "\n"
	if code != ExitProgramTrap || stdout.String() != want || stderr.String() != "" {
		t.Fatalf("native JSON report = (%d, %q, %q), want stdout %q", code, stdout.String(), stderr.String(), want)
	}
}

func TestStableWorkspaceTestSelectsRepeatedExactIdentitiesInCanonicalOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/selection.fy", `module tests::selection;
@test
fn void alpha() { assert(true); }
@test
fn void middle() { assert(false); }
@test
fn void zeta() { assert(true); }
`)

	code, stdout, stderr := runForTest([]string{
		"test",
		"--exact", "tests::selection::zeta",
		"--exact", "tests::selection::alpha",
	}, root)
	want := "test tests::selection::alpha ... ok\n" +
		"test tests::selection::zeta ... ok\n" +
		"test result: ok. 2 passed; 0 failed; lock updated\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("exact test selection = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}

	code, stdout, stderr = runForTest([]string{"test", "--exact", "tests::selection::missing", "--locked"}, root)
	if code != ExitInvalidInvocation || stdout != "" ||
		!strings.Contains(stderr, `exact test selection was not found: "tests::selection::missing"`) {
		t.Fatalf("missing exact test selection = (%d, %q, %q)", code, stdout, stderr)
	}

	code, stdout, stderr = runForTest([]string{"test", "--exact", "missing\nidentity", "--locked"}, root)
	if code != ExitInvalidInvocation || stdout != "" || strings.Contains(stderr, "missing\nidentity") ||
		!strings.Contains(stderr, `"missing\nidentity"`) {
		t.Fatalf("escaped exact test selection = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestNativeWorkspaceTestProgramAddsVerifiedI32Harness(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	native, entry, err := nativeWorkspaceTestProgram(program, "tests::native::passes")
	if err != nil {
		t.Fatal(err)
	}
	if entry != nativeWorkspaceTestEntry || len(native.Functions) != len(program.Functions)+1 {
		t.Fatalf("native test program entry=%q functions=%d", entry, len(native.Functions))
	}
	if failures := mir.Verify(native); len(failures) != 0 {
		t.Fatalf("native test MIR failures = %v", failures)
	}
	unit, err := cbackend.Emit(native, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if unit.Entry != nativeWorkspaceTestEntry || !strings.Contains(unit.Source, "int main(void)") ||
		!strings.Contains(unit.Source, "fy_fn_0();") {
		t.Fatalf("native test C unit = entry %q, functions %v", unit.Entry, unit.Functions)
	}
}

func TestNativeWorkspaceTestsContinueAfterTrapsAndPreserveOrder(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{
		{display: "tests::native::fails", entry: "tests::native::fails"},
		{display: "tests::native::passes", entry: "tests::native::passes"},
	}
	compiled := 0
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		if entry != nativeWorkspaceTestEntry || profile != cdriver.OptimizedProfile || len(mir.Verify(program)) != 0 {
			t.Fatalf("compile native test entry=%q profile=%+v", entry, profile)
		}
		compiled++
		return cdriver.WindowsHostArtifact{Bytes: []byte{byte(compiled)}}, ExitSuccess
	}
	execute := func(artifact cdriver.WindowsHostArtifact) (cdriver.WindowsHostOutcome, error) {
		if artifact.Bytes[0] == 1 {
			code, found := cbackend.TrapCode("assertion-failed")
			if !found {
				t.Fatal("assertion-failed trap code is missing")
			}
			return cdriver.WindowsHostOutcome{
				ExitCode: int32(productTrapExitBase + code),
				Stderr:   cruntime.RunTrapReceiptPrefix + strconv.Itoa(int(code)),
			}, nil
		}
		return cdriver.WindowsHostOutcome{}, nil
	}
	var stderr bytes.Buffer
	outcomes, code := runNativeWorkspaceTestsWith(
		program, tests, cdriver.OptimizedProfile, &stderr, compile, execute,
	)
	if code != ExitSuccess || stderr.String() != "" || compiled != 2 || len(outcomes) != 2 ||
		outcomes[0].display != tests[0].display || outcomes[0].trapCategory != "assertion-failed" ||
		outcomes[1].display != tests[1].display || outcomes[1].trapCategory != "" {
		t.Fatalf("native outcomes=%+v code=%d stderr=%q compiled=%d", outcomes, code, stderr.String(), compiled)
	}
}

func TestNativeWorkspaceTestsRejectProgramOutputAndUnreceiptedExit(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{{display: "tests::native::passes", entry: "tests::native::passes"}}
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		return cdriver.WindowsHostArtifact{Bytes: []byte{1}}, ExitSuccess
	}
	for _, test := range []struct {
		name     string
		outcome  cdriver.WindowsHostOutcome
		wantCode int
		contains string
	}{
		{name: "stdout", outcome: cdriver.WindowsHostOutcome{Stdout: "unexpected"}, wantCode: ExitInvalidInvocation, contains: "produced program output"},
		{name: "unreceipted exit", outcome: cdriver.WindowsHostOutcome{ExitCode: 42}, wantCode: ExitBuildFailure, contains: "without a valid trap receipt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			outcomes, code := runNativeWorkspaceTestsWith(
				program,
				tests,
				cdriver.LowProfile,
				&stderr,
				compile,
				func(cdriver.WindowsHostArtifact) (cdriver.WindowsHostOutcome, error) { return test.outcome, nil },
			)
			if outcomes != nil || code != test.wantCode || !strings.Contains(stderr.String(), test.contains) {
				t.Fatalf("native failure outcomes=%+v code=%d stderr=%q", outcomes, code, stderr.String())
			}
		})
	}
}

func nativeTestFixtureProgram() mir.Program {
	makeTest := func(symbol hir.SymbolID, name string) mir.Function {
		return mir.Function{
			Symbol: symbol, Name: name, ResultType: hir.TypeVoid, Entry: 0,
			Blocks: []mir.Block{{
				ID: 0,
				Terminator: mir.Terminator{
					Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal,
				},
			}},
		}
	}
	return mir.Program{Functions: []mir.Function{
		makeTest(1, "tests::native::passes"),
		makeTest(2, "tests::native::fails"),
	}}
}

func TestStableWorkspaceTestLockedModeRequiresCanonicalProductionLock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/pass.fy", "module tests::pass;\n@test\nfn void passes() { assert(true); }\n")

	code, stdout, stderr := runForTest([]string{"test", "--locked"}, root)
	if code != ExitPackageFailure || stdout != "" || !strings.Contains(stderr, "requires a valid fairy.lock") {
		t.Fatalf("locked test = (%d, %q, %q)", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, lockfile.FileName)); !os.IsNotExist(err) {
		t.Fatalf("locked test created fairy.lock: %v", err)
	}
}

func writeStableTestWorkspace(t *testing.T, root string) {
	t.Helper()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.stable-test.app"
name = "stable-test-app"
version = "1.0.0"
language = "fairyc-language/10"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/library.fy", "module library;\npub fn i32 answer() { return 42; }\n")
}

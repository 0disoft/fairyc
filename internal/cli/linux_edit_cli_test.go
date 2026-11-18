package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

// This executes the shared edit command with real Windows I/O and Linux help.
// Full entry emission proves Linux host selection; the native replay below
// exercises actual Linux I/O when the existing Linux CLI image is available.
func TestLinuxAtlasEditCommand(t *testing.T) {
	main := `module main;
import project_edit_cli as edit;import linux_cli_data as linux;
import std::bytes as bytes;import std::process as process;
@effects(filesystem,console,environment) fn i32 main(){
 var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] prefix=ref raw;
 if(bytes::equal(prefix[..n],b"default-help")){return edit::arguments();}
 return edit::arguments_with_help(linux::help());
}`
	candidateAtlasProbeSize(t, "project_edit_cli", main, 2<<20, func(t *testing.T, image string) {
		verifyLinuxAtlasEdit(t, image)
		verifyLinuxAtlasCommandHelp(t, image, "default-help", "atlas-edit", candidateUsage(), [][]string{nil})
		verifyLinuxEditApplication(t, image)
	}, "linux_cli_data")
}

func selectLinuxEdit(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{"project_edit_cli", "project_edit_preview", "project_edit_snapshot"} {
		path := filepath.Join(root, "src", name+".fy")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		changes := [][2]string{{"import windows_rooted_file as host;", "import linux_rooted_file as host;"}}
		if name != "project_edit_preview" {
			changes = append(changes, [2]string{"import project_remote_fs as files;", "import linux_edit_fs as files;"})
		}
		if name == "project_edit_cli" {
			changes = append(changes, [2]string{"import project_edit_apply as applying;", "import linux_edit_apply as applying;"})
		}
		source := string(data)
		for _, change := range changes {
			if strings.Count(source, change[0]) != 1 {
				t.Fatalf("Linux edit boundary changed: %s: %s", name, change[0])
			}
			source = strings.Replace(source, change[0], change[1], 1)
		}
		writeTestSource(t, root, "src/"+name+".fy", source)
	}
}

func verifyLinuxAtlasEdit(t *testing.T, image string) {
	t.Helper()
	help := linuxFetchHelp(t)
	verifyLinuxAtlasCommandHelp(t, image, "experimental", "atlas-edit", help, [][]string{nil})
	code, _, diagnostic := fetchCLIProcess(t, image, t.TempDir(), []string{"experimental", "atlas-edit", "--help"}, "stdout")
	if code != 4 || diagnostic != "" {
		t.Fatalf("edit help output failure: %d %q", code, diagnostic)
	}
	verifyCandidateEditCommandWithHelp(t, image, help)
}

func verifyLinuxEditApplication(t *testing.T, image string) {
	t.Helper()
	reference, referenceArgs := editFixture(t)
	code, output, diagnostic := runForTest(append(referenceArgs, "--apply"), reference)
	var want sourceedit.Report
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(output), &want) != nil || !want.Applied {
		t.Fatal(code, output, diagnostic)
	}
	updated, err := os.ReadFile(filepath.Join(reference, "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	root, args := editFixture(t)
	before := remoteTree(t, root)
	code, output, diagnostic = fetchCLIProcess(t, image, root, append(args, "--apply"), "")
	var got sourceedit.Report
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(output), &got) != nil || got != want {
		t.Fatalf("edit application: %d %s %s; want %+v", code, output, diagnostic, want)
	}
	before["src/main.fy"] = sourceedit.Digest(updated)
	if !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatal("application changed another input or leaked staging/lock")
	}
	code, output, diagnostic = fetchCLIProcess(t, image, root, append(args, "--apply"), "")
	if code != 7 || output != "" || diagnostic == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("applied-source old-lock refusal: %d %s %s", code, output, diagnostic)
	}
}

func checkLinuxAtlasEdit(t *testing.T, artifact cdriver.NativeArtifact) {
	t.Helper()
	t.Run("atlas-edit", func(t *testing.T) {
		image := filepath.Join(t.TempDir(), "fairyc")
		if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		verifyLinuxAtlasEdit(t, image)
		verifyCandidateEditCLIApply(t, image)
		verifyLinuxEditMetadata(t, image)
	})
}

// Actual Linux inode/permission behavior belongs to this native replay, never
// to the Windows command result or the separate controlled namespace model.
func verifyLinuxEditMetadata(t *testing.T, image string) {
	t.Helper()
	root, args := editFixture(t)
	source := filepath.Join(root, "src", "main.fy")
	if err := os.Chmod(source, 0640); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "old-source")
	if err := os.Link(source, alias); err != nil {
		t.Fatal(err)
	}
	body := filepath.Join(root, "edits", "body.txt")
	if err := os.Link(body, filepath.Join(root, "old-body")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(body, 0400); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	code, output, diagnostic := fetchCLIProcess(t, image, root, append(args, "--apply"), "")
	var report sourceedit.Report
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(output), &report) != nil || !report.Applied {
		t.Fatalf("hard-linked edit: %d %s %s", code, output, diagnostic)
	}
	info, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	oldInfo, err := os.Stat(alias)
	if err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(alias)
	if err != nil || string(old) != string(original) || info.Mode().Perm() != 0640 || os.SameFile(info, oldInfo) {
		t.Fatalf("original alias or permissions changed: %v %v %v", info, oldInfo, err)
	}
	root, args = editFixture(t)
	if err := os.Chmod(filepath.Join(root, "src", "main.fy"), 0400); err != nil {
		t.Fatal(err)
	}
	before := remoteTree(t, root)
	code, output, diagnostic = fetchCLIProcess(t, image, root, append(args, "--apply"), "")
	if code != 7 || output != "" || diagnostic == "" || !reflect.DeepEqual(before, remoteTree(t, root)) {
		t.Fatalf("read-only edit: %d %s %s", code, output, diagnostic)
	}
}

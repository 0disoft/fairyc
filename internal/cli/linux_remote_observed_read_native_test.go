package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

// Authored real-filesystem replay; Windows verifies only its emitted inputs.
func TestLinuxRemoteObservedReadNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires actual Linux amd64; execution remains in the combined native batch")
	}
	root := linuxObservedReadWorkspace(t)
	build, support := linuxRemoteBuild(t, root, target.X8664LinuxGNU, false)
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				artifact, err := cdriver.BuildLinuxTargetParts(newLinuxRemoteBuildStage(t), compiler, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support+linuxObservedReadReplacement, false, time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				image := filepath.Join(t.TempDir(), "observed-read")
				if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				workspace := t.TempDir()
				if err := os.WriteFile(filepath.Join(workspace, "input"), []byte{'a', 0, 255}, 0600); err != nil {
					t.Fatal(err)
				}
				command := exec.CommandContext(t.Context(), image)
				command.Dir, command.Env = workspace, append(os.Environ(), "PATH=")
				if output, err := command.CombinedOutput(); err != nil || len(output) != 0 {
					t.Fatalf("observed-generation guard: %v %q", err, output)
				}
				data, err := os.ReadFile(filepath.Join(workspace, "input"))
				if err != nil || string(data) != "a\x00\xff" {
					t.Fatalf("replacement bytes: %v %q", err, data)
				}
				if _, err := os.Stat(filepath.Join(workspace, "replacement")); !os.IsNotExist(err) {
					t.Fatalf("replacement stage remains: %v", err)
				}
			})
		}
	}
}

func TestLinuxRemoteObservedReadNativeInputs(t *testing.T) {
	_, support := linuxRemoteBuild(t, linuxObservedReadWorkspace(t), target.X8664LinuxGNU, false)
	for _, required := range []string{"O_NOFOLLOW", "st_nlink", "fairyc_private_linux_package_file_info_v1("} {
		if !strings.Contains(support, required) {
			t.Fatalf("native support missing %s", required)
		}
	}
	for _, forbidden := range []string{"remote_nodes[", "remote_fs_probe_", "online_probe_", "offline_fixture("} {
		if strings.Contains(linuxObservedReadEntry+support+linuxObservedReadReplacement, forbidden) {
			t.Fatalf("model in native replay: %s", forbidden)
		}
	}
}

func linuxObservedReadWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_remote_fs")
	writeTestSource(t, root, "src/main.fy", linuxObservedReadEntry)
	return root
}

const linuxObservedReadEntry = `module main;
import linux_rooted_file as rooted;import linux_directory as directory;
import linux_remote_fs as fs;import std::bytes as bytes;
@effects(filesystem) @link_name("offline_probe_replace") unsafe extern C fn void replace();
@effects(filesystem) fn i32 main(){
    let root=switch(rooted::open(b".")){case .ok(p):{p}case .err(_):{return 1;}};
    var usize work=0;let entries=switch(directory::scan_packages(ref root,ref mut work)){case .ok(p):{p}case .err(_):{return 2;}};
    if(directory::count(ref entries)!=1 || !bytes::equal(directory::name(ref entries,0),b"input")){return 3;}
    let expected=directory::metadata(ref entries,0);
    let initial=switch(fs::read_observed(ref root,b"input",expected,3)){case .ok(p):{p}case .err(_):{return 4;}};
    if(!bytes::equal(bytes::view(ref initial),b"a\0\xff")){return 5;}
    unsafe{replace();}
    switch(fs::read_observed(ref root,b"input",expected,3)){case .ok(_):{return 6;}case .err(p):{if(p!=4){return 7;}}};
    let current=switch(fs::read(ref root,b"input",3)){case .ok(p):{p}case .err(_):{return 8;}};
    return if(bytes::equal(bytes::view(ref current),b"a\0\xff")){0}else{9};
}
`

// Test-only namespace mutation, not a replacement filesystem implementation.
// The original inode remains linked until rename, so its replacement is distinct.
const linuxObservedReadReplacement = `
#include <assert.h>
#include <stdio.h>
void offline_probe_replace(void){
    const unsigned char bytes[]={97,0,255};
    int fd=open("replacement",O_WRONLY|O_CREAT|O_EXCL|O_CLOEXEC,0600);assert(fd>=0);
    assert(write(fd,bytes,sizeof bytes)==3);assert(close(fd)==0);
    assert(rename("replacement","input")==0);
}
`

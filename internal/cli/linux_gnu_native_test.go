package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxGNUCompanionNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Linux amd64; portable driver/entry checks run separately")
	}
	unit := linuxGNUCompanionUnit(t)
	support, err := nativesupport.Source(target.X8664LinuxGNU, unit.Unit.ForeignSymbols, nativesupport.Options{SDKStorage: true, CandidateHost: true})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := candidatehost.LinuxSource("fairyc_private_linux_capture_open_v1(")
	if err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.ResolveNativeToolchain(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				companion, err := tc.Build(nativeBuildOptions(unit, profile, support, false))
				if err != nil {
					t.Fatal(err)
				}
				install := t.TempDir()
				image := filepath.Join(install, "fairyc-candidate-linux-build")
				pack := filepath.Join(install, "fairyc-linux-runtime-assets.pack")
				if err := os.WriteFile(image, companion.Artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				command := exec.Command("python3", "-B", "../../tools/package-candidate-linux-runtime.py", "--output", pack)
				if data, err := command.CombinedOutput(); err != nil {
					t.Fatalf("runtime pack: %s %v", data, err)
				}
				original, err := os.ReadFile(pack)
				if err != nil {
					t.Fatal(err)
				}
				wrapper, err := tc.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: capture + linuxGNUOwnerProbe, RuntimeSource: "typedef int fy_gnu_owner_runtime;\n"})
				if err != nil {
					t.Fatal(err)
				}
				for _, mode := range []string{"valid", "arguments", "compile-error", "link-error", "tampered-pack", "run-image"} {
					t.Run(mode, func(t *testing.T) {
						root := t.TempDir()
						stage := filepath.Join(root, "stage")
						if err := os.Mkdir(stage, 0700); err != nil {
							t.Fatal(err)
						}
						packet := linuxGNUCompanionPacket(t, mode)
						input := filepath.Join(stage, "packet.bin")
						if err := os.WriteFile(input, packet, 0600); err != nil {
							t.Fatal(err)
						}
						outputRoot := root
						relative := "program"
						if mode == "run-image" {
							outputRoot = stage
						}
						output := filepath.Join(outputRoot, relative)
						old := []byte("old image")
						if mode != "run-image" {
							if err := os.WriteFile(output, old, 0700); err != nil {
								t.Fatal(err)
							}
						}
						asset := append([]byte(nil), original...)
						if mode == "tampered-pack" {
							asset[len(asset)-1] ^= 1
						}
						if err := os.WriteFile(pack, asset, 0600); err != nil {
							t.Fatal(err)
						}
						args := []string{image, "--packet-input", input, "--stage-root", stage, "--output-root", outputRoot, "--output", relative, "--cc", compiler}
						if profile.Name == cdriver.OptimizedProfile.Name {
							args = append(args, "--release")
						}
						if mode == "run-image" {
							args = append(args, "--run-trap-control")
						}
						outcome, err := tc.Execute(wrapper, cdriver.NativeExecutionOptions{WorkingDirectory: root, Arguments: args, Timeout: time.Minute})
						if err != nil {
							t.Fatal(err)
						}
						want := int32(0)
						if strings.Contains(mode, "error") || mode == "tampered-pack" {
							want = 1
						}
						if outcome.ExitCode != want || outcome.Stdout != "" {
							t.Fatalf("companion: %+v", outcome)
						}
						data, err := os.ReadFile(output)
						if err != nil {
							t.Fatal(err)
						}
						if want != 0 {
							diagnostic := map[string]string{"compile-error": "compile GNU C", "link-error": "link GNU executable", "tampered-pack": "runtime asset pack does not match compiler pin"}[mode]
							if !bytes.Equal(data, old) || !strings.Contains(outcome.Stderr, diagnostic) {
								t.Fatalf("failed build changed output: %+v", outcome)
							}
							return
						}
						if outcome.Stderr != "" {
							t.Fatal(outcome.Stderr)
						}
						program := companion
						program.Artifact.Bytes = data
						programArgs := []string{}
						if mode == "arguments" {
							programArgs = []string{"", "two words"}
						}
						run, err := tc.Execute(program, cdriver.NativeExecutionOptions{WorkingDirectory: root, Arguments: programArgs, Timeout: 5 * time.Second})
						if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
							t.Fatalf("published image: %+v %v", run, err)
						}
					})
				}
			})
		}
	}
}

func linuxGNUCompanionPacket(t *testing.T, mode string) []byte {
	t.Helper()
	header := "#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\n#include <stdint.h>\n#include <stddef.h>\n"
	body := "return 42;"
	if mode == "arguments" {
		header += "extern size_t fy_std_process_arg_count_v0(void);\n"
		body = "return fy_std_process_arg_count_v0()==3 ? 42 : 9;"
	}
	if mode == "compile-error" {
		body = "not valid C;"
	}
	if mode == "link-error" {
		header += "extern int32_t absent_symbol(void);\n"
		body = "return absent_symbol();"
	}
	header += "#endif\n"
	literals := "/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\ntypedef int fy_literals_present;\n"
	unit := "/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\nextern int32_t fy_fn_1(void){" + body + "}\nint32_t fy_candidate_entry(void){return fy_fn_1();}\n#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n"
	var packet bytes.Buffer
	packet.WriteString("FYCPARTS1\n")
	for _, part := range []string{header, literals} {
		fmt.Fprintf(&packet, "%d\n%s", len(part), part)
	}
	packet.WriteString("1\n")
	fmt.Fprintf(&packet, "%d\n%s", len(unit), unit)
	if _, err := candidatebuild.Decode(packet.Bytes()); err != nil {
		t.Fatal(err)
	}
	return packet.Bytes()
}

// Reuse the production outer capture so compiler descendants remain inside the
// owning group even on failure. This harness does not replace the companion.
const linuxGNUOwnerProbe = `
#include <stdio.h>
int main(int argc,char **argv){
    if(argc<2){return 120;}size_t size=0;for(int i=1;i<argc;++i){size+=strlen(argv[i])+1;}if(size>1048576){return 120;}
    uint8_t *data=malloc(size);if(data==NULL){return 120;}size_t at=0;for(int i=1;i<argc;++i){size_t n=strlen(argv[i])+1;memcpy(data+at,argv[i],n);at+=n;}
    char cwd[4096];if(getcwd(cwd,sizeof cwd)==NULL){free(data);return 120;}void *capture=NULL;
    uint32_t code=fairyc_private_linux_capture_open_v1(data,size,(size_t)argc-1,(uint8_t *)cwd,strlen(cwd)+1,&capture);free(data);if(code!=0){return 121;}
    uint8_t out[65536],err[65536];size_t n=0,m=0;uint32_t kind=0,value=0,pid=0;
    code=fairyc_private_linux_capture_collect_v1(capture,45000,out,sizeof out,&n,err,sizeof err,&m,&kind,&value,&pid);
    uint32_t closed=fairyc_private_linux_capture_drop_v1(capture);if(code!=0 || closed!=0 || kind!=1 || value>255){return 122;}
    if(fwrite(out,1,n,stdout)!=n || fwrite(err,1,m,stderr)!=m){return 123;}return (int)value;
}
`

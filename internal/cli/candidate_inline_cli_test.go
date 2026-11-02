package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
)

const inlineCLIProvider = `module answer;
@inline pub fn i32 mix(i32 a,i32 b,i32 c,i32 d){return a*1000+b*100+c*10+d;}
@inline pub fn i32 add(i32 x){return x+2;}
@inline pub fn i32 minimum(){return -2147483648;}
@inline pub fn i32 divide(i32 a,i32 b){return a/b;}
@inline pub fn i32 unused(){return 64;}
fn i32 spare(){return 8;}
`

const inlineCLIChecks = `var i32 n=0;
assert(a::mix(pick(ref mut n),pick(ref mut n),pick(ref mut n),pick(ref mut n))==1234);
assert(n==4);assert(a::minimum()==-2147483648);assert(a::divide(10,2)==5);`

const inlineCLIPick = `fn i32 pick(ref mut i32 n){*n+=1;return *n;}`

func inlineCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.inline-cli"
name = "inline-cli"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	writeTestSource(t, root, "libs/answer/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.inline-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeTestSource(t, root, "libs/answer/src/answer.fy", inlineCLIProvider)
	writeTestSource(t, root, "src/main.fy", "module main;import answer as a;"+inlineCLIPick+"fn i32 main(){"+inlineCLIChecks+"return a::add(40);}")
	writeTestSource(t, root, "tests/inline.fy", "module tests::inline;import answer as a;"+inlineCLIPick+"@test fn void inline_contract(){"+inlineCLIChecks+"assert(a::add(40)==42);}")
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func checkInlineCLIPacket(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := candidatebuild.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(packet.Header, "static inline int32_t fy_fn_"); count != 8 {
		t.Fatalf("expected four reachable inline prototypes and definitions in shared header, got %d", count)
	}
	for _, unit := range packet.Units {
		if strings.Contains(unit, "static inline ") {
			t.Fatal("inline definition escaped shared header")
		}
	}
	t.Logf("current companion packet: four inline definitions in shared header, %d body units", len(packet.Units))
}

func TestCandidateInlineCLI(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", os.Getenv("FAIRYC_INLINE_CLI_PROFILE"))
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			root := inlineCLIWorkspace(t)
			packet := filepath.Join(root, "observed.bin")
			t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
			command := copiedClosureCommand{t: t, image: image, root: root}
			command.cache(0, "cache: cold stored", "check")
			command.cache(0, "cache: hit", "check", "--locked")
			for _, flags := range [][]string{nil, {"--release"}} {
				command.cache(0, "cache: hit", append([]string{"build", "--locked"}, flags...)...)
				checkInlineCLIPacket(t, packet)
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				out, err := exec.CommandContext(ctx, filepath.Join(root, "build", "inline-cli.exe")).CombinedOutput()
				cancel()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 42 || len(out) != 0 {
					t.Fatalf("generated inline executable: %v %s", err, out)
				}
				if out := command.cache(42, "cache: hit", append([]string{"run", "--locked"}, flags...)...); out != "" {
					t.Fatal(out)
				}
			}
			out, _ := command.run(0, "test", "--list", "--json", "--locked")
			if !strings.Contains(out, "inline_contract") || !strings.Contains(out, `"count":1`) {
				t.Fatal(out)
			}
			for _, flags := range [][]string{nil, {"--native"}, {"--native", "--release"}} {
				out, _ := command.run(0, append([]string{"test", "--json", "--locked"}, flags...)...)
				if !strings.Contains(out, `"passed":1`) {
					t.Fatal(out)
				}
			}
			t.Logf("%d independent CLI processes; inline shared header, ordered once-only arguments, i32 minimum and division", command.calls)
		})
	}
}

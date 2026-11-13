package cli

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type extendedTransportFixture struct {
	name, provider, before, after string
	payloadChanges                bool
}

var extendedTransportFixtures = []extendedTransportFixture{
	{"signed-result", `module answer;
fn Result<T,i32> wrap<T>(T value,bool fail){if(fail){return .err(-77);}return .ok(value);}
pub fn T pick<T>(T first,T second){let Result<T,i32> successful=wrap<T>(first,false);let Result<T,i32> failed=wrap<T>(first,true);assert(switch(failed){case .ok(_):{false}case .err(code):{code==-77}});return switch(successful){case .ok(value):{value}case .err(_):{assert(false);second}};}`, "wrap<T>(first,false)", "wrap<T>(second,false)", true},
	{"nested-option", `module answer;
pub fn T pick<T>(T first,T second){let Option<Option<T>> original=.some(.some(first));let Option<Option<T>> copied=original;assert(switch(original){case .none:{false}case .some(inner):{switch(inner){case .none:{false}case .some(_):{true}}}});return switch(copied){case .none:{assert(false);second}case .some(inner):{switch(inner){case .none:{assert(false);second}case .some(value):{value}}}};}`, ".some(.some(first))", ".some(.some(second))", true},
	{"inline-mutation", `module answer;
pub fn T pick<T>(T first,T second){return first;}
@inline pub fn i32 bias(i32 value){var i32 output=value;let old=output;output=old+2;return output;}`, "output=old+2", "output=old+3", false},
}

func extendedTransportWorkspace(t *testing.T, f extendedTransportFixture) string {
	root, _ := compositeOptionCLIWorkspace(t)
	writeTestSource(t, root, "libs/answer/src/answer.fy", f.provider)
	if f.name == "inline-mutation" {
		main := "module main;import answer as a;" + genericCLIWrappers("a::") + genericCLIRoot("a::", 42, 43)
		main = strings.Replace(main, "return a::pick<i32>(42,43);", "return a::bias(40);", 1)
		writeTestSource(t, root, "src/main.fy", main)
	}
	return root
}

func TestExtendedTransportCLIFixtures(t *testing.T) {
	for _, f := range extendedTransportFixtures {
		t.Run(f.name, func(t *testing.T) {
			root := extendedTransportWorkspace(t, f)
			if code, out, diagnostic := runForTest([]string{"run"}, root); code != 42 || out != "" || diagnostic != "" {
				t.Fatalf("Go source fixture: %d %s %s", code, out, diagnostic)
			}
			writeTestSource(t, root, "libs/answer/src/answer.fy", strings.Replace(f.provider, f.before, f.after, 1))
			if code, out, diagnostic := runForTest([]string{"run"}, root); code != 43 || out != "" || diagnostic != "" {
				t.Fatalf("Go changed fixture: %d %s %s", code, out, diagnostic)
			}
		})
	}
}

func TestCandidateExtendedTransportCLIAndCache(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			for _, f := range extendedTransportFixtures {
				t.Run(f.name, func(t *testing.T) {
					root := extendedTransportWorkspace(t, f)
					packet := filepath.Join(root, "observed.bin")
					t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
					command := copiedClosureCommand{t: t, image: image, root: root}
					command.cache(0, "cache: cold stored", "check")
					command.cache(0, "cache: hit", "check", "--locked")
					for _, flags := range [][]string{nil, {"--release"}} {
						command.cache(0, "cache: hit", append([]string{"build", "--locked"}, flags...)...)
						observeGenericCLIPacket(t, packet, false, 42)
						ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
						output, err := exec.CommandContext(ctx, filepath.Join(root, "build", "generic-cli.exe")).CombinedOutput()
						cancel()
						if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 42 || len(output) != 0 {
							t.Fatalf("standalone: %v %s", err, output)
						}
					}
					if out, diagnostic := command.run(0, "test", "--native", "--release", "--locked", "--json"); diagnostic != "" || !strings.Contains(out, `"passed":1`) {
						t.Fatalf("native tests: %s %s", out, diagnostic)
					}
					writeTestSource(t, root, "libs/answer/src/answer.fy", strings.Replace(f.provider, f.before, f.after, 1))
					command.cache(7, "cache: skipped", "check", "--locked")
					if _, diagnostic := command.run(43, "run", "--cache-root", "cache", "--cache-status"); strings.HasPrefix(diagnostic, "cache: hit\n") {
						t.Fatal("provider edit reused stale program")
					}
					observeGenericCLIPacket(t, packet, f.payloadChanges, 43)
					command.cache(43, "cache: hit", "run", "--locked", "--release")
					before := copiedClosureCacheSnapshot(t, root)
					writeTestSource(t, root, "libs/answer/src/answer.fy", f.provider+"fn i32 broken(){return true;}")
					command.cache(3, "cache: cold uncached", "check")
					after := copiedClosureCacheSnapshot(t, root)
					if len(before) != len(after) {
						t.Fatal("invalid source changed generation count")
					}
					for path, digest := range before {
						if after[path] != digest {
							t.Fatal("invalid source changed generation bytes", path)
						}
					}
					t.Logf("%s: %d CLI processes and two standalone products; actual seven-scalar C observer, provider edit and invalid-source preservation", f.name, command.calls)
				})
			}
		})
	}
}

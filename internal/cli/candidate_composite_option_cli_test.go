package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const compositeOptionCLIPick = `fn T pick<T>(T first,T second){
let Option<T> original=.some(first);let Option<T> alternative=.some(second);
let Option<T> chosen=choose<T>(original,alternative);discard original;
let Option<T> empty=.none;assert(switch(empty){case .none:{true}case .some(_):{false}});
return switch(chosen){case .some(actual):{actual}case .none:{assert(false);first}};
}
fn Option<T> choose<T>(Option<T> first,Option<T> second){var Option<T> copy=first;copy=first;return copy;}`

func compositeOptionCLIWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := genericCLIWorkspace(t)
	manifest, err := os.ReadFile(filepath.Join(root, "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", string(manifest)+`
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	writeTestSource(t, root, "libs/answer/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.option-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	provider := "module answer;" + strings.Replace(compositeOptionCLIPick, "fn T pick", "pub fn T pick", 1)
	writeTestSource(t, root, "libs/answer/src/answer.fy", provider)
	writeTestSource(t, root, "src/main.fy", "module main;import answer as a;"+genericCLIWrappers("a::")+genericCLIRoot("a::", 42, 43))
	writeTestSource(t, root, "tests/generics.fy", `module tests::generics;import answer as a;
@test fn void transport(){assert(a::pick<i32>(42,43)==42);assert(a::pick<bool>(true,false));}`)
	return root, provider
}

func TestCandidateCompositeOptionCLIFixture(t *testing.T) {
	root, provider := compositeOptionCLIWorkspace(t)
	for _, text := range []string{provider, strings.Replace(provider, "return copy;", "return second;", 1)} {
		writeTestSource(t, root, "libs/answer/src/answer.fy", text)
		if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
			t.Fatalf("independent Go Option CLI admission: %d %s %s", code, out, diagnostic)
		}
	}
	writeTestSource(t, root, "libs/answer/src/answer.fy", strings.Replace(provider, "return copy;", "return true;", 1))
	if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSourceDiagnostics || diagnostic == "" {
		t.Fatalf("independent Go Option refusal: %d %s", code, diagnostic)
	}
}

func TestCandidateCompositeOptionCLIAndCache(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			root, provider := compositeOptionCLIWorkspace(t)
			packet := filepath.Join(root, "observed.bin")
			t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
			command := copiedClosureCommand{t: t, image: image, root: root}
			command.cache(0, "cache: cold stored", "check")
			command.cache(0, "cache: hit", "check", "--locked")
			for _, flags := range [][]string{nil, {"--release"}} {
				command.cache(0, "cache: hit", append([]string{"build", "--locked"}, flags...)...)
				observeGenericCLIPacket(t, packet, false, 42)
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				out, err := exec.CommandContext(ctx, filepath.Join(root, "build", "generic-cli.exe")).CombinedOutput()
				cancel()
				if status, ok := err.(*exec.ExitError); !ok || status.ExitCode() != 42 || len(out) != 0 {
					t.Fatalf("built Option product: %v %s", err, out)
				}
				command.cache(42, "cache: hit", append([]string{"run", "--locked"}, flags...)...)
			}
			if out, diagnostic := command.run(0, "test", "--native", "--release", "--locked", "--json"); diagnostic != "" || !strings.Contains(out, `"passed":1`) || !strings.Contains(out, `"failed":0`) {
				t.Fatalf("cross-package Option native test: %s %s", out, diagnostic)
			}
			second := strings.Replace(provider, "return copy;", "return second;", 1)
			writeTestSource(t, root, "libs/answer/src/answer.fy", second)
			command.cache(7, "cache: skipped", "check", "--locked")
			if _, diagnostic := command.run(43, "run", "--cache-root", "cache", "--cache-status"); strings.HasPrefix(diagnostic, "cache: hit\n") || !strings.Contains(diagnostic, "stored") && !strings.Contains(diagnostic, "mixed") && !strings.Contains(diagnostic, "units hit") {
				t.Fatalf("Option provider body edit did not rebuild: %s", diagnostic)
			}
			observeGenericCLIPacket(t, packet, true, 43)
			command.cache(43, "cache: hit", "run", "--locked", "--release")
			observeGenericCLIPacket(t, packet, true, 43)
			// Rejected current source must leave previously valid generations intact.
			before := copiedClosureCacheSnapshot(t, root)
			writeTestSource(t, root, "libs/answer/src/answer.fy", strings.Replace(second, "return second;", "return true;", 1))
			command.cache(7, "cache: skipped", "check", "--locked")
			command.cache(3, "cache: cold uncached", "check")
			after := copiedClosureCacheSnapshot(t, root)
			if len(before) != len(after) {
				t.Fatal("rejected Option source changed persistent generation count")
			}
			for name, digest := range before {
				if after[name] != digest {
					t.Fatal("rejected Option source changed persistent generation bytes", name)
				}
			}
			writeTestSource(t, root, "libs/answer/src/answer.fy", second)
			// Ordinary source diagnostics may follow source-set lock publication.
			// Restore the valid source's lock before requiring locked reuse again.
			command.cache(0, "cache: hit", "check")
			rows, err := os.ReadDir(filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				if strings.HasSuffix(row.Name(), ".fyc") {
					if err := os.WriteFile(filepath.Join(root, "cache", row.Name()), []byte("corrupt"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			command.cache(0, "cache: cold stored", "check", "--locked")
			command.cache(43, "cache: hit", "run", "--locked", "--release")
			observeGenericCLIPacket(t, packet, true, 43)
			t.Logf("%d independent CLI processes + 2 products + five seven-scalar observations; full dispatcher, cold/warm cache, Some/None/copy/return, provider edit, invalid-source refusal and selected-generation corruption repair", command.calls)
		})
	}
}

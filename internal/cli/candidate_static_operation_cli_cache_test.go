package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCandidateStaticOperationPersistentCache(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", os.Getenv("FAIRYC_STATIC_CLI_PROFILE"))
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			root := staticOperationCLIWorkspace(t)
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
uid = "test.static-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
			writeTestSource(t, root, "src/main.fy", "module main;import answer as a;"+staticOperationCLIMain("a::"))
			set := func(text string) { t.Helper(); writeTestSource(t, root, "libs/answer/src/answer.fy", text) }
			provider := "module answer;" + staticOperationCLIProvider()
			set(provider)
			packet := filepath.Join(root, "observed.bin")
			t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
			command := copiedClosureCommand{t: t, image: image, root: root}
			command.cache(0, "cache: cold stored", "check")
			before := copiedClosureCacheSnapshot(t, root)
			lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
			if err != nil {
				t.Fatal(err)
			}
			command.cache(0, "cache: hit", "check", "--locked")
			locked, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
			if err != nil || string(locked) != string(lock) || !reflect.DeepEqual(before, copiedClosureCacheSnapshot(t, root)) {
				t.Fatal("warm process changed lock or generation names/bytes", err)
			}
			command.cache(42, "cache: hit", "run", "--locked")
			observeGenericCLIPacket(t, packet, false, 42)
			selected := provider
			for _, scalar := range append([]string{"i32"}, staticOperationCLIExtendedNames()...) {
				old := fmt.Sprintf("select_%s(%s first,%s second){return first;}", scalar, scalar, scalar)
				selected = strings.Replace(selected, old, strings.Replace(old, "return first;", "return second;", 1), 1)
			}
			set(selected)
			command.cache(7, "cache: skipped", "check", "--locked")
			_, status := command.run(43, "run", "--cache-root", "cache", "--cache-status")
			if strings.HasPrefix(status, "cache: hit\n") || strings.Contains(status, "units hit") {
				t.Fatalf("selected provider edit reused every old unit: %s", status)
			}
			observeGenericCLIPacket(t, packet, true, 43)
			command.cache(43, "cache: hit", "run", "--locked")
			unused := strings.Replace(selected, "return 64;", "return 65;", 1)
			set(unused)
			command.cache(7, "cache: skipped", "check", "--locked")
			_, status = command.run(43, "run", "--cache-root", "cache", "--cache-status")
			if strings.HasPrefix(status, "cache: hit\n") || strings.Contains(status, "units hit") {
				t.Fatalf("unused provider edit retained dependent cache units: %s", status)
			}
			command.cache(43, "cache: hit", "run", "--locked")
			private := strings.Replace(unused, "return 8;", "return 9;", 1)
			set(private)
			command.cache(7, "cache: skipped", "check", "--locked")
			command.cache(43, "cache: mixed", "run")
			command.cache(43, "cache: hit", "run", "--locked")
			observeGenericCLIPacket(t, packet, true, 43)
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
			command.cache(43, "cache: hit", "run", "--locked")
			observeGenericCLIPacket(t, packet, true, 43)
			for _, bad := range []struct{ name, source, diagnostic string }{
				{"missing", strings.Replace(private, "@implements(pick,i32)", "", 1), "E4502"},
				{"signature", strings.Replace(private, "select_i32(i32 first,i32 second)", "select_i32(i32 first,bool second)", 1), "E4501"},
				{"duplicate", private + "@implements(pick,i32) fn i32 duplicate(i32 first,i32 second){return first;}", "E4501"},
			} {
				set(bad.source)
				command.cache(7, "cache: skipped", "check", "--locked")
				_, diagnostic := command.run(3, "check", "--cache-root", "cache", "--cache-status")
				if !strings.HasPrefix(diagnostic, "cache: cold uncached\n") || !strings.Contains(diagnostic, bad.diagnostic) || bad.name == "missing" && !strings.Contains(diagnostic, "main.fy") {
					t.Fatalf("%s: bad operation admitted or diagnostic changed: %s", bad.name, diagnostic)
				}
				set(private)
				command.cache(0, "cache: hit", "check")
			}
			t.Logf("%d independent cache CLI processes; selected/unused/private edits, repair and three rejection controls", command.calls)
		})
	}
}

func staticOperationCLIExtendedNames() []string {
	var names []string
	for _, value := range genericCLIValues {
		names = append(names, value.fairy)
	}
	return names
}

package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCandidateGenericTransportPersistentCache(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", os.Getenv("FAIRYC_GENERIC_CLI_PROFILE"))
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
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
uid = "test.generic-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
			provider := "module answer;" + strings.Replace(genericCLIPick, "fn T pick", "pub fn T pick", 1)
			main := "module main;import answer as a;" + genericCLIWrappers("a::") + genericCLIRoot("a::", 42, 43)
			writeTestSource(t, root, "libs/answer/src/answer.fy", provider)
			writeTestSource(t, root, "src/main.fy", main)
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
			second := strings.Replace(provider, "return copy;", "return second;", 1)
			writeTestSource(t, root, "libs/answer/src/answer.fy", second)
			command.cache(7, "cache: skipped", "check", "--locked")
			_, status := command.run(43, "run", "--cache-root", "cache", "--cache-status")
			if strings.HasPrefix(status, "cache: hit\n") || !strings.Contains(status, "stored") && !strings.Contains(status, "mixed") && !strings.Contains(status, "units hit") {
				t.Fatalf("provider edit was not rebuilt: %s", status)
			}
			observeGenericCLIPacket(t, packet, true, 43)
			command.cache(43, "cache: hit", "run", "--locked")
			observeGenericCLIPacket(t, packet, true, 43)
			writeTestSource(t, root, "src/main.fy", strings.Replace(main, "(42,43)", "(44,45)", 1))
			command.cache(7, "cache: skipped", "check", "--locked")
			_, status = command.run(45, "run", "--cache-root", "cache", "--cache-status")
			if strings.HasPrefix(status, "cache: hit\n") {
				t.Fatal("caller edit reused stale complete program", status)
			}
			command.cache(45, "cache: hit", "run", "--locked")
			observeGenericCLIPacket(t, packet, true, 45)
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
			command.cache(45, "cache: hit", "run", "--locked")
			observeGenericCLIPacket(t, packet, true, 45)
			writeTestSource(t, root, "libs/answer/src/answer.fy", strings.Replace(second, "return second;", "return true;", 1))
			command.cache(7, "cache: skipped", "check", "--locked")
			command.cache(3, "cache: cold uncached", "check")
			writeTestSource(t, root, "libs/answer/src/answer.fy", second)
			command.cache(0, "cache: hit", "check")
			t.Logf("%d independent cross-package cache CLI processes; five seven-scalar packet observations", command.calls)
		})
	}
}

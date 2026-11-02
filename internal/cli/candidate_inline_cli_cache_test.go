package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
)

func TestCandidateInlinePersistentCache(t *testing.T) {
	runCandidateInlinePersistentCache(t, inlineCLIWorkspace)
}

func runCandidateInlinePersistentCache(t *testing.T, workspace func(*testing.T) string) {
	t.Helper()
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", os.Getenv("FAIRYC_INLINE_CLI_PROFILE"))
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			root := workspace(t)
			packet := filepath.Join(root, "observed.bin")
			t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
			command := copiedClosureCommand{t: t, image: image, root: root}
			set := func(text string) {
				t.Helper()
				writeTestSource(t, root, "libs/answer/src/answer.fy", text)
			}
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
			checkInlineCLIPacket(t, packet)

			// An inline edit rejects whole-project replay, including an unused
			// export. Rechecked caller IR may still be reused: the changed owner
			// supplies the current inline body during fresh C emission. Canonical
			// spelling and unrelated private bodies retain mixed unit reuse.
			selected := strings.Replace(inlineCLIProvider, "return x+2;", "return x+3;", 1)
			unused := strings.Replace(selected, "return 64;", "return 65;", 1)
			private := strings.Replace(unused, "return 8;", "return 9;", 1)
			formatted := strings.Replace(private, "add(i32 x){return x+3;}", "add(i32 renamed){/*same body*/ return (renamed + 0x3);}", 1)
			removed := strings.Replace(formatted, "@inline pub fn i32 add", "pub fn i32 add", 1)
			for _, edit := range []struct {
				name, source string
				mixed        bool
				definitions  int
			}{
				{"selected", selected, false, 4},
				{"unused", unused, false, 4},
				{"private", private, true, 4},
				{"format", formatted, true, 4},
				{"remove marker", removed, false, 3},
				{"restore marker", formatted, false, 4},
			} {
				t.Log(edit.name)
				set(edit.source)
				command.cache(7, "cache: skipped", "check", "--locked")
				_, status := command.run(43, "run", "--cache-root", "cache", "--cache-status")
				t.Logf("%s: %s", edit.name, strings.TrimSpace(status))
				if edit.mixed {
					if !strings.HasPrefix(status, "cache: mixed\n") {
						t.Fatalf("%s: unchanged inline identity lost mixed reuse: %s", edit.name, status)
					}
				} else if edit.name != "restore marker" && !strings.HasPrefix(status, "cache: cold stored\n") && !strings.HasPrefix(status, "cache: mixed\n") {
					t.Fatalf("%s: changed inline identity retained stale units: %s", edit.name, status)
				}
				if edit.definitions == 4 {
					checkInlineCLIPacket(t, packet)
				} else {
					raw, err := os.ReadFile(packet)
					if err != nil {
						t.Fatal(err)
					}
					current, err := candidatebuild.Decode(raw)
					if err != nil || strings.Count(current.Header, "static inline int32_t fy_fn_") != 2*edit.definitions {
						t.Fatalf("removed inline annotation survived cache restore: %v", err)
					}
				}
				command.cache(43, "cache: hit", "run", "--locked")
			}

			generations := copiedClosureCacheSnapshot(t, root)
			for name := range generations {
				if err := os.WriteFile(filepath.Join(root, "cache", name), []byte("corrupt"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			command.cache(0, "cache: cold stored", "check", "--locked")
			command.cache(43, "cache: hit", "run", "--locked")
			checkInlineCLIPacket(t, packet)

			// Unselected invalid annotations must not be hidden by a valid cache.
			set(strings.Replace(formatted, "fn i32 spare", "@inline fn i32 spare", 1))
			command.cache(7, "cache: skipped", "check", "--locked")
			_, diagnostic := command.run(3, "check", "--cache-root", "cache", "--cache-status")
			if !strings.HasPrefix(diagnostic, "cache: cold uncached\n") || !strings.Contains(diagnostic, "answer.fy") {
				t.Fatalf("private inline annotation admitted or source diagnostic missing: %s", diagnostic)
			}
			set(formatted)
			command.cache(0, "cache: hit", "check")
			t.Logf("%d independent cache CLI processes; inline body/class edits, canonical spelling, marker changes, repair and source rejection", command.calls)
		})
	}
}

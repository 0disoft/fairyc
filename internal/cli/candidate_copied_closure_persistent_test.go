package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func copiedClosureCacheSnapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	result := map[string][32]byte{}
	rows, err := os.ReadDir(filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if strings.HasPrefix(row.Name(), ".stage") {
			t.Fatalf("cache stage leaked: %s", row.Name())
		}
		if !strings.HasPrefix(row.Name(), "fyproject0-") || !strings.HasSuffix(row.Name(), ".fyc") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, "cache", row.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) < 8 || string(raw[:8]) != "FYCACHE0" {
			t.Fatalf("invalid stored generation %s", row.Name())
		}
		result[row.Name()] = sha256.Sum256(raw)
	}
	if len(result) == 0 {
		t.Fatal("no persistent generations")
	}
	return result
}

func copiedClosurePersistent(t *testing.T, image string) {
	for _, family := range []string{"ordinary", "generic"} {
		t.Run(family, func(t *testing.T) {
			root := copiedClosureCLIWorkspace(t)
			writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.closure-cache"
name = "closure-cache"
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
uid = "test.closure-provider"
name = "answer"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
			call, signature := "a::value()", "pub fn i32 value()"
			if family == "generic" {
				call, signature = "a::value<i32>()", "pub fn i32 value<T>()"
			}
			writeTestSource(t, root, "src/main.fy", "module main;import answer as a;fn i32 main(){return "+call+";}")
			provider := "module answer;" + signature + "{var i32 offset=40;let add=fn [i32 offset] i32(i32 n){return offset+n;};offset=100;return add(2);}"
			set := func(source string) { t.Helper(); writeTestSource(t, root, "libs/answer/src/answer.fy", source) }
			set(provider)
			command := copiedClosureCommand{t: t, image: image, root: root}
			command.cache(0, "cache: cold stored", "check")
			before := copiedClosureCacheSnapshot(t, root)
			lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
			if err != nil {
				t.Fatal(err)
			}
			command.cache(0, "cache: hit", "check", "--locked")
			after := copiedClosureCacheSnapshot(t, root)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("warm process rewrote cache generations")
			}
			locked, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
			if err != nil || string(lock) != string(locked) {
				t.Fatal("warm process changed lock", err)
			}
			command.cache(42, "cache: hit", "run", "--locked")
			for _, change := range []struct {
				name, source string
				value        int
			}{
				{"body", strings.Replace(provider, "offset+n", "offset+n+1", 1), 43},
				{"capture", strings.Replace(provider, "offset=40", "offset=42", 1), 44},
			} {
				t.Run(change.name, func(t *testing.T) {
					changed := command
					changed.t = t
					set(change.source)
					changed.cache(7, "cache: skipped", "check", "--locked")
					// Some unaffected package units may be reused, but stale complete hits
					// cannot supply the changed closure's runtime value.
					args := []string{"run", "--cache-root", "cache", "--cache-status"}
					_, status := changed.run(change.value, args...)
					if strings.HasPrefix(status, "cache: hit\n") || !strings.Contains(status, "stored") && !strings.Contains(status, "mixed") && !strings.Contains(status, "units hit") {
						t.Fatalf("changed source was not rebuilt: %s", status)
					}
					changed.cache(change.value, "cache: hit", "run", "--locked")
					command.calls = changed.calls
				})
			}
			rows, err := os.ReadDir(filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				if !strings.HasSuffix(row.Name(), ".fyc") {
					continue
				}
				if err := os.WriteFile(filepath.Join(root, "cache", row.Name()), []byte("corrupt"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			command.cache(0, "cache: cold stored", "check", "--locked")
			command.cache(44, "cache: hit", "run", "--locked")
			valid := strings.Replace(provider, "offset=40", "offset=42", 1)
			set(strings.Replace(valid, "return offset+n;", "return true;", 1))
			command.cache(7, "cache: skipped", "check", "--locked")
			command.cache(3, "cache: cold uncached", "check")
			set(valid)
			command.cache(0, "cache: hit", "check")
			t.Logf("%s: %d independent persistent-cache CLI processes", family, command.calls)
		})
	}
}

func TestCandidateCopiedClosurePersistentCache(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			copiedClosurePersistent(t, copiedClosureCLIHost(t, profile))
		})
	}
}

package cli

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const closureValueCacheProvider = `module answer;
pub struct Holder{fn i32(i32) action;}
pub fn fn i32(i32) make(i32 seed){
 var i32 captured=seed;
 let fn i32(i32) action=fn [i32 captured] i32(i32 x){let value=captured+x;return value;};
 captured=999;return action;
}
pub fn Holder boxed(i32 seed){let action=make(seed);return Holder{.action=action};}
pub fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
`

const closureValueCacheMain = `module main;import answer as a;
fn i32 main(){let holder=a::boxed(40);let copy=holder;let action=copy.action;return a::apply(action,2);}`

// Include every retained generation, even corrupt ones. Failure and warm-hit
// checks must detect additions/removals as well as in-place byte changes.
func closureValueCandidateCacheSnapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	rows, err := os.ReadDir(filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	result := map[string][32]byte{}
	for _, row := range rows {
		if strings.HasPrefix(row.Name(), ".stage") {
			t.Fatalf("unpublished cache stage leaked: %s", row.Name())
		}
		if !strings.HasSuffix(row.Name(), ".fyc") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, "cache", row.Name()))
		if err != nil {
			t.Fatal(err)
		}
		result[row.Name()] = sha256.Sum256(raw)
	}
	return result
}

func closureValueCandidateCacheFamily(t *testing.T, root, envelope, ledger string) {
	t.Helper()
	rows, err := os.ReadDir(filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if !strings.HasSuffix(row.Name(), ".fyc") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, "cache", row.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.HasPrefix(raw, []byte(envelope)) && bytes.Contains(raw, []byte(ledger)) {
			return
		}
	}
	t.Fatalf("no actual %s/%s project generation", envelope, ledger)
}

func closureValueCandidateWarm(t *testing.T, c *copiedClosureCommand, value int) {
	t.Helper()
	before := closureValueCandidateCacheSnapshot(t, c.root)
	lock, err := os.ReadFile(filepath.Join(c.root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	c.cache(0, "cache: hit", "check", "--locked")
	if out := c.cache(value, "cache: hit", "run", "--locked", "--release"); out != "" {
		t.Fatal(out)
	}
	current, err := os.ReadFile(filepath.Join(c.root, "fairy.lock"))
	if err != nil || !bytes.Equal(lock, current) || !reflect.DeepEqual(before, closureValueCandidateCacheSnapshot(t, c.root)) {
		t.Fatal("warm processes rewrote lock or retained generations", err)
	}
}

func TestCandidateClosureValuePersistentCache(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		if !t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			root := writeWorkspaceSessionFixture(t, closureValueCacheMain)
			writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", closureValueCacheProvider)
			if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
				t.Fatal(err)
			}
			c := copiedClosureCommand{t: t, image: image, root: root}
			c.cache(0, "cache: cold stored", "check")
			closureValueCandidateCacheFamily(t, root, "FYCACHE2", "FYCKEYS2")
			closureValueCandidateWarm(t, &c, 42)
			for _, edit := range []struct {
				name, path, source string
				value              int
			}{
				{"provider-body", "libs/answer/src/answer.fy", strings.Replace(closureValueCacheProvider, "captured+x", "captured+x+1", 1), 43},
				{"capture-initializer", "libs/answer/src/answer.fy", strings.Replace(closureValueCacheProvider, "captured=seed", "captured=seed+2", 1), 44},
				{"caller-body", "src/main.fy", strings.Replace(closureValueCacheMain, "action,2", "action,3", 1), 45},
				{"package-identity", "libs/answer/fairy.toml", strings.Replace(workspaceSessionMemberManifest, "example.session.answer", "example.session.changed", 1), 45},
			} {
				t.Logf("edit %s", edit.name)
				before := closureValueCandidateCacheSnapshot(t, root)
				writeWorkspaceFile(t, root, edit.path, edit.source)
				c.cache(7, "cache: skipped", "check", "--locked")
				if !reflect.DeepEqual(before, closureValueCandidateCacheSnapshot(t, root)) {
					t.Fatal("stale locked source published cache data")
				}
				out, diag := c.run(edit.value, "run", "--cache-root", "cache", "--cache-status")
				if out != "" || diag != "cache: cold stored\n" && diag != "cache: mixed\n" && diag != "cache: units hit\n" {
					t.Fatalf("edit did not rebuild and publish: %q %s", out, diag)
				}
				if reflect.DeepEqual(before, closureValueCandidateCacheSnapshot(t, root)) {
					t.Fatal("edit did not change retained generations")
				}
				closureValueCandidateWarm(t, &c, edit.value)
			}
			// Corrupt all actual generations so neither whole-project nor UNIT
			// reuse can hide the required repair.
			for name := range closureValueCandidateCacheSnapshot(t, root) {
				if err := os.WriteFile(filepath.Join(root, "cache", name), []byte("corrupt"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			c.cache(0, "cache: cold stored", "check", "--locked")
			closureValueCandidateCacheFamily(t, root, "FYCACHE2", "FYCKEYS2")
			closureValueCandidateWarm(t, &c, 45)
			before := closureValueCandidateCacheSnapshot(t, root)
			writeWorkspaceFile(t, root, "src/main.fy", "module main;fn i32 main(){return true;}")
			c.cache(7, "cache: skipped", "check", "--locked")
			c.cache(3, "cache: cold uncached", "check")
			if !reflect.DeepEqual(before, closureValueCandidateCacheSnapshot(t, root)) {
				t.Fatal("invalid source published or changed cache generations")
			}
			writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(closureValueCacheMain, "action,2", "action,3", 1))
			c.cache(0, "cache: hit", "check")
			closureValueCandidateWarm(t, &c, 45)
			t.Logf("%d fresh CLI processes: closure cache edits, identity, repair, refusal and immutable warm hits", c.calls)
			// Non-closure sources retain their original envelope and execute
			// through the same CLI after the closure cache routing was added.
			for _, legacy := range []struct{ name, provider, envelope, ledger string }{
				{"scalar", "module answer;pub fn i32 value(){return 42;}", "FYCACHE0", "FYCKEYS0"},
				{"aggregate", "module answer;struct Cell<T>{T value;}pub fn i32 value(){let cell=Cell<i32>{.value=42};return cell.value;}", "FYCACHE1", "FYCKEYS1"},
			} {
				if !t.Run(legacy.name, func(t *testing.T) {
					root := writeWorkspaceSessionFixture(t, "module main;import answer as a;fn i32 main(){return a::value();}")
					writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", legacy.provider)
					if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
						t.Fatal(err)
					}
					c := copiedClosureCommand{t: t, image: image, root: root}
					c.cache(0, "cache: cold stored", "check")
					closureValueCandidateCacheFamily(t, root, legacy.envelope, legacy.ledger)
					closureValueCandidateWarm(t, &c, 42)
					if out := c.cache(42, "cache: hit", "run", "--locked"); out != "" {
						t.Fatal(out)
					}
					t.Logf("%d fresh processes: legacy %s cold/warm and both native profiles", c.calls, legacy.name)
				}) {
					return
				}
			}
		}) {
			return
		}
	}
}

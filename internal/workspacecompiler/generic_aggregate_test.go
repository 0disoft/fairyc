package workspacecompiler

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

func TestGenericAggregateCompilerFallbackAndStableAdmission(t *testing.T) {
	session, root := genericWorkspaceSession(t, 32<<20)
	defer session.Close()
	writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), `module answer;
pub resource Box<T> { T value; cleanup {} }
pub fn Box<T> make<T>(T value) { return Box<T> { .value = value }; }`)
	writeFile(t, filepath.Join(root, "src", "main.fy"), `module main; import answer as a;
fn i32 main() { let a::Box<i32> value = a::make<i32>(42); return value.value; }`)
	workspace := loadCacheWorkspace(t, root)
	for range 2 {
		result, err := session.Check(workspace)
		if err != nil || !result.Compiler.Verified || result.Cache.Outcome != OutcomeNotEligible || !result.Cache.Lowered || result.Cache.Hits != 0 || result.Cache.Stored != 0 {
			t.Fatalf("aggregate cache admission: %v %+v", err, result.Cache)
		}
		assertWorkspaceResult(t, workspace, result.Compiler, 42)
		assertGenericWorkspaceNative(t, workspace, result)
	}
	stable, err := Check(workspace)
	if err != nil || stable.Verified || !slices.ContainsFunc(stable.Semantic.Unsupported, func(item semantic.Unsupported) bool { return item.Feature == "generic-aggregate-specialization" }) {
		t.Fatalf("stable aggregate admission: %v %+v", err, stable)
	}
	source, err := NewSourceSession(Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	for range 2 {
		rejected, err := source.Check(workspace)
		if err != nil || rejected.Compiler.Verified || rejected.Cache.Outcome != OutcomeNotEligible || rejected.Cache.Hits != 0 || rejected.Cache.Stored != 0 {
			t.Fatalf("stable session admitted aggregates: %v %+v", err, rejected.Cache)
		}
	}
}

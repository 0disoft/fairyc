package workspacecompiler

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
)

func TestStorageTransferSessionSkipsOnlyWarmLowering(t *testing.T) {
	_, root := genericWorkspaceSession(t, 32<<20)
	configuration := genericPersistentConfiguration(t, t.TempDir())
	// The private constructor rejects disk policy before opening any root.
	if _, err := NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits()); err == nil {
		t.Fatal("private transfer session admitted a persistent cache")
	}
	configuration = Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
	session, err := NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(session.generic.keys.compilerBuild, ":storage-transfer-v0") {
		t.Fatal("private session lost its key compatibility domain")
	}
	workspace := loadCacheWorkspace(t, root)
	if _, err := session.Check(workspace); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("lowering reached")
	session.generic.lower = func(compiler.Analysis) (compiler.Result, error) { return compiler.Result{}, sentinel }
	warm, err := session.Check(workspace)
	if err != nil || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered {
		t.Fatalf("warm cache invoked lowering: %v %+v", err, warm.Cache)
	}
	writeFile(t, filepath.Join(root, "src", "main.fy"), strings.Replace(genericMain, "42", "43", 1))
	if _, err := session.Check(loadCacheWorkspace(t, root)); !errors.Is(err, sentinel) {
		t.Fatalf("source change failed to invoke fallback lowering: %v", err)
	}
}

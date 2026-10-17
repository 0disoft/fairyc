package workspacecompiler

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
)

func TestStorageTransferSessionSkipsOnlyWarmLowering(t *testing.T) {
	configuration := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
	t.Run("memory", func(t *testing.T) { testStorageTransferSessionLowering(t, configuration) })
	t.Run("disk", func(t *testing.T) {
		testStorageTransferSessionLowering(t, genericPersistentConfiguration(t, t.TempDir()))
	})
}

func testStorageTransferSessionLowering(t *testing.T, configuration Configuration) {
	t.Helper()
	root := writeCacheWorkspace(t, genericAnswer)
	writeFile(t, filepath.Join(root, "src", "main.fy"), genericMain)
	session, err := NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	domain := ":storage-transfer-v0"
	if configuration.Persistent != nil {
		domain = "fairyc.compiler-storage-stable-cache/0;"
	}
	if !strings.Contains(session.generic.keys.compilerBuild, domain) {
		t.Fatal("private session lost its key compatibility domain")
	}
	workspace := loadCacheWorkspace(t, root)
	if _, err := session.Check(workspace); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("lowering reached")
	session.generic.lower = func(compiler.Analysis, admission.Capabilities) (compiler.Result, error) {
		return compiler.Result{}, sentinel
	}
	warm, err := session.Check(workspace)
	if err != nil || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered {
		t.Fatalf("warm cache invoked lowering: %v %+v", err, warm.Cache)
	}
	writeFile(t, filepath.Join(root, "src", "main.fy"), strings.Replace(genericMain, "42", "43", 1))
	if _, err := session.Check(loadCacheWorkspace(t, root)); !errors.Is(err, sentinel) {
		t.Fatalf("source change failed to invoke fallback lowering: %v", err)
	}
}

func TestStorageTransferSessionPersistentRequiresExactPolicy(t *testing.T) {
	configuration := genericPersistentConfiguration(t, t.TempDir())
	configuration.Cache = packagecache.Limits{MaxEntries: 1, MaxBytes: 1}
	if _, err := NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits()); err == nil {
		t.Fatal("transfer session accepted mixed cache policies")
	}
	configuration.Cache = packagecache.Limits{}
	configuration.Persistent.CompilerBuild = "latest"
	if _, err := NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits()); err == nil {
		t.Fatal("transfer session accepted a moving compiler identity")
	}
}

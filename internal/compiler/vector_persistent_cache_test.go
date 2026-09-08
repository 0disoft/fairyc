package compiler_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func vectorPersistentConfiguration(t *testing.T, root string) workspacecompiler.Configuration {
	t.Helper()
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	return workspacecompiler.Configuration{Target: target.X8664WindowsMSVC, MaxArtifactBytes: 8 << 20,
		Persistent: &workspacecompiler.PersistentConfiguration{Root: root, CompilerBuild: identity.String(),
			Limits: packagecache.DiskLimits{MaxEntries: 8, MaxBytes: 32 << 20, MaxRecordBytes: 16 << 20, TempMaxAge: time.Minute}}}
}

func TestPrivateVectorPersistentCacheCrossProcess(t *testing.T) {
	client, provider := vectorPackageSources(t, nil)
	root := writeVectorCacheWorkspace(t, client, provider)
	cacheRoot := t.TempDir()
	configuration := vectorPersistentConfiguration(t, cacheRoot)
	session, err := workspacecompiler.NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cold, err := session.Check(workspace)
	if err != nil || !cold.Compiler.Verified || cold.Cache.Storage != workspacecompiler.StorageDisk || cold.Cache.Stored != 2 {
		t.Fatalf("cold vector disk cache: %v %+v", err, cold.Cache)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPrivateVectorPersistentCacheChild$", "-test.count=1", "-test.v")
	command.Env = append(os.Environ(), "FAIRYC_VECTOR_DISK_WORKSPACE="+root, "FAIRYC_VECTOR_DISK_CACHE="+cacheRoot)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "private-vector-disk-hit") {
		t.Fatalf("fresh vector disk process: %v %s", err, output)
	}
	t.Logf("fresh-process evidence:\n%s", output)

	// The same source compiled by a different executable identity cannot reuse
	// the first compiler's records, even in the same format directory.
	identity, err := buildidentity.Parse(configuration.Persistent.CompilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	identity[0] ^= 1
	configuration.Persistent.CompilerBuild = identity.String()
	other, err := workspacecompiler.NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	changed, err := other.Check(workspace)
	if err != nil || !changed.Compiler.Verified || changed.Cache.Hits != 0 || changed.Cache.Misses != 2 || !changed.Cache.Lowered {
		t.Fatalf("different compiler reused transfer cache: %v %+v", err, changed.Cache)
	}
}

func TestPrivateVectorPersistentCacheChild(t *testing.T) {
	root := os.Getenv("FAIRYC_VECTOR_DISK_WORKSPACE")
	if root == "" {
		t.Skip("fresh-process consumer only")
	}
	session, err := workspacecompiler.NewStorageTransferSession(vectorPersistentConfiguration(t, os.Getenv("FAIRYC_VECTOR_DISK_CACHE")), genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Check(workspace)
	if err != nil || !result.Compiler.Verified || result.Cache.Outcome != workspacecompiler.OutcomeAllHit ||
		result.Cache.Hits != 2 || result.Cache.Lowered || result.Cache.Stored != 0 {
		t.Fatalf("fresh vector disk cache did not reuse both artifacts: %v %+v", err, result.Cache)
	}
	if len(result.Compiler.GenericProvenance.Declarations()) == 0 || len(result.Compiler.GenericProvenance.Specializations()) == 0 {
		t.Fatal("fresh disk hit lost current source provenance")
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("native", func(t *testing.T) {
		runVectorStorageMIRNativeLane(t, result.Compiler.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", vectorResourceProbe, true)
	})
	fmt.Println("private-vector-disk-hit")
}

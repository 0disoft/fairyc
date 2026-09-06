package compiler_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func openGenericDiskFixture(t *testing.T, root string) *packageartifact.GenericStableCache {
	t.Helper()
	cache, err := packageartifact.NewGenericStableCache(root, packagecache.DiskLimits{MaxEntries: 3, MaxBytes: 32 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cache.Close() })
	return cache
}

func TestGenericStableCacheRunsInFreshProcess(t *testing.T) {
	compiled := genericCachedPackageResult(t)
	_, coordinates := genericPackageInput(t)
	artifacts, err := packageartifact.ExtractAll(compiled.MIR, map[hir.PackageID]packagegraph.InstanceID{1: {1}, 2: {2}, 3: {3}}, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	cache := openGenericDiskFixture(t, root)
	// Closed fixture identities test transport; ComputeGeneric invalidation has
	// separate source/package tests and is not replaced by these test keys.
	for _, artifact := range artifacts {
		id := byte(artifact.PackageID)
		if err := cache.Put(packagecache.GenericKey{id}, moduleinterface.GenericHash{id}, artifact); err != nil {
			t.Fatal(err)
		}
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGenericStableCacheChild$", "-test.count=1")
	command.Env = append(os.Environ(), "FAIRYC_GENERIC_DISK_CHILD="+root)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "generic-disk-rebound-42") {
		t.Fatalf("fresh process: %v %s", err, output)
	}
}

func TestGenericStableCacheChild(t *testing.T) {
	root := os.Getenv("FAIRYC_GENERIC_DISK_CHILD")
	if root == "" {
		t.Skip("subprocess consumer only")
	}
	_, coordinates := genericPackageInput(t)
	cache := openGenericDiskFixture(t, root)
	var artifacts []packageartifact.StableArtifact
	for _, id := range []byte{1, 2, 3} {
		lookup := cache.Get(packagecache.GenericKey{id}, moduleinterface.GenericHash{id}, packagegraph.InstanceID{id}, coordinates[hir.PackageID(id)])
		if lookup.Status != packageartifact.LookupHit {
			t.Fatalf("disk read: %s %s", lookup.Status, lookup.Reason)
		}
		artifacts = append(artifacts, lookup.Artifact)
	}
	// Fresh package, function, foreign and local numbering must not depend on
	// anything the producer retained in memory. No lowering runs in this child.
	remapped := map[hir.PackageID]packagegraph.CoordinateID{4: coordinates[1], 7: coordinates[2], 9: coordinates[3]}
	program, err := packageartifact.RebindStableArtifacts(artifacts, remapped, 100, 200, 300)
	if err != nil {
		t.Fatal(err)
	}
	entry := hir.QualifiedModulePath(4, "main") + "::main"
	outcome, err := interpreter.Execute(program, entry)
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("disk execution: %+v %v", outcome, err)
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		toolchain, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		runGenericNative(t, toolchain, compiler.Result{MIR: program, Verified: true}, entry, 42)
	}
	fmt.Println("generic-disk-rebound-42")
}

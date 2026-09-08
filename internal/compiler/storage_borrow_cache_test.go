package compiler_test

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestPrivateVectorSDKWorkspaceBorrowRebind(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("borrow rebind execution requires Windows amd64")
	}
	program := storageBorrowNativeMIR(t, true)
	coordinate := genericCoordinate(t)
	artifacts, err := packageartifact.ExtractAllStorageTransfers(program,
		map[hir.PackageID]packagegraph.InstanceID{1: {1}}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil || len(artifacts) != 1 {
		t.Fatalf("extract borrow: %v", err)
	}
	const limit = 8 << 20
	stable, err := packageartifact.DescribeStorageTransferStableArtifact(artifacts[0], limit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := packageartifact.EncodeStorageTransferStableArtifact(stable, limit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packageartifact.EncodeStableArtifact(stable, limit); err == nil {
		t.Fatal("ordinary writer admitted storage borrow")
	}
	forged := bytes.Replace(encoded, []byte(packageartifact.StorageStableArtifactSchema), []byte(packageartifact.StableArtifactSchema), 1)
	forged = bytes.Replace(forged, []byte(packageartifact.StorageStableArtifactVersion), []byte(packageartifact.StableArtifactVersion), 1)
	if _, err := packageartifact.DecodeStableArtifact(forged, limit); err == nil {
		t.Fatal("schema substitution admitted storage borrow")
	}
	decoded, err := packageartifact.DecodeStorageTransferStableArtifact(encoded, limit)
	if err != nil {
		t.Fatal(err)
	}
	rebound, err := packageartifact.RebindStorageTransferStableArtifacts([]packageartifact.StableArtifact{decoded},
		map[hir.PackageID]packagegraph.CoordinateID{71: coordinate}, 1000, 10000, 500)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.EmitStorageTransfers(rebound, hir.QualifiedModulePath(71, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	probe := strings.Replace(storageBorrowNativeProbe, "return &item.value;", "return &item;", 1)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar+probe)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("rebound borrow: %+v %v", outcome, err)
			}
		})
	}
}

package packageartifact

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestBodyPackageIndexPreservesScanResults(t *testing.T) {
	process := typedProjectionArtifact(t, 1)
	body, err := DescribeStableBody(process.Functions[0], process, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	symbols := map[FunctionID]hir.SymbolID{body.ID: 30000}
	unused := packagegraph.CoordinateID{0xff}
	for _, tc := range []struct {
		name        string
		coordinates map[hir.PackageID]packagegraph.CoordinateID
		owner       hir.PackageID
		wantError   string
	}{
		{"fresh owner", map[hir.PackageID]packagegraph.CoordinateID{77: body.Package}, 77, ""},
		{"next operation", map[hir.PackageID]packagegraph.CoordinateID{177: body.Package}, 177, ""},
		{"unused ambiguity", map[hir.PackageID]packagegraph.CoordinateID{77: body.Package, 88: unused, 99: unused}, 77, ""},
		{"used ambiguity", map[hir.PackageID]packagegraph.CoordinateID{77: body.Package, 78: body.Package}, 0, "multiple local packages"},
		{"missing owner", map[hir.PackageID]packagegraph.CoordinateID{88: unused}, 0, "has no local package"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scanned, scanErr := rebindStableBody(body, process.StableContext, tc.coordinates, symbols, nil, 1, stableBodyLimit)
			context := &bodyContext{
				descriptor: process.StableContext, maxBytes: stableBodyLimit,
				packages: indexedPackageResolver(tc.coordinates),
			}
			indexed, indexErr := rebindValidatedStableBodyInContext(body, context, tc.coordinates, symbols, nil, 1)
			if tc.wantError != "" {
				if scanErr == nil || indexErr == nil || scanErr.Error() != indexErr.Error() || !strings.Contains(indexErr.Error(), tc.wantError) {
					t.Fatalf("lookup errors differ: scan=%v, index=%v", scanErr, indexErr)
				}
				return
			}
			if scanErr != nil || indexErr != nil || !reflect.DeepEqual(scanned, indexed) {
				t.Fatalf("lookup results differ: scan=%v, index=%v", scanErr, indexErr)
			}
			wantType := hir.ParseType(hir.QualifiedModulePath(tc.owner, "main::Type_0"))
			if indexed.Span.Package != tc.owner || indexed.ParameterTypes[0] != wantType {
				t.Fatalf("body or nominal parameter has wrong fresh owner: %+v", indexed)
			}
		})
	}
}

func BenchmarkRebindBodiesAcrossPackages(b *testing.B) {
	for _, count := range []int{8, 64, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			process := typedProjectionArtifact(b, 256)
			stable, err := DescribeStableArtifact(process, stablePackageArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
			for id, coordinate := range process.PackageCoordinates {
				coordinates[id] = coordinate
			}
			if len(coordinates) > count {
				b.Fatal("fixture has more packages than the benchmark target")
			}
			for i := 0; len(coordinates) < count; i++ {
				coordinates[hir.PackageID(1000+i)] = packagegraph.CoordinateID{0xff, byte(i), byte(i >> 8)}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				rebound, err := RebindStableArtifactsWithBindings([]StableArtifact{stable}, coordinates, 30000, 40000, 1)
				if err != nil || len(rebound.MIR.Functions) != 256 {
					b.Fatalf("rebind bodies across packages: %v", err)
				}
			}
		})
	}
}

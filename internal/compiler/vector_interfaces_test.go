package compiler_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestPrivateVectorPackageInterfaces(t *testing.T) {
	input, coordinates := vectorPackageInput(t)
	analysis, err := compiler.AnalyzeStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(analysis.Interfaces) != 2 || len(analysis.OrdinaryInterfaces) != 2 || len(analysis.Specializations[2]) == 0 {
		t.Fatalf("transfer interface publication: %+v %v", analysis, err)
	}
	records := analysis.Analysis.GenericProvenance.PackageInterfaces()
	if len(records) != 2 {
		t.Fatalf("interface provenance: %+v", records)
	}
	for _, record := range records {
		if record.Coordinate != coordinates[record.Package] || record.Generic != analysis.Interfaces[record.Package] || record.Ordinary != analysis.OrdinaryInterfaces[record.Package] {
			t.Fatal("interface provenance owner/hash mismatch")
		}
	}
	records[0].Coordinate[0] ^= 1
	if reflect.DeepEqual(records, analysis.Analysis.GenericProvenance.PackageInterfaces()) {
		t.Fatal("interface getter aliases retained records")
	}
	checked, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !checked.Verified || !reflect.DeepEqual(checked.GenericProvenance.PackageInterfaces(), analysis.Analysis.GenericProvenance.PackageInterfaces()) {
		t.Fatalf("lowering dropped source identities: %v", err)
	}
	ordinary, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err == nil && len(ordinary.Interfaces) != 0 {
		t.Fatal("ordinary lane published transfer interface")
	}

	slices.Reverse(input.Files)
	for i := range input.Files {
		input.Files[i].Package += 10
	}
	input.Imports[0].FromPackage += 10
	input.Imports[0].ToPackage += 10
	repeated, err := compiler.AnalyzeStorageTransferPackages(input,
		map[hir.PackageID]packagegraph.CoordinateID{11: coordinates[1], 12: coordinates[2]}, genericplan.DefaultPlanLimits())
	if err != nil || !reflect.DeepEqual(analysis.Plan, repeated.Plan) {
		t.Fatalf("reordered plan: %v", err)
	}
	for id, hash := range analysis.Interfaces {
		if repeated.Interfaces[id+10] != hash || repeated.OrdinaryInterfaces[id+10] != analysis.OrdinaryInterfaces[id] {
			t.Fatal("local IDs or input order changed source interface")
		}
	}
}

func TestPrivateVectorPackageInterfaceInvalidation(t *testing.T) {
	input, coordinates := vectorPackageInput(t)
	baseline, err := compiler.AnalyzeStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	changed, changedCoordinates := vectorPackageInputEdited(t, func(provider string) string {
		return strings.Replace(provider, "return storage.length;", "let usize count = storage.length; return count;", 1)
	})
	provider, err := compiler.AnalyzeStorageTransferPackages(changed, changedCoordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(provider.Interfaces) != 2 || provider.Interfaces[2] == baseline.Interfaces[2] {
		t.Fatalf("provider closure change missing: %v", err)
	}
	changed, changedCoordinates = vectorPackageInput(t, [2]string{"cleanup { let u32 status = unsafe { drop_raw(this.handle) }; }", "cleanup { let u32 status = unsafe { drop_raw(this.handle) }; assert(status == 0); }"})
	nominal, err := compiler.AnalyzeStorageTransferPackages(changed, changedCoordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(nominal.Interfaces) != 2 || reflect.DeepEqual(baseline.Plan, nominal.Plan) {
		t.Fatalf("nominal definition change missing from plan: %v", err)
	}
	changed, changedCoordinates = vectorPackageInput(t, [2]string{"return 41;", "assert(moved.length == 8); return 41;"})
	rejected, err := compiler.AnalyzeStorageTransferPackages(changed, changedCoordinates, genericplan.DefaultPlanLimits())
	if err == nil && (len(rejected.Interfaces) != 0 || len(rejected.OrdinaryInterfaces) != 0 || len(rejected.Analysis.GenericProvenance.PackageInterfaces()) != 0) {
		t.Fatal("failed Flow published interface identities")
	}
}

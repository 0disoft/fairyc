package compiler_test

import (
	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"reflect"
	"testing"
)

func TestPolicyScalarCompatibility(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	legacy, legacyPlan, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	current, plan, err := compiler.CheckWithPolicy(input, coordinates, genericplan.DefaultPlanLimits(), admission.Policy[hir.PackageID]{})
	if err != nil || !current.Verified || !reflect.DeepEqual(current, legacy) || !reflect.DeepEqual(plan, legacyPlan) {
		t.Fatalf("policy changed scalar compilation: %v", err)
	}
}

func TestPolicyStorageAdmission(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", storageSource)
	input := semantic.PackageInput{Files: []semantic.File{file}}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{file.Package: genericCoordinate(t)}
	policy := admission.Policy[hir.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageQueries, Arguments: admission.NominalArguments}, StorageOwners: []hir.PackageID{hir.DefaultPackage}}
	result, _, err := compiler.CheckWithPolicy(input, coordinates, genericplan.DefaultPlanLimits(), policy)
	if err != nil || !result.Verified {
		t.Fatalf("scoped query: %v %+v", err, result.Semantic)
	}
	if len(mir.Verify(result.MIR)) == 0 || len(mir.VerifyWithCapabilities(result.MIR, policy.Capabilities)) != 0 {
		t.Fatal("MIR admission boundary changed")
	}
	denied, _, err := compiler.CheckWithPolicy(input, coordinates, genericplan.DefaultPlanLimits(), admission.Policy[hir.PackageID]{})
	if err != nil || denied.Verified || len(denied.Semantic.Diagnostics) == 0 {
		t.Fatalf("ordinary query admission: %v %+v", err, denied.Semantic)
	}
	for name, invalid := range map[string]admission.Policy[hir.PackageID]{
		"missing-owner":    {Capabilities: policy.Capabilities, StorageOwners: []hir.PackageID{999}},
		"empty-owners":     {Capabilities: policy.Capabilities, StorageOwners: []hir.PackageID{}},
		"duplicate-owners": {Capabilities: policy.Capabilities, StorageOwners: []hir.PackageID{0, 0}},
		"unowned-sdk":      {Capabilities: admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments, Templates: admission.SDKOwnerTemplates}},
	} {
		t.Run(name, func(t *testing.T) {
			result, _, err := compiler.CheckWithPolicy(input, coordinates, genericplan.DefaultPlanLimits(), invalid)
			if err == nil || result.Verified {
				t.Fatalf("malformed policy accepted: %v", err)
			}
		})
	}
	if _, err := compiler.LowerAnalyzedWithCapabilities(compiler.Analysis{}, admission.Capabilities{Storage: 255}); err == nil {
		t.Fatal("lowering accepted unknown admission")
	}
}

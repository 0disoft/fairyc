package genericplan

import (
	"slices"
	"sync"
	"testing"
)

func TestPlanBuildsDependencyFirstStaticStorageSpecializations(t *testing.T) {
	catalog := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	request := Request{Declaration: writeDecl, Arguments: []TypeID{addressType, u64Type}}
	plan, err := catalog.Plan([]Request{request, request}, DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Roots) != 1 || len(plan.Specializations) != 3 {
		t.Fatalf("roots = %d, specializations = %d, want 1 and 3", len(plan.Roots), len(plan.Specializations))
	}
	addressEncode := specializationBy(t, plan, encodeDecl, addressType)
	u64Encode := specializationBy(t, plan, encodeDecl, u64Type)
	write := specializationBy(t, plan, writeDecl, addressType, u64Type)
	if write.ID != plan.Roots[0] {
		t.Fatalf("root = %s, want %s", plan.Roots[0], write.ID)
	}
	if !slices.Contains(write.Dependencies, addressEncode.ID) || !slices.Contains(write.Dependencies, u64Encode.ID) {
		t.Fatalf("write dependencies = %v", write.Dependencies)
	}
	positions := make(map[SpecializationID]int, len(plan.Specializations))
	for index, specialization := range plan.Specializations {
		positions[specialization.ID] = index
	}
	if positions[addressEncode.ID] >= positions[write.ID] || positions[u64Encode.ID] >= positions[write.ID] {
		t.Fatal("consumer was emitted before a dependency")
	}
	if len(write.Constraints) != 2 || write.Constraints[0].Provider != "sdk::encode_address" || write.Constraints[1].Provider != "sdk::encode_u64" {
		t.Fatalf("resolved constraints = %#v", write.Constraints)
	}
}

func TestPlanIsIndependentOfCatalogAndRequestInsertionOrder(t *testing.T) {
	first := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	second := testCatalog(t, catalogVariant{reverse: true, addressProviderBody: "encode-address/v1"})
	requestsA := []Request{
		{Declaration: writeDecl, Arguments: []TypeID{addressType, u64Type}},
		{Declaration: encodeDecl, Arguments: []TypeID{coinType}},
	}
	requestsB := []Request{requestsA[1], requestsA[0]}
	planA, err := first.Plan(requestsA, DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	planB, err := second.Plan(requestsB, DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(planA.Roots, planB.Roots) {
		t.Fatalf("roots differ: %v != %v", planA.Roots, planB.Roots)
	}
	if len(planA.Specializations) != len(planB.Specializations) {
		t.Fatalf("specialization count differs")
	}
	for index := range planA.Specializations {
		if planA.Specializations[index].ID != planB.Specializations[index].ID {
			t.Fatalf("specialization %d differs: %s != %s", index, planA.Specializations[index].ID, planB.Specializations[index].ID)
		}
	}
}

func TestPlanRejectsMissingAbilityBeforeSpecialization(t *testing.T) {
	catalog := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	_, err := catalog.Plan([]Request{{Declaration: writeDecl, Arguments: []TypeID{coinType, u64Type}}}, DefaultPlanLimits())
	if ErrorCode(err) != CodeAbilityUnsatisfied {
		t.Fatalf("error = %v, want %s", err, CodeAbilityUnsatisfied)
	}
}

func TestPlanRejectsMissingExactCapabilityImplementation(t *testing.T) {
	catalog := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	_, err := catalog.Plan([]Request{{Declaration: encodeDecl, Arguments: []TypeID{opaqueType}}}, DefaultPlanLimits())
	if ErrorCode(err) != CodeImplementationMissing {
		t.Fatalf("error = %v, want %s", err, CodeImplementationMissing)
	}
}

func TestReachableProviderChangeInvalidatesOnlyDependentSpecializations(t *testing.T) {
	first := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	second := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v2"})
	request := []Request{{Declaration: writeDecl, Arguments: []TypeID{addressType, u64Type}}}
	planA, err := first.Plan(request, DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	planB, err := second.Plan(request, DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	addressA := specializationBy(t, planA, encodeDecl, addressType)
	addressB := specializationBy(t, planB, encodeDecl, addressType)
	u64A := specializationBy(t, planA, encodeDecl, u64Type)
	u64B := specializationBy(t, planB, encodeDecl, u64Type)
	writeA := specializationBy(t, planA, writeDecl, addressType, u64Type)
	writeB := specializationBy(t, planB, writeDecl, addressType, u64Type)
	if addressA.ID == addressB.ID || writeA.ID == writeB.ID {
		t.Fatal("reachable provider change did not invalidate dependent specialization IDs")
	}
	if u64A.ID != u64B.ID {
		t.Fatal("unrelated specialization was invalidated")
	}
}

func TestPlanRejectsDependencyCycle(t *testing.T) {
	builder, err := NewBuilder(DefaultCatalogLimits())
	if err != nil {
		t.Fatal(err)
	}
	mustAddType(t, builder, TypeDescriptor{ID: u64Type, Owner: "sdk", Abilities: []Ability{AbilityScalar}, Definition: DigestText("type", "U64/v1")})
	for _, declaration := range []Declaration{
		{
			ID: "sdk::A", Owner: "sdk", Parameters: []TypeParameter{{Name: "T"}},
			Dependencies: []Dependency{{Declaration: "sdk::B", Arguments: []TypeExpression{ParameterType("T")}}},
			Definition:   DigestText("declaration", "A/v1"),
		},
		{
			ID: "sdk::B", Owner: "sdk", Parameters: []TypeParameter{{Name: "T"}},
			Dependencies: []Dependency{{Declaration: "sdk::A", Arguments: []TypeExpression{ParameterType("T")}}},
			Definition:   DigestText("declaration", "B/v1"),
		},
	} {
		if err := builder.AddDeclaration(declaration); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	_, err = catalog.Plan([]Request{{Declaration: "sdk::A", Arguments: []TypeID{u64Type}}}, DefaultPlanLimits())
	if ErrorCode(err) != CodeSpecializationCycle {
		t.Fatalf("error = %v, want %s", err, CodeSpecializationCycle)
	}
}

func TestPlanEnforcesTotalAndPerDeclarationBudgets(t *testing.T) {
	catalog := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	_, err := catalog.Plan(
		[]Request{{Declaration: writeDecl, Arguments: []TypeID{addressType, u64Type}}},
		PlanLimits{MaxSpecializations: 2, MaxSpecializationsPerDeclaration: 8, MaxExpansionDepth: 8},
	)
	if ErrorCode(err) != CodePlanningBudgetExceeded {
		t.Fatalf("total budget error = %v, want %s", err, CodePlanningBudgetExceeded)
	}
	_, err = catalog.Plan(
		[]Request{
			{Declaration: encodeDecl, Arguments: []TypeID{addressType}},
			{Declaration: encodeDecl, Arguments: []TypeID{u64Type}},
		},
		PlanLimits{MaxSpecializations: 8, MaxSpecializationsPerDeclaration: 1, MaxExpansionDepth: 8},
	)
	if ErrorCode(err) != CodePlanningBudgetExceeded {
		t.Fatalf("per-declaration budget error = %v, want %s", err, CodePlanningBudgetExceeded)
	}
}

func TestCatalogPlanIsConcurrentAndReadOnly(t *testing.T) {
	catalog := testCatalog(t, catalogVariant{addressProviderBody: "encode-address/v1"})
	request := []Request{{Declaration: writeDecl, Arguments: []TypeID{addressType, u64Type}}}
	baseline, err := catalog.Plan(request, DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	const workers = 32
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			plan, err := catalog.Plan(request, DefaultPlanLimits())
			if err != nil {
				errors <- err
				return
			}
			if !slices.Equal(plan.Roots, baseline.Roots) || len(plan.Specializations) != len(baseline.Specializations) {
				errors <- &Error{Code: CodeInvalidDeclaration, Message: "concurrent plan differs"}
				return
			}
			for index := range plan.Specializations {
				if plan.Specializations[index].ID != baseline.Specializations[index].ID {
					errors <- &Error{Code: CodeInvalidDeclaration, Message: "concurrent specialization differs"}
					return
				}
			}
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
}

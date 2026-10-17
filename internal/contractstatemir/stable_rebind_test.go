package contractstatemir

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableStateBodyRebindsFreshOwnersCapabilitiesAndExecution(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-e8756812ba7956f1f997a35a522b7f1735d8679c99c48b140570f4cd690c1a91")
	original := stableStateBodyPackage(t, 2, 17, 33)
	artifacts, err := DescribeStableHIRStateBodies(original, []hir.PackageID{2}, map[hir.PackageID]packagegraph.CoordinateID{2: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[2]
	fresh := stableStateBodyPackage(t, 9, 501, 700)
	fresh.StateTransactions = nil
	fresh.Modules[0].Functions = append(fresh.Modules[0].Functions, hir.Function{
		Symbol: 900, Name: "unrelated", ResultType: hir.Type("unsupported-local-test-type"),
	})
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{9: coordinate}

	rebound, err := RebindStableStateBody(body, body.Hash, fresh, 9, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	if rebound.Package != 9 || rebound.Hash != body.Hash || rebound.Layout.Hash == (contractstate.LayoutHash{}) {
		t.Fatalf("rebound identity = %+v", rebound)
	}
	if len(rebound.Transactions) != 2 || rebound.Transactions[0].Function != 501 || rebound.Transactions[1].Function != 700 {
		t.Fatalf("rebound transactions = %+v", rebound.Transactions)
	}
	if rebound.Transactions[0].Program.Capability == rebound.Transactions[1].Program.Capability {
		t.Fatal("separate rebound transactions share one capability")
	}
	for transactionIndex, transaction := range rebound.Transactions {
		if failures := Verify(transaction.Program); len(failures) != 0 {
			t.Fatalf("rebound transaction %d verification = %v", transactionIndex, failures)
		}
		for operationIndex, operation := range transaction.Program.Operations {
			if operation.Capability != transaction.Program.Capability {
				t.Fatalf("transaction %d operation %d has another capability", transactionIndex, operationIndex)
			}
		}
	}

	again, err := RebindStableStateBody(body, body.Hash, fresh, 9, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	for index := range rebound.Transactions {
		if rebound.Transactions[index].Program.Capability == again.Transactions[index].Program.Capability {
			t.Fatalf("rebind %d reused transaction capability", index)
		}
	}
	aligned, err := RebindStableStateBodyWithSymbols(body, body.Hash, fresh, 9, coordinates, map[string]hir.SymbolID{
		"ledger::apply": 1001, "ledger::observe": 1002,
	})
	if err != nil {
		t.Fatal(err)
	}
	if aligned.Transactions[0].Function != 1001 || aligned.Transactions[1].Function != 1002 {
		t.Fatalf("aligned transactions = %+v", aligned.Transactions)
	}
	if _, err := RebindStableStateBodyWithSymbols(body, body.Hash, fresh, 9, coordinates, map[string]hir.SymbolID{
		"ledger::apply": 1001,
	}); err == nil || !strings.Contains(err.Error(), "no aligned MIR symbol") {
		t.Fatalf("missing aligned function symbol error = %v", err)
	}
	if _, err := RebindStableStateBodyWithSymbols(body, body.Hash, fresh, 9, coordinates, map[string]hir.SymbolID{
		"ledger::apply": 1001, "ledger::observe": 1001,
	}); err == nil || !strings.Contains(err.Error(), "share fresh symbol") {
		t.Fatalf("duplicate aligned function symbol error = %v", err)
	}

	store, err := NewMemoryStore(rebound.Layout)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := Execute(store, rebound.Transactions[0].Program, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != ExitCommit || len(outcome.Reads) != 1 {
		t.Fatalf("rebound execution = %+v", outcome)
	}
	assertRead(t, outcome.Reads[0], 1, true, Value{TypeID: contractstate.U64.TypeID(), Encoded: stateBodyU64(t, 10).Encoded})
}

func TestStableStateBodyRebindRejectsIdentityAndCurrentHIRDrift(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-903a7b9974986cccf6166e8978ca9cb26273d4776af6d1a5dd3b04e7060e2f41")
	otherCoordinate := stateBodyCoordinate(t, "fypc1-sha256-04fc4400f5e875c5c22ab2d50765d721df38a12a19ef3508deb432c7f4da92f3")
	original := stableStateBodyPackage(t, 1, 7, 9)
	artifacts, err := DescribeStableHIRStateBodies(original, []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	baseBody := artifacts[1]
	tests := []struct {
		name       string
		mutateBody func(*StableStateBodyArtifact, *StateBodyHash)
		mutateHIR  func(*hir.Package)
		coords     map[hir.PackageID]packagegraph.CoordinateID
		contains   string
	}{
		{name: "missing expected hash", mutateBody: func(_ *StableStateBodyArtifact, expected *StateBodyHash) { *expected = StateBodyHash{} }, contains: "expected body hash"},
		{name: "wrong expected hash", mutateBody: func(_ *StableStateBodyArtifact, expected *StateBodyHash) { expected[0] ^= 0xff }, contains: "does not match"},
		{name: "self-inconsistent body", mutateBody: func(body *StableStateBodyArtifact, expected *StateBodyHash) {
			body.Hash[0] ^= 0xff
			*expected = body.Hash
		}, contains: "hash does not match"},
		{name: "missing coordinate", coords: map[hir.PackageID]packagegraph.CoordinateID{}, contains: "has no PackageCoordinateId"},
		{name: "wrong owner coordinate", coords: map[hir.PackageID]packagegraph.CoordinateID{9: otherCoordinate}, contains: "does not own"},
		{name: "ambiguous coordinate", coords: map[hir.PackageID]packagegraph.CoordinateID{9: coordinate, 10: coordinate}, contains: "share one PackageCoordinateId"},
		{name: "missing current layout", mutateHIR: func(input *hir.Package) { input.StateLayouts = nil }, contains: "layout does not match"},
		{name: "current layout drift", mutateHIR: func(input *hir.Package) { input.StateLayouts[0].Fields[0].Name = "changed" }, contains: "layout does not match"},
		{name: "missing function", mutateHIR: func(input *hir.Package) { input.Modules = nil }, contains: "missing stable function"},
		{name: "duplicate logical function", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions = append(input.Modules[0].Functions, input.Modules[0].Functions[0])
		}, contains: "multiple functions"},
		{name: "signature drift", mutateHIR: func(input *hir.Package) { input.Modules[0].Functions[0].Parameters[0].Type = hir.TypeU32 }, contains: "signature does not match"},
		{name: "missing fresh symbol", mutateHIR: func(input *hir.Package) { input.Modules[0].Functions[0].Symbol = hir.NoSymbol }, contains: "no fresh symbol"},
		{name: "current unsafe owner", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].Unsafe = true
		}, contains: "unsafe owners are not admitted"},
		{name: "current unsafe owner body", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].Body.Tail = &hir.Expression{Kind: hir.ExpressionUnsafe}
		}, contains: "unsafe code is not admitted"},
		{name: "current raw-pointer owner local", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].Locals = []hir.Local{{Type: hir.RawPointer(hir.TypeU8, false)}}
		}, contains: "raw-pointer types are not admitted"},
		{name: "current receiver-bound owner", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].OwnerType = hir.Type("ledger::Account")
			input.Modules[0].Functions[0].Receiver = "shared"
		}, contains: "receiver-bound owners are not admitted"},
		{name: "current drop-glue owner", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].DropGlueFor = hir.TypeU32
		}, contains: "signature does not match"},
		{name: "current potentially trapping owner", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].Body.Tail = &hir.Expression{Kind: hir.ExpressionBinary, Operator: "/"}
		}, contains: "potentially trapping operations are not admitted"},
		{name: "current owned resource local", mutateHIR: func(input *hir.Package) {
			resourceType := hir.Type("ledger::FreshHandle")
			input.Modules[0].Resources = append(input.Modules[0].Resources, hir.Resource{Type: resourceType, DropKind: hir.ResourceDropEmpty})
			input.Modules[0].Functions[0].Locals = []hir.Local{{Type: resourceType}}
		}, contains: "owned resource values are not admitted"},
		{name: "current future owner expression", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].Body.Tail = &hir.Expression{Kind: hir.ExpressionKind(255)}
		}, contains: "expression kind \"ExpressionKind(255)\" is not admitted"},
		{name: "current owner call", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].Body.Tail = &hir.Expression{Kind: hir.ExpressionCall}
		}, contains: "contains a call; calls are not admitted"},
		{name: "current owner control flow", mutateHIR: func(input *hir.Package) {
			input.Modules[0].Functions[0].Body.Tail = &hir.Expression{Kind: hir.ExpressionMatch}
		}, contains: "contains dynamic control flow; dynamic control flow is not admitted"},
		{name: "shared fresh symbol", mutateHIR: func(input *hir.Package) { input.Modules[0].Functions[1].Symbol = input.Modules[0].Functions[0].Symbol }, contains: "share fresh symbol"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := cloneStableStateBodyArtifact(baseBody)
			expected := body.Hash
			if test.mutateBody != nil {
				test.mutateBody(&body, &expected)
			}
			fresh := stableStateBodyPackage(t, 9, 501, 700)
			fresh.StateTransactions = nil
			if test.mutateHIR != nil {
				test.mutateHIR(&fresh)
			}
			coordinates := test.coords
			if coordinates == nil {
				coordinates = map[hir.PackageID]packagegraph.CoordinateID{9: coordinate}
			}
			if _, err := RebindStableStateBody(body, expected, fresh, 9, coordinates); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestStableStateBodyRebindPreservesEmptyPackageBody(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-0f05e8aec70e99fa5420f3c82f7bbb082bf9126349965e2414300320236eeeda")
	artifacts, err := DescribeStableHIRStateBodies(hir.Package{}, []hir.PackageID{3}, map[hir.PackageID]packagegraph.CoordinateID{3: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[3]
	rebound, err := RebindStableStateBody(body, body.Hash, hir.Package{}, 77, map[hir.PackageID]packagegraph.CoordinateID{77: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	if rebound.Package != 77 || rebound.Hash != body.Hash || rebound.Layout.ContractUID != "" || rebound.Layout.Hash != (contractstate.LayoutHash{}) || rebound.Layout.Fields != nil || rebound.Transactions == nil || len(rebound.Transactions) != 0 {
		t.Fatalf("empty rebound body = %#v", rebound)
	}
}

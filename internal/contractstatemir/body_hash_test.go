package contractstatemir

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestHIRStateBodyHashIsStableAcrossLocalIdentitiesAndOrder(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-e8756812ba7956f1f997a35a522b7f1735d8679c99c48b140570f4cd690c1a91")
	base := stateBodyPackage(t, 2, 17)
	base.Modules[0].Functions = append(base.Modules[0].Functions, hir.Function{
		Symbol: 33, Name: "observe", ResultType: hir.TypeVoid, Effects: []hir.Effect{hir.EffectStorageRead},
	})
	base.StateTransactions = append(base.StateTransactions, hir.StateTransaction{
		Package: 2, Function: 33, Exit: hir.StateTransactionReject,
		Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}},
	})
	base.StateTransactions[0], base.StateTransactions[1] = base.StateTransactions[1], base.StateTransactions[0]

	hashes, err := ComputeHIRStateBodyHashes(base, []hir.PackageID{2}, map[hir.PackageID]packagegraph.CoordinateID{2: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	got := hashes[2]
	const want = "fycsbody1-sha256-e17e8bd4062f7a371bbb888927f12918ba77c9d03f31e60367baea8fea136a07"
	if got.String() != want {
		t.Fatalf("body hash = %s, want %s", got, want)
	}
	parsed, err := ParseStateBodyHash(got.String())
	if err != nil || parsed != got {
		t.Fatalf("ParseStateBodyHash() = %s, %v", parsed, err)
	}
	text, err := got.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var unmarshaled StateBodyHash
	if err := unmarshaled.UnmarshalText(text); err != nil || unmarshaled != got {
		t.Fatalf("UnmarshalText() = %s, %v", unmarshaled, err)
	}

	renumbered := stateBodyPackage(t, 9, 501)
	renumbered.Modules[0].Functions = append(renumbered.Modules[0].Functions, hir.Function{
		Symbol: 700, Name: "observe", ResultType: hir.TypeVoid, Effects: []hir.Effect{hir.EffectStorageRead},
	})
	renumbered.StateTransactions = append(renumbered.StateTransactions, hir.StateTransaction{
		Package: 9, Function: 700, Exit: hir.StateTransactionReject,
		Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}},
	})
	renumberedHashes, err := ComputeHIRStateBodyHashes(renumbered, []hir.PackageID{9}, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	if renumberedHashes[9] != got {
		t.Fatalf("renumbered hash = %s, want %s", renumberedHashes[9], got)
	}
}

func TestHIRStateBodyHashTracksExactLayoutSignatureAndOperations(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-903a7b9974986cccf6166e8978ca9cb26273d4776af6d1a5dd3b04e7060e2f41")
	compute := func(input hir.Package) StateBodyHash {
		t.Helper()
		result, err := ComputeHIRStateBodyHashes(input, []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
		if err != nil {
			t.Fatal(err)
		}
		return result[1]
	}
	base := stateBodyPackage(t, 1, 7)
	baseline := compute(base)

	changes := []struct {
		name   string
		mutate func(*hir.Package)
	}{
		{name: "coordinate-owned layout", mutate: func(value *hir.Package) { value.StateLayouts[0].ContractUID = "example.other" }},
		{name: "function signature", mutate: func(value *hir.Package) {
			value.Modules[0].Functions[0].Parameters = []hir.Parameter{{Type: hir.TypeU32}}
		}},
		{name: "operation value", mutate: func(value *hir.Package) { value.StateTransactions[0].Accesses[0].Value = stateBodyU64(t, 11) }},
		{name: "operation order", mutate: func(value *hir.Package) { slices.Reverse(value.StateTransactions[0].Accesses) }},
		{name: "exit", mutate: func(value *hir.Package) { value.StateTransactions[0].Exit = hir.StateTransactionReject }},
	}
	for _, test := range changes {
		t.Run(test.name, func(t *testing.T) {
			changed := stateBodyPackage(t, 1, 7)
			test.mutate(&changed)
			if got := compute(changed); got == baseline {
				t.Fatalf("change preserved body hash %s", got)
			}
		})
	}

	empty := hir.Package{StateLayouts: slices.Clone(base.StateLayouts)}
	emptyHash := compute(empty)
	if emptyHash == baseline || emptyHash == (StateBodyHash{}) {
		t.Fatalf("empty body hash = %s, baseline %s", emptyHash, baseline)
	}
}

func TestHIRStateBodyHashRejectsUnverifiedOrAmbiguousInput(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-04fc4400f5e875c5c22ab2d50765d721df38a12a19ef3508deb432c7f4da92f3")
	base := stateBodyPackage(t, 1, 7)
	tests := []struct {
		name     string
		packages []hir.PackageID
		mutate   func(*hir.Package)
		coords   map[hir.PackageID]packagegraph.CoordinateID
		contains string
	}{
		{name: "missing effect", packages: []hir.PackageID{1}, mutate: func(value *hir.Package) { value.Modules[0].Functions[0].Effects = nil }, coords: map[hir.PackageID]packagegraph.CoordinateID{1: coordinate}, contains: "CSM1005"},
		{name: "duplicate request", packages: []hir.PackageID{1, 1}, coords: map[hir.PackageID]packagegraph.CoordinateID{1: coordinate}, contains: "duplicated"},
		{name: "missing coordinate", packages: []hir.PackageID{1}, coords: map[hir.PackageID]packagegraph.CoordinateID{}, contains: "PackageCoordinateId"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			input.Modules = slices.Clone(base.Modules)
			input.Modules[0].Functions = slices.Clone(base.Modules[0].Functions)
			if test.mutate != nil {
				test.mutate(&input)
			}
			if _, err := ComputeHIRStateBodyHashes(input, test.packages, test.coords); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want %q", err, test.contains)
			}
		})
	}
	for _, text := range []string{"", strings.TrimPrefix(StateBodyHashPrefix+strings.Repeat("0", 64), StateBodyHashPrefix), StateBodyHashPrefix + strings.Repeat("A", 64), StateBodyHashPrefix + strings.Repeat("g", 64)} {
		if _, err := ParseStateBodyHash(text); err == nil {
			t.Fatalf("ParseStateBodyHash(%q) accepted", text)
		}
	}
}

func stateBodyPackage(t *testing.T, packageID hir.PackageID, symbol hir.SymbolID) hir.Package {
	t.Helper()
	return hir.Package{
		Modules: []hir.Module{{Package: packageID, Path: "ledger", Functions: []hir.Function{{
			Symbol: symbol, Name: "apply", ResultType: hir.TypeVoid,
			Parameters: []hir.Parameter{{Type: hir.ParseType(hir.QualifiedModulePath(packageID, "types::Account"))}},
			Effects:    []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite},
		}}}},
		StateLayouts: []hir.StateLayout{{Package: packageID, ContractUID: "example.ledger", Fields: []hir.StateField{{
			Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID(),
		}}}},
		StateTransactions: []hir.StateTransaction{{
			Package: packageID, Function: symbol, Exit: hir.StateTransactionCommit,
			Accesses: []hir.StateAccess{
				{Kind: hir.StateAccessCellWrite, Field: "value", Value: stateBodyU64(t, 10)},
				{Kind: hir.StateAccessCellRead, Field: "value"},
			},
		}},
	}
}

func stateBodyU64(t *testing.T, value uint64) hir.StateValue {
	t.Helper()
	encoded, err := contractstate.U64.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: encoded}
}

func stateBodyCoordinate(t *testing.T, value string) packagegraph.CoordinateID {
	t.Helper()
	result, err := packagegraph.ParseCoordinateID(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

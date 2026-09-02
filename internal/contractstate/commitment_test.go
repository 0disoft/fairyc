package contractstate

import (
	"encoding/hex"
	"errors"
	"testing"
)

func TestCommitmentGoldenVector(t *testing.T) {
	var layoutHash LayoutHash
	for index := range layoutHash {
		layoutHash[index] = byte(index)
	}
	var firstKey Key
	var secondKey Key
	for index := range firstKey {
		firstKey[index] = 0x11
		secondKey[index] = 0x22
	}
	snapshot := commitmentSnapshot{
		layoutHash: layoutHash,
		version:    7,
		entries: []Entry{
			{Key: firstKey, Value: []byte("alpha")},
			{Key: secondKey, Value: []byte{0, 1, 2}},
		},
	}
	commitment := commitmentFromSnapshot(snapshot, buildCommitmentLevels(snapshot))
	const expected = "9c8ce41bbe14cec6875130b9373a2f96230efe8650d1957f1f20cbf160248cb1"
	if hex.EncodeToString(commitment.Root[:]) != expected {
		t.Fatalf("root = %x, want %s", commitment.Root, expected)
	}
}

func TestCommitmentInclusionAndAbsenceProofs(t *testing.T) {
	store, layout, counter, balances := newCommitmentFixture(t)
	executionLimits := Limits{MaxWrites: 8, MaxWriteBytes: 128, MaxValueBytes: 32, MaxMapKeyBytes: 32}
	if _, err := Execute(store, layout, executionLimits, func(transaction *Transaction) error {
		if err := counter.Set(transaction, 42); err != nil {
			return err
		}
		if err := balances.Set(transaction, 7, 70); err != nil {
			return err
		}
		return balances.Set(transaction, 9, 90)
	}); err != nil {
		t.Fatal(err)
	}

	limits := CommitmentLimits{MaxLeaves: 16, MaxDepth: 8, MaxValueBytes: 64}
	commitment, err := BuildCommitment(store, limits)
	if err != nil {
		t.Fatal(err)
	}
	if commitment.Algorithm != CommitmentAlgorithm || commitment.LeafCount != 3 || commitment.StoreVersion != 1 {
		t.Fatalf("unexpected commitment: %+v", commitment)
	}

	for _, entry := range store.Dump() {
		built, proof, err := BuildInclusionProof(store, entry.Key, limits)
		if err != nil {
			t.Fatal(err)
		}
		if built != commitment {
			t.Fatalf("proof commitment changed: %+v != %+v", built, commitment)
		}
		if err := VerifyInclusion(commitment, proof, limits); err != nil {
			t.Fatalf("verify inclusion %s: %v", entry.Key, err)
		}
	}

	query := Key{}
	if _, _, err := BuildInclusionProof(store, query, limits); !errors.Is(err, ErrCommitmentMissing) {
		t.Fatalf("missing inclusion error = %v", err)
	}
	built, absence, err := BuildAbsenceProof(store, query, limits)
	if err != nil {
		t.Fatal(err)
	}
	if built != commitment {
		t.Fatalf("absence commitment changed: %+v != %+v", built, commitment)
	}
	if err := VerifyAbsence(commitment, absence, limits); err != nil {
		t.Fatalf("verify absence: %v", err)
	}

	present := store.Dump()[0].Key
	if _, _, err := BuildAbsenceProof(store, present, limits); !errors.Is(err, ErrCommitmentPresent) {
		t.Fatalf("present absence error = %v", err)
	}
}

func TestCommitmentDeletionBecomesProvableAbsence(t *testing.T) {
	store, layout, counter, _ := newCommitmentFixture(t)
	executionLimits := Limits{MaxWrites: 4, MaxWriteBytes: 64, MaxValueBytes: 32, MaxMapKeyBytes: 32}
	if _, err := Execute(store, layout, executionLimits, func(transaction *Transaction) error {
		return counter.Set(transaction, 5)
	}); err != nil {
		t.Fatal(err)
	}
	limits := CommitmentLimits{MaxLeaves: 8, MaxDepth: 4, MaxValueBytes: 32}
	before, proof, err := BuildInclusionProof(store, counter.Field().Prefix, limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyInclusion(before, proof, limits); err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(store, layout, executionLimits, func(transaction *Transaction) error {
		return counter.Delete(transaction)
	}); err != nil {
		t.Fatal(err)
	}
	after, absence, err := BuildAbsenceProof(store, counter.Field().Prefix, limits)
	if err != nil {
		t.Fatal(err)
	}
	if before.Root == after.Root || before.StoreVersion == after.StoreVersion {
		t.Fatalf("deletion did not change commitment identity")
	}
	if after.LeafCount != 0 {
		t.Fatalf("leaf count after deletion = %d", after.LeafCount)
	}
	if err := VerifyAbsence(after, absence, limits); err != nil {
		t.Fatal(err)
	}
}

func TestCommitmentRejectsMutatedProofs(t *testing.T) {
	store, layout, counter, balances := newCommitmentFixture(t)
	executionLimits := Limits{MaxWrites: 8, MaxWriteBytes: 128, MaxValueBytes: 32, MaxMapKeyBytes: 32}
	if _, err := Execute(store, layout, executionLimits, func(transaction *Transaction) error {
		if err := counter.Set(transaction, 1); err != nil {
			return err
		}
		return balances.Set(transaction, 2, 3)
	}); err != nil {
		t.Fatal(err)
	}
	limits := CommitmentLimits{MaxLeaves: 8, MaxDepth: 4, MaxValueBytes: 32}
	entry := store.Dump()[0]
	commitment, proof, err := BuildInclusionProof(store, entry.Key, limits)
	if err != nil {
		t.Fatal(err)
	}

	mutatedValue := proof
	mutatedValue.Value = append([]byte(nil), proof.Value...)
	mutatedValue.Value[0] ^= 0xff
	if err := VerifyInclusion(commitment, mutatedValue, limits); !errors.Is(err, ErrCommitmentProof) {
		t.Fatalf("mutated value error = %v", err)
	}

	mutatedAlgorithm := proof
	mutatedAlgorithm.Algorithm = "fairyc.contract-state.merkle/v2"
	if err := VerifyInclusion(commitment, mutatedAlgorithm, limits); !errors.Is(err, ErrCommitmentProof) {
		t.Fatalf("mutated algorithm error = %v", err)
	}

	mutatedPath := proof
	mutatedPath.Siblings = append([]CommitmentHash(nil), proof.Siblings...)
	mutatedPath.Siblings[0][0] ^= 1
	if err := VerifyInclusion(commitment, mutatedPath, limits); !errors.Is(err, ErrCommitmentProof) {
		t.Fatalf("mutated path error = %v", err)
	}

	query := Key{}
	_, absence, err := BuildAbsenceProof(store, query, limits)
	if err != nil {
		t.Fatal(err)
	}
	if absence.After == nil {
		t.Fatal("expected first-leaf successor")
	}
	mutatedAbsence := absence
	copyAfter := *absence.After
	copyAfter.Index++
	mutatedAbsence.After = &copyAfter
	if err := VerifyAbsence(commitment, mutatedAbsence, limits); !errors.Is(err, ErrCommitmentProof) {
		t.Fatalf("mutated absence error = %v", err)
	}
}

func TestCommitmentLimitsFailClosed(t *testing.T) {
	store, layout, counter, balances := newCommitmentFixture(t)
	executionLimits := Limits{MaxWrites: 8, MaxWriteBytes: 128, MaxValueBytes: 32, MaxMapKeyBytes: 32}
	if _, err := Execute(store, layout, executionLimits, func(transaction *Transaction) error {
		if err := counter.Set(transaction, 1); err != nil {
			return err
		}
		return balances.Set(transaction, 2, 3)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildCommitment(store, CommitmentLimits{MaxLeaves: 1, MaxDepth: 1, MaxValueBytes: 32}); !errors.Is(err, ErrCommitmentLimit) {
		t.Fatalf("leaf-limit error = %v", err)
	}
	if _, err := BuildCommitment(store, CommitmentLimits{MaxLeaves: 8, MaxDepth: 4, MaxValueBytes: 2}); !errors.Is(err, ErrCommitmentLimit) {
		t.Fatalf("value-limit error = %v", err)
	}
}

func TestCommitmentLayoutAndVersionAreBound(t *testing.T) {
	store, layout, _, _ := newCommitmentFixture(t)
	limits := CommitmentLimits{MaxLeaves: 8, MaxDepth: 4, MaxValueBytes: 32}
	before, err := BuildCommitment(store, limits)
	if err != nil {
		t.Fatal(err)
	}
	builder, err := NewLayoutBuilder(layout.ContractUID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "counter", U32); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareMap(builder, "balances", U32, U32); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "enabled", Bool); err != nil {
		t.Fatal(err)
	}
	next, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upgrade(next); err != nil {
		t.Fatal(err)
	}
	after, err := BuildCommitment(store, limits)
	if err != nil {
		t.Fatal(err)
	}
	if before.LayoutHash == after.LayoutHash || before.StoreVersion == after.StoreVersion || before.Root == after.Root {
		t.Fatalf("layout upgrade was not bound: before=%+v after=%+v", before, after)
	}
}

func FuzzVerifyInclusionRejectsChangedValue(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, changed []byte) {
		if len(changed) > 32 {
			t.Skip()
		}
		store, layout, counter, _ := newCommitmentFixture(t)
		if _, err := Execute(store, layout, Limits{MaxWrites: 1, MaxWriteBytes: 64, MaxValueBytes: 32, MaxMapKeyBytes: 32}, func(transaction *Transaction) error {
			return counter.Set(transaction, 7)
		}); err != nil {
			t.Fatal(err)
		}
		limits := CommitmentLimits{MaxLeaves: 4, MaxDepth: 2, MaxValueBytes: 32}
		commitment, proof, err := BuildInclusionProof(store, counter.Field().Prefix, limits)
		if err != nil {
			t.Fatal(err)
		}
		if string(changed) == string(proof.Value) {
			t.Skip()
		}
		proof.Value = append([]byte(nil), changed...)
		if err := VerifyInclusion(commitment, proof, limits); !errors.Is(err, ErrCommitmentProof) {
			t.Fatalf("changed value error = %v", err)
		}
	})
}

func newCommitmentFixture(t *testing.T) (*MemoryStore, Layout, Cell[uint32], Map[uint32, uint32]) {
	t.Helper()
	builder, err := NewLayoutBuilder("tests.commitment")
	if err != nil {
		t.Fatal(err)
	}
	counter, err := DeclareCell(builder, "counter", U32)
	if err != nil {
		t.Fatal(err)
	}
	balances, err := DeclareMap(builder, "balances", U32, U32)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	return store, layout, counter, balances
}

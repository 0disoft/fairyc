package contractstate

import (
	"errors"
	"testing"
)

func TestCommitmentEmptyAbsenceRejectsWrongRoot(t *testing.T) {
	builder, err := NewLayoutBuilder("tests.commitment.empty")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "counter", U32); err != nil {
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
	limits := CommitmentLimits{MaxLeaves: 8, MaxDepth: 4, MaxValueBytes: 32}
	commitment, proof, err := BuildAbsenceProof(store, Key{}, limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyAbsence(commitment, proof, limits); err != nil {
		t.Fatal(err)
	}
	commitment.Root[0] ^= 1
	if err := VerifyAbsence(commitment, proof, limits); !errors.Is(err, ErrCommitmentProof) {
		t.Fatalf("wrong empty root error = %v", err)
	}
}

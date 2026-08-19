package chainprimitives

import (
	"errors"
	"testing"
)

func TestCostScheduleIsCanonicalAndBounded(t *testing.T) {
	t.Parallel()

	rules := []CostRule{
		{ID: IntrinsicEd25519Verify, Base: 900, PerChunk: 0, ChunkBytes: 1, MaxInputBytes: 128},
		{ID: IntrinsicSHA256, Base: 60, PerChunk: 12, ChunkBytes: 64, MaxInputBytes: 1 << 20},
	}
	first, err := NewCostSchedule("wasm32-contract-v0", 1, rules)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCostSchedule("wasm32-contract-v0", 1, []CostRule{rules[1], rules[0]})
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() {
		t.Fatal("cost schedule digest depends on input order")
	}
	if got, err := first.Cost(IntrinsicSHA256, 65); err != nil || got != 84 {
		t.Fatalf("SHA-256 cost = %d, %v; want 84", got, err)
	}
	if _, err := first.Cost(IntrinsicSHA256, (1<<20)+1); !errors.Is(err, ErrInvalidCostSchedule) {
		t.Fatalf("oversize input error = %v", err)
	}
	if _, err := first.Cost(IntrinsicKeccak256, 32); !errors.Is(err, ErrUnknownIntrinsic) {
		t.Fatalf("missing rule error = %v", err)
	}

	if _, err := NewCostSchedule("wasm32-contract-v0", 1, []CostRule{rules[0], rules[0]}); !errors.Is(err, ErrInvalidCostSchedule) {
		t.Fatalf("duplicate schedule error = %v", err)
	}
	if _, err := NewCostSchedule("wasm32-contract-v0", 1, []CostRule{{ID: IntrinsicSHA256, ChunkBytes: 64, MaxInputBytes: 1024}}); !errors.Is(err, ErrInvalidCostSchedule) {
		t.Fatalf("zero-cost schedule error = %v", err)
	}
}

func TestReferenceHashFailsClosed(t *testing.T) {
	t.Parallel()

	got, err := ExecuteReferenceHash(IntrinsicSHA256, []byte("fairy"))
	if err != nil {
		t.Fatal(err)
	}
	const want = "0x5a81e6aa7556585ba7c569881d1103683adc9e0124ff7952df423afba2f167b5"
	if got.String() != want {
		t.Fatalf("SHA-256 = %s, want %s", got, want)
	}
	if _, err := ExecuteReferenceHash(IntrinsicKeccak256, nil); !errors.Is(err, ErrUnsupportedReference) {
		t.Fatalf("unsupported reference error = %v", err)
	}
	if _, err := ExecuteReferenceHash(IntrinsicID("crypto.unknown"), nil); !errors.Is(err, ErrUnknownIntrinsic) {
		t.Fatalf("unknown intrinsic error = %v", err)
	}
}

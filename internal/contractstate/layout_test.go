package contractstate

import (
	"errors"
	"testing"
)

func TestLayoutHashIsDeterministicAndAppendCompatible(t *testing.T) {
	t.Parallel()
	first, _, _ := testBankLayout(t, false)
	second, _, _ := testBankLayout(t, false)
	if first.Hash() != second.Hash() {
		t.Fatalf("equal layouts produced %s and %s", first.Hash(), second.Hash())
	}
	if got := first.Fields(); len(got) != 2 || got[0].Prefix == got[1].Prefix {
		t.Fatalf("Fields() = %+v", got)
	}

	extended, _, _ := testBankLayout(t, true)
	if err := extended.Extends(first); err != nil {
		t.Fatalf("append-only extension rejected: %v", err)
	}
	if extended.Hash() == first.Hash() {
		t.Fatal("appended field did not change layout hash")
	}
	for index, field := range first.Fields() {
		if extended.Fields()[index] != field {
			t.Fatalf("existing field %d changed across append: %+v != %+v", index, extended.Fields()[index], field)
		}
	}

	reorderedBuilder, err := NewLayoutBuilder("example.bank")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareMap(reorderedBuilder, "balances", Byte20, U64); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(reorderedBuilder, "total_supply", U64); err != nil {
		t.Fatal(err)
	}
	reordered, err := reorderedBuilder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := reordered.Extends(first); !errors.Is(err, ErrIncompatibleLayout) {
		t.Fatalf("reordered Extends() error = %v", err)
	}
}

func TestLayoutRejectsDuplicateAndUnstableCodecIdentity(t *testing.T) {
	t.Parallel()
	builder, err := NewLayoutBuilder("example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "value", U64); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(builder, "value", U32); err == nil {
		t.Fatal("duplicate field accepted")
	}
	invalidCodec := FuncCodec[uint64]{ID: "bad id", EncodeFunc: U64.Encode, DecodeFunc: U64.Decode}
	if _, err := DeclareCell(builder, "other", invalidCodec); err == nil {
		t.Fatal("invalid codec type ID accepted")
	}
	if _, err := builder.Build(); err != nil {
		t.Fatal(err)
	}
	if _, err := builder.Build(); err == nil {
		t.Fatal("sealed layout builder built twice")
	}
}

func TestMemoryStoreUpgradePreservesStateAndRejectsBreakingLayout(t *testing.T) {
	t.Parallel()
	base, supply, _ := testBankLayout(t, false)
	store, err := NewMemoryStore(base)
	if err != nil {
		t.Fatal(err)
	}
	limits := testLimits()
	if _, err := Execute(store, base, limits, func(transaction *Transaction) error {
		return supply.Set(transaction, 42)
	}); err != nil {
		t.Fatal(err)
	}

	extended, extendedSupply, _ := testBankLayout(t, true)
	if err := store.Upgrade(extended); err != nil {
		t.Fatalf("Upgrade(append) error = %v", err)
	}
	if err := View(store, extended, limits, func(transaction *Transaction) error {
		value, found, err := extendedSupply.Get(transaction)
		if err != nil {
			return err
		}
		if !found || value != 42 {
			t.Fatalf("supply after upgrade = %d, %v", value, found)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	brokenBuilder, err := NewLayoutBuilder("example.bank")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareCell(brokenBuilder, "total_supply", U32); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareMap(brokenBuilder, "balances", Byte20, U64); err != nil {
		t.Fatal(err)
	}
	broken, err := brokenBuilder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Upgrade(broken); !errors.Is(err, ErrIncompatibleLayout) {
		t.Fatalf("Upgrade(breaking) error = %v", err)
	}
}

func testBankLayout(t *testing.T, withPaused bool) (Layout, Cell[uint64], Map[Bytes20, uint64]) {
	t.Helper()
	builder, err := NewLayoutBuilder("example.bank")
	if err != nil {
		t.Fatal(err)
	}
	supply, err := DeclareCell(builder, "total_supply", U64)
	if err != nil {
		t.Fatal(err)
	}
	balances, err := DeclareMap(builder, "balances", Byte20, U64)
	if err != nil {
		t.Fatal(err)
	}
	if withPaused {
		if _, err := DeclareCell(builder, "paused", Bool); err != nil {
			t.Fatal(err)
		}
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	return layout, supply, balances
}

func testLimits() Limits {
	return Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 1024, MaxMapKeyBytes: 128}
}

package contractstatemir

import (
	"errors"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
)

func TestBuildLayoutMatchesTypedKernelIdentity(t *testing.T) {
	layout := testLayout(t)

	builder, err := contractstate.NewLayoutBuilder("example.token")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contractstate.DeclareCell(builder, "total_supply", contractstate.U64); err != nil {
		t.Fatal(err)
	}
	if _, err := contractstate.DeclareMap(builder, "balances", contractstate.Byte20, contractstate.U64); err != nil {
		t.Fatal(err)
	}
	if _, err := contractstate.DeclareCell(builder, "paused", contractstate.Bool); err != nil {
		t.Fatal(err)
	}
	typed, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if layout.Hash != typed.Hash() {
		t.Fatalf("MIR layout hash %s differs from typed kernel hash %s", layout.Hash, typed.Hash())
	}
	for index, field := range typed.Fields() {
		if actual := layout.Fields[index]; actual != layoutField(field) {
			t.Fatalf("field %d = %+v, want %+v", index, actual, layoutField(field))
		}
	}
}

func TestVerifyRejectsLayoutOperationAndExitDrift(t *testing.T) {
	layout := testLayout(t)
	badLayout := layout
	badLayout.Fields = slices.Clone(layout.Fields)
	badLayout.Fields[0].Prefix = contractstate.Key{}
	if failures := Verify(Program{Layout: badLayout, Exit: ExitCommit}); len(failures) != 1 || failures[0].Code != "CSM1001" {
		t.Fatalf("layout failures = %+v", failures)
	}

	noncanonical := Program{
		Layout: layout,
		Operations: []Operation{{
			Kind: OperationCellWrite, FieldIndex: 2,
			Value: Value{TypeID: contractstate.Bool.TypeID(), Encoded: []byte{2}},
		}},
		Exit: ExitCommit,
	}
	if failures := Verify(noncanonical); len(failures) != 1 || failures[0].Code != "CSM1002" {
		t.Fatalf("operation failures = %+v", failures)
	}

	badExit := Program{Layout: layout, Exit: ExitTrap, TrapCategory: "Host_Failure"}
	if failures := Verify(badExit); len(failures) != 1 || failures[0].Code != "CSM1003" {
		t.Fatalf("exit failures = %+v", failures)
	}
}

func TestExecuteCommitsReadYourWritesAndDelete(t *testing.T) {
	layout := testLayout(t)
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	account := encode(t, contractstate.Byte20, contractstate.Bytes20{19: 7})
	ten := encode(t, contractstate.U64, uint64(10))
	seven := encode(t, contractstate.U64, uint64(7))
	program := Program{
		Layout: layout,
		Operations: []Operation{
			{Kind: OperationCellWrite, FieldIndex: 0, Value: ten},
			{Kind: OperationCellRead, FieldIndex: 0},
			{Kind: OperationMapWrite, FieldIndex: 1, Key: account, Value: seven},
			{Kind: OperationMapRead, FieldIndex: 1, Key: account},
			{Kind: OperationCellDelete, FieldIndex: 0},
			{Kind: OperationCellRead, FieldIndex: 0},
		},
		Exit: ExitCommit,
	}
	outcome, err := Execute(store, program, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != ExitCommit || outcome.Receipt.LayoutHash != layout.Hash || outcome.Receipt.Version != 1 || outcome.Receipt.Writes != 2 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if len(outcome.Reads) != 3 {
		t.Fatalf("reads = %+v", outcome.Reads)
	}
	assertRead(t, outcome.Reads[0], 1, true, ten)
	assertRead(t, outcome.Reads[1], 3, true, seven)
	assertRead(t, outcome.Reads[2], 5, false, Value{TypeID: contractstate.U64.TypeID()})
	if dump := store.Dump(); len(dump) != 1 {
		t.Fatalf("committed dump = %+v", dump)
	}
}

func TestExecuteRollsBackRejectTrapAndLimit(t *testing.T) {
	layout := testLayout(t)
	write := Operation{Kind: OperationCellWrite, FieldIndex: 0, Value: encode(t, contractstate.U64, uint64(10))}

	t.Run("reject", func(t *testing.T) {
		store, err := NewMemoryStore(layout)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := Execute(store, Program{Layout: layout, Operations: []Operation{write}, Exit: ExitReject}, testLimits())
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Exit != ExitReject || outcome.Receipt != (contractstate.CommitReceipt{}) {
			t.Fatalf("outcome = %+v", outcome)
		}
		assertUnchanged(t, store)
	})

	t.Run("trap", func(t *testing.T) {
		store, err := NewMemoryStore(layout)
		if err != nil {
			t.Fatal(err)
		}
		recovered := executeAndRecover(store, Program{
			Layout: layout, Operations: []Operation{write}, Exit: ExitTrap, TrapCategory: "assertion-failed",
		}, testLimits())
		trap, ok := recovered.(Trap)
		if !ok || trap.Category != "assertion-failed" {
			t.Fatalf("recovered = %#v", recovered)
		}
		assertUnchanged(t, store)
	})

	t.Run("limit", func(t *testing.T) {
		store, err := NewMemoryStore(layout)
		if err != nil {
			t.Fatal(err)
		}
		program := Program{
			Layout: layout,
			Operations: []Operation{
				write,
				{Kind: OperationCellWrite, FieldIndex: 2, Value: encode(t, contractstate.Bool, true)},
			},
			Exit: ExitCommit,
		}
		limits := testLimits()
		limits.MaxWrites = 1
		if _, err := Execute(store, program, limits); !errors.Is(err, contractstate.ErrWriteLimit) {
			t.Fatalf("Execute error = %v, want ErrWriteLimit", err)
		}
		assertUnchanged(t, store)
	})
}

func testLayout(t *testing.T) Layout {
	t.Helper()
	layout, err := BuildLayout("example.token", []Declaration{
		{Name: "total_supply", Kind: contractstate.FieldCell, ValueTypeID: contractstate.U64.TypeID()},
		{Name: "balances", Kind: contractstate.FieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.U64.TypeID()},
		{Name: "paused", Kind: contractstate.FieldCell, ValueTypeID: contractstate.Bool.TypeID()},
	})
	if err != nil {
		t.Fatal(err)
	}
	return layout
}

func testLimits() contractstate.Limits {
	return contractstate.Limits{MaxWrites: 8, MaxWriteBytes: 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64}
}

func encode[T any](t *testing.T, codec contractstate.Codec[T], value T) Value {
	t.Helper()
	encoded, err := codec.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return Value{TypeID: codec.TypeID(), Encoded: encoded}
}

func assertRead(t *testing.T, actual Read, operation int, found bool, expected Value) {
	t.Helper()
	if actual.Operation != operation || actual.Found != found || actual.Value.TypeID != expected.TypeID || !slices.Equal(actual.Value.Encoded, expected.Encoded) {
		t.Fatalf("read = %+v, want operation=%d found=%t value=%+v", actual, operation, found, expected)
	}
}

func assertUnchanged(t *testing.T, store *contractstate.MemoryStore) {
	t.Helper()
	if store.Version() != 0 || len(store.Dump()) != 0 {
		t.Fatalf("store version=%d dump=%+v", store.Version(), store.Dump())
	}
}

func executeAndRecover(store *contractstate.MemoryStore, program Program, limits contractstate.Limits) (recovered any) {
	defer func() { recovered = recover() }()
	_, _ = Execute(store, program, limits)
	return nil
}

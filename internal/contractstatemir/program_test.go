package contractstatemir

import (
	"errors"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
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
	if failures := Verify(testProgram(badLayout, nil, ExitCommit)); len(failures) != 1 || failures[0].Code != "CSM1001" {
		t.Fatalf("layout failures = %+v", failures)
	}

	noncanonical := testProgram(layout, []Operation{{
		Kind: OperationCellWrite, FieldIndex: 2,
		Value: Value{TypeID: contractstate.Bool.TypeID(), Encoded: []byte{2}},
	}}, ExitCommit)
	if failures := Verify(noncanonical); len(failures) != 1 || failures[0].Code != "CSM1002" {
		t.Fatalf("operation failures = %+v", failures)
	}

	badExit := testProgram(layout, nil, ExitTrap)
	badExit.TrapCategory = "Host_Failure"
	if failures := Verify(badExit); len(failures) != 1 || failures[0].Code != "CSM1003" {
		t.Fatalf("exit failures = %+v", failures)
	}
}

func TestVerifyRejectsCapabilityAndEffectDrift(t *testing.T) {
	layout := testLayout(t)

	missingCapability := testProgram(layout, nil, ExitCommit)
	missingCapability.Capability = TransactionCapability{}
	if failures := Verify(missingCapability); !hasFailureCode(failures, "CSM1004") {
		t.Fatalf("missing capability failures = %+v", failures)
	}

	mismatched := testProgram(layout, []Operation{{Kind: OperationCellRead, FieldIndex: 0}}, ExitCommit)
	mismatched.Operations[0].Capability = newTransactionCapability()
	if failures := Verify(mismatched); !hasFailureCode(failures, "CSM1004") {
		t.Fatalf("capability failures = %+v", failures)
	}
	missingOperationCapability := testProgram(layout, []Operation{{Kind: OperationCellRead, FieldIndex: 0}}, ExitCommit)
	missingOperationCapability.Operations[0].Capability = TransactionCapability{}
	if failures := Verify(missingOperationCapability); !hasFailureCode(failures, "CSM1004") {
		t.Fatalf("missing operation capability failures = %+v", failures)
	}

	missing := testProgram(layout, []Operation{{Kind: OperationCellWrite, FieldIndex: 0, Value: encode(t, contractstate.U64, uint64(1))}}, ExitCommit)
	missing.Effects = nil
	if failures := Verify(missing); !hasFailureCode(failures, "CSM1005") {
		t.Fatalf("missing effect failures = %+v", failures)
	}

	noncanonical := testProgram(layout, []Operation{{Kind: OperationCellRead, FieldIndex: 0}}, ExitCommit)
	noncanonical.Effects = []hir.Effect{hir.EffectStorageWrite, hir.EffectStorageRead}
	if failures := Verify(noncanonical); !hasFailureCode(failures, "CSM1005") {
		t.Fatalf("noncanonical effect failures = %+v", failures)
	}
	duplicate := testProgram(layout, []Operation{{Kind: OperationCellRead, FieldIndex: 0}}, ExitCommit)
	duplicate.Effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageRead}
	if failures := Verify(duplicate); !hasFailureCode(failures, "CSM1005") {
		t.Fatalf("duplicate effect failures = %+v", failures)
	}

	hostEffect := testProgram(layout, nil, ExitCommit)
	hostEffect.Effects = []hir.Effect{hir.EffectFilesystem}
	if failures := Verify(hostEffect); !hasFailureCode(failures, "CSM1005") {
		t.Fatalf("host effect failures = %+v", failures)
	}
	unknownEffect := testProgram(layout, nil, ExitCommit)
	unknownEffect.Effects = []hir.Effect{"future_chain"}
	if failures := Verify(unknownEffect); !hasFailureCode(failures, "CSM1005") {
		t.Fatalf("unknown effect failures = %+v", failures)
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
	program := testProgram(layout, []Operation{
		{Kind: OperationCellWrite, FieldIndex: 0, Value: ten},
		{Kind: OperationCellRead, FieldIndex: 0},
		{Kind: OperationMapWrite, FieldIndex: 1, Key: account, Value: seven},
		{Kind: OperationMapRead, FieldIndex: 1, Key: account},
		{Kind: OperationCellDelete, FieldIndex: 0},
		{Kind: OperationCellRead, FieldIndex: 0},
	}, ExitCommit)
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
		outcome, err := Execute(store, testProgram(layout, []Operation{write}, ExitReject), testLimits())
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
		program := testProgram(layout, []Operation{write}, ExitTrap)
		program.TrapCategory = "assertion-failed"
		recovered := executeAndRecover(store, program, testLimits())
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
		program := testProgram(layout, []Operation{
			write,
			{Kind: OperationCellWrite, FieldIndex: 2, Value: encode(t, contractstate.Bool, true)},
		}, ExitCommit)
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

func testProgram(layout Layout, operations []Operation, exit ExitKind) Program {
	capability := newTransactionCapability()
	operations = slices.Clone(operations)
	var effects []hir.Effect
	for index := range operations {
		operations[index].Capability = capability
		effects = append(effects, requiredStorageEffects(operations[index].Kind)...)
	}
	return Program{
		Layout: layout, Capability: capability, Effects: hir.CanonicalEffects(effects), Operations: operations, Exit: exit,
	}
}

func hasFailureCode(failures []VerificationError, code string) bool {
	return slices.ContainsFunc(failures, func(failure VerificationError) bool { return failure.Code == code })
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

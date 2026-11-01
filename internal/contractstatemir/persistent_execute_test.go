package contractstatemir

import (
	"errors"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
)

type observedPersistence struct {
	contractstate.PersistencePort
	snapshots, publications int
	beforePublish           func()
}

func (port *observedPersistence) Snapshot(hash contractstate.LayoutHash) (contractstate.PersistentSnapshot, error) {
	port.snapshots++
	return port.PersistencePort.Snapshot(hash)
}

func (port *observedPersistence) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.publications++
	if port.beforePublish != nil {
		port.beforePublish()
	}
	return port.PersistencePort.CompareAndPublish(hash, version, writes)
}

func TestPersistentExecutionLifecycle(t *testing.T) {
	for _, exit := range []ExitKind{ExitCommit, ExitReject, ExitTrap} {
		t.Run(string(exit), func(t *testing.T) {
			layout := testLayout(t)
			bound, err := bindLayout(layout)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := contractstate.NewReferencePersistence(bound.layout, nil)
			if err != nil {
				t.Fatal(err)
			}
			port := &observedPersistence{PersistencePort: reference}
			program := testProgram(layout, []Operation{{Kind: OperationCellWrite, FieldIndex: 0, Value: encode(t, contractstate.U64, uint64(42))}}, exit)
			if exit == ExitTrap {
				program.TrapCategory = "integer-overflow"
			}
			var outcome Outcome
			var executionErr error
			var trapped any
			func() {
				defer func() { trapped = recover() }()
				outcome, executionErr = ExecutePersistent(port, program, testLimits())
			}()
			if executionErr != nil {
				t.Fatal(executionErr)
			}
			if exit == ExitTrap {
				if trap, ok := trapped.(Trap); !ok || trap.Category != "integer-overflow" {
					t.Fatalf("trap = %#v", trapped)
				}
			} else if trapped != nil || outcome.Exit != exit {
				t.Fatalf("outcome/trap = %+v/%v", outcome, trapped)
			}
			snapshot, err := reference.Snapshot(layout.Hash)
			if err != nil {
				t.Fatal(err)
			}
			wantPublications := 0
			if exit == ExitCommit {
				wantPublications = 1
				if snapshot.Version != 1 || len(snapshot.Entries) != 1 || outcome.Receipt.WriteBytes != 41 {
					t.Fatalf("commit = %+v/%+v", snapshot, outcome)
				}
			} else if snapshot.Version != 0 || len(snapshot.Entries) != 0 {
				t.Fatalf("failed exit published: %+v", snapshot)
			}
			if port.snapshots != 1 || port.publications != wantPublications {
				t.Fatalf("calls = %d/%d", port.snapshots, port.publications)
			}
		})
	}
}

func TestPersistentExecutionConflictAndVerification(t *testing.T) {
	layout := testLayout(t)
	bound, err := bindLayout(layout)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := contractstate.NewReferencePersistence(bound.layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	port := &observedPersistence{PersistencePort: reference}
	program := testProgram(layout, []Operation{{Kind: OperationCellWrite, FieldIndex: 0, Value: encode(t, contractstate.U64, uint64(42))}}, ExitCommit)
	port.beforePublish = func() {
		if _, err := reference.CompareAndPublish(layout.Hash, 0, []contractstate.PersistentWrite{{Key: layout.Fields[0].Prefix, Value: []byte("winner")}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ExecutePersistent(port, program, testLimits()); !errors.Is(err, contractstate.ErrConflict) {
		t.Fatalf("conflict = %v", err)
	}
	if port.snapshots != 1 || port.publications != 1 {
		t.Fatalf("unexpected retry: %+v", port)
	}
	before, _ := reference.Snapshot(layout.Hash)
	program.Capability = TransactionCapability{}
	if _, err := ExecutePersistent(port, program, testLimits()); err == nil {
		t.Fatal("invalid MIR executed")
	}
	if port.snapshots != 1 || port.publications != 1 {
		t.Fatal("verifier failure touched persistence")
	}
	after, _ := reference.Snapshot(layout.Hash)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed attempts changed winner")
	}
}

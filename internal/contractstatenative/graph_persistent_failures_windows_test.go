package contractstatenative

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

func TestGraphPersistenceNativeFailureAndCancellation(t *testing.T) {
	h, layout := overlayMIRFixture(t, contractstatemir.ExitCommit, false)
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildGraph(p, profile)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			store, err := contractstate.CreateLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			before := persistentStateBytes(t, directory)
			for _, policy := range []struct {
				name   string
				limits contractstate.Limits
				cause  error
			}{
				{"count", contractstate.Limits{MaxWrites: 13, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64}, contractstate.ErrWriteLimit},
				{"value", contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 31, MaxMapKeyBytes: 64}, contractstate.ErrValueTooLarge},
				{"bytes", contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 32, MaxValueBytes: 64, MaxMapKeyBytes: 64}, contractstate.ErrWriteBytesLimit},
				{"key", contractstate.Limits{MaxWrites: 16, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 31}, contractstate.ErrMapKeyTooLarge},
			} {
				t.Run(policy.name, func(t *testing.T) {
					port := &overlayObservedPort{PersistencePort: store}
					if _, err := ExecuteGraphPersistent(context.Background(), port, executable, policy.limits); !errors.Is(err, policy.cause) || port.snapshots != 1 || port.publishes != 0 {
						t.Fatal("native graph policy failure published", err, port.snapshots, port.publishes)
					}
				})
			}
			cause := errors.New("graph persistence backend refused publication")
			failed := &failingNativePort{PersistencePort: store, cause: cause}
			if _, err := ExecuteGraphPersistent(context.Background(), failed, executable, overlayPersistentLimits()); !errors.Is(err, cause) || failed.publishes != 1 {
				t.Fatal("graph publication error retried or lost cause", err, failed.publishes)
			}
			localLimits := overlayLocalLimits()
			localLimits.MaxValueBytes = 31
			limited, err := contractstate.OpenLocalPersistence(directory, layout, localLimits)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ExecuteGraphPersistent(context.Background(), limited, executable, overlayPersistentLimits()); !errors.Is(err, contractstate.ErrValueTooLarge) {
				t.Fatal("native graph bypassed real local capacity", err)
			}
			if err := limited.Close(); err != nil {
				t.Fatal(err)
			}
			broken := executable
			broken.image.Bytes = []byte("not a native graph image")
			if _, err := ExecuteGraphPersistent(context.Background(), store, broken, overlayPersistentLimits()); err == nil {
				t.Fatal("invalid graph image published")
			}
			ctx, cancel := context.WithCancel(context.Background())
			port := &overlayObservedPort{PersistencePort: store, afterSnapshot: cancel}
			if _, err := ExecuteGraphPersistent(ctx, port, executable, overlayPersistentLimits()); !errors.Is(err, context.Canceled) || port.snapshots != 1 || port.publishes != 0 {
				t.Fatal("snapshot cancellation reached graph CAS", err)
			}
			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			port = &overlayObservedPort{PersistencePort: store}
			calls := 0
			_, err = executeGraphPersistent(ctx, port, p, overlayPersistentLimits(), func(ctx context.Context, input GraphInput) (cdriver.WindowsHostOutcome, error) {
				calls++
				out, err := executable.run(ctx, input)
				cancel()
				return out, err
			})
			cancel()
			if !errors.Is(err, context.Canceled) || calls != 1 || port.snapshots != 1 || port.publishes != 0 {
				t.Fatal("completed C execution ignored pre-publication cancellation", err, calls, port.publishes)
			}
			if !bytes.Equal(before, persistentStateBytes(t, directory)) {
				t.Fatal("failed/canceled graph changed durable bytes")
			}
			ctx, cancelPublished := context.WithCancel(context.Background())
			defer cancelPublished()
			published := &overlayObservedPort{PersistencePort: store, afterPublish: cancelPublished}
			out, err := ExecuteGraphPersistent(ctx, published, executable, overlayPersistentLimits())
			if err != nil || !errors.Is(ctx.Err(), context.Canceled) || out.Receipt.Version != 1 || out.Receipt.Writes != 14 || published.snapshots != 1 || published.publishes != 1 {
				t.Fatal("acknowledged graph became cancellation", out, err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			actual, err := reopened.Snapshot(layout.Hash())
			if err != nil {
				t.Fatal(err)
			}
			reference, err := contractstate.NewReferencePersistence(layout, nil)
			if err != nil {
				t.Fatal(err)
			}
			want, trap, err := persistentOracle(reference, h.program, overlayPersistentLimits())
			if err != nil || trap != "" || !sameOverlayOutcome(out, want) {
				t.Fatal("acknowledged native graph differs from Go", out, want, err)
			}
			expected, err := reference.Snapshot(layout.Hash())
			if err != nil || !sameOverlaySnapshot(actual, expected) {
				t.Fatal("acknowledged graph did not survive reopen", actual, expected, err)
			}
		})
	}
}

type graphBeforeCASPort struct {
	contractstate.PersistencePort
	beforeCAS            func()
	snapshots, publishes int
}

func (port *graphBeforeCASPort) Snapshot(hash contractstate.LayoutHash) (contractstate.PersistentSnapshot, error) {
	port.snapshots++
	return port.PersistencePort.Snapshot(hash)
}

func (port *graphBeforeCASPort) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.publishes++
	port.beforeCAS()
	return port.PersistencePort.CompareAndPublish(hash, version, writes)
}

func TestGraphPersistenceNativeReadOnly(t *testing.T) {
	h, layout := overlayReadFixture(t)
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	value := bytes.Repeat([]byte{0, 255, 26, 13}, 16385)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildGraph(p, profile)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			store, err := contractstate.CreateLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			if _, err := store.CompareAndPublish(layout.Hash(), 0, []contractstate.PersistentWrite{{Key: layout.Fields()[0].Prefix, Value: value}}); err != nil {
				t.Fatal(err)
			}
			policy := contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 1, MaxValueBytes: 1, MaxMapKeyBytes: 1}
			before := persistentStateBytes(t, directory)
			for range 2 {
				port := &overlayObservedPort{PersistencePort: store}
				out, err := ExecuteGraphPersistent(context.Background(), port, executable, policy)
				if err != nil || out.Receipt.Version != 1 || out.Receipt.Writes != 0 || out.Receipt.WriteBytes != 0 || len(out.Reads) != 1 || !out.Reads[0].Found || !bytes.Equal(out.Reads[0].Value.Encoded, value) || port.snapshots != 1 || port.publishes != 1 {
					t.Fatal("read-only graph skipped its version CAS", out.Receipt, err)
				}
				if !bytes.Equal(before, persistentStateBytes(t, directory)) {
					t.Fatal("read-only graph changed durable bytes")
				}
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
				store, err = contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
				if err != nil {
					t.Fatal(err)
				}
			}
			competitor, err := contractstate.OpenLocalPersistence(directory, layout, overlayLocalLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer competitor.Close()
			interference := &graphBeforeCASPort{PersistencePort: store}
			interference.beforeCAS = func() {
				if _, err := competitor.CompareAndPublish(layout.Hash(), 1, []contractstate.PersistentWrite{{Key: layout.Fields()[0].Prefix, Value: []byte{7}}}); err != nil {
					t.Fatal("independent writer did not publish", err)
				}
			}
			if _, err := ExecuteGraphPersistent(context.Background(), interference, executable, policy); !errors.Is(err, contractstate.ErrConflict) || interference.snapshots != 1 || interference.publishes != 1 {
				t.Fatal("read-only graph ignored a real competing publication", err, interference.snapshots, interference.publishes)
			}
			actual, err := store.Snapshot(layout.Hash())
			if err != nil || actual.Version != 2 || len(actual.Entries) != 1 || !bytes.Equal(actual.Entries[0].Value, []byte{7}) {
				t.Fatal("conflicting read-only graph changed the winning state", actual, err)
			}
		})
	}
}

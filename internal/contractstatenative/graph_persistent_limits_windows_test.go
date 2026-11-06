package contractstatenative

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func TestGraphPersistenceNativeResponseCeiling(t *testing.T) {
	const copies = 6
	source, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	source.StateLayouts[0].Fields = []hir.StateField{
		{Name: "large", Kind: hir.StateFieldCell, ValueTypeID: "bytes/v1"},
		{Name: "small", Kind: hir.StateFieldCell, ValueTypeID: "bytes/v1"},
	}
	source.StateTransactions[0].Accesses = nil
	for range copies {
		source.StateTransactions[0].Accesses = append(source.StateTransactions[0].Accesses, hir.StateAccess{Kind: hir.StateAccessCellRead, Field: "large"})
	}
	source.StateTransactions[0].Accesses = append(source.StateTransactions[0].Accesses, hir.StateAccess{Kind: hir.StateAccessCellRead, Field: "small"})
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	layout := graphOracleLayout(t, h.program)
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
			for _, delta := range []int{-1, 0, 1} {
				t.Run(fmt.Sprint(delta), func(t *testing.T) {
					// Six repeated reads reach the response ceiling while the
					// base64-encoded local state fits its unchanged 16MiB limit.
					const payloadBytes = (64 << 20) - 186 - (copies+2)*21
					large, small := payloadBytes/copies, payloadBytes%copies+delta
					values := [][]byte{bytes.Repeat([]byte{255}, large), bytes.Repeat([]byte{0}, small)}
					limits := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 64, MaxValueBytes: 16 << 20}
					directory := t.TempDir()
					store, err := contractstate.CreateLocalPersistence(directory, layout, limits)
					if err != nil {
						t.Fatal(err)
					}
					defer store.Close()
					if _, err := store.CompareAndPublish(layout.Hash(), 0, []contractstate.PersistentWrite{
						{Key: h.program.Layout.Fields[0].Prefix, Value: values[0]}, {Key: h.program.Layout.Fields[1].Prefix, Value: values[1]},
					}); err != nil {
						t.Fatal(err)
					}
					before := persistentStateBytes(t, directory)
					port := &overlayObservedPort{PersistencePort: store}
					out, err := ExecuteGraphPersistent(context.Background(), port, executable, overlayPersistentLimits())
					if delta <= 0 {
						if err != nil || out.Exit != contractstatemir.ExitCommit || out.Receipt.Version != 1 || out.Receipt.Writes != 0 || out.Receipt.WriteBytes != 0 || len(out.Reads) != copies+1 || port.publishes != 1 {
							t.Fatal("bounded graph evidence failed its no-op CAS", delta, out.Receipt, err)
						}
						for i, read := range out.Reads {
							value := values[0]
							if i == copies {
								value = values[1]
							}
							if !read.Found || read.Operation != i || !bytes.Equal(read.Value.Encoded, value) {
								t.Fatal("bounded graph observation changed", delta, i)
							}
						}
					} else {
						var runtime GraphRuntimeError
						if !errors.As(err, &runtime) || runtime.Status != 1 || port.publishes != 0 || out.Exit != "" {
							t.Fatal("oversized graph evidence acquired CAS authority", out.Receipt, err, port.publishes)
						}
					}
					if port.snapshots != 1 || !bytes.Equal(before, persistentStateBytes(t, directory)) {
						t.Fatal("bounded graph changed state or repeated its snapshot", delta, port.snapshots)
					}
					if err := store.Close(); err != nil {
						t.Fatal(err)
					}
					reopened, err := contractstate.OpenLocalPersistence(directory, layout, limits)
					if err != nil {
						t.Fatal(err)
					}
					defer reopened.Close()
					snapshot, err := reopened.Snapshot(layout.Hash())
					if err != nil || snapshot.Version != 1 || len(snapshot.Entries) != 2 {
						t.Fatal("bounded graph state failed reopen", snapshot.Version, err)
					}
				})
			}
		})
	}
}

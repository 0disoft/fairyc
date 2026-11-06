package contractstatenative

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

//go:embed graph_runtime_probe.c
var graphRuntimeProbe string

func graphKeyOracle(t *testing.T) (input, expected []byte, count int) {
	t.Helper()
	type sample struct {
		codec byte
		id    string
		value []byte
	}
	var samples []sample
	for size := 0; size <= 130; size++ {
		value := make([]byte, size)
		for index := range value {
			value[index] = byte(index)
		}
		samples = append(samples, sample{4, contractstate.Bytes.TypeID(), value})
	}
	for _, size := range []int{255, 256, 257, 4095, 4096, 4097, 65535, 65536, 1048576} {
		samples = append(samples, sample{4, contractstate.Bytes.TypeID(), bytes.Repeat([]byte{255}, size)})
	}
	for index, id := range []string{contractstate.Bool.TypeID(), contractstate.U32.TypeID(), contractstate.U64.TypeID(), contractstate.Bytes.TypeID(), contractstate.Text.TypeID(), contractstate.Byte20.TypeID(), contractstate.Byte32.TypeID()} {
		values := [][]byte{nil, {0}, {1}, {2}, bytes.Repeat([]byte{255}, 3), bytes.Repeat([]byte{255}, 4), bytes.Repeat([]byte{255}, 7), bytes.Repeat([]byte{255}, 8), bytes.Repeat([]byte{128}, 19), bytes.Repeat([]byte{128}, 20), bytes.Repeat([]byte{128}, 21), bytes.Repeat([]byte{128}, 31), bytes.Repeat([]byte{128}, 32), bytes.Repeat([]byte{128}, 33), []byte("키🔑"), {0xed, 0xa0, 0x80}, {0xf4, 0x8f, 0xbf, 0xbf}}
		for _, value := range values {
			samples = append(samples, sample{byte(index + 1), id, value})
		}
	}
	samples = append(samples, sample{0, "unknown", nil}, sample{255, "unknown", []byte{1}})
	input = append(input, 0, 0, 0, 0, 0)
	for sampleIndex, sample := range samples {
		prefix := sha256.Sum256([]byte(fmt.Sprintf("native.graph.map.%d", sampleIndex)))
		maximums := []uint64{uint64(max(1, len(sample.value))), uint64(len(sample.value) + 1), uint64(max(0, len(sample.value)-1))}
		for _, maximum := range maximums {
			for alias := byte(0); alias <= 2; alias++ {
				input = append(input, alias, sample.codec)
				input = binary.BigEndian.AppendUint64(input, maximum)
				input = binary.BigEndian.AppendUint32(input, uint32(len(sample.value)))
				input = append(input, prefix[:]...)
				input = append(input, sample.value...)
				status := byte(0)
				var digest [32]byte
				if maximum == 0 {
					status = 1
				} else if err := contractstate.ValidateCanonicalEncoding(sample.id, sample.value); err != nil {
					status = 6
				} else if uint64(len(sample.value)) > maximum {
					status = 7
				} else {
					payload := append([]byte("fairyc-contract-state-map-entry/v1"), prefix[:]...)
					payload = binary.BigEndian.AppendUint64(payload, uint64(len(sample.value)))
					digest = sha256.Sum256(append(payload, sample.value...))
				}
				expected = append(expected, status)
				expected = append(expected, digest[:]...)
				count++
			}
		}
	}
	binary.BigEndian.PutUint32(input[1:5], uint32(count))
	return input, expected, count
}

func graphStateOracle(t *testing.T) (input, expected []byte) {
	t.Helper()
	builder, err := contractstate.NewLayoutBuilder("native.graph.runtime")
	if err != nil {
		t.Fatal(err)
	}
	a, err := contractstate.DeclareEncodedCell(builder, "source", contractstate.Bytes.TypeID())
	if err != nil {
		t.Fatal(err)
	}
	b, err := contractstate.DeclareEncodedCell(builder, "target", contractstate.Bytes.TypeID())
	if err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	store, err := contractstate.NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := store.Begin(layout, contractstate.Limits{MaxWrites: 3, MaxWriteBytes: 256, MaxValueBytes: 64, MaxMapKeyBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	status := func(err error) byte {
		if err == nil {
			return 0
		}
		if errors.Is(err, contractstate.ErrBorrowInvalidated) {
			return 9
		}
		t.Fatal("unexpected oracle error", err)
		return 255
	}
	put := func(err error) { expected = append(expected, status(err)) }
	value := func(data []byte, found bool, err error) {
		expected = append(expected, status(err), 0)
		if found {
			expected[len(expected)-1] = 1
		}
		expected = binary.BigEndian.AppendUint32(expected, uint32(len(data)))
		expected = append(expected, data...)
	}
	input = append(input, 1)
	keyA, keyB := a.Field().Prefix, b.Field().Prefix
	input = append(input, keyA[:]...)
	input = append(input, keyB[:]...)
	put(a.Set(tx, []byte("alpha")))
	saved, found, err := a.Get(tx)
	value(saved, found, err)
	borrow, found, err := a.Borrow(tx)
	put(err)
	if !found {
		t.Fatal("missing oracle borrow")
	}
	expected = append(expected, 1)
	view, err := borrow.View(1, 3)
	put(err)
	put(b.Set(tx, []byte("beta")))
	data, err := borrow.Bytes()
	value(data, true, err)
	data, err = view.Bytes()
	value(data, true, err)
	put(a.Set(tx, []byte("next")))
	value(saved, true, nil)
	_, err = borrow.Bytes()
	put(err)
	_, err = view.Bytes()
	put(err)
	put(b.Set(tx, saved))
	second, found, err := b.Borrow(tx)
	put(err)
	if !found {
		t.Fatal("missing second oracle borrow")
	}
	put(a.Delete(tx))
	// The state MIR assigns canonical typed zero, rather than an untyped empty
	// slice, to absent SSA reads. Their observation remains explicitly absent.
	boolean, _ := contractstate.Bool.Encode(false)
	u32, _ := contractstate.U32.Encode(0)
	u64, _ := contractstate.U64.Encode(0)
	for _, zero := range [][]byte{boolean, u32, u64, nil, nil, make([]byte, 20), make([]byte, 32)} {
		_, present, err := a.Get(tx)
		if present || err != nil {
			t.Fatal("deleted oracle cell became present", err)
		}
		value(zero, false, nil)
	}
	data, err = second.Bytes()
	value(data, true, err)
	put(b.Set(tx, []byte("next")))
	_, err = second.Bytes()
	put(err)
	return input, expected
}

func TestNativeStateGraphRuntime(t *testing.T) {
	header, err := cstdlib.ContractStateGraphDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cstdlib.ContractStateGraphRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	keyInput, keyExpected, count := graphKeyOracle(t)
	stateInput, stateExpected := graphStateOracle(t)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, header+"\n"+graphRuntimeProbe, runtimeSource)
			if err != nil {
				t.Fatal(err)
			}
			for _, trial := range []struct {
				name            string
				input, expected []byte
			}{
				{"map-keys", keyInput, keyExpected}, {"values-borrows", stateInput, stateExpected},
			} {
				t.Run(trial.name, func(t *testing.T) {
					out, err := cdriver.ExecuteWindowsMSVCTargetWithContextAndOutputLimit(context.Background(), image, bytes.NewReader(trial.input), time.Minute, len(trial.expected)+1)
					if err != nil || out.ExitCode != 0 || out.Stderr != "" || !bytes.Equal([]byte(out.Stdout), trial.expected) {
						first := 0
						for first < min(len(out.Stdout), len(trial.expected)) && out.Stdout[first] == trial.expected[first] {
							first++
						}
						t.Fatalf("native graph %s: exit %d error %v, output %d/%d, first difference %d", trial.name, out.ExitCode, err, len(out.Stdout), len(trial.expected), first)
					}
				})
			}
			t.Logf("matched %d independently hashed/validated map keys and Go value/borrow observations", count)
		})
	}
}

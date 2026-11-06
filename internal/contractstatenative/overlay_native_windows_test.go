package contractstatenative

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func overlayReferencePort(t *testing.T, layout contractstate.Layout, snapshot contractstate.PersistentSnapshot) *overlayOraclePort {
	t.Helper()
	store, err := contractstate.NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entries) != 0 {
		var seed []contractstate.PersistentWrite
		for _, entry := range snapshot.Entries {
			seed = append(seed, contractstate.PersistentWrite{Key: entry.Key, Value: entry.Value})
		}
		if _, err := store.CompareAndPublish(layout.Hash(), 0, seed); err != nil {
			t.Fatal(err)
		}
	}
	return &overlayOraclePort{PersistencePort: store}
}

func compareOverlayNative(t *testing.T, got OverlayResult, handoff Handoff, layout contractstate.Layout, snapshot contractstate.PersistentSnapshot, limits contractstate.Limits) {
	t.Helper()
	port := overlayReferencePort(t, layout, snapshot)
	want, trap, err := persistentOracle(port, handoff.program, limits)
	status := byte(0)
	switch {
	case errors.Is(err, contractstate.ErrValueTooLarge):
		status = 3
	case errors.Is(err, contractstate.ErrWriteLimit):
		status = 4
	case errors.Is(err, contractstate.ErrWriteBytesLimit):
		status = 5
	case errors.Is(err, contractstate.ErrMapKeyTooLarge):
		status = 7
	case err != nil:
		t.Fatal(err)
	}
	exit := want.Exit
	if trap != "" {
		exit = contractstatemir.ExitTrap
	} else if status != 0 {
		exit = contractstatemir.ExitCommit
	}
	if got.Status != status || got.Exit != exit || got.Version != snapshot.Version || got.TrapCategory != trap || got.WriteBytes != want.Receipt.WriteBytes || !slices.EqualFunc(got.Writes, port.writes, func(a, b contractstate.PersistentWrite) bool {
		return a.Key == b.Key && a.Deleted == b.Deleted && bytes.Equal(a.Value, b.Value)
	}) {
		t.Fatalf("native %+v; Go %+v/%q/%v", got, want, trap, err)
	}
	reads := want.Reads
	if trap != "" || status != 0 {
		control := handoff.program
		control.Exit = contractstatemir.ExitReject
		control.TrapCategory = ""
		if status != 0 {
			length := 0
			if len(got.Reads) != 0 {
				length = got.Reads[len(got.Reads)-1].Operation + 1
			}
			control.Operations = control.Operations[:length]
		}
		observed, category, err := persistentOracle(overlayReferencePort(t, layout, snapshot), control, limits)
		if err != nil || category != "" {
			t.Fatal("independent read-prefix control", category, err)
		}
		reads = observed.Reads
	}
	if !slices.EqualFunc(got.Reads, reads, func(a, b contractstatemir.Read) bool {
		return a.Operation == b.Operation && a.Found == b.Found && a.Value.TypeID == b.Value.TypeID && bytes.Equal(a.Value.Encoded, b.Value.Encoded)
	}) {
		t.Fatal("native read prefix differs from independent Go control")
	}
}

func TestOverlayFramingNative(t *testing.T) {
	for _, family := range []string{"commit", "reject", "trap", "empty"} {
		t.Run(family, func(t *testing.T) {
			exit := contractstatemir.ExitCommit
			if family == "reject" {
				exit = contractstatemir.ExitReject
			} else if family == "trap" {
				exit = contractstatemir.ExitTrap
			}
			handoff, layout := overlayMIRFixture(t, exit, family == "empty")
			prepared, err := PrepareOverlayTransport(handoff, target.X8664WindowsMSVC)
			if err != nil {
				t.Fatal(err)
			}
			source, err := prepared.Source()
			if err != nil {
				t.Fatal(err)
			}
			repeat, err := prepared.Source()
			if err != nil || source != repeat {
				t.Fatal("nondeterministic framing source", err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					executable, err := BuildOverlay(prepared, profile)
					if err != nil {
						t.Fatal(err)
					}
					for _, policy := range []struct {
						name       string
						count      int
						size       uint64
						value, key int
					}{{"exact", 14, 2048, 32, 32}, {"count-before", 13, 2048, 32, 32}, {"value-before", 14, 2048, 31, 32}, {"key-before", 14, 2048, 32, 31}, {"bytes-before-delete", 14, 32, 32, 32}, {"after", 15, 2049, 33, 33}, {"full-width", math.MaxInt, math.MaxUint64, math.MaxInt, math.MaxInt}} {
						t.Run(policy.name, func(t *testing.T) {
							limits := contractstate.Limits{MaxWrites: policy.count, MaxWriteBytes: policy.size, MaxValueBytes: policy.value, MaxMapKeyBytes: policy.key}
							snapshot := contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: math.MaxUint64}
							input, err := prepared.Input(snapshot, limits)
							if err != nil {
								t.Fatal(err)
							}
							got, err := RunOverlay(context.Background(), executable, input)
							if err != nil {
								t.Fatal(err)
							}
							compareOverlayNative(t, got, handoff, layout, snapshot, limits)
						})
					}
				})
			}
		})
	}
}

func overlayReadFixture(t *testing.T) (Handoff, contractstate.Layout) {
	t.Helper()
	input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	input.StateLayouts[0].Fields[0].ValueTypeID = contractstate.Bytes.TypeID()
	input.StateTransactions[0].Accesses = []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}}
	handoff, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	builder, err := contractstate.NewLayoutBuilder("native.state")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contractstate.DeclareEncodedCell(builder, "value", contractstate.Bytes.TypeID()); err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	return handoff, layout
}

func TestOverlayFramingLargeRead(t *testing.T) {
	handoff, layout := overlayReadFixture(t)
	prepared, err := PrepareOverlayTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	value := make([]byte, 65537)
	for index := range value {
		value[index] = byte(index)
	}
	snapshot := contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: math.MaxUint64, Entries: []contractstate.Entry{{Key: layout.Fields()[0].Prefix, Value: value}}}
	limits := contractstate.Limits{MaxWrites: math.MaxInt, MaxWriteBytes: math.MaxUint64, MaxValueBytes: 1, MaxMapKeyBytes: math.MaxInt}
	input, err := prepared.Input(snapshot, limits)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildOverlay(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			got, err := RunOverlay(context.Background(), executable, input)
			if err != nil {
				t.Fatal(err)
			}
			compareOverlayNative(t, got, handoff, layout, snapshot, limits)
			// 142 header + 10 read metadata + the exact large binary value.
			for _, bound := range []int{65688, 65689, 65690} {
				out, err := cdriver.ExecuteWindowsMSVCTargetWithContextAndOutputLimit(context.Background(), executable.image, bytes.NewReader(input.Bytes()), time.Minute, bound)
				if bound == 65688 {
					if err == nil {
						t.Fatal("capture admitted one-byte overflow")
					}
					continue
				}
				if err != nil || len(out.Stdout) != 65689 {
					t.Fatal("large response capture", len(out.Stdout), err)
				}
				if _, err := prepared.Decode(input, []byte(out.Stdout)); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestOverlayFramingRejectsNativeRequests(t *testing.T) {
	_, prepared, _, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	limits := contractstate.Limits{MaxWrites: 1024, MaxWriteBytes: 1 << 20, MaxValueBytes: 1 << 20, MaxMapKeyBytes: 1 << 20}
	input, err := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Entries: []contractstate.Entry{{Key: layout.Fields()[0].Prefix, Value: []byte{1}}}}, limits)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildOverlay(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, fault := range []string{"short-header", "short-payload", "trailing", "magic", "layout", "body", "program", "zero-writes", "zero-bytes", "zero-value", "zero-key", "count", "size", "minimum-size", "value-size", "canonical-value"} {
				t.Run(fault, func(t *testing.T) {
					packet := input.Bytes()
					switch fault {
					case "short-header":
						packet = packet[:151]
					case "short-payload":
						packet = packet[:len(packet)-1]
					case "trailing":
						packet = append(packet, 0)
					case "magic":
						packet[0] ^= 1
					case "layout":
						packet[8] ^= 1
					case "body":
						packet[40] ^= 1
					case "program":
						packet[72] ^= 1
					case "zero-writes":
						clear(packet[112:120])
					case "zero-bytes":
						clear(packet[120:128])
					case "zero-value":
						clear(packet[128:136])
					case "zero-key":
						clear(packet[136:144])
					case "count":
						binary.BigEndian.PutUint32(packet[144:148], 65537)
					case "size":
						binary.BigEndian.PutUint32(packet[148:152], (16<<20)+1)
					case "minimum-size":
						binary.BigEndian.PutUint32(packet[144:148], 2)
					case "value-size":
						binary.BigEndian.PutUint32(packet[184:188], math.MaxUint32)
					case "canonical-value":
						packet[188] = 2
					}
					out, err := cdriver.ExecuteWindowsMSVCTargetWithContextAndOutputLimit(context.Background(), executable.image, bytes.NewReader(packet), time.Minute, 64<<20)
					if err != nil || out.ExitCode != 4 || out.Stdout != "" || out.Stderr != "" {
						t.Fatal("malformed request escaped", out, err)
					}
				})
			}
		})
	}
}

func TestOverlayFramingResponseCeiling(t *testing.T) {
	input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	input.StateLayouts[0].Fields = []hir.StateField{
		{Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bytes.TypeID()},
		{Name: "tail", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bytes.TypeID()},
	}
	input.StateTransactions[0].Accesses = nil
	for range 5 {
		input.StateTransactions[0].Accesses = append(input.StateTransactions[0].Accesses, hir.StateAccess{Kind: hir.StateAccessCellRead, Field: "value"})
	}
	input.StateTransactions[0].Accesses = append(input.StateTransactions[0].Accesses, hir.StateAccess{Kind: hir.StateAccessCellRead, Field: "tail"})
	handoff, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareOverlayTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	// 142 header + six 10-byte records + five large values + two tail bytes
	// is exactly 64 MiB. The entire snapshot stays below its separate 16 MiB cap.
	large := bytes.Repeat([]byte{0, 255, 26, 13}, 3355433)
	limits := contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 1, MaxValueBytes: 1, MaxMapKeyBytes: 1}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			executable, err := BuildOverlay(prepared, profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, size := range []int{1, 2, 3} {
				snapshot := contractstate.PersistentSnapshot{LayoutHash: handoff.program.Layout.Hash, Entries: []contractstate.Entry{
					{Key: handoff.program.Layout.Fields[0].Prefix, Value: large},
					{Key: handoff.program.Layout.Fields[1].Prefix, Value: bytes.Repeat([]byte{255}, size)},
				}}
				slices.SortFunc(snapshot.Entries, func(a, b contractstate.Entry) int { return bytes.Compare(a.Key[:], b.Key[:]) })
				request, err := prepared.Input(snapshot, limits)
				if err != nil {
					t.Fatal(err)
				}
				out, err := cdriver.ExecuteWindowsMSVCTargetWithContextAndOutputLimit(context.Background(), executable.image, bytes.NewReader(request.Bytes()), time.Minute, 64<<20)
				if err != nil || out.ExitCode != 0 || out.Stderr != "" {
					t.Fatal("response ceiling process", size, out.ExitCode, err)
				}
				got, err := prepared.Decode(request, []byte(out.Stdout))
				if err != nil || len(got.Writes) != 0 || got.WriteBytes != 0 {
					t.Fatal("response ceiling admission", size, err)
				}
				count := 6
				if size == 3 {
					count = 5
					if got.Status != 1 || len(out.Stdout) >= 64<<20 {
						t.Fatal("one-byte evidence overflow did not fail closed", got.Status, len(out.Stdout))
					}
				} else if got.Status != 0 || len(out.Stdout) != (64<<20)+size-2 {
					t.Fatal("exact evidence boundary", size, got.Status, len(out.Stdout))
				}
				if len(got.Reads) != count {
					t.Fatal("lost bounded read prefix", len(got.Reads), count)
				}
				for index, read := range got.Reads {
					want := large
					if index == 5 {
						want = bytes.Repeat([]byte{255}, size)
					}
					if !read.Found || read.Operation != index || !bytes.Equal(read.Value.Encoded, want) {
						t.Fatal("large binary observation differs", index)
					}
				}
			}
		})
	}
}

const overlayAllocationProbe = `
#include <stdlib.h>
static size_t fyov_test_calls, fyov_test_alive;
static void fyov_test_check(void) { if (fyov_test_alive != 0u) { _Exit(91); } }
static void *fyov_test_alloc(size_t size) {
    void *value;
    const char *failure = getenv("FAIRYC_TEST_OVERLAY_ALLOC_FAIL");
    if (fyov_test_calls == 0u && atexit(fyov_test_check) != 0) { return NULL; }
    ++fyov_test_calls;
    if (failure != NULL && strtoul(failure, NULL, 10) == fyov_test_calls) { return NULL; }
    value = malloc(size); if (value != NULL) { ++fyov_test_alive; } return value;
}
static void fyov_test_release(void *value) { if (value != NULL) { --fyov_test_alive; } free(value); }
#define FYOV_ALLOC fyov_test_alloc
#define FYOV_RELEASE fyov_test_release
`

func TestOverlayFramingAllocationCleanup(t *testing.T) {
	_, prepared, input, _ := overlayWireFixture(t, contractstatemir.ExitCommit)
	source, err := prepared.Source()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cstdlib.ContractStateOverlayRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsHost(toolchain, profile, overlayAllocationProbe+source, runtimeSource)
			if err != nil {
				t.Fatal(err)
			}
			for _, failure := range []string{"1", "2", "3", "4", "5", "6", "0"} {
				t.Setenv("FAIRYC_TEST_OVERLAY_ALLOC_FAIL", failure)
				out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input.Bytes()))
				if failure == "0" {
					if err != nil || out.ExitCode != 0 {
						t.Fatal("successful allocations not cleaned", out, err)
					}
					if _, err := prepared.Decode(input, []byte(out.Stdout)); err != nil {
						t.Fatal(err)
					}
					continue
				}
				if err != nil || out.ExitCode != 4 || out.Stdout != "" || out.Stderr != "" {
					t.Fatal("allocation failure leaked or published", failure, out, err)
				}
			}
		})
	}
}

func TestOverlayFramingRefusesZeroOwner(t *testing.T) {
	if _, err := BuildOverlay(OverlayTransport{}, cdriver.LowProfile); err == nil {
		t.Fatal("zero transport built")
	}
	if _, err := RunOverlay(context.Background(), OverlayExecutable{}, OverlayInput{}); err == nil {
		t.Fatal("zero built/request owners ran")
	}
}

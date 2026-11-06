package contractstatenative

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatefixture"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func TestGraphFramingAuthoredWire(t *testing.T) {
	for _, helper := range []bool{false, true} {
		for _, exit := range []contractstatemir.ExitKind{contractstatemir.ExitCommit, contractstatemir.ExitReject, contractstatemir.ExitTrap} {
			t.Run(fmt.Sprintf("%s/helper-%t", exit, helper), func(t *testing.T) {
				_, prepared, input, expected := graphWireFixture(t, exit, helper)
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(profile.Name, func(t *testing.T) {
						image, err := BuildGraph(prepared, profile)
						if err != nil {
							t.Fatal(err)
						}
						out, err := image.run(context.Background(), input)
						if err != nil || out.ExitCode != 0 || out.Stderr != "" || !bytes.Equal([]byte(out.Stdout), expected) {
							t.Fatalf("actual C frame differs from independent authored wire: exit=%d error=%v stderr=%s got=%x want=%x", out.ExitCode, err, out.Stderr, out.Stdout, expected)
						}
						if _, err := prepared.decodeProcess(input, out); err != nil {
							t.Fatal(err)
						}
					})
				}
			})
		}
	}
}

// Project the admitted result into the preexisting independent Go-oracle probe
// format. This compares semantics, not two copies of the new wire writer.
func graphProbeResult(p GraphTransport, got GraphResult) []byte {
	exit, trap := byte(0), 0
	if got.Exit == contractstatemir.ExitReject {
		exit = 1
	} else if got.Exit == contractstatemir.ExitTrap {
		exit, trap = 2, slices.Index(p.entry.traps, got.TrapCategory)+1
	}
	result := append([]byte{got.Status, exit}, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(result[2:], uint32(trap))
	result = binary.BigEndian.AppendUint64(result, got.Cost)
	var reads []byte
	ids := map[string]byte{"bool/v1": 1, "u32/be/v1": 2, "u64/be/v1": 3, "bytes/v1": 4, "utf8/v1": 5, "bytes20/v1": 6, "bytes32/v1": 7}
	for _, read := range got.Reads {
		reads = binary.BigEndian.AppendUint64(reads, uint64(read.Operation))
		reads = append(reads, ids[read.Value.TypeID], 0)
		if read.Found {
			reads[len(reads)-1] = 1
		}
		reads = binary.BigEndian.AppendUint32(reads, uint32(len(read.Value.Encoded)))
		reads = append(reads, read.Value.Encoded...)
	}
	result = binary.BigEndian.AppendUint32(result, uint32(len(reads)))
	result = binary.BigEndian.AppendUint16(result, uint16(len(got.Writes)))
	result = binary.BigEndian.AppendUint64(result, got.WriteBytes)
	result = append(result, reads...)
	for _, w := range got.Writes {
		result = append(result, w.Key[:]...)
		result = append(result, 0)
		if w.Deleted {
			result[len(result)-1] = 1
		}
		result = binary.BigEndian.AppendUint32(result, uint32(len(w.Value)))
		result = append(result, w.Value...)
	}
	return result
}

func TestGraphFramingNative(t *testing.T) {
	_, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	for _, version := range []int{0, 1, 2, 3, 4} {
		path := fmt.Sprintf("../../spec/fixtures/contract-state/manifest-v%d.json", version)
		cases, err := contractstatefixture.LoadHIRCases(path)
		if err != nil {
			t.Fatal(err)
		}
		if version == 4 {
			cases = append(cases, dynamicGraphCases(t)...)
		}
		for _, trial := range cases {
			t.Run(fmt.Sprintf("v%d-%s", version, trial.Name), func(t *testing.T) {
				h, err := Prepare(trial.Input, 1, 1, coordinate)
				if err != nil {
					t.Fatal(err)
				}
				prepared, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
				if err != nil {
					t.Fatal(err)
				}
				_, want := graphExpected(t, h, prepared.entry, trial.Limits)
				input, err := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash}, trial.Limits)
				if err != nil {
					t.Fatal(err)
				}
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(profile.Name, func(t *testing.T) {
						executable, err := BuildGraph(prepared, profile)
						if err != nil {
							t.Fatal(err)
						}
						got, err := RunGraph(context.Background(), executable, input)
						if err != nil {
							t.Fatal(err)
						}
						if want[0] != 0 {
							if got.Status != want[0] || len(got.Writes) != 0 || got.WriteBytes != 0 || got.Exit != "" {
								t.Fatal("failed native graph published", got)
							}
						} else if observed := graphProbeResult(prepared, got); !bytes.Equal(observed, want) {
							t.Fatalf("admitted native semantics %x; independent Go %x", observed, want)
						}
					})
				}
			})
		}
	}
}

func TestGraphFramingMalformedAndCleanup(t *testing.T) {
	_, prepared, input, _ := graphWireFixture(t, contractstatemir.ExitCommit, false)
	source, err := prepared.Source()
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
	probe := strings.ReplaceAll(strings.ReplaceAll(overlayAllocationProbe, "fyov_", "fygr_"), "FYOV_", "FYGR_")
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, probe+source, runtimeSource)
			if err != nil {
				t.Fatal(err)
			}
			for failure := 0; failure <= 8; failure++ {
				t.Setenv("FAIRYC_TEST_OVERLAY_ALLOC_FAIL", fmt.Sprint(failure))
				out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input.Bytes()))
				if failure == 0 {
					if err != nil || out.ExitCode != 0 {
						t.Fatal("successful allocation cleanup", out, err)
					}
					if _, err := prepared.decodeProcess(input, out); err != nil {
						t.Fatal(err)
					}
				} else if err != nil || out.ExitCode != 4 || out.Stdout != "" || out.Stderr != "" {
					t.Fatal("allocation failure leaked or published", failure, out, err)
				}
			}
			t.Setenv("FAIRYC_TEST_OVERLAY_ALLOC_FAIL", "0")
			bad := map[string][]byte{"empty": nil, "short-header": input.Bytes()[:151], "trailing": append(input.Bytes(), 0)}
			for _, offset := range []int{0, 8, 40, 72} {
				packet := input.Bytes()
				packet[offset] ^= 1
				bad[fmt.Sprint("identity-", offset)] = packet
			}
			for _, offset := range []int{112, 120, 128, 136} {
				packet := input.Bytes()
				clear(packet[offset : offset+8])
				bad[fmt.Sprint("zero-policy-", offset)] = packet
			}
			for _, offset := range []int{144, 148} {
				packet := input.Bytes()
				binary.BigEndian.PutUint32(packet[offset:], ^uint32(0))
				bad[fmt.Sprint("overflow-", offset)] = packet
			}
			for name, packet := range bad {
				out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(packet))
				if err != nil || out.ExitCode != 4 || out.Stdout != "" || out.Stderr != "" {
					t.Fatal("malformed request leaked or published", name, out, err)
				}
			}
		})
	}
}

func TestGraphFramingSnapshot(t *testing.T) {
	p, snapshot, trace, key := graphBranchFixture(t)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := BuildGraph(p, profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, full := range []bool{false, true} {
				limits := contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 41, MaxValueBytes: 8, MaxMapKeyBytes: 8}
				if full {
					limits = contractstate.Limits{MaxWrites: math.MaxInt, MaxWriteBytes: math.MaxUint64, MaxValueBytes: math.MaxInt, MaxMapKeyBytes: math.MaxInt}
				}
				input, err := p.Input(snapshot, limits)
				if err != nil {
					t.Fatal(err)
				}
				out, err := image.run(context.Background(), input)
				want := graphWireResponse(input, 0, 0, 0, 4, 2, 4, trace, []contractstate.PersistentWrite{{Key: key, Value: snapshot.Entries[0].Value}})
				if err != nil || out.ExitCode != 0 || !bytes.Equal([]byte(out.Stdout), want) {
					t.Fatal("snapshot or full-width policy changed actual execution", full, out.ExitCode, err)
				}
				if _, err := p.decodeProcess(input, out); err != nil {
					t.Fatal(err)
				}
				if _, err := RunGraph(nil, image, input); err == nil {
					t.Fatal("nil context accepted")
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := RunGraph(ctx, image, input); !errors.Is(err, context.Canceled) {
					t.Fatal("canceled graph run succeeded")
				}
				packet := input.Bytes()
				bad := map[string][]byte{"short-body": packet[:len(packet)-1], "trailing-body": append(slices.Clone(packet), 0)}
				oversize := slices.Clone(packet)
				binary.BigEndian.PutUint32(oversize[184:], math.MaxUint32)
				bad["value-length-overflow"] = oversize
				codec := slices.Clone(packet[:len(packet)-1])
				binary.BigEndian.PutUint32(codec[148:], 43)
				binary.BigEndian.PutUint32(codec[184:], 7)
				bad["cell-codec"] = codec
				duplicate := append(slices.Clone(packet), packet[152:]...)
				binary.BigEndian.PutUint32(duplicate[144:], 2)
				binary.BigEndian.PutUint32(duplicate[148:], 88)
				bad["duplicate-keys"] = duplicate
				for name, bytes := range bad {
					out, err := image.run(context.Background(), GraphInput{packet: bytes, valid: true})
					if err != nil || out.ExitCode != 4 || out.Stdout != "" || out.Stderr != "" {
						t.Fatal("malformed snapshot published", name, out, err)
					}
				}
			}
		})
	}
}

func TestGraphFramingLargeBinary(t *testing.T) {
	source, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	source.StateLayouts[0].Fields[0].ValueTypeID = "bytes/v1"
	source.StateTransactions[0].Accesses = []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}}
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := BuildGraph(p, profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, size := range []int{0, 65535, 65536, 65537} {
				value := make([]byte, size)
				for i := range value {
					value[i] = byte(i)
				}
				input, err := p.Input(contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash, Entries: []contractstate.Entry{{Key: h.program.Layout.Fields[0].Prefix, Value: value}}},
					contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 1, MaxValueBytes: 1, MaxMapKeyBytes: 1})
				if err != nil {
					t.Fatal(err)
				}
				out, err := RunGraph(context.Background(), image, input)
				if err != nil || out.Status != 0 || len(out.Reads) != 1 || !out.Reads[0].Found || !bytes.Equal(out.Reads[0].Value.Encoded, value) {
					t.Fatal("large/present-empty binary read changed", size, err)
				}
			}
		})
	}
}

func TestGraphFramingResponseCeiling(t *testing.T) {
	source, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	source.StateLayouts[0].Fields = []hir.StateField{
		{Name: "large", Kind: hir.StateFieldCell, ValueTypeID: "bytes/v1"},
		{Name: "small", Kind: hir.StateFieldCell, ValueTypeID: "bytes/v1"},
	}
	source.StateTransactions[0].Accesses = []hir.StateAccess{
		{Kind: hir.StateAccessCellRead, Field: "large"}, {Kind: hir.StateAccessCellRead, Field: "large"},
		{Kind: hir.StateAccessCellRead, Field: "large"}, {Kind: hir.StateAccessCellRead, Field: "large"}, {Kind: hir.StateAccessCellRead, Field: "small"},
	}
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := BuildGraph(p, profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, delta := range []int{-1, 0, 1} {
				large, small := (16<<20)-78, 0
				if delta == -1 {
					large, small = large-1, 3
				} else if delta == 1 {
					small = 1
				}
				values := [][]byte{bytes.Repeat([]byte{255}, large), bytes.Repeat([]byte{0}, small)}
				snapshot := contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash, Entries: []contractstate.Entry{
					{Key: h.program.Layout.Fields[0].Prefix, Value: values[0]}, {Key: h.program.Layout.Fields[1].Prefix, Value: values[1]},
				}}
				slices.SortFunc(snapshot.Entries, func(a, b contractstate.Entry) int { return bytes.Compare(a.Key[:], b.Key[:]) })
				input, err := p.Input(snapshot, overlayPersistentLimits())
				if err != nil {
					t.Fatal(err)
				}
				out, err := image.run(context.Background(), input)
				if err != nil || out.ExitCode != 0 || out.Stderr != "" {
					t.Fatal("bounded native framing", delta, out.ExitCode, err)
				}
				got, err := p.decodeProcess(input, out)
				if err != nil || len(got.Reads) != 5 || len(got.Writes) != 0 || got.WriteBytes != 0 {
					t.Fatal("bounded native admission", delta, err)
				}
				if delta <= 0 && (got.Status != 0 || len(out.Stdout) != (64<<20)+delta) ||
					delta > 0 && (got.Status != 1 || got.Exit != "" || len(out.Stdout) != (64<<20)-20) {
					t.Fatal("exact native response ceiling", delta, got.Status, len(out.Stdout))
				}
				for i, read := range got.Reads {
					value := values[0]
					if i == 4 {
						value = values[1]
					}
					if !read.Found || read.Operation != i || !bytes.Equal(read.Value.Encoded, value) {
						t.Fatal("bounded binary trace changed", delta, i)
					}
				}
			}
		})
	}
}

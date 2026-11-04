package packageartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableJSONLimitBoundariesAndEscapes(t *testing.T) {
	artifact := stableCompilationDecodeFixture(t, 1)
	artifact.MIR.Functions[0].Function.InlineBody = "<>&\"\\\n\u2028\u2029\xff"
	cases := []struct {
		name     string
		value    any
		encode   func(uint64) ([]byte, error)
		tooLarge error
	}{
		{"compilation", stableCompilationEnvelope{StableCompilationSchema, StableCompilationVersion, artifact}, func(n uint64) ([]byte, error) { return EncodeStableCompilationArtifact(artifact, n) }, ErrStableCompilationTooLarge},
		{"MIR", stableArtifactEnvelope{StableArtifactSchema, StableArtifactVersion, artifact.MIR}, func(n uint64) ([]byte, error) { return EncodeStableArtifact(artifact.MIR, n) }, ErrStableArtifactTooLarge},
		{"body", bodyEnvelope{BodySchema, BodyVersion, artifact.MIR.Functions[0]}, func(n uint64) ([]byte, error) { return EncodeStableBody(artifact.MIR.Functions[0], n) }, ErrBodyArtifactTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			got, err := tc.encode(uint64(len(want)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatal("encoding changed")
			}
			if _, err = tc.encode(uint64(len(want) - 1)); !errors.Is(err, tc.tooLarge) {
				t.Fatalf("boundary error: %v", err)
			}
			if exceeds, err := stableJSONExceedsLimit(tc.value, uint64(len(want))); err != nil || exceeds {
				t.Fatal("exact limit rejected", err)
			}
			if exceeds, err := stableJSONExceedsLimit(tc.value, uint64(len(want)-1)); err != nil || !exceeds {
				t.Fatal("counter missed exact boundary", err)
			}
		})
	}
}

func TestStableJSONLimitLargeDataAndValidationPrecedence(t *testing.T) {
	artifact := stableCompilationDecodeFixture(t, 1)
	artifact.MIR.Functions[0].Function.InlineBody = strings.Repeat("x", 16<<20)
	// The validation path must run first, even when the payload is oversized.
	invalid := artifact
	invalid.StateBody.Hash[0] ^= 1
	if _, err := EncodeStableCompilationArtifact(invalid, 1024); err == nil || errors.Is(err, ErrStableCompilationTooLarge) || !strings.Contains(err.Error(), "hash does not match") {
		t.Fatalf("validation precedence: %v", err)
	}
	// Measure after input creation and a warm-up, so the result tracks encoding allocations.
	for _, encode := range []struct {
		name   string
		fn     func() error
		target error
	}{
		{"compilation", func() error { _, err := EncodeStableCompilationArtifact(artifact, 1024); return err }, ErrStableCompilationTooLarge},
		{"MIR", func() error { _, err := EncodeStableArtifact(artifact.MIR, 1024); return err }, ErrStableArtifactTooLarge},
		{"body", func() error { _, err := EncodeStableBody(artifact.MIR.Functions[0], 1024); return err }, ErrBodyArtifactTooLarge},
	} {
		t.Run(encode.name, func(t *testing.T) {
			if err := encode.fn(); !errors.Is(err, encode.target) {
				t.Fatalf("error: %v", err)
			}
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			err := encode.fn()
			runtime.ReadMemStats(&after)
			if !errors.Is(err, encode.target) {
				t.Fatalf("error: %v", err)
			}
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 2<<20 {
				t.Fatalf("oversized rejection allocated %d bytes", allocated)
			}
		})
	}
}

func TestStableJSONLimitCountsOperandStaticBytes(t *testing.T) {
	operand := mir.Operand{StaticData: strings.Repeat("x", 1<<20)}
	if exceeds, err := stableJSONExceedsLimit(operand, 1024); err != nil || !exceeds {
		t.Fatal("operand custom marshaler bypassed limit")
	}
	encoded, err := json.Marshal(operand)
	if err != nil {
		t.Fatal(err)
	}
	if exceeds, err := stableJSONExceedsLimit(operand, uint64(len(encoded))); err != nil || exceeds {
		t.Fatal("operand lower bound exceeds JSON")
	}
}

func TestStableJSONLimitManySmallValues(t *testing.T) {
	artifact := stableCompilationDecodeFixture(t, 1)
	artifact.MIR.Functions[0].Function.Entities = make([]string, 200000)
	if exceeds, err := stableJSONExceedsLimit(artifact, 4096); err != nil || !exceeds {
		t.Fatal("many small values bypassed limit")
	}
	encoded, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if exceeds, err := stableJSONExceedsLimit(artifact, uint64(len(encoded))); err != nil || exceeds {
		t.Fatal("lower bound exceeds actual JSON")
	}
}

func TestStableJSONLimitPreservesMarshalError(t *testing.T) {
	artifact := stableCompilationDecodeFixture(t, 1)
	f := &artifact.MIR.Functions[0].Function
	f.InlineBody = strings.Repeat("x", 1<<20)
	f.Blocks[0].Operations = append(f.Blocks[0].Operations, mir.Operation{Kind: mir.OperationKind(255)})
	if err := ValidateStableCompilationArtifact(artifact); err != nil {
		t.Fatalf("fixture validation: %v", err)
	}
	_, err := EncodeStableCompilationArtifact(artifact, 1024)
	var marshalError *json.MarshalerError
	if !errors.As(err, &marshalError) || !strings.Contains(err.Error(), "unknown MIR operation kind 255") {
		t.Fatalf("marshal error: %v", err)
	}
}

func TestStableJSONLimitEscapedOversizeAllocation(t *testing.T) {
	artifact := stableCompilationDecodeFixture(t, 1)
	artifact.MIR.Functions[0].Function.InlineBody = strings.Repeat("<", 16<<20)
	const limit = 32 << 20
	for _, tc := range []struct {
		name string
		fn   func() error
		want error
	}{
		{"compilation", func() error { _, err := EncodeStableCompilationArtifact(artifact, limit); return err }, ErrStableCompilationTooLarge},
		{"MIR", func() error { _, err := EncodeStableArtifact(artifact.MIR, limit); return err }, ErrStableArtifactTooLarge},
		{"body", func() error { _, err := EncodeStableBody(artifact.MIR.Functions[0], limit); return err }, ErrBodyArtifactTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); !errors.Is(err, tc.want) {
				t.Fatalf("warmup: %v", err)
			}
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			err := tc.fn()
			runtime.ReadMemStats(&after)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error: %v", err)
			}
			if n := after.TotalAlloc - before.TotalAlloc; n > 2<<20 {
				t.Fatalf("escaped rejection allocated %d bytes", n)
			} else {
				t.Logf("escaped rejection allocated %d bytes", n)
			}
		})
	}
}

func TestStableJSONLimitTypeSpelling(t *testing.T) {
	deep := hir.TypeI32
	for i := 0; i < 150; i++ {
		deep = hir.Option(deep)
	}
	retained := hir.ParseType("[i32; " + strings.Repeat(" ", 1<<16) + "0001]")
	for _, typ := range []hir.Type{deep, retained, hir.ParseType("a::<\"&>"), hir.FixedArray(hir.TypeI32, 123456)} {
		raw, err := json.Marshal(typ)
		if err != nil {
			t.Fatal(err)
		}
		if exceeds, err := stableJSONExceedsLimit(typ, uint64(len(raw))); err != nil || exceeds {
			t.Fatalf("exact type bound: %v", err)
		}
		if exceeds, err := stableJSONExceedsLimit(typ, uint64(len(raw)-1)); err != nil || !exceeds {
			t.Fatalf("type undercount: %v", err)
		}
	}
}

type unexportedEmbeddedPlain struct{ Visible string }

type unsupportedStableJSON struct{}

func (unsupportedStableJSON) MarshalJSON() ([]byte, error) { return []byte(`"future"`), nil }
func TestStableJSONLimitRejectsUnknownSchema(t *testing.T) {
	for _, v := range []any{unsupportedStableJSON{}, struct{ Value map[string]int }{}, struct{ unsupportedStableJSON }{}, struct{ unexportedEmbeddedPlain }{}} {
		if _, err := stableJSONExceedsLimit(v, 1<<20); err == nil {
			t.Fatalf("unsupported schema %T accepted", v)
		}
	}
}

func TestStableJSONLimitKnownCustomExactBoundary(t *testing.T) {
	for _, v := range []any{FunctionID{}, ContextID{}, packagegraph.InstanceID{}, packagegraph.CoordinateID{}, contractstatemir.StateBodyHash{}, mir.OperationAssign, mir.Operand{Kind: mir.OperandByteString, Type: hir.TypeU8, StaticData: "<\x00\xff", I64: -32}, []byte{0, 1, 2, 3}, []byte{}, []byte(nil)} {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if over, err := stableJSONExceedsLimit(v, uint64(len(raw))); err != nil || over {
			t.Fatalf("%T exact: %v", v, err)
		}
		if over, err := stableJSONExceedsLimit(v, uint64(len(raw)-1)); err != nil || !over {
			t.Fatalf("%T undercount: %v", v, err)
		}
	}
}

package mircodec_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

type closureCodec struct {
	name, schema string
	encode       func(mir.Program, uint64) ([]byte, error)
	decode       func([]byte, uint64) (mir.Program, error)
}

func closureCodecs() []closureCodec {
	return []closureCodec{
		{"ordinary", mircodec.Schema, mircodec.Encode, mircodec.Decode},
		{"storage-transfer", mircodec.StorageTransferSchema, mircodec.EncodeStorageTransfers, mircodec.DecodeStorageTransfers},
	}
}

func closureCodecLocal(id mir.LocalID, typ hir.Type) mir.Operand {
	return mir.Operand{Kind: mir.OperandLocal, Local: id, Type: typ}
}

func closureCodecReturn(value mir.Operand) mir.Terminator {
	return mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: value, DestinationLocal: mir.NoLocal, UnwindSuccessor: mir.NoBlock}
}

// Explicit private MIR is the codec's input contract. No verifier is bypassed;
// source preparation and native backend execution have separate parent gates.
func closureCodecProgram(captures int) mir.Program {
	target := mir.Function{Symbol: 7, Name: "main::target", ClosureTarget: true, ClosureCaptures: uint8(captures), ResultType: hir.TypeI32, Entry: 0}
	values := []int32{math.MinInt32, -1, 0, math.MaxInt32}
	var operands []mir.Operand
	for i := 0; i <= captures; i++ {
		target.Parameters = append(target.Parameters, mir.LocalID(i+1))
		target.ParameterTypes = append(target.ParameterTypes, hir.TypeI32)
		if i < captures {
			operands = append(operands, mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: values[i]})
		}
	}
	target.Blocks = []mir.Block{{ID: 0, Terminator: closureCodecReturn(closureCodecLocal(mir.LocalID(captures+1), hir.TypeI32))}}
	main := mir.Function{Symbol: 8, Name: "main::main", ResultType: hir.TypeI32, Entry: 0, Blocks: []mir.Block{
		{ID: 0, Operations: []mir.Operation{{Kind: mir.OperationClosureCreate, Symbol: 7, DestinationLocal: 10, Type: hir.TypeClosureI32, Operands: operands}}, Terminator: mir.Terminator{Kind: mir.TerminatorClosureCall, Arguments: []mir.Operand{closureCodecLocal(10, hir.TypeClosureI32), {Kind: mir.OperandI32, I32: 42, Type: hir.TypeI32}}, DestinationLocal: 11, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}},
		{ID: 1, Terminator: closureCodecReturn(closureCodecLocal(11, hir.TypeI32))},
	}}
	return mir.Program{Functions: []mir.Function{target, main}}
}

func closureCodecWire(t *testing.T, schema, version string, program mir.Program) []byte {
	t.Helper()
	wire, err := json.Marshal(struct {
		Schema  string      `json:"schema"`
		Version string      `json:"mirVersion"`
		Program mir.Program `json:"program"`
	}{schema, version, program})
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestClosureValueCodecRoundTrip(t *testing.T) {
	for _, codec := range closureCodecs() {
		for _, captures := range []int{0, 4} {
			t.Run(fmt.Sprintf("%s/%d-captures", codec.name, captures), func(t *testing.T) {
				program := closureCodecProgram(captures)
				if failures := mir.Verify(program); len(failures) != 0 {
					t.Fatalf("parent verifier gate: %+v", failures)
				}
				encoded, err := codec.encode(program, artifactLimit)
				if err != nil {
					t.Fatal(err)
				}
				prefix := fmt.Sprintf(`{"schema":%q,"mirVersion":"fairyc-mir-closure-values/1","program":`, codec.schema)
				if !bytes.HasPrefix(encoded, []byte(prefix)) {
					t.Fatalf("closure version prefix: %s", encoded)
				}
				decoded, err := codec.decode(encoded, artifactLimit)
				if err != nil || !reflect.DeepEqual(decoded, program) {
					t.Fatalf("round trip lost closure records: %v", err)
				}
				again, err := codec.encode(decoded, artifactLimit)
				if err != nil || !bytes.Equal(encoded, again) {
					t.Fatalf("noncanonical round trip: %v", err)
				}
				outcome, err := interpreter.Execute(decoded, "main::main")
				if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
					t.Fatalf("restored MIR invocation: %+v %v", outcome, err)
				}
				// Only source-independent symbols, types and scalar operands cross the
				// boundary. Runtime closure slots, invoke addresses and environments do not.
				create := decoded.Functions[1].Blocks[0].Operations[0]
				if create.Symbol != 7 || len(create.Operands) != captures || !decoded.Functions[0].ClosureTarget || int(decoded.Functions[0].ClosureCaptures) != captures {
					t.Fatal("closure target/capture identity lost")
				}
				for _, operand := range create.Operands {
					if operand.Kind != mir.OperandI32 || operand.Type != hir.TypeI32 || operand.U64 != 0 || operand.StaticData != "" {
						t.Fatalf("capture is not a portable scalar: %+v", operand)
					}
				}
				for _, forbidden := range []string{`"NativePointer"`, `"InvokePointer"`, `"Environment"`, `"handle"`, `0x`} {
					if bytes.Contains(encoded, []byte(forbidden)) {
						t.Fatalf("runtime pointer state leaked: %s", forbidden)
					}
				}
				if _, err := codec.encode(program, uint64(len(encoded))); err != nil {
					t.Fatalf("exact encode limit: %v", err)
				}
				if _, err := codec.decode(encoded, uint64(len(encoded))); err != nil {
					t.Fatalf("exact decode limit: %v", err)
				}
				if _, err := codec.encode(program, uint64(len(encoded)-1)); !errors.Is(err, mircodec.ErrArtifactTooLarge) {
					t.Fatalf("encode limit: %v", err)
				}
				if _, err := codec.decode(encoded, uint64(len(encoded)-1)); !errors.Is(err, mircodec.ErrArtifactTooLarge) {
					t.Fatalf("decode limit: %v", err)
				}
				if _, err := codec.decode(append(append([]byte{}, encoded...), '\n'), artifactLimit); !errors.Is(err, mircodec.ErrNonCanonical) {
					t.Fatalf("whitespace canonical gate: %v", err)
				}
				unknown := bytes.Replace(encoded, []byte(`"ClosureTarget":true`), []byte(`"ClosureTarget":true,"NativePointer":1311768467463790320`), 1)
				if bytes.Equal(unknown, encoded) {
					t.Fatal("pointer injection fixture missing")
				}
				if _, err := codec.decode(unknown, artifactLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
					t.Fatalf("native pointer field accepted: %v", err)
				}
			})
		}
	}
}

func TestClosureValueCodecRejectsForgedMIR(t *testing.T) {
	mutations := []struct {
		name   string
		mutate func(*mir.Program)
	}{
		{"missing-target", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Symbol = 999 }},
		{"unmarked-target", func(p *mir.Program) { p.Functions[0].ClosureTarget = false }},
		{"wrong-target-result", func(p *mir.Program) { p.Functions[0].ResultType = hir.TypeI64 }},
		{"effectful-target", func(p *mir.Program) { p.Functions[0].Effects = []hir.Effect{"console"} }},
		{"five-target-captures", func(p *mir.Program) { p.Functions[0].ClosureCaptures = 5 }},
		{"five-construction-captures", func(p *mir.Program) {
			op := &p.Functions[1].Blocks[0].Operations[0]
			op.Operands = append(op.Operands, mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: 1})
		}},
		{"wrong-capture-type", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations[0].Operands[0] = mir.Operand{Kind: mir.OperandU32, Type: hir.TypeU32, U64: 1}
		}},
		{"pointer-in-capture", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Operands[0].U64 = 0x123456789abcdef0 }},
		{"callback-type", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Type = hir.TypeCCallbackI32UserData }},
		{"uninitialized-callable", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Arguments[0].Local = 99 }},
		{"scalar-callable", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Terminator.Arguments[0] = mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: 7}
		}},
		{"wrong-call-argument", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Arguments[1].Type = hir.TypeI64 }},
		{"extra-call-argument", func(p *mir.Program) {
			term := &p.Functions[1].Blocks[0].Terminator
			term.Arguments = append(term.Arguments, mir.Operand{Kind: mir.OperandI32, I32: 1})
		}},
		{"direct-target-on-value-call", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Callee = 7 }},
		{"unwind-call", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.UnwindSuccessor = 1 }},
	}
	for _, codec := range closureCodecs() {
		for _, tc := range mutations {
			t.Run(codec.name+"/"+tc.name, func(t *testing.T) {
				program := closureCodecProgram(4)
				tc.mutate(&program)
				if encoded, err := codec.encode(program, artifactLimit); err == nil || encoded != nil || !strings.Contains(err.Error(), "verify MIR before encoding") {
					t.Fatalf("forged encode: %s %v", encoded, err)
				}
				wire := closureCodecWire(t, codec.schema, mircodec.ClosureMIRVersion, program)
				if decoded, err := codec.decode(wire, artifactLimit); err == nil || !reflect.DeepEqual(decoded, mir.Program{}) || !strings.Contains(err.Error(), "verify decoded MIR") {
					t.Fatalf("forged decode: %+v %v", decoded, err)
				}
			})
		}
	}
}

package mircodec_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

func TestClosureValueCodecExactVersionGate(t *testing.T) {
	legacy := verifiedProgram(t)
	for _, codec := range closureCodecs() {
		t.Run(codec.name, func(t *testing.T) {
			for _, version := range []string{mircodec.MIRVersion, "fairyc-mir/1", "fairyc-mir-closure-values/0", "fairyc-mir-closure-values/2", "fairyc-mir-closure-values/1 "} {
				wire := closureCodecWire(t, codec.schema, version, closureCodecProgram(0))
				if decoded, err := codec.decode(wire, artifactLimit); err == nil || !reflect.DeepEqual(decoded, mir.Program{}) || !strings.Contains(err.Error(), "version") {
					t.Fatalf("closure under version %q: %+v %v", version, decoded, err)
				}
			}
			wire := closureCodecWire(t, codec.schema, mircodec.ClosureMIRVersion, legacy)
			if _, err := codec.decode(wire, artifactLimit); err == nil || !strings.Contains(err.Error(), "version") {
				t.Fatalf("feature-free closure version admitted: %v", err)
			}
			// Even an unused helper is a format feature. A type-only signature also
			// needs the new gate without a construction or invocation instruction.
			for _, program := range []mir.Program{
				{Functions: []mir.Function{closureCodecProgram(0).Functions[0]}},
				{Functions: []mir.Function{{Symbol: 1, Name: "main::type_only", Parameters: []mir.LocalID{1}, ParameterTypes: []hir.Type{hir.TypeClosureI32}, ResultType: hir.TypeClosureI32, Entry: 0, Blocks: []mir.Block{{ID: 0, Terminator: closureCodecReturn(closureCodecLocal(1, hir.TypeClosureI32))}}}}},
			} {
				encoded, err := codec.encode(program, artifactLimit)
				if err != nil {
					t.Fatalf("dormant closure feature: %v", err)
				}
				if !bytes.Contains(encoded, []byte(`"mirVersion":"fairyc-mir-closure-values/1"`)) {
					t.Fatal("dormant feature used legacy version")
				}
				if _, err := codec.decode(encoded, artifactLimit); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	ordinary, err := mircodec.Encode(closureCodecProgram(0), artifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mircodec.DecodeStorageTransfers(ordinary, artifactLimit); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("ordinary schema crossed storage boundary: %v", err)
	}
	storage, err := mircodec.EncodeStorageTransfers(closureCodecProgram(0), artifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mircodec.Decode(storage, artifactLimit); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("storage schema crossed ordinary boundary: %v", err)
	}
}

// These records deliberately need not be executable: version selection must
// close the legacy gate before a future verifier can gain wrapper admission.
func TestClosureValueCodecStructuralFeatureGate(t *testing.T) {
	locations := []struct {
		name  string
		place func(hir.Type) mir.Program
	}{
		{"struct-identity", func(typ hir.Type) mir.Program { return mir.Program{StructTypes: []mir.StructType{{Type: typ}}} }},
		{"struct-field", func(typ hir.Type) mir.Program {
			return mir.Program{StructTypes: []mir.StructType{{Fields: []mir.StructField{{Type: typ}}}}}
		}},
		{"enum-identity", func(typ hir.Type) mir.Program { return mir.Program{EnumTypes: []mir.EnumType{{Type: typ}}} }},
		{"enum-payload", func(typ hir.Type) mir.Program {
			return mir.Program{EnumTypes: []mir.EnumType{{Variants: []mir.EnumVariant{{Fields: []mir.EnumField{{Type: typ}}}}}}}
		}},
		{"resource-identity", func(typ hir.Type) mir.Program { return mir.Program{ResourceTypes: []mir.ResourceType{{Type: typ}}} }},
		{"resource-field", func(typ hir.Type) mir.Program {
			return mir.Program{ResourceTypes: []mir.ResourceType{{Fields: []mir.ResourceField{{Type: typ}}}}}
		}},
		{"foreign-result", func(typ hir.Type) mir.Program {
			return mir.Program{ForeignFunctions: []mir.ForeignFunction{{ResultType: typ}}}
		}},
		{"foreign-parameter", func(typ hir.Type) mir.Program {
			return mir.Program{ForeignFunctions: []mir.ForeignFunction{{ParameterTypes: []hir.Type{typ}}}}
		}},
		{"function-result", func(typ hir.Type) mir.Program { return mir.Program{Functions: []mir.Function{{ResultType: typ}}} }},
		{"function-parameter", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{ParameterTypes: []hir.Type{typ}}}}
		}},
		{"function-resource-local", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Resources: []mir.ResourceLocal{{Type: typ}}}}}
		}},
		{"drop-glue", func(typ hir.Type) mir.Program { return mir.Program{Functions: []mir.Function{{DropGlueFor: typ}}} }},
		{"operation-type", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Operations: []mir.Operation{{Type: typ}}}}}}}
		}},
		{"operation-owner", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Operations: []mir.Operation{{OwnerType: typ}}}}}}}
		}},
		{"operation-index", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Operations: []mir.Operation{{IndexType: typ}}}}}}}
		}},
		{"operation-operand", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Operations: []mir.Operation{{Operands: []mir.Operand{{Type: typ}}}}}}}}}
		}},
		{"return-operand", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Terminator: mir.Terminator{ReturnValue: mir.Operand{Type: typ}}}}}}}
		}},
		{"switch-type", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Terminator: mir.Terminator{SwitchType: typ}}}}}}
		}},
		{"call-operand", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Terminator: mir.Terminator{Arguments: []mir.Operand{{Type: typ}}}}}}}}
		}},
		{"switch-operand", func(typ hir.Type) mir.Program {
			return mir.Program{Functions: []mir.Function{{Blocks: []mir.Block{{Terminator: mir.Terminator{SwitchOperands: []mir.Operand{{Type: typ}}}}}}}}
		}},
	}
	shapes := []struct {
		name string
		typ  hir.Type
	}{
		{"direct", hir.TypeClosureI32},
		{"option", hir.Option(hir.TypeClosureI32)},
		{"result-success", hir.Result(hir.TypeClosureI32, hir.TypeU32)},
		{"result-failure", hir.Result(hir.TypeU32, hir.TypeClosureI32)},
		{"borrow", hir.SharedBorrow(hir.TypeClosureI32)},
		{"mutable-borrow", hir.MutableBorrow(hir.TypeClosureI32)},
		{"pointer", hir.RawPointer(hir.TypeClosureI32, false)},
		{"slice", hir.Slice(hir.TypeClosureI32)},
		{"array", hir.FixedArray(hir.TypeClosureI32, 2)},
		{"nested", hir.Result(hir.Option(hir.FixedArray(hir.TypeI32, 2)), hir.SharedBorrow(hir.Option(hir.TypeClosureI32)))},
		{"noncanonical-wrapper-text", hir.ParseType("Option< fn i32(i32) >")},
	}
	for _, codec := range closureCodecs() {
		for _, location := range locations {
			for _, shape := range shapes {
				t.Run(codec.name+"/"+location.name+"/"+shape.name, func(t *testing.T) {
					wire := closureCodecWire(t, codec.schema, mircodec.MIRVersion, location.place(shape.typ))
					if _, err := codec.decode(wire, artifactLimit); err == nil || !strings.Contains(err.Error(), "version") {
						t.Fatalf("legacy structural closure gate: %v", err)
					}
				})
			}
		}
	}
	for _, codec := range closureCodecs() {
		for name, program := range map[string]mir.Program{
			"target":   {Functions: []mir.Function{{ClosureTarget: true}}},
			"captures": {Functions: []mir.Function{{ClosureCaptures: 1}}},
			"create":   {Functions: []mir.Function{{Blocks: []mir.Block{{Operations: []mir.Operation{{Kind: mir.OperationClosureCreate}}}}}}},
			"call":     {Functions: []mir.Function{{Blocks: []mir.Block{{Terminator: mir.Terminator{Kind: mir.TerminatorClosureCall}}}}}},
		} {
			t.Run(codec.name+"/marker/"+name, func(t *testing.T) {
				if _, err := codec.decode(closureCodecWire(t, codec.schema, mircodec.MIRVersion, program), artifactLimit); err == nil || !strings.Contains(err.Error(), "version") {
					t.Fatalf("legacy marker gate: %v", err)
				}
			})
		}
	}
}

func TestClosureValueCodecLegacyBytesStayExact(t *testing.T) {
	// Reconstruct the previous envelope independently, explicitly omitting the
	// new function metadata regardless of its current production JSON tags.
	type oldFunction struct {
		*mir.Function
		ClosureTarget   *bool  `json:",omitempty"`
		ClosureCaptures *uint8 `json:",omitempty"`
	}
	type oldProgram struct {
		*mir.Program
		Functions []oldFunction
	}
	for _, parameter := range []hir.Type{hir.TypeI32, hir.TypeCCallbackI32UserData, hir.RawPointer(hir.TypeI32, true)} {
		program := mir.Program{Functions: []mir.Function{{Symbol: 1, Name: "main::main", Parameters: []mir.LocalID{1}, ParameterTypes: []hir.Type{parameter}, ResultType: hir.TypeI32, Entry: 0, Blocks: []mir.Block{{ID: 0, Terminator: closureCodecReturn(mir.Operand{Kind: mir.OperandI32, I32: 42})}}}}}
		if parameter == hir.TypeCCallbackI32UserData {
			// The existing callback contract permits this final foreign callback /
			// user-data pair, not a general internal callback parameter.
			program.Functions[0].ParameterTypes[0] = hir.TypeI32
			program.ForeignFunctions = []mir.ForeignFunction{{Symbol: 2, Name: "ffi::apply", LinkName: "apply", ABI: "C", ParameterTypes: []hir.Type{parameter, hir.RawPointer(hir.TypeVoid, true)}, ResultType: hir.TypeVoid}}
		}
		legacy := oldProgram{Program: &program, Functions: []oldFunction{{Function: &program.Functions[0]}}}
		for _, codec := range closureCodecs() {
			t.Run(fmt.Sprintf("%s/%s", codec.name, parameter), func(t *testing.T) {
				want, err := json.Marshal(struct {
					Schema  string     `json:"schema"`
					Version string     `json:"mirVersion"`
					Program oldProgram `json:"program"`
				}{codec.schema, "fairyc-mir/2", legacy})
				if err != nil {
					t.Fatal(err)
				}
				encoded, err := codec.encode(program, artifactLimit)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(encoded, want) {
					t.Fatalf("legacy wire changed\n got %s\nwant %s", encoded, want)
				}
				if bytes.Contains(encoded, []byte(`ClosureTarget`)) || bytes.Contains(encoded, []byte(`ClosureCaptures`)) {
					t.Fatal("zero closure metadata changed legacy bytes")
				}
				if _, err := codec.decode(want, artifactLimit); err != nil {
					t.Fatalf("old bytes rejected: %v", err)
				}
				// Source strings that resemble a closure never select its format.
				program.Functions[0].Name = "main::fn i32(i32) closure-create"
				if mir.HasCopiedClosureValues(program) {
					t.Fatal("text resembles a closure but carries no structured feature")
				}
				if _, err := codec.decode(closureCodecWire(t, codec.schema, mircodec.ClosureMIRVersion, program), artifactLimit); err == nil || !strings.Contains(err.Error(), "version") {
					t.Fatalf("source text enabled closure format: %v", err)
				}
				if _, err := codec.encode(program, 0); err == nil {
					t.Fatal("zero encode limit admitted")
				}
				if _, err := codec.decode(encoded, 0); err == nil {
					t.Fatal("zero decode limit admitted")
				}
				if _, err := codec.decode(append(append([]byte{}, encoded...), []byte(` {}`)...), artifactLimit); err == nil || errors.Is(err, mircodec.ErrNonCanonical) {
					t.Fatalf("trailing value admitted: %v", err)
				}
			})
		}
	}
}

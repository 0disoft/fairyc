package packageartifact

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestClosureValuesLegacyPackageFormatsStayClosed(t *testing.T) {
	_, original, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(original, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	program := mir.Clone(original)
	var symbol hir.SymbolID
	for _, fn := range program.Functions {
		symbol = max(symbol, fn.Symbol)
	}
	owner := program.Functions[0].Span
	program.Functions = append(program.Functions, mir.Function{
		Symbol: symbol + 1, Name: "main::closure_target", Span: owner,
		ClosureTarget: true, Parameters: []mir.LocalID{0}, ParameterTypes: []hir.Type{hir.TypeI32}, ResultType: hir.TypeI32,
		Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: 0}}}},
	})
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("whole-program closure fixture must verify before the legacy format gate: %v", failures)
	}
	if _, err := ExtractAll(program, identities, coordinates); !errors.Is(err, ErrClosureValuesUnsupported) {
		t.Fatalf("extract all: %v", err)
	}
	if _, err := Extract(program, owner.Package, identities, coordinates); !errors.Is(err, ErrClosureValuesUnsupported) {
		t.Fatalf("extract package: %v", err)
	}
	published := 0
	err = VisitAllContext(t.Context(), program, identities, coordinates, func(Artifact) error { published++; return nil })
	if !errors.Is(err, ErrClosureValuesUnsupported) || published != 0 {
		t.Fatalf("legacy visitor published %d artifacts before refusal: %v", published, err)
	}
	codec, err := NewProcessCodec(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		good, err := codec.Encode(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := codec.Decode(good); err != nil {
			t.Fatal(err)
		}
		bad := cloneArtifactForTest(artifact)
		bad.Functions[0].Body.ClosureTarget = true
		if err := Verify(bad); !errors.Is(err, ErrClosureValuesUnsupported) {
			t.Fatalf("verify closure marker in legacy package: %v", err)
		}
		if data, err := codec.Encode(bad); !errors.Is(err, ErrClosureValuesUnsupported) || data != nil {
			t.Fatalf("encode closure marker: bytes=%d error=%v", len(data), err)
		}
		raw, err := json.Marshal(processEnvelope{Schema: ProcessCodecSchema, Version: ProcessCodecVersion, Process: codec.Identity(), Artifact: bad})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := codec.Decode(raw); !errors.Is(err, ErrClosureValuesUnsupported) {
			t.Fatalf("decode forged legacy package: %v", err)
		}
		stable, err := DescribeStableArtifact(artifact, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		body := stable.Functions[0]
		body.Function.ClosureTarget = true
		if err := ValidateStableBody(body); !errors.Is(err, ErrClosureValuesUnsupported) {
			t.Fatalf("stable body marker: %v", err)
		}
		if data, err := EncodeStableBody(body, 1<<20); !errors.Is(err, ErrClosureValuesUnsupported) || data != nil {
			t.Fatalf("stable body encode: bytes=%d error=%v", len(data), err)
		}
	}
	context := mir.Program{StructTypes: []mir.StructType{{Type: hir.ParseType("main::Box"), Name: "Box", Fields: []mir.StructField{{Name: "action", Type: hir.TypeClosureI32}}}}}
	if _, err := DescribeContext(context, coordinates, nil); !errors.Is(err, ErrClosureValuesUnsupported) {
		t.Fatalf("closure-bearing legacy context: %v", err)
	}
}

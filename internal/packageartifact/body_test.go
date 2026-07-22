package packageartifact

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const stableBodyLimit = 8 << 20

func TestStableLeafBodyRoundTripsRebindsAndExecutesAcrossFreshNumbers(t *testing.T) {
	t.Parallel()
	_, program, identities := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities)
	if err != nil {
		t.Fatal(err)
	}
	record, context := callFreeI32Record(t, artifacts)
	body, err := DescribeStableLeafBody(record, context, identities, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableBody(body, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeStableBody(encoded, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, body) {
		t.Fatal("decoded stable body differs from source")
	}

	freshIdentities := map[hir.PackageID]packagegraph.InstanceID{77: body.Package}
	rebound, err := RebindStableLeafBody(decoded, context, freshIdentities, 900, 500, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	reboundProgram := mir.Program{Functions: []mir.Function{rebound}}
	if failures := mir.Verify(reboundProgram); len(failures) != 0 {
		t.Fatalf("rebound stable leaf failed MIR verification: %v", failures[0])
	}
	originalOutcome, err := interpreter.Execute(mir.Program{Functions: []mir.Function{record.Body}}, record.Body.Name)
	if err != nil {
		t.Fatal(err)
	}
	reboundOutcome, err := interpreter.Execute(reboundProgram, rebound.Name)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reboundOutcome, originalOutcome) || reboundOutcome.Kind != interpreter.OutcomeReturn {
		t.Fatalf("rebound execution = %+v, original = %+v", reboundOutcome, originalOutcome)
	}

	freshRecord := FunctionRecord{ID: record.ID, Descriptor: record.Descriptor, Body: rebound}
	freshBody, err := DescribeStableLeafBody(freshRecord, context, freshIdentities, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	freshBytes, err := EncodeStableBody(freshBody, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, freshBytes) {
		t.Fatal("stable body bytes changed after package, function-symbol, and local-ID rebinding")
	}
}

func TestStableLeafBodyCodecRejectsBoundsUnknownFieldsAndNoncanonicalBytes(t *testing.T) {
	t.Parallel()
	_, program, identities := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities)
	if err != nil {
		t.Fatal(err)
	}
	record, context := callFreeI32Record(t, artifacts)
	body, err := DescribeStableLeafBody(record, context, identities, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableBody(body, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeStableBody(body, uint64(len(encoded)-1)); !errors.Is(err, ErrBodyArtifactTooLarge) {
		t.Fatalf("small encode bound error = %v", err)
	}
	if _, err := DecodeStableBody(encoded, uint64(len(encoded)-1)); !errors.Is(err, ErrBodyArtifactTooLarge) {
		t.Fatalf("small decode bound error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1)
	if _, err := DecodeStableBody(unknown, stableBodyLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	withWhitespace := append(append([]byte(nil), encoded...), '\n')
	if _, err := DecodeStableBody(withWhitespace, stableBodyLimit); !errors.Is(err, ErrBodyArtifactEncoding) {
		t.Fatalf("noncanonical error = %v", err)
	}
	corrupt := body
	corrupt.Function.Symbol = 7
	if _, err := EncodeStableBody(corrupt, stableBodyLimit); err == nil || !strings.Contains(err.Error(), "build-local function identity") {
		t.Fatalf("build-local symbol error = %v", err)
	}
}

func TestStableLeafBodyRejectsCallsAndContextDrift(t *testing.T) {
	t.Parallel()
	_, program, identities := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities)
	if err != nil {
		t.Fatal(err)
	}
	var called FunctionRecord
	var calledContext ContextDescriptor
	found := false
	for _, artifact := range artifacts {
		for _, record := range artifact.Functions {
			if slices.ContainsFunc(record.Body.Blocks, func(block mir.Block) bool { return block.Terminator.Kind == mir.TerminatorCall }) {
				called, calledContext, found = record, artifact.StableContext, true
				break
			}
		}
	}
	if !found {
		t.Fatal("workspace has no function containing a call")
	}
	if _, err := DescribeStableLeafBody(called, calledContext, identities, stableBodyLimit); err == nil || !strings.Contains(err.Error(), "does not admit calls") {
		t.Fatalf("call rejection error = %v", err)
	}

	record, context := callFreeI32Record(t, artifacts)
	body, err := DescribeStableLeafBody(record, context, identities, stableBodyLimit)
	if err != nil {
		t.Fatal(err)
	}
	drifted := cloneContextDescriptor(context)
	drifted.Structs = append(drifted.Structs, ContextStruct{Type: "pkg[" + body.Package.String() + "]::drift::Type", Name: "Type"})
	slices.SortFunc(drifted.Structs, func(left, right ContextStruct) int { return strings.Compare(left.Type, right.Type) })
	if _, err := RebindStableLeafBody(body, drifted, map[hir.PackageID]packagegraph.InstanceID{77: body.Package}, 900, 500, stableBodyLimit); err == nil || !strings.Contains(err.Error(), "context does not match") {
		t.Fatalf("context drift error = %v", err)
	}
}

func callFreeI32Record(t *testing.T, artifacts []Artifact) (FunctionRecord, ContextDescriptor) {
	t.Helper()
	for _, artifact := range artifacts {
		for _, record := range artifact.Functions {
			if len(record.Body.Parameters) != 0 || record.Body.ResultType != hir.TypeI32 || len(record.Body.Resources) != 0 {
				continue
			}
			hasCall := slices.ContainsFunc(record.Body.Blocks, func(block mir.Block) bool { return block.Terminator.Kind == mir.TerminatorCall })
			if !hasCall {
				return record, artifact.StableContext
			}
		}
	}
	t.Fatal("workspace has no call-free zero-parameter i32 function")
	return FunctionRecord{}, ContextDescriptor{}
}

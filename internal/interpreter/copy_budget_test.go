package interpreter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func copyBudgetState(limit int) *executionState {
	return &executionState{context: context.Background(), copyUnits: limit}
}

func requireCopyLimit(t *testing.T, err error) {
	t.Helper()
	var capability CapabilityError
	if !errors.As(err, &capability) || capability.Kind != LimitCopy {
		t.Fatalf("error %v is not copy-limit", err)
	}
}

func largeNestedCopyValue(t *testing.T) runtimeValue {
	t.Helper()
	innerDefinition := &mir.StructType{Type: hir.ParseType("test::Inner"), Fields: make([]mir.StructField, 16)}
	innerValues := make([]runtimeValue, len(innerDefinition.Fields))
	for index := range innerDefinition.Fields {
		innerDefinition.Fields[index] = mir.StructField{Name: "value", Type: hir.TypeI32}
		innerValues[index] = i32Value(int32(index + 1))
	}
	inner, err := structValue(innerDefinition, innerValues)
	if err != nil {
		t.Fatal(err)
	}
	middleDefinition := &mir.StructType{Type: hir.ParseType("test::Middle"), Fields: []mir.StructField{
		{Name: "left", Type: inner.typ}, {Name: "right", Type: inner.typ},
	}}
	middle, err := structValue(middleDefinition, []runtimeValue{inner, inner})
	if err != nil {
		t.Fatal(err)
	}
	elements := make([]runtimeValue, 128)
	for index := range elements {
		elements[index] = middle
	}
	return runtimeValue{kind: valueArray, typ: hir.Slice(middle.typ), payload: &arrayState{elements: elements}}
}

func TestCopyUnitsBelowExactAndAboveBudget(t *testing.T) {
	definition := &mir.StructType{Type: hir.ParseType("test::Wide"), Fields: make([]mir.StructField, 64)}
	values := make([]runtimeValue, len(definition.Fields))
	for index := range definition.Fields {
		definition.Fields[index] = mir.StructField{Name: "value", Type: hir.TypeI32}
		values[index] = i32Value(int32(index))
	}
	original, err := structValue(definition, values)
	if err != nil {
		t.Fatal(err)
	}
	measurement := copyBudgetState(DefaultMaxCopyUnits)
	if _, err := measurement.cloneForValueCopy(original); err != nil {
		t.Fatal(err)
	}
	needed := DefaultMaxCopyUnits - measurement.copyUnits
	const explicitCost = 1 + 1 + 1 + 64 + 64 // visits, state, slice, slots, scalar visits
	if needed != explicitCost {
		t.Fatalf("copy cost=%d, want %d", needed, explicitCost)
	}

	if _, err := copyBudgetState(needed - 1).cloneForValueCopy(original); err == nil {
		t.Fatal("copy below measured cost succeeded")
	} else {
		requireCopyLimit(t, err)
	}
	for _, limit := range []int{needed, needed + 1} {
		state := copyBudgetState(limit)
		copy, err := state.cloneForValueCopy(original)
		if err != nil {
			t.Fatalf("limit=%d: %v", limit, err)
		}
		if copy.structure() == original.structure() || copy.structure().fields[0].i32() != 0 {
			t.Fatalf("limit=%d: copy did not detach storage", limit)
		}
	}
}

func TestLargeNestedCopyDetachesStorageWithinBudget(t *testing.T) {
	original := largeNestedCopyValue(t)
	const initialBudget = 100_000
	state := copyBudgetState(initialBudget)
	copy, err := state.cloneForValueCopy(original)
	if err != nil {
		t.Fatal(err)
	}
	if state.copyUnits >= initialBudget {
		t.Fatal("large nested copy consumed no units")
	}
	copyMid := copy.array().elements[0]
	originalMid := original.array().elements[0]
	if copyMid.structure() == originalMid.structure() || copyMid.structure().fields[0].structure() == originalMid.structure().fields[0].structure() {
		t.Fatal("large nested copy shares aggregate storage")
	}
	copyMid.structure().fields[0].structure().fields[0] = i32Value(999)
	if originalMid.structure().fields[0].structure().fields[0].i32() != 1 {
		t.Fatal("large nested copy mutated its source")
	}
}

type copyCheckpointContext struct {
	context.Context
	cancel   context.CancelFunc
	checks   int
	cancelAt int
}

func (ctx *copyCheckpointContext) Err() error {
	ctx.checks++
	if ctx.checks == ctx.cancelAt {
		ctx.cancel()
	}
	return ctx.Context.Err()
}

func TestCopyCancellationIsDeterministicDuringTraversal(t *testing.T) {
	active, cancel := context.WithCancel(context.Background())
	ctx := &copyCheckpointContext{Context: active, cancel: cancel, cancelAt: 3}
	state := &executionState{context: ctx, copyUnits: DefaultMaxCopyUnits}
	if _, err := state.cloneForValueCopy(largeNestedCopyValue(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("copy cancellation: %v", err)
	}
	if ctx.checks != 3 {
		t.Fatalf("cancellation checkpoint count=%d, want 3", ctx.checks)
	}
}

func TestCyclicAggregateCopyFailsBeforeStackExhaustion(t *testing.T) {
	definition := &mir.StructType{Type: hir.ParseType("test::Cycle"), Fields: []mir.StructField{{Name: "next", Type: hir.TypeI32}}}
	cycleStruct := runtimeValue{kind: valueStruct, typ: definition.Type, payload: &structState{definition: definition}}
	cycleStruct.structure().fields = []runtimeValue{cycleStruct}
	cycleArray := runtimeValue{kind: valueArray, typ: hir.Slice(hir.TypeI32), payload: &arrayState{}}
	cycleArray.array().elements = []runtimeValue{cycleArray}
	for name, value := range map[string]runtimeValue{"struct": cycleStruct, "array": cycleArray} {
		t.Run(name, func(t *testing.T) {
			_, err := copyBudgetState(DefaultMaxCopyUnits).cloneForValueCopy(value)
			requireCopyLimit(t, err)
		})
	}
}

func TestFailedCopiesLeaveDestinationsUnchanged(t *testing.T) {
	innerDefinition := &mir.StructType{Type: hir.ParseType("test::Inner"), Fields: []mir.StructField{{Name: "value", Type: hir.TypeI32}}}
	outerDefinition := &mir.StructType{Type: hir.ParseType("test::Outer"), Fields: []mir.StructField{{Name: "child", Type: innerDefinition.Type}}}
	inner, err := structValue(innerDefinition, []runtimeValue{i32Value(1)})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := structValue(outerDefinition, []runtimeValue{inner})
	if err != nil {
		t.Fatal(err)
	}
	source, err := structValue(outerDefinition, []runtimeValue{inner})
	if err != nil {
		t.Fatal(err)
	}
	sourceInner, err := structValue(innerDefinition, []runtimeValue{i32Value(2)})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("field-store", func(t *testing.T) {
		originalChild := owner.structure().fields[0]
		locals := map[mir.LocalID]runtimeValue{0: owner, 1: sourceInner}
		operation := mir.Operation{Kind: mir.OperationFieldStore, Symbol: hir.NoSymbol, DestinationLocal: mir.NoLocal,
			OwnerType: owner.typ, Type: sourceInner.typ, Field: "child",
			Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}, {Kind: mir.OperandLocal, Local: 1}}}
		structs := map[hir.Type]*mir.StructType{innerDefinition.Type: innerDefinition, outerDefinition.Type: outerDefinition}
		err := storeField(copyBudgetState(2), locals, operation, structs, &aggregateFieldIndex{})
		requireCopyLimit(t, err)
		if owner.structure().fields[0].structure() != originalChild.structure() {
			t.Fatal("failed field copy published a replacement")
		}
	})

	t.Run("deref-store", func(t *testing.T) {
		originalChild := owner.structure().fields[0]
		receiver := owner
		receiver.typ = hir.MutableBorrow(owner.typ)
		locals := map[mir.LocalID]runtimeValue{0: receiver, 1: source}
		operation := mir.Operation{Kind: mir.OperationDerefStore, Symbol: hir.NoSymbol, DestinationLocal: mir.NoLocal,
			OwnerType: receiver.typ, Type: owner.typ,
			Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}, {Kind: mir.OperandLocal, Local: 1}}}
		structs := map[hir.Type]*mir.StructType{innerDefinition.Type: innerDefinition, outerDefinition.Type: outerDefinition}
		err := storeDeref(copyBudgetState(2), locals, operation, structs)
		requireCopyLimit(t, err)
		if owner.structure().fields[0].structure() != originalChild.structure() {
			t.Fatal("failed dereference copy published replacement fields")
		}
	})

	t.Run("indexed-store", func(t *testing.T) {
		initial := i64Value(7)
		view := runtimeValue{kind: valueView, typ: hir.MutableBorrow(hir.Slice(hir.TypeI64)),
			payload: &indexedViewState{array: &arrayState{elements: []runtimeValue{initial}}, length: 1, mutable: true}}
		err := copyBudgetState(0).storeIndexed(view, 0, i64Value(42))
		requireCopyLimit(t, err)
		if got := view.view().array.elements[0].i64(); got != 7 {
			t.Fatalf("failed indexed copy stored %d", got)
		}
	})
}

func TestTaggedExtractionCopyHonorsBudgetAndMoveSemantics(t *testing.T) {
	payload := largeNestedCopyValue(t)
	result, err := resultValue("ok", payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := copyBudgetState(1).resultPayload(result, "ok", false); err == nil {
		t.Fatal("tagged copy extraction ignored copy budget")
	} else {
		requireCopyLimit(t, err)
	}
	if result.tagged().payload == nil {
		t.Fatal("failed tagged copy extraction cleared its payload")
	}
	moved, err := copyBudgetState(0).resultPayload(result, "ok", true)
	if err != nil || moved.kind != valueArray || result.tagged().payload != nil {
		t.Fatalf("tagged move behavior changed: %+v, %v", moved, err)
	}
}

func TestArrayConstructionPrechargesSliceAndSlotsBeforeCopying(t *testing.T) {
	arrayType := hir.FixedArray(hir.TypeI32, 2)
	function := &mir.Function{Name: "main", ResultType: hir.TypeI32, Entry: 0, Blocks: []mir.Block{{
		ID: 0,
		Operations: []mir.Operation{{
			Kind: mir.OperationArrayConstruct, DestinationLocal: 1, Type: arrayType, OwnerType: hir.TypeI32, Operator: "list",
			Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 7}, {Kind: mir.OperandI32, I32: 9}},
		}},
		Terminator: mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: 1}},
	}}}
	for budget := 2; budget <= 4; budget++ {
		state := copyBudgetState(budget)
		state.steps, state.callDepthLimit = 100, DefaultMaxCallDepth
		_, err := state.call(function, nil, 0)
		if err == nil || !strings.Contains(err.Error(), "fixed-array construction") {
			t.Fatalf("budget=%d: production array construction error=%v", budget, err)
		}
		requireCopyLimit(t, err)
	}
	state := copyBudgetState(5)
	state.steps, state.callDepthLimit = 100, DefaultMaxCallDepth
	completion, err := state.call(function, nil, 0)
	if err != nil || completion.value.array() == nil || completion.value.array().elements[0].i32() != 7 {
		t.Fatalf("exact array construction budget: %+v, %v", completion, err)
	}
}

func TestFailedRegionResultCopyDoesNotPublishPlacement(t *testing.T) {
	stored := copyTestStruct(t)
	borrow := hir.SharedBorrow(stored.typ)
	region := &runtimeRegion{active: true, placements: []runtimeValue{i32Value(1)}}
	if _, err := copyBudgetState(0).placeRegionValue(region, stored, borrow); err == nil {
		t.Fatal("Region placement result copy ignored budget")
	} else {
		requireCopyLimit(t, err)
	}
	if len(region.placements) != 1 || region.placements[0].i32() != 1 {
		t.Fatal("failed result copy published a Region placement")
	}
	placed, err := copyBudgetState(1).placeRegionValue(region, stored, borrow)
	if err != nil {
		t.Fatal(err)
	}
	if len(region.placements) != 2 || region.placements[1].typ != stored.typ || placed.tagged().payload.typ != borrow {
		t.Fatal("successful Region placement changed stored or result type identity")
	}
}

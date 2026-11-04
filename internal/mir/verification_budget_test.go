package mir

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/hir"
)

func reportMarkerCount(failures []VerificationError) int {
	count := 0
	for _, failure := range failures {
		if failure.Code == CodeVerificationReportLimit {
			count++
		}
	}
	return count
}

func TestVerificationReportsInclusiveLimitAndNestedMerges(t *testing.T) {
	for _, count := range []int{255, 256, 257, 4096} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var reports []VerificationError
			for i := range count {
				reports = appendVerificationErrors(reports, VerificationError{Code: CodeOperationKind, Entity: fmt.Sprint(i)})
			}
			want := min(count, DefaultVerificationReportLimit)
			if count > DefaultVerificationReportLimit {
				want++
			}
			if len(reports) != want || verificationReportsFull(reports) != (count > DefaultVerificationReportLimit) || reportMarkerCount(reports) != max(0, want-DefaultVerificationReportLimit) {
				t.Fatalf("report limit: %d reports, %d markers", len(reports), reportMarkerCount(reports))
			}
			if verificationReportsFull(reports) {
				parent := []VerificationError{{Code: CodeControlFlow, Entity: "parent"}}
				parent = appendVerificationErrors(parent, reports...)
				parent = appendVerificationErrors(parent, VerificationError{Code: CodeControlFlow, Entity: "omitted"})
				if len(parent) != DefaultVerificationReportLimit+1 || reportMarkerCount(parent) != 1 || !verificationReportsFull(parent) {
					t.Fatal("child overflow lost, duplicated or expanded during parent merge")
				}
			}
		})
	}
}

func invalidOperationsProgram(count int) Program {
	return Program{Functions: []Function{{
		Symbol: 1, Name: "main", ResultType: hir.TypeI32, Entry: 0,
		Blocks: []Block{{ID: 0, Operations: make([]Operation, count), Terminator: Terminator{
			Kind: TerminatorReturn, UnwindSuccessor: NoBlock,
			ReturnValue: Operand{Kind: OperandI32, Type: hir.TypeI32},
		}}},
	}}}
}

func TestVerificationReportsBoundOperationsAcrossFunctions(t *testing.T) {
	for _, count := range []int{255, 256, 257, 4096} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			t.Parallel()
			failures := Verify(invalidOperationsProgram(count))
			want := min(count, DefaultVerificationReportLimit)
			markers := 0
			if count > DefaultVerificationReportLimit {
				want++
				markers++
			}
			if len(failures) != want || reportMarkerCount(failures) != markers {
				t.Fatalf("%d invalid operations: %d failures, %d markers", count, len(failures), reportMarkerCount(failures))
			}
		})
	}
	program := Program{}
	for i := range 40 {
		fn := invalidOperationsProgram(10).Functions[0]
		fn.Symbol, fn.Name = hir.SymbolID(i+1), fmt.Sprintf("f%d", i)
		for j := range fn.Blocks[0].Operations {
			fn.Blocks[0].Operations[j].Entity = fmt.Sprintf("f%d:operation:%d", i, j)
		}
		program.Functions = append(program.Functions, fn)
	}
	if failures := Verify(program); len(failures) != DefaultVerificationReportLimit+1 || reportMarkerCount(failures) != 1 {
		t.Fatalf("per-function reports bypassed program bound: %d failures", len(failures))
	}
}

func TestVerificationReportsBoundInnerHelpersAndDeclarations(t *testing.T) {
	operands := make([]Operand, 4096)
	for i := range operands {
		operands[i] = Operand{Kind: OperandLocal, Local: LocalID(i), Type: hir.TypeI32}
	}
	if failures := verifyOperands(verificationState{}, operands, "input"); len(failures) != DefaultVerificationReportLimit+1 || reportMarkerCount(failures) != 1 {
		t.Fatalf("operand errors expanded before parent merge: %d", len(failures))
	}
	program := Program{Functions: make([]Function, 4096)}
	for i := range program.Functions {
		program.Functions[i].Name = fmt.Sprintf("missing-symbol-%d", i)
	}
	if failures := Verify(program); len(failures) != DefaultVerificationReportLimit+1 || reportMarkerCount(failures) != 1 {
		t.Fatalf("declaration errors expanded before parent merge: %d", len(failures))
	}
	structure := StructType{Type: hir.ParseType("$p1::main::S"), Name: "S", Fields: make([]StructField, 4096)}
	if failures := Verify(Program{StructTypes: []StructType{structure}}); len(failures) > DefaultVerificationReportLimit+1 || reportMarkerCount(failures) != 1 {
		t.Fatalf("aggregate helper overflow lost after deduplication: %v", failures)
	}
}

func TestVerificationReportsPreserveSmallCanonicalErrorsAndCancellation(t *testing.T) {
	want := []VerificationError{{Code: "V1701", Entity: "z"}, {Code: "V1001", Entity: "a"}, {Code: "V1001", Entity: "a"}}
	got := appendVerificationErrors(nil, want...)
	if !reflect.DeepEqual(canonicalVerificationErrors(got), canonicalVerificationErrors(want)) {
		t.Fatal("ordinary canonical ordering or duplicate removal changed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if failures, err := VerifyWithCapabilitiesContext(ctx, invalidOperationsProgram(4096), admission.Capabilities{}); !errors.Is(err, context.Canceled) || len(failures) != 0 {
		t.Fatalf("cancellation became a truncated error report: %v, %v", failures, err)
	}
}

func TestVerificationReportsTruncationIsDeterministic(t *testing.T) {
	for _, kind := range []string{"struct", "enum", "resource-exit"} {
		t.Run(kind, func(t *testing.T) {
			program := invalidOperationsProgram(0)
			resourceType := hir.ParseType("main::Resource")
			for i := range 300 {
				typ := hir.ParseType(fmt.Sprintf("main::Type%03d", i))
				switch kind {
				case "struct":
					program.StructTypes = append(program.StructTypes, StructType{Type: typ, Name: typ.String()})
					program.ResourceTypes = append(program.ResourceTypes, ResourceType{Type: typ, DropKind: hir.ResourceDropEmpty})
				case "enum":
					program.EnumTypes = append(program.EnumTypes, EnumType{Type: typ, Name: typ.String()})
					program.ResourceTypes = append(program.ResourceTypes, ResourceType{Type: typ, DropKind: hir.ResourceDropEmpty})
				case "resource-exit":
					program.Functions[0].Resources = append(program.Functions[0].Resources, ResourceLocal{
						Symbol: hir.SymbolID(i + 2), Type: resourceType, Parameter: true, Entity: fmt.Sprintf("resource:%03d", i),
					})
				}
			}
			if kind == "resource-exit" {
				program.ResourceTypes = []ResourceType{{Type: resourceType, DropKind: hir.ResourceDropEmpty}}
				// Repeated records must retain the map's last-definition selection.
				program.Functions[0].Resources = append(program.Functions[0].Resources, program.Functions[0].Resources[0])
			}
			want := Verify(program)
			if len(want) != DefaultVerificationReportLimit+1 || reportMarkerCount(want) != 1 {
				t.Fatalf("expected truncated %s reports, got %v", kind, want)
			}
			for range 8 {
				if got := Verify(program); !reflect.DeepEqual(got, want) {
					t.Fatal("map iteration changed the retained failure prefix")
				}
			}
		})
	}
	regions := map[hir.SymbolID]string{0: ""}
	for i := range 300 {
		regions[hir.SymbolID(i+1)] = fmt.Sprintf("region:%03d", i)
	}
	want := verifyRegionExits(regions)
	if len(want) != DefaultVerificationReportLimit+1 || want[0].Entity != "region:000" || want[255].Entity != "region:255" || reportMarkerCount(want) != 1 {
		t.Fatal("region exits did not retain a bounded canonical prefix")
	}
	for range 8 {
		if got := verifyRegionExits(regions); !reflect.DeepEqual(got, want) {
			t.Fatal("unordered region state changed truncated reports")
		}
	}
	if got := verifyRegionExits(map[hir.SymbolID]string{0: ""}); len(got) != 0 {
		t.Fatal("caller-owned region incorrectly needs reset")
	}
}

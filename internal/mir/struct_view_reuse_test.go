package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func repeatedStructViewFixture(operations, fields int) (Function, typeProperties) {
	typ := hir.ParseType("main::Record")
	definition := &StructType{Type: typ, Name: "Record", Fields: make([]StructField, fields)}
	for i := range definition.Fields {
		definition.Fields[i] = StructField{Name: fmt.Sprint(i), Type: hir.TypeI32}
	}
	view := hir.SharedBorrow(hir.Slice(typ))
	function := Function{Name: "main::views", ResultType: view, ParameterTypes: []hir.Type{view},
		Blocks: []Block{{ID: 0, Operations: make([]Operation, operations)}}}
	for i := range function.Blocks[0].Operations {
		function.Blocks[0].Operations[i] = Operation{Kind: OperationAssign, Type: view, OwnerType: view,
			Operands: []Operand{{Type: view}}}
	}
	return function, typeProperties{structs: map[hir.Type]*StructType{typ: definition}}
}

func TestStructViewRepeatedFailuresRetainDiagnostics(t *testing.T) {
	for _, mutation := range []string{"empty", "missing", "pointer", "cycle"} {
		t.Run(mutation, func(t *testing.T) {
			function, properties := repeatedStructViewFixture(4, 1)
			definition := properties.structs[hir.ParseType("main::Record")]
			switch mutation {
			case "empty":
				definition.Fields = nil
			case "missing":
				delete(properties.structs, definition.Type)
			case "pointer":
				definition.Fields[0].Type = hir.RawPointer(hir.TypeI32, true)
			case "cycle":
				definition.Fields[0].Type = definition.Type
			}
			failures := verifyStructViewTypes(&function, properties)
			// Result, parameter, and three occurrences in each operation.
			if len(failures) != 14 {
				t.Fatalf("diagnostics lost: %+v", failures)
			}
			for _, failure := range failures {
				if failure.Code != CodeOperationType || failure.Entity != function.Name {
					t.Fatalf("diagnostic identity changed: %+v", failure)
				}
			}
		})
	}
}

func TestStructViewResultsDoNotOutliveVerification(t *testing.T) {
	function, properties := repeatedStructViewFixture(4, 1)
	definition := properties.structs[hir.ParseType("main::Record")]
	for _, valid := range []bool{true, false, true} {
		definition.Fields[0].Type = hir.TypeI32
		if !valid {
			definition.Fields[0].Type = hir.SharedBorrow(hir.TypeI32)
		}
		failures := verifyStructViewTypes(&function, properties)
		if valid && len(failures) != 0 || !valid && len(failures) != 14 {
			t.Fatalf("stale result, valid=%t: %+v", valid, failures)
		}
	}
}

func TestStructViewRepeatedTypesKeepDistinctAdmissionRules(t *testing.T) {
	function, properties := repeatedStructViewFixture(0, 1)
	view := function.ResultType
	record := hir.ParseType("main::Record")
	function.ResultType, function.ParameterTypes = hir.TypeVoid, nil
	for _, typ := range []hir.Type{view, hir.Slice(record), hir.FixedArray(record, hir.MaxExecutableFixedArrayElements+1), hir.FixedArray(record, 1), view} {
		function.Blocks[0].Operations = append(function.Blocks[0].Operations, Operation{Type: typ})
	}
	if failures := verifyStructViewTypes(&function, properties); len(failures) != 2 {
		t.Fatalf("bare struct slice and oversized struct array must still fail: %+v", failures)
	}
}

func TestStructViewWrappedAndUnreachableTypes(t *testing.T) {
	function, properties := repeatedStructViewFixture(0, 1)
	view := function.ResultType
	for _, wrap := range []struct {
		name string
		typ  hir.Type
	}{
		{"view", view},
		{"option", hir.Option(view)},
		{"result-success", hir.Result(view, hir.TypeI32)},
		{"result-failure", hir.Result(hir.TypeI32, view)},
		{"pointer", hir.RawPointer(view, true)},
	} {
		t.Run(wrap.name, func(t *testing.T) {
			function.ResultType = hir.TypeVoid
			function.ParameterTypes = nil
			function.Blocks = []Block{{ID: 0}, {ID: 1, Operations: []Operation{{Type: wrap.typ}, {Type: wrap.typ}}}}
			definition := properties.structs[hir.ParseType("main::Record")]
			definition.Fields[0].Type = hir.TypeI32
			if failures := verifyStructViewTypes(&function, properties); len(failures) != 0 {
				t.Fatal(failures)
			}
			definition.Fields[0].Type = hir.TypeRegion
			if failures := verifyStructViewTypes(&function, properties); len(failures) != 2 {
				t.Fatalf("invalid unreachable view accepted: %+v", failures)
			}
		})
	}
}

func BenchmarkRepeatedStructViewVerification(b *testing.B) {
	for _, fields := range []int{0, 256} {
		b.Run(fmt.Sprintf("fields=%d", fields), func(b *testing.B) {
			function, properties := repeatedStructViewFixture(4096, fields)
			if fields == 0 {
				function.ResultType, function.ParameterTypes[0] = hir.TypeI32, hir.TypeI32
				for i := range function.Blocks[0].Operations {
					op := &function.Blocks[0].Operations[i]
					op.Type, op.OwnerType, op.Operands[0].Type = hir.TypeI32, hir.TypeI32, hir.TypeI32
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if failures := verifyStructViewTypes(&function, properties); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}

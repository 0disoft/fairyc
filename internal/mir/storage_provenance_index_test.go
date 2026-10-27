package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/hir"
)

func storageProvenanceFixture(size int) Function {
	function := Function{Symbol: 1, Name: "main::storage", Entry: 0, ResultType: hir.TypeVoid,
		Parameters: []LocalID{0}, ParameterTypes: []hir.Type{hir.RawPointer(hir.TypeVoid, true)},
		Blocks: []Block{{ID: 0, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}}}
	for i := range size {
		span := hir.Span{Package: hir.DefaultPackage, Source: "main.fy", StartByte: i * 2, EndByte: i*2 + 1}
		scope := hir.Span{Package: span.Package, Source: span.Source, EndByte: size*2 + 2}
		op := Operation{Kind: OperationStorageTake, Entity: sourceEntity("storage-take", span), Span: span,
			UnsafeScope: scope, Symbol: hir.NoSymbol, DestinationLocal: LocalID(i + 1), Type: hir.TypeI32,
			OwnerType: hir.RawPointer(hir.TypeVoid, true), Operands: []Operand{{Kind: OperandLocal, Local: 0}}}
		function.Blocks[0].Operations = append(function.Blocks[0].Operations, op)
		function.UnsafeOperations = append(function.UnsafeOperations, UnsafeOperation{Entity: op.Entity,
			SourceSpanEntity: sourceEntity("span", span), ScopeEntity: sourceEntity("unsafe-scope", scope)})
	}
	return function
}

func storageProvenanceFailures(function *Function) []VerificationError {
	return verifyStorageOperations(function, typeProperties{}, admission.Capabilities{Storage: admission.StorageTransfers})
}

func TestStorageTransferProvenanceExactMatchCount(t *testing.T) {
	for _, count := range []int{0, 1, 2, 257} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			function := storageProvenanceFixture(1)
			entry := function.UnsafeOperations[0]
			function.UnsafeOperations = make([]UnsafeOperation, count)
			for i := range count {
				function.UnsafeOperations[i] = entry
			}
			failures := storageProvenanceFailures(&function)
			if count == 1 && len(failures) != 0 || count != 1 &&
				(len(failures) != 1 || failures[0].Code != CodeStorageProvenance) {
				t.Fatalf("matches=%d: %+v", count, failures)
			}
		})
	}
}

func TestStorageTransferProvenanceSpanAndRecordIdentity(t *testing.T) {
	for _, name := range []string{"entity", "empty-scope", "package", "source", "start", "end", "record-span", "record-scope", "unrelated-record"} {
		t.Run(name, func(t *testing.T) {
			function := storageProvenanceFixture(1)
			op := &function.Blocks[0].Operations[0]
			switch name {
			case "entity":
				op.Entity = "forged"
			case "empty-scope":
				op.UnsafeScope.Source = ""
			case "package":
				op.UnsafeScope.Package++
			case "source":
				op.UnsafeScope.Source = "other.fy"
			case "start":
				op.UnsafeScope.StartByte = op.Span.StartByte + 1
			case "end":
				op.UnsafeScope.EndByte = op.Span.EndByte - 1
			case "record-span":
				function.UnsafeOperations[0].SourceSpanEntity = "wrong"
			case "record-scope":
				function.UnsafeOperations[0].ScopeEntity = "wrong"
			case "unrelated-record":
				entry := function.UnsafeOperations[0]
				entry.ScopeEntity = "different scope"
				function.UnsafeOperations = append(function.UnsafeOperations, entry)
			}
			failures := storageProvenanceFailures(&function)
			if name == "unrelated-record" {
				if len(failures) != 0 {
					t.Fatal(failures)
				}
			} else if len(failures) != 1 || failures[0].Code != CodeStorageProvenance || failures[0].Entity != op.Entity {
				t.Fatalf("unexpected diagnostic: %+v", failures)
			}
		})
	}
}

func TestStorageTransferProvenanceUnreachableAndFreshVerification(t *testing.T) {
	for _, kind := range []OperationKind{OperationStorageTake, OperationStoragePut, OperationStorageBorrow, OperationStorageBorrowMut, OperationStorageElementAddress} {
		t.Run(kind.String(), func(t *testing.T) {
			function := storageProvenanceFixture(1)
			op := function.Blocks[0].Operations[0]
			op.Kind = kind
			op.Entity = sourceEntity(kind.String(), op.Span)
			function.UnsafeOperations[0].Entity = op.Entity
			function.Blocks[0].Operations = nil
			function.Blocks = append(function.Blocks, Block{ID: 1, Operations: []Operation{op}, Terminator: Terminator{Kind: TerminatorUnreachable}})
			for _, duplicate := range []bool{false, true} {
				if duplicate {
					function.UnsafeOperations = append(function.UnsafeOperations, function.UnsafeOperations[0])
				}
				count := 0
				for _, failure := range storageProvenanceFailures(&function) {
					if failure.Code == CodeStorageProvenance {
						count++
					}
				}
				if count != 0 && !duplicate || count != 1 && duplicate {
					t.Fatalf("duplicate=%t: provenance failures=%d", duplicate, count)
				}
			}
		})
	}
}

func BenchmarkStorageTransferProvenance(b *testing.B) {
	for _, size := range []int{256, 4096} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			function := storageProvenanceFixture(size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if failures := storageProvenanceFailures(&function); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}

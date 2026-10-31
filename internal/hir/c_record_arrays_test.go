package hir

import "testing"

func TestCRecordArrayRequiresResolvedAcyclicRecords(t *testing.T) {
	record, leaf := ParseType("test::Record"), ParseType("test::Leaf")
	for name, graph := range map[string]map[Type][]Type{
		"scalar":         {record: {TypeF64, TypeCLong}},
		"nested":         {record: {leaf, leaf}, leaf: {TypeI32}},
		"nested-arrays":  {record: {FixedArray(leaf, 2)}, leaf: {FixedArray(TypeF32, 3)}},
		"cycle":          {record: {FixedArray(record, 2)}},
		"indirect-cycle": {record: {leaf}, leaf: {record}},
		"empty":          {record: {}},
		"missing":        {record: {leaf}},
		"bool":           {record: {TypeBool}},
		"array-of-array": {record: {FixedArray(FixedArray(TypeI32, 2), 3)}},
	} {
		t.Run(name, func(t *testing.T) {
			fields := func(typ Type) ([]Type, bool) { v, ok := graph[typ]; return v, ok }
			want := name == "scalar" || name == "nested" || name == "nested-arrays"
			for _, length := range []uint64{1, 4096, 4097} {
				if got := CRecordArray(FixedArray(record, length), fields, nil); got != (want && length <= 4096) {
					t.Fatalf("length=%d got=%v", length, got)
				}
			}
		})
	}
}

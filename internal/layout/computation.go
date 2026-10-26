package layout

import "fmt"

// A calculation owns its cache: neither mutable inputs nor returned field slices
// are retained by Engine between Of calls, and concurrent calls remain isolated.
type layoutComputation struct {
	*Engine
	results map[layoutKey]Result
}

// Backend descriptors share array element pointers and aggregate field slices.
// Include all Type metadata, the slice length, and depth: a result computed near
// the root must not hide a depth-limit error when reused on a deeper path.
type layoutKey struct {
	kind       Kind
	name       string
	element    *Type
	length     uint64
	fields     *Field
	fieldCount int
	depth      int
}

func (e *layoutComputation) layout(value Type, depth int) (Result, error) {
	if depth >= maxLayoutDepth {
		return Result{}, Error{Code: ErrorDepthLimit, Detail: fmt.Sprintf("layout nesting exceeds %d levels", maxLayoutDepth)}
	}
	// The root is visited once. Scalars need no cache, so flat aggregates keep
	// their original allocation cost.
	if depth == 0 || value.Kind != KindStruct && value.Kind != KindUnion && value.Kind != KindArray {
		return e.compute(value, depth)
	}
	key := layoutKey{kind: value.Kind, name: value.Name, element: value.Element,
		length: value.Length, fieldCount: len(value.Fields), depth: depth}
	if len(value.Fields) > 0 {
		key.fields = &value.Fields[0]
	}
	if result, found := e.results[key]; found {
		return result, nil
	}
	result, err := e.compute(value, depth)
	if err == nil {
		if e.results == nil {
			e.results = make(map[layoutKey]Result)
		}
		e.results[key] = result
	}
	return result, err
}

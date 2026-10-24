package mir

// A checked operation has at most three operands (view slicing). Marker owner
// and result types and categories are excluded: a matching but malformed marker still
// owns the diagnostic, rather than disappearing from the candidate list.
type checkedMarkerKey struct {
	kind     OperationKind
	operator string
	count    int
	operands [3]Operand
}

type checkedMarker struct {
	block     *Block
	operation int
}

type checkedMarkerIndex map[checkedMarkerKey][]checkedMarker

func markerKey(kind OperationKind, operator string, operands []Operand) (checkedMarkerKey, bool) {
	key := checkedMarkerKey{kind: kind, operator: operator, count: len(operands)}
	if len(operands) > len(key.operands) {
		return key, false
	}
	copy(key.operands[:], operands)
	return key, true
}

func indexCheckedMarkers(function *Function) checkedMarkerIndex {
	index := make(checkedMarkerIndex)
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex, operation := range block.Operations {
			switch operation.Kind {
			case OperationArithmeticCheck, OperationCheckedDivision, OperationArrayIndexCheck, OperationViewIndexCheck, OperationViewSliceCheck:
			default:
				continue
			}
			key, valid := markerKey(operation.Kind, operation.Operator, operation.Operands)
			if !valid {
				continue
			}
			candidates := index[key]
			marker := checkedMarker{block: block, operation: operationIndex}
			if last := len(candidates) - 1; last >= 0 && candidates[last].block == block {
				// Preserve the reverse scan's last matching marker per block.
				candidates[last] = marker
			} else {
				candidates = append(candidates, marker)
			}
			index[key] = candidates
		}
	}
	return index
}

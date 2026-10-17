package hir

import (
	"encoding/json"
	"fmt"
)

// StatementKind is the closed HIR statement vocabulary.
type StatementKind uint8

const (
	StatementInvalid StatementKind = iota
	StatementAssert
	StatementAssignment
	StatementBinding
	StatementBreak
	StatementBurn

	StatementContinue
	StatementDefer
	StatementDiscard
	StatementExpressionStatement

	StatementFor

	StatementRegion
	StatementReturn
	StatementWhile
)

var statementNames = [...]string{"", "assert", "assignment", "binding", "break", "burn", "continue", "defer", "discard", "expression-statement", "for", "region", "return", "while"}

func (k StatementKind) String() string {
	if int(k) < len(statementNames) {
		return statementNames[k]
	}
	return fmt.Sprintf("StatementKind(%d)", k)
}
func (k StatementKind) MarshalJSON() ([]byte, error) { return json.Marshal(k.String()) }
func (k *StatementKind) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	for i, name := range statementNames {
		if s != nil && *s == name {
			*k = StatementKind(i)
			return nil
		}
	}
	return fmt.Errorf("unknown HIR statement kind %s", data)
}

// ExpressionKind is the closed HIR expression vocabulary.
type ExpressionKind uint8

const (
	ExpressionInvalid ExpressionKind = iota
	ExpressionArray
	ExpressionArrayIndex
	ExpressionArrayLength
	ExpressionArrayViewBorrow
	ExpressionBinary
	ExpressionBlock
	ExpressionBorrowedViewFromRawParts
	ExpressionCall
	ExpressionCast
	ExpressionEnumVariant
	ExpressionFieldBorrow
	ExpressionFieldExpression

	ExpressionGroup
	ExpressionIf

	ExpressionLiteral
	ExpressionMatch
	ExpressionNominalUnwrap
	ExpressionNominalWrap
	ExpressionOmittedRangeBound
	ExpressionOptionVariant
	ExpressionPath
	ExpressionPrefix
	ExpressionPropagate
	ExpressionRecord
	ExpressionRegion
	ExpressionResultVariant
	ExpressionScalarCast
	ExpressionSharedReborrow
	ExpressionStorageBorrow
	ExpressionStorageBorrowMut
	ExpressionStorageElementAddress
	ExpressionStorageLayout
	ExpressionStoragePut
	ExpressionStorageTake
	ExpressionUnsafe
	ExpressionViewElementBorrow
	ExpressionViewEmpty
	ExpressionViewIndex
	ExpressionViewLength
	ExpressionViewSlice
	ExpressionWhile
)

var expressionNames = [...]string{"", "array", "array-index", "array-length", "array-view-borrow", "binary", "block", "borrowed-view-from-raw-parts", "call", "cast", "enum-variant", "field-borrow", "field-expression", "group", "if", "literal", "match", "nominal-unwrap", "nominal-wrap", "omitted-range-bound", "option-variant", "path", "prefix", "propagate", "record", "region", "result-variant", "scalar-cast", "shared-reborrow", "storage-borrow", "storage-borrow-mut", "storage-element-address", "storage-layout", "storage-put", "storage-take", "unsafe", "view-element-borrow", "view-empty", "view-index", "view-length", "view-slice", "while"}

func (k ExpressionKind) String() string {
	if int(k) < len(expressionNames) {
		return expressionNames[k]
	}
	return fmt.Sprintf("ExpressionKind(%d)", k)
}
func (k ExpressionKind) MarshalJSON() ([]byte, error) { return json.Marshal(k.String()) }
func (k *ExpressionKind) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	for i, name := range expressionNames {
		if s != nil && *s == name {
			*k = ExpressionKind(i)
			return nil
		}
	}
	return fmt.Errorf("unknown HIR expression kind %s", data)
}

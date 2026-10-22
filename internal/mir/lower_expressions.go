package mir

import (
	"encoding/hex"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (l *functionLowerer) lowerExpression(expression *hir.Expression) (Operand, error) {
	if err := l.contextError(); err != nil {
		return Operand{}, err
	}
	if expression == nil {
		return Operand{}, nil
	}
	switch expression.Kind {
	case hir.ExpressionLiteral:
		if expression.Type == hir.SharedBorrow(hir.TypeStr) && strings.HasPrefix(expression.Value, "\"") {
			decoded, err := hex.DecodeString(expression.Cooked)
			if err != nil || !utf8.Valid(decoded) {
				return Operand{}, fmt.Errorf("invalid cooked UTF-8 string literal %q", expression.Value)
			}
			return Operand{Kind: OperandString, Type: expression.Type, StaticData: string(decoded)}, nil
		}
		if expression.Type == hir.SharedBorrow(hir.TypeByteSlice) && strings.HasPrefix(expression.Value, "b\"") {
			decoded, err := hex.DecodeString(expression.Cooked)
			if err != nil {
				return Operand{}, fmt.Errorf("invalid cooked byte-string literal %q", expression.Value)
			}
			return Operand{Kind: OperandByteString, Type: expression.Type, StaticData: string(decoded)}, nil
		}
		if expression.Type == hir.SharedBorrow(hir.TypeCStr) && strings.HasPrefix(expression.Value, "c\"") {
			decoded, err := hex.DecodeString(expression.Cooked)
			if err != nil || !validCStringStaticData(string(decoded)) {
				return Operand{}, fmt.Errorf("invalid cooked C string literal %q", expression.Value)
			}
			return Operand{Kind: OperandCString, Type: expression.Type, StaticData: string(decoded)}, nil
		}
		if expression.Type == hir.TypeBool {
			switch expression.Value {
			case "true":
				return Operand{Kind: OperandI32, I32: 1, Type: hir.TypeBool}, nil
			case "false":
				return Operand{Kind: OperandI32, I32: 0, Type: hir.TypeBool}, nil
			default:
				return Operand{}, fmt.Errorf("invalid bool literal %q", expression.Value)
			}
		}
		switch expression.Type {
		case hir.TypeI64:
			value, err := parseSigned(expression.Value, 64)
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandI64, I64: value}, nil
		case hir.TypeI32:
			value, err := parseI32(expression.Value)
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandI32, I32: value}, nil
		case hir.TypeU8:
			value, err := parseUnsigned(expression.Value, 8, "u8")
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandU8, U64: value}, nil
		case hir.TypeU32:
			value, err := parseUnsigned(expression.Value, 32, "u32")
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandU32, U64: value}, nil
		case hir.TypeUSize:
			value, err := parseUnsigned(expression.Value, 64, "usize")
			if err != nil {
				return Operand{}, err
			}
			return Operand{Kind: OperandUSize, U64: value}, nil
		default:
			if _, _, pointer := hir.RawPointerElement(expression.Type); pointer && expression.Value == "null" {
				return Operand{Kind: OperandNull, Type: expression.Type}, nil
			}
			return Operand{}, fmt.Errorf("unsupported literal %q of type %q", expression.Value, expression.Type)
		}
	case hir.ExpressionPath:
		return Operand{Kind: OperandLocal, Local: LocalID(expression.Symbol)}, nil
	case hir.ExpressionGroup:
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("group expression has %d children", len(expression.Children))
		}
		return l.lowerExpression(&expression.Children[0])
	case hir.ExpressionRecord:
		return l.lowerAggregateConstruction(expression)
	case hir.ExpressionArray:
		return l.lowerArrayConstruction(expression)
	case hir.ExpressionArrayViewBorrow:
		return l.lowerArrayBorrowView(expression)
	case hir.ExpressionArrayLength:
		return l.lowerArrayLength(expression)
	case hir.ExpressionStorageLayout:
		return l.lowerStorageLayout(expression)
	case hir.ExpressionStorageTake:
		return l.lowerStorageTake(expression)
	case hir.ExpressionStoragePut:
		return l.lowerStoragePut(expression)
	case hir.ExpressionStorageBorrow, hir.ExpressionStorageBorrowMut:
		return l.lowerStorageBorrow(expression)
	case hir.ExpressionStorageElementAddress:
		return l.lowerStorageElementAddress(expression)
	case hir.ExpressionFieldExpression:
		return l.lowerFieldRead(expression)
	case hir.ExpressionFieldBorrow:
		return l.lowerFieldBorrow(expression)
	case hir.ExpressionViewLength:
		return l.lowerViewLength(expression)
	case hir.ExpressionViewEmpty:
		return l.lowerViewEmpty(expression)
	case hir.ExpressionViewIndex, hir.ExpressionViewElementBorrow:
		return l.lowerViewIndex(expression)
	case hir.ExpressionArrayIndex:
		return l.lowerArrayIndex(expression)
	case hir.ExpressionViewSlice:
		return l.lowerViewSlice(expression)
	case hir.ExpressionScalarCast:
		return l.lowerScalarCast(expression)
	case hir.ExpressionSharedReborrow:
		return l.lowerSharedReborrow(expression)
	case hir.ExpressionCast:
		return l.lowerRawAddressCast(expression)
	case hir.ExpressionBorrowedViewFromRawParts:
		return l.lowerBorrowedViewFromRawParts(expression)
	case hir.ExpressionPrefix:
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("prefix expression has %d children", len(expression.Children))
		}
		operand, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		if expression.Operator == "-" {
			l.lowerCheckedArithmetic(expression, []Operand{operand}, "checked-unary", []string{"integer-overflow"})
		}
		destination := l.newTemporary()
		l.appendOperation(Operation{
			Kind: OperationUnary, Span: expression.Span, DestinationLocal: destination,
			Operands: []Operand{operand}, OwnerType: expression.Children[0].Type,
			Type: expression.Type, Operator: expression.Operator,
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	case hir.ExpressionCall:
		if (expression.Operator == "place" || expression.Operator == "alloc_zeroed_bytes" || expression.Operator == "copy_bytes") && len(expression.Children) == 2 && expression.Children[0].Type == hir.TypeRegion {
			return l.lowerRegionPlace(expression)
		}
		return l.lowerCall(expression)
	case hir.ExpressionUnsafe:
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("unsafe expression has %d children", len(expression.Children))
		}
		return l.lowerExpression(&expression.Children[0])
	case hir.ExpressionResultVariant:
		return l.lowerResultVariant(expression)
	case hir.ExpressionOptionVariant:
		return l.lowerOptionVariant(expression)
	case hir.ExpressionEnumVariant:
		return l.lowerEnumVariant(expression)
	case hir.ExpressionBinary:
		if len(expression.Children) != 2 {
			return Operand{}, fmt.Errorf("binary expression has %d children", len(expression.Children))
		}
		if expression.Operator == "&&" || expression.Operator == "||" {
			return l.lowerShortCircuit(expression)
		}
		left, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		// Capture the read before a later operand can mutate its source local.
		if _, sourceLocal := l.bindings[hir.SymbolID(left.Local)]; left.Kind == OperandLocal && sourceLocal {
			left = l.freezeScalarOperand(left, expression.Children[0].Span)
		}
		right, err := l.lowerExpression(&expression.Children[1])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		return l.lowerBinaryOperands(expression, left, right)
	case hir.ExpressionPropagate:
		return l.lowerPropagate(expression)
	case hir.ExpressionIf:
		return l.lowerIf(expression)
	case hir.ExpressionMatch:
		return l.lowerMatch(expression)
	case hir.ExpressionWhile:
		return l.lowerWhile(expression)
	case hir.ExpressionBlock:
		if expression.Block == nil {
			return Operand{}, fmt.Errorf("block expression has no block")
		}
		value, _, err := l.lowerNestedBlock(*expression.Block, true)
		return value, err
	case hir.ExpressionRegion:
		return l.lowerRegion(expression)
	default:
		return Operand{}, fmt.Errorf("unsupported HIR expression %q", expression.Kind)
	}
}

func (l *functionLowerer) lowerBinaryOperands(expression *hir.Expression, left, right Operand) (Operand, error) {
	if expression == nil || len(expression.Children) != 2 || expression.Type == hir.TypeInvalid {
		return Operand{}, fmt.Errorf("malformed binary operation")
	}
	if expression.Operator == "/" || expression.Operator == "%" {
		if err := l.lowerCheckedDivision(expression, left, right); err != nil {
			return Operand{}, err
		}
	} else {
		switch expression.Operator {
		case "+", "-", "*":
			l.lowerCheckedArithmetic(expression, []Operand{left, right}, "checked-arithmetic", []string{"integer-overflow"})
		case ">>":
			l.lowerCheckedArithmetic(expression, []Operand{left, right}, "checked-shift", []string{"invalid-shift"})
		case "<<":
			l.lowerCheckedArithmetic(expression, []Operand{left, right}, "checked-shift", []string{"invalid-shift", "integer-overflow"})
		}
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationBinary, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{left, right}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func expressionSpan(expression *hir.Expression, fallback hir.Span) hir.Span {
	if expression == nil {
		return fallback
	}
	return expression.Span
}

func containsCall(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if expression.Kind == hir.ExpressionCall {
		return true
	}
	for index := range expression.Children {
		if containsCall(&expression.Children[index]) {
			return true
		}
	}
	for index := range expression.Arms {
		if containsCall(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func parseI32(text string) (int32, error) {
	value, err := parseSigned(text, 32)
	return int32(value), err
}

func parseSigned(text string, width int) (int64, error) {
	clean := strings.ReplaceAll(text, "_", "")
	base := 10
	digits := clean
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		base, digits = 16, clean[2:]
	} else if strings.HasPrefix(clean, "0b") || strings.HasPrefix(clean, "0B") {
		base, digits = 2, clean[2:]
	}
	value, err := strconv.ParseInt(digits, base, width)
	if err != nil {
		return 0, fmt.Errorf("lower i%d literal %q: %w", width, text, err)
	}
	return value, nil
}

func parseUnsigned(text string, bitSize int, typeName string) (uint64, error) {
	clean := strings.ReplaceAll(text, "_", "")
	base := 10
	digits := clean
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		base, digits = 16, clean[2:]
	} else if strings.HasPrefix(clean, "0b") || strings.HasPrefix(clean, "0B") {
		base, digits = 2, clean[2:]
	}
	value, err := strconv.ParseUint(digits, base, bitSize)
	if err != nil {
		return 0, fmt.Errorf("lower %s literal %q: %w", typeName, text, err)
	}
	return value, nil
}

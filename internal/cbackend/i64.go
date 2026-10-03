package cbackend

import (
	"fmt"
	"math"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
)

type i64Helpers struct {
	bitcast     bool
	shift       bool
	addOverflow bool
	subOverflow bool
	mulOverflow bool
}

// A right-shift helper also calls the bitcast helper. Emit only helpers used by
// the inspected MIR, so strict C consumers never see unused static functions.
func (helpers i64Helpers) needsBitcast() bool {
	return helpers.bitcast || helpers.shift
}

func (helpers i64Helpers) emit(output *strings.Builder) {
	if helpers.needsBitcast() {
		output.WriteString(`
static inline int64_t fy_bits_to_i64(uint64_t bits) {
    int64_t value;
    memcpy(&value, &bits, sizeof value);
    return value;
}
`)
	}
	if helpers.shift {
		output.WriteString(`
static inline int64_t fy_sar_i64(int64_t value, int64_t amount) {
    uint64_t bits = (uint64_t)value;
    if (amount == 0 || value >= 0) return fy_bits_to_i64(bits >> (uint64_t)amount);
    return fy_bits_to_i64((bits >> (uint64_t)amount) | (UINT64_MAX << (64u - (uint64_t)amount)));
}
`)
	}
	if helpers.addOverflow {
		output.WriteString(`
static inline int fy_i64_add_overflow(int64_t left, int64_t right) {
    return (right > 0 && left > INT64_MAX - right) || (right < 0 && left < INT64_MIN - right);
}
`)
	}
	if helpers.subOverflow {
		output.WriteString(`
static inline int fy_i64_sub_overflow(int64_t left, int64_t right) {
    return (right < 0 && left > INT64_MAX + right) || (right > 0 && left < INT64_MIN + right);
}
`)
	}
	if helpers.mulOverflow {
		output.WriteString(`
static inline int fy_i64_mul_overflow(int64_t left, int64_t right) {
    if (left > 0) return right > 0 ? left > INT64_MAX / right : right < INT64_MIN / left;
    if (left < 0) return right > 0 ? left < INT64_MIN / right : right < 0 && left < INT64_MAX / right;
    return 0;
}
`)
	}
}

func cI64(value int64) string {
	if value == math.MinInt64 {
		return "INT64_MIN"
	}
	if value < 0 {
		return fmt.Sprintf("(-INT64_C(%d))", -value)
	}
	return fmt.Sprintf("INT64_C(%d)", value)
}

func emitI64Binary(output *strings.Builder, operation mir.Operation, left, right string) {
	destination := operation.DestinationLocal
	switch operation.Operator {
	case "+", "-", "*", "&", "^", "|", "<<":
		fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i64((uint64_t)(%s) %s (uint64_t)(%s));\n", destination, left, operation.Operator, right)
	case ">>":
		fmt.Fprintf(output, "    fy_l_%d = fy_sar_i64(%s, %s);\n", destination, left, right)
	case "/", "%":
		fmt.Fprintf(output, "    fy_checked_divisor_i64 = %s;\n    fy_l_%d = (%s) %s fy_checked_divisor_i64;\n", right, destination, left, operation.Operator)
	case "==", "!=", "<", "<=", ">", ">=":
		fmt.Fprintf(output, "    fy_l_%d = ((%s) %s (%s)) ? INT32_C(1) : INT32_C(0);\n", destination, left, operation.Operator, right)
	}
}

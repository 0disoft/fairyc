package chainprimitives

import (
	"math"
	"math/bits"
)

// AddU64Checked implements Fairy C's ordinary checked unsigned addition.
func AddU64Checked(left, right uint64) (uint64, error) {
	result, carry := bits.Add64(left, right, 0)
	if carry != 0 {
		return 0, ErrOverflow
	}
	return result, nil
}

// SubU64Checked implements Fairy C's ordinary checked unsigned subtraction.
func SubU64Checked(left, right uint64) (uint64, error) {
	result, borrow := bits.Sub64(left, right, 0)
	if borrow != 0 {
		return 0, ErrUnderflow
	}
	return result, nil
}

// MulU64Checked implements Fairy C's ordinary checked unsigned multiplication.
func MulU64Checked(left, right uint64) (uint64, error) {
	high, low := bits.Mul64(left, right)
	if high != 0 {
		return 0, ErrOverflow
	}
	return low, nil
}

func AddU64Wrapping(left, right uint64) uint64 { return left + right }
func SubU64Wrapping(left, right uint64) uint64 { return left - right }
func MulU64Wrapping(left, right uint64) uint64 { return left * right }

func AddU64Saturating(left, right uint64) uint64 {
	result, carry := bits.Add64(left, right, 0)
	if carry != 0 {
		return math.MaxUint64
	}
	return result
}

func SubU64Saturating(left, right uint64) uint64 {
	result, borrow := bits.Sub64(left, right, 0)
	if borrow != 0 {
		return 0
	}
	return result
}

func MulU64Saturating(left, right uint64) uint64 {
	high, low := bits.Mul64(left, right)
	if high != 0 {
		return math.MaxUint64
	}
	return low
}

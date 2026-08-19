package chainprimitives

import (
	"encoding/binary"
	"encoding/hex"
	"math/bits"
	"strings"
)

// Uint256 is an unsigned 256-bit integer stored as little-endian 64-bit limbs.
// Its zero value is numeric zero and every operation is allocation-free except
// decimal formatting and parsing.
type Uint256 struct {
	limbs [4]uint64
}

var maxUint256 = Uint256{limbs: [4]uint64{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}}

// Uint256Max returns the exact maximum without exposing mutable package state.
func Uint256Max() Uint256 { return maxUint256 }

func Uint256FromUint64(value uint64) Uint256 { return Uint256{limbs: [4]uint64{value}} }

func Uint256FromLimbs(limb0, limb1, limb2, limb3 uint64) Uint256 {
	return Uint256{limbs: [4]uint64{limb0, limb1, limb2, limb3}}
}

func (value Uint256) Limbs() [4]uint64 { return value.limbs }

func (value Uint256) IsZero() bool {
	return value.limbs[0]|value.limbs[1]|value.limbs[2]|value.limbs[3] == 0
}

func (value Uint256) Compare(other Uint256) int {
	for index := len(value.limbs) - 1; index >= 0; index-- {
		if value.limbs[index] < other.limbs[index] {
			return -1
		}
		if value.limbs[index] > other.limbs[index] {
			return 1
		}
	}
	return 0
}

func (value Uint256) AddChecked(other Uint256) (Uint256, error) {
	var result Uint256
	var carry uint64
	for index := range value.limbs {
		result.limbs[index], carry = bits.Add64(value.limbs[index], other.limbs[index], carry)
	}
	if carry != 0 {
		return Uint256{}, ErrOverflow
	}
	return result, nil
}

func (value Uint256) AddWrapping(other Uint256) Uint256 {
	var result Uint256
	var carry uint64
	for index := range value.limbs {
		result.limbs[index], carry = bits.Add64(value.limbs[index], other.limbs[index], carry)
	}
	return result
}

func (value Uint256) AddSaturating(other Uint256) Uint256 {
	result, err := value.AddChecked(other)
	if err != nil {
		return maxUint256
	}
	return result
}

func (value Uint256) SubChecked(other Uint256) (Uint256, error) {
	var result Uint256
	var borrow uint64
	for index := range value.limbs {
		result.limbs[index], borrow = bits.Sub64(value.limbs[index], other.limbs[index], borrow)
	}
	if borrow != 0 {
		return Uint256{}, ErrUnderflow
	}
	return result, nil
}

func (value Uint256) SubWrapping(other Uint256) Uint256 {
	var result Uint256
	var borrow uint64
	for index := range value.limbs {
		result.limbs[index], borrow = bits.Sub64(value.limbs[index], other.limbs[index], borrow)
	}
	return result
}

func (value Uint256) SubSaturating(other Uint256) Uint256 {
	result, err := value.SubChecked(other)
	if err != nil {
		return Uint256{}
	}
	return result
}

func multiplyFull256(left, right Uint256) [8]uint64 {
	var output [8]uint64
	for leftIndex, leftLimb := range left.limbs {
		for rightIndex, rightLimb := range right.limbs {
			high, low := bits.Mul64(leftLimb, rightLimb)
			index := leftIndex + rightIndex
			var carry uint64
			output[index], carry = bits.Add64(output[index], low, 0)
			output[index+1], carry = bits.Add64(output[index+1], high, carry)
			for next := index + 2; carry != 0 && next < len(output); next++ {
				output[next], carry = bits.Add64(output[next], 0, carry)
			}
		}
	}
	return output
}

func (value Uint256) MulChecked(other Uint256) (Uint256, error) {
	full := multiplyFull256(value, other)
	if full[4]|full[5]|full[6]|full[7] != 0 {
		return Uint256{}, ErrOverflow
	}
	return Uint256FromLimbs(full[0], full[1], full[2], full[3]), nil
}

func (value Uint256) MulWrapping(other Uint256) Uint256 {
	full := multiplyFull256(value, other)
	return Uint256FromLimbs(full[0], full[1], full[2], full[3])
}

func (value Uint256) MulSaturating(other Uint256) Uint256 {
	result, err := value.MulChecked(other)
	if err != nil {
		return maxUint256
	}
	return result
}

func (value Uint256) MulU64Checked(multiplier uint64) (Uint256, error) {
	var result Uint256
	var carry uint64
	for index, limb := range value.limbs {
		high, low := bits.Mul64(limb, multiplier)
		low, addCarry := bits.Add64(low, carry, 0)
		high, overflow := bits.Add64(high, 0, addCarry)
		if overflow != 0 {
			return Uint256{}, ErrOverflow
		}
		result.limbs[index] = low
		carry = high
	}
	if carry != 0 {
		return Uint256{}, ErrOverflow
	}
	return result, nil
}

func (value Uint256) AddU64Checked(addend uint64) (Uint256, error) {
	result := value
	var carry uint64
	result.limbs[0], carry = bits.Add64(result.limbs[0], addend, 0)
	for index := 1; carry != 0 && index < len(result.limbs); index++ {
		result.limbs[index], carry = bits.Add64(result.limbs[index], 0, carry)
	}
	if carry != 0 {
		return Uint256{}, ErrOverflow
	}
	return result, nil
}

func (value Uint256) DivModU64(divisor uint64) (Uint256, uint64, error) {
	if divisor == 0 {
		return Uint256{}, 0, ErrDivisionByZero
	}
	var quotient Uint256
	var remainder uint64
	for index := len(value.limbs) - 1; index >= 0; index-- {
		quotient.limbs[index], remainder = bits.Div64(remainder, value.limbs[index], divisor)
	}
	return quotient, remainder, nil
}

func (value Uint256) Bit(index uint) uint64 {
	if index >= 256 {
		return 0
	}
	return (value.limbs[index/64] >> (index % 64)) & 1
}

func (value Uint256) withBit(index uint) Uint256 {
	if index < 256 {
		value.limbs[index/64] |= uint64(1) << (index % 64)
	}
	return value
}

func shiftLeftOne(value Uint256, lowBit uint64) (Uint256, uint64) {
	var result Uint256
	carry := lowBit & 1
	for index, limb := range value.limbs {
		nextCarry := limb >> 63
		result.limbs[index] = (limb << 1) | carry
		carry = nextCarry
	}
	return result, carry
}

// DivMod performs deterministic binary long division. It avoids host-sized or
// floating-point intermediates, which keeps compiler and target oracles aligned.
func (value Uint256) DivMod(divisor Uint256) (Uint256, Uint256, error) {
	if divisor.IsZero() {
		return Uint256{}, Uint256{}, ErrDivisionByZero
	}
	var quotient Uint256
	var remainder Uint256
	for bitIndex := 256; bitIndex > 0; bitIndex-- {
		index := uint(bitIndex - 1)
		shifted, overflow := shiftLeftOne(remainder, value.Bit(index))
		remainder = shifted
		if overflow != 0 {
			var borrow uint64
			for limbIndex := range remainder.limbs {
				remainder.limbs[limbIndex], borrow = bits.Sub64(remainder.limbs[limbIndex], divisor.limbs[limbIndex], borrow)
			}
			if borrow != 1 {
				panic("chainprimitives: impossible 257-bit remainder subtraction")
			}
			quotient = quotient.withBit(index)
			continue
		}
		if remainder.Compare(divisor) >= 0 {
			var err error
			remainder, err = remainder.SubChecked(divisor)
			if err != nil {
				panic("chainprimitives: compare/subtract disagreement")
			}
			quotient = quotient.withBit(index)
		}
	}
	return quotient, remainder, nil
}

func (value Uint256) And(other Uint256) Uint256 {
	return Uint256FromLimbs(
		value.limbs[0]&other.limbs[0],
		value.limbs[1]&other.limbs[1],
		value.limbs[2]&other.limbs[2],
		value.limbs[3]&other.limbs[3],
	)
}

func (value Uint256) Or(other Uint256) Uint256 {
	return Uint256FromLimbs(
		value.limbs[0]|other.limbs[0],
		value.limbs[1]|other.limbs[1],
		value.limbs[2]|other.limbs[2],
		value.limbs[3]|other.limbs[3],
	)
}

func (value Uint256) Xor(other Uint256) Uint256 {
	return Uint256FromLimbs(
		value.limbs[0]^other.limbs[0],
		value.limbs[1]^other.limbs[1],
		value.limbs[2]^other.limbs[2],
		value.limbs[3]^other.limbs[3],
	)
}

func (value Uint256) Not() Uint256 {
	return Uint256FromLimbs(^value.limbs[0], ^value.limbs[1], ^value.limbs[2], ^value.limbs[3])
}

func (value Uint256) ShiftLeft(amount uint) (Uint256, error) {
	if amount >= 256 {
		return Uint256{}, ErrInvalidShift
	}
	if amount == 0 {
		return value, nil
	}
	wordShift := int(amount / 64)
	bitShift := amount % 64
	var result Uint256
	for sourceIndex := 0; sourceIndex < len(value.limbs); sourceIndex++ {
		targetIndex := sourceIndex + wordShift
		if targetIndex >= len(result.limbs) {
			if value.limbs[sourceIndex] != 0 {
				return Uint256{}, ErrOverflow
			}
			continue
		}
		result.limbs[targetIndex] |= value.limbs[sourceIndex] << bitShift
		if bitShift != 0 {
			high := value.limbs[sourceIndex] >> (64 - bitShift)
			if targetIndex+1 >= len(result.limbs) {
				if high != 0 {
					return Uint256{}, ErrOverflow
				}
			} else {
				result.limbs[targetIndex+1] |= high
			}
		}
	}
	return result, nil
}

func (value Uint256) ShiftRight(amount uint) (Uint256, error) {
	if amount >= 256 {
		return Uint256{}, ErrInvalidShift
	}
	if amount == 0 {
		return value, nil
	}
	wordShift := int(amount / 64)
	bitShift := amount % 64
	var result Uint256
	for targetIndex := 0; targetIndex < len(result.limbs); targetIndex++ {
		sourceIndex := targetIndex + wordShift
		if sourceIndex >= len(value.limbs) {
			break
		}
		result.limbs[targetIndex] |= value.limbs[sourceIndex] >> bitShift
		if bitShift != 0 && sourceIndex+1 < len(value.limbs) {
			result.limbs[targetIndex] |= value.limbs[sourceIndex+1] << (64 - bitShift)
		}
	}
	return result, nil
}

func Uint256FromBytesBE(input []byte) (Uint256, error) {
	if len(input) != 32 {
		return Uint256{}, ErrInvalidEncoding
	}
	return Uint256FromLimbs(
		binary.BigEndian.Uint64(input[24:32]),
		binary.BigEndian.Uint64(input[16:24]),
		binary.BigEndian.Uint64(input[8:16]),
		binary.BigEndian.Uint64(input[0:8]),
	), nil
}

func (value Uint256) BytesBE() [32]byte {
	var output [32]byte
	binary.BigEndian.PutUint64(output[0:8], value.limbs[3])
	binary.BigEndian.PutUint64(output[8:16], value.limbs[2])
	binary.BigEndian.PutUint64(output[16:24], value.limbs[1])
	binary.BigEndian.PutUint64(output[24:32], value.limbs[0])
	return output
}

func ParseUint256(text string) (Uint256, error) {
	if text == "" {
		return Uint256{}, ErrInvalidEncoding
	}
	if strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
		return Uint256{}, ErrInvalidEncoding
	}
	if strings.HasPrefix(text, "0x") {
		return ParseUint256Hex(text)
	}
	var value Uint256
	for _, digit := range []byte(text) {
		if digit < '0' || digit > '9' {
			return Uint256{}, ErrInvalidEncoding
		}
		var err error
		value, err = value.MulU64Checked(10)
		if err != nil {
			return Uint256{}, err
		}
		value, err = value.AddU64Checked(uint64(digit - '0'))
		if err != nil {
			return Uint256{}, err
		}
	}
	return value, nil
}

func ParseUint256Hex(text string) (Uint256, error) {
	text = strings.TrimPrefix(text, "0x")
	if text == "" || len(text) > 64 {
		return Uint256{}, ErrInvalidEncoding
	}
	if len(text)%2 != 0 {
		text = "0" + text
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return Uint256{}, ErrInvalidEncoding
	}
	var full [32]byte
	copy(full[len(full)-len(decoded):], decoded)
	return Uint256FromBytesBE(full[:])
}

func (value Uint256) String() string {
	if value.IsZero() {
		return "0"
	}
	const base = uint64(1_000_000_000_000_000_000)
	parts := make([]uint64, 0, 5)
	current := value
	for !current.IsZero() {
		quotient, remainder, err := current.DivModU64(base)
		if err != nil {
			panic(err)
		}
		parts = append(parts, remainder)
		current = quotient
	}
	var builder strings.Builder
	last := len(parts) - 1
	builder.WriteString(formatUint64(parts[last], false))
	for index := last - 1; index >= 0; index-- {
		builder.WriteString(formatUint64(parts[index], true))
	}
	return builder.String()
}

func formatUint64(value uint64, pad18 bool) string {
	var buffer [20]byte
	index := len(buffer)
	for {
		index--
		buffer[index] = byte(value%10) + '0'
		value /= 10
		if value == 0 {
			break
		}
	}
	if pad18 {
		for len(buffer)-index < 18 {
			index--
			buffer[index] = '0'
		}
	}
	return string(buffer[index:])
}

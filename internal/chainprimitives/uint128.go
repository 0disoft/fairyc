package chainprimitives

import (
	"encoding/hex"
	"math/bits"
	"strings"
)

// Uint128 is an unsigned 128-bit integer stored as little-endian 64-bit limbs.
// The representation is intentionally explicit so compiler hosts never depend
// on a host C compiler's __int128 extension.
type Uint128 struct {
	limbs [2]uint64
}

var maxUint128 = Uint128{limbs: [2]uint64{^uint64(0), ^uint64(0)}}

// Uint128Max returns the exact maximum without exposing mutable package state.
func Uint128Max() Uint128 { return maxUint128 }

func Uint128FromUint64(value uint64) Uint128 { return Uint128{limbs: [2]uint64{value, 0}} }

func Uint128FromLimbs(low, high uint64) Uint128 {
	return Uint128{limbs: [2]uint64{low, high}}
}

func (value Uint128) Limbs() [2]uint64 { return value.limbs }
func (value Uint128) IsZero() bool     { return value.limbs[0] == 0 && value.limbs[1] == 0 }

func (value Uint128) Compare(other Uint128) int {
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

func (value Uint128) AddChecked(other Uint128) (Uint128, error) {
	low, carry := bits.Add64(value.limbs[0], other.limbs[0], 0)
	high, carry := bits.Add64(value.limbs[1], other.limbs[1], carry)
	if carry != 0 {
		return Uint128{}, ErrOverflow
	}
	return Uint128FromLimbs(low, high), nil
}

func (value Uint128) AddWrapping(other Uint128) Uint128 {
	low, carry := bits.Add64(value.limbs[0], other.limbs[0], 0)
	high, _ := bits.Add64(value.limbs[1], other.limbs[1], carry)
	return Uint128FromLimbs(low, high)
}

func (value Uint128) AddSaturating(other Uint128) Uint128 {
	result, err := value.AddChecked(other)
	if err != nil {
		return maxUint128
	}
	return result
}

func (value Uint128) SubChecked(other Uint128) (Uint128, error) {
	low, borrow := bits.Sub64(value.limbs[0], other.limbs[0], 0)
	high, borrow := bits.Sub64(value.limbs[1], other.limbs[1], borrow)
	if borrow != 0 {
		return Uint128{}, ErrUnderflow
	}
	return Uint128FromLimbs(low, high), nil
}

func (value Uint128) SubWrapping(other Uint128) Uint128 {
	low, borrow := bits.Sub64(value.limbs[0], other.limbs[0], 0)
	high, _ := bits.Sub64(value.limbs[1], other.limbs[1], borrow)
	return Uint128FromLimbs(low, high)
}

func (value Uint128) SubSaturating(other Uint128) Uint128 {
	result, err := value.SubChecked(other)
	if err != nil {
		return Uint128{}
	}
	return result
}

func (value Uint128) MulChecked(other Uint128) (Uint128, error) {
	high00, low00 := bits.Mul64(value.limbs[0], other.limbs[0])
	high := high00
	overflow := false

	high01, low01 := bits.Mul64(value.limbs[0], other.limbs[1])
	high10, low10 := bits.Mul64(value.limbs[1], other.limbs[0])
	if high01 != 0 || high10 != 0 || (value.limbs[1] != 0 && other.limbs[1] != 0) {
		overflow = true
	}

	var carry uint64
	high, carry = bits.Add64(high, low01, 0)
	if carry != 0 {
		overflow = true
	}
	high, carry = bits.Add64(high, low10, 0)
	if carry != 0 {
		overflow = true
	}
	if overflow {
		return Uint128{}, ErrOverflow
	}
	return Uint128FromLimbs(low00, high), nil
}

func (value Uint128) MulWrapping(other Uint128) Uint128 {
	high00, low00 := bits.Mul64(value.limbs[0], other.limbs[0])
	_, low01 := bits.Mul64(value.limbs[0], other.limbs[1])
	_, low10 := bits.Mul64(value.limbs[1], other.limbs[0])
	high, _ := bits.Add64(high00, low01, 0)
	high, _ = bits.Add64(high, low10, 0)
	return Uint128FromLimbs(low00, high)
}

func (value Uint128) MulSaturating(other Uint128) Uint128 {
	result, err := value.MulChecked(other)
	if err != nil {
		return maxUint128
	}
	return result
}

func Uint128FromBytesBE(input []byte) (Uint128, error) {
	if len(input) != 16 {
		return Uint128{}, ErrInvalidEncoding
	}
	var padded [32]byte
	copy(padded[16:], input)
	value, err := Uint256FromBytesBE(padded[:])
	if err != nil {
		return Uint128{}, err
	}
	limbs := value.Limbs()
	return Uint128FromLimbs(limbs[0], limbs[1]), nil
}

func (value Uint128) BytesBE() [16]byte {
	wide := Uint256FromLimbs(value.limbs[0], value.limbs[1], 0, 0).BytesBE()
	var output [16]byte
	copy(output[:], wide[16:])
	return output
}

func ParseUint128(text string) (Uint128, error) {
	wide, err := ParseUint256(text)
	if err != nil {
		return Uint128{}, err
	}
	limbs := wide.Limbs()
	if limbs[2] != 0 || limbs[3] != 0 {
		return Uint128{}, ErrOverflow
	}
	return Uint128FromLimbs(limbs[0], limbs[1]), nil
}

func ParseUint128Hex(text string) (Uint128, error) {
	text = strings.TrimPrefix(text, "0x")
	if text == "" || len(text) > 32 {
		return Uint128{}, ErrInvalidEncoding
	}
	if len(text)%2 != 0 {
		text = "0" + text
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return Uint128{}, ErrInvalidEncoding
	}
	var full [16]byte
	copy(full[len(full)-len(decoded):], decoded)
	return Uint128FromBytesBE(full[:])
}

func (value Uint128) String() string {
	return Uint256FromLimbs(value.limbs[0], value.limbs[1], 0, 0).String()
}

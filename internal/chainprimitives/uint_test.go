package chainprimitives

import (
	"errors"
	"math/big"
	"math/rand"
	"testing"
)

var twoTo256 = new(big.Int).Lsh(big.NewInt(1), 256)

func bigFromUint256(value Uint256) *big.Int {
	bytes := value.BytesBE()
	return new(big.Int).SetBytes(bytes[:])
}

func uint256FromBigForTest(t *testing.T, value *big.Int) Uint256 {
	t.Helper()
	if value.Sign() < 0 || value.BitLen() > 256 {
		t.Fatalf("test oracle value out of range: %s", value)
	}
	bytes := value.FillBytes(make([]byte, 32))
	result, err := Uint256FromBytesBE(bytes)
	if err != nil {
		t.Fatalf("Uint256FromBytesBE: %v", err)
	}
	return result
}

func TestUint128BoundaryPolicies(t *testing.T) {
	t.Parallel()

	one := Uint128FromUint64(1)
	if _, err := Uint128Max().AddChecked(one); !errors.Is(err, ErrOverflow) {
		t.Fatalf("AddChecked error = %v, want ErrOverflow", err)
	}
	if got := Uint128Max().AddWrapping(one); !got.IsZero() {
		t.Fatalf("AddWrapping = %v, want zero", got)
	}
	if got := Uint128Max().AddSaturating(one); got != Uint128Max() {
		t.Fatalf("AddSaturating = %v, want max", got)
	}

	left := Uint128FromLimbs(0, 1)
	right := Uint128FromUint64(2)
	got, err := left.MulChecked(right)
	if err != nil {
		t.Fatalf("MulChecked: %v", err)
	}
	if want := Uint128FromLimbs(0, 2); got != want {
		t.Fatalf("MulChecked = %v, want %v", got, want)
	}
	if _, err := Uint128Max().MulChecked(right); !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflowing MulChecked error = %v, want ErrOverflow", err)
	}
}

func TestUint256AgainstBigIntOracle(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewSource(0xFA17C))
	for iteration := 0; iteration < 256; iteration++ {
		leftBytes := make([]byte, 32)
		rightBytes := make([]byte, 32)
		if _, err := random.Read(leftBytes); err != nil {
			t.Fatal(err)
		}
		if _, err := random.Read(rightBytes); err != nil {
			t.Fatal(err)
		}
		left, _ := Uint256FromBytesBE(leftBytes)
		right, _ := Uint256FromBytesBE(rightBytes)
		leftBig := new(big.Int).SetBytes(leftBytes)
		rightBig := new(big.Int).SetBytes(rightBytes)

		addBig := new(big.Int).Add(new(big.Int).Set(leftBig), rightBig)
		addWrappingWant := new(big.Int).Mod(new(big.Int).Set(addBig), twoTo256)
		if got := bigFromUint256(left.AddWrapping(right)); got.Cmp(addWrappingWant) != 0 {
			t.Fatalf("iteration %d wrapping add = %s, want %s", iteration, got, addWrappingWant)
		}
		addChecked, addErr := left.AddChecked(right)
		if addBig.BitLen() > 256 {
			if !errors.Is(addErr, ErrOverflow) {
				t.Fatalf("iteration %d add error = %v, want ErrOverflow", iteration, addErr)
			}
		} else if addErr != nil || bigFromUint256(addChecked).Cmp(addBig) != 0 {
			t.Fatalf("iteration %d checked add = %s, %v; want %s", iteration, bigFromUint256(addChecked), addErr, addBig)
		}

		subBig := new(big.Int).Sub(new(big.Int).Set(leftBig), rightBig)
		subWrappingWant := new(big.Int).Mod(new(big.Int).Set(subBig), twoTo256)
		if got := bigFromUint256(left.SubWrapping(right)); got.Cmp(subWrappingWant) != 0 {
			t.Fatalf("iteration %d wrapping sub = %s, want %s", iteration, got, subWrappingWant)
		}
		subChecked, subErr := left.SubChecked(right)
		if subBig.Sign() < 0 {
			if !errors.Is(subErr, ErrUnderflow) {
				t.Fatalf("iteration %d sub error = %v, want ErrUnderflow", iteration, subErr)
			}
		} else if subErr != nil || bigFromUint256(subChecked).Cmp(subBig) != 0 {
			t.Fatalf("iteration %d checked sub mismatch", iteration)
		}

		mulBig := new(big.Int).Mul(new(big.Int).Set(leftBig), rightBig)
		mulWrappingWant := new(big.Int).Mod(new(big.Int).Set(mulBig), twoTo256)
		if got := bigFromUint256(left.MulWrapping(right)); got.Cmp(mulWrappingWant) != 0 {
			t.Fatalf("iteration %d wrapping mul = %s, want %s", iteration, got, mulWrappingWant)
		}
		mulChecked, mulErr := left.MulChecked(right)
		if mulBig.BitLen() > 256 {
			if !errors.Is(mulErr, ErrOverflow) {
				t.Fatalf("iteration %d mul error = %v, want ErrOverflow", iteration, mulErr)
			}
		} else if mulErr != nil || bigFromUint256(mulChecked).Cmp(mulBig) != 0 {
			t.Fatalf("iteration %d checked mul mismatch", iteration)
		}

		if !right.IsZero() {
			quotient, remainder, err := left.DivMod(right)
			if err != nil {
				t.Fatalf("iteration %d DivMod: %v", iteration, err)
			}
			wantQuotient := new(big.Int)
			wantRemainder := new(big.Int)
			wantQuotient.QuoRem(leftBig, rightBig, wantRemainder)
			if bigFromUint256(quotient).Cmp(wantQuotient) != 0 || bigFromUint256(remainder).Cmp(wantRemainder) != 0 {
				t.Fatalf("iteration %d DivMod = (%s, %s), want (%s, %s)", iteration, bigFromUint256(quotient), bigFromUint256(remainder), wantQuotient, wantRemainder)
			}
		}
	}
}

func TestUint256TextRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []string{
		"0",
		"1",
		"18446744073709551615",
		"340282366920938463463374607431768211455",
		"115792089237316195423570985008687907853269984665640564039457584007913129639935",
	}
	for _, text := range cases {
		value, err := ParseUint256(text)
		if err != nil {
			t.Fatalf("ParseUint256(%q): %v", text, err)
		}
		if got := value.String(); got != text {
			t.Fatalf("ParseUint256(%q).String() = %q", text, got)
		}
		bytes := value.BytesBE()
		roundTrip, err := Uint256FromBytesBE(bytes[:])
		if err != nil || roundTrip != value {
			t.Fatalf("byte round trip for %q = %v, %v", text, roundTrip, err)
		}
	}

	if _, err := ParseUint256("115792089237316195423570985008687907853269984665640564039457584007913129639936"); !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow parse error = %v, want ErrOverflow", err)
	}
	if _, err := ParseUint256("-1"); !errors.Is(err, ErrInvalidEncoding) {
		t.Fatalf("negative parse error = %v, want ErrInvalidEncoding", err)
	}
}

func TestUint256Shifts(t *testing.T) {
	t.Parallel()

	value := Uint256FromLimbs(1, 2, 3, 4)
	for amount := uint(0); amount < 256; amount++ {
		leftBig := new(big.Int).Lsh(bigFromUint256(value), amount)
		left, err := value.ShiftLeft(amount)
		if leftBig.BitLen() > 256 {
			if !errors.Is(err, ErrOverflow) {
				t.Fatalf("ShiftLeft(%d) error = %v, want ErrOverflow", amount, err)
			}
		} else if err != nil || bigFromUint256(left).Cmp(leftBig) != 0 {
			t.Fatalf("ShiftLeft(%d) mismatch", amount)
		}

		rightBig := new(big.Int).Rsh(bigFromUint256(value), amount)
		right, err := value.ShiftRight(amount)
		if err != nil || bigFromUint256(right).Cmp(rightBig) != 0 {
			t.Fatalf("ShiftRight(%d) mismatch", amount)
		}
	}

	if _, err := value.ShiftLeft(256); !errors.Is(err, ErrInvalidShift) {
		t.Fatalf("ShiftLeft(256) error = %v", err)
	}
	if _, err := value.ShiftRight(256); !errors.Is(err, ErrInvalidShift) {
		t.Fatalf("ShiftRight(256) error = %v", err)
	}
}

func TestUint256FromBigHelper(t *testing.T) {
	t.Parallel()
	want := new(big.Int).Sub(new(big.Int).Set(twoTo256), big.NewInt(1))
	if got := bigFromUint256(uint256FromBigForTest(t, want)); got.Cmp(want) != 0 {
		t.Fatalf("helper round trip = %s, want %s", got, want)
	}
}

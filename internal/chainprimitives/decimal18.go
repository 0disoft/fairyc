package chainprimitives

import "strings"

const Decimal18Scale uint64 = 1_000_000_000_000_000_000

// Decimal18 is an unsigned fixed-point amount with exactly eighteen decimal
// places of storage. Canonical formatting trims insignificant trailing zeroes.
type Decimal18 struct {
	atoms Uint256
}

func Decimal18FromAtoms(atoms Uint256) Decimal18 { return Decimal18{atoms: atoms} }
func (value Decimal18) Atoms() Uint256           { return value.atoms }

func ParseDecimal18(text string) (Decimal18, error) {
	if text == "" || strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
		return Decimal18{}, ErrInvalidEncoding
	}
	if strings.Count(text, ".") > 1 {
		return Decimal18{}, ErrInvalidEncoding
	}
	parts := strings.SplitN(text, ".", 2)
	wholeText := parts[0]
	if wholeText == "" {
		return Decimal18{}, ErrInvalidEncoding
	}
	whole, err := ParseUint256(wholeText)
	if err != nil {
		return Decimal18{}, err
	}
	atoms, err := whole.MulU64Checked(Decimal18Scale)
	if err != nil {
		return Decimal18{}, err
	}
	if len(parts) == 1 {
		return Decimal18FromAtoms(atoms), nil
	}
	fractionText := parts[1]
	if fractionText == "" || len(fractionText) > 18 {
		return Decimal18{}, ErrInvalidEncoding
	}
	for _, digit := range []byte(fractionText) {
		if digit < '0' || digit > '9' {
			return Decimal18{}, ErrInvalidEncoding
		}
	}
	fractionText += strings.Repeat("0", 18-len(fractionText))
	fraction, err := ParseUint256(fractionText)
	if err != nil {
		return Decimal18{}, err
	}
	atoms, err = atoms.AddChecked(fraction)
	if err != nil {
		return Decimal18{}, err
	}
	return Decimal18FromAtoms(atoms), nil
}

func (value Decimal18) String() string {
	whole, fraction, err := value.atoms.DivModU64(Decimal18Scale)
	if err != nil {
		panic(err)
	}
	if fraction == 0 {
		return whole.String()
	}
	fractionText := formatUint64(fraction, true)
	fractionText = strings.TrimRight(fractionText, "0")
	return whole.String() + "." + fractionText
}

func (value Decimal18) AddChecked(other Decimal18) (Decimal18, error) {
	atoms, err := value.atoms.AddChecked(other.atoms)
	if err != nil {
		return Decimal18{}, err
	}
	return Decimal18FromAtoms(atoms), nil
}

func (value Decimal18) SubChecked(other Decimal18) (Decimal18, error) {
	atoms, err := value.atoms.SubChecked(other.atoms)
	if err != nil {
		return Decimal18{}, err
	}
	return Decimal18FromAtoms(atoms), nil
}

func (value Decimal18) MulIntegerChecked(multiplier uint64) (Decimal18, error) {
	atoms, err := value.atoms.MulU64Checked(multiplier)
	if err != nil {
		return Decimal18{}, err
	}
	return Decimal18FromAtoms(atoms), nil
}

func (value Decimal18) DivInteger(divisor uint64) (Decimal18, uint64, error) {
	atoms, remainder, err := value.atoms.DivModU64(divisor)
	if err != nil {
		return Decimal18{}, 0, err
	}
	return Decimal18FromAtoms(atoms), remainder, nil
}

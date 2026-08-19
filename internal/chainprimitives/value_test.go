package chainprimitives

import (
	"bytes"
	"errors"
	"testing"
)

func TestAddressWidthsAndCopies(t *testing.T) {
	t.Parallel()

	for _, size := range []int{20, 32} {
		input := make([]byte, size)
		for index := range input {
			input[index] = byte(index + 1)
		}
		address, err := NewAddress(input)
		if err != nil {
			t.Fatalf("NewAddress(%d): %v", size, err)
		}
		if address.Size() != size || !bytes.Equal(address.Bytes(), input) {
			t.Fatalf("address %d-byte round trip failed", size)
		}
		input[0] = 0xff
		if address.Bytes()[0] == 0xff {
			t.Fatal("address retained caller-owned input")
		}
		roundTrip, err := ParseAddressHex(address.String())
		if err != nil || roundTrip != address {
			t.Fatalf("address string round trip = %v, %v", roundTrip, err)
		}
	}
	if _, err := NewAddress(make([]byte, 21)); !errors.Is(err, ErrInvalidAddressLength) {
		t.Fatalf("21-byte address error = %v", err)
	}
}

func TestHashAndSignatureValidation(t *testing.T) {
	t.Parallel()

	hashInput := make([]byte, 32)
	hashInput[31] = 7
	hash, err := NewHash256(hashInput)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := ParseHash256(hash.String())
	if err != nil || roundTrip != hash {
		t.Fatalf("hash round trip = %v, %v", roundTrip, err)
	}

	algorithms := []SignatureAlgorithm{
		SignatureSecp256k1Compact,
		SignatureSecp256k1Recoverable,
		SignatureEd25519,
		SignatureBLS12381,
	}
	for _, algorithm := range algorithms {
		input := make([]byte, signatureLength(algorithm))
		input[len(input)-1] = byte(algorithm)
		signature, err := NewSignature(algorithm, input)
		if err != nil {
			t.Fatalf("NewSignature(%s): %v", algorithm, err)
		}
		if signature.Algorithm() != algorithm || !bytes.Equal(signature.Bytes(), input) {
			t.Fatalf("signature %s round trip failed", algorithm)
		}
	}
	if _, err := NewSignature(SignatureEd25519, make([]byte, 63)); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("short signature error = %v", err)
	}
}

func TestDecimal18CanonicalParsing(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"0":                    "0",
		"1":                    "1",
		"1.0":                  "1",
		"1.000000000000000001": "1.000000000000000001",
		"0.100000000000000000": "0.1",
		"42.125":               "42.125",
	}
	for input, want := range cases {
		value, err := ParseDecimal18(input)
		if err != nil {
			t.Fatalf("ParseDecimal18(%q): %v", input, err)
		}
		if got := value.String(); got != want {
			t.Fatalf("ParseDecimal18(%q).String() = %q, want %q", input, got, want)
		}
	}

	invalid := []string{"", ".1", "1.", "-1", "+1", "1.0000000000000000001", "1..0"}
	for _, input := range invalid {
		if _, err := ParseDecimal18(input); err == nil {
			t.Fatalf("ParseDecimal18(%q) unexpectedly succeeded", input)
		}
	}

	left, _ := ParseDecimal18("1.25")
	right, _ := ParseDecimal18("2.75")
	sum, err := left.AddChecked(right)
	if err != nil || sum.String() != "4" {
		t.Fatalf("decimal sum = %s, %v", sum.String(), err)
	}
	product, err := left.MulIntegerChecked(4)
	if err != nil || product.String() != "5" {
		t.Fatalf("decimal integer product = %s, %v", product.String(), err)
	}
}

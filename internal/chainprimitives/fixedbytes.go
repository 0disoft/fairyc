package chainprimitives

import (
	"encoding/hex"
	"strings"
)

const MaxAddressBytes = 32

// Address is a target-neutral opaque address value. It admits the two address
// widths used by the first planned contract profiles without choosing one as a
// language-wide ABI promise.
type Address struct {
	bytes [MaxAddressBytes]byte
	size  uint8
}

func NewAddress(input []byte) (Address, error) {
	if len(input) != 20 && len(input) != 32 {
		return Address{}, ErrInvalidAddressLength
	}
	var value Address
	value.size = uint8(len(input))
	copy(value.bytes[MaxAddressBytes-len(input):], input)
	return value, nil
}

func ParseAddressHex(text string) (Address, error) {
	text = strings.TrimPrefix(text, "0x")
	if len(text) != 40 && len(text) != 64 {
		return Address{}, ErrInvalidAddressLength
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return Address{}, ErrInvalidEncoding
	}
	return NewAddress(decoded)
}

func (value Address) Valid() bool { return value.size == 20 || value.size == 32 }
func (value Address) Size() int   { return int(value.size) }

func (value Address) Bytes() []byte {
	if !value.Valid() {
		return nil
	}
	start := MaxAddressBytes - int(value.size)
	return append([]byte(nil), value.bytes[start:]...)
}

func (value Address) String() string {
	if !value.Valid() {
		return ""
	}
	return "0x" + hex.EncodeToString(value.Bytes())
}

// Hash256 is one opaque 256-bit digest. No algorithm identity is implied.
type Hash256 struct {
	bytes [32]byte
}

func NewHash256(input []byte) (Hash256, error) {
	if len(input) != 32 {
		return Hash256{}, ErrInvalidEncoding
	}
	var value Hash256
	copy(value.bytes[:], input)
	return value, nil
}

func ParseHash256(text string) (Hash256, error) {
	text = strings.TrimPrefix(text, "0x")
	if len(text) != 64 {
		return Hash256{}, ErrInvalidEncoding
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return Hash256{}, ErrInvalidEncoding
	}
	return NewHash256(decoded)
}

func (value Hash256) Bytes() [32]byte { return value.bytes }
func (value Hash256) String() string  { return "0x" + hex.EncodeToString(value.bytes[:]) }

type SignatureAlgorithm uint8

const (
	SignatureAlgorithmInvalid SignatureAlgorithm = iota
	SignatureSecp256k1Compact
	SignatureSecp256k1Recoverable
	SignatureEd25519
	SignatureBLS12381
)

func (algorithm SignatureAlgorithm) String() string {
	switch algorithm {
	case SignatureSecp256k1Compact:
		return "secp256k1-compact"
	case SignatureSecp256k1Recoverable:
		return "secp256k1-recoverable"
	case SignatureEd25519:
		return "ed25519"
	case SignatureBLS12381:
		return "bls12-381"
	default:
		return "invalid"
	}
}

func signatureLength(algorithm SignatureAlgorithm) int {
	switch algorithm {
	case SignatureSecp256k1Compact, SignatureEd25519:
		return 64
	case SignatureSecp256k1Recoverable:
		return 65
	case SignatureBLS12381:
		return 96
	default:
		return 0
	}
}

// Signature is an opaque algorithm-tagged signature. Verification remains a
// target intrinsic; this type only preserves exact bytes and algorithm identity.
type Signature struct {
	bytes     [96]byte
	algorithm SignatureAlgorithm
}

func NewSignature(algorithm SignatureAlgorithm, input []byte) (Signature, error) {
	expected := signatureLength(algorithm)
	if expected == 0 || len(input) != expected {
		return Signature{}, ErrInvalidSignature
	}
	var value Signature
	value.algorithm = algorithm
	copy(value.bytes[:], input)
	return value, nil
}

func (value Signature) Valid() bool                   { return signatureLength(value.algorithm) != 0 }
func (value Signature) Algorithm() SignatureAlgorithm { return value.algorithm }
func (value Signature) Bytes() []byte {
	length := signatureLength(value.algorithm)
	if length == 0 {
		return nil
	}
	return append([]byte(nil), value.bytes[:length]...)
}

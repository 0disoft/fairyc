package contractstate

import (
	"encoding/binary"
	"fmt"
	"slices"
	"unicode/utf8"
)

// Codec defines one deterministic canonical representation used by persistent
// state. TypeID participates in the storage layout hash and is therefore a
// compatibility contract, not display text.
type Codec[T any] interface {
	TypeID() string
	Encode(T) ([]byte, error)
	Decode([]byte) (T, error)
}

// FuncCodec adapts explicit deterministic encode and decode functions. Callers
// must keep ID and byte behavior stable for every deployed layout.
type FuncCodec[T any] struct {
	ID         string
	EncodeFunc func(T) ([]byte, error)
	DecodeFunc func([]byte) (T, error)
}

func (codec FuncCodec[T]) TypeID() string {
	return codec.ID
}

func (codec FuncCodec[T]) Encode(value T) ([]byte, error) {
	if codec.EncodeFunc == nil {
		return nil, fmt.Errorf("contract state codec %q has no encoder", codec.ID)
	}
	encoded, err := codec.EncodeFunc(value)
	if err != nil {
		return nil, err
	}
	return slices.Clone(encoded), nil
}

func (codec FuncCodec[T]) Decode(encoded []byte) (T, error) {
	var zero T
	if codec.DecodeFunc == nil {
		return zero, fmt.Errorf("contract state codec %q has no decoder", codec.ID)
	}
	return codec.DecodeFunc(slices.Clone(encoded))
}

// Bytes20 and Bytes32 are fixed-width canonical byte values useful for account,
// hash, and identifier keys without assigning a chain-specific meaning.
type Bytes20 [20]byte
type Bytes32 [32]byte

var (
	Bool   Codec[bool]    = boolCodec{}
	U32    Codec[uint32]  = u32Codec{}
	U64    Codec[uint64]  = u64Codec{}
	Bytes  Codec[[]byte]  = bytesCodec{}
	Text   Codec[string]  = textCodec{}
	Byte20 Codec[Bytes20] = bytes20Codec{}
	Byte32 Codec[Bytes32] = bytes32Codec{}
)

type boolCodec struct{}

func (boolCodec) TypeID() string { return "bool/v1" }

func (boolCodec) Encode(value bool) ([]byte, error) {
	if value {
		return []byte{1}, nil
	}
	return []byte{0}, nil
}

func (boolCodec) Decode(encoded []byte) (bool, error) {
	if len(encoded) != 1 || encoded[0] > 1 {
		return false, fmt.Errorf("bool/v1 requires exactly one canonical 0 or 1 byte")
	}
	return encoded[0] == 1, nil
}

type u32Codec struct{}

func (u32Codec) TypeID() string { return "u32/be/v1" }

func (u32Codec) Encode(value uint32) ([]byte, error) {
	encoded := make([]byte, 4)
	binary.BigEndian.PutUint32(encoded, value)
	return encoded, nil
}

func (u32Codec) Decode(encoded []byte) (uint32, error) {
	if len(encoded) != 4 {
		return 0, fmt.Errorf("u32/be/v1 requires exactly 4 bytes")
	}
	return binary.BigEndian.Uint32(encoded), nil
}

type u64Codec struct{}

func (u64Codec) TypeID() string { return "u64/be/v1" }

func (u64Codec) Encode(value uint64) ([]byte, error) {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, value)
	return encoded, nil
}

func (u64Codec) Decode(encoded []byte) (uint64, error) {
	if len(encoded) != 8 {
		return 0, fmt.Errorf("u64/be/v1 requires exactly 8 bytes")
	}
	return binary.BigEndian.Uint64(encoded), nil
}

type bytesCodec struct{}

func (bytesCodec) TypeID() string { return "bytes/v1" }
func (bytesCodec) Encode(value []byte) ([]byte, error) {
	return slices.Clone(value), nil
}
func (bytesCodec) Decode(encoded []byte) ([]byte, error) {
	return slices.Clone(encoded), nil
}

type textCodec struct{}

func (textCodec) TypeID() string { return "utf8/v1" }
func (textCodec) Encode(value string) ([]byte, error) {
	if !utf8.ValidString(value) {
		return nil, fmt.Errorf("utf8/v1 rejects invalid UTF-8")
	}
	return []byte(value), nil
}
func (textCodec) Decode(encoded []byte) (string, error) {
	if !utf8.Valid(encoded) {
		return "", fmt.Errorf("utf8/v1 rejects invalid UTF-8")
	}
	return string(encoded), nil
}

type bytes20Codec struct{}

func (bytes20Codec) TypeID() string { return "bytes20/v1" }
func (bytes20Codec) Encode(value Bytes20) ([]byte, error) {
	return slices.Clone(value[:]), nil
}
func (bytes20Codec) Decode(encoded []byte) (Bytes20, error) {
	var value Bytes20
	if len(encoded) != len(value) {
		return value, fmt.Errorf("bytes20/v1 requires exactly 20 bytes")
	}
	copy(value[:], encoded)
	return value, nil
}

type bytes32Codec struct{}

func (bytes32Codec) TypeID() string { return "bytes32/v1" }
func (bytes32Codec) Encode(value Bytes32) ([]byte, error) {
	return slices.Clone(value[:]), nil
}
func (bytes32Codec) Decode(encoded []byte) (Bytes32, error) {
	var value Bytes32
	if len(encoded) != len(value) {
		return value, fmt.Errorf("bytes32/v1 requires exactly 32 bytes")
	}
	copy(value[:], encoded)
	return value, nil
}

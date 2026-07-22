package atlasmap

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"
)

const (
	TransportSchema     = "fairyc.fymap/v1"
	TransportEncoding   = "fairyc.fymap-json/v1"
	ChecksumAlgorithm   = "fairyc.fymap-checksum/v1-sha256"
	ChecksumPrefix      = "fymap1-sha256-"
	checksumInputDomain = "fairyc.fymap-checksum"
)

var (
	ErrMapTooLarge      = errors.New(".fymap exceeds the caller byte limit")
	ErrNonCanonical     = errors.New(".fymap is not canonically encoded")
	ErrChecksumMismatch = errors.New(".fymap checksum mismatch")
)

// Checksum is the full digest of the transport schema, encoding, and canonical logical document.
type Checksum [sha256.Size]byte

func (checksum Checksum) String() string {
	return ChecksumPrefix + hex.EncodeToString(checksum[:])
}

// ParseChecksum accepts only the canonical full-width lowercase transport checksum.
func ParseChecksum(value string) (Checksum, error) {
	var result Checksum
	if !strings.HasPrefix(value, ChecksumPrefix) {
		return result, fmt.Errorf(".fymap checksum has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, ChecksumPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf(".fymap checksum must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode .fymap checksum: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// Encode validates one logical package partition and returns deterministic, checksummed exact-v1
// JSON bytes. The caller must choose a positive maximum; the transport has no guessed global cap.
func Encode(partition Partition, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf(".fymap byte limit must be positive")
	}
	if err := validateTransportPartition(partition); err != nil {
		return nil, fmt.Errorf("validate .fymap partition: %w", err)
	}
	document := documentFromPartition(partition)
	checksum, err := computeTransportChecksum(document)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(transportEnvelope{
		Schema: TransportSchema, Encoding: TransportEncoding,
		ChecksumAlgorithm: ChecksumAlgorithm, Checksum: checksum.String(), Map: document,
	})
	if err != nil {
		return nil, fmt.Errorf("encode .fymap: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrMapTooLarge
	}
	return encoded, nil
}

// Decode enforces the caller's byte limit, exact transport versions, strict fields, checksum,
// semantic record invariants, and byte-for-byte canonical encoding before returning immutable data.
func Decode(encoded []byte, maxBytes uint64) (Partition, error) {
	if maxBytes == 0 {
		return Partition{}, fmt.Errorf(".fymap byte limit must be positive")
	}
	if len(encoded) == 0 {
		return Partition{}, fmt.Errorf(".fymap is empty")
	}
	if uint64(len(encoded)) > maxBytes {
		return Partition{}, ErrMapTooLarge
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope transportEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return Partition{}, fmt.Errorf("decode .fymap: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Partition{}, fmt.Errorf("decode .fymap: trailing JSON value")
		}
		return Partition{}, fmt.Errorf("decode .fymap trailing data: %w", err)
	}
	if envelope.Schema != TransportSchema {
		return Partition{}, fmt.Errorf(".fymap schema %q is incompatible with %q", envelope.Schema, TransportSchema)
	}
	if envelope.Encoding != TransportEncoding {
		return Partition{}, fmt.Errorf(".fymap encoding %q is incompatible with %q", envelope.Encoding, TransportEncoding)
	}
	if envelope.ChecksumAlgorithm != ChecksumAlgorithm {
		return Partition{}, fmt.Errorf(".fymap checksum algorithm %q is incompatible with %q", envelope.ChecksumAlgorithm, ChecksumAlgorithm)
	}
	storedChecksum, err := ParseChecksum(envelope.Checksum)
	if err != nil {
		return Partition{}, err
	}
	computedChecksum, err := computeTransportChecksum(envelope.Map)
	if err != nil {
		return Partition{}, err
	}
	if subtle.ConstantTimeCompare(storedChecksum[:], computedChecksum[:]) != 1 {
		return Partition{}, ErrChecksumMismatch
	}
	partition, err := partitionFromDocument(envelope.Map)
	if err != nil {
		return Partition{}, fmt.Errorf("decode .fymap records: %w", err)
	}
	if err := validateTransportPartition(partition); err != nil {
		return Partition{}, fmt.Errorf("validate decoded .fymap: %w", err)
	}
	canonical, err := Encode(partition, maxBytes)
	if err != nil {
		return Partition{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return Partition{}, ErrNonCanonical
	}
	return partition, nil
}

func computeTransportChecksum(document transportDocument) (Checksum, error) {
	payload, err := json.Marshal(document)
	if err != nil {
		return Checksum{}, fmt.Errorf("encode .fymap checksum payload: %w", err)
	}
	digest := sha256.New()
	writeChecksumHeader(digest)
	writeChecksumBytes(digest, []byte(TransportSchema))
	writeChecksumBytes(digest, []byte(TransportEncoding))
	writeChecksumBytes(digest, payload)
	var result Checksum
	copy(result[:], digest.Sum(nil))
	return result, nil
}

func writeChecksumHeader(destination hash.Hash) {
	_, _ = destination.Write([]byte(checksumInputDomain))
	_, _ = destination.Write([]byte{0, 1})
}

func writeChecksumBytes(destination hash.Hash, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = destination.Write(length[:])
	_, _ = destination.Write(value)
}

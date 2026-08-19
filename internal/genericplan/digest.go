package genericplan

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
)

// Digest is one exact semantic or implementation identity.
type Digest [sha256.Size]byte

// DigestText creates a domain-separated digest for checked-in definitions,
// implementation bodies, fixtures, and examples.
func DigestText(domain, text string) Digest {
	value := sha256.New()
	writeString(value, "fairyc.genericplan.digest.v0")
	writeString(value, domain)
	writeString(value, text)
	var result Digest
	copy(result[:], value.Sum(nil))
	return result
}

// IsZero reports whether no semantic identity was supplied.
func (d Digest) IsZero() bool {
	return d == Digest{}
}

// String returns the canonical lowercase hexadecimal representation.
func (d Digest) String() string {
	return hex.EncodeToString(d[:])
}

func writeUint64(target hash.Hash, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = target.Write(encoded[:])
}

func writeString(target hash.Hash, value string) {
	writeUint64(target, uint64(len(value)))
	_, _ = target.Write([]byte(value))
}

func writeDigest(target hash.Hash, value Digest) {
	_, _ = target.Write(value[:])
}

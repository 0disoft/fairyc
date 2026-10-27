package cdriver

import "crypto/sha256"

func sourceDigest(source string) [sha256.Size]byte {
	if len(source) <= 32 {
		// Preserve the stack-backed conversion for small inputs.
		return sha256.Sum256([]byte(source))
	}
	return chunkedSourceDigest(source)
}

// Keep scratch storage in a separate frame so small inputs do not pay for it.
// Cache identity hashes the same exact bytes without copying an entire source
// or preprocessed unit into a temporary byte slice.
func chunkedSourceDigest(source string) [sha256.Size]byte {
	hasher := sha256.New()
	var scratch [4096]byte
	for len(source) > 0 {
		n := copy(scratch[:], source)
		_, _ = hasher.Write(scratch[:n])
		source = source[n:]
	}
	var digest [sha256.Size]byte
	hasher.Sum(digest[:0])
	return digest
}

// Package buildidentity computes an exact identity for the running compiler executable.
package buildidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

const Encoding = "fairyc.compiler-executable-sha256/v1"

// Identity is the complete SHA-256 digest of one executable image.
type Identity [sha256.Size]byte

func (identity Identity) String() string {
	return Encoding + ":" + hex.EncodeToString(identity[:])
}

// Parse accepts only the canonical exact-width lowercase executable identity.
func Parse(value string) (Identity, error) {
	var result Identity
	prefix := Encoding + ":"
	if !strings.HasPrefix(value, prefix) {
		return result, fmt.Errorf("compiler executable identity has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, prefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("compiler executable identity must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode compiler executable identity: %w", err)
	}
	copy(result[:], decoded)
	if result == (Identity{}) {
		return Identity{}, fmt.Errorf("compiler executable identity is zero")
	}
	return result, nil
}

// Current streams the executable image backing this process into SHA-256. Linux prefers
// /proc/self/exe so a renamed or unlinked on-disk path still identifies the running image; other
// hosts use os.Executable. There is no release/version fallback because ambiguity must disable
// persistent cache reuse instead of widening compatibility.
func Current() (Identity, error) {
	path, pathErr := os.Executable()
	candidates := make([]string, 0, 2)
	if runtime.GOOS == "linux" {
		candidates = append(candidates, "/proc/self/exe")
	}
	if pathErr == nil && path != "" {
		candidates = append(candidates, path)
	}
	var lastErr error
	for _, candidate := range candidates {
		file, err := os.Open(candidate)
		if err != nil {
			lastErr = err
			continue
		}
		identity, computeErr := compute(file)
		closeErr := file.Close()
		if computeErr != nil {
			lastErr = computeErr
			continue
		}
		if closeErr != nil {
			lastErr = closeErr
			continue
		}
		return identity, nil
	}
	if lastErr == nil {
		lastErr = pathErr
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no executable path is available")
	}
	return Identity{}, fmt.Errorf("identify running compiler executable: %w", lastErr)
}

func compute(reader io.Reader) (Identity, error) {
	if reader == nil {
		return Identity{}, fmt.Errorf("compiler executable reader is nil")
	}
	hash := sha256.New()
	buffer := make([]byte, 128<<10)
	written, err := io.CopyBuffer(hash, reader, buffer)
	if err != nil {
		return Identity{}, fmt.Errorf("hash compiler executable: %w", err)
	}
	if written == 0 {
		return Identity{}, fmt.Errorf("compiler executable is empty")
	}
	var identity Identity
	copy(identity[:], hash.Sum(nil))
	return identity, nil
}

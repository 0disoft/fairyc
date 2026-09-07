package moduleinterface

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const GenericHashEncoding = "fairyc.generic-public-interface/v0"
const GenericHashPrefix = "fygif0-sha256-"

// ParseGenericHash accepts only the canonical full-width source-interface digest.
func ParseGenericHash(value string) (GenericHash, error) {
	var result GenericHash
	if !strings.HasPrefix(value, GenericHashPrefix) || len(value) != len(GenericHashPrefix)+64 || strings.ToLower(value) != value {
		return result, fmt.Errorf("invalid generic interface hash")
	}
	decoded, err := hex.DecodeString(value[len(GenericHashPrefix):])
	if err != nil {
		return result, fmt.Errorf("decode generic interface hash: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

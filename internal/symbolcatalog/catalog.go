// Package symbolcatalog owns canonical output-only exact-current Atlas SymbolId catalog pages.
// It accepts no source text, paths, documentation, untrusted encoded input, or fuzzy search terms;
// its optional declaration-name filter is exact and compiler-validated.
package symbolcatalog

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const (
	CatalogVersion    = "fairyc.symbol-catalog/v1"
	CatalogEncoding   = "fairyc.symbol-catalog-json/v1"
	ChecksumAlgorithm = "fairyc.symbol-catalog-checksum/v1-sha256"
	ChecksumPrefix    = "fyscat1-sha256-"
	checksumDomain    = "fairyc.symbol-catalog-checksum"
)

var ErrCatalogTooLarge = errors.New("Atlas symbol catalog exceeds the caller byte limit")

type Checksum [sha256.Size]byte

func (checksum Checksum) String() string {
	return ChecksumPrefix + hex.EncodeToString(checksum[:])
}

func ParseChecksum(value string) (Checksum, error) {
	var result Checksum
	if !strings.HasPrefix(value, ChecksumPrefix) {
		return result, fmt.Errorf("symbol catalog checksum has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, ChecksumPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("symbol catalog checksum must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode symbol catalog checksum: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

type envelope struct {
	CatalogVersion    string                       `json:"catalog_version"`
	Encoding          string                       `json:"encoding"`
	ChecksumAlgorithm string                       `json:"checksum_algorithm"`
	Checksum          string                       `json:"checksum"`
	Catalog           atlasmap.SymbolCatalogResult `json:"catalog"`
}

// Encode runs one bounded exact-current symbol query and emits one compact checksummed JSON value.
func Encode(
	index *atlasmap.QueryIndex,
	request atlasmap.SymbolCatalogRequest,
	maxBytes uint64,
) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("symbol catalog byte limit must be positive")
	}
	result, err := index.CatalogSymbols(request)
	if err != nil {
		return nil, err
	}
	if result.Context.MapOmissions == nil {
		result.Context.MapOmissions = make([]string, 0)
	}
	if result.Context.Completeness == nil {
		result.Context.Completeness = make([]semanticsnapshot.Completeness, 0)
	}
	if result.Symbols == nil {
		result.Symbols = make([]atlasmap.SymbolCatalogEntry, 0)
	}
	if result.Omissions == nil {
		result.Omissions = make([]atlasmap.QueryOmission, 0)
	}
	checksum, err := computeChecksum(result)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(envelope{
		CatalogVersion: CatalogVersion, Encoding: CatalogEncoding,
		ChecksumAlgorithm: ChecksumAlgorithm, Checksum: checksum.String(), Catalog: result,
	})
	if err != nil {
		return nil, fmt.Errorf("encode symbol catalog: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrCatalogTooLarge
	}
	return encoded, nil
}

func computeChecksum(result atlasmap.SymbolCatalogResult) (Checksum, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return Checksum{}, fmt.Errorf("encode symbol catalog checksum payload: %w", err)
	}
	digest := sha256.New()
	writeChecksumHeader(digest)
	writeChecksumBytes(digest, []byte(CatalogVersion))
	writeChecksumBytes(digest, []byte(CatalogEncoding))
	writeChecksumBytes(digest, payload)
	var resultChecksum Checksum
	copy(resultChecksum[:], digest.Sum(nil))
	return resultChecksum, nil
}

func writeChecksumHeader(destination hash.Hash) {
	_, _ = destination.Write([]byte(checksumDomain))
	_, _ = destination.Write([]byte{0, 0})
}

func writeChecksumBytes(destination hash.Hash, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = destination.Write(length[:])
	_, _ = destination.Write(value)
}

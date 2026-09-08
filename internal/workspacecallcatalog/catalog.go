// Package workspacecallcatalog owns canonical output-only exact-current joined-call catalogs.
// It accepts no source text, documentation, decoded map bytes, fuzzy selector, or authority.
package workspacecallcatalog

import (
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const (
	CatalogVersion    = "fairyc.workspace-call-catalog/v0"
	CatalogEncoding   = "fairyc.workspace-call-catalog-json/v0"
	ChecksumAlgorithm = "fairyc.workspace-call-catalog-checksum/v0-sha256"
	ChecksumPrefix    = "fywcall0-sha256-"
	checksumDomain    = "fairyc.workspace-call-catalog-checksum"
)

var ErrCatalogTooLarge = errors.New("Atlas workspace call catalog exceeds the caller byte limit")

type Checksum [sha256.Size]byte

func (checksum Checksum) String() string {
	return ChecksumPrefix + hex.EncodeToString(checksum[:])
}

func ParseChecksum(value string) (Checksum, error) {
	var result Checksum
	if !strings.HasPrefix(value, ChecksumPrefix) {
		return result, fmt.Errorf("workspace call catalog checksum has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, ChecksumPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("workspace call catalog checksum must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode workspace call catalog checksum: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

type Symbol struct {
	PackageInstance string `json:"package_instance"`
	Symbol          string `json:"symbol"`
	Specialization  string `json:"specialization,omitempty"`
}

type Span struct {
	PackageInstance string `json:"package_instance"`
	Source          string `json:"source"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
}

type Call struct {
	SourceBody string                      `json:"source_body,omitempty"`
	Source     Symbol                      `json:"source"`
	Target     Symbol                      `json:"target"`
	Span       Span                        `json:"span"`
	Certainty  semanticsnapshot.Certainty  `json:"certainty"`
	Provenance semanticsnapshot.Provenance `json:"provenance"`
}

type Budget struct {
	MaxPackages int `json:"max_packages"`
	MaxCalls    int `json:"max_calls"`
}

type Usage struct {
	Packages          int `json:"packages"`
	Calls             int `json:"calls"`
	CrossPackageCalls int `json:"cross_package_calls"`
	SelectedCalls     int `json:"selected_calls"`
}

type Catalog struct {
	Context   atlasmap.QueryContext   `json:"context"`
	Subject   Symbol                  `json:"subject"`
	Direction atlasmap.QueryDirection `json:"direction"`
	Budget    Budget                  `json:"budget"`
	Usage     Usage                   `json:"usage"`
	Calls     []Call                  `json:"calls"`
	Complete  bool                    `json:"complete"`
}

type envelope struct {
	CatalogVersion    string  `json:"catalog_version"`
	Encoding          string  `json:"encoding"`
	ChecksumAlgorithm string  `json:"checksum_algorithm"`
	Checksum          string  `json:"checksum"`
	Catalog           Catalog `json:"catalog"`
}

// Encode resolves one exact package-qualified subject, selects the requested joined-call direction,
// and emits one compact checksummed JSON value. Construction budgets are already enforced by index.
func Encode(
	index *atlasmap.WorkspaceCallIndex,
	packageInstance, symbolID string,
	direction atlasmap.QueryDirection,
	maxBytes uint64,
) ([]byte, error) {
	return encode(index, packageInstance, symbolID, direction, maxBytes, 0)
}

func encode(index *atlasmap.WorkspaceCallIndex, packageInstance, symbolID string, direction atlasmap.QueryDirection, maxBytes uint64, catalogVersion byte) ([]byte, error) {
	if index == nil {
		return nil, fmt.Errorf("Atlas workspace call index is not initialized")
	}
	if maxBytes == 0 {
		return nil, fmt.Errorf("workspace call catalog byte limit must be positive")
	}
	if direction != atlasmap.DirectionOutgoing && direction != atlasmap.DirectionIncoming && direction != atlasmap.DirectionBoth {
		return nil, fmt.Errorf("workspace call catalog direction %q is invalid", direction)
	}
	subject, err := index.ResolveSymbol(packageInstance, symbolID)
	if err != nil {
		return nil, err
	}
	context, err := index.PackageContext(packageInstance)
	if err != nil {
		return nil, err
	}
	schema := atlasmap.LogicalSchema
	switch catalogVersion {
	case 0:
	case 1:
		schema = atlasmap.LogicalSchemaV2
	case 2:
		schema = atlasmap.LogicalSchemaV3
	default:
		return nil, fmt.Errorf("unsupported workspace call catalog version %d", catalogVersion)
	}
	if context.LogicalSchema != schema {
		return nil, fmt.Errorf("workspace call catalog requires %s maps", schema)
	}
	calls, err := selectCalls(index, packageInstance, symbolID, direction)
	if err != nil {
		return nil, err
	}
	budget := index.Budget()
	usage := index.Usage()
	catalog := Catalog{
		Context:   context,
		Subject:   Symbol{PackageInstance: subject.PackageInstance, Symbol: subject.Symbol},
		Direction: direction,
		Budget:    Budget{MaxPackages: budget.MaxPackages, MaxCalls: budget.MaxCalls},
		Usage: Usage{
			Packages: usage.Packages, Calls: usage.Calls, CrossPackageCalls: usage.CrossPackageCalls,
			SelectedCalls: len(calls),
		},
		Calls: calls, Complete: true,
	}
	if catalog.Context.MapOmissions == nil {
		catalog.Context.MapOmissions = make([]string, 0)
	}
	if catalog.Context.Completeness == nil {
		catalog.Context.Completeness = make([]semanticsnapshot.Completeness, 0)
	}
	if catalog.Calls == nil {
		catalog.Calls = make([]Call, 0)
	}
	version, encoding, algorithm, prefix, checksumVersion := CatalogVersion, CatalogEncoding, ChecksumAlgorithm, ChecksumPrefix, byte(0)
	if catalogVersion == 1 {
		version, encoding, algorithm, prefix, checksumVersion = CatalogVersionV1, CatalogEncodingV1, ChecksumAlgorithmV1, ChecksumPrefixV1, 1
	} else if catalogVersion == 2 {
		version, encoding, algorithm, prefix, checksumVersion = CatalogVersionV2, CatalogEncodingV2, ChecksumAlgorithmV2, ChecksumPrefixV2, 2
	}
	checksum, err := computeVersionedChecksum(catalog, version, encoding, checksumVersion)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(envelope{
		CatalogVersion: version, Encoding: encoding,
		ChecksumAlgorithm: algorithm, Checksum: prefix + hex.EncodeToString(checksum[:]), Catalog: catalog,
	})
	if err != nil {
		return nil, fmt.Errorf("encode workspace call catalog: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrCatalogTooLarge
	}
	return encoded, nil
}

func selectCalls(
	index *atlasmap.WorkspaceCallIndex,
	packageInstance, symbolID string,
	direction atlasmap.QueryDirection,
) ([]Call, error) {
	joined := make([]atlasmap.JoinedCall, 0)
	if direction == atlasmap.DirectionOutgoing || direction == atlasmap.DirectionBoth {
		outgoing, err := index.OutgoingCalls(packageInstance, symbolID)
		if err != nil {
			return nil, err
		}
		joined = append(joined, outgoing...)
	}
	if direction == atlasmap.DirectionIncoming || direction == atlasmap.DirectionBoth {
		incoming, err := index.IncomingCalls(packageInstance, symbolID)
		if err != nil {
			return nil, err
		}
		joined = append(joined, incoming...)
	}
	slices.SortFunc(joined, compareJoinedCall)
	joined = slices.Compact(joined)
	result := make([]Call, 0, len(joined))
	for _, call := range joined {
		result = append(result, Call{
			SourceBody: call.SourceBody,
			Source:     Symbol{PackageInstance: call.Source.PackageInstance, Symbol: call.Source.Symbol, Specialization: call.SourceSpecialization},
			Target:     Symbol{PackageInstance: call.Target.PackageInstance, Symbol: call.Target.Symbol, Specialization: call.TargetSpecialization},
			Span: Span{
				PackageInstance: call.Span.PackageInstance, Source: call.Span.Source,
				StartByte: call.Span.StartByte, EndByte: call.Span.EndByte,
			},
			Certainty: call.Certainty, Provenance: call.Provenance,
		})
	}
	return result, nil
}

func compareJoinedCall(left, right atlasmap.JoinedCall) int {
	for _, value := range []int{
		cmp.Compare(left.Source.PackageInstance, right.Source.PackageInstance),
		cmp.Compare(left.Source.Symbol, right.Source.Symbol),
		cmp.Compare(left.SourceBody, right.SourceBody),
		cmp.Compare(left.SourceSpecialization, right.SourceSpecialization),
		cmp.Compare(left.Target.PackageInstance, right.Target.PackageInstance),
		cmp.Compare(left.Target.Symbol, right.Target.Symbol),
		cmp.Compare(left.TargetSpecialization, right.TargetSpecialization),
		cmp.Compare(left.Span.PackageInstance, right.Span.PackageInstance),
		cmp.Compare(left.Span.Source, right.Span.Source),
		cmp.Compare(left.Span.StartByte, right.Span.StartByte),
		cmp.Compare(left.Span.EndByte, right.Span.EndByte),
	} {
		if value != 0 {
			return value
		}
	}
	return 0
}

func computeVersionedChecksum(catalog Catalog, version, encoding string, checksumVersion byte) (Checksum, error) {
	payload, err := json.Marshal(catalog)
	if err != nil {
		return Checksum{}, fmt.Errorf("encode workspace call catalog checksum payload: %w", err)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte(checksumDomain))
	_, _ = digest.Write([]byte{0, checksumVersion})
	writeChecksumBytes(digest, []byte(version))
	writeChecksumBytes(digest, []byte(encoding))
	writeChecksumBytes(digest, payload)
	var result Checksum
	copy(result[:], digest.Sum(nil))
	return result, nil
}

func writeChecksumBytes(destination hash.Hash, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = destination.Write(length[:])
	_, _ = destination.Write(value)
}

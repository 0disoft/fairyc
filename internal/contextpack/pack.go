// Package contextpack owns deterministic compiler-fact-only Atlas Context Pack output. It consumes
// exact-current bounded atlasmap queries and has no source, documentation, filesystem, process,
// network, model, or authority-bearing input.
package contextpack

import (
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
	PackVersion       = "fairyc.context-pack/v1"
	PackEncoding      = "fairyc.context-pack-json/v1"
	ChecksumAlgorithm = "fairyc.context-pack-checksum/v1-sha256"
	ChecksumPrefix    = "fyctx1-sha256-"
	checksumDomain    = "fairyc.context-pack-checksum"
)

const (
	QueryOverview QueryKind = "overview"
	QueryShow     QueryKind = "show"
	QueryAround   QueryKind = "around"
)

var ErrPackTooLarge = errors.New("Atlas Context Pack exceeds the caller byte limit")

type QueryKind string

// Checksum is the full digest of the pack version, encoding, and canonical logical document.
type Checksum [sha256.Size]byte

func (checksum Checksum) String() string {
	return ChecksumPrefix + hex.EncodeToString(checksum[:])
}

func ParseChecksum(value string) (Checksum, error) {
	var result Checksum
	if !strings.HasPrefix(value, ChecksumPrefix) {
		return result, fmt.Errorf("Context Pack checksum has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, ChecksumPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("Context Pack checksum must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode Context Pack checksum: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// EncodeOverview runs one bounded exact-current overview query and returns one canonical pack.
func EncodeOverview(index *atlasmap.QueryIndex, budget atlasmap.QueryBudget, maxBytes uint64) ([]byte, error) {
	if err := validateMaxBytes(maxBytes); err != nil {
		return nil, err
	}
	result, err := index.Overview(budget)
	if err != nil {
		return nil, err
	}
	header := result.Package.Header
	document := baseDocument(
		packageIdentity{UID: header.PackageUID, Instance: header.PackageInstance, Coordinate: header.PackageCoordinate, Source: header.PackageSource},
		compatibility{
			CompilerBuild: header.CompilerBuild, Language: header.Language, InterfaceHash: header.InterfaceHash,
			Generation: header.GenerationStatus, MapOmissions: nonNil(header.Omissions),
		},
		header.LogicalSchema,
		queryDescriptor{Kind: QueryOverview, Subject: header.PackageInstance, Relations: make([]atlasmap.RelationKind, 0)},
		budget,
		result.Usage,
		result.Completeness,
		result.Omissions,
	)
	document.Facts.Overview = &overviewFacts{Package: result.Package, Modules: nonNil(result.Modules)}
	return encode(document, maxBytes)
}

// EncodeShow runs one bounded exact-current symbol query and returns one canonical pack.
func EncodeShow(
	index *atlasmap.QueryIndex,
	symbolID string,
	request atlasmap.RelationRequest,
	maxBytes uint64,
) ([]byte, error) {
	if err := validateMaxBytes(maxBytes); err != nil {
		return nil, err
	}
	result, err := index.Show(symbolID, request)
	if err != nil {
		return nil, err
	}
	document := documentFromContext(
		result.Context,
		queryDescriptor{
			Kind: QueryShow, Subject: result.Subject.Symbol.Visible, Direction: result.Direction,
			Relations: nonNil(result.RelationKinds),
		},
		result.Budget,
		result.Usage,
		result.Omissions,
	)
	document.Facts.Show = &showFacts{Subject: result.Subject}
	document.Relations = nonNil(result.Relations)
	return encode(document, maxBytes)
}

// EncodeAround runs one bounded exact-current neighborhood query and returns one canonical pack.
func EncodeAround(
	index *atlasmap.QueryIndex,
	symbolID string,
	request atlasmap.RelationRequest,
	maxBytes uint64,
) ([]byte, error) {
	if err := validateMaxBytes(maxBytes); err != nil {
		return nil, err
	}
	result, err := index.Around(symbolID, request)
	if err != nil {
		return nil, err
	}
	document := documentFromContext(
		result.Context,
		queryDescriptor{
			Kind: QueryAround, Subject: result.Subject, Direction: result.Direction,
			Relations: nonNil(result.RelationKinds),
		},
		result.Budget,
		result.Usage,
		result.Omissions,
	)
	document.Facts.Around = &aroundFacts{Nodes: nonNil(result.Nodes)}
	document.Relations = nonNil(result.Relations)
	return encode(document, maxBytes)
}

type packEnvelope struct {
	PackVersion       string       `json:"pack_version"`
	Encoding          string       `json:"encoding"`
	ChecksumAlgorithm string       `json:"checksum_algorithm"`
	Checksum          string       `json:"checksum"`
	Pack              packDocument `json:"pack"`
}

type packDocument struct {
	MapVersion      string                          `json:"map_version"`
	PackageIdentity packageIdentity                 `json:"package_identity"`
	Compatibility   compatibility                   `json:"compatibility"`
	Query           queryDescriptor                 `json:"query"`
	Budget          atlasmap.QueryBudget            `json:"budget"`
	Usage           atlasmap.QueryUsage             `json:"usage"`
	Completeness    []semanticsnapshot.Completeness `json:"completeness"`
	Facts           packFacts                       `json:"facts"`
	Relations       []atlasmap.QueryRelation        `json:"relations"`
	Documentation   []disabledTextRecord            `json:"documentation"`
	SourceExcerpts  []disabledTextRecord            `json:"source_excerpts"`
	Omissions       []atlasmap.QueryOmission        `json:"omissions"`
	Warnings        []string                        `json:"warnings"`
}

type packageIdentity struct {
	UID        string `json:"uid"`
	Instance   string `json:"instance"`
	Coordinate string `json:"coordinate"`
	Source     string `json:"source"`
}

type compatibility struct {
	CompilerBuild string   `json:"compiler_build"`
	Language      string   `json:"language"`
	InterfaceHash string   `json:"interface_hash"`
	Generation    string   `json:"generation"`
	MapOmissions  []string `json:"map_omissions"`
}

type queryDescriptor struct {
	Kind      QueryKind               `json:"kind"`
	Subject   string                  `json:"subject"`
	Direction atlasmap.QueryDirection `json:"direction"`
	Relations []atlasmap.RelationKind `json:"relations"`
}

type packFacts struct {
	Overview *overviewFacts `json:"overview"`
	Show     *showFacts     `json:"show"`
	Around   *aroundFacts   `json:"around"`
}

type overviewFacts struct {
	Package atlasmap.PackageOverview  `json:"package"`
	Modules []semanticsnapshot.Module `json:"modules"`
}

type showFacts struct {
	Subject atlasmap.QueryNode `json:"subject"`
}

type aroundFacts struct {
	Nodes []atlasmap.AroundNode `json:"nodes"`
}

// disabledTextRecord is deliberately uninhabited by every current builder. Keeping concrete empty
// arrays in the schema makes the absence of free text visible without accepting an unsafe text path.
type disabledTextRecord struct{}

func documentFromContext(
	context atlasmap.QueryContext,
	query queryDescriptor,
	budget atlasmap.QueryBudget,
	usage atlasmap.QueryUsage,
	omissions []atlasmap.QueryOmission,
) packDocument {
	return baseDocument(
		packageIdentity{
			UID: context.PackageUID, Instance: context.PackageInstance,
			Coordinate: context.PackageCoordinate, Source: context.PackageSource,
		},
		compatibility{
			CompilerBuild: context.CompilerBuild, Language: context.Language, InterfaceHash: context.InterfaceHash,
			Generation: context.Generation, MapOmissions: nonNil(context.MapOmissions),
		},
		context.LogicalSchema,
		query,
		budget,
		usage,
		context.Completeness,
		omissions,
	)
}

func baseDocument(
	identity packageIdentity,
	compatibility compatibility,
	mapVersion string,
	query queryDescriptor,
	budget atlasmap.QueryBudget,
	usage atlasmap.QueryUsage,
	completeness []semanticsnapshot.Completeness,
	omissions []atlasmap.QueryOmission,
) packDocument {
	return packDocument{
		MapVersion: mapVersion, PackageIdentity: identity, Compatibility: compatibility, Query: query,
		Budget: budget, Usage: usage, Completeness: nonNil(completeness), Facts: packFacts{},
		Relations: make([]atlasmap.QueryRelation, 0), Documentation: make([]disabledTextRecord, 0),
		SourceExcerpts: make([]disabledTextRecord, 0), Omissions: cloneOmissions(omissions),
		Warnings: make([]string, 0),
	}
}

func encode(document packDocument, maxBytes uint64) ([]byte, error) {
	checksum, err := computeChecksum(document)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(packEnvelope{
		PackVersion: PackVersion, Encoding: PackEncoding,
		ChecksumAlgorithm: ChecksumAlgorithm, Checksum: checksum.String(), Pack: document,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Atlas Context Pack: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrPackTooLarge
	}
	return encoded, nil
}

func validateMaxBytes(maxBytes uint64) error {
	if maxBytes == 0 {
		return fmt.Errorf("Atlas Context Pack byte limit must be positive")
	}
	return nil
}

func computeChecksum(document packDocument) (Checksum, error) {
	payload, err := json.Marshal(document)
	if err != nil {
		return Checksum{}, fmt.Errorf("encode Context Pack checksum payload: %w", err)
	}
	digest := sha256.New()
	writeChecksumHeader(digest)
	writeChecksumBytes(digest, []byte(PackVersion))
	writeChecksumBytes(digest, []byte(PackEncoding))
	writeChecksumBytes(digest, payload)
	var result Checksum
	copy(result[:], digest.Sum(nil))
	return result, nil
}

func writeChecksumHeader(destination hash.Hash) {
	_, _ = destination.Write([]byte(checksumDomain))
	_, _ = destination.Write([]byte{0, 1})
}

func writeChecksumBytes(destination hash.Hash, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = destination.Write(length[:])
	_, _ = destination.Write(value)
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return make([]T, 0)
	}
	return slices.Clone(values)
}

func cloneOmissions(values []atlasmap.QueryOmission) []atlasmap.QueryOmission {
	result := nonNil(values)
	for index := range result {
		result[index].Relations = nonNil(result[index].Relations)
	}
	return result
}

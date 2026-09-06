package atlasmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

const testMapLimit = 8 << 20

func TestCodecRoundTripsCanonicalRealWorkspacePartitions(t *testing.T) {
	t.Parallel()
	partitions := testWorkspacePartitions(t)
	for _, partition := range partitions {
		header := partition.Header()
		t.Run(header.PackageUID, func(t *testing.T) {
			t.Parallel()
			first, err := Encode(partition, testMapLimit)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			second, err := Encode(partition, testMapLimit)
			if err != nil || !bytes.Equal(first, second) {
				t.Fatalf("repeated Encode() differs: %v", err)
			}
			if len(first) == 0 || first[0] != '{' || first[len(first)-1] != '}' || bytes.Contains(first, []byte("\r\n")) {
				t.Fatalf("encoded map is not compact canonical JSON")
			}
			decoded, err := Decode(first, testMapLimit)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if !reflect.DeepEqual(documentFromPartition(decoded), documentFromPartition(partition)) {
				t.Fatalf("round trip changed partition:\nwant=%+v\n got=%+v", partition, decoded)
			}
			var envelope transportEnvelope
			if err := json.Unmarshal(first, &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Schema != TransportSchema || envelope.Encoding != TransportEncoding ||
				envelope.ChecksumAlgorithm != ChecksumAlgorithm {
				t.Fatalf("transport compatibility = %+v", envelope)
			}
			if _, err := ParseChecksum(envelope.Checksum); err != nil {
				t.Fatalf("checksum = %q: %v", envelope.Checksum, err)
			}
		})
	}
}

func TestCodecRejectsBoundsCorruptionAndNoncanonicalJSON(t *testing.T) {
	t.Parallel()
	partition := testWorkspacePartitions(t)[0]
	encoded, err := Encode(partition, testMapLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Encode(partition, 0); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("zero-limit Encode() error = %v", err)
	}
	if _, err := Decode(encoded, 0); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("zero-limit Decode() error = %v", err)
	}
	if _, err := Encode(partition, uint64(len(encoded)-1)); !errors.Is(err, ErrMapTooLarge) {
		t.Fatalf("bounded Encode() error = %v", err)
	}
	if _, err := Decode(encoded, uint64(len(encoded)-1)); !errors.Is(err, ErrMapTooLarge) {
		t.Fatalf("bounded Decode() error = %v", err)
	}
	if _, err := Decode(nil, testMapLimit); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty Decode() error = %v", err)
	}

	corrupt := bytes.Clone(encoded)
	marker := []byte(partition.Header().PackageUID)
	index := bytes.Index(corrupt, marker)
	if index < 0 {
		t.Fatalf("package UID marker is absent")
	}
	corrupt[index] ^= 1
	if _, err := Decode(corrupt, testMapLimit); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("corrupt Decode() error = %v", err)
	}
	withWhitespace := append([]byte{' '}, encoded...)
	if _, err := Decode(withWhitespace, testMapLimit); !errors.Is(err, ErrNonCanonical) {
		t.Fatalf("whitespace Decode() error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`"header":{`), []byte(`"header":{"unknown":0,`), 1)
	if _, err := Decode(unknown, testMapLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field Decode() error = %v", err)
	}
	trailing := append(bytes.Clone(encoded), []byte(`{}`)...)
	if _, err := Decode(trailing, testMapLimit); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
		t.Fatalf("trailing Decode() error = %v", err)
	}
}

func TestCodecRejectsIncompatibleEnvelopeAndChecksumText(t *testing.T) {
	t.Parallel()
	encoded, err := Encode(testWorkspacePartitions(t)[0], testMapLimit)
	if err != nil {
		t.Fatal(err)
	}
	var original transportEnvelope
	if err := json.Unmarshal(encoded, &original); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		mutate   func(*transportEnvelope)
		contains string
	}{
		{name: "schema", mutate: func(value *transportEnvelope) { value.Schema = "fairyc.fymap/v2" }, contains: "schema"},
		{name: "encoding", mutate: func(value *transportEnvelope) { value.Encoding = "fairyc.fymap-json/v2" }, contains: "encoding"},
		{name: "checksum algorithm", mutate: func(value *transportEnvelope) { value.ChecksumAlgorithm = "sha1" }, contains: "checksum algorithm"},
		{name: "checksum prefix", mutate: func(value *transportEnvelope) { value.Checksum = "bad" }, contains: "checksum"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			value := original
			test.mutate(&value)
			mutated, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if _, decodeErr := Decode(mutated, testMapLimit); decodeErr == nil || !strings.Contains(decodeErr.Error(), test.contains) {
				t.Fatalf("Decode() error = %v", decodeErr)
			}
		})
	}
	for _, value := range []string{
		"", ChecksumPrefix + strings.Repeat("0", 63), ChecksumPrefix + strings.Repeat("A", 64),
		"other-" + strings.Repeat("0", 64),
	} {
		if _, parseErr := ParseChecksum(value); parseErr == nil {
			t.Fatalf("ParseChecksum(%q) succeeded", value)
		}
	}
}

func TestEncodeRejectsContradictoryLogicalRecords(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*Partition)
		match  string
	}{
		{name: "count", mutate: func(value *Partition) { value.data.header.Counts.Files++ }, match: "counts"},
		{name: "generic declaration", mutate: func(value *Partition) { value.data.declarations[0].TypeParameters = []string{"T"} }, match: "cannot preserve a generic declaration"},
		{name: "algorithm", mutate: func(value *Partition) { value.data.header.SymbolAlgorithm = "other" }, match: "algorithm"},
		{name: "package reference", mutate: func(value *Partition) {
			for index := range value.data.header.PackageReferences {
				if value.data.header.PackageReferences[index].Instance == value.data.header.PackageInstance {
					value.data.header.PackageReferences[index].UID = "other.uid"
				}
			}
		}, match: "reference"},
		{name: "duplicate file", mutate: func(value *Partition) {
			value.data.files = append(value.data.files, value.data.files[0])
			value.data.header.Counts.Files++
		}, match: "files are not canonical"},
		{name: "unknown span source", mutate: func(value *Partition) { value.data.declarations[0].Span.Source = "missing.fy" }, match: "unknown source"},
		{name: "symbol mismatch", mutate: func(value *Partition) {
			full, err := symbolidentity.ParseFullDigest(value.data.symbols[0].Full)
			if err != nil {
				panic(err)
			}
			full[0] ^= 0xff
			value.data.symbols[0].Full = full.String()
		}, match: "does not match"},
		{name: "declaration identity mismatch", mutate: func(value *Partition) {
			value.data.declarations[0].Unsafe = !value.data.declarations[0].Unsafe
		}, match: "does not match its declaration"},
		{name: "type depth", mutate: func(value *Partition) {
			typeRef := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}
			for index := 0; index <= maxTransportTypeDepth; index++ {
				typeRef = semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeOption, Arguments: []semanticsnapshot.TypeRef{typeRef}}
			}
			value.data.declarations[0].Type = typeRef
		}, match: "nesting"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			partition := testWorkspacePartitions(t)[0]
			test.mutate(&partition)
			if _, err := Encode(partition, testMapLimit); err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("Encode() error = %v", err)
			}
		})
	}
}

func TestDecodeRevalidatesChecksummedRecordPayload(t *testing.T) {
	t.Parallel()
	encoded, err := Encode(testWorkspacePartitions(t)[0], testMapLimit)
	if err != nil {
		t.Fatal(err)
	}
	var envelope transportEnvelope
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Map.Header.Counts.Files++
	checksum, err := computeTransportChecksum(envelope.Map)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Checksum = checksum.String()
	mutated, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(mutated, testMapLimit); err == nil || !strings.Contains(err.Error(), "counts") {
		t.Fatalf("Decode() error = %v", err)
	}
}

func TestEncodeRejectsCallOutsidePackageReferenceClosure(t *testing.T) {
	t.Parallel()
	var partition Partition
	found := false
	functionIndex := 0
	for _, candidate := range testWorkspacePartitions(t) {
		for index, function := range candidate.data.functions {
			if len(function.Calls) != 0 {
				partition = candidate
				functionIndex = index
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("workspace has no call fixture")
	}
	var outside packagegraph.InstanceID
	outside[0] = 0xff
	partition.data.functions[functionIndex].Calls[0].Target.PackageInstance = outside.String()
	if _, err := Encode(partition, testMapLimit); err == nil || !strings.Contains(err.Error(), "reference closure") {
		t.Fatalf("Encode() error = %v", err)
	}
}

func TestChecksumInputVector(t *testing.T) {
	t.Parallel()
	document := transportDocument{
		Header: transportHeader{LogicalSchema: "example", Features: []string{}, Dependencies: []string{}, PackageReferences: []transportPackageReference{}, Omissions: []string{}},
		Files:  []transportFile{}, Modules: []transportModule{}, Symbols: []transportSymbol{},
		Declarations: []transportDeclaration{}, Functions: []transportFunctionSummary{}, Completeness: []transportCompleteness{},
	}
	checksum, err := computeTransportChecksum(document)
	if err != nil {
		t.Fatal(err)
	}
	const want = "fymap1-sha256-f9b1e4db19b424ec1a121eec5cb18d71a56ca9da4bb886253bcdff9eb6f8e016"
	if checksum.String() != want {
		t.Fatalf("checksum vector = %s, want %s", checksum, want)
	}
}

func testWorkspacePartitions(t *testing.T) []Partition {
	t.Helper()
	workspace, result := loadQuickstart(t)
	_, partitions, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	return partitions
}

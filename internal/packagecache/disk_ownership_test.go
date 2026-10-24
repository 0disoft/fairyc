package packagecache

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

func diskOwnershipRecord() Record {
	key := Key(sha256.Sum256([]byte("disk-ownership")))
	return Record{Key: key, PublicInterface: moduleinterface.Hash(key), Artifacts: []Artifact{
		{Kind: ArtifactMIR, Data: []byte("independent MIR bytes")},
		{Kind: ArtifactObject, Data: []byte("independent object bytes")},
	}}
}

func TestDiskCodecOwnsEncodedAndDecodedArtifacts(t *testing.T) {
	record := diskOwnershipRecord()
	expected := cloneRecord(record)
	encoded, err := encodeDiskRecord(record, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for index := range record.Artifacts {
		record.Artifacts[index].Kind = "changed"
		record.Artifacts[index].Data[0] ^= 0xff
	}
	first, err := decodeDiskRecord(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	second, err := decodeDiskRecord(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	clear(encoded)
	if !recordsEqual(first, expected) || !recordsEqual(second, expected) {
		t.Fatal("codec returned bytes aliased to its input")
	}
	for index := range first.Artifacts {
		first.Artifacts[index].Kind = "changed"
		first.Artifacts[index].Data[0] ^= 0xff
	}
	if !recordsEqual(second, expected) {
		t.Fatal("decoded records share mutable artifact storage")
	}
}

func TestDiskStoreGetsOwnIndependentArtifacts(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), DiskLimits{
		MaxEntries: 4, MaxBytes: 1 << 20, MaxRecordBytes: 1 << 20, TempMaxAge: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	record := diskOwnershipRecord()
	if err := store.Put(record); err != nil {
		t.Fatal(err)
	}
	first, found := store.Get(record.Key)
	if !found {
		t.Fatal("first cache miss")
	}
	second, found := store.Get(record.Key)
	if !found {
		t.Fatal("second cache miss")
	}
	for index := range first.Artifacts {
		first.Artifacts[index].Kind = "changed"
		first.Artifacts[index].Data[0] ^= 0xff
	}
	if !recordsEqual(second, record) {
		t.Fatal("Get returned shared artifact storage")
	}
	third, found := store.Get(record.Key)
	if !found || !recordsEqual(third, record) {
		t.Fatal("caller mutation changed a future disk hit")
	}
}

func TestDiskCodecStillRejectsNoncanonicalRecords(t *testing.T) {
	encoded, err := encodeDiskRecord(diskOwnershipRecord(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for name, invalid := range map[string][]byte{
		"whitespace":          append([]byte(" "), encoded...),
		"trailing whitespace": append(bytes.Clone(encoded), '\n'),
		"trailing":            append(bytes.Clone(encoded), []byte(" {}")...),
		"trailing null":       append(bytes.Clone(encoded), []byte(" null")...),
		"unknown":             bytes.Replace(encoded, []byte(`"record":`), []byte(`"extra":true,"record":`), 1),
		"unknown record":      bytes.Replace(encoded, []byte(`"publicInterface":`), []byte(`"extra":true,"publicInterface":`), 1),
		"unknown artifact":    bytes.Replace(encoded, []byte(`"kind":`), []byte(`"extra":true,"kind":`), 1),
		"duplicate schema":    bytes.Replace(encoded, []byte(`"schema":`), []byte(`"schema":"ignored","schema":`), 1),
		"duplicate key":       bytes.Replace(encoded, []byte(`"key":`), []byte(`"key":[],"key":`), 1),
		"duplicate artifacts": bytes.Replace(encoded, []byte(`"artifacts":`), []byte(`"artifacts":[],"artifacts":`), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeDiskRecord(invalid, 1<<20); err == nil {
				t.Fatal("noncanonical record was accepted")
			}
		})
	}
}

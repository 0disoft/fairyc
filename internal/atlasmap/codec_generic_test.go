package atlasmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestCodecV2RoundTripAndVersionIsolation(t *testing.T) {
	for _, used := range []bool{false, true} {
		workspace, result := loadGenericPlanWorkspace(t, used)
		_, plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		for _, plan := range plans {
			encoded, err := EncodeV2(plan, testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeV2(encoded, uint64(len(encoded)))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(documentFromPartition(plan), documentFromPartition(decoded)) {
				t.Fatal("v2 round trip lost facts")
			}
			again, err := EncodeV2(decoded, testMapLimit)
			if err != nil || !bytes.Equal(encoded, again) {
				t.Fatalf("noncanonical v2 round trip: %v", err)
			}
			if _, err := Decode(encoded, testMapLimit); err == nil {
				t.Fatal("v1 decoded v2 bytes")
			}
			if _, err := EncodeV2(plan, uint64(len(encoded)-1)); !errors.Is(err, ErrMapTooLarge) {
				t.Fatalf("encode limit: %v", err)
			}
			if _, err := DecodeV2(encoded, uint64(len(encoded)-1)); !errors.Is(err, ErrMapTooLarge) {
				t.Fatalf("decode limit: %v", err)
			}
			if _, err := DecodeV2(append(encoded, ' '), testMapLimit); !errors.Is(err, ErrNonCanonical) {
				t.Fatalf("trailing whitespace: %v", err)
			}
			if _, err := DecodeV2(append(encoded, []byte("{}")...), testMapLimit); err == nil {
				t.Fatal("trailing JSON accepted")
			}
		}
	}
	for _, legacy := range testWorkspacePartitions(t) {
		oldBytes, err := Encode(legacy, testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeV2(oldBytes, testMapLimit); err == nil {
			t.Fatal("v2 decoded v1 bytes")
		}
		legacy.data.header.LogicalSchema = LogicalSchemaV2
		newBytes, err := EncodeV2(legacy, testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeV2(newBytes, testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
		decoded.data.header.LogicalSchema = LogicalSchema
		restored, err := Encode(decoded, testMapLimit)
		if err != nil || !bytes.Equal(oldBytes, restored) {
			t.Fatal("v1 bytes changed through ordinary v2 conversion")
		}
	}
}

func TestCodecV2RejectsResealedContradictions(t *testing.T) {
	workspace, result := loadGenericPlanWorkspace(t, true)
	_, plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*transportDocument)
	}{
		{"arity", func(d *transportDocument) { d.Specializations[0].TypeArguments = nil }},
		{"duplicate binding", func(d *transportDocument) {
			for i := range d.Specializations {
				for j := i + 1; j < len(d.Specializations); j++ {
					if d.Specializations[i].Declaration == d.Specializations[j].Declaration {
						d.Specializations[j].TypeArguments = d.Specializations[i].TypeArguments
						return
					}
				}
			}
		}},
		{"foreign owner", func(d *transportDocument) { d.Specializations[0].Declaration.PackageInstance = "missing" }},
		{"wrong summary", func(d *transportDocument) { d.Specializations[0].Function.Function = d.Functions[0].Function }},
		{"bad id", func(d *transportDocument) { d.Specializations[0].ID = "g0_invalid" }},
		{"missing call target", func(d *transportDocument) { d.Functions[0].Calls[0].Specialization = "g0_" + strings.Repeat("0", 64) }},
		{"erased call target", func(d *transportDocument) { d.Functions[0].Calls[0].Specialization = "" }},
		{"unknown concrete type", func(d *transportDocument) { d.Specializations[0].TypeArguments[0].Name = "invented" }},
		{"wrong count", func(d *transportDocument) { d.Header.Counts.Specializations++ }},
		{"ordinary template body", func(d *transportDocument) { d.Functions[0].Function = d.Specializations[0].Declaration }},
		{"unbound parameter", func(d *transportDocument) {
			for i := range d.Declarations {
				if len(d.Declarations[i].TypeParameters) > 0 {
					d.Declarations[i].Type.Name = "999"
					return
				}
			}
		}},
		{"false template completeness", func(d *transportDocument) {
			for i := range d.Completeness {
				if d.Completeness[i].Relation == "generic-template-bodies" {
					d.Completeness[i].Status = string(semanticsnapshot.CompletenessExact)
					d.Completeness[i].Reasons = nil
				}
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := documentFromPartition(plans[0])
			test.mutate(&document)
			encoded := resealV2(t, document)
			if _, err := DecodeV2(encoded, testMapLimit); err == nil {
				t.Fatal("contradictory v2 payload accepted with valid checksum")
			}
		})
	}
	document := documentFromPartition(plans[0])
	encoded := resealV2(t, document)
	unknown := bytes.Replace(encoded, []byte(`"map":{`), []byte(`"map":{"unknown":true,`), 1)
	if _, err := DecodeV2(unknown, testMapLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field: %v", err)
	}
	corrupt := bytes.Replace(encoded, []byte(`"generic-plan"`), []byte(`"different-plan"`), 1)
	// Alter a required header field without resealing; semantic validity is irrelevant here.
	if bytes.Equal(corrupt, encoded) {
		corrupt = bytes.Replace(encoded, []byte(`"target-independent"`), []byte(`"changed-target"`), 1)
	}
	if _, err := DecodeV2(corrupt, testMapLimit); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("checksum corruption: %v", err)
	}
}

func resealV2(t *testing.T, document transportDocument) []byte {
	t.Helper()
	checksum, err := computeVersionChecksum(document, TransportSchemaV2, TransportEncodingV2)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(transportEnvelope{Schema: TransportSchemaV2, Encoding: TransportEncodingV2, ChecksumAlgorithm: ChecksumAlgorithm, Checksum: checksum.String(), Map: document})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

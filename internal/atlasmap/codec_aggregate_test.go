package atlasmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestAggregateMapV3RoundTripAndIsolation(t *testing.T) {
	for _, used := range []bool{false, true} {
		workspace, result := loadAggregateMapWorkspace(t, used)
		snapshot, plans, err := PrepareWorkspaceV3(workspace, result, testCompilerBuild())
		if err != nil || len(plans) != 1 {
			t.Fatalf("plans: %v", err)
		}
		if _, _, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild()); err == nil {
			t.Fatal("v2 admitted aggregates")
		}
		plan := plans[0]
		encoded, err := EncodeV3(plan, testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeV3(encoded, uint64(len(encoded)))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(documentFromPartition(plan), documentFromPartition(decoded)) {
			t.Fatal("round trip lost aggregate facts")
		}
		expectedItems, decodedItems := snapshot.Specializations(), decoded.Specializations()
		if len(expectedItems) != len(decodedItems) {
			t.Fatal("specialization count changed")
		}
		for i := range expectedItems {
			// The wire normalizes nil and empty collections, but retains every field.
			if !reflect.DeepEqual(specializationToTransport(expectedItems[i]), specializationToTransport(decodedItems[i])) {
				t.Fatal("snapshot specialization differs from decoded records")
			}
		}
		beforeMutation := documentFromPartition(decoded)
		again, err := EncodeV3(decoded, testMapLimit)
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatalf("noncanonical round trip: %v", err)
		}
		if _, err := DecodeV2(encoded, testMapLimit); err == nil {
			t.Fatal("v2 decoded v3")
		}
		if _, err := Decode(encoded, testMapLimit); err == nil {
			t.Fatal("v1 decoded v3")
		}
		if _, err := EncodeV2(plan, testMapLimit); err == nil {
			t.Fatal("v2 encoded v3")
		}
		if _, err := EncodeV3(plan, uint64(len(encoded)-1)); !errors.Is(err, ErrMapTooLarge) {
			t.Fatalf("encode bound: %v", err)
		}
		if _, err := DecodeV3(encoded, uint64(len(encoded)-1)); !errors.Is(err, ErrMapTooLarge) {
			t.Fatalf("decode bound: %v", err)
		}
		if _, err := DecodeV3(append(encoded, ' '), testMapLimit); !errors.Is(err, ErrNonCanonical) {
			t.Fatalf("canonicality: %v", err)
		}
		if bytes.Contains(encoded, []byte("_fy_generic_")) {
			t.Fatal("generated name leaked")
		}
		changed := decoded.Specializations()
		cleanup, aggregate := 0, 0
		for i := range changed {
			if changed[i].Aggregate != nil {
				aggregate++
				changed[i].Aggregate.Fields[0].Type.Name = "corrupt"
			}
			if changed[i].Cleanup != nil {
				cleanup++
				if len(changed[i].Cleanup.Borrows) != 1 || len(changed[i].Cleanup.Calls) != 1 {
					t.Fatal("cleanup facts lost")
				}
				changed[i].Cleanup.Calls[0].Specialization = "corrupt"
			}
		}
		if used && (aggregate != 5 || cleanup != 2) {
			t.Fatalf("missing kinds: %d/%d", aggregate, cleanup)
		}
		if !reflect.DeepEqual(beforeMutation, documentFromPartition(decoded)) {
			t.Fatal("getter aliases aggregate or cleanup")
		}
	}
	workspace, result := loadGenericPlanWorkspace(t, true)
	_, plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	before, err := EncodeV2(plans[0], testMapLimit)
	if err != nil {
		t.Fatal(err)
	}
	plans[0].data.header.LogicalSchema = LogicalSchemaV3
	encoded, err := EncodeV3(plans[0], testMapLimit)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeV3(encoded, testMapLimit)
	if err != nil {
		t.Fatal(err)
	}
	decoded.data.header.LogicalSchema = LogicalSchemaV2
	after, err := EncodeV2(decoded, testMapLimit)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("v2 bytes changed: %v", err)
	}
}

func TestAggregateMapV3RejectsResealedContradictions(t *testing.T) {
	workspace, result := loadAggregateMapWorkspace(t, true)
	_, plans, err := PrepareWorkspaceV3(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	resource := func(d *transportDocument) *transportSpecialization {
		for i := range d.Specializations {
			if d.Specializations[i].Cleanup != nil {
				return &d.Specializations[i]
			}
		}
		t.Fatal("missing resource")
		return nil
	}
	for _, test := range []struct {
		name   string
		mutate func(*transportDocument)
	}{
		{"missing aggregate", func(d *transportDocument) { resource(d).Aggregate = nil }},
		{"missing cleanup", func(d *transportDocument) { resource(d).Cleanup = nil }},
		{"mixed body kinds", func(d *transportDocument) { resource(d).Function = &d.Functions[0] }},
		{"wrong resource owner", func(d *transportDocument) { resource(d).Cleanup.Function = d.Functions[0].Function }},
		{"wrong concrete field", func(d *transportDocument) { resource(d).Aggregate.Fields[0].Type.Arguments[0].Name = "bool" }},
		{"unresolved cleanup call", func(d *transportDocument) {
			resource(d).Cleanup.Calls[0].Specialization = "g0_" + strings.Repeat("0", 64)
		}},
		{"wrong provenance", func(d *transportDocument) {
			resource(d).Aggregate.Provenance = string(semanticsnapshot.ProvenanceCompilerDeclared)
		}},
		{"wrong drop kind", func(d *transportDocument) { resource(d).Aggregate.DropKind = "invented" }},
		{"empty with cleanup", func(d *transportDocument) { resource(d).Aggregate.DropKind = "empty" }},
		{"wrong relation count", func(d *transportDocument) { d.Header.Counts.Relations++ }},
		{"unknown key", func(d *transportDocument) { resource(d).Aggregate.Key.Name = "missing" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := documentFromPartition(plans[0])
			test.mutate(&document)
			if _, err := DecodeV3(resealV3(t, document), testMapLimit); err == nil {
				t.Fatal("resealed contradiction accepted")
			}
		})
	}
	encoded := resealV3(t, documentFromPartition(plans[0]))
	unknown := bytes.Replace(encoded, []byte(`"aggregate":{`), []byte(`"aggregate":{"unknown":true,`), 1)
	if _, err := DecodeV3(unknown, testMapLimit); err == nil {
		t.Fatal("unknown field accepted")
	}
	corrupt := bytes.Replace(encoded, []byte(`"non-empty"`), []byte(`"empty"`), 1)
	if _, err := DecodeV3(corrupt, testMapLimit); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("checksum: %v", err)
	}
}

func resealV3(t *testing.T, document transportDocument) []byte {
	t.Helper()
	checksum, err := computeVersionChecksum(document, TransportSchemaV3, TransportEncodingV3)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(transportEnvelope{Schema: TransportSchemaV3, Encoding: TransportEncodingV3, ChecksumAlgorithm: ChecksumAlgorithm, Checksum: checksum.String(), Map: document})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func loadAggregateMapWorkspace(t *testing.T, used bool) (workspaceloader.Workspace, compiler.Result) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	text := `module main;
pub struct Cell<T> { T value; }
resource Empty<T> { T value; cleanup {} }
pub resource Token<T> { Cell<T> payload; cleanup { let T copy = this.payload.value; let ref T view = ref copy; let T observed = finish<T>(*view); } }
fn T finish<T>(T value) { return value; }
fn Token<T> make<T>(T value) { return Token<T> { .payload = Cell<T> { .value = value } }; }
`
	if used {
		text += `fn i32 main() { let Empty<i32> empty = Empty<i32> { .value = 1 }; let Token<i32> a = make<i32>(42); let Token<u32> b = make<u32>(7); return a.payload.value; }`
	} else {
		text += `fn i32 main() { return 0; }`
	}
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.atlas.aggregate\"\nname = \"aggregate\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/18\"\nsource = \"src\"\n"
	for path, contents := range map[string]string{"fairy.toml": manifest, "src/main.fy": text} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	return workspace, result
}

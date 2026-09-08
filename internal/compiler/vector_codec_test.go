package compiler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

func TestPrivateVectorCodecRoundTrip(t *testing.T) {
	input, coordinates := vectorPackageInput(t)
	result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("codec input: %v %+v", err, result.Semantic)
	}
	const limit = 4 << 20
	encoded, err := mircodec.EncodeStorageTransfers(result.MIR, limit)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.DecodeStorageTransfers(encoded, limit)
	if err != nil || !reflect.DeepEqual(result.MIR, decoded) {
		t.Fatalf("transfer round trip: %v", err)
	}
	repeated, err := mircodec.EncodeStorageTransfers(decoded, limit)
	if err != nil || !bytes.Equal(encoded, repeated) {
		t.Fatalf("noncanonical round trip: %v", err)
	}
	if _, err := mircodec.Encode(result.MIR, limit); err == nil {
		t.Fatal("ordinary encoder admitted transfers")
	}
	if _, err := mircodec.Decode(encoded, limit); err == nil {
		t.Fatal("ordinary decoder admitted transfer envelope")
	}
	forged := bytes.Replace(encoded, []byte(mircodec.StorageTransferSchema), []byte(mircodec.Schema), 1)
	if _, err := mircodec.Decode(forged, limit); err == nil {
		t.Fatal("schema substitution bypassed ordinary verifier")
	}
	if _, err := mircodec.DecodeStorageTransfers(forged, limit); err == nil {
		t.Fatal("transfer decoder admitted ordinary schema")
	}
	if _, err := mircodec.DecodeStorageTransfers(encoded, uint64(len(encoded)-1)); !errors.Is(err, mircodec.ErrArtifactTooLarge) {
		t.Fatalf("decode byte limit: %v", err)
	}
	if _, err := mircodec.EncodeStorageTransfers(result.MIR, uint64(len(encoded)-1)); !errors.Is(err, mircodec.ErrArtifactTooLarge) {
		t.Fatalf("encode byte limit: %v", err)
	}
	if _, err := mircodec.DecodeStorageTransfers(encoded, 0); err == nil {
		t.Fatal("zero limit admitted")
	}
	if _, err := mircodec.DecodeStorageTransfers(append(bytes.Clone(encoded), '\n'), limit); !errors.Is(err, mircodec.ErrNonCanonical) {
		t.Fatalf("noncanonical artifact: %v", err)
	}
	var artifact struct {
		Schema     string      `json:"schema"`
		MIRVersion string      `json:"mirVersion"`
		Program    mir.Program `json:"program"`
	}
	if err := json.Unmarshal(encoded, &artifact); err != nil {
		t.Fatal(err)
	}
	mutated := false
	for f := range artifact.Program.Functions {
		for b := range artifact.Program.Functions[f].Blocks {
			for o := range artifact.Program.Functions[f].Blocks[b].Operations {
				operation := &artifact.Program.Functions[f].Blocks[b].Operations[o]
				if operation.Kind == mir.OperationStoragePut {
					operation.Operands = nil
					mutated = true
				}
			}
		}
	}
	if !mutated {
		t.Fatal("no transfer operation exercised")
	}
	malformed, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if rejected, err := mircodec.DecodeStorageTransfers(malformed, limit); err == nil || len(rejected.Functions) != 0 {
		t.Fatal("malformed transfer MIR published")
	}
	if _, err := mircodec.EncodeStorageTransfers(artifact.Program, limit); err == nil {
		t.Fatal("malformed transfer MIR encoded")
	}
	runVectorStorageMIRNativeLane(t, decoded, hir.QualifiedModulePath(1, "main")+"::main", vectorResourceProbe, true)
}

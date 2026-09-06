package packagecache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// GenericKey is a private source-generic compatibility key, not a v4 store key.
type GenericKey [sha256.Size]byte

func (key GenericKey) String() string { return "fycacheg0-sha256-" + hex.EncodeToString(key[:]) }

type GenericDependency struct {
	Coordinate packagegraph.CoordinateID
	Interface  moduleinterface.GenericHash
}

// GenericInputs supplements the complete existing environment and ordinary
// source-interface projection with generic facts. Dependencies must cover exactly
// Base.Dependencies in the same canonical order. Specializations are the sorted,
// unique concrete IDs emitted into this package, not global roots or catalog IDs.
type GenericInputs struct {
	Base             Inputs
	CurrentInterface moduleinterface.GenericHash
	Dependencies     []GenericDependency
	Specializations  []genericplan.SpecializationID
}

func ComputeGeneric(inputs GenericInputs) (GenericKey, error) {
	base, err := Encode(inputs.Base)
	if err != nil {
		return GenericKey{}, err
	}
	if inputs.CurrentInterface == (moduleinterface.GenericHash{}) {
		return GenericKey{}, fmt.Errorf("generic cache current interface must be set")
	}
	if len(inputs.Dependencies) != len(inputs.Base.Dependencies) {
		return GenericKey{}, fmt.Errorf("generic cache dependencies must cover every ordinary dependency")
	}
	var encoded bytes.Buffer
	writeString(&encoded, "fairyc.generic-package-cache-key/v0")
	writeUint64(&encoded, uint64(len(base)))
	encoded.Write(base)
	encoded.Write(inputs.CurrentInterface[:])
	writeUint64(&encoded, uint64(len(inputs.Dependencies)))
	for i, dependency := range inputs.Dependencies {
		if dependency.Coordinate != inputs.Base.Dependencies[i].Coordinate || dependency.Interface == (moduleinterface.GenericHash{}) {
			return GenericKey{}, fmt.Errorf("generic cache dependency %d has mismatched coordinate or missing interface", i)
		}
		encoded.Write(dependency.Coordinate[:])
		encoded.Write(dependency.Interface[:])
	}
	writeUint64(&encoded, uint64(len(inputs.Specializations)))
	for i, id := range inputs.Specializations {
		value := string(id)
		if len(value) != 67 || !strings.HasPrefix(value, "g0_") || strings.ToLower(value) != value {
			return GenericKey{}, fmt.Errorf("generic cache specialization must be a canonical v0 ID")
		}
		if _, err := hex.DecodeString(value[3:]); err != nil {
			return GenericKey{}, fmt.Errorf("generic cache specialization has invalid digest")
		}
		if i > 0 && inputs.Specializations[i-1] >= id {
			return GenericKey{}, fmt.Errorf("generic cache specializations must be sorted and unique")
		}
		writeString(&encoded, value)
	}
	return sha256.Sum256(encoded.Bytes()), nil
}

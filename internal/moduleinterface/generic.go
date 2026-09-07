package moduleinterface

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/genericplan"
)

// GenericHash is a compiler-private source interface identity, deliberately not
// interchangeable with the released v5 Hash or current package cache records.
type GenericHash [sha256.Size]byte

func (hash GenericHash) String() string { return GenericHashPrefix + hex.EncodeToString(hash[:]) }

type GenericTemplate struct {
	Module  string
	Name    string
	Closure genericplan.Digest
}

// ComputeGeneric combines a canonical ordinary descriptor (with generated
// specializations removed) and exported template closures. Input slices are not
// mutated. Template order and current instantiation requests are not interface facts.
func ComputeGeneric(ordinary Descriptor, templates []GenericTemplate) (GenericHash, error) {
	base, err := Compute(ordinary)
	if err != nil {
		return GenericHash{}, err
	}
	modules := map[string]Module{}
	for _, module := range ordinary.Modules {
		modules[module.Path] = module
	}
	templates = slices.Clone(templates)
	slices.SortFunc(templates, func(a, b GenericTemplate) int {
		if order := cmp.Compare(a.Module, b.Module); order != 0 {
			return order
		}
		return cmp.Compare(a.Name, b.Name)
	})
	var encoded bytes.Buffer
	writeString(&encoded, GenericHashEncoding)
	writeString(&encoded, HashEncoding)
	encoded.Write(base[:])
	writeUint64(&encoded, uint64(len(templates)))
	for i, template := range templates {
		module, exists := modules[template.Module]
		if !exists || template.Name == "" || template.Closure.IsZero() {
			return GenericHash{}, fmt.Errorf("invalid public generic template")
		}
		if i > 0 && template.Module == templates[i-1].Module && template.Name == templates[i-1].Name {
			return GenericHash{}, fmt.Errorf("duplicate public generic template")
		}
		for _, item := range module.Items {
			if item.Name == template.Name {
				return GenericHash{}, fmt.Errorf("generic template conflicts with ordinary public item")
			}
		}
		writeString(&encoded, template.Module)
		writeString(&encoded, template.Name)
		encoded.Write(template.Closure[:])
	}
	return sha256.Sum256(encoded.Bytes()), nil
}

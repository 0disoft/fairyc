package packagecache

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/moduleinterface"
)

func genericInputs(t *testing.T) GenericInputs {
	base := testInputs(t)
	return GenericInputs{Base: base, CurrentInterface: moduleinterface.GenericHash{1}, Dependencies: []GenericDependency{{Coordinate: base.Dependencies[0].Coordinate, Interface: moduleinterface.GenericHash{2}}}, Specializations: []genericplan.SpecializationID{genericplan.SpecializationID("g0_" + strings.Repeat("1", 64))}}
}

func TestGenericKeyCoversEnvironmentInterfacesAndSpecializations(t *testing.T) {
	baseline := genericInputs(t)
	first, err := ComputeGeneric(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseKey(first.String()); err == nil {
		t.Fatal("generic key accepted by stable v4 parser")
	}
	for _, name := range []string{"compiler", "source", "interface", "dependency", "specialization", "empty"} {
		t.Run(name, func(t *testing.T) {
			changed := genericInputs(t)
			switch name {
			case "compiler":
				changed.Base.CompilerBuild = "compiler:b"
			case "source":
				changed.Base.Source[0] ^= 1
			case "interface":
				changed.CurrentInterface[0] ^= 2
			case "dependency":
				changed.Dependencies[0].Interface[0] ^= 1
			case "specialization":
				changed.Specializations = append(changed.Specializations, genericplan.SpecializationID("g0_"+strings.Repeat("2", 64)))
			case "empty":
				changed.Specializations = nil
			}
			key, err := ComputeGeneric(changed)
			if err != nil || key == first {
				t.Fatalf("invalidation absent: %v", err)
			}
		})
	}
	if repeated, err := ComputeGeneric(baseline); err != nil || first != repeated {
		t.Fatal("nondeterministic key")
	}
}

func TestGenericKeyRejectsIncompleteOrNoncanonicalInputs(t *testing.T) {
	for _, name := range []string{"environment", "interface", "missing-dependency", "wrong-coordinate", "dependency-interface", "id", "hex", "duplicate", "order"} {
		t.Run(name, func(t *testing.T) {
			input := genericInputs(t)
			switch name {
			case "environment":
				input.Base.CompilerBuild = ""
			case "interface":
				input.CurrentInterface = moduleinterface.GenericHash{}
			case "missing-dependency":
				input.Dependencies = nil
			case "wrong-coordinate":
				input.Dependencies[0].Coordinate[0] ^= 1
			case "dependency-interface":
				input.Dependencies[0].Interface = moduleinterface.GenericHash{}
			case "id":
				input.Specializations[0] = "g0_short"
			case "hex":
				input.Specializations[0] = genericplan.SpecializationID("g0_" + strings.Repeat("z", 64))
			case "duplicate":
				input.Specializations = append(input.Specializations, input.Specializations[0])
			case "order":
				input.Specializations = append(input.Specializations, genericplan.SpecializationID("g0_"+strings.Repeat("0", 64)))
			}
			if _, err := ComputeGeneric(input); err == nil {
				t.Fatal("invalid generic key accepted")
			}
		})
	}
}

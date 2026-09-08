package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

// ValueStorageLayout describes this backend's representation, not the source
// language's layout or permission to copy, relocate, borrow or destroy a value.
type ValueStorageLayout struct {
	Target         target.ID
	Type           hir.Type
	Representation hir.Type
	SizeBytes      uint64
	AlignmentBytes uint64
}

// StorageLayout queries an admitted scalar, inert struct/enum or resource in a
// verified MIR universe using the same layout preparation as C emission. It is
// compiler-private and does not admit source sizeof, arbitrary pointers, tagged
// values, arrays or safe borrows. In particular, bool currently occupies i32
// storage in generated C even though its source layout is a different contract.
func StorageLayout(program mir.Program, value hir.Type, profile Profile) (ValueStorageLayout, error) {
	spec, err := emissionTarget(profile)
	if err != nil {
		return ValueStorageLayout{}, err
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return ValueStorageLayout{}, fmt.Errorf("C storage layout requires verified MIR: %s at %s", failures[0].Code, failures[0].Entity)
	}
	engine, err := layout.New(spec)
	if err != nil {
		return ValueStorageLayout{}, err
	}
	structures, structByType, enumerations, enumByType, err := prepareAggregateTypes(program.StructTypes, program.EnumTypes, engine)
	if err != nil {
		return ValueStorageLayout{}, err
	}
	resources, resourceByType, err := prepareResourceTypes(program.ResourceTypes, engine, structures, structByType)
	if err != nil {
		return ValueStorageLayout{}, err
	}
	canonical := canonicalCValueType(value)
	var computed layout.Result
	switch canonical {
	case hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
		computed, err = engine.Of(layout.Type{Kind: layout.KindLanguageScalar, Name: string(canonical)})
	default:
		if index, found := structByType[value]; found {
			computed = structures[index].layout
		} else if index, found := enumByType[value]; found {
			computed = enumerations[index].layout
		} else if index, found := resourceByType[value]; found {
			computed = resources[index].layout
		} else {
			return ValueStorageLayout{}, CapabilityError{Feature: fmt.Sprintf("storage layout for type %q", value)}
		}
	}
	if err != nil {
		return ValueStorageLayout{}, err
	}
	return ValueStorageLayout{Target: profile.Target, Type: value, Representation: canonical,
		SizeBytes: computed.SizeBytes, AlignmentBytes: computed.AlignmentBytes}, nil
}

func emissionTarget(profile Profile) (target.Spec, error) {
	if profile != Stage0Profile() && profile != Stage0WindowsMSVCProfile() && profile != Stage0DropTraceProfile() {
		return target.Spec{}, fmt.Errorf("unsupported C emission profile target=%q dialect=%q instrumentation=%q", profile.Target, profile.Dialect, profile.Instrumentation)
	}
	spec, found := target.Lookup(profile.Target)
	if !found {
		return target.Spec{}, fmt.Errorf("C emission target %q is missing", profile.Target)
	}
	if err := target.Validate(spec); err != nil {
		return target.Spec{}, fmt.Errorf("C emission target %q is invalid: %w", profile.Target, err)
	}
	return spec, nil
}

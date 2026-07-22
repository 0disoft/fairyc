package symbolidentity

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const signatureDomain = "fairyc.symbol-signature"

// Symbol binds one exact-snapshot declaration to its source-instance-independent SymbolId.
type Symbol struct {
	Declaration semanticsnapshot.DeclarationKey
	ID          ID
}

// FromSnapshot derives every SymbolId without observing source text, source positions, package
// instances, compiler releases, or build-local compiler identity.
func FromSnapshot(snapshot semanticsnapshot.Snapshot) ([]Symbol, error) {
	packages := snapshot.Packages()
	instanceUIDs := make(map[string]string, len(packages))
	coordinateUIDs := make(map[string]string, len(packages))
	for _, packageValue := range packages {
		if packageValue.Instance == "" || packageValue.Coordinate == "" || packageValue.UID == "" {
			return nil, fmt.Errorf("symbol identity requires complete snapshot package identity")
		}
		if previous, duplicate := instanceUIDs[packageValue.Instance]; duplicate && previous != packageValue.UID {
			return nil, fmt.Errorf("package instance %q maps to conflicting package UIDs", packageValue.Instance)
		}
		instanceUIDs[packageValue.Instance] = packageValue.UID
		if previous, duplicate := coordinateUIDs[packageValue.Coordinate]; duplicate && previous != packageValue.UID {
			return nil, fmt.Errorf("package coordinate %q maps to conflicting package UIDs", packageValue.Coordinate)
		}
		coordinateUIDs[packageValue.Coordinate] = packageValue.UID
	}

	return FromDeclarations(snapshot.Declarations(), instanceUIDs, coordinateUIDs)
}

// FromDeclarations derives and collision-checks SymbolIds from detached declarations and the
// exact package namespace maps required by their nominal types. It is used by map readers to
// reject a checksummed declaration/identity contradiction without rebuilding a compiler snapshot.
func FromDeclarations(declarations []semanticsnapshot.Declaration, instanceUIDs, coordinateUIDs map[string]string) ([]Symbol, error) {
	if len(declarations) == 0 || len(instanceUIDs) == 0 || len(coordinateUIDs) == 0 {
		return nil, fmt.Errorf("symbol identity requires declarations and package namespace mappings")
	}
	result := make([]Symbol, 0, len(declarations))
	ids := make([]ID, 0, len(declarations))
	for _, declaration := range declarations {
		packageUID, found := instanceUIDs[declaration.Key.PackageInstance]
		if !found {
			return nil, fmt.Errorf("declaration %+v has no package UID", declaration.Key)
		}
		signature, err := encodeDeclarationSignature(declaration, coordinateUIDs)
		if err != nil {
			return nil, fmt.Errorf("declaration %+v signature: %w", declaration.Key, err)
		}
		identity, err := Compute(Descriptor{
			PackageUID: packageUID, Module: declaration.Key.Module, Kind: string(declaration.Key.Kind),
			DeclarationPath: declaration.Key.Name, GenericArity: 0, Signature: signature,
		})
		if err != nil {
			return nil, fmt.Errorf("declaration %+v identity: %w", declaration.Key, err)
		}
		result = append(result, Symbol{Declaration: declaration.Key, ID: identity})
		ids = append(ids, identity)
	}
	if err := ValidateCollisionSet(ids); err != nil {
		return nil, err
	}
	slices.SortFunc(result, compareSymbol)
	return result, nil
}

// CollisionError means two distinct full digests share one visible 128-bit ID. Maps must reject
// this state rather than silently selecting either declaration.
type CollisionError struct {
	Visible VisibleID
	Left    FullDigest
	Right   FullDigest
}

func (e *CollisionError) Error() string {
	return fmt.Sprintf("symbol identity collision for %s between %s and %s", e.Visible, e.Left, e.Right)
}

// ValidateCollisionSet also rejects malformed ID pairs whose visible form is not the prefix of the
// stored full digest.
func ValidateCollisionSet(values []ID) error {
	seen := make(map[VisibleID]FullDigest, len(values))
	for _, value := range values {
		var expected VisibleID
		copy(expected[:], value.Full[:len(expected)])
		if value.Visible != expected {
			return fmt.Errorf("symbol visible id %s does not match full digest %s", value.Visible, value.Full)
		}
		if previous, duplicate := seen[value.Visible]; duplicate && previous != value.Full {
			return &CollisionError{Visible: value.Visible, Left: previous, Right: value.Full}
		}
		seen[value.Visible] = value.Full
	}
	return nil
}

func encodeDeclarationSignature(declaration semanticsnapshot.Declaration, coordinateUIDs map[string]string) ([]byte, error) {
	var encoded bytes.Buffer
	encoded.WriteString(signatureDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(1)
	writeString(&encoded, 'k', string(declaration.Key.Kind))
	writeBool(&encoded, 'u', declaration.Unsafe)
	writeBool(&encoded, 'f', declaration.Foreign)
	writeBool(&encoded, 'c', declaration.ReprC)
	writeString(&encoded, 'a', declaration.ABI)
	writeString(&encoded, 'l', declaration.LinkName)
	if err := writeOptionalType(&encoded, 'o', declaration.Owner, coordinateUIDs); err != nil {
		return nil, err
	}
	writeString(&encoded, 'r', declaration.Receiver)
	if err := writeType(&encoded, 't', declaration.Type, coordinateUIDs); err != nil {
		return nil, err
	}
	encoded.WriteByte('p')
	writeUint64(&encoded, uint64(len(declaration.Parameters)))
	for _, parameter := range declaration.Parameters {
		if err := writeType(&encoded, 't', parameter.Type, coordinateUIDs); err != nil {
			return nil, err
		}
	}
	if err := writeSortedStrings(&encoded, 'e', declaration.Effects); err != nil {
		return nil, err
	}
	encoded.WriteByte('d')
	writeUint64(&encoded, uint64(len(declaration.Fields)))
	for _, field := range declaration.Fields {
		writeString(&encoded, 'n', field.Name)
		if err := writeType(&encoded, 't', field.Type, coordinateUIDs); err != nil {
			return nil, err
		}
	}
	encoded.WriteByte('v')
	writeUint64(&encoded, uint64(len(declaration.Variants)))
	for _, variant := range declaration.Variants {
		writeString(&encoded, 'n', variant.Name)
		encoded.WriteByte('d')
		writeUint64(&encoded, uint64(len(variant.Fields)))
		for _, field := range variant.Fields {
			writeString(&encoded, 'n', field.Name)
			if err := writeType(&encoded, 't', field.Type, coordinateUIDs); err != nil {
				return nil, err
			}
		}
	}
	writeString(&encoded, 'x', declaration.DropKind)
	if err := writeSortedStrings(&encoded, 'y', declaration.DropEffects); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func writeOptionalType(buffer *bytes.Buffer, tag byte, value *semanticsnapshot.TypeRef, coordinateUIDs map[string]string) error {
	buffer.WriteByte(tag)
	if value == nil {
		buffer.WriteByte(0)
		return nil
	}
	buffer.WriteByte(1)
	return writeType(buffer, 't', *value, coordinateUIDs)
}

func writeType(buffer *bytes.Buffer, tag byte, value semanticsnapshot.TypeRef, coordinateUIDs map[string]string) error {
	buffer.WriteByte(tag)
	writeString(buffer, 'k', string(value.Kind))
	writeString(buffer, 'n', value.Name)
	packageUID := ""
	if value.PackageCoordinate != "" {
		var found bool
		packageUID, found = coordinateUIDs[value.PackageCoordinate]
		if !found {
			return fmt.Errorf("type %q references unknown package coordinate %q", value.Name, value.PackageCoordinate)
		}
	}
	writeString(buffer, 'u', packageUID)
	writeBool(buffer, 'm', value.Mutable)
	buffer.WriteByte('l')
	writeUint64(buffer, value.Length)
	buffer.WriteByte('a')
	writeUint64(buffer, uint64(len(value.Arguments)))
	for _, argument := range value.Arguments {
		if err := writeType(buffer, 't', argument, coordinateUIDs); err != nil {
			return err
		}
	}
	return nil
}

func writeBool(buffer *bytes.Buffer, tag byte, value bool) {
	buffer.WriteByte(tag)
	if value {
		buffer.WriteByte(1)
		return
	}
	buffer.WriteByte(0)
}

func writeSortedStrings(buffer *bytes.Buffer, tag byte, values []string) error {
	if !slices.IsSorted(values) {
		return fmt.Errorf("signature values for tag %q are not sorted", tag)
	}
	buffer.WriteByte(tag)
	writeUint64(buffer, uint64(len(values)))
	for index, value := range values {
		if index > 0 && values[index-1] == value {
			return fmt.Errorf("signature values for tag %q are duplicated", tag)
		}
		writeString(buffer, 's', value)
	}
	return nil
}

func compareSymbol(left, right Symbol) int {
	if value := bytes.Compare(left.ID.Full[:], right.ID.Full[:]); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Declaration.PackageInstance, right.Declaration.PackageInstance); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Declaration.Module, right.Declaration.Module); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Declaration.Kind, right.Declaration.Kind); value != 0 {
		return value
	}
	return cmp.Compare(left.Declaration.Name, right.Declaration.Name)
}

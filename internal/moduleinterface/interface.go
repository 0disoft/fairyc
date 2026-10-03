// Package moduleinterface extracts and hashes the source-visible interface of one typed package.
package moduleinterface

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const (
	HashAlgorithm = "sha256"
	HashEncoding  = "fairyc.public-interface/v5"
	HashPrefix    = "fyif5-sha256-"
)

const interfaceDomain = "fairyc.public-interface"

// Hash is the full digest of one canonical public-interface descriptor.
type Hash [sha256.Size]byte

func (hash Hash) String() string {
	return HashPrefix + hex.EncodeToString(hash[:])
}

// MarshalText implements encoding.TextMarshaler for cache and map metadata.
func (hash Hash) MarshalText() ([]byte, error) {
	return []byte(hash.String()), nil
}

// UnmarshalText accepts only the complete canonical lowercase representation.
func (hash *Hash) UnmarshalText(text []byte) error {
	parsed, err := ParseHash(string(text))
	if err != nil {
		return err
	}
	*hash = parsed
	return nil
}

// ParseHash parses the complete canonical public-interface hash.
func ParseHash(value string) (Hash, error) {
	var result Hash
	if !strings.HasPrefix(value, HashPrefix) {
		return result, fmt.Errorf("public interface hash has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, HashPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("public interface hash must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode public interface hash: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// Descriptor is the declaration-order-independent public source interface of one package.
type Descriptor struct {
	StateLayouts []StateLayout
	Modules      []Module
}

// StateLayout is one package-visible persistent-state compatibility identity.
type StateLayout struct {
	ContractUID string
	Hash        contractstate.LayoutHash
}

// Module names one source-visible module and its sorted public items.
type Module struct {
	Path  string
	Items []Item
}

// Item contains only source-visible facts that can affect another package.
type Item struct {
	Kind       string
	Name       string
	ReprC      bool
	Unsafe     bool
	MustUse    bool
	ABI        string
	LinkName   string
	OwnerType  string
	Effects    []string
	Abilities  []string
	Parameters []string
	ResultType string
	Fields     []Field
	Variants   []Variant
}

// Field is one ordered public aggregate or enum-payload field.
type Field struct {
	Name string
	Type string
}

// Variant is one ordered public enum variant.
type Variant struct {
	Name   string
	Fields []Field
}

// Extract projects one fully typed package into its deterministic public interface. instances maps
// disposable HIR package numbers to source-independent package coordinates; the current package is rendered as
// self so private source or version changes do not invalidate an unchanged interface.
func Extract(value hir.Package, packageID hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (Descriptor, error) {
	if _, found := coordinates[packageID]; !found {
		return Descriptor{}, fmt.Errorf("package %d has no PackageCoordinateId", packageID)
	}
	publicTypes := make(map[hir.Type]bool)
	for _, module := range value.Modules {
		if module.Package != packageID {
			continue
		}
		for _, structure := range module.Structs {
			publicTypes[structure.Type] = structure.Public
		}
		for _, enumeration := range module.Enums {
			publicTypes[enumeration.Type] = enumeration.Public
		}
		for _, resource := range module.Resources {
			publicTypes[resource.Type] = resource.Public
		}
	}

	loweredLayout, hasStateLayout, err := contractstatemir.LowerHIRLayoutForPackage(value, packageID)
	if err != nil {
		return Descriptor{}, fmt.Errorf("lower contract state layouts: %w", err)
	}
	descriptor := Descriptor{}
	if hasStateLayout {
		descriptor.StateLayouts = append(descriptor.StateLayouts, StateLayout{
			ContractUID: loweredLayout.ContractUID,
			Hash:        loweredLayout.Hash,
		})
	}
	seenModules := make(map[string]struct{})
	for _, sourceModule := range value.Modules {
		if sourceModule.Package != packageID {
			continue
		}
		if sourceModule.Path == "" {
			return Descriptor{}, fmt.Errorf("package %d contains an empty module path", packageID)
		}
		if _, duplicate := seenModules[sourceModule.Path]; duplicate {
			return Descriptor{}, fmt.Errorf("package %d contains duplicate module %q", packageID, sourceModule.Path)
		}
		seenModules[sourceModule.Path] = struct{}{}
		module, err := extractModule(sourceModule, packageID, coordinates, publicTypes)
		if err != nil {
			return Descriptor{}, fmt.Errorf("module %q: %w", sourceModule.Path, err)
		}
		descriptor.Modules = append(descriptor.Modules, module)
	}
	if len(descriptor.Modules) == 0 {
		return Descriptor{}, fmt.Errorf("package %d has no typed modules", packageID)
	}
	slices.SortFunc(descriptor.Modules, func(left, right Module) int {
		return strings.Compare(left.Path, right.Path)
	})
	return descriptor, nil
}

func extractModule(source hir.Module, packageID hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID, publicTypes map[hir.Type]bool) (Module, error) {
	result := Module{Path: source.Path}
	for _, structure := range source.Structs {
		if !structure.Public {
			continue
		}
		item := Item{Kind: "struct", Name: structure.Name, ReprC: structure.ReprC}
		fields, err := stableStructFields(structure.Fields, packageID, coordinates, publicTypes)
		if err != nil {
			return Module{}, fmt.Errorf("public struct %q: %w", structure.Name, err)
		}
		item.Fields = fields
		result.Items = append(result.Items, item)
	}
	for _, enumeration := range source.Enums {
		if !enumeration.Public {
			continue
		}
		item := Item{Kind: "enum", Name: enumeration.Name}
		for _, variant := range enumeration.Variants {
			stableVariant := Variant{Name: variant.Name}
			for _, field := range variant.Fields {
				stable, err := stableType(field.Type, packageID, coordinates, publicTypes)
				if err != nil {
					return Module{}, fmt.Errorf("public enum %q variant %q field %q: %w", enumeration.Name, variant.Name, field.Name, err)
				}
				stableVariant.Fields = append(stableVariant.Fields, Field{Name: field.Name, Type: stable})
			}
			item.Variants = append(item.Variants, stableVariant)
		}
		result.Items = append(result.Items, item)
	}
	for _, resource := range source.Resources {
		if !resource.Public {
			continue
		}
		abilities, err := stableResourceAbilities(resource.Abilities)
		if err != nil {
			return Module{}, fmt.Errorf("public resource %q: %w", resource.Name, err)
		}
		effects, err := stableEffects(resource.DropEffects)
		if err != nil {
			return Module{}, fmt.Errorf("public resource %q: %w", resource.Name, err)
		}
		item := Item{Kind: "resource", Name: resource.Name, Effects: effects, Abilities: abilities}
		result.Items = append(result.Items, item)
	}
	for _, foreign := range source.ForeignFunctions {
		if !foreign.Public {
			continue
		}
		effects, err := stableEffects(foreign.Effects)
		if err != nil {
			return Module{}, fmt.Errorf("public foreign function %q: %w", foreign.Name, err)
		}
		item := Item{
			Kind: "foreign-function", Name: foreign.Name, Unsafe: true, ABI: foreign.ABI,
			LinkName: foreign.LinkName, Effects: effects, MustUse: foreign.MustUse,
		}
		for _, parameter := range foreign.Parameters {
			stable, err := stableType(parameter.Type, packageID, coordinates, publicTypes)
			if err != nil {
				return Module{}, fmt.Errorf("public foreign function %q parameter: %w", foreign.Name, err)
			}
			item.Parameters = append(item.Parameters, stable)
		}
		stable, err := stableType(foreign.ResultType, packageID, coordinates, publicTypes)
		if err != nil {
			return Module{}, fmt.Errorf("public foreign function %q result: %w", foreign.Name, err)
		}
		item.ResultType = stable
		result.Items = append(result.Items, item)
	}
	for _, function := range source.Functions {
		if !function.Public || function.DropGlueFor != "" {
			continue
		}
		effects, err := stableEffects(function.Effects)
		if err != nil {
			return Module{}, fmt.Errorf("public function %q: %w", function.Name, err)
		}
		item := Item{Kind: "function", Name: function.Name, Unsafe: function.Unsafe, MustUse: function.MustUse, Effects: effects}
		if function.OwnerType != "" {
			stable, err := stableType(function.OwnerType, packageID, coordinates, publicTypes)
			if err != nil {
				return Module{}, fmt.Errorf("public function %q owner: %w", function.Name, err)
			}
			item.OwnerType = stable
		}
		for _, parameter := range function.Parameters {
			stable, err := stableType(parameter.Type, packageID, coordinates, publicTypes)
			if err != nil {
				return Module{}, fmt.Errorf("public function %q parameter: %w", function.Name, err)
			}
			item.Parameters = append(item.Parameters, stable)
		}
		stable, err := stableType(function.ResultType, packageID, coordinates, publicTypes)
		if err != nil {
			return Module{}, fmt.Errorf("public function %q result: %w", function.Name, err)
		}
		item.ResultType = stable
		result.Items = append(result.Items, item)
	}
	slices.SortFunc(result.Items, func(left, right Item) int {
		if byKind := strings.Compare(left.Kind, right.Kind); byKind != 0 {
			return byKind
		}
		return strings.Compare(left.Name, right.Name)
	})
	for index := 1; index < len(result.Items); index++ {
		if result.Items[index-1].Kind == result.Items[index].Kind && result.Items[index-1].Name == result.Items[index].Name {
			return Module{}, fmt.Errorf("duplicate public %s %q", result.Items[index].Kind, result.Items[index].Name)
		}
	}
	return result, nil
}

func stableResourceAbilities(values []hir.ResourceAbility) ([]string, error) {
	if len(values) == 0 {
		values = []hir.ResourceAbility{hir.ResourceAbilityDrop}
	}
	canonical := []hir.ResourceAbility{
		hir.ResourceAbilityDrop,
		hir.ResourceAbilityStore,
		hir.ResourceAbilityKey,
	}
	position := 0
	result := make([]string, 0, len(values))
	for _, value := range values {
		for position < len(canonical) && canonical[position] != value {
			position++
		}
		if position == len(canonical) {
			return nil, fmt.Errorf("resource abilities are unknown, duplicated, or noncanonical")
		}
		result = append(result, string(value))
		position++
	}
	if slices.Contains(values, hir.ResourceAbilityKey) && !slices.Contains(values, hir.ResourceAbilityStore) {
		return nil, fmt.Errorf("resource ability key requires store")
	}
	return result, nil
}

func stableStructFields(fields []hir.StructField, packageID hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID, publicTypes map[hir.Type]bool) ([]Field, error) {
	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		stable, err := stableType(field.Type, packageID, coordinates, publicTypes)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", field.Name, err)
		}
		result = append(result, Field{Name: field.Name, Type: stable})
	}
	return result, nil
}

func stableType(value hir.Type, packageID hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID, publicTypes map[hir.Type]bool) (string, error) {
	if element, slice := hir.IntegerViewElement(value); slice {
		return string(element) + "[]", nil
	}
	if builtinType(value) {
		return string(value), nil
	}
	if element, mutable, ok := hir.BorrowElement(value); ok {
		stable, err := stableType(element, packageID, coordinates, publicTypes)
		if err != nil {
			return "", err
		}
		if mutable {
			return "ref mut " + stable, nil
		}
		return "ref " + stable, nil
	}
	if element, mutable, ok := hir.RawPointerElement(value); ok {
		stable, err := stableType(element, packageID, coordinates, publicTypes)
		if err != nil {
			return "", err
		}
		if mutable {
			return "ptr mut " + stable, nil
		}
		return "ptr " + stable, nil
	}
	if element, length, ok := hir.FixedArrayParts(value); ok {
		stable, err := stableType(element, packageID, coordinates, publicTypes)
		if err != nil {
			return "", err
		}
		return stable + "[" + strconv.FormatUint(length, 10) + "]", nil
	}
	if element, ok := hir.OptionElement(value); ok {
		stable, err := stableType(element, packageID, coordinates, publicTypes)
		if err != nil {
			return "", err
		}
		return "Option<" + stable + ">", nil
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		stableSuccess, err := stableType(success, packageID, coordinates, publicTypes)
		if err != nil {
			return "", err
		}
		stableFailure, err := stableType(failure, packageID, coordinates, publicTypes)
		if err != nil {
			return "", err
		}
		return "Result<" + stableSuccess + ", " + stableFailure + ">", nil
	}

	typePackage, name, ok := userTypeParts(value, packageID)
	if !ok {
		return "", fmt.Errorf("unsupported canonical type %q", value)
	}
	if typePackage == packageID {
		if public, known := publicTypes[value]; !known || !public {
			return "", fmt.Errorf("type %q is not a public type of package %d", value, packageID)
		}
		return "self::" + name, nil
	}
	coordinate, found := coordinates[typePackage]
	if !found {
		return "", fmt.Errorf("type %q references package %d without PackageCoordinateId", value, typePackage)
	}
	return "dep[" + coordinate.String() + "]::" + name, nil
}

func userTypeParts(value hir.Type, current hir.PackageID) (hir.PackageID, string, bool) {
	text := string(value)
	if strings.HasPrefix(text, "$p") {
		separator := strings.Index(text, "::")
		if separator <= 2 || separator+2 >= len(text) {
			return 0, "", false
		}
		parsed, err := strconv.ParseUint(text[2:separator], 10, 32)
		if err != nil || parsed == 0 {
			return 0, "", false
		}
		return hir.PackageID(parsed), text[separator+2:], true
	}
	if !strings.Contains(text, "::") {
		return 0, "", false
	}
	return current, text, true
}

func builtinType(value hir.Type) bool {
	_, integerSlice := hir.IntegerViewElement(value)
	return value == hir.TypeVoid || value == hir.TypeBool || value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 ||
		value == hir.TypeU32 || value == hir.TypeUSize || integerSlice || value == hir.TypeStr ||
		value == hir.TypeCStr || value == hir.TypeRegion || hir.IsCScalar(value)
}

func stableEffects(values []hir.Effect) ([]string, error) {
	if !slices.IsSorted(values) {
		return nil, fmt.Errorf("effects are not sorted")
	}
	result := make([]string, len(values))
	for index, value := range values {
		if index > 0 && values[index-1] == value {
			return nil, fmt.Errorf("effects are not unique")
		}
		result[index] = string(value)
	}
	return result, nil
}

// Encode returns the deterministic v5 bytes hashed for cache invalidation.
func Encode(value Descriptor) ([]byte, error) {
	if err := validateDescriptor(value); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	encoded.WriteString(interfaceDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(5)
	writeUint64(&encoded, uint64(len(value.StateLayouts)))
	for _, layout := range value.StateLayouts {
		writeString(&encoded, layout.ContractUID)
		encoded.Write(layout.Hash[:])
	}
	writeUint64(&encoded, uint64(len(value.Modules)))
	for _, module := range value.Modules {
		writeString(&encoded, module.Path)
		writeUint64(&encoded, uint64(len(module.Items)))
		for _, item := range module.Items {
			writeItem(&encoded, item)
		}
	}
	return encoded.Bytes(), nil
}

// Compute validates and hashes one canonical descriptor.
func Compute(value Descriptor) (Hash, error) {
	encoded, err := Encode(value)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(encoded), nil
}

// ComputePackages extracts and hashes one exact public interface per requested build-local package.
// Input package order is ignored; duplicate IDs and missing coordinate metadata fail closed.
func ComputePackages(value hir.Package, packageIDs []hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.PackageID]Hash, error) {
	ordered := slices.Clone(packageIDs)
	if len(ordered) == 0 {
		return nil, fmt.Errorf("public interface package set is empty")
	}
	slices.Sort(ordered)
	for index, packageID := range ordered {
		if index > 0 && ordered[index-1] == packageID {
			return nil, fmt.Errorf("public interface package %d is duplicated", packageID)
		}
		if _, found := coordinates[packageID]; !found {
			return nil, fmt.Errorf("package %d has no PackageCoordinateId", packageID)
		}
	}
	result := make(map[hir.PackageID]Hash, len(ordered))
	for _, packageID := range ordered {
		descriptor, err := Extract(value, packageID, coordinates)
		if err != nil {
			return nil, fmt.Errorf("extract public interface for package %d: %w", packageID, err)
		}
		hash, err := Compute(descriptor)
		if err != nil {
			return nil, fmt.Errorf("hash public interface for package %d: %w", packageID, err)
		}
		result[packageID] = hash
	}
	return result, nil
}

func validateDescriptor(value Descriptor) error {
	if len(value.StateLayouts) > 1 {
		return fmt.Errorf("public interface has %d state layouts; at most one is allowed", len(value.StateLayouts))
	}
	for index, layout := range value.StateLayouts {
		if _, err := contractstate.NewLayoutBuilder(layout.ContractUID); err != nil {
			return fmt.Errorf("state layout %d: %w", index, err)
		}
		if layout.Hash == (contractstate.LayoutHash{}) {
			return fmt.Errorf("state layout %q has an empty hash", layout.ContractUID)
		}
		if index > 0 && value.StateLayouts[index-1].ContractUID >= layout.ContractUID {
			return fmt.Errorf("state layouts are not sorted and unique")
		}
	}
	if len(value.Modules) == 0 {
		return fmt.Errorf("public interface has no modules")
	}
	for index, module := range value.Modules {
		if module.Path == "" || index > 0 && value.Modules[index-1].Path >= module.Path {
			return fmt.Errorf("public interface modules are not sorted and unique")
		}
		for itemIndex, item := range module.Items {
			if item.Kind == "" || item.Name == "" || itemIndex > 0 && compareItem(module.Items[itemIndex-1], item) >= 0 {
				return fmt.Errorf("module %q public items are not sorted and unique", module.Path)
			}
			if err := validateItemAbilities(item); err != nil {
				return fmt.Errorf("module %q public %s %q: %w", module.Path, item.Kind, item.Name, err)
			}
		}
	}
	return nil
}

func validateItemAbilities(item Item) error {
	if item.Kind != "resource" {
		if len(item.Abilities) != 0 {
			return fmt.Errorf("non-resource item carries resource abilities")
		}
		return nil
	}
	if len(item.Abilities) == 0 {
		return fmt.Errorf("resource ability set is empty")
	}
	values := make([]hir.ResourceAbility, len(item.Abilities))
	for index, value := range item.Abilities {
		values[index] = hir.ResourceAbility(value)
	}
	_, err := stableResourceAbilities(values)
	return err
}

func compareItem(left, right Item) int {
	if byKind := strings.Compare(left.Kind, right.Kind); byKind != 0 {
		return byKind
	}
	return strings.Compare(left.Name, right.Name)
}

func writeItem(buffer *bytes.Buffer, item Item) {
	writeString(buffer, item.Kind)
	writeString(buffer, item.Name)
	writeBool(buffer, item.ReprC)
	writeBool(buffer, item.Unsafe)
	writeBool(buffer, item.MustUse)
	writeString(buffer, item.ABI)
	writeString(buffer, item.LinkName)
	writeString(buffer, item.OwnerType)
	writeStrings(buffer, item.Effects)
	writeStrings(buffer, item.Abilities)
	writeStrings(buffer, item.Parameters)
	writeString(buffer, item.ResultType)
	writeUint64(buffer, uint64(len(item.Fields)))
	for _, field := range item.Fields {
		writeString(buffer, field.Name)
		writeString(buffer, field.Type)
	}
	writeUint64(buffer, uint64(len(item.Variants)))
	for _, variant := range item.Variants {
		writeString(buffer, variant.Name)
		writeUint64(buffer, uint64(len(variant.Fields)))
		for _, field := range variant.Fields {
			writeString(buffer, field.Name)
			writeString(buffer, field.Type)
		}
	}
}

func writeStrings(buffer *bytes.Buffer, values []string) {
	writeUint64(buffer, uint64(len(values)))
	for _, value := range values {
		writeString(buffer, value)
	}
}

func writeString(buffer *bytes.Buffer, value string) {
	writeUint64(buffer, uint64(len(value)))
	buffer.WriteString(value)
}

func writeBool(buffer *bytes.Buffer, value bool) {
	if value {
		buffer.WriteByte(1)
		return
	}
	buffer.WriteByte(0)
}

func writeUint64(buffer *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buffer.Write(raw[:])
}

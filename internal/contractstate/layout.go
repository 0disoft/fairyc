package contractstate

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
)

const (
	maxContractUIDBytes = 255
	maxFieldNameBytes   = 127
	maxTypeIDBytes      = 127
)

var ErrIncompatibleLayout = errors.New("contract state layout is not append-compatible")

// FieldKind separates one direct cell from one key-derived map namespace.
type FieldKind uint8

const (
	FieldCell FieldKind = iota + 1
	FieldMap
)

func (kind FieldKind) String() string {
	switch kind {
	case FieldCell:
		return "cell"
	case FieldMap:
		return "map"
	default:
		return fmt.Sprintf("field-kind-%d", kind)
	}
}

// Key is one target-neutral persistent-state address. Its derivation is domain
// separated and independent of Go map iteration or host layout.
type Key [sha256.Size]byte

func (key Key) String() string {
	return "fycs-key1-" + hex.EncodeToString(key[:])
}

// LayoutHash identifies the complete ordered persistent-state schema.
type LayoutHash [sha256.Size]byte

func (hash LayoutHash) String() string {
	return "fycs-layout1-sha256-" + hex.EncodeToString(hash[:])
}

// Field is one ordered layout declaration. Prefix is the cell key or map
// namespace root derived from the complete field identity.
type Field struct {
	Index       uint32
	Name        string
	Kind        FieldKind
	KeyTypeID   string
	ValueTypeID string
	Prefix      Key
}

// Layout is an immutable ordered schema. Existing fields retain their exact
// indexes and prefixes across an append-only extension.
type Layout struct {
	contractUID string
	fields      []Field
	hash        LayoutHash
}

func (layout Layout) ContractUID() string { return layout.contractUID }
func (layout Layout) Hash() LayoutHash    { return layout.hash }
func (layout Layout) Fields() []Field     { return slices.Clone(layout.fields) }

func (layout Layout) Field(name string) (Field, bool) {
	for _, field := range layout.fields {
		if field.Name == name {
			return field, true
		}
	}
	return Field{}, false
}

// Extends rejects UID changes, field removal, reordering, renaming, kind
// changes, or codec changes. Appending new fields is the only admitted upgrade.
func (layout Layout) Extends(previous Layout) error {
	if err := validateLayout(layout); err != nil {
		return fmt.Errorf("next layout: %w", err)
	}
	if err := validateLayout(previous); err != nil {
		return fmt.Errorf("previous layout: %w", err)
	}
	if layout.contractUID != previous.contractUID {
		return fmt.Errorf("%w: contract UID changed from %q to %q", ErrIncompatibleLayout, previous.contractUID, layout.contractUID)
	}
	if len(layout.fields) < len(previous.fields) {
		return fmt.Errorf("%w: field count shrank from %d to %d", ErrIncompatibleLayout, len(previous.fields), len(layout.fields))
	}
	for index, oldField := range previous.fields {
		newField := layout.fields[index]
		if newField != oldField {
			return fmt.Errorf("%w: field %d changed from %+v to %+v", ErrIncompatibleLayout, index, oldField, newField)
		}
	}
	return nil
}

// LayoutBuilder constructs one ordered layout and seals after Build.
type LayoutBuilder struct {
	contractUID string
	fields      []Field
	names       map[string]struct{}
	sealed      bool
}

func NewLayoutBuilder(contractUID string) (*LayoutBuilder, error) {
	if err := validateContractUID(contractUID); err != nil {
		return nil, err
	}
	return &LayoutBuilder{contractUID: contractUID, names: make(map[string]struct{})}, nil
}

// Build seals the builder and computes the complete layout digest.
func (builder *LayoutBuilder) Build() (Layout, error) {
	if builder == nil {
		return Layout{}, fmt.Errorf("contract state layout builder is nil")
	}
	if builder.sealed {
		return Layout{}, fmt.Errorf("contract state layout builder is already sealed")
	}
	builder.sealed = true
	layout := Layout{
		contractUID: builder.contractUID,
		fields:      slices.Clone(builder.fields),
	}
	layout.hash = computeLayoutHash(layout.contractUID, layout.fields)
	return layout, validateLayout(layout)
}

func (builder *LayoutBuilder) addField(name string, kind FieldKind, keyTypeID, valueTypeID string) (Field, error) {
	if builder == nil {
		return Field{}, fmt.Errorf("contract state layout builder is nil")
	}
	if builder.sealed {
		return Field{}, fmt.Errorf("contract state layout builder is sealed")
	}
	if err := validateFieldName(name); err != nil {
		return Field{}, err
	}
	if _, exists := builder.names[name]; exists {
		return Field{}, fmt.Errorf("contract state field %q is already declared", name)
	}
	if len(builder.fields) >= math.MaxUint32 {
		return Field{}, fmt.Errorf("contract state field count exceeds uint32")
	}
	if kind == FieldCell && keyTypeID != "" {
		return Field{}, fmt.Errorf("cell %q cannot declare a key codec", name)
	}
	if kind == FieldMap {
		if err := validateTypeID(keyTypeID); err != nil {
			return Field{}, fmt.Errorf("map %q key codec: %w", name, err)
		}
	}
	if err := validateTypeID(valueTypeID); err != nil {
		return Field{}, fmt.Errorf("field %q value codec: %w", name, err)
	}
	field := Field{
		Index:       uint32(len(builder.fields)),
		Name:        name,
		Kind:        kind,
		KeyTypeID:   keyTypeID,
		ValueTypeID: valueTypeID,
	}
	field.Prefix = deriveFieldPrefix(builder.contractUID, field)
	builder.fields = append(builder.fields, field)
	builder.names[name] = struct{}{}
	return field, nil
}

func deriveFieldPrefix(contractUID string, field Field) Key {
	payload := []byte("fairyc-contract-state-field/v1")
	payload = appendCanonicalString(payload, contractUID)
	payload = appendU32(payload, field.Index)
	payload = append(payload, byte(field.Kind))
	payload = appendCanonicalString(payload, field.Name)
	payload = appendCanonicalString(payload, field.KeyTypeID)
	payload = appendCanonicalString(payload, field.ValueTypeID)
	return sha256.Sum256(payload)
}

func deriveMapEntryKey(prefix Key, encodedKey []byte) Key {
	payload := []byte("fairyc-contract-state-map-entry/v1")
	payload = append(payload, prefix[:]...)
	payload = appendCanonicalBytes(payload, encodedKey)
	return sha256.Sum256(payload)
}

func computeLayoutHash(contractUID string, fields []Field) LayoutHash {
	payload := []byte("fairyc-contract-state-layout/v1")
	payload = appendCanonicalString(payload, contractUID)
	payload = appendU32(payload, uint32(len(fields)))
	for _, field := range fields {
		payload = appendU32(payload, field.Index)
		payload = append(payload, byte(field.Kind))
		payload = appendCanonicalString(payload, field.Name)
		payload = appendCanonicalString(payload, field.KeyTypeID)
		payload = appendCanonicalString(payload, field.ValueTypeID)
		payload = append(payload, field.Prefix[:]...)
	}
	return sha256.Sum256(payload)
}

func validateLayout(layout Layout) error {
	if err := validateContractUID(layout.contractUID); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(layout.fields))
	for index, field := range layout.fields {
		if field.Index != uint32(index) {
			return fmt.Errorf("contract state field %q has index %d, expected %d", field.Name, field.Index, index)
		}
		if err := validateFieldName(field.Name); err != nil {
			return err
		}
		if _, exists := seen[field.Name]; exists {
			return fmt.Errorf("contract state field %q is duplicated", field.Name)
		}
		seen[field.Name] = struct{}{}
		if field.Kind != FieldCell && field.Kind != FieldMap {
			return fmt.Errorf("contract state field %q has invalid kind %d", field.Name, field.Kind)
		}
		if field.Kind == FieldCell && field.KeyTypeID != "" {
			return fmt.Errorf("contract state cell %q has a key codec", field.Name)
		}
		if field.Kind == FieldMap {
			if err := validateTypeID(field.KeyTypeID); err != nil {
				return err
			}
		}
		if err := validateTypeID(field.ValueTypeID); err != nil {
			return err
		}
		if field.Prefix != deriveFieldPrefix(layout.contractUID, field) {
			return fmt.Errorf("contract state field %q has a noncanonical prefix", field.Name)
		}
	}
	if layout.hash != computeLayoutHash(layout.contractUID, layout.fields) {
		return fmt.Errorf("contract state layout hash is inconsistent")
	}
	return nil
}

func validateContractUID(value string) error {
	if len(value) == 0 || len(value) > maxContractUIDBytes {
		return fmt.Errorf("contract state UID must contain 1..%d ASCII bytes", maxContractUIDBytes)
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if isASCIIAlphaNumeric(character) || character == '.' || character == '-' || character == '_' {
			continue
		}
		return fmt.Errorf("contract state UID %q contains invalid byte 0x%02x", value, character)
	}
	return nil
}

func validateFieldName(value string) error {
	if len(value) == 0 || len(value) > maxFieldNameBytes {
		return fmt.Errorf("contract state field name must contain 1..%d ASCII bytes", maxFieldNameBytes)
	}
	if !isASCIIAlpha(value[0]) && value[0] != '_' {
		return fmt.Errorf("contract state field name %q must start with an ASCII letter or underscore", value)
	}
	for index := 1; index < len(value); index++ {
		if !isASCIIAlphaNumeric(value[index]) && value[index] != '_' {
			return fmt.Errorf("contract state field name %q contains invalid byte 0x%02x", value, value[index])
		}
	}
	return nil
}

func validateTypeID(value string) error {
	if len(value) == 0 || len(value) > maxTypeIDBytes {
		return fmt.Errorf("codec type ID must contain 1..%d ASCII bytes", maxTypeIDBytes)
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if isASCIIAlphaNumeric(character) || character == '.' || character == '-' || character == '_' || character == '/' || character == ':' {
			continue
		}
		return fmt.Errorf("codec type ID %q contains invalid byte 0x%02x", value, character)
	}
	return nil
}

func validateCodec[T any](codec Codec[T]) error {
	if codec == nil {
		return fmt.Errorf("contract state codec is nil")
	}
	value := reflect.ValueOf(codec)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return fmt.Errorf("contract state codec is nil")
		}
	}
	return validateTypeID(codec.TypeID())
}

func appendCanonicalString(destination []byte, value string) []byte {
	return appendCanonicalBytes(destination, []byte(value))
}

func appendCanonicalBytes(destination, value []byte) []byte {
	destination = appendU64(destination, uint64(len(value)))
	return append(destination, value...)
}

func appendU32(destination []byte, value uint32) []byte {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], value)
	return append(destination, encoded[:]...)
}

func appendU64(destination []byte, value uint64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return append(destination, encoded[:]...)
}

func isASCIIAlpha(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isASCIIAlphaNumeric(value byte) bool {
	return isASCIIAlpha(value) || value >= '0' && value <= '9'
}

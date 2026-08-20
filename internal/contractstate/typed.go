package contractstate

import (
	"fmt"
)

// Cell is one typed optional value at a fixed layout key.
type Cell[T any] struct {
	field Field
	codec Codec[T]
}

// Map is one typed key/value namespace. Map keys are encoded canonically and
// hashed beneath the field prefix, so variable-length keys cannot overlap.
type Map[K, V any] struct {
	field      Field
	keyCodec   Codec[K]
	valueCodec Codec[V]
}

func DeclareCell[T any](builder *LayoutBuilder, name string, codec Codec[T]) (Cell[T], error) {
	if err := validateCodec(codec); err != nil {
		return Cell[T]{}, err
	}
	field, err := builder.addField(name, FieldCell, "", codec.TypeID())
	if err != nil {
		return Cell[T]{}, err
	}
	return Cell[T]{field: field, codec: codec}, nil
}

func DeclareMap[K, V any](builder *LayoutBuilder, name string, keyCodec Codec[K], valueCodec Codec[V]) (Map[K, V], error) {
	if err := validateCodec(keyCodec); err != nil {
		return Map[K, V]{}, fmt.Errorf("map %q key codec: %w", name, err)
	}
	if err := validateCodec(valueCodec); err != nil {
		return Map[K, V]{}, fmt.Errorf("map %q value codec: %w", name, err)
	}
	field, err := builder.addField(name, FieldMap, keyCodec.TypeID(), valueCodec.TypeID())
	if err != nil {
		return Map[K, V]{}, err
	}
	return Map[K, V]{field: field, keyCodec: keyCodec, valueCodec: valueCodec}, nil
}

func (cell Cell[T]) Field() Field     { return cell.field }
func (mapped Map[K, V]) Field() Field { return mapped.field }

func (cell Cell[T]) Get(transaction *Transaction) (T, bool, error) {
	var zero T
	if err := cell.validate(transaction); err != nil {
		return zero, false, err
	}
	encoded, found, err := transaction.readCell(cell.field)
	if err != nil || !found {
		return zero, found, err
	}
	value, err := cell.codec.Decode(encoded)
	if err != nil {
		return zero, false, fmt.Errorf("decode contract state cell %q with %q: %w", cell.field.Name, cell.field.ValueTypeID, err)
	}
	return value, true, nil
}

func (cell Cell[T]) Set(transaction *Transaction, value T) error {
	if err := cell.validate(transaction); err != nil {
		return err
	}
	encoded, err := cell.codec.Encode(value)
	if err != nil {
		return fmt.Errorf("encode contract state cell %q with %q: %w", cell.field.Name, cell.field.ValueTypeID, err)
	}
	return transaction.writeCell(cell.field, encoded)
}

func (cell Cell[T]) Delete(transaction *Transaction) error {
	if err := cell.validate(transaction); err != nil {
		return err
	}
	return transaction.deleteCell(cell.field)
}

func (cell Cell[T]) validate(transaction *Transaction) error {
	if transaction == nil {
		return fmt.Errorf("contract state transaction is nil")
	}
	if err := validateCodec(cell.codec); err != nil {
		return fmt.Errorf("contract state cell %q codec: %w", cell.field.Name, err)
	}
	if cell.field.Kind != FieldCell || cell.field.ValueTypeID != cell.codec.TypeID() {
		return fmt.Errorf("contract state cell handle is inconsistent")
	}
	return nil
}

func (mapped Map[K, V]) Get(transaction *Transaction, key K) (V, bool, error) {
	var zero V
	if err := mapped.validate(transaction); err != nil {
		return zero, false, err
	}
	encodedKey, err := mapped.keyCodec.Encode(key)
	if err != nil {
		return zero, false, fmt.Errorf("encode contract state map %q key with %q: %w", mapped.field.Name, mapped.field.KeyTypeID, err)
	}
	encodedValue, found, err := transaction.readMap(mapped.field, encodedKey)
	if err != nil || !found {
		return zero, found, err
	}
	value, err := mapped.valueCodec.Decode(encodedValue)
	if err != nil {
		return zero, false, fmt.Errorf("decode contract state map %q value with %q: %w", mapped.field.Name, mapped.field.ValueTypeID, err)
	}
	return value, true, nil
}

func (mapped Map[K, V]) Set(transaction *Transaction, key K, value V) error {
	if err := mapped.validate(transaction); err != nil {
		return err
	}
	encodedKey, err := mapped.keyCodec.Encode(key)
	if err != nil {
		return fmt.Errorf("encode contract state map %q key with %q: %w", mapped.field.Name, mapped.field.KeyTypeID, err)
	}
	encodedValue, err := mapped.valueCodec.Encode(value)
	if err != nil {
		return fmt.Errorf("encode contract state map %q value with %q: %w", mapped.field.Name, mapped.field.ValueTypeID, err)
	}
	return transaction.writeMap(mapped.field, encodedKey, encodedValue)
}

func (mapped Map[K, V]) Delete(transaction *Transaction, key K) error {
	if err := mapped.validate(transaction); err != nil {
		return err
	}
	encodedKey, err := mapped.keyCodec.Encode(key)
	if err != nil {
		return fmt.Errorf("encode contract state map %q key with %q: %w", mapped.field.Name, mapped.field.KeyTypeID, err)
	}
	return transaction.deleteMap(mapped.field, encodedKey)
}

func (mapped Map[K, V]) validate(transaction *Transaction) error {
	if transaction == nil {
		return fmt.Errorf("contract state transaction is nil")
	}
	if err := validateCodec(mapped.keyCodec); err != nil {
		return fmt.Errorf("contract state map %q key codec: %w", mapped.field.Name, err)
	}
	if err := validateCodec(mapped.valueCodec); err != nil {
		return fmt.Errorf("contract state map %q value codec: %w", mapped.field.Name, err)
	}
	if mapped.field.Kind != FieldMap || mapped.field.KeyTypeID != mapped.keyCodec.TypeID() || mapped.field.ValueTypeID != mapped.valueCodec.TypeID() {
		return fmt.Errorf("contract state map handle is inconsistent")
	}
	return nil
}

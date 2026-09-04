package contractstate

import "fmt"

// EncodedCell is the compiler-private bridge between canonical MIR bytes and
// one typed state cell. It is intentionally lower level than Cell[T].
type EncodedCell struct {
	field Field
}

// EncodedMap is the compiler-private bridge between canonical MIR bytes and
// one typed state map. It is intentionally lower level than Map[K, V].
type EncodedMap struct {
	field Field
}

// DeclareEncodedCell declares one cell whose canonical value codec is selected
// by stable type identity rather than a Go type parameter.
func DeclareEncodedCell(builder *LayoutBuilder, name, valueTypeID string) (EncodedCell, error) {
	if err := validateCanonicalTypeID(valueTypeID); err != nil {
		return EncodedCell{}, fmt.Errorf("cell %q value codec: %w", name, err)
	}
	field, err := builder.addField(name, FieldCell, "", valueTypeID)
	if err != nil {
		return EncodedCell{}, err
	}
	return EncodedCell{field: field}, nil
}

// DeclareEncodedMap declares one map whose canonical key and value codecs are
// selected by stable type identity rather than Go type parameters.
func DeclareEncodedMap(builder *LayoutBuilder, name, keyTypeID, valueTypeID string) (EncodedMap, error) {
	if err := validateCanonicalTypeID(keyTypeID); err != nil {
		return EncodedMap{}, fmt.Errorf("map %q key codec: %w", name, err)
	}
	if err := validateCanonicalTypeID(valueTypeID); err != nil {
		return EncodedMap{}, fmt.Errorf("map %q value codec: %w", name, err)
	}
	field, err := builder.addField(name, FieldMap, keyTypeID, valueTypeID)
	if err != nil {
		return EncodedMap{}, err
	}
	return EncodedMap{field: field}, nil
}

func (cell EncodedCell) Field() Field  { return cell.field }
func (mapped EncodedMap) Field() Field { return mapped.field }

func (cell EncodedCell) Get(transaction *Transaction) ([]byte, bool, error) {
	if err := cell.validate(transaction); err != nil {
		return nil, false, err
	}
	return transaction.readCell(cell.field)
}

// Borrow returns a transaction-owned encoded view. The view is invalidated by
// any replacement or deletion of this cell and cannot survive transaction close.
func (cell EncodedCell) Borrow(transaction *Transaction) (BorrowedValue, bool, error) {
	if transaction == nil {
		return BorrowedValue{}, false, fmt.Errorf("contract state transaction is nil")
	}
	return transaction.borrowCell(cell.field)
}

func (cell EncodedCell) Set(transaction *Transaction, encoded []byte) error {
	if err := cell.validate(transaction); err != nil {
		return err
	}
	if err := ValidateCanonicalEncoding(cell.field.ValueTypeID, encoded); err != nil {
		return fmt.Errorf("cell %q value: %w", cell.field.Name, err)
	}
	return transaction.writeCell(cell.field, encoded)
}

func (cell EncodedCell) Delete(transaction *Transaction) error {
	if err := cell.validate(transaction); err != nil {
		return err
	}
	return transaction.deleteCell(cell.field)
}

func (cell EncodedCell) validate(transaction *Transaction) error {
	if transaction == nil {
		return fmt.Errorf("contract state transaction is nil")
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	return transaction.validateFieldLocked(cell.field, FieldCell)
}

func (mapped EncodedMap) Get(transaction *Transaction, encodedKey []byte) ([]byte, bool, error) {
	if err := mapped.validate(transaction); err != nil {
		return nil, false, err
	}
	if err := ValidateCanonicalEncoding(mapped.field.KeyTypeID, encodedKey); err != nil {
		return nil, false, fmt.Errorf("map %q key: %w", mapped.field.Name, err)
	}
	return transaction.readMap(mapped.field, encodedKey)
}

// Borrow returns a transaction-owned encoded view for one exact canonical key.
func (mapped EncodedMap) Borrow(transaction *Transaction, encodedKey []byte) (BorrowedValue, bool, error) {
	if transaction == nil {
		return BorrowedValue{}, false, fmt.Errorf("contract state transaction is nil")
	}
	if err := ValidateCanonicalEncoding(mapped.field.KeyTypeID, encodedKey); err != nil {
		return BorrowedValue{}, false, fmt.Errorf("map %q key: %w", mapped.field.Name, err)
	}
	return transaction.borrowMap(mapped.field, encodedKey)
}

func (mapped EncodedMap) Set(transaction *Transaction, encodedKey, encodedValue []byte) error {
	if err := mapped.validate(transaction); err != nil {
		return err
	}
	if err := ValidateCanonicalEncoding(mapped.field.KeyTypeID, encodedKey); err != nil {
		return fmt.Errorf("map %q key: %w", mapped.field.Name, err)
	}
	if err := ValidateCanonicalEncoding(mapped.field.ValueTypeID, encodedValue); err != nil {
		return fmt.Errorf("map %q value: %w", mapped.field.Name, err)
	}
	return transaction.writeMap(mapped.field, encodedKey, encodedValue)
}

func (mapped EncodedMap) Delete(transaction *Transaction, encodedKey []byte) error {
	if err := mapped.validate(transaction); err != nil {
		return err
	}
	if err := ValidateCanonicalEncoding(mapped.field.KeyTypeID, encodedKey); err != nil {
		return fmt.Errorf("map %q key: %w", mapped.field.Name, err)
	}
	return transaction.deleteMap(mapped.field, encodedKey)
}

func (mapped EncodedMap) validate(transaction *Transaction) error {
	if transaction == nil {
		return fmt.Errorf("contract state transaction is nil")
	}
	transaction.mu.Lock()
	defer transaction.mu.Unlock()
	return transaction.validateFieldLocked(mapped.field, FieldMap)
}

// ValidateCanonicalEncoding rejects unknown compiler codecs and noncanonical
// byte representations before they enter the transaction overlay.
func ValidateCanonicalEncoding(typeID string, encoded []byte) error {
	switch typeID {
	case Bool.TypeID():
		_, err := Bool.Decode(encoded)
		return err
	case U32.TypeID():
		_, err := U32.Decode(encoded)
		return err
	case U64.TypeID():
		_, err := U64.Decode(encoded)
		return err
	case Bytes.TypeID():
		_, err := Bytes.Decode(encoded)
		return err
	case Text.TypeID():
		_, err := Text.Decode(encoded)
		return err
	case Byte20.TypeID():
		_, err := Byte20.Decode(encoded)
		return err
	case Byte32.TypeID():
		_, err := Byte32.Decode(encoded)
		return err
	default:
		return fmt.Errorf("codec type ID %q is not a compiler-known canonical codec", typeID)
	}
}

func validateCanonicalTypeID(typeID string) error {
	if err := validateTypeID(typeID); err != nil {
		return err
	}
	switch typeID {
	case Bool.TypeID(), U32.TypeID(), U64.TypeID(), Bytes.TypeID(), Text.TypeID(), Byte20.TypeID(), Byte32.TypeID():
		return nil
	default:
		return fmt.Errorf("codec type ID %q is not a compiler-known canonical codec", typeID)
	}
}

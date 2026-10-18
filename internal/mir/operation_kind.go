package mir

import "fmt"

// OperationKind is a closed compiler-private opcode. Numeric values are not wire identities.
type OperationKind uint8

const (
	OperationInvalid OperationKind = iota
	OperationInitialize
	OperationMove
	OperationDrop
	OperationBurn
	OperationCheckedDivision
	OperationArithmeticCheck
	OperationResultSwitch
	OperationUse
	OperationDropFlagInit
	OperationDropFlagClear
	OperationAssign
	OperationUnary
	OperationBinary
	OperationScalarCast
	OperationSharedReborrow
	OperationArrayConstruct
	OperationArrayBorrowView
	OperationArrayLength
	OperationArrayIndexCheck
	OperationArrayIndex
	OperationArrayIndexStore
	OperationResultConstruct
	OperationResultPayload
	OperationOptionConstruct
	OperationOptionPayload
	OperationEnumConstruct
	OperationEnumPayload
	OperationRawAddress
	OperationRawBorrowPointer
	OperationStrFromRawParts
	OperationCStrFromRawParts
	OperationByteSliceFromRawParts
	OperationMutByteSliceFromRawParts
	OperationCStrDataAddress
	OperationStructConstruct
	OperationResourceConstruct
	OperationFieldRead
	OperationFieldBorrow
	OperationFieldStore
	OperationDerefStore
	OperationViewLength
	OperationViewEmpty
	OperationViewIndexCheck
	OperationViewIndex
	OperationViewElementBorrow
	OperationViewIndexStore
	OperationViewSliceCheck
	OperationViewSlice
	OperationRegionEnter
	OperationRegionPlace
	OperationRegionReset
	OperationByteSliceDataAddress
	OperationStrDataAddress
	OperationStorageBorrow
	OperationStorageBorrowMut
	OperationStorageElementAddress
	OperationStorageLayout
	OperationStoragePut
	OperationStorageTake
	operationKindCount
)

var operationNames = [operationKindCount]string{
	OperationInvalid:                  "",
	OperationInitialize:               "initialize",
	OperationMove:                     "move",
	OperationDrop:                     "drop",
	OperationBurn:                     "burn",
	OperationCheckedDivision:          "checked-division",
	OperationArithmeticCheck:          "arithmetic-check",
	OperationResultSwitch:             "result-switch",
	OperationUse:                      "use",
	OperationDropFlagInit:             "drop-flag-init",
	OperationDropFlagClear:            "drop-flag-clear",
	OperationAssign:                   "assign",
	OperationUnary:                    "unary",
	OperationBinary:                   "binary",
	OperationScalarCast:               "scalar-cast",
	OperationSharedReborrow:           "shared-reborrow",
	OperationArrayConstruct:           "array-construct",
	OperationArrayBorrowView:          "array-borrow-view",
	OperationArrayLength:              "array-length",
	OperationArrayIndexCheck:          "array-index-check",
	OperationArrayIndex:               "array-index",
	OperationArrayIndexStore:          "array-index-store",
	OperationResultConstruct:          "result-construct",
	OperationResultPayload:            "result-payload",
	OperationOptionConstruct:          "option-construct",
	OperationOptionPayload:            "option-payload",
	OperationEnumConstruct:            "enum-construct",
	OperationEnumPayload:              "enum-payload",
	OperationRawAddress:               "raw-address",
	OperationRawBorrowPointer:         "raw-borrow-pointer",
	OperationStrFromRawParts:          "str-from-raw-parts",
	OperationCStrFromRawParts:         "cstr-from-raw-parts",
	OperationByteSliceFromRawParts:    "byte-slice-from-raw-parts",
	OperationMutByteSliceFromRawParts: "mut-byte-slice-from-raw-parts",
	OperationCStrDataAddress:          "cstr-data-address",
	OperationStructConstruct:          "struct-construct",
	OperationResourceConstruct:        "resource-construct",
	OperationFieldRead:                "field-read",
	OperationFieldBorrow:              "field-borrow",
	OperationFieldStore:               "field-store",
	OperationDerefStore:               "deref-store",
	OperationViewLength:               "view-length",
	OperationViewEmpty:                "view-empty",
	OperationViewIndexCheck:           "view-index-check",
	OperationViewIndex:                "view-index",
	OperationViewElementBorrow:        "view-element-borrow",
	OperationViewIndexStore:           "view-index-store",
	OperationViewSliceCheck:           "view-slice-check",
	OperationViewSlice:                "view-slice",
	OperationRegionEnter:              "region-enter",
	OperationRegionPlace:              "region-place",
	OperationRegionReset:              "region-reset",
	OperationByteSliceDataAddress:     "byte-slice-data-address",
	OperationStrDataAddress:           "str-data-address",
	OperationStorageBorrow:            "storage-borrow",
	OperationStorageBorrowMut:         "storage-borrow-mut",
	OperationStorageElementAddress:    "storage-element-address",
	OperationStorageLayout:            "storage-layout",
	OperationStoragePut:               "storage-put",
	OperationStorageTake:              "storage-take",
}

var operationKindsByName = func() map[string]OperationKind {
	result := make(map[string]OperationKind, len(operationNames))
	for kind, name := range operationNames {
		result[name] = OperationKind(kind)
	}
	return result
}()

// Valid reports whether kind denotes a declared executable operation.
func (kind OperationKind) Valid() bool { return kind > OperationInvalid && kind < operationKindCount }

func (kind OperationKind) String() string {
	if kind < operationKindCount {
		return operationNames[kind]
	}
	return fmt.Sprintf("OperationKind(%d)", uint8(kind))
}

// AllOperationKinds returns a detached inventory, excluding the invalid sentinel.
func AllOperationKinds() []OperationKind {
	result := make([]OperationKind, 0, int(operationKindCount)-1)
	for kind := OperationInvalid + 1; kind < operationKindCount; kind++ {
		result = append(result, kind)
	}
	return result
}

// ParseOperationKind accepts only exact existing wire names (including the empty invalid sentinel).
func ParseOperationKind(name string) (OperationKind, error) {
	if kind, found := operationKindsByName[name]; found {
		return kind, nil
	}
	return OperationInvalid, fmt.Errorf("unknown MIR operation %q", name)
}

// MarshalText preserves the existing JSON and cache string format.
func (kind OperationKind) MarshalText() ([]byte, error) {
	if kind >= operationKindCount {
		return nil, fmt.Errorf("unknown MIR operation kind %d", uint8(kind))
	}
	return []byte(kind.String()), nil
}

func (kind *OperationKind) UnmarshalText(text []byte) error {
	parsed, err := ParseOperationKind(string(text))
	if err != nil {
		return err
	}
	*kind = parsed
	return nil
}

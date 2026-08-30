// Package contractstatemir defines an experimental compiler-private MIR for
// deterministic persistent-state transactions. It has no source syntax or
// runtime integration; its bounded body payload, checked fresh-owner rebind,
// and cached oracle execution form a compiler-private transport boundary.
package contractstatemir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

// Declaration is one source-ordered compiler-known state field.
type Declaration struct {
	Name        string
	Kind        contractstate.FieldKind
	KeyTypeID   string
	ValueTypeID string
}

// LayoutField preserves the exact canonical field identity consumed by MIR.
type LayoutField struct {
	Index       uint32
	Name        string
	Kind        contractstate.FieldKind
	KeyTypeID   string
	ValueTypeID string
	Prefix      contractstate.Key
}

// Layout is the compiler-private state schema attached to one transaction MIR.
type Layout struct {
	ContractUID string
	Hash        contractstate.LayoutHash
	Fields      []LayoutField
}

// Value is one canonical encoded MIR value with an explicit storage type.
type Value struct {
	TypeID  string
	Encoded []byte
}

type transactionCapabilityToken struct {
	marker byte
}

// TransactionCapability is one compiler-created, non-forgeable transaction
// identity. Copies retain the same authority instead of creating a new token.
type TransactionCapability struct {
	token *transactionCapabilityToken
}

// newTransactionCapability creates one fresh compiler-private transaction
// identity for a complete Program and all of its operations. Construction stays
// package-private so other compiler stages cannot forge verified authority.
func newTransactionCapability() TransactionCapability {
	return TransactionCapability{token: &transactionCapabilityToken{marker: 1}}
}

type OperationKind string

const (
	OperationCellRead   OperationKind = "cell-read"
	OperationCellWrite  OperationKind = "cell-write"
	OperationCellDelete OperationKind = "cell-delete"
	OperationMapRead    OperationKind = "map-read"
	OperationMapWrite   OperationKind = "map-write"
	OperationMapDelete  OperationKind = "map-delete"
)

// Operation is one straight-line storage action. Reads append one observation
// to Outcome.Reads; writes and deletes enter the private transaction overlay.
type Operation struct {
	Kind       OperationKind
	Capability TransactionCapability
	FieldIndex uint32
	Key        Value
	Value      Value
}

type ExitKind string

const (
	ExitCommit ExitKind = "commit"
	ExitReject ExitKind = "reject"
	ExitTrap   ExitKind = "trap"
)

// Program owns one exact layout, one bounded operation sequence, and one
// explicit transaction exit. General Fairy MIR and source grammar are not
// changed by this experimental submodel.
type Program struct {
	Layout       Layout
	Capability   TransactionCapability
	Effects      []hir.Effect
	Operations   []Operation
	Exit         ExitKind
	TrapCategory string
}

// BuildLayout computes the same canonical identity used by the transaction
// oracle while retaining an independently verifiable compiler representation.
func BuildLayout(contractUID string, declarations []Declaration) (Layout, error) {
	builder, err := contractstate.NewLayoutBuilder(contractUID)
	if err != nil {
		return Layout{}, err
	}
	fields := make([]LayoutField, 0, len(declarations))
	for _, declaration := range declarations {
		field, err := declare(builder, declaration)
		if err != nil {
			return Layout{}, err
		}
		fields = append(fields, layoutField(field))
	}
	stateLayout, err := builder.Build()
	if err != nil {
		return Layout{}, err
	}
	return Layout{ContractUID: contractUID, Hash: stateLayout.Hash(), Fields: fields}, nil
}

func declare(builder *contractstate.LayoutBuilder, declaration Declaration) (contractstate.Field, error) {
	switch declaration.Kind {
	case contractstate.FieldCell:
		if declaration.KeyTypeID != "" {
			return contractstate.Field{}, fmt.Errorf("cell %q cannot declare key type %q", declaration.Name, declaration.KeyTypeID)
		}
		cell, err := contractstate.DeclareEncodedCell(builder, declaration.Name, declaration.ValueTypeID)
		if err != nil {
			return contractstate.Field{}, err
		}
		return cell.Field(), nil
	case contractstate.FieldMap:
		mapped, err := contractstate.DeclareEncodedMap(builder, declaration.Name, declaration.KeyTypeID, declaration.ValueTypeID)
		if err != nil {
			return contractstate.Field{}, err
		}
		return mapped.Field(), nil
	default:
		return contractstate.Field{}, fmt.Errorf("field %q has unsupported kind %d", declaration.Name, declaration.Kind)
	}
}

func layoutField(field contractstate.Field) LayoutField {
	return LayoutField{
		Index: field.Index, Name: field.Name, Kind: field.Kind,
		KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID, Prefix: field.Prefix,
	}
}

func declaration(field LayoutField) Declaration {
	return Declaration{
		Name: field.Name, Kind: field.Kind, KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID,
	}
}

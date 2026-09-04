package contractstatemir

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

// VerificationError is one stable compiler-private transaction-MIR failure.
type VerificationError struct {
	Code   string
	Entity string
	Detail string
}

func (failure VerificationError) Error() string {
	return fmt.Sprintf("%s %s: %s", failure.Code, failure.Entity, failure.Detail)
}

type boundField struct {
	descriptor contractstate.Field
	cell       contractstate.EncodedCell
	mapped     contractstate.EncodedMap
}

type boundLayout struct {
	layout contractstate.Layout
	fields []boundField
}

// Verify rejects layout drift, malformed operations, noncanonical values, and
// implicit transaction exits before the oracle captures a snapshot.
func Verify(program Program) []VerificationError {
	bound, err := bindLayout(program.Layout)
	if err != nil {
		return []VerificationError{{Code: "CSM1001", Entity: "layout", Detail: err.Error()}}
	}
	failures := make([]VerificationError, 0)
	if program.Capability.token == nil {
		failures = append(failures, VerificationError{Code: "CSM1004", Entity: "capability", Detail: "program requires one transaction capability"})
	}
	requiredEffects := make([]hir.Effect, 0, 2)
	if len(program.Blocks) != 0 {
		if len(program.Operations) != 0 || program.Exit != "" || program.TrapCategory != "" {
			failures = append(failures, VerificationError{Code: "CSM1006", Entity: "program", Detail: "graph and legacy linear bodies cannot be mixed"})
		}
		graphFailures, graphEffects := verifyGraph(program, bound)
		failures = append(failures, graphFailures...)
		requiredEffects = append(requiredEffects, graphEffects...)
	} else {
		if program.Entry != 0 {
			failures = append(failures, VerificationError{Code: "CSM1006", Entity: "entry", Detail: "linear program cannot declare a graph entry"})
		}
		for index, operation := range program.Operations {
			entity := fmt.Sprintf("operation %d", index)
			if operation.KeyInput != 0 || operation.ValueInput != 0 || operation.Result != 0 || operation.Found != 0 || operation.Cost != 0 {
				failures = append(failures, VerificationError{Code: "CSM1002", Entity: entity, Detail: "legacy operation carries graph-only value or cost fields"})
			}
			if operation.Capability.token == nil || operation.Capability != program.Capability {
				failures = append(failures, VerificationError{Code: "CSM1004", Entity: entity, Detail: "operation does not use the program transaction capability"})
			}
			requiredEffects = append(requiredEffects, requiredStorageEffects(operation.Kind)...)
			if int(operation.FieldIndex) >= len(bound.fields) {
				failures = append(failures, VerificationError{Code: "CSM1002", Entity: entity, Detail: "field index is outside the verified layout"})
				continue
			}
			field := bound.fields[operation.FieldIndex].descriptor
			if detail := verifyOperation(operation, field); detail != "" {
				failures = append(failures, VerificationError{Code: "CSM1002", Entity: entity, Detail: detail})
			}
		}
		if detail := verifyExit(program.Exit, program.TrapCategory); detail != "" {
			failures = append(failures, VerificationError{Code: "CSM1003", Entity: "exit", Detail: detail})
		}
	}
	if detail := verifyEffects(program.Effects, requiredEffects); detail != "" {
		failures = append(failures, VerificationError{Code: "CSM1005", Entity: "effects", Detail: detail})
	}
	return failures
}

func requiredStorageEffects(kind OperationKind) []hir.Effect {
	switch kind {
	case OperationCellRead, OperationMapRead:
		return []hir.Effect{hir.EffectStorageRead}
	case OperationCellWrite, OperationCellDelete, OperationMapWrite, OperationMapDelete:
		return []hir.Effect{hir.EffectStorageWrite}
	default:
		return nil
	}
}

func verifyEffects(declared, required []hir.Effect) string {
	canonical := hir.CanonicalEffects(declared)
	if !slices.Equal(declared, canonical) {
		return "declared effects are not sorted and unique"
	}
	for _, effect := range declared {
		if !hir.IsOnchainEffect(effect) {
			return fmt.Sprintf("declared effect %q is not an on-chain capability", effect)
		}
		if effect != hir.EffectStorageRead && effect != hir.EffectStorageWrite {
			return fmt.Sprintf("declared effect %q has no contract-state transaction MIR operation", effect)
		}
	}
	if missing := hir.MissingEffects(declared, required); len(missing) != 0 {
		return fmt.Sprintf("missing required effects %v", missing)
	}
	return ""
}

func bindLayout(layout Layout) (boundLayout, error) {
	builder, err := contractstate.NewLayoutBuilder(layout.ContractUID)
	if err != nil {
		return boundLayout{}, err
	}
	bound := boundLayout{fields: make([]boundField, 0, len(layout.Fields))}
	for index, expected := range layout.Fields {
		if expected.Index != uint32(index) {
			return boundLayout{}, fmt.Errorf("field %q has index %d, expected %d", expected.Name, expected.Index, index)
		}
		declaration := declaration(expected)
		switch declaration.Kind {
		case contractstate.FieldCell:
			cell, err := contractstate.DeclareEncodedCell(builder, declaration.Name, declaration.ValueTypeID)
			if err != nil {
				return boundLayout{}, err
			}
			if actual := layoutField(cell.Field()); actual != expected {
				return boundLayout{}, fmt.Errorf("field %q identity is noncanonical", expected.Name)
			}
			bound.fields = append(bound.fields, boundField{descriptor: cell.Field(), cell: cell})
		case contractstate.FieldMap:
			mapped, err := contractstate.DeclareEncodedMap(builder, declaration.Name, declaration.KeyTypeID, declaration.ValueTypeID)
			if err != nil {
				return boundLayout{}, err
			}
			if actual := layoutField(mapped.Field()); actual != expected {
				return boundLayout{}, fmt.Errorf("field %q identity is noncanonical", expected.Name)
			}
			bound.fields = append(bound.fields, boundField{descriptor: mapped.Field(), mapped: mapped})
		default:
			return boundLayout{}, fmt.Errorf("field %q has unsupported kind %d", expected.Name, expected.Kind)
		}
	}
	bound.layout, err = builder.Build()
	if err != nil {
		return boundLayout{}, err
	}
	if bound.layout.Hash() != layout.Hash {
		return boundLayout{}, fmt.Errorf("layout hash is noncanonical")
	}
	return bound, nil
}

func verifyOperation(operation Operation, field contractstate.Field) string {
	switch operation.Kind {
	case OperationCellRead:
		if field.Kind != contractstate.FieldCell {
			return "cell read names a non-cell field"
		}
		return verifyAbsentValues(operation)
	case OperationCellWrite:
		if field.Kind != contractstate.FieldCell {
			return "cell write names a non-cell field"
		}
		if operation.Key.TypeID != "" || operation.Key.Encoded != nil {
			return "cell write carries a map key"
		}
		return verifyValue("value", operation.Value, field.ValueTypeID)
	case OperationCellDelete:
		if field.Kind != contractstate.FieldCell {
			return "cell delete names a non-cell field"
		}
		return verifyAbsentValues(operation)
	case OperationMapRead:
		if field.Kind != contractstate.FieldMap {
			return "map read names a non-map field"
		}
		if detail := verifyValue("key", operation.Key, field.KeyTypeID); detail != "" {
			return detail
		}
		if operation.Value.TypeID != "" || operation.Value.Encoded != nil {
			return "map read carries a value"
		}
		return ""
	case OperationMapWrite:
		if field.Kind != contractstate.FieldMap {
			return "map write names a non-map field"
		}
		if detail := verifyValue("key", operation.Key, field.KeyTypeID); detail != "" {
			return detail
		}
		return verifyValue("value", operation.Value, field.ValueTypeID)
	case OperationMapDelete:
		if field.Kind != contractstate.FieldMap {
			return "map delete names a non-map field"
		}
		if detail := verifyValue("key", operation.Key, field.KeyTypeID); detail != "" {
			return detail
		}
		if operation.Value.TypeID != "" || operation.Value.Encoded != nil {
			return "map delete carries a value"
		}
		return ""
	default:
		return fmt.Sprintf("operation kind %q is unsupported", operation.Kind)
	}
}

func verifyAbsentValues(operation Operation) string {
	if operation.Key.TypeID != "" || operation.Key.Encoded != nil {
		return "cell operation carries a map key"
	}
	if operation.Value.TypeID != "" || operation.Value.Encoded != nil {
		return "non-write operation carries a value"
	}
	return ""
}

func verifyValue(role string, value Value, expectedTypeID string) string {
	if value.TypeID != expectedTypeID {
		return fmt.Sprintf("%s type %q does not match field type %q", role, value.TypeID, expectedTypeID)
	}
	if err := contractstate.ValidateCanonicalEncoding(value.TypeID, value.Encoded); err != nil {
		return fmt.Sprintf("%s encoding is noncanonical: %v", role, err)
	}
	return ""
}

func verifyExit(exit ExitKind, trapCategory string) string {
	switch exit {
	case ExitCommit, ExitReject:
		if trapCategory != "" {
			return "non-trap exit carries a trap category"
		}
		return ""
	case ExitTrap:
		if !validTrapCategory(trapCategory) {
			return "trap exit requires a 1..127 byte ASCII category"
		}
		return ""
	default:
		return fmt.Sprintf("exit kind %q is unsupported", exit)
	}
}

func validTrapCategory(value string) bool {
	if len(value) == 0 || len(value) > 127 || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-' {
			continue
		}
		return false
	}
	return true
}

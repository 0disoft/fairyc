package compiler

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

var stateSourceBoolType = contractstate.Bool.TypeID()

func stateGraphCall(expression *ast.Expression) (string, []*ast.Expression, bool) {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 ||
		expression.CallLabels != nil || len(expression.TypeArguments) != 0 || expression.Children[0] == nil || expression.Children[0].Kind != syntax.KindPath {
		return "", nil, false
	}
	return expression.Children[0].Text, expression.Children[1:], true
}

func stateGraphLookup(name string, environment []stateGraphBinding) (stateGraphBinding, bool) {
	for _, binding := range environment {
		if binding.name == name {
			return binding, true
		}
	}
	return stateGraphBinding{}, false
}

func (lower *stateGraphSource) reference(expression *ast.Expression, environment []stateGraphBinding) (hir.StateValueID, string, error) {
	expression = stateGraphUngroup(expression)
	if expression != nil && expression.Kind == syntax.KindPath {
		if binding, found := stateGraphLookup(expression.Text, environment); found {
			return binding.value, binding.typeID, nil
		}
	}
	name, args, ok := stateGraphCall(expression)
	if ok && name == "storage::found" && len(args) == 1 && args[0] != nil && args[0].Kind == syntax.KindPath {
		if binding, found := stateGraphLookup(args[0].Text, environment); found && binding.found != 0 {
			return binding.found, stateSourceBoolType, nil
		}
	}
	return 0, "", fmt.Errorf("experimental state graph requires an available read value or storage::found(read)")
}

func stateGraphUngroup(expression *ast.Expression) *ast.Expression {
	for expression != nil && expression.Kind == syntax.KindGroup && len(expression.Children) == 1 {
		expression = expression.Children[0]
	}
	return expression
}

func (lower *stateGraphSource) binding(statement ast.Statement, current int, environment []stateGraphBinding) ([]stateGraphBinding, error) {
	binding := statement.Binding
	if binding == nil || binding.Name == "" || binding.Mutable {
		return nil, lower.fail(statement.Span, "storage locals must be immutable named bindings")
	}
	if _, exists := stateGraphLookup(binding.Name, environment); exists {
		return nil, lower.fail(statement.Span, "duplicate or shadowed storage local")
	}
	if _, exists := lower.callees[binding.Name]; exists {
		return nil, lower.fail(statement.Span, "storage local cannot shadow a transaction function")
	}
	value := stateGraphBinding{name: binding.Name}
	name, _, ok := stateGraphCall(binding.Expression)
	if ok && name == "storage::read" {
		access, err := lower.access(binding.Expression, environment)
		if err != nil {
			return nil, err
		}
		if err := lower.appendAccess(current, access); err != nil {
			return nil, err
		}
		value.value, value.found, value.typeID = access.Result, access.FoundResult, lower.fields[access.Field].ValueTypeID
	} else {
		var err error
		value.value, value.typeID, err = lower.reference(binding.Expression, environment)
		if err != nil {
			return nil, err
		}
		// Aliases of a read retain its presence result. A found-result alias
		// remains a plain bool, not a second read with invented presence.
		if originalExpression := stateGraphUngroup(binding.Expression); originalExpression.Kind == syntax.KindPath {
			original, _ := stateGraphLookup(originalExpression.Text, environment)
			value.found = original.found
		}
	}
	if binding.Type != nil && stateGraphSourceType(*binding.Type) != value.typeID {
		return nil, lower.fail(statement.Span, "storage binding annotation differs from the read codec")
	}
	return append(environment, value), nil
}

func (lower *stateGraphSource) statement(statement ast.Statement, current int, environment []stateGraphBinding) (int, error) {
	name, args, ok := stateGraphCall(statement.Expression)
	if !ok {
		return -1, lower.fail(statement.Span, "statement requires an explicit storage operation")
	}
	term := hir.StateTerminator{Cost: 1}
	switch name {
	case "storage::commit", "storage::reject":
		if len(args) != 0 {
			return -1, lower.fail(statement.Span, "commit/reject takes no arguments")
		}
		if name == "storage::commit" {
			if lower.helper {
				return -1, lower.fail(statement.Span, "helper cannot commit the root transaction")
			}
			term.Kind = hir.StateTerminatorCommit
		} else {
			term.Kind = hir.StateTerminatorReject
		}
	case "storage::trap":
		if len(args) != 1 {
			return -1, lower.fail(statement.Span, "trap requires one literal category")
		}
		category, ok := stateGraphString(args[0])
		if !ok {
			return -1, lower.fail(statement.Span, "trap requires one literal category")
		}
		term.Kind, term.TrapCategory = hir.StateTerminatorTrap, category
	default:
		if callee, found := lower.callees[name]; found {
			if name == "transaction" || len(args) != 0 {
				return -1, lower.fail(statement.Span, "helper calls require no arguments and cannot call the root")
			}
			return current, lower.appendAccess(current, hir.StateAccess{Kind: hir.StateAccessCall, Callee: callee, Cost: 1, Span: lower.span(statement.Span)})
		}
		access, err := lower.access(statement.Expression, environment)
		if err != nil {
			return -1, err
		}
		return current, lower.appendAccess(current, access)
	}
	lower.blocks[current].Terminator = term
	return -1, nil
}

func (lower *stateGraphSource) appendAccess(current int, access hir.StateAccess) error {
	if *lower.operations >= stateSourceMaxOperations {
		return fmt.Errorf("experimental state graph exceeds 1024 operations")
	}
	(*lower.operations)++
	lower.blocks[current].Accesses = append(lower.blocks[current].Accesses, access)
	return nil
}

func (lower *stateGraphSource) access(expression *ast.Expression, environment []stateGraphBinding) (hir.StateAccess, error) {
	name, args, ok := stateGraphCall(expression)
	if !ok || (name != "storage::read" && name != "storage::write" && name != "storage::delete") || len(args) == 0 {
		return hir.StateAccess{}, fmt.Errorf("experimental state graph requires read/write/delete")
	}
	fieldName, ok := stateGraphString(args[0])
	field, found := lower.fields[fieldName]
	if !ok || !found {
		return hir.StateAccess{}, lower.fail(expression.Span, "storage operation must name a declared field literal")
	}
	count := 1
	if field.Kind == hir.StateFieldMap {
		count++
	}
	if name == "storage::write" {
		count++
	}
	if len(args) != count {
		return hir.StateAccess{}, lower.fail(expression.Span, "storage operand count differs from the declared field kind")
	}
	access := hir.StateAccess{Field: fieldName, Cost: 1, Span: lower.span(expression.Span)}
	var err error
	if field.Kind == hir.StateFieldMap {
		access.Key, access.KeyInput, err = lower.operand(args[1], field.KeyTypeID, environment)
		if err != nil {
			return hir.StateAccess{}, err
		}
	}
	switch name {
	case "storage::read":
		access.Kind = hir.StateAccessCellRead
		if field.Kind == hir.StateFieldMap {
			access.Kind = hir.StateAccessMapRead
		}
		access.Result, access.FoundResult = lower.value(), lower.value()
	case "storage::write":
		access.Kind = hir.StateAccessCellWrite
		if field.Kind == hir.StateFieldMap {
			access.Kind = hir.StateAccessMapWrite
		}
		access.Value, access.ValueInput, err = lower.operand(args[len(args)-1], field.ValueTypeID, environment)
	case "storage::delete":
		access.Kind = hir.StateAccessCellDelete
		if field.Kind == hir.StateFieldMap {
			access.Kind = hir.StateAccessMapDelete
		}
	}
	return access, err
}

func (lower *stateGraphSource) operand(expression *ast.Expression, typeID string, environment []stateGraphBinding) (hir.StateValue, hir.StateValueID, error) {
	if expression == nil || expression.Kind != syntax.KindLiteral {
		id, actual, err := lower.reference(expression, environment)
		if err != nil || actual != typeID {
			return hir.StateValue{}, 0, fmt.Errorf("experimental state graph operand requires an available value of codec %q", typeID)
		}
		return hir.StateValue{}, id, nil
	}
	value := hir.StateValue{TypeID: typeID}
	var err error
	switch typeID {
	case contractstate.Bool.TypeID():
		if expression.Text != "true" && expression.Text != "false" {
			err = fmt.Errorf("bool requires true or false")
		} else {
			value.Encoded, err = contractstate.Bool.Encode(expression.Text == "true")
		}
	case contractstate.U32.TypeID(), contractstate.U64.TypeID():
		bits := 64
		if typeID == contractstate.U32.TypeID() {
			bits = 32
		}
		var number uint64
		number, err = strconv.ParseUint(expression.Text, 10, bits)
		if err == nil && bits == 32 {
			value.Encoded, err = contractstate.U32.Encode(uint32(number))
		} else if err == nil {
			value.Encoded, err = contractstate.U64.Encode(number)
		}
	case contractstate.Byte20.TypeID(), contractstate.Byte32.TypeID():
		if !strings.HasPrefix(expression.Text, "b\"") {
			err = fmt.Errorf("fixed bytes require a byte-string literal")
		} else {
			value.Encoded, err = hex.DecodeString(expression.Cooked)
			if err == nil && typeID == contractstate.Byte20.TypeID() {
				_, err = contractstate.Byte20.Decode(value.Encoded)
			} else if err == nil {
				_, err = contractstate.Byte32.Decode(value.Encoded)
			}
		}
	default:
		err = fmt.Errorf("codec is not admitted by source state declarations")
	}
	if err != nil {
		return hir.StateValue{}, 0, lower.fail(expression.Span, fmt.Sprintf("invalid %s literal: %v", typeID, err))
	}
	return value, 0, nil
}

func stateGraphString(expression *ast.Expression) (string, bool) {
	if expression == nil || expression.Kind != syntax.KindLiteral || !strings.HasPrefix(expression.Text, "\"") {
		return "", false
	}
	decoded, err := hex.DecodeString(expression.Cooked)
	return string(decoded), err == nil
}

func stateGraphSourceType(reference ast.TypeReference) string {
	if reference.Kind != ast.TypeKindNamed || len(reference.Arguments) != 0 {
		return ""
	}
	switch reference.Path {
	case "bool":
		return stateSourceBoolType
	case "u32":
		return contractstate.U32.TypeID()
	case "u64":
		return contractstate.U64.TypeID()
	case "bytes20":
		return contractstate.Byte20.TypeID()
	case "bytes32":
		return contractstate.Byte32.TypeID()
	default:
		return ""
	}
}

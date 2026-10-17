package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/mir"
)

func supportedUnary(operator string) bool {
	return operator == "+" || operator == "-" || operator == "!" || operator == "~" ||
		operator == "&" || operator == "&mut" || operator == "*"
}

func supportedBinary(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", "<<", ">>", "&", "^", "|", "==", "!=", "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
}

func oneOperand(operands []mir.Operand) (mir.Operand, error) {
	if len(operands) != 1 || operands[0].Kind == mir.OperandInvalid {
		return mir.Operand{}, fmt.Errorf("expected one scalar operand")
	}
	return operands[0], nil
}

func twoOperands(operands []mir.Operand) (mir.Operand, mir.Operand, error) {
	if len(operands) != 2 || operands[0].Kind == mir.OperandInvalid || operands[1].Kind == mir.OperandInvalid {
		return mir.Operand{}, mir.Operand{}, fmt.Errorf("expected two scalar operands")
	}
	return operands[0], operands[1], nil
}

func threeOperands(operands []mir.Operand) (mir.Operand, mir.Operand, mir.Operand, error) {
	if len(operands) != 3 || operands[0].Kind == mir.OperandInvalid || operands[1].Kind == mir.OperandInvalid || operands[2].Kind == mir.OperandInvalid {
		return mir.Operand{}, mir.Operand{}, mir.Operand{}, fmt.Errorf("expected three scalar operands")
	}
	return operands[0], operands[1], operands[2], nil
}

func oneSuccessor(terminator mir.Terminator) (mir.BlockID, error) {
	if len(terminator.Successors) != 1 {
		return mir.NoBlock, fmt.Errorf("expected one successor, got %d", len(terminator.Successors))
	}
	return terminator.Successors[0], nil
}

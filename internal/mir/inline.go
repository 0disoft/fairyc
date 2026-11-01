package mir

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyInlineFunction(fn *Function) []VerificationError {
	if fn.InlineBody == "" {
		return nil
	}
	reject := func(detail string) []VerificationError {
		return []VerificationError{{Code: CodeFunctionIdentity, Entity: fn.Name, Detail: "inline body: " + detail}}
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(fn.InlineBody, hir.InlineFunctionPrefix))
	if err != nil || len(raw) != 32 || !strings.HasPrefix(fn.InlineBody, hir.InlineFunctionPrefix) || fn.InlineBody != strings.ToLower(fn.InlineBody) {
		return reject("invalid versioned identity")
	}
	if fn.ResultType != hir.TypeI32 || len(fn.ParameterTypes) > 4 || len(fn.Resources) != 0 || len(fn.Effects) != 0 || len(fn.UnsafeOperations) != 0 || fn.DropGlueFor != (hir.Type{}) || len(fn.Blocks) > 256 {
		return reject("unsupported function signature or state")
	}
	for _, typ := range fn.ParameterTypes {
		if typ != hir.TypeI32 {
			return reject("non-i32 parameter")
		}
	}
	operations := 0
	for _, block := range fn.Blocks {
		for _, op := range block.Operations {
			operations++
			switch op.Kind {
			case OperationAssign, OperationUnary, OperationBinary, OperationArithmeticCheck, OperationCheckedDivision:
			default:
				return reject(fmt.Sprintf("unsupported operation %s", op.Kind))
			}
		}
		switch block.Terminator.Kind {
		case TerminatorReturn, TerminatorGoto, TerminatorSwitch, TerminatorTrap:
		default:
			return reject("calls and resource/control-flow extensions are closed")
		}
	}
	if operations > 128 {
		return reject("body exceeds operation budget")
	}
	state := map[BlockID]uint8{}
	var visit func(BlockID) bool
	visit = func(id BlockID) bool {
		if state[id] == 1 {
			return false
		}
		if state[id] == 2 {
			return true
		}
		if id < 0 || int(id) >= len(fn.Blocks) {
			return false
		}
		state[id] = 1
		for _, child := range fn.Blocks[id].Terminator.Successors {
			if !visit(child) {
				return false
			}
		}
		state[id] = 2
		return true
	}
	for _, b := range fn.Blocks {
		if !visit(b.ID) {
			return reject("cyclic or invalid control flow")
		}
	}
	return nil
}

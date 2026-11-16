package mir

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
)

const (
	inlineIdentityHexBytes = 2 * sha256.Size
	inlineMaxParameters    = 4
	inlineMaxBlocks        = 256
	inlineMaxOperations    = 128
	inlineMaxCallFunctions = 64
	inlineVisitActive      = 1
	inlineVisitComplete    = 2
)

func verifyInlineFunction(fn *Function, functions map[hir.SymbolID]*Function) []VerificationError {
	if fn.InlineBody == "" {
		return nil
	}
	reject := func(detail string) []VerificationError {
		return []VerificationError{{Code: CodeFunctionIdentity, Entity: fn.Name, Detail: "inline body: " + detail}}
	}
	if !validInlineIdentity(fn.InlineBody) {
		return reject("invalid versioned identity")
	}
	if detail := inlineSignatureFailure(fn); detail != "" {
		return reject(detail)
	}
	if detail := inlineBodyFailure(fn.Blocks); detail != "" {
		return reject(detail)
	}
	if !inlineControlFlowAcyclic(fn.Blocks) {
		return reject("cyclic or invalid control flow")
	}
	// Whole-program call authority is required even for restored artifacts.
	if !inlineCallGraphAcyclic(fn, functions) {
		return reject("calls require an acyclic graph of at most 64 same-module pure inline functions")
	}
	return nil
}

func validInlineIdentity(identity string) bool {
	// Check the fixed wire shape before inspecting bytes; never decode an
	// untrusted, potentially oversized marker just to validate its spelling.
	if len(identity) != len(hir.InlineFunctionPrefix)+inlineIdentityHexBytes || !strings.HasPrefix(identity, hir.InlineFunctionPrefix) {
		return false
	}
	for index := len(hir.InlineFunctionPrefix); index < len(identity); index++ {
		value := identity[index]
		if !(value >= '0' && value <= '9' || value >= 'a' && value <= 'f') {
			return false
		}
	}
	return true
}

func inlineSignatureFailure(fn *Function) string {
	if fn.ResultType != hir.TypeI32 || len(fn.ParameterTypes) > inlineMaxParameters || len(fn.Resources) != 0 || len(fn.Effects) != 0 || len(fn.UnsafeOperations) != 0 || fn.DropGlueFor != (hir.Type{}) || len(fn.Blocks) > inlineMaxBlocks {
		return "unsupported function signature or state"
	}
	for _, typ := range fn.ParameterTypes {
		if typ != hir.TypeI32 {
			return "non-i32 parameter"
		}
	}
	return ""
}

func inlineBodyFailure(blocks []Block) string {
	operations := 0
	for _, block := range blocks {
		for _, op := range block.Operations {
			operations++
			switch op.Kind {
			case OperationAssign, OperationUnary, OperationBinary, OperationArithmeticCheck, OperationCheckedDivision:
			default:
				return fmt.Sprintf("unsupported operation %s", op.Kind)
			}
		}
		switch block.Terminator.Kind {
		case TerminatorReturn, TerminatorGoto, TerminatorSwitch, TerminatorTrap, TerminatorCall:
		default:
			return "calls and resource/control-flow extensions are closed"
		}
	}
	if operations > inlineMaxOperations {
		return "body exceeds operation budget"
	}
	return ""
}

func inlineControlFlowAcyclic(blocks []Block) bool {
	state := map[BlockID]uint8{}
	var visit func(BlockID) bool
	visit = func(id BlockID) bool {
		if state[id] == inlineVisitActive {
			return false
		}
		if state[id] == inlineVisitComplete {
			return true
		}
		if id < 0 || int(id) >= len(blocks) {
			return false
		}
		state[id] = inlineVisitActive
		for _, child := range blocks[id].Terminator.Successors {
			if !visit(child) {
				return false
			}
		}
		state[id] = inlineVisitComplete
		return true
	}
	for _, b := range blocks {
		if !visit(b.ID) {
			return false
		}
	}
	return true
}

func inlineCallGraphAcyclic(fn *Function, functions map[hir.SymbolID]*Function) bool {
	visited := make(map[hir.SymbolID]uint8)
	var calls func(*Function) bool
	calls = func(current *Function) bool {
		if visited[current.Symbol] == inlineVisitActive {
			return false
		}
		if visited[current.Symbol] == inlineVisitComplete {
			return true
		}
		if len(visited) >= inlineMaxCallFunctions {
			return false
		}
		visited[current.Symbol] = inlineVisitActive
		for _, block := range current.Blocks {
			term := block.Terminator
			if term.Kind != TerminatorCall {
				continue
			}
			target := functions[term.Callee]
			if !validInlineCallTarget(current, target, term) || !calls(target) {
				return false
			}
		}
		visited[current.Symbol] = inlineVisitComplete
		return true
	}
	return calls(fn)
}

func validInlineCallTarget(current, target *Function, term Terminator) bool {
	if target == nil || target.InlineBody == "" {
		return false
	}
	module := inlineModule(current.Name)
	return module != "" && inlineModule(target.Name) == module && len(target.Effects) == 0 &&
		len(target.UnsafeOperations) == 0 && len(target.Resources) == 0 && target.ResultType == hir.TypeI32 &&
		target.DropGlueFor == (hir.Type{}) && term.ForeignABI == "" && term.UnsafeScope == (hir.Span{}) &&
		len(term.Arguments) == len(target.ParameterTypes)
}

func inlineModule(name string) string {
	i := strings.LastIndex(name, "::")
	if i <= 0 {
		return ""
	}
	return name[:i]
}

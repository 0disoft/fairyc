package packageartifact

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// ClosureSourceProof compares checked source and restored bodies within the
// current package namespace. Local/block/function numbers are not authority.
// It does not assert source provenance by itself; callers supply checked HIR.
func ClosureSourceProof(function mir.Function, symbols map[hir.SymbolID]string) ([32]byte, error) {
	canonical, _, err := canonicalizeFunctionGraphIDs(function)
	if err != nil {
		return [32]byte{}, err
	}
	canonical.Symbol = hir.NoSymbol
	var callees []string
	for bi := range canonical.Blocks {
		block := &canonical.Blocks[bi]
		for oi := range block.Operations {
			op := &block.Operations[oi]
			if op.Kind != mir.OperationClosureCreate && op.Kind != mir.OperationCCallback {
				continue
			}
			name, found := symbols[op.Symbol]
			if !found {
				return [32]byte{}, fmt.Errorf("closure source proof has an unresolved target")
			}
			op.Symbol, op.Variant = hir.NoSymbol, name
		}
		term := &block.Terminator
		if term.Kind == mir.TerminatorCall {
			name, found := symbols[term.Callee]
			if !found {
				return [32]byte{}, fmt.Errorf("closure source proof has an unresolved callee")
			}
			callees = append(callees, name)
			term.Callee = hir.NoSymbol
		}
	}
	encoded, err := json.Marshal(struct {
		Schema  string
		Body    mir.Function
		Callees []string
	}{"fairyc.closure-source-body-proof/v1", canonical, callees})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(encoded), nil
}

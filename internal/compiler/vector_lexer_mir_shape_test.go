package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/mir"
)

// This projection omits source spans, entities and numeric local IDs. It is
// structural evidence, not an equivalence proof for the two source forms.
type scannerMIRShape struct {
	Name                string         `json:"name"`
	Blocks              int            `json:"blocks"`
	Operations          int            `json:"operations"`
	ReachableBlocks     int            `json:"reachable_blocks"`
	ReachableOperations int            `json:"reachable_operations"`
	ReachableEdges      map[string]int `json:"reachable_edges"`
	OperationKinds      map[string]int `json:"operation_kinds"`
	Terminators         map[string]int `json:"terminators"`
	Calls               map[string]int `json:"calls"`
	DropFlags           int            `json:"drop_flags"`
	Loans               int            `json:"loans"`
	CleanupPaths        map[string]int `json:"cleanup_paths"`
	CleanupActions      map[string]int `json:"cleanup_actions"`
	TrapPaths           int            `json:"trap_paths"`
}

func scannerMIRFunctionShape(program mir.Program, name string) (scannerMIRShape, error) {
	bySymbol := make(map[int]string, len(program.Functions)+len(program.ForeignFunctions))
	for _, function := range program.Functions {
		bySymbol[int(function.Symbol)] = function.Name
	}
	for _, function := range program.ForeignFunctions {
		bySymbol[int(function.Symbol)] = "extern:" + function.LinkName
	}
	var function *mir.Function
	for index := range program.Functions {
		if program.Functions[index].Name == name {
			if function != nil {
				return scannerMIRShape{}, fmt.Errorf("duplicate MIR function %s", name)
			}
			function = &program.Functions[index]
		}
	}
	if function == nil {
		return scannerMIRShape{}, fmt.Errorf("missing MIR function %s", name)
	}
	shape := scannerMIRShape{
		Name: name, Blocks: len(function.Blocks), DropFlags: len(function.DropFlags),
		Loans: len(function.Loans), TrapPaths: len(function.TrapPaths),
		OperationKinds: make(map[string]int), Terminators: make(map[string]int),
		Calls: make(map[string]int), CleanupPaths: make(map[string]int),
		CleanupActions: make(map[string]int), ReachableEdges: make(map[string]int),
	}
	blocks := make(map[mir.BlockID]*mir.Block, len(function.Blocks))
	for index := range function.Blocks {
		block := &function.Blocks[index]
		blocks[block.ID] = block
	}
	seen := make(map[mir.BlockID]bool, len(blocks))
	queue := []mir.BlockID{function.Entry}
	for len(queue) != 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		block := blocks[id]
		if block == nil {
			return scannerMIRShape{}, fmt.Errorf("%s has missing reachable block %d", name, id)
		}
		seen[id] = true
		shape.ReachableBlocks++
		shape.ReachableOperations += len(block.Operations)
		shape.ReachableEdges[string(block.Terminator.Kind)] += len(block.Terminator.Successors)
		queue = append(queue, block.Terminator.Successors...)
	}
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			shape.Operations++
			shape.OperationKinds[string(operation.Kind)]++
		}
		shape.Terminators[string(block.Terminator.Kind)]++
		if block.Terminator.Kind == mir.TerminatorCall {
			callee := bySymbol[int(block.Terminator.Callee)]
			if callee == "" {
				return scannerMIRShape{}, fmt.Errorf("%s call has unknown callee %d", name, block.Terminator.Callee)
			}
			shape.Calls[callee]++
		}
	}
	for _, path := range function.CleanupPaths {
		shape.CleanupPaths[path.ExitKind]++
		for _, action := range path.Actions {
			shape.CleanupActions[action.Kind]++
		}
	}
	return shape, nil
}

func TestPrivateVectorSDKCandidateScannerMIRShape(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_MIR_SHAPE") != "1" {
		t.Skip("opt-in scanner MIR shape attribution")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	rows := make([]map[string]any, 0, 4)
	var shapes [2][2]scannerMIRShape
	for variantIndex, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, _, specializations := scannerBudgetFairyProgram(t, variant)
		if specializations != 21 {
			t.Fatalf("%s specializations = %d", variant.name, specializations)
		}
		for functionIndex, name := range []string{"$p1::scanner::report_work", "$p1::scanner::storage"} {
			shape, err := scannerMIRFunctionShape(program, name)
			if err != nil || shape.Blocks == 0 || shape.Operations == 0 {
				t.Fatalf("%s %s MIR shape: %+v %v", variant.name, name, shape, err)
			}
			shapes[variantIndex][functionIndex] = shape
			rows = append(rows, map[string]any{
				"variant": variant.name, "scanner_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(variant.scanner))),
				"shape": shape,
			})
		}
	}
	for index := range shapes[0] {
		indexed, candidate := shapes[0][index], shapes[1][index]
		if indexed.Terminators[string(mir.TerminatorCall)] != candidate.Terminators[string(mir.TerminatorCall)] ||
			len(indexed.Calls) != len(candidate.Calls) ||
			indexed.OperationKinds[string(mir.OperationDrop)] != candidate.OperationKinds[string(mir.OperationDrop)] ||
			!maps.Equal(indexed.CleanupPaths, candidate.CleanupPaths) ||
			!maps.Equal(indexed.CleanupActions, candidate.CleanupActions) ||
			indexed.TrapPaths != candidate.TrapPaths {
			t.Fatalf("%s changed call-site or cleanup/trap shape: indexed=%+v candidate=%+v", indexed.Name, indexed, candidate)
		}
	}
	report := map[string]any{
		"schema": "fairyc.vector-lexer-mir-shape/v2", "rows": rows,
		"scope": "Verified executable MIR for two scanner functions. Reachability follows terminator successors from each entry; edge totals do not identify targets. Counts omit source spans, entities and numeric local IDs; they do not prove CFG, dataflow or full semantic equivalence. Native output and ownership parity have separate tests.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_MIR_SHAPE " + string(encoded))
}

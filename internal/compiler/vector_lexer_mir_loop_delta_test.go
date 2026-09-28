package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/mir"
)

type scannerMIRLoopDelta struct {
	Variant        string         `json:"variant"`
	Function       string         `json:"function"`
	ScannerSHA256  string         `json:"scanner_sha256"`
	Blocks         int            `json:"blocks"`
	Operations     int            `json:"operations"`
	OperationKinds map[string]int `json:"operation_kinds"`
	OperandShapes  map[string]int `json:"operand_shapes"`
	SuccessorEdges map[string]int `json:"successor_edges"`
}

func scannerMIRIntegerDelta(before, after map[string]int) map[string]int {
	result := make(map[string]int)
	for key, count := range before {
		result[key] = -count
	}
	for key, count := range after {
		result[key] += count
	}
	for key, count := range result {
		if count == 0 {
			delete(result, key)
		}
	}
	return result
}

func scannerMIRCanonicalType(raw string, specializations map[string]string) (string, error) {
	for generated, identity := range specializations {
		raw = strings.ReplaceAll(raw, generated, "generic<"+identity+">")
	}
	if strings.Contains(raw, "_fy_generic_") {
		return "", fmt.Errorf("unresolved scanner generic type %s", raw)
	}
	return raw, nil
}

// Operand shapes omit numeric local identities and source positions. Generated
// SDK type names are resolved to their declaration and type bindings first.
func scannerMIROperandShapes(program mir.Program, name string, specializations map[string]string) (map[string]int, error) {
	result := make(map[string]int)
	for _, function := range program.Functions {
		if function.Name != name {
			continue
		}
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				typeName, err := scannerMIRCanonicalType(string(operation.Type), specializations)
				if err != nil {
					return nil, err
				}
				key := fmt.Sprintf("%s type=%s operator=%s field=%s", operation.Kind, typeName,
					operation.Operator, operation.Field)
				for _, operand := range operation.Operands {
					typeName, err := scannerMIRCanonicalType(string(operand.Type), specializations)
					if err != nil {
						return nil, err
					}
					key += fmt.Sprintf(" input=%s:%s", operand.Kind, typeName)
					if operand.Kind != mir.OperandLocal {
						key += fmt.Sprintf(":%d:%d", operand.I32, operand.U64)
					}
				}
				result[key]++
			}
		}
	}
	return result, nil
}

func TestPrivateVectorSDKCandidateScannerMIRLoopDelta(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_MIR_LOOP_DELTA") != "1" {
		t.Skip("opt-in scanner MIR loop attribution")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	names := []string{"$p1::scanner::report_work", "$p1::scanner::storage"}
	var shapes [5][2]scannerMIRShape
	var operands [5][2]map[string]int
	for variantIndex, variant := range variants {
		var plan genericplan.Plan
		program, _, count := scannerBudgetFairyProgramWithMainTimingPlan(t, variant,
			realScannerExecutionMain, nil, &plan)
		if count != 21 || len(plan.Specializations) != count {
			t.Fatalf("%s specializations = %d/%d", variant.name, count, len(plan.Specializations))
		}
		canonical := scannerCanonicalSpecializations(plan)
		for functionIndex, name := range names {
			var err error
			shapes[variantIndex][functionIndex], err = scannerMIRFunctionShape(program, name)
			if err != nil {
				t.Fatalf("%s %s: %v", variant.name, name, err)
			}
			if shape := shapes[variantIndex][functionIndex]; shape.ReachableBlocks != shape.Blocks ||
				shape.ReachableOperations != shape.Operations {
				t.Fatalf("%s %s has unreachable MIR: %+v", variant.name, name, shape)
			}
			operands[variantIndex][functionIndex], err = scannerMIROperandShapes(program, name, canonical)
			if err != nil {
				t.Fatalf("%s %s operand shape: %v", variant.name, name, err)
			}
		}
	}
	rows := make([]scannerMIRLoopDelta, 0, 8)
	for variantIndex := 1; variantIndex < len(variants); variantIndex++ {
		for functionIndex, name := range names {
			baseline, current := shapes[0][functionIndex], shapes[variantIndex][functionIndex]
			rows = append(rows, scannerMIRLoopDelta{
				Variant: variants[variantIndex].name, Function: name,
				ScannerSHA256:  fmt.Sprintf("%x", sha256.Sum256([]byte(variants[variantIndex].scanner))),
				Blocks:         current.ReachableBlocks - baseline.ReachableBlocks,
				Operations:     current.ReachableOperations - baseline.ReachableOperations,
				OperationKinds: scannerMIRIntegerDelta(baseline.OperationKinds, current.OperationKinds),
				OperandShapes:  scannerMIRIntegerDelta(operands[0][functionIndex], operands[variantIndex][functionIndex]),
				SuccessorEdges: scannerMIRIntegerDelta(baseline.ReachableEdges, current.ReachableEdges),
			})
		}
	}
	for functionIndex, name := range names {
		all := rows[6+functionIndex]
		parts := make(map[string]int)
		operandParts := make(map[string]int)
		edgeParts := make(map[string]int)
		blocks, operations := 0, 0
		for variantIndex := 0; variantIndex < 3; variantIndex++ {
			row := rows[variantIndex*2+functionIndex]
			blocks += row.Blocks
			operations += row.Operations
			for key, value := range row.OperationKinds {
				parts[key] += value
			}
			for key, value := range row.OperandShapes {
				operandParts[key] += value
			}
			for key, value := range row.SuccessorEdges {
				edgeParts[key] += value
			}
		}
		for key, value := range parts {
			if value == 0 {
				delete(parts, key)
			}
		}
		for key, value := range operandParts {
			if value == 0 {
				delete(operandParts, key)
			}
		}
		for key, value := range edgeParts {
			if value == 0 {
				delete(edgeParts, key)
			}
		}
		if all.Blocks != blocks || all.Operations != operations || !maps.Equal(all.OperationKinds, parts) ||
			!maps.Equal(all.OperandShapes, operandParts) || !maps.Equal(all.SuccessorEdges, edgeParts) {
			t.Fatalf("%s loop deltas do not account for combined MIR: all=%+v parts=%v blocks=%d operations=%d",
				name, all, parts, blocks, operations)
		}
	}
	report := map[string]any{
		"schema": "fairyc.vector-lexer-mir-loop-delta/v2", "rows": rows,
		"baseline_scanner_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(variants[0].scanner))),
		"scope":                   "Five independently analyzed scanner variants. Operation and successor counts are reachable MIR counts. Operand shapes normalize SDK generic names to declaration and bindings but omit local identities and source positions. Additivity covers operation kinds, operand shapes, successor kinds, blocks and operations, not CFG target or dataflow equivalence.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_MIR_LOOP_DELTA " + string(encoded))
}

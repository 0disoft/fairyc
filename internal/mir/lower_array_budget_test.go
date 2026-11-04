package mir

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestLowerArrayOperandBudgetBoundaryAndCrossFunctionCharge(t *testing.T) {
	const text = `module main;
fn void alpha() { let u8[2] values = [1, 2]; }
fn void beta() { let u8[3] values = [1, 2, 3]; }
fn i32 main() { return 0; }
`

	for _, testCase := range []struct {
		name         string
		budget       uint64
		wantError    bool
		wantOperands int
	}{
		{name: "below", budget: 6, wantOperands: 5},
		{name: "at", budget: 5, wantOperands: 5},
		{name: "over", budget: 4, wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := lowerArrayBudgetSource(t, text, testCase.budget)
			if testCase.wantError {
				if err == nil || !strings.Contains(err.Error(), "expanded fixed-array operand budget exceeded") {
					t.Fatalf("lower with budget %d error = %v, want budget rejection", testCase.budget, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("lower with budget %d error = %v", testCase.budget, err)
			}
			if got := countArrayConstructOperands(program); got != testCase.wantOperands {
				t.Fatalf("array construct operands = %d, want %d", got, testCase.wantOperands)
			}
			if failures := Verify(program); len(failures) != 0 {
				t.Fatalf("Verify() failures = %+v", failures)
			}
		})
	}
}

func TestLowerRepeatedArrayChargesExpandedOperandsAndEvaluatesInitializerOnce(t *testing.T) {
	program, err := lowerArrayBudgetSource(t, `module main;
fn i32 next() { return 9; }
fn i32 main() {
    let i32[3] values = [next(); 3];
    return values[0];
}
`, 3)
	if err != nil {
		t.Fatalf("lower repeated array error = %v", err)
	}
	mainFunction := findArrayBudgetFunction(program, "main::main")
	if mainFunction == nil || len(mainFunction.Calls) != 1 {
		t.Fatalf("main calls = %+v, want one repeated-array initializer evaluation", mainFunction)
	}
	construct := findArrayConstruct(mainFunction)
	if construct == nil || len(construct.Operands) != 3 {
		t.Fatalf("repeated array construct = %+v, want three expanded operands", construct)
	}
	for index := range construct.Operands {
		if construct.Operands[index] != construct.Operands[0] {
			t.Fatalf("expanded operand %d = %+v, want initializer operand %+v", index, construct.Operands[index], construct.Operands[0])
		}
	}
}

func TestArrayOperandBudgetRejectsOverflowWithoutCharging(t *testing.T) {
	budget := arrayOperandBudget{remaining: 3}
	if err := budget.reserve(^uint64(0)); err == nil {
		t.Fatal("overflow-sized reservation succeeded")
	}
	if budget.remaining != 3 {
		t.Fatalf("remaining after rejected reservation = %d, want 3", budget.remaining)
	}
	if err := budget.reserve(2); err != nil {
		t.Fatalf("reserve below boundary = %v", err)
	}
	if err := budget.reserve(2); err == nil {
		t.Fatal("reservation beyond exact remaining budget succeeded")
	}
	if budget.remaining != 1 {
		t.Fatalf("remaining after over-budget reservation = %d, want 1", budget.remaining)
	}
	if err := budget.reserve(1); err != nil {
		t.Fatalf("reserve at exact boundary = %v", err)
	}
}

func TestLowerArrayConstructionValidatesShapeBeforeBudget(t *testing.T) {
	input := arrayBudgetHIR(t, `module main;
fn void fill() { let u8[2] values = [1, 2]; }
`)
	expression := input.Modules[0].Functions[0].Body.Statements[0].Expression
	expression.Type = hir.FixedArray(hir.TypeU8, hir.MaxExecutableFixedArrayElements+1)

	_, err := lowerContextWithArrayOperandBudget(context.Background(), input, 0)
	if err == nil || !strings.Contains(err.Error(), "malformed fixed-array construction") {
		t.Fatalf("oversized malformed construction error = %v", err)
	}
	if strings.Contains(err.Error(), "operand budget") {
		t.Fatalf("budget error preceded malformed-shape validation: %v", err)
	}
}

func TestLowerArrayConstructionHonorsCanceledContextBeforeBudget(t *testing.T) {
	input := arrayBudgetHIR(t, `module main;
fn void fill() { let u8[2] values = [1, 2]; }
`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := lowerContextWithArrayOperandBudget(ctx, input, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled lower error = %v, want context.Canceled", err)
	}
}

func lowerArrayBudgetSource(t *testing.T, text string, budget uint64) (Program, error) {
	t.Helper()
	return lowerContextWithArrayOperandBudget(context.Background(), arrayBudgetHIR(t, text), budget)
}

func arrayBudgetHIR(t *testing.T, text string) hir.Package {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
		t.Fatalf("AnalyzeSyntax() diagnostics = %+v, AST = %+v", parsed.Tree.Diagnostics, parsed.AST)
	}
	result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("Analyze() result = %+v", result)
	}
	return result.HIR
}

func countArrayConstructOperands(program Program) int {
	total := 0
	for _, function := range program.Functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind == OperationArrayConstruct {
					total += len(operation.Operands)
				}
			}
		}
	}
	return total
}

func findArrayBudgetFunction(program Program, name string) *Function {
	for index := range program.Functions {
		if program.Functions[index].Name == name {
			return &program.Functions[index]
		}
	}
	return nil
}

func findArrayConstruct(function *Function) *Operation {
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationArrayConstruct {
				return operation
			}
		}
	}
	return nil
}

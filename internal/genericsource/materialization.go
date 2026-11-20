package genericsource

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
)

// The parser admits at most one million syntax nodes in one source file. Give
// original declarations and generic specializations independent source-sized
// allowances so the amount of written code does not spend the expansion budget.
const defaultMaterializedASTNodeBudget uint64 = 1 << 20

type materializationTask struct {
	module      *expander
	span        diagnostic.Span
	nodes       uint64
	copies      uint64
	specialized bool
}

func checkMaterializationBudget(modules []*expander, plan genericplan.Plan, budget uint64) error {
	if budget == 0 {
		return materializationBudgetError("materialized AST node budget must be positive")
	}
	tasks, err := materializationTasks(modules, plan)
	if err != nil {
		return err
	}
	var ordinary, specialized uint64
	for _, task := range tasks {
		total := &ordinary
		if task.specialized {
			total = &specialized
		}
		*total, err = addMaterializedNodeWork(*total, task.nodes, task.copies, budget)
		if err != nil {
			return task.module.at(task.span, FailureExpansion, err)
		}
	}
	return nil
}

func materializedASTWork(modules []*expander, plan genericplan.Plan) (uint64, error) {
	tasks, err := materializationTasks(modules, plan)
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, task := range tasks {
		total, err = addMaterializedNodeWork(total, task.nodes, task.copies, ^uint64(0))
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

func materializationTasks(modules []*expander, plan genericplan.Plan) ([]materializationTask, error) {
	specializations := make(map[genericplan.DeclarationID]uint64, len(plan.Specializations))
	for _, specialization := range plan.Specializations {
		specializations[specialization.Declaration]++
	}
	var tasks []materializationTask
	for _, module := range modules {
		if module == nil || module.module == nil {
			continue
		}
		for i := range module.module.Functions {
			function := &module.module.Functions[i]
			copies := uint64(1)
			if len(function.TypeParameters) != 0 {
				copies = specializations[module.id(function.Name)]
			}
			if copies == 0 {
				continue
			}
			nodes, err := materializedFunctionNodes(*function, module.inferred)
			if err != nil {
				return nil, module.at(function.Span, FailureExpansion, err)
			}
			tasks = append(tasks, materializationTask{module: module, span: function.Span, nodes: nodes, copies: copies, specialized: len(function.TypeParameters) != 0})
		}
		for _, aggregate := range module.sourceAggregates() {
			shape := aggregate.shape()
			copies := uint64(1)
			if len(shape.TypeParameters) != 0 {
				copies = specializations[module.id(shape.Name)]
			}
			if copies == 0 {
				continue
			}
			nodes, err := materializedAggregateNodes(aggregate, module.inferred)
			if err != nil {
				return nil, module.at(shape.Span, FailureExpansion, err)
			}
			tasks = append(tasks, materializationTask{module: module, span: shape.Span, nodes: nodes, copies: copies, specialized: len(shape.TypeParameters) != 0})
		}
	}
	return tasks, nil
}

func materializedFunctionNodes(function ast.Function, inferred map[*ast.Expression][]ast.TypeReference) (uint64, error) {
	var nodes uint64
	err := (sourceVisitor{nodes: &nodes, inferred: inferred}).function(function)
	return nodes, err
}

func materializedAggregateNodes(aggregate aggregateTemplate, inferred map[*ast.Expression][]ast.TypeReference) (uint64, error) {
	var nodes uint64
	visitor := sourceVisitor{nodes: &nodes, inferred: inferred}
	shape := aggregate.shape()
	for _, field := range shape.Fields {
		visitor.node()
		visitor.typ(field.Type)
	}
	if aggregate.resource != nil && aggregate.resource.Cleanup != nil {
		if err := visitor.block(aggregate.resource.Cleanup.Body); err != nil {
			return 0, err
		}
	}
	return nodes, nil
}

func addMaterializedNodeWork(total, nodes, copies, budget uint64) (uint64, error) {
	if budget == 0 {
		return total, materializationBudgetError("materialized AST node budget must be positive")
	}
	if nodes == 0 || copies == 0 {
		return total, nil
	}
	if copies > (^uint64(0)-total)/nodes {
		return total, materializationBudgetError("materialized AST node work overflows uint64")
	}
	increase := nodes * copies
	if total > budget || increase > budget-total {
		return total, materializationBudgetError("materialized AST work exceeds the %d node budget", budget)
	}
	return total + increase, nil
}

func materializationBudgetError(format string, arguments ...any) error {
	return &genericplan.Error{Code: genericplan.CodePlanningBudgetExceeded, Message: fmt.Sprintf(format, arguments...)}
}

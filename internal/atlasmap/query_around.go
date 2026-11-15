package atlasmap

func (index *QueryIndex) Around(symbolID string, request RelationRequest) (AroundResult, error) {
	relations, filter, err := validateRelationRequest(request)
	if err != nil {
		return AroundResult{}, err
	}
	rootPosition, err := index.resolveSymbol(symbolID)
	if err != nil {
		return AroundResult{}, err
	}
	context := index.contextFor(relations)
	root := AroundNode{Depth: 0, Node: index.node(rootPosition)}
	selection := newSelectionBudget(request.Budget)
	mandatory := struct {
		Context QueryContext `json:"context"`
		Root    AroundNode   `json:"root"`
	}{Context: context, Root: root}
	accepted, _, err := selection.reserveRecord(mandatory)
	if err != nil {
		return AroundResult{}, err
	}
	if !accepted {
		return AroundResult{}, ErrQueryBudgetTooSmall
	}
	result := AroundResult{
		Context: context, Subject: index.nodes[rootPosition].symbol.Visible, Direction: request.Direction,
		RelationKinds: relations, Nodes: []AroundNode{root}, Budget: request.Budget,
		Omissions: filterOmissions(relations),
	}
	walk := aroundTraversal{
		index: index, direction: request.Direction, filter: filter, selection: selection, result: &result,
		seenNodes: map[int]struct{}{rootPosition: {}}, seenRelations: make(map[int]struct{}),
	}
	if err := walk.run(rootPosition, request.Budget.MaxDepth); err != nil {
		return AroundResult{}, err
	}
	return result, nil
}

// Each traversal owns its result and working sets; the request and index stay immutable.
type aroundTraversal struct {
	index         *QueryIndex
	direction     QueryDirection
	filter        map[RelationKind]struct{}
	selection     *selectionBudget
	result        *AroundResult
	seenNodes     map[int]struct{}
	seenRelations map[int]struct{}
}

func (walk *aroundTraversal) run(root, maxDepth int) error {
	frontier := []int{root}
	for depth := 0; depth < maxDepth && len(frontier) != 0; depth++ {
		next, exhausted, err := walk.nextDepth(frontier, depth+1)
		if err != nil {
			return err
		}
		if exhausted {
			walk.finish()
			return nil
		}
		frontier = next
	}
	if len(frontier) != 0 && walk.index.hasUnselectedExpansion(frontier, walk.direction, walk.filter, walk.seenRelations) {
		walk.result.Omissions = append(walk.result.Omissions, unknownOmission(OmissionDepthLimit, "relations"))
	}
	walk.finish()
	return nil
}

func (walk *aroundTraversal) nextDepth(frontier []int, depth int) ([]int, bool, error) {
	nextSet := make(map[int]struct{})
	var frontierSymbols map[string]struct{}
	for relationIndex := range walk.index.frontierRelations(frontier, walk.direction, walk.filter, walk.seenRelations) {
		relation := cloneQueryRelation(walk.index.relations[relationIndex])
		accepted, code, err := walk.selection.reserveRelation(relation)
		if err != nil {
			return nil, false, err
		}
		if !accepted {
			walk.result.Omissions = append(walk.result.Omissions, unknownOmission(code, "relations"))
			return nil, true, nil
		}
		walk.result.Relations = append(walk.result.Relations, relation)
		walk.seenRelations[relationIndex] = struct{}{}
		// Build membership once per depth, after the first relation fits the budget.
		if frontierSymbols == nil {
			frontierSymbols = walk.index.frontierSymbols(frontier)
		}
		neighbors, count := walk.index.adjacentNodes(frontierSymbols, relation, walk.direction)
		if exhausted, err := walk.selectNeighbors(neighbors[:count], depth, nextSet); err != nil || exhausted {
			return nil, exhausted, err
		}
	}
	return sortedNodePositions(nextSet, walk.index.nodes), false, nil
}

func (walk *aroundTraversal) selectNeighbors(neighbors []int, depth int, nextSet map[int]struct{}) (bool, error) {
	for _, neighbor := range neighbors {
		if _, seen := walk.seenNodes[neighbor]; seen {
			continue
		}
		node := AroundNode{Depth: depth, Node: walk.index.node(neighbor)}
		accepted, code, err := walk.selection.reserveRecord(node)
		if err != nil {
			return false, err
		}
		if !accepted {
			walk.result.Omissions = append(walk.result.Omissions, unknownOmission(code, "nodes"))
			return true, nil
		}
		walk.seenNodes[neighbor] = struct{}{}
		nextSet[neighbor] = struct{}{}
		walk.result.Nodes = append(walk.result.Nodes, node)
		walk.selection.usage.Depth = max(walk.selection.usage.Depth, depth)
	}
	return false, nil
}

func (walk *aroundTraversal) finish() {
	walk.result.Omissions = appendJoinOmissions(walk.result.Omissions, walk.result.Relations, walk.direction, walk.filter)
	walk.result.Usage = walk.selection.usage
}

func (index *QueryIndex) frontierSymbols(frontier []int) map[string]struct{} {
	symbols := make(map[string]struct{}, len(frontier))
	for _, position := range frontier {
		symbols[index.nodes[position].symbol.Visible] = struct{}{}
	}
	return symbols
}

// A relation can reach only its source and target. Return owned fixed storage.
const queryRelationEndpointCount = 2

func (index *QueryIndex) adjacentNodes(frontierSymbols map[string]struct{}, relation QueryRelation, direction QueryDirection) (positions [queryRelationEndpointCount]int, count int) {
	if relation.Target == nil || relation.Target.Symbol == "" {
		return positions, 0
	}
	if direction == DirectionOutgoing || direction == DirectionBoth {
		if _, found := frontierSymbols[relation.Source]; found {
			positions[count] = index.byVisible[relation.Target.Symbol]
			count++
		}
	}
	if direction != DirectionIncoming && direction != DirectionBoth {
		return positions, count
	}
	if _, found := frontierSymbols[relation.Target.Symbol]; !found {
		return positions, count
	}
	position := index.byVisible[relation.Source]
	if count == 0 || positions[0] != position {
		positions[count] = position
		count++
	}
	if count == queryRelationEndpointCount && index.nodes[positions[1]].symbol.Full < index.nodes[positions[0]].symbol.Full {
		positions[0], positions[1] = positions[1], positions[0]
	}
	return positions, count
}

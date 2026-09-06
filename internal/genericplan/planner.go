package genericplan

import (
	"cmp"
	"slices"
	"strings"
)

// PlanLimits bound monomorphization growth and recursive dependency expansion.
type PlanLimits struct {
	MaxSpecializations               int
	MaxSpecializationsPerDeclaration int
	MaxExpansionDepth                int
}

// DefaultPlanLimits returns conservative first-kernel limits.
func DefaultPlanLimits() PlanLimits {
	return PlanLimits{
		MaxSpecializations:               4096,
		MaxSpecializationsPerDeclaration: 256,
		MaxExpansionDepth:                64,
	}
}

type planState uint8

const (
	planUnseen planState = iota
	planVisiting
	planDone
)

type plannedNode struct {
	state          planState
	specialization Specialization
}

type planner struct {
	catalog *Catalog
	limits  PlanLimits
	nodes   map[string]*plannedNode
	perDecl map[DeclarationID]int
	stack   []string
	ordered []Specialization
}

// Plan resolves requests under exact static constraints and returns each
// dependency before every consumer. It is safe to call concurrently because the
// catalog is immutable and all planning state is local to this call.
func (c *Catalog) Plan(requests []Request, limits PlanLimits) (Plan, error) {
	if c == nil {
		return Plan{}, fail(CodeUnknownCatalogItem, "catalog is nil")
	}
	if limits.MaxSpecializations <= 0 || limits.MaxSpecializationsPerDeclaration <= 0 || limits.MaxExpansionDepth <= 0 {
		return Plan{}, fail(CodePlanningBudgetExceeded, "every planning limit must be positive")
	}
	if len(requests) == 0 {
		return Plan{CatalogDigest: c.digest}, nil
	}

	roots := cloneRequests(requests)
	slices.SortFunc(roots, compareRequest)
	roots = slices.CompactFunc(roots, func(left, right Request) bool {
		return compareRequest(left, right) == 0
	})
	value := &planner{
		catalog: c,
		limits:  limits,
		nodes:   make(map[string]*plannedNode),
		perDecl: make(map[DeclarationID]int),
	}
	rootIDs := make([]SpecializationID, 0, len(roots))
	for _, request := range roots {
		id, err := value.expand(request, 1)
		if err != nil {
			return Plan{}, &RequestError{Request: Request{Declaration: request.Declaration, Arguments: slices.Clone(request.Arguments)}, Cause: err}
		}
		rootIDs = append(rootIDs, id)
	}
	return Plan{
		CatalogDigest:   c.digest,
		Roots:           slices.Clone(rootIDs),
		Specializations: cloneSpecializations(value.ordered),
	}, nil
}

func (p *planner) expand(request Request, depth int) (SpecializationID, error) {
	if depth > p.limits.MaxExpansionDepth {
		return "", fail(CodePlanningBudgetExceeded, "specialization expansion exceeds depth %d at %s", p.limits.MaxExpansionDepth, requestKey(request))
	}
	declaration, found := p.catalog.declarations[request.Declaration]
	if !found {
		return "", fail(CodeUnknownCatalogItem, "unknown generic declaration %q", request.Declaration)
	}
	if len(request.Arguments) != len(declaration.Parameters) {
		return "", fail(CodeTypeArgumentArity, "request for %q has %d arguments, expected %d", request.Declaration, len(request.Arguments), len(declaration.Parameters))
	}

	key := requestKey(request)
	if node := p.nodes[key]; node != nil {
		switch node.state {
		case planDone:
			return node.specialization.ID, nil
		case planVisiting:
			cycle := append(slices.Clone(p.stack), key)
			return "", fail(CodeSpecializationCycle, "specialization dependency cycle: %s", strings.Join(cycle, " -> "))
		}
	}
	if len(p.nodes) >= p.limits.MaxSpecializations {
		return "", fail(CodePlanningBudgetExceeded, "specialization count exceeds %d", p.limits.MaxSpecializations)
	}
	if p.perDecl[declaration.ID] >= p.limits.MaxSpecializationsPerDeclaration {
		return "", fail(CodePlanningBudgetExceeded, "declaration %q exceeds %d specializations", declaration.ID, p.limits.MaxSpecializationsPerDeclaration)
	}

	bindings := make(map[string]TypeDescriptor, len(declaration.Parameters))
	bindingList := make([]Binding, len(declaration.Parameters))
	for index, parameter := range declaration.Parameters {
		typeID := request.Arguments[index]
		typeValue, found := p.catalog.types[typeID]
		if !found {
			return "", fail(CodeUnknownCatalogItem, "request for %q references unknown type %q", declaration.ID, typeID)
		}
		for _, required := range parameter.Abilities {
			if !hasAbility(typeValue.Abilities, required) {
				return "", fail(CodeAbilityUnsatisfied, "type %q bound to %s in %q lacks ability %q", typeID, parameter.Name, declaration.ID, required)
			}
		}
		bindings[parameter.Name] = typeValue
		bindingList[index] = Binding{Parameter: parameter.Name, Type: typeID}
	}

	resolvedConstraints := make([]ResolvedConstraint, 0, len(declaration.Constraints))
	for _, constraint := range declaration.Constraints {
		arguments, err := p.resolveExpressions(constraint.Arguments, bindings)
		if err != nil {
			return "", err
		}
		implementation, found := p.catalog.implementations[implementationKey(constraint.Capability, arguments)]
		if !found {
			return "", fail(CodeImplementationMissing, "specialization %s has no exact implementation for %s", key, implementationKey(constraint.Capability, arguments))
		}
		resolvedConstraints = append(resolvedConstraints, ResolvedConstraint{
			Capability:     constraint.Capability,
			Arguments:      slices.Clone(arguments),
			Provider:       implementation.Provider,
			ProviderDigest: implementation.ProviderDigest,
		})
	}
	slices.SortFunc(resolvedConstraints, compareResolvedConstraint)

	node := &plannedNode{state: planVisiting}
	p.nodes[key] = node
	p.perDecl[declaration.ID]++
	p.stack = append(p.stack, key)
	decrement := true
	defer func() {
		p.stack = p.stack[:len(p.stack)-1]
		if decrement {
			delete(p.nodes, key)
			p.perDecl[declaration.ID]--
		}
	}()

	dependencyRequests := make([]Request, 0, len(declaration.Dependencies))
	for _, dependency := range declaration.Dependencies {
		arguments, err := p.resolveExpressions(dependency.Arguments, bindings)
		if err != nil {
			return "", err
		}
		dependencyRequests = append(dependencyRequests, Request{Declaration: dependency.Declaration, Arguments: arguments})
	}
	slices.SortFunc(dependencyRequests, compareRequest)
	dependencyRequests = slices.CompactFunc(dependencyRequests, func(left, right Request) bool {
		return compareRequest(left, right) == 0
	})
	dependencyIDs := make([]SpecializationID, 0, len(dependencyRequests))
	for _, dependency := range dependencyRequests {
		id, err := p.expand(dependency, depth+1)
		if err != nil {
			return "", err
		}
		dependencyIDs = append(dependencyIDs, id)
	}
	slices.SortFunc(dependencyIDs, func(left, right SpecializationID) int {
		return cmp.Compare(string(left), string(right))
	})

	specialization := Specialization{
		Declaration:  declaration.ID,
		Bindings:     bindingList,
		Constraints:  resolvedConstraints,
		Dependencies: dependencyIDs,
	}
	specialization.ID = p.specializationID(declaration, request.Arguments, resolvedConstraints, dependencyIDs)
	node.state = planDone
	node.specialization = specialization
	p.ordered = append(p.ordered, specialization)
	decrement = false
	return specialization.ID, nil
}

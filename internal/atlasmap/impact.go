package atlasmap

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

type ImpactNode struct {
	Subject WorkspaceSymbol `json:"subject"`
	Depth   int             `json:"depth"`
}

type ImpactResult struct {
	Schema                string          `json:"schema"`
	Scope                 string          `json:"scope"`
	State                 string          `json:"state"`
	Subject               WorkspaceSymbol `json:"subject"`
	SubjectSpecialization string          `json:"subjectSpecialization,omitempty"`
	Nodes                 []ImpactNode    `json:"nodes"`
	Contexts              []QueryContext  `json:"contexts"`
	TraversalComplete     bool            `json:"traversalComplete"`
	Omissions             []string        `json:"omissions"`
}

// ValidateImpactSelector checks the syntax before opening a workspace.
func ValidateImpactSelector(selector string) error {
	_, _, err := parseImpactSelector(selector)
	return err
}

func parseImpactSelector(selector string) (string, string, error) {
	symbol, implementation, concrete := strings.Cut(selector, "#")
	if _, err := symbolidentity.ParseVisibleID(symbol); err != nil {
		if _, err := symbolidentity.ParseFullDigest(symbol); err != nil {
			return "", "", fmt.Errorf("canonical SymbolId required")
		}
	}
	if concrete && !validSpecializationID(implementation) {
		return "", "", fmt.Errorf("canonical specialization ID required after #")
	}
	return symbol, implementation, nil
}

// Impact traverses only compiler-resolved reverse calls in the exact-current joined closure.
// A SymbolId#specialization selector starts at exactly one verified implementation.
// Even a complete traversal is bounded evidence, never proof that a change is safe.
func (index *WorkspaceCallIndex) Impact(instance, symbol string, maxDepth, maxSymbols int) (ImpactResult, error) {
	if maxDepth < 0 || maxDepth > 64 || maxSymbols < 1 || maxSymbols > 4096 {
		return ImpactResult{}, fmt.Errorf("invalid impact traversal limits")
	}
	symbol, specialization, err := parseImpactSelector(symbol)
	if err != nil {
		return ImpactResult{}, err
	}
	root, err := index.ResolveSymbol(instance, symbol)
	if err != nil {
		return ImpactResult{}, err
	}
	if specialization != "" && index.packages[instance].specializationOwners[specialization] != root.Symbol {
		return ImpactResult{}, fmt.Errorf("impact specialization is not owned by the selected declaration")
	}
	r := ImpactResult{Schema: "fairyc.atlas-impact/v1", Scope: "resolved-direct-call-reachability", State: "bounded", Subject: root, SubjectSpecialization: specialization, Nodes: []ImpactNode{{Subject: root}}, Contexts: []QueryContext{}, TraversalComplete: true, Omissions: []string{"non-call-dependencies-not-analyzed", "external-callback-and-dynamic-targets-not-inferred", "outside-workspace-callers-not-analyzed"}}
	instances := make([]string, 0, len(index.packages))
	for instance := range index.packages {
		instances = append(instances, instance)
	}
	slices.Sort(instances)
	for _, instance := range instances {
		context, _ := index.PackageContext(instance)
		r.Contexts = append(r.Contexts, context)
		if context.Generation != GenerationExact || len(context.MapOmissions) != 0 {
			r.State = "partial"
		}
		for _, c := range context.Completeness {
			if c.Status != semanticsnapshot.CompletenessExact || c.Unknown != 0 || c.External != 0 {
				r.State = "partial"
			}
		}
	}
	type step struct {
		endpoint workspaceCallEndpoint
		depth    int
	}
	start := workspaceCallEndpoint{symbol: root, specialization: specialization}
	queue := []step{{endpoint: start}}
	visited := map[workspaceCallEndpoint]bool{start: true}
	reported := map[WorkspaceSymbol]bool{root: true}
	depthCut, symbolCut := false, false
	for next := 0; next < len(queue); next++ {
		node := queue[next]
		positions := index.incoming[node.endpoint.symbol]
		if node.endpoint.specialization != "" {
			positions = index.incomingSpecializations[node.endpoint]
		}
		// Declaration queries cover the root's full incoming call set. Concrete
		// steps follow only edges for the matching implementation.
		for _, position := range positions {
			call := index.calls[position]
			caller := workspaceCallEndpoint{call.Source, call.SourceSpecialization}
			if visited[caller] || (specialization == "" && caller.symbol == root) {
				continue
			}
			if node.depth >= maxDepth {
				depthCut = true
				continue
			}
			if !reported[caller.symbol] {
				if len(r.Nodes) >= maxSymbols {
					symbolCut = true
					continue
				}
				reported[caller.symbol] = true
				r.Nodes = append(r.Nodes, ImpactNode{Subject: caller.symbol, Depth: node.depth + 1})
			}
			visited[caller] = true
			queue = append(queue, step{endpoint: caller, depth: node.depth + 1})
		}
	}
	if depthCut || symbolCut {
		r.State = "partial"
		r.TraversalComplete = false
	}
	if depthCut {
		r.Omissions = append(r.Omissions, "depth-limit")
	}
	if symbolCut {
		r.Omissions = append(r.Omissions, "symbol-limit")
	}
	slices.SortFunc(r.Nodes, func(a, b ImpactNode) int {
		if n := cmp.Compare(a.Depth, b.Depth); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Subject.PackageInstance, b.Subject.PackageInstance); n != 0 {
			return n
		}
		return cmp.Compare(a.Subject.Symbol, b.Subject.Symbol)
	})
	return r, nil
}

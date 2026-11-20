package atlasmap

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type ImpactNode struct {
	Subject WorkspaceSymbol `json:"subject"`
	Depth   int             `json:"depth"`
}

type ImpactResult struct {
	Schema            string          `json:"schema"`
	Scope             string          `json:"scope"`
	State             string          `json:"state"`
	Subject           WorkspaceSymbol `json:"subject"`
	Nodes             []ImpactNode    `json:"nodes"`
	Contexts          []QueryContext  `json:"contexts"`
	TraversalComplete bool            `json:"traversalComplete"`
	Omissions         []string        `json:"omissions"`
}

// Impact traverses only compiler-resolved reverse calls in the exact-current joined closure.
// Even a complete traversal is bounded evidence, never proof that a change is safe.
func (index *WorkspaceCallIndex) Impact(instance, symbol string, maxDepth, maxSymbols int) (ImpactResult, error) {
	if maxDepth < 0 || maxDepth > 64 || maxSymbols < 1 || maxSymbols > 4096 {
		return ImpactResult{}, fmt.Errorf("invalid impact traversal limits")
	}
	root, err := index.ResolveSymbol(instance, symbol)
	if err != nil {
		return ImpactResult{}, err
	}
	r := ImpactResult{Schema: "fairyc.atlas-impact/v1", Scope: "resolved-direct-call-reachability", State: "bounded", Subject: root, Nodes: []ImpactNode{{Subject: root}}, Contexts: []QueryContext{}, TraversalComplete: true, Omissions: []string{"non-call-dependencies-not-analyzed", "external-callback-and-dynamic-targets-not-inferred", "outside-workspace-callers-not-analyzed"}}
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
	queue := []step{{endpoint: workspaceCallEndpoint{symbol: root}}}
	visited := make(map[workspaceCallEndpoint]bool)
	reported := map[WorkspaceSymbol]bool{root: true}
	depthCut, symbolCut := false, false
	for next := 0; next < len(queue); next++ {
		node := queue[next]
		positions := index.incoming[node.endpoint.symbol]
		if node.endpoint.specialization != "" {
			positions = index.incomingSpecializations[node.endpoint]
		}
		// The root query covers all its implementations. Every subsequent step
		// follows the specific implementation that reached the prior endpoint.
		for _, position := range positions {
			call := index.calls[position]
			caller := workspaceCallEndpoint{call.Source, call.SourceSpecialization}
			if caller.symbol == root || visited[caller] {
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

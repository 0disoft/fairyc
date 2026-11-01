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
	seen := map[WorkspaceSymbol]bool{root: true}
	depthCut, symbolCut := false, false
	for next := 0; next < len(r.Nodes); next++ {
		node := r.Nodes[next]
		// The ledger's deterministic order makes capped selection independent of caller map order.
		for _, position := range index.incoming[node.Subject] {
			caller := index.calls[position].Source
			if seen[caller] {
				continue
			}
			if node.Depth >= maxDepth {
				depthCut = true
				continue
			}
			if len(r.Nodes) >= maxSymbols {
				symbolCut = true
				continue
			}
			seen[caller] = true
			r.Nodes = append(r.Nodes, ImpactNode{Subject: caller, Depth: node.Depth + 1})
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

package contractstatenative

import "github.com/0disoft/fairyc/internal/contractstatemir"

// Count every static block, including mutually exclusive paths, and expand each
// acyclic helper call. Saturation is a transport resource ceiling, not a smaller
// language or transaction limit. This upper bound sizes native scratch safely
// without allocating 64MiB for every small transaction.
func graphWorkBounds(p contractstatemir.Program) (uint64, int) {
	bodies := map[string]graphBody{"": {p.Operations, p.Entry, p.Blocks, p.Exit, p.TrapCategory}}
	for _, h := range p.Helpers {
		bodies[h.Name] = graphBody{h.Operations, h.Entry, h.Blocks, h.Exit, h.TrapCategory}
	}
	known := make(map[string]uint64)
	maxLiteral := 0
	var count func(string) uint64
	count = func(name string) uint64 {
		if n, found := known[name]; found {
			return n
		}
		body := bodies[name]
		n := uint64(max(1, len(body.blocks)))
		groups := body.blocks
		if len(groups) == 0 {
			groups = []contractstatemir.Block{{Operations: body.operations}}
		}
		for _, b := range groups {
			for _, op := range b.Operations {
				maxLiteral = max(maxLiteral, len(op.Key.Encoded), len(op.Value.Encoded))
				n++
				if op.Kind == contractstatemir.OperationCall {
					n += count(op.Callee)
				}
				n = min(n, overlayResponseBytes)
			}
		}
		known[name] = n
		return n
	}
	return count(""), maxLiteral
}

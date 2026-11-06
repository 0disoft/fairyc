package contractstatenative

import (
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
)

// Operation sites retain EmitGraph's numbering. Terminators use a disjoint
// namespace, including a synthetic zero-cost exit for each legacy body.
const graphTerminatorSite = uint32(1 << 31)

type graphSite struct {
	op        contractstatemir.Operation
	field     contractstatemir.LayoutField
	next      []uint32
	callee    uint32
	exit      contractstatemir.ExitKind
	trap      uint32
	cost      uint64
	operation bool
}

func graphPlan(p contractstatemir.Program, traps []string) (map[uint32]graphSite, uint32) {
	bodies := []graphBody{{p.Operations, p.Entry, p.Blocks, p.Exit, p.TrapCategory}}
	helpers := make(map[string]int)
	for i, h := range p.Helpers {
		helpers[h.Name] = i + 1
		bodies = append(bodies, graphBody{h.Operations, h.Entry, h.Blocks, h.Exit, h.TrapCategory})
	}
	type blockSites struct {
		first, term uint32
	}
	blocks := make([][]blockSites, len(bodies))
	starts := make([]uint32, len(bodies))
	var site, term uint32
	for i, body := range bodies {
		groups := body.blocks
		if len(groups) == 0 {
			groups = []contractstatemir.Block{{Operations: body.operations}}
		}
		for _, b := range groups {
			term++
			first := site + 1
			if len(b.Operations) == 0 {
				first = graphTerminatorSite | term
			}
			blocks[i] = append(blocks[i], blockSites{first, graphTerminatorSite | term})
			site += uint32(len(b.Operations))
		}
		starts[i] = blocks[i][body.entry].first
	}
	result := make(map[uint32]graphSite)
	site = 0
	for i, body := range bodies {
		groups := body.blocks
		legacy := len(groups) == 0
		if legacy {
			groups = []contractstatemir.Block{{Operations: body.operations}}
		}
		for j, b := range groups {
			for k, op := range b.Operations {
				site++
				op.Key.Encoded = slices.Clone(op.Key.Encoded)
				op.Value.Encoded = slices.Clone(op.Value.Encoded)
				n := graphSite{op: op, operation: true, next: []uint32{site + 1}}
				if !legacy || op.Kind == contractstatemir.OperationCall {
					n.cost = 1
				}
				if k == len(b.Operations)-1 {
					n.next[0] = blocks[i][j].term
				}
				if op.Kind == contractstatemir.OperationCall {
					n.callee = starts[helpers[op.Callee]]
				} else if op.Kind != contractstatemir.OperationBorrowView && op.Kind != contractstatemir.OperationBorrowCopy {
					n.field = p.Layout.Fields[op.FieldIndex]
				}
				result[site] = n
			}
			n := graphSite{cost: 1}
			if legacy {
				n.cost, n.exit = 0, body.exit
				n.trap = uint32(slices.Index(traps, body.trap) + 1)
			} else {
				t := b.Terminator
				switch t.Kind {
				case contractstatemir.TerminatorJump:
					n.next = []uint32{blocks[i][t.Target].first}
				case contractstatemir.TerminatorBranch:
					n.next = []uint32{blocks[i][t.TrueTarget].first, blocks[i][t.FalseTarget].first}
				case contractstatemir.TerminatorCommit:
					n.exit = contractstatemir.ExitCommit
				case contractstatemir.TerminatorReject:
					n.exit = contractstatemir.ExitReject
				case contractstatemir.TerminatorTrap:
					n.exit, n.trap = contractstatemir.ExitTrap, uint32(slices.Index(traps, t.TrapCategory)+1)
				case contractstatemir.TerminatorReturn:
					n.exit = contractstatemir.ExitReturn
				}
			}
			result[blocks[i][j].term] = n
		}
	}
	return result, starts[0]
}

type graphPath struct {
	next      []uint32
	stack     []uint32
	cost, ops uint64
	exit      contractstatemir.ExitKind
	trap      uint32
}

func (path *graphPath) advance(site uint32, n graphSite) bool {
	if path.exit != "" || !slices.Contains(path.next, site) {
		return false
	}
	path.cost += n.cost
	if n.operation {
		path.ops++
	}
	path.next = n.next
	if n.callee != 0 {
		path.stack = append(path.stack, n.next[0])
		path.next = []uint32{n.callee}
	} else if n.exit == contractstatemir.ExitReturn {
		if len(path.stack) == 0 {
			return false
		}
		path.next = []uint32{path.stack[len(path.stack)-1]}
		path.stack = path.stack[:len(path.stack)-1]
	} else if n.exit != "" {
		path.exit, path.trap = n.exit, n.trap
		path.next = nil
	}
	return true
}

func graphStorageKind(kind contractstatemir.OperationKind) (read, write, deleted bool) {
	switch kind {
	case contractstatemir.OperationCellRead, contractstatemir.OperationMapRead, contractstatemir.OperationCellBorrow, contractstatemir.OperationMapBorrow:
		read = true
	case contractstatemir.OperationCellWrite, contractstatemir.OperationMapWrite:
		write = true
	case contractstatemir.OperationCellDelete, contractstatemir.OperationMapDelete:
		write, deleted = true, true
	}
	return
}

func graphKnownCells(fields []contractstatemir.LayoutField) map[contractstate.Key]string {
	known := make(map[contractstate.Key]string)
	for _, field := range fields {
		if field.Kind == contractstate.FieldCell {
			known[field.Prefix] = field.ValueTypeID
		}
	}
	return known
}

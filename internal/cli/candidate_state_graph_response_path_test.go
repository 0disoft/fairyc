package cli

import (
	"encoding/hex"
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstatemir"
)

// Current source -> Complete -> sealed Verified, with no response/wire parser
// and no test-only authority factory. make_plan returns beyond proof lifetime.
const graphResponsePathProbe = `module main;
import module_state_graph_response_path as path;import module_state_graph_response_common as common;
import project_compiler_core as core;import project_compiler_state_graph as graph;import module_state_graph_program as program;
import module_public_type_text as coordinates;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn path::Plan make_plan(ref u8[] actual,ref u8[] coordinate){
 let data=switch(bytes::copy(coordinate)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};
 var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=coordinate.len}).is_ok);
 let table=coordinates::assemble(data,rows);var usize setup=0;
 let checked=switch(core::build_complete_state_graph(actual,ref table,0,ref mut setup)){case .ok(p):{p}case .err(_):{assert(false);return make_plan(b"",b"");}};
 let proof=graph::program(core::checked_state_graph(ref checked));var work=common::empty_work();
 let plan=switch(path::build(proof,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return make_plan(b"",b"");}};let used=work;
 assert(used.structure>0);var exact=common::Work{.structure=16777216-used.structure,.bytes=1073741824-used.bytes};
 switch(path::build(proof,ref mut exact)){case .ok(_):{assert(exact.structure==16777216 && exact.bytes==1073741824);}case .err(_):{assert(false);}};
 var short=common::Work{.structure=16777217-used.structure,.bytes=0};switch(path::build(proof,ref mut short)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002 && short.structure<=16777216);}};
 if(used.bytes!=0){short=common::Work{.structure=0,.bytes=1073741825-used.bytes};switch(path::build(proof,ref mut short)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002 && short.bytes<=1073741824);}};}
 var usize i=0;while(i<2){var over=common::Work{.structure=if(i==0){16777217}else{0},.bytes=if(i==1){1073741825}else{0}};let before=over;
  switch(path::build(proof,ref mut over)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002 && over.structure==before.structure && over.bytes==before.bytes);}};i+=1;}
 return plan;
}
fn void count(ref path::Plan plan,ref path::Path p,usize status,usize cost,usize ops,bool want){
 var work=common::empty_work();switch(path::counters(plan,p,status,cost,ops,ref mut work)){case .ok(ok):{assert(ok==want);}case .err(_):{assert(false);}};let used=work.structure;
 work=common::Work{.structure=16777216-used,.bytes=1073741824};switch(path::counters(plan,p,status,cost,ops,ref mut work)){case .ok(ok):{assert(ok==want && work.structure==16777216);}case .err(_):{assert(false);}};
 work=common::Work{.structure=16777217-used,.bytes=0};switch(path::counters(plan,p,status,cost,ops,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002 && work.structure<=16777216);}};
 work=common::Work{.structure=0,.bytes=1073741825};switch(path::counters(plan,p,status,cost,ops,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002);}};
 work=common::Work{.structure=16777217,.bytes=0};switch(path::counters(plan,p,status,cost,ops,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002 && work.structure==16777217);}};
}
fn void refuse(ref path::Plan plan,ref mut path::Path p,usize site,usize op){
 let old_cost=path::cost(p);let old_ops=path::operations(p);var work=common::empty_work();
 switch(path::advance(plan,p,site,op,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1001 && work.structure==1 && path::cost(p)==old_cost && path::operations(p)==old_ops);}};
}
fn void take(ref path::Plan plan,ref mut path::Path p,usize site,usize op,usize body,usize index,bool operation){
 let old_cost=path::cost(p);let old_ops=path::operations(p);
 // Bad calls consume work but preserve the execution path.
 var work=common::empty_work();switch(path::advance(plan,p,0,op,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1001 && work.structure==1);}};
 work=common::empty_work();switch(path::advance(plan,p,site,op+1,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1001 && work.structure==1);}};
 work=common::empty_work();switch(path::advance(plan,p,4294967295,op,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1001);}};
 assert(path::cost(p)==old_cost && path::operations(p)==old_ops);
 count(plan,p,0,old_cost,old_ops,false);count(plan,p,1,old_cost,old_ops,true);
 count(plan,p,1,old_cost+1,op,true);count(plan,p,0,old_cost+1,op,false);count(plan,p,1,old_cost+2,op,false);
 count(plan,p,1,old_cost,old_ops+1,false);
 count(plan,p,1,old_cost+1,old_ops+2,false);
 work=common::Work{.structure=16777216,.bytes=0};switch(path::advance(plan,p,site,op,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002);}};
 work=common::Work{.structure=0,.bytes=1073741825};switch(path::advance(plan,p,site,op,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002);}};
 work=common::Work{.structure=16777217,.bytes=0};switch(path::advance(plan,p,site,op,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002 && work.structure==16777217);}};
 work=common::Work{.structure=16777215,.bytes=1073741824};let step=switch(path::advance(plan,p,site,op,ref mut work)){case .ok(s):{s}case .err(_):{assert(false);common::Step{.body=0,.index=0,.operation=false}}};
 assert(step.body==body && step.index==index && step.operation==operation && work.structure==16777216 && path::cost(p)==old_cost+1 && path::operations(p)==op);
}
fn void finish(ref path::Plan plan,ref mut path::Path p,usize exit,usize trap,usize cost,usize ops){
 assert(path::exit(p)==exit && path::trap(p)==trap && path::cost(p)==cost && path::operations(p)==ops);
 count(plan,p,0,cost,ops,true);count(plan,p,1,cost,ops,true);count(plan,p,1,cost+1,ops,false);count(plan,p,1,cost+1,ops+1,false);
 var work=common::empty_work();switch(path::advance(plan,p,1,ops+1,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1001 && work.structure==1);}};
}
`

type graphResponsePathSite struct {
	id, next, alternative, callee uint32
	body, index, exit, trap       int
	operation                     bool
}

// Structural oracle follows graph_plan.go's two site namespaces. Paths explore
// both CFG branch arms; helper return continuations are carried independently.
func graphResponsePathOracle(p contractstatemir.Program) (map[uint32]graphResponsePathSite, uint32, []string) {
	bodies := [][]contractstatemir.Block{p.Blocks}
	entries := []contractstatemir.BlockID{p.Entry}
	helpers := map[string]int{}
	for i, h := range p.Helpers {
		helpers[h.Name] = i + 1
		bodies = append(bodies, h.Blocks)
		entries = append(entries, h.Entry)
	}
	var traps []string
	var op, term uint32
	first, terms := make([][]uint32, len(bodies)), make([][]uint32, len(bodies))
	for i, blocks := range bodies {
		for _, b := range blocks {
			term++
			t := uint32(1<<31) | term
			f := op + 1
			if len(b.Operations) == 0 {
				f = t
			}
			first[i], terms[i] = append(first[i], f), append(terms[i], t)
			op += uint32(len(b.Operations))
			if b.Terminator.Kind == contractstatemir.TerminatorTrap {
				traps = append(traps, b.Terminator.TrapCategory)
			}
		}
	}
	slices.Sort(traps)
	traps = slices.Compact(traps)
	sites := map[uint32]graphResponsePathSite{}
	op = 0
	for i, blocks := range bodies {
		index := 0
		for j, b := range blocks {
			for k, o := range b.Operations {
				op++
				n := graphResponsePathSite{id: op, next: op + 1, body: i, index: index, operation: true}
				index++
				if k == len(b.Operations)-1 {
					n.next = terms[i][j]
				}
				if o.Kind == contractstatemir.OperationCall {
					callee := helpers[o.Callee]
					n.callee = first[callee][entries[callee]]
				}
				sites[op] = n
			}
			n := graphResponsePathSite{id: terms[i][j], body: i, index: j}
			t := b.Terminator
			switch t.Kind {
			case contractstatemir.TerminatorJump:
				n.next = first[i][t.Target]
			case contractstatemir.TerminatorBranch:
				n.next, n.alternative = first[i][t.TrueTarget], first[i][t.FalseTarget]
			case contractstatemir.TerminatorCommit:
				n.exit = 1
			case contractstatemir.TerminatorReject:
				n.exit = 2
			case contractstatemir.TerminatorTrap:
				n.exit, n.trap = 3, slices.Index(traps, t.TrapCategory)+1
			case contractstatemir.TerminatorReturn:
				n.exit = 4
			}
			sites[n.id] = n
		}
	}
	return sites, first[0][entries[0]], traps
}

func TestCandidateStateGraphResponsePath(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native path probe; Linux execution is batched")
	}
	selected := map[string]bool{"empty-commit": true, "empty-reject": true, "trap": true, "helper-transitive": true, "helper-branch": true, "branch-join": true, "helper-count-32": true}
	var trials []stateGraphAuthoredTrial
	for _, trial := range graphTransportTrials() {
		if selected[trial.name] {
			trials = append(trials, trial)
		}
	}
	trials = append(trials, stateGraphAuthoredTrial{"sorted-traps", stateGraphAuthoredSource("storage_read", `route();storage::trap("a");`, `@effects(storage_read)fn void route(){let b=storage::read("flag");if(b){storage::trap("z-last");}else{storage::trap("a-first");}}`), ""})
	var probe strings.Builder
	probe.WriteString(graphResponsePathProbe)
	var totalPaths, totalSteps int
	for i, trial := range trials {
		f, code := stateGraphAuthoredOracle(t, trial)
		if code != "" {
			t.Fatal(trial.name, code)
		}
		lowered, err := contractstatemir.LowerHIRTransactions(f.input)
		if err != nil || len(lowered) != 1 {
			t.Fatal("lower current graph", err)
		}
		sites, start, traps := graphResponsePathOracle(lowered[0].Program)
		fmt.Fprintf(&probe, "fn void trial%d(){let plan=make_plan(b%s,b%s);\n", i, strconv.Quote(trial.text), strconv.Quote("fypc1-sha256-"+hex.EncodeToString(f.coordinate[:])))
		for j, trap := range traps {
			fmt.Fprintf(&probe, "assert(bytes::equal(path::trap_category(ref plan,%d),b%s));\n", j+1, strconv.Quote(trap))
		}
		fmt.Fprintf(&probe, "assert(path::trap_category(ref plan,0).len==0 && path::trap_category(ref plan,%d).len==0);\n", len(traps)+1)
		var visit func(uint32, []uint32, []graphResponsePathSite)
		visit = func(id uint32, stack []uint32, prefix []graphResponsePathSite) {
			n, ok := sites[id]
			if !ok || len(prefix) > 256 {
				t.Fatal("invalid path oracle", trial.name, id)
			}
			prefix = append(slices.Clone(prefix), n)
			if n.callee != 0 {
				visit(n.callee, append(slices.Clone(stack), n.next), prefix)
			} else if n.exit == 4 {
				if len(stack) == 0 {
					t.Fatal("return without call")
				}
				visit(stack[len(stack)-1], stack[:len(stack)-1], prefix)
			} else if n.exit != 0 {
				totalPaths++
				probe.WriteString("{var p=path::start(ref plan);\n")
				ops := 0
				for event, step := range prefix {
					if step.operation {
						ops++
					}
					var ids []uint32
					for id := range sites {
						ids = append(ids, id)
					}
					slices.Sort(ids)
					for _, id := range ids {
						if id == step.id {
							continue
						}
						if event > 0 {
							previous := prefix[event-1]
							if previous.alternative != 0 && (id == previous.next || id == previous.alternative) {
								continue
							}
						}
						fmt.Fprintf(&probe, "refuse(ref plan,ref mut p,%d,%d);\n", id, ops)
						break
					}
					fmt.Fprintf(&probe, "take(ref plan,ref mut p,%d,%d,%d,%d,%v);\n", step.id, ops, step.body, step.index, step.operation)
					totalSteps++
				}
				fmt.Fprintf(&probe, "finish(ref plan,ref mut p,%d,%d,%d,%d);}\n", n.exit, n.trap, len(prefix), ops)
			} else {
				visit(n.next, stack, prefix)
				if n.alternative != 0 && n.alternative != n.next {
					visit(n.alternative, stack, prefix)
				}
			}
		}
		visit(start, nil, nil)
		probe.WriteString("}\n")
	}
	probe.WriteString("@effects(console)fn i32 main(){")
	for i := range trials {
		fmt.Fprintf(&probe, "trial%d();", i)
	}
	probe.WriteString(`assert(io::write("ok\n").is_ok);return 0;}`)
	candidateAtlasProbeSize(t, "module_state_graph_response_path", probe.String(), 512<<10, func(t *testing.T, image string) {
		code, output := candidateAtlasProcess(t, image, nil)
		if code != 0 || output != "ok\n" {
			t.Fatalf("path probe exit=%d output=%q", code, output)
		}
		t.Logf("actual verified graphs=%d, CFG paths=%d, events=%d; exact/short/overdrawn ledgers", len(trials), totalPaths, totalSteps)
	}, "project_compiler_core", "module_public_type_text")
}

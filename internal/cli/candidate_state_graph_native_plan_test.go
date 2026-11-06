package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Independent graphSlots/encoding-json oracle, derived from the verified Go
// model rather than the candidate transport or its prepared slot maps.
func stateNativePlanOracle(p contractstatemir.Program) (string, int) {
	bodies := [][]contractstatemir.Block{p.Blocks}
	for _, h := range p.Helpers {
		bodies = append(bodies, h.Blocks)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "%d;", len(bodies))
	var traps []string
	total := 0
	for _, blocks := range bodies {
		values := map[contractstatemir.ValueID]int{}
		borrows := map[contractstatemir.BorrowID]int{}
		for _, b := range blocks {
			for _, v := range b.Parameters {
				values[v.ID] = len(values)
			}
			for _, op := range b.Operations {
				if op.Result != 0 {
					values[op.Result] = len(values)
				}
				if op.Found != 0 {
					values[op.Found] = len(values)
				}
				if op.BorrowResult != 0 {
					borrows[op.BorrowResult] = len(borrows)
				}
			}
			if b.Terminator.TrapCategory != "" {
				traps = append(traps, b.Terminator.TrapCategory)
			}
		}
		total += len(values) + len(borrows)
		fmt.Fprintf(&out, "%d,%d:", len(values), len(borrows))
		for _, b := range blocks {
			for _, v := range b.Parameters {
				fmt.Fprintf(&out, "v%d=%d,", v.ID, values[v.ID])
			}
			for _, op := range b.Operations {
				if op.Result != 0 {
					fmt.Fprintf(&out, "v%d=%d,", op.Result, values[op.Result])
				}
				if op.Found != 0 {
					fmt.Fprintf(&out, "v%d=%d,", op.Found, values[op.Found])
				}
				if op.BorrowResult != 0 {
					fmt.Fprintf(&out, "b%d=%d,", op.BorrowResult, borrows[op.BorrowResult])
				}
			}
		}
		out.WriteByte(';')
	}
	slices.Sort(traps)
	traps = slices.Compact(traps)
	encoded, _ := json.Marshal(traps)
	out.Write(encoded)
	out.WriteByte(';')
	for i, s := range traps {
		fmt.Fprintf(&out, "%s=%d,", s, i+1)
	}
	return out.String(), total
}

func stateNativePlanFixtures(t *testing.T) []struct {
	name string
	p    contractstatemir.Program
} {
	var trials []struct {
		name string
		p    contractstatemir.Program
	}
	add := func(name string, p contractstatemir.Program) {
		trials = append(trials, struct {
			name string
			p    contractstatemir.Program
		}{name, p})
	}
	add("branch-parameters-zero-based-fields", stateGraphIRBase(t))
	for _, shape := range []string{"empty", "diamond", "helper-branch", "transitive"} {
		add(shape, stateGraphHelperProgram(t, shape))
	}
	for _, mapped := range []bool{false, true} {
		add(fmt.Sprintf("borrow-%v", mapped), stateGraphBorrowBase(t, mapped))
	}
	sparse := stateGraphIRBase(t)
	remap := func(id contractstatemir.ValueID) contractstatemir.ValueID {
		if id == 0 {
			return 0
		}
		return ^contractstatemir.ValueID(0) - id + 1
	}
	for i := range sparse.Blocks {
		b := &sparse.Blocks[i]
		for j := range b.Parameters {
			b.Parameters[j].ID = remap(b.Parameters[j].ID)
		}
		for j := range b.Operations {
			o := &b.Operations[j]
			o.Result = remap(o.Result)
			o.Found = remap(o.Found)
			o.KeyInput = remap(o.KeyInput)
			o.ValueInput = remap(o.ValueInput)
		}
		b.Terminator.Condition = remap(b.Terminator.Condition)
		for _, args := range [][]contractstatemir.ValueID{b.Terminator.Arguments, b.Terminator.TrueArguments, b.Terminator.FalseArguments} {
			for j := range args {
				args[j] = remap(args[j])
			}
		}
	}
	add("sparse-u32-max", sparse)
	sparseBorrow := stateGraphBorrowBase(t, false)
	for i := range sparseBorrow.Blocks[0].Operations {
		o := &sparseBorrow.Blocks[0].Operations[i]
		if o.BorrowResult != 0 {
			o.BorrowResult = ^contractstatemir.BorrowID(0) - o.BorrowResult + 1
		}
		if o.BorrowInput != 0 {
			o.BorrowInput = ^contractstatemir.BorrowID(0) - o.BorrowInput + 1
		}
	}
	add("sparse-borrow-u32-max", sparseBorrow)
	p := stateGraphHelperProgram(t, "branch")
	p.Helpers[0].Blocks[0].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: "z-last", Cost: 1}
	p.Helpers[1].Blocks[0].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: "a-first", Cost: 1}
	p.Blocks[1].Terminator = p.Helpers[0].Blocks[0].Terminator
	p.Blocks[2].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: "a", Cost: 1}
	add("traps-sort-deduplicate-root-helpers", p)
	p = stateGraphHelperProgram(t, "helper-branch")
	p.Helpers[0].Blocks[1].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: strings.Repeat("a", 127), Cost: 1}
	p.Helpers[0].Blocks[2].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: "a-0", Cost: 1}
	add("trap-prefix-and-max-length", p)
	// A single read can feed arbitrary distinct block parameters. This admits
	// all three slot boundaries without exceeding the 1024-operation ceiling.
	for _, n := range []int{4095, 4096, 4097} {
		p := stateGraphIRBase(t)
		p.Effects = []hir.Effect{hir.EffectStorageRead}
		read := p.Blocks[0].Operations[0]
		args := make([]contractstatemir.ValueID, n-2)
		params := make([]contractstatemir.BlockParameter, n-2)
		for i := range args {
			args[i] = read.Result
			params[i] = contractstatemir.BlockParameter{ID: contractstatemir.ValueID(i + 3), TypeID: contractstate.Bool.TypeID()}
		}
		p.Blocks = []contractstatemir.Block{{Operations: []contractstatemir.Operation{read}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorJump, Target: 1, Arguments: args, Cost: 1}}, {ID: 1, Parameters: params, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}}}
		add(fmt.Sprintf("slots-%d", n), p)
		p.Blocks = slices.Clone(p.Blocks)
		// Split the combined bound across root and helper, including independent
		// value/borrow slots carrying the same numeric ID in the root.
		rootSlots := n - 2048 - 2
		helper := stateGraphIRBase(t)
		helper.Blocks = []contractstatemir.Block{{Operations: []contractstatemir.Operation{read}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorJump, Target: 1, Cost: 1}}, {ID: 1, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorReturn, Cost: 1}}}
		for i := 0; i < 2046; i++ {
			helper.Blocks[0].Terminator.Arguments = append(helper.Blocks[0].Terminator.Arguments, read.Result)
			helper.Blocks[1].Parameters = append(helper.Blocks[1].Parameters, contractstatemir.BlockParameter{ID: contractstatemir.ValueID(i + 3), TypeID: contractstate.Bool.TypeID()})
		}
		p.Blocks[0].Terminator.Arguments = p.Blocks[0].Terminator.Arguments[:rootSlots-2]
		p.Blocks[1].Parameters = p.Blocks[1].Parameters[:rootSlots-2]
		p.Blocks[0].Operations = append(p.Blocks[0].Operations, contractstatemir.Operation{Kind: contractstatemir.OperationCellBorrow, Capability: p.Capability, FieldIndex: 5, Found: ^contractstatemir.ValueID(0), BorrowResult: ^contractstatemir.BorrowID(0), Cost: 1}, contractstatemir.Operation{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: "main::a", Cost: 1})
		p.Helpers = []contractstatemir.Helper{{Name: "main::a", Effects: p.Effects, Blocks: helper.Blocks}}
		add(fmt.Sprintf("combined-value-borrow-helper-slots-%d", n), p)
	}
	return trials
}

func stateNativePlanProbe() string {
	probe := stateGraphHelperProbe()
	probe = probe[:strings.Index(probe, "@effects(console) fn i32 main()")]
	probe = strings.Replace(probe, "import module_state_graph_program as program;", "import module_state_graph_native_plan as native;import module_state_graph_program as program;", 1)
	return probe + stateNativePlanMain
}

const stateNativePlanMain = `
fn Result<native::Plan,u32> detached(ref u8[] raw){
 var usize work=0;let g=switch(read_program(raw,ref mut work)){case .ok(p):{p}case .err(c):{return .err(c);}};
 let layout=switch(layout()){case .ok(p):{p}case .err(c):{return .err(c);}};work=0;
 let proof=switch(program::verify(g,ref layout,ref mut work)){case .ok(p):{p}case .err(c):{return .err(c);}};
 work=0;let plan=switch(native::build(ref proof,ref mut work)){case .ok(p):{p}case .err(c):{return .err(c);}};let cost=work;if(cost==0 || cost>16777216){return .err(91);}
 work=16777216-cost;let exact=switch(native::build(ref proof,ref mut work)){case .ok(p):{p}case .err(_):{return .err(92);}};if(work!=16777216 || !bytes::equal(native::trap_json(ref plan),native::trap_json(ref exact))){return .err(93);}
 work=16777216-cost+1;switch(native::build(ref proof,ref mut work)){case .ok(_):{return .err(94);}case .err(c):{if(c!=1002 || work>16777216){return .err(95);}}};
 work=16777217;switch(native::build(ref proof,ref mut work)){case .ok(_):{return .err(96);}case .err(c):{if(c!=1002 || work!=16777217){return .err(97);}}};return .ok(plan);
}
fn Result<usize,u32> query(ref native::Plan p,usize b,usize id,bool borrow,ref mut usize work){return if(borrow){native::borrow_slot(p,b,id,work)}else{native::value_slot(p,b,id,work)};}
@effects(console) fn void show(ref native::Plan p,usize b,usize id,bool borrow){
 var usize work=0;let result=switch(query(p,b,id,borrow,ref mut work)){case .ok(s):{s}case .err(_):{assert(false);0}};let cost=work;
 work=16777216-cost;switch(query(p,b,id,borrow,ref mut work)){case .ok(s):{assert(s==result && work==16777216);}case .err(_):{assert(false);}};
 work=16777216-cost+1;switch(query(p,b,id,borrow,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==1002 && work<=16777216);}};
 discard io::write(if(borrow){"b"}else{"v"});discard io::write_usize(id);discard io::write("=");discard io::write_usize(result);discard io::write(",");
}
@effects(console) fn void show_body(ref native::Plan p,ref m::Graph g,usize b){
 discard io::write_usize(native::value_count(p,b));discard io::write(",");discard io::write_usize(native::borrow_count(p,b));discard io::write(":");var usize i=0;
 while(i<v::len<m::Block>(m::blocks(g))){let block=*v::at<m::Block>(m::blocks(g),i);var usize j=0;
  while(j<block.parameter_count){show(p,b,v::at<m::Value>(m::parameters(g),block.parameter_start+j).id,false);j+=1;}j=0;
  while(j<block.operation_count){let o=*v::at<m::Operation>(m::operations(g),block.operation_start+j);if(o.result!=0){show(p,b,o.result,false);}if(o.found!=0){show(p,b,o.found,false);}if(o.borrow_result!=0){show(p,b,o.borrow_result,true);}j+=1;}i+=1;}discard io::write(";");
}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>2097152-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let plan=switch(detached(bytes::view(ref input))){case .ok(p):{p}case .err(c):{discard io::write_u32(c);return 0;}};
 var usize work=0;let g=switch(read_program(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(_):{return 3;}};input=bytes::empty();let layouts=switch(layout()){case .ok(p):{p}case .err(_):{return 19;}};work=0;let owner=switch(program::verify(g,ref layouts,ref mut work)){case .ok(p):{p}case .err(_):{return 20;}};
 work=0;switch(native::value_slot(ref plan,native::body_count(ref plan),1,ref mut work)){case .ok(_):{return 4;}case .err(c):{if(c!=1001){return 5;}}};
 work=0;switch(native::borrow_slot(ref plan,0,0,ref mut work)){case .ok(_):{return 6;}case .err(c):{if(c!=1001){return 7;}}};
 work=0;switch(native::value_slot(ref plan,0,0,ref mut work)){case .ok(_):{return 8;}case .err(c):{if(c!=1001){return 9;}}};
 work=0;switch(native::trap_id(ref plan,b"",ref mut work)){case .ok(id):{if(id!=0){return 10;}}case .err(_):{return 11;}};
 work=0;switch(native::trap_id(ref plan,b"unknown-not-in-plan",ref mut work)){case .ok(_):{return 12;}case .err(c):{if(c!=1001){return 13;}}};
 discard io::write_usize(native::body_count(ref plan));discard io::write(";");show_body(ref plan,program::root(ref owner),0);var usize i=0;
 while(i<program::helper_count(ref owner)){show_body(ref plan,program::helper(ref owner,i),i+1);i+=1;}
 discard io::write_bytes(native::trap_json(ref plan));discard io::write(";");i=0;
 while(i<native::trap_count(ref plan)){let name=native::trap(ref plan,i);work=0;let id=switch(native::trap_id(ref plan,name,ref mut work)){case .ok(id):{id}case .err(_):{return 14;}};let cost=work;
  work=16777216-cost;switch(native::trap_id(ref plan,name,ref mut work)){case .ok(n):{if(n!=id || work!=16777216){return 15;}}case .err(_):{return 16;}};
  work=16777216-cost+1;switch(native::trap_id(ref plan,name,ref mut work)){case .ok(_):{return 17;}case .err(c):{if(c!=1002 || work>16777216){return 18;}}};
  discard io::write_bytes(name);discard io::write("=");discard io::write_usize(id);discard io::write(",");i+=1;}return 0;
}`

func TestCandidateStateGraphNativePlan(t *testing.T) {
	trials := stateNativePlanFixtures(t)
	for _, trial := range trials {
		if failures := contractstatemir.Verify(trial.p); len(failures) != 0 {
			t.Fatalf("fixture %s invalid: %v", trial.name, failures)
		}
	}
	candidateAtlasProbeSize(t, "module_state_graph_native_plan", stateNativePlanProbe(), 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				want, total := stateNativePlanOracle(trial.p)
				if total > 4096 {
					want = "1002"
				}
				code, got := candidateAtlasProcess(t, image, stateGraphHelperEncode(trial.p).bytes())
				if code != 0 || got != want {
					t.Fatalf("slots=%d exit=%d got=%q want=%q", total, code, got, want)
				}
			})
		}
	}, "module_state_declarations", "candidate_artifact_wire")
}

func TestCandidateStateGraphNativePlanSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"private-constructor", "E0103", `fn native::Plan forge(){return native::Plan{.bodies=0,.traps=0,.data=0,.json=0};}`},
		{"private-slots", "E0103", `fn void change(ref mut native::Plan p){p.bodies=0;}`},
		{"immutable-json", "E1008", `fn void change(ref native::Plan p){let ref mut u8[] raw=ref mut native::trap_json(p);}`},
		{"immutable-trap", "E1008", `fn void change(ref native::Plan p){let ref mut u8[] raw=ref mut native::trap(p,0);}`},
		{"owner-lifetime", "E2102", `fn ref u8[] escape(native::Plan p){return native::trap_json(ref p);}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_native_plan")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_native_plan as native;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed native plan", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), trial.code) {
				t.Fatalf("sealed-owner escape: %d %s", code, &diagnostics)
			}
		})
	}
}

package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type stateTransportPlanTrial struct {
	name, source string
	p            contractstatemir.Program
}

// Independent Program oracle: evaluate ready bodies bottom-up with arbitrary
// precision integers, rather than replaying the candidate's fixed-stack DFS.
// The Go graph_bounds.go contract clamps after EACH operation, not at body end.
func stateTransportPlanOracle(t *testing.T, p contractstatemir.Program) (string, uint64, int) {
	t.Helper()
	type body struct {
		ops    []contractstatemir.Operation
		blocks []contractstatemir.Block
	}
	bodies := map[string]body{"": {p.Operations, p.Blocks}}
	for _, h := range p.Helpers {
		bodies[h.Name] = body{h.Operations, h.Blocks}
	}
	// Only helpers reachable from the root contribute literal lengths.
	reached := map[string]bool{"": true}
	queue := []string{""}
	groups := func(b body) []contractstatemir.Block {
		if len(b.blocks) == 0 {
			return []contractstatemir.Block{{Operations: b.ops}}
		}
		return b.blocks
	}
	for i := 0; i < len(queue); i++ {
		for _, block := range groups(bodies[queue[i]]) {
			for _, op := range block.Operations {
				if op.Kind == contractstatemir.OperationCall && !reached[op.Callee] {
					if _, ok := bodies[op.Callee]; !ok {
						t.Fatal("oracle missing callee", op.Callee)
					}
					reached[op.Callee] = true
					queue = append(queue, op.Callee)
				}
			}
		}
	}
	known := map[string]*big.Int{}
	maximum := big.NewInt(67108864)
	literal := 0
	for len(known) < len(reached) {
		progress := false
		for _, name := range queue {
			if known[name] != nil {
				continue
			}
			b := bodies[name]
			n := big.NewInt(int64(max(1, len(b.blocks))))
			ready := true
			for _, block := range groups(b) {
				for _, op := range block.Operations {
					literal = max(literal, len(op.Key.Encoded), len(op.Value.Encoded))
					n.Add(n, big.NewInt(1))
					if op.Kind == contractstatemir.OperationCall {
						child := known[op.Callee]
						if child == nil {
							ready = false
							break
						}
						n.Add(n, child)
					}
					if n.Cmp(maximum) > 0 {
						n.Set(maximum)
					}
				}
				if !ready {
					break
				}
			}
			if ready {
				known[name] = n
				progress = true
			}
		}
		if !progress {
			t.Fatal("oracle cycle")
		}
	}
	codecs := map[string]int{"bool/v1": 1, "u32/be/v1": 2, "u64/be/v1": 3, "bytes/v1": 4, "utf8/v1": 5, "bytes20/v1": 6, "bytes32/v1": 7}
	cells := map[contractstate.Key]int{}
	for _, f := range p.Layout.Fields {
		if f.Kind == contractstate.FieldCell {
			codec, ok := codecs[f.ValueTypeID]
			if !ok {
				t.Fatal("oracle unknown cell codec", f.ValueTypeID)
			}
			cells[f.Prefix] = codec
		}
	}
	var keys []contractstate.Key
	for key := range cells {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b contractstate.Key) int { return bytes.Compare(a[:], b[:]) })
	var out strings.Builder
	bound := known[""].Uint64()
	fmt.Fprintf(&out, "%d;%d;%d;", bound, literal, len(keys))
	for _, key := range keys {
		for i := 0; i < 32; i += 4 {
			fmt.Fprintf(&out, "%d,", binary.BigEndian.Uint32(key[i:]))
		}
		fmt.Fprintf(&out, "=%d;", cells[key])
	}
	return out.String(), bound, literal
}

func stateTransportPlanTrials(t *testing.T) []stateTransportPlanTrial {
	t.Helper()
	var trials []stateTransportPlanTrial
	add := func(name string, p contractstatemir.Program) {
		trials = append(trials, stateTransportPlanTrial{name, stateGraphIRSource, p})
	}
	for _, shape := range []string{"empty", "no-helpers", "leaf", "transitive", "diamond", "branch", "helper-branch"} {
		add(shape, stateGraphHelperProgram(t, shape))
	}
	p := stateGraphHelperProgram(t, "leaf")
	p.Blocks[0].Operations = slices.Repeat(p.Blocks[0].Operations, 5)
	add("repeated-helper-five", p)
	p = stateGraphHelperProgram(t, "diamond")
	p.Blocks[0].Operations = p.Blocks[0].Operations[1:]
	p.Helpers[1].Blocks[0].Operations[0].Callee = "main::a"
	add("backward-lexical-helper", p)
	add("branch-ssa-and-key-literals", stateGraphIRBase(t))
	for _, fixture := range stateNativePlanFixtures(t) {
		if strings.HasPrefix(fixture.name, "sparse-") {
			add(fixture.name, fixture.p)
		}
	}
	for _, mapped := range []bool{false, true} {
		add(fmt.Sprintf("borrow-%v", mapped), stateGraphBorrowBase(t, mapped))
	}
	p = stateGraphHelperProgram(t, "no-helpers")
	p.Effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}
	p.Blocks[0].Operations = []contractstatemir.Operation{
		{Kind: contractstatemir.OperationCellRead, Capability: p.Capability, FieldIndex: 6, Result: ^contractstatemir.ValueID(0), Found: ^contractstatemir.ValueID(0) - 1, Cost: 1},
		{Kind: contractstatemir.OperationCellWrite, Capability: p.Capability, FieldIndex: 6, ValueInput: ^contractstatemir.ValueID(0), Cost: 1},
	}
	p.Blocks[0].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: strings.Repeat("z", 127), Cost: 1}
	add("ssa-and-trap-are-not-literals", p)
	for _, keyLarger := range []bool{false, true} {
		p = stateGraphHelperProgram(t, "leaf")
		keySize, valueSize := 3, 255
		if keyLarger {
			keySize, valueSize = valueSize, keySize
		}
		key, err := contractstate.Text.Encode(strings.Repeat("k", keySize))
		if err != nil {
			t.Fatal(err)
		}
		value, err := contractstate.Bytes.Encode(bytes.Repeat([]byte{255}, valueSize))
		if err != nil {
			t.Fatal(err)
		}
		p.Helpers[0].Blocks[0].Operations = []contractstatemir.Operation{{Kind: contractstatemir.OperationMapWrite, Capability: p.Capability, FieldIndex: 8, Key: contractstatemir.Value{TypeID: contractstate.Text.TypeID(), Encoded: key}, Value: contractstatemir.Value{TypeID: contractstate.Bytes.TypeID(), Encoded: value}, Cost: 1}}
		add(fmt.Sprintf("helper-encoded-span-key-larger-%v", keyLarger), p)
	}
	// 33 bodies but billions of expanded calls. The candidate must never walk
	// the expanded tree or size arrays by sparse IDs or expanded costs.
	for _, copies := range []int{1, 2} {
		p = stateGraphHelperProgram(t, "leaf")
		write := p.Helpers[0].Blocks[0].Operations[0]
		p.Helpers = nil
		for i := 0; i < 32; i++ {
			ops := []contractstatemir.Operation{write}
			if i < 31 {
				ops = slices.Repeat([]contractstatemir.Operation{{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: fmt.Sprintf("main::h%02d", i+1), Cost: 1}}, copies)
			}
			p.Helpers = append(p.Helpers, contractstatemir.Helper{Name: fmt.Sprintf("main::h%02d", i), Effects: p.Effects, Blocks: []contractstatemir.Block{{Operations: ops, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorReturn, Cost: 1}}}})
		}
		p.Blocks[0].Operations[0].Callee = "main::h00"
		// A larger literal AFTER saturation must still be measured, and a later
		// operation must not wrap or reset the already saturated work bound.
		later := write
		if copies == 2 {
			encoded, err := contractstate.Bytes.Encode(bytes.Repeat([]byte{255}, 64))
			if err != nil {
				t.Fatal(err)
			}
			later.FieldIndex = 5
			later.Value = contractstatemir.Value{TypeID: contractstate.Bytes.TypeID(), Encoded: encoded}
		}
		p.Blocks[0].Operations = append(p.Blocks[0].Operations, later)
		add(fmt.Sprintf("deep-32-helpers-copies-%d", copies), p)
	}
	// Empty and map-only root layouts still exclude the decoy Cell layout.
	for _, fields := range []int{0, 1, 1024} {
		var declarations []contractstatemir.Declaration
		var text strings.Builder
		text.WriteString(`module main;@contract_uid("fixture.transport-fields")state{`)
		for i := 0; i < fields; i++ {
			kind, syntax := contractstate.FieldCell, "cell u32"
			key := ""
			if fields == 1 {
				kind, syntax, key = contractstate.FieldMap, "map u32 u32", contractstate.U32.TypeID()
			}
			name := fmt.Sprintf("f%04d", i)
			declarations = append(declarations, contractstatemir.Declaration{Name: name, Kind: kind, KeyTypeID: key, ValueTypeID: contractstate.U32.TypeID()})
			fmt.Fprintf(&text, "%s %s;", syntax, name)
		}
		text.WriteByte('}')
		layout, err := contractstatemir.BuildLayout("fixture.transport-fields", declarations)
		if err != nil {
			t.Fatal(err)
		}
		p = stateGraphHelperProgram(t, "no-helpers")
		p.Layout, p.Effects, p.Blocks[0].Operations = layout, nil, nil
		trials = append(trials, stateTransportPlanTrial{fmt.Sprintf("root-fields-%d", fields), text.String(), p})
	}
	return trials
}

func TestCandidateStateGraphTransportPlanOracle(t *testing.T) {
	for _, trial := range stateTransportPlanTrials(t) {
		t.Run(trial.name, func(t *testing.T) {
			if failures := contractstatemir.Verify(trial.p); len(failures) != 0 {
				t.Fatal("invalid independent Program fixture", failures)
			}
			_, bound, literal := stateTransportPlanOracle(t, trial.p)
			want := map[string]uint64{"empty": 3, "no-helpers": 2, "leaf": 4, "transitive": 6, "diamond": 11, "branch": 10, "helper-branch": 12, "repeated-helper-five": 16, "deep-32-helpers-copies-1": 67, "deep-32-helpers-copies-2": 67108864}
			if n, ok := want[trial.name]; ok && bound != n {
				t.Fatalf("bound=%d want independent arithmetic=%d", bound, n)
			}
			if trial.name == "ssa-and-trap-are-not-literals" && literal != 0 {
				t.Fatalf("SSA/trap counted as literal: %d", literal)
			}
			if trial.name == "deep-32-helpers-copies-2" && literal != 64 {
				t.Fatalf("encoded literal after saturation=%d want=64", literal)
			}
		})
	}
	// Preserve the min-one and reached-only rules even though Verified graphs
	// currently require explicit blocks and reject unreachable helpers.
	p := stateGraphHelperProgram(t, "no-helpers")
	p.Blocks, p.Operations = nil, nil
	p.Helpers = []contractstatemir.Helper{{Name: "unreached", Operations: []contractstatemir.Operation{{Value: contractstatemir.Value{Encoded: make([]byte, 99)}}}}}
	_, bound, literal := stateTransportPlanOracle(t, p)
	if bound != 1 || literal != 0 {
		t.Fatalf("empty legacy/reached-only oracle: bound=%d literal=%d", bound, literal)
	}
}

func stateTransportPlanProbe() string {
	probe := stateGraphHelperProbe()
	probe = probe[:strings.Index(probe, "@effects(console) fn i32 main()")]
	probe = strings.Replace(probe, "import module_state_graph_program as program;", "import module_state_graph_transport_plan as transport;import module_state_graph_program as program;", 1)
	start := strings.Index(probe, "fn Result<state::Plan,u32> layout(){")
	end := strings.Index(probe, "fn Result<m::Graph,u32> read(")
	return probe[:start] + stateTransportPlanLayout + probe[end:] + stateTransportPlanMain
}

// The decoy is a separate package's layout before the root. This checks both
// selection by root layout and a nonzero global field offset on every fixture.
const stateTransportPlanLayout = `
fn Result<syntax::Parsed,u32> parsed_layout(ref u8[] raw){
 let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(1003);}};
 let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return .err(1001);}};
 let parsed=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{return .err(1001);}};
 if(syntax::problem_count(ref parsed)!=0){return .err(1001);}return .ok(parsed);
}
fn Result<state::Plan,u32> layout(ref u8[] raw){
 let decoy=switch(parsed_layout(b"module decoy;@contract_uid(\"fixture.transport-decoy\")state{cell text unrelated;}")){case .ok(p):{p}case .err(c):{return .err(c);}};
 let root=switch(parsed_layout(raw)){case .ok(p):{p}case .err(c):{return .err(c);}};
 var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,decoy).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,root).is_ok);
 var owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok);assert(v::push<usize>(ref mut owners,1).is_ok);let bindings=v::empty<catalog::Binding>();
 let c=switch(catalog::build_scoped(ref sources,ref owners,ref bindings)){case .ok(p):{p}case .err(_):{return .err(1001);}};
 return switch(declarations::build_graph(ref sources,ref c)){case .ok(p):{.ok(p)}case .err(e):{.err(if(e.code==10){1002}else{1001})}};
}
`

const stateTransportPlanMain = `
fn bool equal(ref transport::Plan a,ref transport::Plan b){
 if(transport::operation_bound(a)!=transport::operation_bound(b) || transport::max_literal(a)!=transport::max_literal(b) || transport::cell_count(a)!=transport::cell_count(b)){return false;}
 var usize i=0;while(i<transport::cell_count(a)){let x=transport::cell_at(a,i);let y=transport::cell_at(b,i);if(x.codec!=y.codec || !state::equal(x.key,y.key)){return false;}i+=1;}return true;
}
fn Result<transport::Plan,u32> detached(ref u8[] source,ref u8[] raw){
 var usize work=0;let g=switch(read_program(raw,ref mut work)){case .ok(p):{p}case .err(c):{return .err(c);}};
 var layouts=switch(layout(source)){case .ok(p):{p}case .err(c):{return .err(c);}};work=0;
 let proof=switch(program::verify(g,ref layouts,ref mut work)){case .ok(p):{p}case .err(c):{return .err(c);}};
 layouts=state::empty();if(state::count(program::layouts(ref proof))!=2){return .err(90);}
 work=0;let plan=switch(transport::build(ref proof,ref mut work)){case .ok(p):{p}case .err(c):{return .err(c);}};let cost=work;
 if(cost==0 || cost>16777216){return .err(91);}
 work=16777216-cost;let exact=switch(transport::build(ref proof,ref mut work)){case .ok(p):{p}case .err(_):{return .err(92);}};
 if(work!=16777216 || !equal(ref plan,ref exact)){return .err(93);}
 work=16777217-cost;switch(transport::build(ref proof,ref mut work)){case .ok(_):{return .err(94);}case .err(c):{if(c!=1002 || work>16777216){return .err(95);}}};
 work=16777217;switch(transport::build(ref proof,ref mut work)){case .ok(_):{return .err(96);}case .err(c):{if(c!=1002 || work!=16777217){return .err(97);}}};
 // Failure does not consume or mutate either caller-owned sealed input.
 work=0;if(program::validate(program::program(ref proof),program::layouts(ref proof),ref mut work)!=0){return .err(98);}
 work=0;let again=switch(transport::build(ref proof,ref mut work)){case .ok(p):{p}case .err(_):{return .err(99);}};
 if(work!=cost || !equal(ref plan,ref again)){return .err(89);}return .ok(plan);
}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>2097152-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 var usize work=0;let raw=bytes::view(ref input);var c=wire::Cursor{.offset=0,.code=0};let size=number(raw,ref mut c,ref mut work);if(c.code!=0 || !m::range(c.offset,size,raw.len)){return 3;}
 let plan=switch(detached(raw[c.offset..c.offset+size],raw[c.offset+size..])){case .ok(p):{p}case .err(c):{discard io::write_u32(c);return 0;}};
 // The proof, graph, layouts and all original bytes have now been dropped.
 {let changed=bytes::view_mut(ref mut input);changed[0]=255;}input=bytes::empty();
 discard io::write_usize(transport::operation_bound(ref plan));discard io::write(";");discard io::write_usize(transport::max_literal(ref plan));discard io::write(";");discard io::write_usize(transport::cell_count(ref plan));discard io::write(";");var usize i=0;
 while(i<transport::cell_count(ref plan)){
  let original=transport::cell_at(ref plan,i);var copied=original;copied.codec=999;let first=copied.key[0];copied.key[0]=first^1;
  let cell=transport::cell_at(ref plan,i);if(cell.codec!=original.codec || !state::equal(cell.key,original.key)){return 4;}
  var usize word=0;while(word<8){discard io::write_u32(cell.key[word]);discard io::write(",");word+=1;}
  discard io::write("=");discard io::write_usize(cell.codec);discard io::write(";");i+=1;
 }return 0;
}`

func TestCandidateStateGraphTransportPlan(t *testing.T) {
	trials := stateTransportPlanTrials(t)
	for _, trial := range trials {
		if failures := contractstatemir.Verify(trial.p); len(failures) != 0 {
			t.Fatalf("invalid fixture %s: %v", trial.name, failures)
		}
	}
	candidateAtlasProbeSize(t, "module_state_graph_transport_plan", stateTransportPlanProbe(), 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				want, _, _ := stateTransportPlanOracle(t, trial.p)
				packet := stateGraphHelperEncode(trial.p)
				for _, g := range packet.graphs {
					stateGraphHelperWord(g, 2, 1)
				}
				input := binary.BigEndian.AppendUint64(nil, uint64(len(trial.source)))
				input = append(input, trial.source...)
				input = append(input, packet.bytes()...)
				code, got := candidateAtlasProcess(t, image, input)
				if code != 0 || got != want {
					t.Fatalf("exit=%d got=%q want=%q", code, got, want)
				}
			})
		}
	}, "module_state_declarations", "candidate_artifact_wire")
}

func TestCandidateStateGraphTransportPlanSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"private-constructor", "E0103", `fn transport::Plan forge(){return transport::Plan{.operations=1,.literal=0,.cells=0};}`},
		{"private-cells", "E0103", `fn void change(ref mut transport::Plan p){p.cells=0;}`},
		{"private-bound", "E0103", `fn void change(ref mut transport::Plan p){p.operations=0;}`},
		{"private-literal", "E0103", `fn void change(ref mut transport::Plan p){p.literal=0;}`},
		{"verified-input-required", "E1006", `fn void forge(ref program::Program p){var usize work=0;discard transport::build(p,ref mut work);}`},
		{"owner-lifetime", "E2102", `fn ref transport::Plan escape(transport::Plan p){return ref p;}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_transport_plan")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_transport_plan as transport;import module_state_graph_program as program;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed transport plan", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), trial.code) {
				t.Fatalf("sealed-owner escape: %d %s", code, &diagnostics)
			}
		})
	}
}

package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
)

func stateNativeOperationsProbe() string {
	probe := stateGraphHelperProbe()
	probe = probe[:strings.Index(probe, "@effects(console) fn i32 main()")]
	probe = strings.Replace(probe, "module main;", "module main;import module_state_graph_native_operations as emit;import module_state_graph_native_plan as plan;import module_state_graph_native_text as w;", 1)
	return probe + `
fn ref m::Graph body_graph(ref program::Verified proof,usize body){if(body==0){return program::root(proof);}return program::helper(proof,body-1);}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(!bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 var usize work=0;let g=switch(read_program(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(_):{return 3;}};
 let layouts=switch(layout()){case .ok(p):{p}case .err(_):{return 4;}};
 work=0;let proof=switch(program::verify(g,ref layouts,ref mut work)){case .ok(p):{p}case .err(_):{return 5;}};
 work=0;let p=switch(plan::build(ref proof,ref mut work)){case .ok(p):{p}case .err(_):{return 6;}};
 var usize mode=0;while(mode<2){var output=bytes::empty();var s=w::State{.code=0,.work=0,.site=0,.term=0,.audit=mode==1};var usize body=0;var usize count=0;
  while(body<=program::helper_count(ref proof)){let graph=body_graph(ref proof,body);var usize i=0;while(i<v::len<m::Operation>(m::operations(graph))){let op=*v::at<m::Operation>(m::operations(graph),i);
   if(!emit::operation(ref proof,ref p,body,graph,op,ref mut output,ref mut s) || s.code!=0){return 7;}count+=1;if(s.site!=count){return 8;}i+=1;}body+=1;}
  if(!io::write_bytes(bytes::view(ref output)).is_ok || !io::write("\n===AUDIT===\n").is_ok){return 9;}mode+=1;
 }
 // Exact budget succeeds; one less fails and every later call stays inert.
 let root=program::root(ref proof);if(v::len<m::Operation>(m::operations(root))!=0){let op=*v::at<m::Operation>(m::operations(root),0);var output=bytes::empty();var s=w::State{.code=0,.work=0,.site=0,.term=0,.audit=true};
  if(!emit::operation(ref proof,ref p,0,root,op,ref mut output,ref mut s)){return 10;}let cost=s.work;let size=bytes::len(ref output);
  output=bytes::empty();s=w::State{.code=0,.work=16777216-cost,.site=0,.term=0,.audit=true};if(!emit::operation(ref proof,ref p,0,root,op,ref mut output,ref mut s) || s.work!=16777216 || bytes::len(ref output)!=size){return 11;}
  output=bytes::empty();s=w::State{.code=0,.work=16777217-cost,.site=0,.term=0,.audit=true};if(emit::operation(ref proof,ref p,0,root,op,ref mut output,ref mut s) || s.code!=1002 || s.work>16777216){return 12;}
  let before=bytes::len(ref output);let site=s.site;let used=s.work;if(emit::operation(ref proof,ref p,0,root,op,ref mut output,ref mut s) || s.code!=1002 || s.site!=site || s.work!=used || bytes::len(ref output)!=before){return 13;}
 }
 return 0;
}`
}

func TestCandidateStateGraphNativeOperations(t *testing.T) {
	type trial struct {
		name string
		p    contractstatemir.Program
		want []string
	}
	trials := []trial{
		{"flow", stateGraphIRBase(t), []string{"fycs_graph_read_v1", "fycs_graph_map_key_v1(prefix,&literal_key,max_key,key)", "fycs_graph_write_v1(context,key,&values["}},
		{"cell-borrow", stateGraphBorrowBase(t, false), []string{"fycs_graph_borrow_v1_begin(context,key,4u,&borrows[0],&found)", "fycs_graph_borrow_view_v1(context,&borrows[0],1u,3u,&borrows[1])", "fycs_graph_borrow_copy_v1(context,&borrows[1],&values[1])"}},
		{"map-borrow", stateGraphBorrowBase(t, true), []string{"literal_key_bytes[5] = {0x61, 0x6c, 0x70, 0x68, 0x61}", "literal_key = {literal_key_bytes,5u,5u}", "fycs_graph_borrow_v1_begin(context,key,4u,&borrows[0],&found)"}},
		{"helpers", stateGraphHelperProgram(t, "diamond"), []string{"fygr_body_1(", "fygr_body_2(", "fygr_body_3(", "if (*exit_kind != 3u) { goto finished; }"}},
	}
	all := stateGraphIRBase(t)
	all.Blocks = []contractstatemir.Block{{Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}}}
	for i, raw := range [][]byte{{1}, {0, 0, 0, 7}, {0, 0, 0, 0, 0, 0, 0, 8}, make([]byte, 20), make([]byte, 32), {}, {}} {
		all.Blocks[0].Operations = append(all.Blocks[0].Operations,
			contractstatemir.Operation{Kind: contractstatemir.OperationCellWrite, Capability: all.Capability, FieldIndex: uint32(i), Value: contractstatemir.Value{TypeID: stateGraphIRType(uint64(i + 1)), Encoded: raw}, Cost: 1},
			contractstatemir.Operation{Kind: contractstatemir.OperationCellRead, Capability: all.Capability, FieldIndex: uint32(i), Result: contractstatemir.ValueID(i*2 + 1), Found: contractstatemir.ValueID(i*2 + 2), Cost: 1},
			contractstatemir.Operation{Kind: contractstatemir.OperationCellDelete, Capability: all.Capability, FieldIndex: uint32(i), Cost: 1})
	}
	all.Blocks[0].Operations = append(all.Blocks[0].Operations,
		contractstatemir.Operation{Kind: contractstatemir.OperationMapWrite, Capability: all.Capability, FieldIndex: 8, Key: contractstatemir.Value{TypeID: contractstate.Text.TypeID(), Encoded: []byte{}}, Value: contractstatemir.Value{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte{}}, Cost: 1},
		contractstatemir.Operation{Kind: contractstatemir.OperationMapRead, Capability: all.Capability, FieldIndex: 8, Key: contractstatemir.Value{TypeID: contractstate.Text.TypeID(), Encoded: []byte{}}, Result: 99, Found: 100, Cost: 1},
		contractstatemir.Operation{Kind: contractstatemir.OperationMapDelete, Capability: all.Capability, FieldIndex: 8, Key: contractstatemir.Value{TypeID: contractstate.Text.TypeID(), Encoded: []byte{}}, Cost: 1},
		contractstatemir.Operation{Kind: contractstatemir.OperationMapDelete, Capability: all.Capability, FieldIndex: 8, KeyInput: 13, Cost: 1})
	trials = append(trials, trial{"codecs-empty-dynamic-key", all, []string{"literal_value_bytes[1] = {0x00}", "literal_value = {literal_value_bytes,0u,4u}", "literal_value = {literal_value_bytes,0u,5u}", "literal_key = {literal_key_bytes,0u,5u}", "fycs_graph_map_key_v1(prefix,&values[12],max_key,key)", "fycs_graph_write_v1(context,key,NULL,1u)"}})
	for _, trial := range trials {
		if failures := contractstatemir.Verify(trial.p); len(failures) != 0 {
			t.Fatalf("fixture %s: %v", trial.name, failures)
		}
	}
	candidateAtlasProbeSize(t, "module_state_graph_native_operations", stateNativeOperationsProbe(), 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, stateGraphHelperEncode(trial.p).bytes())
				if code != 0 {
					t.Fatalf("exit=%d output=%q", code, output)
				}
				parts := strings.Split(output, "\n===AUDIT===\n")
				if len(parts) != 3 || parts[2] != "" {
					t.Fatalf("missing modes: %q", output)
				}
				count, reads, calls := 0, 0, 0
				bodies := [][]contractstatemir.Block{trial.p.Blocks}
				for _, h := range trial.p.Helpers {
					bodies = append(bodies, h.Blocks)
				}
				for _, blocks := range bodies {
					for _, b := range blocks {
						for _, op := range b.Operations {
							count++
							switch op.Kind {
							case contractstatemir.OperationCellRead, contractstatemir.OperationMapRead, contractstatemir.OperationCellBorrow, contractstatemir.OperationMapBorrow:
								reads++
							case contractstatemir.OperationCall:
								calls++
							}
						}
					}
				}
				for mode, source := range parts[:2] {
					for _, want := range trial.want {
						if !strings.Contains(source, want) {
							t.Fatalf("mode %d missing %q in %s", mode, want, source)
						}
					}
					wantObservers := reads
					if mode == 1 {
						wantObservers = count
					}
					if strings.Count(source, "status = observe(") != wantObservers || strings.Count(source, "fycs_graph_step_v1(context,1u,1u)") != count || strings.Count(source, "fycs_graph_value_v1 audit_key") != mode*(count-calls) {
						t.Fatalf("mode %d observer/step/audit counts: %s", mode, source)
					}
				}
				for site := 1; site <= count; site++ {
					needle := fmt.Sprintf("status = observe(user,context->operation,%du,", site)
					at := strings.Index(parts[1], needle)
					if at < 0 || !strings.HasSuffix(parts[1][:at], "    ") || !strings.Contains(parts[1][:at], "fycs_graph_step_v1(context,1u,1u)") {
						t.Fatalf("audit site %d missing post-step observation", site)
					}
				}
			})
		}
	}, "module_state_declarations", "candidate_artifact_wire")
}

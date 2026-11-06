package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
)

const stateGraphIRSource = `module main;@contract_uid("fixture.graph-ir")state{cell bool flag;cell u32 count;cell u64 total;cell bytes20 owner;cell bytes32 root;cell bytes blob;cell text label;map u32 u32 values;map text bytes labels;}`

const stateGraphIRProbe = `module main;
import module_state_graph_model as m;import module_state_graph_verify as verify;
import module_state_declarations as declarations;import module_state_layout as state;
import module_catalog as catalog;import syntax as syntax;import scanner as scanner;
import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn usize number(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){return wire::number_from(raw,c,work);}
fn m::Operand operand(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){return m::Operand{.input=number(raw,c,work),.codec=wire::word_from(raw,c,work),.start=number(raw,c,work),.end=number(raw,c,work)};}
fn Result<state::Plan,u32> layout(){
 let data=switch(bytes::copy(b"` + stateGraphIRSource + `")){case .ok(p):{p}case .err(_):{return .err(1003);}};
 let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return .err(1001);}};
 let parsed=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{return .err(1001);}};if(syntax::problem_count(ref parsed)!=0){return .err(1001);}
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed).is_ok);
 let c=switch(catalog::build(ref sources)){case .ok(p):{p}case .err(_):{return .err(1001);}};
 return switch(declarations::build_graph(ref sources,ref c)){case .ok(p):{.ok(p)}case .err(_):{.err(1001)}};
}
fn Result<m::Graph,u32> read(ref u8[] raw,ref mut usize work){
 var c=switch(wire::cursor(raw,b"FYSGTEST",work)){case .ok(c):{c}case .err(code):{return .err(code);}};
 let effects=wire::word_from(raw,ref mut c,work);let capability=number(raw,ref mut c,work);let layout=number(raw,ref mut c,work);let entry=number(raw,ref mut c,work);
 var v::Vec<m::Value> ps=v::empty<m::Value>();var v::Vec<usize> args=v::empty<usize>();var v::Vec<m::Operation> ops=v::empty<m::Operation>();var v::Vec<m::Block> bs=v::empty<m::Block>();
 let parameters=wire::count_from(raw,ref mut c,4096,16,work);let arguments=wire::count_from(raw,ref mut c,8192,8,work);let operations=wire::count_from(raw,ref mut c,1025,112,work);let blocks=wire::count_from(raw,ref mut c,65,128,work);let size=wire::count_from(raw,ref mut c,1048577,1,work);
 if(wire::status(ref c)!=0){return .err(wire::status(ref c));}var usize i=0;
 while(i<parameters){let row=m::Value{.id=number(raw,ref mut c,work),.codec=wire::word_from(raw,ref mut c,work)};if(!v::push<m::Value>(ref mut ps,row).is_ok){return .err(1003);}i+=1;}i=0;
 while(i<arguments){if(!v::push<usize>(ref mut args,number(raw,ref mut c,work)).is_ok){return .err(1003);}i+=1;}i=0;
 while(i<operations){let row=m::Operation{.kind=wire::word_from(raw,ref mut c,work),.capability=number(raw,ref mut c,work),.field=number(raw,ref mut c,work),.key=operand(raw,ref mut c,work),.value=operand(raw,ref mut c,work),.result=number(raw,ref mut c,work),.found=number(raw,ref mut c,work),.cost=number(raw,ref mut c,work)};if(!v::push<m::Operation>(ref mut ops,row).is_ok){return .err(1003);}i+=1;}i=0;
 while(i<blocks){let id=number(raw,ref mut c,work);let ps=number(raw,ref mut c,work);let pc=number(raw,ref mut c,work);let os=number(raw,ref mut c,work);let oc=number(raw,ref mut c,work);
 let term=m::Terminator{.kind=wire::word_from(raw,ref mut c,work),.condition=number(raw,ref mut c,work),.target=number(raw,ref mut c,work),.argument_start=number(raw,ref mut c,work),.argument_count=number(raw,ref mut c,work),.alternative=number(raw,ref mut c,work),.alternate_start=number(raw,ref mut c,work),.alternate_count=number(raw,ref mut c,work),.category_start=number(raw,ref mut c,work),.category_end=number(raw,ref mut c,work),.cost=number(raw,ref mut c,work)};
 if(!v::push<m::Block>(ref mut bs,m::Block{.id,.parameter_start=ps,.parameter_count=pc,.operation_start=os,.operation_count=oc,.term}).is_ok){return .err(1003);}i+=1;}
 if(wire::status(ref c)!=0){return .err(wire::status(ref c));}if(c.offset>raw.len || size!=raw.len-c.offset){return .err(1001);}
 let data=switch(bytes::copy(raw[c.offset..])){case .ok(p):{p}case .err(_):{return .err(1003);}};return .ok(m::assemble(m::Header{.effects,.capability,.layout,.entry},data,ps,args,ops,bs));
}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>2097152-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 var usize work=0;let g=switch(read(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
 let p=switch(layout()){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
 work=0;let code=verify::validate(ref g,ref p,ref mut work);if(code!=0){discard io::write_u32(code);return 0;}let cost=work;
 work=16777216-cost;if(verify::validate(ref g,ref p,ref mut work)!=0 || work!=16777216){return 31;}
 work=16777216-cost+1;if(verify::validate(ref g,ref p,ref mut work)!=1002 || work>16777216){return 32;}
 work=16777217;if(verify::validate(ref g,ref p,ref mut work)!=1002 || work!=16777217){return 33;}
 work=0;let proof=switch(verify::verify(g,ref p,ref mut work)){case .ok(proof):{proof}case .err(_):{return 34;}};
 if(v::len<m::Block>(m::blocks(verify::graph(ref proof)))==0 || state::count(verify::layouts(ref proof))!=1){return 35;}
 discard io::write("0");return 0;
}`

func stateGraphIRBase(t *testing.T) contractstatemir.Program {
	t.Helper()
	fields := []hir.StateField{}
	for i, name := range []string{"flag", "count", "total", "owner", "root", "blob", "label"} {
		fields = append(fields, hir.StateField{Name: name, Kind: hir.StateFieldCell, ValueTypeID: stateGraphIRType(uint64(i + 1))})
	}
	fields = append(fields, hir.StateField{Name: "values", Kind: hir.StateFieldMap, KeyTypeID: contractstate.U32.TypeID(), ValueTypeID: contractstate.U32.TypeID()}, hir.StateField{Name: "labels", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Text.TypeID(), ValueTypeID: contractstate.Bytes.TypeID()})
	key, _ := contractstate.U32.Encode(7)
	p := hir.Package{Modules: []hir.Module{{Package: hir.DefaultPackage, Path: "main", Functions: []hir.Function{{Symbol: 1, Name: "transaction", ResultType: hir.TypeVoid, Effects: []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}}}}}, StateLayouts: []hir.StateLayout{{Package: hir.DefaultPackage, ContractUID: "fixture.graph-ir", Fields: fields}}, StateTransactions: []hir.StateTransaction{{Package: hir.DefaultPackage, Function: 1, Blocks: []hir.StateBlock{
		{ID: 0, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "flag", Result: 1, FoundResult: 2, Cost: 1}, {Kind: hir.StateAccessCellRead, Field: "count", Result: 3, FoundResult: 4, Cost: 1}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorBranch, Condition: 1, TrueTarget: 1, TrueArguments: []hir.StateValueID{3}, FalseTarget: 2, FalseArguments: []hir.StateValueID{3}, Cost: 1}},
		{ID: 1, Parameters: []hir.StateBlockParameter{{ID: 5, TypeID: contractstate.U32.TypeID()}}, Accesses: []hir.StateAccess{{Kind: hir.StateAccessMapWrite, Field: "values", Key: hir.StateValue{TypeID: contractstate.U32.TypeID(), Encoded: key}, ValueInput: 5, Cost: 1}, {Kind: hir.StateAccessMapRead, Field: "values", Key: hir.StateValue{TypeID: contractstate.U32.TypeID(), Encoded: key}, Result: 7, FoundResult: 8, Cost: 1}, {Kind: hir.StateAccessCellWrite, Field: "count", ValueInput: 7, Cost: 1}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: 3, Arguments: []hir.StateValueID{7}, Cost: 1}},
		{ID: 2, Parameters: []hir.StateBlockParameter{{ID: 6, TypeID: contractstate.U32.TypeID()}}, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellWrite, Field: "count", ValueInput: 6, Cost: 1}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: 3, Arguments: []hir.StateValueID{6}, Cost: 1}},
		{ID: 3, Parameters: []hir.StateBlockParameter{{ID: 9, TypeID: contractstate.U32.TypeID()}}, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellWrite, Field: "count", ValueInput: 9, Cost: 1}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}},
	}}}}
	lowered, err := contractstatemir.LowerHIRTransactions(p)
	if err != nil || len(lowered) != 1 {
		t.Fatalf("graph IR oracle lowering: %v %+v", err, lowered)
	}
	return lowered[0].Program
}

func stateGraphIRType(id uint64) string {
	return []string{"", "bool/v1", "u32/be/v1", "u64/be/v1", "bytes20/v1", "bytes32/v1", "bytes/v1", "utf8/v1"}[id]
}

func stateGraphIRPacket(p contractstatemir.Program) []byte {
	var out, data bytes.Buffer
	out.WriteString("FYSGTEST")
	number := func(n uint64) { _ = binary.Write(&out, binary.BigEndian, n) }
	codec := func(text string) uint64 {
		for i := uint64(1); i <= 7; i++ {
			if text == stateGraphIRType(i) {
				return i
			}
		}
		return 0
	}
	capability := func(c contractstatemir.TransactionCapability) uint64 {
		if c == (contractstatemir.TransactionCapability{}) {
			return 0
		}
		if c == p.Capability {
			return 1
		}
		return 2
	}
	var effects uint64
	for _, e := range p.Effects {
		switch e {
		case hir.EffectStorageRead:
			effects |= 1
		case hir.EffectStorageWrite:
			effects |= 2
		default:
			effects |= 4
		}
	}
	var parameters, operations, arguments uint64
	for _, b := range p.Blocks {
		parameters += uint64(len(b.Parameters))
		operations += uint64(len(b.Operations))
		arguments += uint64(len(b.Terminator.Arguments) + len(b.Terminator.TrueArguments) + len(b.Terminator.FalseArguments))
	}
	for _, n := range []uint64{effects, capability(p.Capability), 0, uint64(p.Entry), parameters, arguments, operations, uint64(len(p.Blocks)), 0} {
		number(n)
	}
	for _, b := range p.Blocks {
		for _, param := range b.Parameters {
			number(uint64(param.ID))
			number(codec(param.TypeID))
		}
	}
	for _, b := range p.Blocks {
		for _, list := range [][]contractstatemir.ValueID{b.Terminator.Arguments, b.Terminator.TrueArguments, b.Terminator.FalseArguments} {
			for _, value := range list {
				number(uint64(value))
			}
		}
	}
	operand := func(input contractstatemir.ValueID, value contractstatemir.Value) {
		number(uint64(input))
		number(codec(value.TypeID))
		start, end := uint64(0), uint64(0)
		if value.TypeID != "" || value.Encoded != nil {
			start = uint64(data.Len())
			data.Write(value.Encoded)
			end = uint64(data.Len())
		}
		number(start)
		number(end)
	}
	kinds := map[contractstatemir.OperationKind]uint64{contractstatemir.OperationCellRead: 1, contractstatemir.OperationCellWrite: 2, contractstatemir.OperationCellDelete: 3, contractstatemir.OperationMapRead: 4, contractstatemir.OperationMapWrite: 5, contractstatemir.OperationMapDelete: 6}
	for _, b := range p.Blocks {
		for _, op := range b.Operations {
			number(kinds[op.Kind])
			number(capability(op.Capability))
			number(uint64(op.FieldIndex))
			operand(op.KeyInput, op.Key)
			operand(op.ValueInput, op.Value)
			for _, n := range []uint64{uint64(op.Result), uint64(op.Found), uint64(op.Cost)} {
				number(n)
			}
		}
	}
	parameters, operations, arguments = 0, 0, 0
	terms := map[contractstatemir.TerminatorKind]uint64{contractstatemir.TerminatorJump: 1, contractstatemir.TerminatorBranch: 2, contractstatemir.TerminatorCommit: 3, contractstatemir.TerminatorReject: 4, contractstatemir.TerminatorTrap: 5}
	for _, b := range p.Blocks {
		term := b.Terminator
		for _, n := range []uint64{uint64(b.ID), parameters, uint64(len(b.Parameters)), operations, uint64(len(b.Operations)), terms[term.Kind], uint64(term.Condition)} {
			number(n)
		}
		target, first, count, alternate, second, other := uint64(term.Target), uint64(0), uint64(0), uint64(0), uint64(0), uint64(0)
		if term.Kind == contractstatemir.TerminatorJump {
			first, count = arguments, uint64(len(term.Arguments))
			arguments += count
		} else if term.Kind == contractstatemir.TerminatorBranch {
			target, first, count, alternate = uint64(term.TrueTarget), arguments, uint64(len(term.TrueArguments)), uint64(term.FalseTarget)
			arguments += count
			second, other = arguments, uint64(len(term.FalseArguments))
			arguments += other
		}
		start, end := uint64(0), uint64(0)
		if term.TrapCategory != "" {
			start = uint64(data.Len())
			data.WriteString(term.TrapCategory)
			end = uint64(data.Len())
		}
		for _, n := range []uint64{target, first, count, alternate, second, other, start, end, uint64(term.Cost)} {
			number(n)
		}
		parameters += uint64(len(b.Parameters))
		operations += uint64(len(b.Operations))
	}
	packet := out.Bytes()
	binary.BigEndian.PutUint64(packet[8+8*8:], uint64(data.Len()))
	return append(packet, data.Bytes()...)
}

func TestCandidateStateGraphIR(t *testing.T) {
	base := stateGraphIRBase(t)
	if failures := contractstatemir.Verify(base); len(failures) != 0 {
		t.Fatal(failures)
	}
	type trial struct {
		name string
		p    contractstatemir.Program
	}
	trials := []trial{{"typed-branch-join", base}}
	add := func(name string, mutate func(*contractstatemir.Program)) {
		p := stateGraphIRBase(t)
		mutate(&p)
		trials = append(trials, trial{name, p})
	}
	add("cross-arm-value", func(p *contractstatemir.Program) { p.Blocks[2].Operations[0].ValueInput = 7 })
	add("duplicate-parameter", func(p *contractstatemir.Program) { p.Blocks[2].Parameters[0].ID = 5 })
	add("duplicate-read", func(p *contractstatemir.Program) { p.Blocks[0].Operations[1].Result = 1 })
	add("read-no-found", func(p *contractstatemir.Program) { p.Blocks[0].Operations[0].Found = 0 })
	add("found-aliases-value", func(p *contractstatemir.Program) { p.Blocks[0].Operations[0].Found = 1 })
	add("future-value", func(p *contractstatemir.Program) { p.Blocks[1].Operations[0].ValueInput = 7 })
	add("dynamic-literal-mixed", func(p *contractstatemir.Program) {
		p.Blocks[1].Operations[0].Value = contractstatemir.Value{TypeID: contractstate.U32.TypeID(), Encoded: []byte{0, 0, 0, 1}}
	})
	add("non-bool-condition", func(p *contractstatemir.Program) { p.Blocks[0].Terminator.Condition = 3 })
	add("wrong-edge-type", func(p *contractstatemir.Program) { p.Blocks[0].Terminator.TrueArguments[0] = 1 })
	add("missing-edge-argument", func(p *contractstatemir.Program) { p.Blocks[0].Terminator.TrueArguments = nil })
	add("edge-out-of-range", func(p *contractstatemir.Program) { p.Blocks[0].Terminator.TrueTarget = 4 })
	add("unknown-value", func(p *contractstatemir.Program) { p.Blocks[1].Terminator.Arguments[0] = 999 })
	add("wrong-field-kind", func(p *contractstatemir.Program) { p.Blocks[1].Operations[0].FieldIndex = 1 })
	add("field-out-of-range", func(p *contractstatemir.Program) { p.Blocks[0].Operations[0].FieldIndex = 999 })
	add("noncanonical-u32", func(p *contractstatemir.Program) { p.Blocks[1].Operations[0].Key.Encoded = []byte{1} })
	add("wrong-key-type", func(p *contractstatemir.Program) { p.Blocks[1].Operations[0].Key.TypeID = contractstate.U64.TypeID() })
	add("missing-read-effect", func(p *contractstatemir.Program) { p.Effects = []hir.Effect{hir.EffectStorageWrite} })
	add("missing-write-effect", func(p *contractstatemir.Program) { p.Effects = []hir.Effect{hir.EffectStorageRead} })
	add("foreign-effect", func(p *contractstatemir.Program) { p.Effects = append(p.Effects, hir.EffectConsole) })
	add("zero-capability", func(p *contractstatemir.Program) { p.Capability = contractstatemir.TransactionCapability{} })
	add("foreign-capability", func(p *contractstatemir.Program) {
		p.Blocks[0].Operations[0].Capability = contractstatemir.TransactionCapability{}
	})
	add("operation-cost", func(p *contractstatemir.Program) { p.Blocks[0].Operations[0].Cost = 0 })
	add("terminator-cost", func(p *contractstatemir.Program) { p.Blocks[0].Terminator.Cost = 2 })
	add("non-dense-block", func(p *contractstatemir.Program) { p.Blocks[1].ID = 4 })
	add("entry-parameters", func(p *contractstatemir.Program) { p.Entry = 1 })
	add("unreachable-arm", func(p *contractstatemir.Program) { p.Blocks[0].Terminator.FalseTarget = 1 })
	add("reachable-cycle", func(p *contractstatemir.Program) {
		p.Blocks[3].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorJump, Target: 1, Arguments: []contractstatemir.ValueID{9}, Cost: 1}
	})
	for _, n := range []int{63, 64, 65} {
		add(fmt.Sprintf("blocks-%d", n), func(p *contractstatemir.Program) {
			p.Blocks = make([]contractstatemir.Block, n)
			for i := range p.Blocks {
				term := contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}
				if i+1 < n {
					term = contractstatemir.Terminator{Kind: contractstatemir.TerminatorJump, Target: contractstatemir.BlockID(i + 1), Cost: 1}
				}
				p.Blocks[i] = contractstatemir.Block{ID: contractstatemir.BlockID(i), Terminator: term}
			}
		})
	}
	for _, n := range []int{1023, 1024, 1025} {
		add(fmt.Sprintf("operations-%d", n), func(p *contractstatemir.Program) {
			p.Blocks = []contractstatemir.Block{{Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}}}
			for range n {
				p.Blocks[0].Operations = append(p.Blocks[0].Operations, contractstatemir.Operation{Kind: contractstatemir.OperationCellDelete, Capability: p.Capability, FieldIndex: 1, Cost: 1})
			}
		})
	}
	for c := uint64(1); c <= 7; c++ {
		var raw []byte
		switch c {
		case 1:
			raw = []byte{1}
		case 2:
			raw = []byte{255, 255, 255, 255}
		case 3:
			raw = bytes.Repeat([]byte{255}, 8)
		case 4:
			raw = bytes.Repeat([]byte{255}, 20)
		case 5:
			raw = bytes.Repeat([]byte{255}, 32)
		case 6:
			raw = []byte{0, 255, 1}
		case 7:
			raw = []byte("가🙂")
		}
		for _, bad := range []bool{false, true} {
			add(fmt.Sprintf("codec-%d-bad-%v", c, bad), func(p *contractstatemir.Program) {
				value := append([]byte{}, raw...)
				if bad {
					if c == 1 {
						value[0] = 2
					} else if c == 7 {
						value = []byte{0xff}
					} else {
						value = value[:len(value)-1]
					}
				}
				p.Blocks = []contractstatemir.Block{{Operations: []contractstatemir.Operation{{Kind: contractstatemir.OperationCellWrite, Capability: p.Capability, FieldIndex: uint32(c - 1), Value: contractstatemir.Value{TypeID: stateGraphIRType(c), Encoded: value}, Cost: 1}}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}}}
			})
		}
	}
	for _, category := range []string{"a", strings.Repeat("a", 127), "", strings.Repeat("a", 128), "-a", "a-", "Bad", "é"} {
		add("trap-"+category, func(p *contractstatemir.Program) {
			p.Blocks = []contractstatemir.Block{{Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: category, Cost: 1}}}
		})
	}
	// The probe source embeds a literal Fairy program, so escape only its quotes.
	probe := strings.Replace(stateGraphIRProbe, `b"`+stateGraphIRSource+`"`, `b"`+strings.ReplaceAll(stateGraphIRSource, `"`, `\"`)+`"`, 1)
	candidateAtlasProbeSize(t, "module_state_graph_verify", probe, 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				want := len(contractstatemir.Verify(trial.p)) == 0
				code, output := candidateAtlasProcess(t, image, stateGraphIRPacket(trial.p))
				if code != 0 || output != "0" && output != "1001" && output != "1002" || (output == "0") != want {
					t.Fatalf("Go accepted=%v, Fairy exit=%d output=%q, Go failures=%v", want, code, output, contractstatemir.Verify(trial.p))
				}
			})
		}
	}, "module_state_declarations", "candidate_artifact_wire")
}

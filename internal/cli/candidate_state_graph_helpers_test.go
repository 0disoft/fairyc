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
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Expected admissions are authored independently of Verify: native parity must
// not bless a shared false positive by deriving its expectation from the oracle.
type stateGraphHelperTrial struct {
	name string
	p    contractstatemir.Program
	want bool
	code string
}

func stateGraphHelperProgram(t *testing.T, shape string) contractstatemir.Program {
	t.Helper()
	p := stateGraphIRBase(t)
	p.Operations, p.Exit, p.TrapCategory, p.Entry = nil, "", "", 0
	p.Effects = []hir.Effect{hir.EffectStorageWrite}
	call := func(name string) contractstatemir.Operation {
		return contractstatemir.Operation{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: name, Cost: 1}
	}
	write := contractstatemir.Operation{Kind: contractstatemir.OperationCellWrite, Capability: p.Capability, FieldIndex: 1, Value: contractstatemir.Value{TypeID: contractstate.U32.TypeID(), Encoded: []byte{0, 0, 0, 7}}, Cost: 1}
	read := contractstatemir.Operation{Kind: contractstatemir.OperationCellRead, Capability: p.Capability, FieldIndex: 0, Result: 1, Found: 2, Cost: 1}
	block := func(kind contractstatemir.TerminatorKind, ops ...contractstatemir.Operation) contractstatemir.Block {
		return contractstatemir.Block{Operations: ops, Terminator: contractstatemir.Terminator{Kind: kind, Cost: 1}}
	}
	helper := func(name string, ops ...contractstatemir.Operation) contractstatemir.Helper {
		return contractstatemir.Helper{Name: name, Effects: []hir.Effect{hir.EffectStorageWrite}, Blocks: []contractstatemir.Block{block(contractstatemir.TerminatorReturn, ops...)}}
	}
	p.Blocks = []contractstatemir.Block{block(contractstatemir.TerminatorCommit, call("main::a"))}
	p.Helpers = []contractstatemir.Helper{helper("main::a", write)}
	switch shape {
	case "leaf":
	case "read-only":
		p.Effects = []hir.Effect{hir.EffectStorageRead}
		p.Helpers[0].Effects = []hir.Effect{hir.EffectStorageRead}
		p.Helpers[0].Blocks[0].Operations = []contractstatemir.Operation{read}
	case "empty":
		p.Effects, p.Helpers[0].Effects = nil, nil
		p.Helpers[0].Blocks[0].Operations = nil
	case "no-helpers":
		p.Helpers = nil
		p.Blocks[0].Operations = []contractstatemir.Operation{write}
	case "transitive":
		p.Helpers = []contractstatemir.Helper{helper("main::a", call("main::b")), helper("main::b", write)}
	case "diamond":
		p.Blocks[0].Operations = []contractstatemir.Operation{call("main::a"), call("main::b")}
		p.Helpers = []contractstatemir.Helper{helper("main::a", call("main::c")), helper("main::b", call("main::c")), helper("main::c", write)}
	case "branch", "helper-branch":
		p.Effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}
		branches := []contractstatemir.Block{
			{Operations: []contractstatemir.Operation{read}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorBranch, Condition: 1, TrueTarget: 1, FalseTarget: 2, Cost: 1}},
			{ID: 1, Operations: []contractstatemir.Operation{call("main::b")}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}},
			{ID: 2, Operations: []contractstatemir.Operation{call("main::c")}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}},
		}
		p.Helpers = []contractstatemir.Helper{helper("main::b", write), helper("main::c", write)}
		if shape == "branch" {
			p.Blocks = branches
		} else {
			branches[1].Terminator.Kind, branches[2].Terminator.Kind = contractstatemir.TerminatorReturn, contractstatemir.TerminatorReturn
			p.Helpers = append([]contractstatemir.Helper{{Name: "main::a", Effects: append([]hir.Effect(nil), p.Effects...), Blocks: branches}}, p.Helpers...)
		}
	default:
		t.Fatalf("unknown helper shape %q", shape)
	}
	return p
}

func stateGraphHelperTrials(t *testing.T) []stateGraphHelperTrial {
	t.Helper()
	var trials []stateGraphHelperTrial
	add := func(name, shape string, want bool, code string, mutate func(*contractstatemir.Program)) {
		p := stateGraphHelperProgram(t, shape)
		if mutate != nil {
			mutate(&p)
		}
		trials = append(trials, stateGraphHelperTrial{name, p, want, code})
	}
	for _, shape := range []string{"leaf", "read-only", "empty", "no-helpers", "transitive", "diamond", "branch", "helper-branch"} {
		add(shape, shape, true, "", nil)
	}
	add("transitive-read-only", "transitive", true, "", func(p *contractstatemir.Program) {
		p.Effects = []hir.Effect{hir.EffectStorageRead}
		for i := range p.Helpers {
			p.Helpers[i].Effects = []hir.Effect{hir.EffectStorageRead}
		}
		p.Helpers[1].Blocks[0].Operations = []contractstatemir.Operation{{Kind: contractstatemir.OperationCellRead, Capability: p.Capability, FieldIndex: 0, Result: 1, Found: 2, Cost: 1}}
	})
	add("diamond-mixed-effects", "diamond", true, "", func(p *contractstatemir.Program) {
		p.Effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}
		for i := range p.Helpers {
			p.Helpers[i].Effects = append([]hir.Effect(nil), p.Effects...)
		}
		p.Helpers[2].Blocks[0].Operations = append(p.Helpers[2].Blocks[0].Operations, contractstatemir.Operation{Kind: contractstatemir.OperationCellRead, Capability: p.Capability, FieldIndex: 0, Result: 1, Found: 2, Cost: 1})
	})
	add("backward-lexical-call", "diamond", true, "", func(p *contractstatemir.Program) {
		p.Blocks[0].Operations = p.Blocks[0].Operations[1:]
		p.Helpers[1].Blocks[0].Operations[0].Callee = "main::a"
	})
	add("root-extra-effects-without-helpers", "no-helpers", true, "", func(p *contractstatemir.Program) {
		p.Effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}
	})
	add("repeated-call", "leaf", true, "", func(p *contractstatemir.Program) {
		p.Blocks[0].Operations = append(p.Blocks[0].Operations, p.Blocks[0].Operations[0])
	})
	for _, shape := range []string{"read-preserves", "write-invalidates", "copy-survives-write"} {
		base, valid, diagnostic := "leaf", true, ""
		if shape == "read-preserves" {
			base = "read-only"
		} else if shape == "write-invalidates" {
			valid, diagnostic = false, "CSM1010"
		}
		add("root-borrow-"+shape, base, valid, diagnostic, func(p *contractstatemir.Program) {
			p.Effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}
			if shape == "read-preserves" {
				p.Effects = []hir.Effect{hir.EffectStorageRead}
			}
			borrow := contractstatemir.Operation{Kind: contractstatemir.OperationCellBorrow, Capability: p.Capability, FieldIndex: 5, BorrowResult: 1, Found: 1, Cost: 1}
			copy := contractstatemir.Operation{Kind: contractstatemir.OperationBorrowCopy, Capability: p.Capability, BorrowInput: 1, Result: 2, Cost: 1}
			call := p.Blocks[0].Operations[0]
			p.Blocks[0].Operations = []contractstatemir.Operation{borrow, call, copy}
			if shape == "copy-survives-write" {
				p.Blocks[0].Operations = []contractstatemir.Operation{borrow, copy, call, {Kind: contractstatemir.OperationCellWrite, Capability: p.Capability, FieldIndex: 5, ValueInput: 2, Cost: 1}}
			}
		})
	}
	for _, name := range []string{"_main::_a0", "main::nested::a", strings.Repeat("a", 128) + "::a"} {
		add("canonical-name-"+name, "leaf", true, "", func(p *contractstatemir.Program) { p.Helpers[0].Name, p.Blocks[0].Operations[0].Callee = name, name })
	}
	for _, name := range []string{"", "a", "::a", "main::", "main:::a", "main::9a", "main::bad-name", "main::é", "main::a\x00"} {
		add(fmt.Sprintf("noncanonical-name-%q", name), "leaf", false, "CSM1009", func(p *contractstatemir.Program) { p.Helpers[0].Name, p.Blocks[0].Operations[0].Callee = name, name })
	}
	add("foreign-call", "leaf", false, "CSM1009", func(p *contractstatemir.Program) { p.Blocks[0].Operations[0].Callee = "foreign::missing" })
	add("zero-callee", "leaf", false, "CSM1009", func(p *contractstatemir.Program) { p.Blocks[0].Operations[0].Callee = "" })
	add("unused-helper", "leaf", false, "CSM1009", func(p *contractstatemir.Program) {
		h := p.Helpers[0]
		h.Name = "main::z"
		p.Helpers = append(p.Helpers, h)
	})
	add("duplicate-name", "leaf", false, "CSM1009", func(p *contractstatemir.Program) { p.Helpers = append(p.Helpers, p.Helpers[0]) })
	add("unsorted-names", "diamond", false, "CSM1009", func(p *contractstatemir.Program) { p.Helpers[0], p.Helpers[1] = p.Helpers[1], p.Helpers[0] })
	add("self-recursion", "leaf", false, "CSM1009", func(p *contractstatemir.Program) {
		p.Helpers[0].Blocks[0].Operations = []contractstatemir.Operation{p.Blocks[0].Operations[0]}
	})
	add("mutual-recursion", "transitive", false, "CSM1009", func(p *contractstatemir.Program) {
		p.Helpers[1].Blocks[0].Operations = []contractstatemir.Operation{p.Blocks[0].Operations[0]}
	})
	add("unreachable-recursion", "leaf", false, "CSM1009", func(p *contractstatemir.Program) {
		h := p.Helpers[0]
		h.Name = "main::z"
		h.Blocks = []contractstatemir.Block{{Operations: []contractstatemir.Operation{{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: h.Name, Cost: 1}}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorReturn, Cost: 1}}}
		p.Helpers = append(p.Helpers, h)
	})
	for _, owner := range []string{"root", "helper"} {
		for _, effect := range []string{"missing", "excess", "foreign", "duplicate", "unsorted"} {
			add(owner+"-effect-"+effect, "transitive", false, "CSM1005", func(p *contractstatemir.Program) {
				effects := &p.Effects
				if owner == "helper" {
					effects = &p.Helpers[0].Effects
				}
				switch effect {
				case "missing":
					*effects = nil
				case "excess":
					*effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}
				case "foreign":
					*effects = append(*effects, hir.EffectConsole)
				case "duplicate":
					*effects = append(*effects, hir.EffectStorageWrite)
				case "unsorted":
					*effects = []hir.Effect{hir.EffectStorageWrite, hir.EffectStorageRead}
				}
			})
		}
		add(owner+"-operation-cost", "leaf", false, "CSM1008", func(p *contractstatemir.Program) {
			if owner == "root" {
				p.Blocks[0].Operations[0].Cost = 0
			} else {
				p.Helpers[0].Blocks[0].Operations[0].Cost = 2
			}
		})
		add(owner+"-terminator-cost", "leaf", false, "CSM1008", func(p *contractstatemir.Program) {
			if owner == "root" {
				p.Blocks[0].Terminator.Cost = 0
			} else {
				p.Helpers[0].Blocks[0].Terminator.Cost = 2
			}
		})
		for _, foreign := range []bool{false, true} {
			add(fmt.Sprintf("%s-operation-capability-foreign-%v", owner, foreign), "leaf", false, "CSM1004", func(p *contractstatemir.Program) {
				capability := contractstatemir.TransactionCapability{}
				if foreign {
					capability = stateGraphIRBase(t).Capability
				}
				if owner == "root" {
					p.Blocks[0].Operations[0].Capability = capability
				} else {
					p.Helpers[0].Blocks[0].Operations[0].Capability = capability
				}
			})
		}
	}
	add("root-zero-capability", "leaf", false, "CSM1004", func(p *contractstatemir.Program) { p.Capability = contractstatemir.TransactionCapability{} })
	add("root-return", "leaf", false, "CSM1006", func(p *contractstatemir.Program) { p.Blocks[0].Terminator.Kind = contractstatemir.TerminatorReturn })
	add("helper-commit", "leaf", false, "CSM1006", func(p *contractstatemir.Program) {
		p.Helpers[0].Blocks[0].Terminator.Kind = contractstatemir.TerminatorCommit
	})
	add("helper-return-fields", "leaf", false, "CSM1006", func(p *contractstatemir.Program) { p.Helpers[0].Blocks[0].Terminator.Target = 1 })
	add("helper-reject", "leaf", true, "", func(p *contractstatemir.Program) {
		p.Helpers[0].Blocks[0].Terminator.Kind = contractstatemir.TerminatorReject
	})
	add("helper-trap", "leaf", true, "", func(p *contractstatemir.Program) {
		p.Helpers[0].Blocks[0].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorTrap, TrapCategory: "helper-failed", Cost: 1}
	})
	for _, kind := range []contractstatemir.OperationKind{contractstatemir.OperationCellBorrow, contractstatemir.OperationMapBorrow, contractstatemir.OperationBorrowView, contractstatemir.OperationBorrowCopy} {
		add("helper-forbidden-"+string(kind), "leaf", false, "CSM1010", func(p *contractstatemir.Program) {
			p.Helpers[0].Blocks[0].Operations = []contractstatemir.Operation{{Kind: kind, Capability: p.Capability, FieldIndex: 5, BorrowResult: 1, Found: 1, Cost: 1}}
		})
	}
	add("call-storage-fields", "leaf", false, "CSM1009", func(p *contractstatemir.Program) {
		p.Blocks[0].Operations[0].Value = contractstatemir.Value{TypeID: contractstate.U32.TypeID(), Encoded: []byte{0, 0, 0, 7}}
	})
	add("transitive-call-result", "transitive", false, "CSM1009", func(p *contractstatemir.Program) { p.Helpers[0].Blocks[0].Operations[0].Result = 3 })
	add("helper-bad-data", "leaf", false, "CSM1002", func(p *contractstatemir.Program) { p.Helpers[0].Blocks[0].Operations[0].Value.Encoded = []byte{7} })
	add("root-bad-data", "leaf", false, "CSM1002", func(p *contractstatemir.Program) {
		op := p.Helpers[0].Blocks[0].Operations[0]
		op.Value.Encoded = []byte{7}
		p.Blocks[0].Operations = append(p.Blocks[0].Operations, op)
	})
	add("layout-drift", "leaf", false, "CSM1001", func(p *contractstatemir.Program) { p.Layout.Hash[0] ^= 1 })
	for _, n := range []int{31, 32, 33} {
		code := ""
		if n > 32 {
			code = "CSM1009"
		}
		add(fmt.Sprintf("helpers-%d", n), "leaf", n <= 32, code, func(p *contractstatemir.Program) {
			write := p.Helpers[0].Blocks[0].Operations[0]
			p.Helpers = nil
			for i := 0; i < n; i++ {
				op := write
				if i+1 < n {
					op = contractstatemir.Operation{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: fmt.Sprintf("main::h%02d", i+1), Cost: 1}
				}
				p.Helpers = append(p.Helpers, contractstatemir.Helper{Name: fmt.Sprintf("main::h%02d", i), Effects: []hir.Effect{hir.EffectStorageWrite}, Blocks: []contractstatemir.Block{{Operations: []contractstatemir.Operation{op}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorReturn, Cost: 1}}}})
			}
			p.Blocks[0].Operations[0].Callee = "main::h00"
		})
	}
	for _, n := range []int{1023, 1024, 1025} {
		code := ""
		if n > 1024 {
			code = "CSM1009"
		}
		add(fmt.Sprintf("combined-operations-%d", n), "leaf", n <= 1024, code, func(p *contractstatemir.Program) {
			op := p.Helpers[0].Blocks[0].Operations[0]
			p.Helpers[0].Blocks[0].Operations = make([]contractstatemir.Operation, n-1)
			for i := range p.Helpers[0].Blocks[0].Operations {
				p.Helpers[0].Blocks[0].Operations[i] = op
			}
		})
	}
	return trials
}

func stateGraphHelperCheckOracle(t *testing.T, trial stateGraphHelperTrial) {
	t.Helper()
	failures := contractstatemir.Verify(trial.p)
	if (len(failures) == 0) != trial.want {
		t.Fatalf("independent admission want=%v: %v", trial.want, failures)
	}
	if trial.code != "" {
		for _, failure := range failures {
			if failure.Code == trial.code {
				return
			}
		}
		t.Fatalf("missing rejection %s: %v", trial.code, failures)
	}
}

func TestCandidateStateGraphHelpersOracle(t *testing.T) {
	for _, trial := range stateGraphHelperTrials(t) {
		t.Run(trial.name, func(t *testing.T) { stateGraphHelperCheckOracle(t, trial) })
	}
}

// The envelope is test-only. Each embedded graph uses the parent's existing
// 19-field packet and unchanged storage codecs/layout; names are densely packed.
type stateGraphHelperPacket struct {
	graphs [][]byte
	names  [][2]uint64
	data   []byte
}

func stateGraphHelperEncode(p contractstatemir.Program) stateGraphHelperPacket {
	packet := stateGraphHelperPacket{graphs: [][]byte{stateGraphIRPacket(p)}}
	for _, h := range p.Helpers {
		// Every helper must retain the ENTIRE global table, including helpers
		// earlier in lexical order, when translating string callees to 1..N.
		body := contractstatemir.Program{Layout: p.Layout, Capability: p.Capability, Effects: h.Effects, Operations: h.Operations, Exit: h.Exit, TrapCategory: h.TrapCategory, Entry: h.Entry, Blocks: h.Blocks, Helpers: p.Helpers}
		packet.graphs = append(packet.graphs, stateGraphIRPacket(body))
		start := uint64(len(packet.data))
		packet.data = append(packet.data, h.Name...)
		packet.names = append(packet.names, [2]uint64{start, uint64(len(packet.data))})
	}
	return packet
}

func (p stateGraphHelperPacket) bytes() []byte {
	var out bytes.Buffer
	out.WriteString("FYSPTEST")
	number := func(n uint64) { _ = binary.Write(&out, binary.BigEndian, n) }
	number(uint64(len(p.graphs) - 1))
	number(uint64(len(p.names)))
	number(uint64(len(p.data)))
	for _, span := range p.names {
		number(span[0])
		number(span[1])
	}
	for _, graph := range p.graphs {
		number(uint64(len(graph)))
		out.Write(graph)
	}
	out.Write(p.data)
	return out.Bytes()
}

func stateGraphHelperWord(graph []byte, field int, value uint64) {
	binary.BigEndian.PutUint64(graph[8+field*8:], value)
}

func TestCandidateStateGraphHelpersPacket(t *testing.T) {
	p := stateGraphHelperProgram(t, "diamond")
	packet := stateGraphHelperEncode(p)
	for i, want := range []uint64{1, 3, 3} { // root->a, a->c, b->c
		graph := packet.graphs[i]
		if got := binary.BigEndian.Uint64(graph[80+14*8:]); got != want {
			t.Fatalf("graph %d callee=%d want=%d", i, got, want)
		}
	}
	// Backward lexical calls also require the global table on helper graphs.
	p.Blocks[0].Operations = p.Blocks[0].Operations[1:]
	p.Helpers[1].Blocks[0].Operations[0].Callee = "main::a"
	packet = stateGraphHelperEncode(p)
	if got := binary.BigEndian.Uint64(packet.graphs[2][80+14*8:]); got != 1 {
		t.Fatalf("backward helper callee=%d", got)
	}
	for i, span := range packet.names {
		if string(packet.data[span[0]:span[1]]) != p.Helpers[i].Name {
			t.Fatalf("name %d lost", i)
		}
	}
	if !bytes.HasPrefix(packet.bytes(), []byte("FYSPTEST")) {
		t.Fatal("program envelope marker missing")
	}
}

// Keep the tested graph reader and layout builder verbatim, replacing only main
// and adding the new program envelope. This is authored Fairy code, not a Go
// emulation of native validation.
func stateGraphHelperProbe() string {
	probe := stateGraphIRProbe[:strings.Index(stateGraphIRProbe, "@effects(console) fn i32 main()")]
	probe = strings.Replace(probe, `b"`+stateGraphIRSource+`"`, `b"`+strings.ReplaceAll(stateGraphIRSource, `"`, `\"`)+`"`, 1)
	probe = strings.Replace(probe, "import module_state_graph_model as m;", "import module_state_graph_program as program;import module_state_graph_model as m;", 1)
	return probe + stateGraphHelperProgramProbe
}

const stateGraphHelperProgramProbe = `
fn Result<m::Graph,u32> graph_from(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){
 let size=wire::count_from(raw,c,1048576,1,work);if(wire::status(c)!=0){return .err(wire::status(c));}
 let start=c.offset;c.offset+=size;return read(raw[start..c.offset],work);
}
fn Result<program::Program,u32> read_program(ref u8[] raw,ref mut usize work){
 var c=switch(wire::cursor(raw,b"FYSPTEST",work)){case .ok(c):{c}case .err(code):{return .err(code);}};
 let count=wire::count_from(raw,ref mut c,33,8,work);let named=wire::count_from(raw,ref mut c,34,16,work);let size=number(raw,ref mut c,work);
 if(wire::status(ref c)!=0){return .err(wire::status(ref c));}
 var v::Vec<program::Name> names=v::empty<program::Name>();var usize i=0;
 while(i<named){let name=program::Name{.start=number(raw,ref mut c,work),.end=number(raw,ref mut c,work)};if(!v::push<program::Name>(ref mut names,name).is_ok){return .err(1003);}i+=1;}
 let root=switch(graph_from(raw,ref mut c,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
 var v::Vec<m::Graph> helpers=v::empty<m::Graph>();i=0;
 while(i<count){let h=switch(graph_from(raw,ref mut c,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!v::push<m::Graph>(ref mut helpers,h).is_ok){return .err(1003);}i+=1;}
 if(wire::status(ref c)!=0){return .err(wire::status(ref c));}if(c.offset>raw.len || size!=raw.len-c.offset){return .err(1001);}
 let data=switch(bytes::copy(raw[c.offset..])){case .ok(p):{p}case .err(_):{return .err(1003);}};
 return .ok(program::assemble(root,helpers,names,data));
}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>2097152-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 var usize work=0;let g=switch(read_program(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
 var p=switch(layout()){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
 work=0;let code=program::validate(ref g,ref p,ref mut work);if(code!=0){
  work=0;switch(program::verify(g,ref p,ref mut work)){case .ok(_):{return 48;}case .err(found):{if(found!=code){return 49;}}};
  discard io::write_u32(code);return 0;}let cost=work;
 if(cost==0 || cost>16777216){return 30;}
 work=16777216-cost;if(program::validate(ref g,ref p,ref mut work)!=0 || work!=16777216){return 31;}
 work=16777216-cost+1;if(program::validate(ref g,ref p,ref mut work)!=1002 || work>16777216){return 32;}
 work=16777217;if(program::validate(ref g,ref p,ref mut work)!=1002 || work!=16777217){return 33;}
 work=0;let proof=switch(program::verify(g,ref p,ref mut work)){case .ok(proof):{proof}case .err(_):{return 34;}};let seal_cost=work;
 work=0;let exact=switch(read_program(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(_):{return 35;}};
 work=16777216-seal_cost;let exact_proof=switch(program::verify(exact,ref p,ref mut work)){case .ok(p):{p}case .err(_):{return 36;}};if(work!=16777216){return 37;}
 work=0;let short=switch(read_program(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(_):{return 38;}};
 work=16777216-seal_cost+1;switch(program::verify(short,ref p,ref mut work)){case .ok(_):{return 39;}case .err(code):{if(code!=1002 || work>16777216){return 40;}}};
 // Replacing the caller's layout must not mutate either immutable owner.
 p=state::empty();if(state::count(ref p)!=0 || state::count(program::layouts(ref proof))!=1 || state::count(program::layouts(ref exact_proof))!=1){return 41;}
 work=0;if(state::validate_graph(program::layouts(ref proof),ref mut work)!=0){return 42;}
 let root=program::root(ref proof);let exact_root=program::root(ref exact_proof);
 if(m::header(root).capability==0 || v::len<m::Block>(m::blocks(root))==0 || !bytes::equal(m::data(root),m::data(exact_root)) || m::header(root).effects!=m::header(exact_root).effects || v::len<m::Operation>(m::operations(root))!=v::len<m::Operation>(m::operations(exact_root))){return 43;}
 work=0;var cursor=switch(wire::cursor(bytes::view(ref input),b"FYSPTEST",ref mut work)){case .ok(c):{c}case .err(_):{return 44;}};
 let expected=number(bytes::view(ref input),ref mut cursor,ref mut work);discard number(bytes::view(ref input),ref mut cursor,ref mut work);let name_bytes=number(bytes::view(ref input),ref mut cursor,ref mut work);
 if(program::helper_count(ref proof)!=expected || program::helper_count(ref exact_proof)!=expected){return 45;}
 let raw=bytes::view(ref input);let data_start=raw.len-name_bytes;var usize i=0;
 while(i<expected){let start=number(raw,ref mut cursor,ref mut work);let end=number(raw,ref mut cursor,ref mut work);
  if(!bytes::equal(program::helper_name(ref proof,i),raw[data_start+start..data_start+end]) || !bytes::equal(program::helper_name(ref proof,i),program::helper_name(ref exact_proof,i))){return 46;}
  let h=program::helper(ref proof,i);let exact_h=program::helper(ref exact_proof,i);
  if(m::header(h).capability!=m::header(root).capability || m::header(h).layout!=m::header(root).layout || v::len<m::Block>(m::blocks(h))==0 || !bytes::equal(m::data(h),m::data(exact_h)) || m::header(h).effects!=m::header(exact_h).effects || v::len<m::Operation>(m::operations(h))!=v::len<m::Operation>(m::operations(exact_h))){return 47;}i+=1;}
 discard io::write("0");return 0;
}`

func TestCandidateStateGraphHelpers(t *testing.T) {
	trials := stateGraphHelperTrials(t)
	for _, trial := range trials {
		t.Run("oracle/"+trial.name, func(t *testing.T) { stateGraphHelperCheckOracle(t, trial) })
	}
	if t.Failed() {
		t.Fatal("independent oracle expectations failed before native execution")
	}
	candidateAtlasProbeSize(t, "module_state_graph_program", stateGraphHelperProbe(), 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				// Layout hash drift has no representation in a selected-layout
				// graph packet. Selected-index/header drift is tested below.
				if trial.name == "layout-drift" || strings.HasSuffix(trial.name, "-effect-duplicate") || strings.HasSuffix(trial.name, "-effect-unsorted") {
					t.Skip("Go-only identity/order; native packet carries selected layout and effect mask")
				}
				code, output := candidateAtlasProcess(t, image, stateGraphHelperEncode(trial.p).bytes())
				want := "1001"
				if trial.want {
					want = "0"
				} else if trial.name == "helpers-33" || trial.name == "combined-operations-1025" {
					want = "1002"
				}
				if code != 0 || output != want {
					t.Fatalf("want=%s native exit=%d output=%q", want, code, output)
				}
			})
		}
		stateGraphHelperNativeMalformed(t, image)
	}, "module_state_graph_program", "module_state_declarations", "candidate_artifact_wire")
}

func stateGraphHelperNativeMalformed(t *testing.T, image string) {
	t.Helper()
	tests := []struct {
		name   string
		code   string
		mutate func(*stateGraphHelperPacket)
	}{
		{"helper-header-capability", "1001", func(p *stateGraphHelperPacket) { stateGraphHelperWord(p.graphs[1], 1, 2) }},
		{"helper-header-and-body-capability", "1001", func(p *stateGraphHelperPacket) {
			stateGraphHelperWord(p.graphs[1], 1, 2)
			binary.BigEndian.PutUint64(p.graphs[1][80+8:], 2)
		}},
		{"helper-zero-capability", "1001", func(p *stateGraphHelperPacket) { stateGraphHelperWord(p.graphs[1], 1, 0) }},
		{"root-header-capability", "1001", func(p *stateGraphHelperPacket) { stateGraphHelperWord(p.graphs[0], 1, 2) }},
		{"helper-layout", "1001", func(p *stateGraphHelperPacket) { stateGraphHelperWord(p.graphs[1], 2, 1) }},
		{"root-layout", "1001", func(p *stateGraphHelperPacket) { stateGraphHelperWord(p.graphs[0], 2, 1) }},
		{"all-layout-out-of-range", "1001", func(p *stateGraphHelperPacket) {
			for _, g := range p.graphs {
				stateGraphHelperWord(g, 2, 1)
			}
		}},
		{"callee-out-of-range", "1001", func(p *stateGraphHelperPacket) { binary.BigEndian.PutUint64(p.graphs[0][80+14*8:], 2) }},
		{"name-count-short", "1001", func(p *stateGraphHelperPacket) { p.names = nil }},
		{"name-count-long", "1001", func(p *stateGraphHelperPacket) { p.names = append(p.names, p.names[0]) }},
		{"name-leading-gap", "1001", func(p *stateGraphHelperPacket) {
			p.data = append([]byte{'x'}, p.data...)
			p.names[0][0]++
			p.names[0][1]++
		}},
		{"name-reversed-span", "1001", func(p *stateGraphHelperPacket) { p.names[0][0] = p.names[0][1] + 1 }},
		{"name-out-of-range", "1001", func(p *stateGraphHelperPacket) { p.names[0][1]++ }},
		{"name-unused-trailing-data", "1001", func(p *stateGraphHelperPacket) { p.data = append(p.data, 'x') }},
		{"helper-noncanonical-data", "1001", func(p *stateGraphHelperPacket) { g := p.graphs[1]; binary.BigEndian.PutUint64(g[80+10*8:], 3) }},
		{"root-call-borrow-metadata", "1001", func(p *stateGraphHelperPacket) { binary.BigEndian.PutUint64(p.graphs[0][80+15*8:], 1) }},
		{"helper-storage-callee", "1001", func(p *stateGraphHelperPacket) { binary.BigEndian.PutUint64(p.graphs[1][80+14*8:], 1) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packet := stateGraphHelperEncode(stateGraphHelperProgram(t, "leaf"))
			test.mutate(&packet)
			code, output := candidateAtlasProcess(t, image, packet.bytes())
			if code != 0 || output != test.code {
				t.Fatalf("native exit=%d output=%q want=%s", code, output, test.code)
			}
		})
	}
}

// These semantic-only negative probes establish that a valid proof cannot be
// forged, reborrowed mutably, or stripped into mutable graph/layout ownership.
func TestCandidateStateGraphHelpersSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"private-constructor", "E0103", "fn program::Verified forge(){return program::Verified{.program=0,.layouts=0};}"},
		{"private-root-header", "E0103", "fn void change(ref program::Verified proof){program::root(proof).header.capability=0;}"},
		{"immutable-root", "E1008", "fn void change(ref program::Verified proof){let ref mut m::Graph g=ref mut program::root(proof);}"},
		{"immutable-layout", "E1008", "fn void change(ref program::Verified proof){let ref mut state::Plan p=ref mut program::layouts(proof);}"},
		{"immutable-helper", "E1008", "fn void change(ref program::Verified proof){let ref mut m::Graph g=ref mut program::helper(proof,0);}"},
		{"immutable-name", "E1008", "fn void change(ref program::Verified proof){let ref mut u8[] name=ref mut program::helper_name(proof,0);}"},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_program")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_program as program;import module_state_graph_model as m;import module_state_layout as state;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed helper owner", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), trial.code) {
				t.Fatalf("sealed-owner escape accepted or failed outside probe: %d %s", code, &diagnostics)
			}
		})
	}
}

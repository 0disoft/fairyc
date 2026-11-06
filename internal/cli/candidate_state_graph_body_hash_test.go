package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Author HIR from the existing independent fixture matrices, then ask the Go
// producer to lower and describe it. The Fairy transport is NOT the hash oracle.
func stateGraphBodyHIR(p contractstatemir.Program) hir.Package {
	const owner hir.PackageID = 7
	fields := make([]hir.StateField, 0, len(p.Layout.Fields))
	for _, f := range p.Layout.Fields {
		fields = append(fields, hir.StateField{Name: f.Name, Kind: hir.StateFieldKind(f.Kind.String()), KeyTypeID: f.KeyTypeID, ValueTypeID: f.ValueTypeID})
	}
	input := hir.Package{StateLayouts: []hir.StateLayout{{Package: owner, ContractUID: p.Layout.ContractUID, Fields: fields}}}
	symbols := map[string]hir.SymbolID{"main::transaction": 100}
	for i, h := range p.Helpers {
		symbols[h.Name] = hir.SymbolID(200 + i)
	}
	value := func(v contractstatemir.Value) hir.StateValue {
		return hir.StateValue{TypeID: v.TypeID, Encoded: slices.Clone(v.Encoded)}
	}
	ids := func(v []contractstatemir.ValueID) []hir.StateValueID {
		r := make([]hir.StateValueID, len(v))
		for i, id := range v {
			r[i] = hir.StateValueID(id)
		}
		return r
	}
	add := func(name string, effects []hir.Effect, blocks []contractstatemir.Block) {
		split := strings.LastIndex(name, "::")
		input.Modules = append(input.Modules, hir.Module{Package: owner, Path: name[:split], Functions: []hir.Function{{Symbol: symbols[name], Name: name[split+2:], ResultType: hir.TypeVoid, Effects: slices.Clone(effects)}}})
		function := &input.Modules[len(input.Modules)-1].Functions[0]
		tr := hir.StateTransaction{Package: owner, Function: symbols[name]}
		for _, b := range blocks {
			block := hir.StateBlock{ID: hir.StateBlockID(b.ID)}
			for _, param := range b.Parameters {
				block.Parameters = append(block.Parameters, hir.StateBlockParameter{ID: hir.StateValueID(param.ID), TypeID: param.TypeID})
			}
			for _, op := range b.Operations {
				if op.Kind == contractstatemir.OperationCall {
					var calleeEffects []hir.Effect
					for _, h := range p.Helpers {
						if h.Name == op.Callee {
							calleeEffects = slices.Clone(h.Effects)
						}
					}
					function.Body.Statements = append(function.Body.Statements, hir.Statement{Kind: hir.StatementExpressionStatement, Expression: &hir.Expression{Kind: hir.ExpressionCall, Type: hir.TypeVoid, Symbol: symbols[op.Callee], Effects: calleeEffects, Children: []hir.Expression{{Kind: hir.ExpressionPath, Symbol: symbols[op.Callee]}}}})
				}
				field := ""
				if op.Kind != contractstatemir.OperationCall && op.Kind != contractstatemir.OperationBorrowView && op.Kind != contractstatemir.OperationBorrowCopy {
					field = p.Layout.Fields[op.FieldIndex].Name
				}
				block.Accesses = append(block.Accesses, hir.StateAccess{Kind: hir.StateAccessKind(op.Kind), Field: field, Key: value(op.Key), Value: value(op.Value), KeyInput: hir.StateValueID(op.KeyInput), ValueInput: hir.StateValueID(op.ValueInput), Result: hir.StateValueID(op.Result), FoundResult: hir.StateValueID(op.Found), BorrowInput: hir.StateBorrowID(op.BorrowInput), BorrowResult: hir.StateBorrowID(op.BorrowResult), ViewOffset: op.ViewOffset, ViewLength: op.ViewLength, Callee: symbols[op.Callee], Cost: op.Cost})
			}
			term := b.Terminator
			block.Terminator = hir.StateTerminator{Kind: hir.StateTerminatorKind(term.Kind), Condition: hir.StateValueID(term.Condition), Target: hir.StateBlockID(term.Target), Arguments: ids(term.Arguments), TrueTarget: hir.StateBlockID(term.TrueTarget), TrueArguments: ids(term.TrueArguments), FalseTarget: hir.StateBlockID(term.FalseTarget), FalseArguments: ids(term.FalseArguments), TrapCategory: term.TrapCategory, Cost: term.Cost}
			tr.Blocks = append(tr.Blocks, block)
		}
		input.StateTransactions = append(input.StateTransactions, tr)
	}
	add("main::transaction", p.Effects, p.Blocks)
	for _, helper := range p.Helpers {
		add(helper.Name, helper.Effects, helper.Blocks)
	}
	return input
}

type stateBodyFixture struct {
	name                  string
	input                 hir.Package
	coordinate            packagegraph.CoordinateID
	packet                stateGraphHelperPacket
	descriptor, canonical []byte
	hash                  string
}

func stateGraphBodyFixture(t *testing.T, name string, input hir.Package, coordinate packagegraph.CoordinateID) stateBodyFixture {
	t.Helper()
	const owner hir.PackageID = 7
	coords := map[hir.PackageID]packagegraph.CoordinateID{owner: coordinate}
	hashes, err := contractstatemir.ComputeHIRStateBodyHashes(input, []hir.PackageID{owner}, coords)
	if err != nil {
		t.Fatal(err)
	}
	// The actual current Go API is DescribeStableHIRStateBodies, not the proposed
	// BuildStableStateBodyArtifacts spelling in the work order.
	artifacts, err := contractstatemir.DescribeStableHIRStateBodies(input, []hir.PackageID{owner}, coords)
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[owner]
	if len(body.Transactions) != 1 || body.Hash != hashes[owner] {
		t.Fatal("expected one independently verified transaction")
	}
	lowered, err := contractstatemir.LowerHIRTransactions(input)
	if err != nil || len(lowered) != 1 {
		t.Fatalf("lower: %v", err)
	}
	canonical := stateGraphBodyCanonical(body, lowered[0].Program.Layout.Hash)
	sum := sha256.Sum256(canonical)
	want := contractstatemir.StateBodyHashPrefix + hex.EncodeToString(sum[:])
	if want != hashes[owner].String() {
		t.Fatalf("independent byte writer differs from Go body hash: %s %s", want, hashes[owner])
	}
	return stateBodyFixture{name, input, coordinate, stateGraphHelperEncode(lowered[0].Program), body.Transactions[0].Function, canonical, want}
}

// Independently encode the public stable artifact model, never the private
// test packet or a candidate-produced byte stream.
func stateGraphBodyCanonical(body contractstatemir.StableStateBodyArtifact, layout contractstate.LayoutHash) []byte {
	var out bytes.Buffer
	n := func(v uint64) { _ = binary.Write(&out, binary.BigEndian, v) }
	s := func(v string) { n(uint64(len(v))); out.WriteString(v) }
	ss := func(v []string) {
		n(uint64(len(v)))
		for _, item := range v {
			s(item)
		}
	}
	ids := func(v []uint32) {
		n(uint64(len(v)))
		for _, id := range v {
			n(uint64(id))
		}
	}
	value := func(v contractstatemir.StableStateValue) {
		s(v.TypeID)
		n(uint64(len(v.Encoded)))
		out.Write(v.Encoded)
	}
	mode := 0
	root := body.Transactions[0]
	if len(root.Helpers) != 0 {
		mode = 1
	}
	borrow := func(blocks []contractstatemir.StableStateBlock) {
		for _, b := range blocks {
			for _, op := range b.Operations {
				if op.BorrowInput != 0 || op.BorrowResult != 0 || op.ViewOffset != 0 || op.ViewLength != 0 {
					mode = 2
				}
			}
		}
	}
	borrow(root.Blocks)
	for _, h := range root.Helpers {
		borrow(h.Blocks)
	}
	op := func(o contractstatemir.StableStateOperation) {
		s(o.Kind)
		n(uint64(o.FieldIndex))
		value(o.Key)
		value(o.Value)
		for _, v := range []uint32{o.KeyInput, o.ValueInput, o.Result, o.Found, o.Cost} {
			n(uint64(v))
		}
		if mode != 0 {
			s(o.Callee)
		}
		if mode == 2 {
			for _, v := range []uint32{o.BorrowInput, o.BorrowResult, o.ViewOffset, o.ViewLength} {
				n(uint64(v))
			}
		}
	}
	executable := func(effects []string, entry uint32, blocks []contractstatemir.StableStateBlock) {
		ss(effects)
		if mode == 0 {
			s("graph-v1")
		} else {
			s("graph")
		}
		n(uint64(entry))
		n(uint64(len(blocks)))
		for _, b := range blocks {
			n(uint64(b.ID))
			n(uint64(len(b.Parameters)))
			for _, p := range b.Parameters {
				n(uint64(p.ID))
				s(p.TypeID)
			}
			n(uint64(len(b.Operations)))
			for _, o := range b.Operations {
				op(o)
			}
			term := b.Terminator
			s(term.Kind)
			n(uint64(term.Condition))
			n(uint64(term.Target))
			ids(term.Arguments)
			n(uint64(term.TrueTarget))
			ids(term.TrueArguments)
			n(uint64(term.FalseTarget))
			ids(term.FalseArguments)
			s(term.TrapCategory)
			n(uint64(term.Cost))
		}
	}
	out.WriteString("fairyc.contract-state-package-body\x00\x01")
	out.Write(body.Package[:])
	out.WriteByte(1)
	out.Write(layout[:])
	n(1)
	n(uint64(len(root.Function)))
	out.Write(root.Function)
	if mode == 1 {
		s("helpers-v1")
	} else if mode == 2 {
		s("borrows-v1")
	}
	executable(root.Effects, root.Entry, root.Blocks)
	if mode != 0 {
		n(uint64(len(root.Helpers)))
		for _, h := range root.Helpers {
			s(h.Name)
			executable(h.Effects, h.Entry, h.Blocks)
		}
	}
	return out.Bytes()
}

func (f stateBodyFixture) wire() []byte {
	var out bytes.Buffer
	out.WriteString("FYSBTEST")
	out.Write(f.coordinate[:])
	_ = binary.Write(&out, binary.BigEndian, uint64(len(f.descriptor)))
	out.Write(f.descriptor)
	out.Write(f.packet.bytes())
	return out.Bytes()
}

func stateGraphBodyFixtures(t *testing.T) []stateBodyFixture {
	t.Helper()
	coordinate := packagegraph.CoordinateID(sha256.Sum256([]byte("independent state graph body coordinate")))
	var fixtures []stateBodyFixture
	add := func(name string, p contractstatemir.Program) {
		fixtures = append(fixtures, stateGraphBodyFixture(t, name, stateGraphBodyHIR(p), coordinate))
	}
	add("graph-typed-join", stateGraphIRBase(t))
	for _, trial := range stateGraphHelperTrials(t) {
		if trial.want {
			add("helper-"+trial.name, trial.p)
		}
	}
	for _, trial := range stateGraphBorrowTrials(t) {
		if trial.want && trial.body == 0 {
			add("borrow-"+trial.name, trial.program)
		}
	}
	base := fixtures[0]
	for _, mutation := range []struct {
		name   string
		mutate func(*hir.Package)
	}{
		{"root-name", func(p *hir.Package) { p.Modules[0].Functions[0].Name = "changed" }},
		{"root-parameter", func(p *hir.Package) {
			p.Modules[0].Functions[0].Parameters = []hir.Parameter{{Name: "input", Type: hir.TypeU32}}
		}},
		{"literal", func(p *hir.Package) { p.StateTransactions[0].Blocks[1].Accesses[0].Key.Encoded = []byte{0, 0, 0, 8} }},
		{"exit", func(p *hir.Package) { p.StateTransactions[0].Blocks[3].Terminator.Kind = hir.StateTerminatorReject }},
		{"trap-category", func(p *hir.Package) {
			p.StateTransactions[0].Blocks[3].Terminator.Kind = hir.StateTerminatorTrap
			p.StateTransactions[0].Blocks[3].Terminator.TrapCategory = "body-change"
		}},
	} {
		input := stateGraphBodyHIR(stateGraphIRBase(t))
		mutation.mutate(&input)
		f := stateGraphBodyFixture(t, "change-"+mutation.name, input, coordinate)
		if f.hash == base.hash {
			t.Fatalf("%s did not affect hash", mutation.name)
		}
		fixtures = append(fixtures, f)
	}
	input := stateGraphBodyHIR(stateGraphIRBase(t))
	other := coordinate
	other[0] ^= 1
	f := stateGraphBodyFixture(t, "change-coordinate", input, other)
	if f.hash == base.hash {
		t.Fatal("coordinate did not affect hash")
	}
	fixtures = append(fixtures, f)
	// The packet uses private numeric capability 1. Renumber every owner/op to
	// another nonzero identity without changing the portable bytes.
	renumbered := stateGraphBodyFixture(t, "renumbered-capability", stateGraphBodyHIR(stateGraphHelperProgram(t, "diamond")), coordinate)
	for _, graph := range renumbered.packet.graphs {
		stateGraphHelperWord(graph, 1, 999)
		parameters := binary.BigEndian.Uint64(graph[8+4*8:])
		arguments := binary.BigEndian.Uint64(graph[8+5*8:])
		operations := binary.BigEndian.Uint64(graph[8+6*8:])
		start := 80 + int(parameters)*16 + int(arguments)*8
		for i := 0; i < int(operations); i++ {
			binary.BigEndian.PutUint64(graph[start+i*19*8+8:], 999)
		}
	}
	fixtures = append(fixtures, renumbered)
	arena := stateGraphBodyFixture(t, "relocated-literal-arena", stateGraphBodyHIR(stateGraphIRBase(t)), coordinate)
	for gi, graph := range arena.packet.graphs {
		parameters := int(binary.BigEndian.Uint64(graph[8+4*8:]))
		arguments := int(binary.BigEndian.Uint64(graph[8+5*8:]))
		operations := int(binary.BigEndian.Uint64(graph[8+6*8:]))
		blocks := int(binary.BigEndian.Uint64(graph[8+7*8:]))
		start := 80 + parameters*16 + arguments*8
		for i := 0; i < operations; i++ {
			for _, codec := range []int{4, 8} {
				at := start + i*19*8 + codec*8
				if binary.BigEndian.Uint64(graph[at:]) != 0 {
					for _, off := range []int{at + 8, at + 16} {
						binary.BigEndian.PutUint64(graph[off:], binary.BigEndian.Uint64(graph[off:])+3)
					}
				}
			}
		}
		dataStart := start + operations*19*8 + blocks*16*8
		for i := 0; i < blocks; i++ {
			at := start + operations*19*8 + i*16*8 + 13*8
			if binary.BigEndian.Uint64(graph[at+8:]) > binary.BigEndian.Uint64(graph[at:]) {
				binary.BigEndian.PutUint64(graph[at:], binary.BigEndian.Uint64(graph[at:])+3)
				binary.BigEndian.PutUint64(graph[at+8:], binary.BigEndian.Uint64(graph[at+8:])+3)
			}
		}
		binary.BigEndian.PutUint64(graph[8+8*8:], binary.BigEndian.Uint64(graph[8+8*8:])+3)
		arena.packet.graphs[gi] = append(append(slices.Clone(graph[:dataStart]), 0xe1, 0, 0xff), graph[dataStart:]...)
	}
	fixtures = append(fixtures, arena)
	// Original HIR producer order and local symbol identities are irrelevant.
	reordered := stateGraphBodyHIR(stateGraphHelperProgram(t, "diamond"))
	slices.Reverse(reordered.Modules)
	slices.Reverse(reordered.StateTransactions)
	fixtures = append(fixtures, stateGraphBodyFixture(t, "reordered-authored-helpers", reordered, coordinate))
	for _, mutation := range []struct {
		name   string
		shape  string
		mutate func(*hir.Package)
	}{
		{"helper-name-and-callee", "leaf", func(p *hir.Package) { p.Modules[1].Functions[0].Name = "renamed" }},
		{"helper-literal", "leaf", func(p *hir.Package) { p.StateTransactions[1].Blocks[0].Accesses[0].Value.Encoded = []byte{0, 0, 0, 8} }},
		{"borrow-id", "borrow", func(p *hir.Package) {
			ops := p.StateTransactions[0].Blocks[0].Accesses
			ops[0].BorrowResult = 17
			ops[1].BorrowInput = 17
		}},
		{"view-offset", "borrow", func(p *hir.Package) { p.StateTransactions[0].Blocks[0].Accesses[1].ViewOffset++ }},
		{"view-length", "borrow", func(p *hir.Package) { p.StateTransactions[0].Blocks[0].Accesses[1].ViewLength++ }},
	} {
		var before hir.Package
		if mutation.shape == "borrow" {
			before = stateGraphBodyHIR(stateGraphBorrowBase(t, false))
		} else {
			before = stateGraphBodyHIR(stateGraphHelperProgram(t, mutation.shape))
		}
		original := stateGraphBodyFixture(t, "before-"+mutation.name, before, coordinate)
		mutation.mutate(&before)
		changed := stateGraphBodyFixture(t, "change-"+mutation.name, before, coordinate)
		if original.hash == changed.hash {
			t.Fatalf("%s preserved hash", mutation.name)
		}
		fixtures = append(fixtures, changed)
	}
	return fixtures
}

func stateGraphBodyProbe() string {
	probe := stateGraphHelperProbe()
	probe = probe[:strings.Index(probe, "@effects(console) fn i32 main()")]
	return strings.Replace(probe, "import module_state_graph_program as program;", "import module_state_graph_body_hash as body;import module_state_graph_hash_context as identity;import module_state_graph_program as program;", 1) + stateGraphBodyMain
}

const stateGraphBodyMain = `
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>4194304-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);var usize work=0;var c=switch(wire::cursor(raw,b"FYSBTEST",ref mut work)){case .ok(c):{c}case .err(code):{discard io::write_u32(code);return 7;}};
 if(raw.len<48){discard io::write("1001");return 7;}c.offset=40;let size=number(raw,ref mut c,ref mut work);if(c.code!=0 || size>raw.len-c.offset){discard io::write("1001");return 7;}
 let descriptor=raw[c.offset..c.offset+size];let coordinate=raw[8..40];c.offset+=size;work=0;
 let context=switch(identity::validate(coordinate,descriptor,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 7;}};let identity_cost=work;
 work=16777216-identity_cost;let exact=switch(identity::validate(coordinate,descriptor,ref mut work)){case .ok(p):{p}case .err(_):{return 31;}};if(work!=16777216){return 32;}
 work=16777216-identity_cost+1;switch(identity::validate(coordinate,descriptor,ref mut work)){case .ok(_):{return 33;}case .err(code):{if(code!=1002 || work>16777216){return 34;}}};
 work=16777217;switch(identity::validate(coordinate,descriptor,ref mut work)){case .ok(_):{return 35;}case .err(code):{if(code!=1002 || work!=16777217){return 36;}}};
 work=0;let g=switch(read_program(raw[c.offset..],ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 7;}};
 let layouts=switch(layout()){case .ok(p):{p}case .err(_):{return 37;}};work=0;let proof=switch(program::verify(g,ref layouts,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 7;}};
 // Both factories own their inputs. Drop the private byte transport before use.
 input=bytes::empty();work=0;let result=switch(body::derive(ref proof,ref context,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 7;}};let cost=work;
 work=16777216-cost;let exact_result=switch(body::derive(ref proof,ref exact,ref mut work)){case .ok(p):{p}case .err(_):{return 38;}};if(work!=16777216 || !bytes::equal(body::canonical(ref result),body::canonical(ref exact_result)) || !bytes::equal(body::digest(ref result),body::digest(ref exact_result))){return 39;}
 work=16777216-cost+1;switch(body::derive(ref proof,ref context,ref mut work)){case .ok(_):{return 40;}case .err(code):{if(code!=1002 || work>16777216){return 41;}}};
 work=16777217;switch(body::derive(ref proof,ref context,ref mut work)){case .ok(_):{return 42;}case .err(code):{if(code!=1002 || work!=16777217){return 43;}}};
 if(!io::write_bytes(body::digest(ref result)).is_ok || !io::write("\n").is_ok || !io::write_bytes(body::canonical(ref result)).is_ok){return 44;}return 0;
}`

func TestCandidateStateGraphBodyHash(t *testing.T) {
	fixtures := stateGraphBodyFixtures(t)
	fixtures = append(fixtures, stateGraphBodyTypedContexts(t, fixtures[0])...)
	candidateAtlasProbeSize(t, "module_state_graph_body_hash", stateGraphBodyProbe(), 512<<10, func(t *testing.T, image string) {
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, f.wire())
				if code != 0 || out != f.hash+"\n"+string(f.canonical) {
					t.Fatalf("code=%d bytes=%d want=%d; first mismatch at %d; prefix=%q", code, len(out), len(f.hash)+1+len(f.canonical), stateBodyFirstMismatch([]byte(out), append([]byte(f.hash+"\n"), f.canonical...)), out[:min(len(out), 100)])
				}
			})
		}
		for _, trial := range stateGraphBodyBadDescriptors(fixtures[0]) {
			t.Run("context-"+trial.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, trial.fixture.wire())
				if code != 7 || out != "1001" {
					t.Fatalf("malformed descriptor: code=%d output=%q", code, out)
				}
			})
		}
	}, "module_state_declarations", "candidate_artifact_wire")
}

func TestCandidateStateGraphBodyHashOracle(t *testing.T) {
	fixtures := stateGraphBodyFixtures(t)
	t.Logf("%d independent authored HIR -> Go hash/stable artifact -> lowered graph fixtures", len(fixtures))
	base := fixtures[0]
	reordered := stateGraphBodyHIR(stateGraphHelperProgram(t, "diamond"))
	slices.Reverse(reordered.Modules)
	slices.Reverse(reordered.StateTransactions)
	original := stateGraphBodyFixture(t, "ordered", stateGraphBodyHIR(stateGraphHelperProgram(t, "diamond")), base.coordinate)
	changed := stateGraphBodyFixture(t, "reordered", reordered, base.coordinate)
	if original.hash != changed.hash {
		t.Fatal("authored helper order leaked into portable identity")
	}
	if !bytes.Contains(base.canonical, []byte("graph-v1")) {
		t.Fatal("graph-v1 oracle marker missing")
	}
	coords := map[hir.PackageID]packagegraph.CoordinateID{7: base.coordinate}
	bodies, err := contractstatemir.DescribeStableHIRStateBodies(base.input, []hir.PackageID{7}, coords)
	if err != nil {
		t.Fatal(err)
	}
	lowered, err := contractstatemir.LowerHIRTransactions(base.input)
	if err != nil {
		t.Fatal(err)
	}
	for _, trial := range stateGraphBodyBadDescriptors(base) {
		t.Run("descriptor-refusal-"+trial.name, func(t *testing.T) {
			body := bodies[7]
			body.Package = trial.fixture.coordinate
			body.Transactions = slices.Clone(body.Transactions)
			body.Transactions[0].Function = trial.fixture.descriptor
			body.Hash = contractstatemir.StateBodyHash(sha256.Sum256(stateGraphBodyCanonical(body, lowered[0].Program.Layout.Hash)))
			if err := contractstatemir.ValidateStableStateBody(body); err == nil {
				t.Fatal("independent Go stable body validator admitted invalid identity context")
			}
		})
	}
	if stateGraphBodyFixture(t, "effect-read", stateGraphBodyHIR(stateGraphHelperProgram(t, "read-only")), base.coordinate).hash == stateGraphBodyFixture(t, "effect-write", stateGraphBodyHIR(stateGraphHelperProgram(t, "leaf")), base.coordinate).hash {
		t.Fatal("effects/body change preserved hash")
	}
}

func TestCandidateStateGraphBodyHashSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"construct-identity", "E0103", "fn void forge(){let p=body::Identity{.canonical=bytes::empty(),.digest=bytes::empty()};}"},
		{"construct-context", "E0103", "fn void forge(){let p=identity::Context{.coordinate=bytes::empty(),.descriptor=bytes::empty(),.effects=0};}"},
		{"mutate-canonical", "E1008", "fn void forge(ref body::Identity p){let ref mut u8[] raw=ref mut body::canonical(p);}"},
		{"mutate-descriptor", "E1008", "fn void forge(ref identity::Context p){let ref mut u8[] raw=ref mut identity::descriptor(p);}"},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_body_hash")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_body_hash as body;import module_state_graph_hash_context as identity;import std::bytes as bytes;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed body identity", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), trial.code) {
				t.Fatalf("seal failed outside intended refusal: %d %s", code, &diagnostics)
			}
		})
	}
}

func stateBodyFirstMismatch(a, b []byte) int {
	for i := 0; i < min(len(a), len(b)); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

func stateBodyDescriptor(name string, params []string, result string, effects []string, drop string) []byte {
	var out bytes.Buffer
	n := func(v uint64) { _ = binary.Write(&out, binary.BigEndian, v) }
	s := func(v string) { n(uint64(len(v))); out.WriteString(v) }
	s(name)
	n(uint64(len(params)))
	for _, p := range params {
		s(p)
	}
	s(result)
	n(uint64(len(effects)))
	for _, effect := range effects {
		s(effect)
	}
	s(drop)
	return out.Bytes()
}

// Descriptor grammar is wider than executable state operations. Validate these
// contexts through the Go stable-artifact reader; do not claim HIR execution
// support merely because a portable signature is canonically spelled.
func stateGraphBodyTypedContexts(t *testing.T, base stateBodyFixture) []stateBodyFixture {
	t.Helper()
	coords := map[hir.PackageID]packagegraph.CoordinateID{7: base.coordinate}
	artifacts, err := contractstatemir.DescribeStableHIRStateBodies(base.input, []hir.PackageID{7}, coords)
	if err != nil {
		t.Fatal(err)
	}
	lowered, err := contractstatemir.LowerHIRTransactions(base.input)
	if err != nil {
		t.Fatal(err)
	}
	nominal := "pkg[" + base.coordinate.String() + "]::types::Account"
	types := []string{"void", "bool", "u8", "i32", "u32", "usize", "str", "CStr", "Region", "c::int", "c::ulonglong", "c::double", "[bool]", "[u8]", "[i64]", "[i32]", "[u32]", "[usize]", "ref u32", "ref mut u32", "ptr c::int", "ptr mut c::char", "u32[1]", "u32[18446744073709551615]", "u32[2][3]", "Option<u32>", "Result<u32, bool>", "Option<Result<ref [u8], ptr mut c::int>>", nominal, nominal + "[]", "ref " + nominal, "Result<" + nominal + ", Option<u32>>", strings.Repeat("ref ", 256) + "u32"}
	var result []stateBodyFixture
	for i, typ := range types {
		body := artifacts[7]
		body.Transactions = slices.Clone(body.Transactions)
		descriptor := stateBodyDescriptor("main::transaction", []string{typ}, "void", []string{"storage_read", "storage_write"}, "")
		if i == 0 {
			descriptor = stateBodyDescriptor("main::transaction", nil, "void", []string{"storage_read", "storage_write"}, "ref u32")
		}
		body.Transactions[0].Function = descriptor
		canonical := stateGraphBodyCanonical(body, lowered[0].Program.Layout.Hash)
		body.Hash = contractstatemir.StateBodyHash(sha256.Sum256(canonical))
		if err := contractstatemir.ValidateStableStateBody(body); err != nil {
			t.Fatalf("Go descriptor type %q: %v", typ, err)
		}
		f := base
		f.name = fmt.Sprintf("typed-context-%02d", i)
		f.descriptor = descriptor
		f.canonical = canonical
		f.hash = body.Hash.String()
		result = append(result, f)
	}
	return result
}

type stateBodyBadDescriptor struct {
	name    string
	fixture stateBodyFixture
}

func stateGraphBodyBadDescriptors(base stateBodyFixture) []stateBodyBadDescriptor {
	var trials []stateBodyBadDescriptor
	add := func(name string, descriptor []byte) {
		f := base
		f.descriptor = descriptor
		trials = append(trials, stateBodyBadDescriptor{name, f})
	}
	effects := []string{"storage_read", "storage_write"}
	for _, name := range []string{"main", "main::", "main::9bad", "main::é", "main::transaction\x00"} {
		add(fmt.Sprintf("name-%q", name), stateBodyDescriptor(name, nil, "void", effects, ""))
	}
	for _, typ := range []string{"", "i64", "bool[]", "i8[]", "ref  u32", "u32[0]", "u32[01]", "u32[18446744073709551616]", "Result<u32,u32>", "Option<>", strings.Repeat("ref ", 257) + "u32", "pkg[fypc1-sha256-" + strings.Repeat("A", 64) + "]::types::T"} {
		add(fmt.Sprintf("type-%q", typ), stateBodyDescriptor("main::transaction", []string{typ}, "void", effects, ""))
	}
	for _, set := range [][]string{{"storage_write", "storage_read"}, {"storage_read", "storage_read"}, {"console"}, {"storage_read"}, nil} {
		add(fmt.Sprintf("effects-%v", set), stateBodyDescriptor("main::transaction", nil, "void", set, ""))
	}
	add("result", stateBodyDescriptor("main::transaction", nil, "missing", effects, ""))
	add("drop-glue", stateBodyDescriptor("main::transaction", nil, "void", effects, "missing"))
	add("trailing", append(slices.Clone(base.descriptor), 0))
	add("truncated", base.descriptor[:len(base.descriptor)-1])
	for _, at := range []int{0, len(base.descriptor) - 8} {
		bad := slices.Clone(base.descriptor)
		binary.BigEndian.PutUint64(bad[at:], ^uint64(0))
		add(fmt.Sprintf("overflow-length-%d", at), bad)
	}
	f := base
	f.coordinate = packagegraph.CoordinateID{}
	trials = append(trials, stateBodyBadDescriptor{"zero-coordinate", f})
	return trials
}

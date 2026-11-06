package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const stateGraphSourceSchema = `@contract_uid("fixture.authored-graph") state {cell bool flag;cell u32 count;cell u64 total;cell bytes20 account;cell bytes32 hash;cell bytes blob;cell text title;map u32 u32 values;map text bytes labels;}`

func stateGraphSourceText(body string) string {
	return "module ledger;\n" + stateGraphSourceSchema + "\n@effects(storage_read,storage_write)\nfn void transaction(){" + body + "}"
}

type stateGraphSourceTrial struct {
	name, text string
	code       string
	goValid    bool
}

func stateGraphSourceTrials() []stateGraphSourceTrial {
	var trials []stateGraphSourceTrial
	valid := func(name, body string) {
		trials = append(trials, stateGraphSourceTrial{name, stateGraphSourceText(body), "", true})
	}
	invalid := func(name, body string) {
		trials = append(trials, stateGraphSourceTrial{name, stateGraphSourceText(body), "1001", false})
	}
	valid("empty-commit", `storage::commit();`)
	valid("empty-reject", `storage::reject();`)
	valid("trap", `storage::trap("stale-state");`)
	valid("six-storage-actions", `storage::read("count");storage::write("count",42);storage::delete("count");storage::read("values",7);storage::write("values",7,99);storage::delete("values",7);storage::commit();`)
	for _, fixture := range []struct{ name, field, literal string }{
		{"false", "flag", "false"}, {"true", "flag", "true"}, {"u32-zero", "count", "0"}, {"u32-max-minus-one", "count", "4294967294"}, {"u32-max", "count", "4294967295"},
		{"u64-zero", "total", "0"}, {"u64-max-minus-one", "total", "18446744073709551614"}, {"u64-max", "total", "18446744073709551615"}, {"bytes20", "account", `b"` + strings.Repeat("a", 20) + `"`},
		{"bytes32", "hash", `b"` + strings.Repeat("b", 32) + `"`}, {"bytes-empty", "blob", `b""`}, {"bytes-binary", "blob", `b"\x00\xff"`},
		{"text-empty", "title", `""`}, {"text-unicode", "title", `"한글\n\""`},
	} {
		valid(fixture.name, fmt.Sprintf(`storage::write(%q,%s);storage::read(%q);storage::delete(%q);storage::commit();`, fixture.field, fixture.literal, fixture.field, fixture.field))
	}
	valid("map-text-empty-bytes", `storage::write("labels","",b"");storage::read("labels","한글");storage::delete("labels","other");storage::reject();`)
	valid("escaped-field", `storage::write("co\u{75}nt",1);storage::commit();`)
	valid("distinct-read-ids", `storage::read("count");storage::read("flag");storage::read("values",1);storage::commit();`)
	valid("operations-1024", strings.Repeat(`storage::delete("count");`, 1024)+`storage::commit();`)
	valid("reads-1024", strings.Repeat(`storage::read("count");`, 1024)+`storage::commit();`)
	for _, fixture := range []struct{ name, body string }{
		{"u32-overflow", `storage::write("count",4294967296);storage::commit();`},
		{"u64-overflow", `storage::write("total",18446744073709551616);storage::commit();`},
		{"integer-negative", `storage::write("count",-1);storage::commit();`},
		{"integer-float", `storage::write("count",1.0);storage::commit();`},
		{"integer-string", `storage::write("count","1");storage::commit();`},
		{"hex-integer-closed", `storage::write("count",0x1);storage::commit();`},
		{"binary-integer-closed", `storage::write("count",0b1);storage::commit();`},
		{"separator-integer-closed", `storage::write("count",1_000);storage::commit();`},
		{"bool-number", `storage::write("flag",1);storage::commit();`},
		{"bytes20-short", `storage::write("account",b"short");storage::commit();`},
		{"bytes32-long", `storage::write("hash",b"` + strings.Repeat("x", 33) + `");storage::commit();`},
		{"bytes-text", `storage::write("blob","text");storage::commit();`},
		{"text-bytes", `storage::write("title",b"text");storage::commit();`},
		{"unknown-field", `storage::read("missing");storage::commit();`},
		{"dynamic-field", `storage::read(count);storage::commit();`},
		{"cell-key", `storage::read("count",1);storage::commit();`},
		{"map-no-key", `storage::read("values");storage::commit();`},
		{"map-key-type", `storage::read("values",true);storage::commit();`},
		{"write-no-value", `storage::write("count");storage::commit();`},
		{"write-extra-operand", `storage::write("values",1,2,3);storage::commit();`},
		{"generic-call-closed", `storage::read<u32>("count");storage::commit();`},
		{"labeled-operand", `storage::write(.field="count",.value=1);storage::commit();`},
		{"exit-extra-operand", `storage::commit(1);`}, {"reject-extra-operand", `storage::reject(1);`},
		{"trap-empty", `storage::trap("");`}, {"trap-invalid", `storage::trap("Bad_category");`},
		{"trap-no-operand", `storage::trap();`}, {"trap-bytes", `storage::trap(b"bad");`},
		{"missing-exit", `storage::delete("count");`}, {"after-exit", `storage::commit();storage::delete("count");`},
		{"root-return", `return;`}, {"unknown-call", `other();storage::commit();`},
	} {
		invalid(fixture.name, fixture.body)
	}
	base := stateGraphSourceText(`storage::write("count",1);storage::commit();`)
	for _, fixture := range []struct {
		name, text string
		goValid    bool
	}{
		{"public-root", strings.Replace(base, "fn void", "pub fn void", 1), false},
		{"unsafe-root", strings.Replace(base, "fn void", "unsafe fn void", 1), false},
		{"named-owner", strings.Replace(base, "transaction()", "Foo::transaction()", 1), false},
		{"generic-root", strings.Replace(base, "transaction()", "transaction<T>()", 1), false},
		{"parameter-root", strings.Replace(base, "transaction()", "transaction(u32 x)", 1), false},
		{"wrong-result", strings.Replace(base, "fn void", "fn u32", 1), false},
		{"wrong-owner-name", strings.Replace(base, "transaction()", "execute()", 1), false},
		{"missing-write-effect", strings.Replace(base, "storage_read,storage_write", "storage_read", 1), false},
		{"foreign-effect", strings.Replace(base, "storage_read,storage_write", "console,storage_write", 1), false},
		{"duplicate-effect", strings.Replace(base, "storage_read,storage_write", "storage_write,storage_write", 1), false},
		{"unknown-attribute", strings.Replace(base, "@effects", "@other", 1), false},
		{"extra-declaration", base + "const u32 extra=1;", false},
		{"second-state", base + `@contract_uid("other")state{cell bool f;}`, false},
		{"duplicate-field", strings.Replace(base, "cell u32 count;", "cell u32 count;cell u32 count;", 1), false},
		{"bad-layout-codec", strings.Replace(base, "cell u32 count;", "cell i32 count;", 1), false},
		{"duplicate-uid", strings.Replace(base, "state {", `@contract_uid("other") state {`, 1), false},
	} {
		trials = append(trials, stateGraphSourceTrial{fixture.name, fixture.text, "1001", fixture.goValid})
	}
	for _, fixture := range []struct{ name, text string }{
		{"root-before-state", "module ledger;@effects(storage_write)fn void transaction(){storage::write(\"count\",1);storage::commit();}" + stateGraphSourceSchema},
		{"comments-and-crlf", strings.ReplaceAll(strings.Replace(base, `storage::write`, "/* retained */ storage :: write", 1), "\n", "\r\n")},
		{"no-effects-empty-exit", "module ledger;" + stateGraphSourceSchema + "fn void transaction(){storage::commit();}"},
	} {
		trials = append(trials, stateGraphSourceTrial{fixture.name, fixture.text, "", true})
	}
	trials = append(trials,
		stateGraphSourceTrial{"operations-1025", stateGraphSourceText(strings.Repeat(`storage::delete("count");`, 1025) + `storage::commit();`), "1002", false},
		stateGraphSourceTrial{"source-65536", base + strings.Repeat(" ", 65536-len(base)), "", true},
		stateGraphSourceTrial{"source-65537", base + strings.Repeat(" ", 65537-len(base)), "1002", true},
		stateGraphSourceTrial{"malformed-source", base[:len(base)-1], "1001", false},
		stateGraphSourceTrial{"invalid-utf8", strings.Replace(base, "ledger", "led\xffger", 1), "1001", false},
	)
	return trials
}

func stateGraphSourceOracle(t *testing.T, trial stateGraphSourceTrial) string {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("ledger.fy", []byte(trial.text)), source.DefaultLimits())
	valid := false
	var result compiler.Result
	if err == nil && parsed.AST != nil && len(parsed.Tree.Diagnostics) == 0 {
		result, err = compiler.CheckExperimentalStateGraph(semantic.File{Package: hir.DefaultPackage, Path: "ledger.fy", Module: parsed.AST})
		valid = err == nil && result.Verified
	}
	if valid != trial.goValid {
		t.Fatalf("Go source admission=%v want=%v err=%v syntax=%+v semantic=%+v unsupported=%+v", valid, trial.goValid, err, parsed.Tree.Diagnostics, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	if trial.code != "" {
		return trial.code
	}
	if len(result.StateTransactions) != 1 {
		t.Fatal("missing source transaction")
	}
	p := result.StateTransactions[0].Program
	if failures := contractstatemir.Verify(p); len(failures) != 0 {
		t.Fatal(failures)
	}
	f := parsed.AST.Functions[0]
	var out strings.Builder
	fmt.Fprintf(&out, "ok\nowner 0 0 %d 1 %d %d %d %d ", 3, f.Span.StartByte, f.Span.EndByte, f.Body.Span.StartByte, f.Body.Span.EndByte)
	if f.Span.StartByte < parsed.AST.States[0].Span.StartByte { // private root before layout
		out.Reset()
		fmt.Fprintf(&out, "ok\nowner 0 0 2 1 %d %d %d %d ", f.Span.StartByte, f.Span.EndByte, f.Body.Span.StartByte, f.Body.Span.EndByte)
	}
	if len(f.Attributes) == 0 {
		out.WriteString("0 0\n")
	} else {
		fmt.Fprintf(&out, "%d %d\n", f.Attributes[0].Span.StartByte, f.Attributes[0].Span.EndByte)
	}
	fmt.Fprintf(&out, "body %x\n", []byte(trial.text)[f.Body.Span.StartByte:f.Body.Span.EndByte])
	mask := 0
	for _, e := range p.Effects {
		if e == hir.EffectStorageRead {
			mask |= 1
		}
		if e == hir.EffectStorageWrite {
			mask |= 2
		}
	}
	fmt.Fprintf(&out, "graph %d %d\n", mask, len(p.Blocks[0].Operations))
	fmt.Fprintf(&out, "uid %x\n", p.Layout.ContractUID)
	writeHash := func(hash []byte) {
		for i := 0; i < 8; i++ {
			fmt.Fprintf(&out, "%d ", binary.BigEndian.Uint32(hash[i*4:]))
		}
		out.WriteByte('\n')
	}
	out.WriteString("hash ")
	writeHash(p.Layout.Hash[:])
	for _, field := range p.Layout.Fields {
		fmt.Fprintf(&out, "field %d %x %s %s ", field.Kind, field.Name, field.KeyTypeID, field.ValueTypeID)
		writeHash(field.Prefix[:])
	}
	kinds := map[contractstatemir.OperationKind]int{contractstatemir.OperationCellRead: 1, contractstatemir.OperationCellWrite: 2, contractstatemir.OperationCellDelete: 3, contractstatemir.OperationMapRead: 4, contractstatemir.OperationMapWrite: 5, contractstatemir.OperationMapDelete: 6}
	for i, op := range p.Blocks[0].Operations {
		span := f.Body.Statements[i].Expression.Span
		fmt.Fprintf(&out, "op %d %d %d %d %d %s:%x %s:%x %d %d\n", kinds[op.Kind], op.FieldIndex, op.Result, op.Found, op.Cost, op.Key.TypeID, op.Key.Encoded, op.Value.TypeID, op.Value.Encoded, span.StartByte, span.EndByte)
	}
	exit := f.Body.Statements[len(f.Body.Statements)-1].Span
	term := p.Blocks[0].Terminator
	kind := map[contractstatemir.TerminatorKind]int{contractstatemir.TerminatorCommit: 3, contractstatemir.TerminatorReject: 4, contractstatemir.TerminatorTrap: 5}[term.Kind]
	fmt.Fprintf(&out, "exit %d %d %x %d %d", kind, term.Cost, term.TrapCategory, exit.StartByte, exit.EndByte)
	return out.String()
}

const stateGraphSourceProbe = `module main;
import module_state_graph_source as authored;import module_state_graph_model as m;import module_state_graph_verify as core;import module_state_layout as state;import literal_types as literals;import std::bytes as bytes;import std::vec as v;import std::io as io;
@effects(console)fn void n(usize x){discard io::write_usize(x);discard io::write(" ");}
@effects(console)fn void hex(ref u8[] x){var bytes::Buffer out=bytes::empty();assert(literals::hex(ref mut out,x));discard io::write_bytes(bytes::view(ref out));}
@effects(console)fn void hash(u32[8] x){var usize i=0;while(i<8){n(x[i] as usize);i+=1;}discard io::write("\n");}
@effects(console)fn void operand(m::Operand op,ref u8[] data){discard io::write_bytes(state::codec(op.codec));discard io::write(":");hex(data[op.start..op.end]);}
@effects(console)fn i32 main(){var bytes::Buffer input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let count=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 2;}};if(count==0){break;}let ref u8[] view=ref chunk;if(bytes::len(ref input)>70000-count || !bytes::append(ref mut input,view[..count]).is_ok){return 3;}}
 var usize work=0;let proof=switch(authored::produce(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};let cost=work;
 if(!authored::matches(ref proof,bytes::view(ref input)) || !bytes::equal(authored::source(ref proof),bytes::view(ref input))){return 30;}
 work=16777216-cost;let exact=switch(authored::produce(bytes::view(ref input),ref mut work)){case .ok(p):{p}case .err(_):{return 31;}};if(work!=16777216 || !bytes::equal(authored::body(ref exact),authored::body(ref proof))){return 32;}
 work=16777216-cost+1;let u32 short=switch(authored::produce(bytes::view(ref input),ref mut work)){case .ok(_):{0}case .err(code):{code}};if(short!=1002 || work>16777216){return 33;}
 work=16777217;let u32 overflow=switch(authored::produce(bytes::view(ref input),ref mut work)){case .ok(_):{0}case .err(code):{code}};if(overflow!=1002 || work!=16777217){return 34;}
 {let changed=bytes::view_mut(ref mut input);changed[0]=99;}if(authored::matches(ref proof,bytes::view(ref input)) || authored::source(ref proof)[0]!=109){return 35;}
 work=0;if(core::validate(authored::graph(ref proof),authored::layouts(ref proof),ref mut work)!=0){return 36;}
 let owner=authored::owner(ref proof);discard io::write("ok\nowner ");n(owner.module_index);n(owner.package);n(owner.declaration);n(owner.identity);n(owner.declaration_span.start);n(owner.declaration_span.end);n(owner.body_span.start);n(owner.body_span.end);n(owner.effects_span.start);discard io::write_usize(owner.effects_span.end);discard io::write("\nbody ");hex(authored::body(ref proof));
 let graph=authored::graph(ref proof);let layouts=authored::layouts(ref proof);let h=m::header(graph);discard io::write("\ngraph ");n(h.effects as usize);discard io::write_usize(v::len<m::Operation>(m::operations(graph)));discard io::write("\nuid ");hex(state::uid(layouts,0));discard io::write("\nhash ");hash(state::at(layouts,0).hash);
 var usize i=0;while(i<state::field_count(layouts)){let f=state::field_at(layouts,i);discard io::write("field ");n(f.kind as usize);hex(state::name(layouts,i));discard io::write(" ");discard io::write_bytes(state::codec(f.key));discard io::write(" ");discard io::write_bytes(state::codec(f.value));discard io::write(" ");hash(f.prefix);i+=1;}
 i=0;while(i<v::len<m::Operation>(m::operations(graph))){let op=*v::at<m::Operation>(m::operations(graph),i);if(op.capability!=owner.identity){return 37;}let span=authored::operation_span(ref proof,i);discard io::write("op ");n(op.kind as usize);n(op.field);n(op.result);n(op.found);n(op.cost);operand(op.key,m::data(graph));discard io::write(" ");operand(op.value,m::data(graph));discard io::write(" ");n(span.start);discard io::write_usize(span.end);discard io::write("\n");i+=1;}
 let term=v::at<m::Block>(m::blocks(graph),0).term;let exit=authored::exit_span(ref proof);discard io::write("exit ");n(term.kind as usize);n(term.cost);hex(m::data(graph)[term.category_start..term.category_end]);discard io::write(" ");n(exit.start);discard io::write_usize(exit.end);return 0;}`

func TestCandidateStateGraphSourceOracle(t *testing.T) {
	for _, trial := range stateGraphSourceTrials() {
		t.Run(trial.name, func(t *testing.T) { stateGraphSourceOracle(t, trial) })
	}
}

func TestCandidateStateGraphSource(t *testing.T) {
	trials := stateGraphSourceTrials()
	wants := make([]string, len(trials))
	for i, trial := range trials {
		wants[i] = stateGraphSourceOracle(t, trial)
	}
	candidateAtlasProbeSize(t, "module_state_graph_source", stateGraphSourceProbe, 512<<10, func(t *testing.T, image string) {
		for i, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, []byte(trial.text))
				if code != 0 || out != wants[i] {
					t.Fatalf("exit=%d output=%q want=%q", code, out, wants[i])
				}
			})
		}
	})
}

func TestCandidateStateGraphSourceSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"private-constructor", "E0103", `fn authored::Produced forge(){return authored::Produced{.sources=0,.proof=0,.owner=0,.spans=0,.exit=0};}`},
		{"immutable-graph", "E1008", `fn void change(ref authored::Produced p){let ref mut m::Graph g=ref mut authored::graph(p);}`},
		{"immutable-layout", "E1008", `fn void change(ref authored::Produced p){let ref mut state::Plan s=ref mut authored::layouts(p);}`},
		{"immutable-source", "E1008", `fn void change(ref authored::Produced p){let ref mut u8[] s=ref mut authored::source(p);}`},
		{"immutable-body", "E1008", `fn void change(ref authored::Produced p){let ref mut u8[] s=ref mut authored::body(p);}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_source")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_source as authored;import module_state_graph_model as m;import module_state_layout as state;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed source owner", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), "main.fy:") || !strings.Contains(diagnostic.String(), trial.code) {
				t.Fatalf("sealed owner code=%d diagnostics=%s", code, &diagnostic)
			}
		})
	}
}

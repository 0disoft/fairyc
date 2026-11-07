package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

// Only actual source enters the candidate. All identity expectations are
// independently compiled in Go with the developer command's coordinate tuple.
const stateCommandSourceProbe = `module main;
import module_state_command_source as command;import project_compiler_core as core;import project_compiler_state_graph as graph;import module_state_graph_source as authored;import module_state_graph_source_identity as identity;import module_state_graph_hash_context as context;import module_state_graph_body_hash as hash;import module_project_interfaces as records;import module_public_interface as public;import std::bytes as bytes;import std::io as io;
@effects(console)fn void number(usize value){assert(io::write_usize(value).is_ok && io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok && io::write("\n").is_ok);}
@effects(console)fn i32 main(){
 var incoming=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(p):{p}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;assert(bytes::len(ref incoming)<=131072-n && bytes::append(ref mut incoming,raw[..n]).is_ok);}
 let original=switch(bytes::copy(bytes::view(ref incoming))){case .ok(p):{p}case .err(_):{return 2;}};
 var usize work=37;let complete=switch(command::prepare(bytes::view(ref incoming),ref mut work)){case .ok(p):{p}case .err(code):{assert(bytes::equal(bytes::view(ref incoming),bytes::view(ref original)));number(code as usize);number(work);return 0;}};
 assert(core::is_state_graph(ref complete) && graph::owner(core::checked_state_graph(ref complete))==0 && work>37 && work<=16777216);
 let cost=work-37;
 work=16777216-cost;switch(command::prepare(bytes::view(ref incoming),ref mut work)){case .ok(_):{}case .err(_):{return 3;}};assert(work==16777216);
 work=16777217-cost;switch(command::prepare(bytes::view(ref incoming),ref mut work)){case .ok(_):{return 4;}case .err(code):{assert(code==1002 && work<=16777216);}};
 work=16777216;switch(command::prepare(bytes::view(ref incoming),ref mut work)){case .ok(_):{return 5;}case .err(code):{assert(code==1002 && work==16777216);}};
 work=16777217;switch(command::prepare(bytes::view(ref incoming),ref mut work)){case .ok(_):{return 6;}case .err(code):{assert(code==1002 && work==16777217);}};
 assert(bytes::equal(bytes::view(ref incoming),bytes::view(ref original)));
 {let raw=bytes::view_mut(ref mut incoming);var usize i=0;while(i<raw.len){raw[i]=255;i+=1;}}incoming=bytes::empty();
 let checked=core::checked_state_graph(ref complete);let current=graph::source(checked);assert(authored::matches(current,bytes::view(ref original)));
 let bound=graph::body_identity(checked);let ctx=identity::context(bound);let body=identity::identity(bound);let all=core::public_interfaces(ref complete);assert(records::count(all)==1);let record=records::at(all,0);assert(records::owner(record)==0);
 var usize output_work=0;let encoded=switch(public::encode_graph(authored::layouts(current),authored::owner(current).package,records::descriptor(record),ref mut output_work)){case .ok(p):{p}case .err(_):{return 7;}};
 number(0);number(cost);blob(authored::source(current));blob(context::coordinate(ctx));blob(context::descriptor(ctx));blob(hash::canonical(body));blob(hash::digest(body));blob(records::ordinary_hash(record));blob(records::source_hash(record));blob(bytes::view(ref encoded));return 0;
}`

func TestCandidateStateCommandSource(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_state_command_source.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("module_state_command_source.fy sha256=%x; probe sha256=%x", sha256.Sum256(data), sha256.Sum256([]byte(stateCommandSourceProbe)))
	type trial struct {
		name, text string
		code       uint64
		want       [][]byte
	}
	var trials []trial
	for _, item := range graphTransportTrials() {
		switch item.name {
		case "empty-commit", "trap", "six-storage-actions", "helper-transitive", "map-local-key-value", "present-borrow-copy":
			trials = append(trials, trial{name: item.name, text: item.text})
		}
	}
	if len(trials) != 6 {
		t.Fatalf("representative source cases=%d want=6", len(trials))
	}
	base := trials[0].text
	trials = append(trials, trial{name: "changed-uid", text: strings.Replace(base, "fixture.authored-graph", "command.other-uid", 1)})
	if trials[len(trials)-1].text == base {
		t.Fatal("UID mutation did not change actual source")
	}
	trials = append(trials,
		trial{name: "source-65536", text: base + strings.Repeat(" ", 65536-len(base))},
		trial{name: "source-65537", text: base + strings.Repeat(" ", 65537-len(base)), code: 1002},
		trial{name: "empty", code: 1001},
		trial{name: "malformed", text: base[:len(base)-1], code: 1001},
		trial{name: "missing-uid", text: strings.Replace(base, `@contract_uid("fixture.authored-graph")`, "", 1), code: 1001},
		trial{name: "unknown-field", text: stateGraphAuthoredSource("storage_write", `storage::delete("missing");storage::commit();`, ""), code: 1001},
		trial{name: "invalid-utf8", text: strings.Replace(base, "ledger", "led\xffger", 1), code: 1001},
	)
	for i := range trials {
		item := &trials[i]
		if item.code != 0 {
			continue
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("ledger.fy", []byte(item.text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			t.Fatalf("actual source syntax %s: %v %+v", item.name, err, parsed.Tree.Diagnostics)
		}
		compiled, err := compiler.CheckExperimentalStateGraph(semantic.File{Package: 7, Path: "ledger.fy", Module: parsed.AST})
		if err != nil || !compiled.Verified || len(compiled.StateTransactions) != 1 {
			t.Fatalf("actual compiler %s: %v %+v", item.name, err, compiled.Semantic.Diagnostics)
		}
		owner := compiled.StateTransactions[0]
		coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: owner.Program.Layout.ContractUID, VersionIdentity: "experimental-state-graph-source/1"})
		if err != nil {
			t.Fatal(err)
		}
		fixture := stateGraphBodyFixture(t, item.name, compiled.Semantic.HIR, coordinate)
		handoff, err := contractstatenative.PrepareCompiled(compiled, owner.Package, owner.Function, coordinate)
		if err != nil {
			t.Fatal(err)
		}
		native, err := contractstatenative.EmitGraph(handoff, target.X8664WindowsMSVC)
		if err != nil || native.BodyHash().String() != fixture.hash || native.LayoutHash() != owner.Program.Layout.Hash {
			t.Fatalf("actual compiler/native handoff identity %s: %v", item.name, err)
		}
		descriptor, err := moduleinterface.Extract(compiled.Semantic.HIR, 7, map[hir.PackageID]packagegraph.CoordinateID{7: coordinate})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := moduleinterface.Encode(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		ordinary, err := moduleinterface.Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		sourceHash, err := moduleinterface.ComputeGeneric(descriptor, nil)
		if err != nil {
			t.Fatal(err)
		}
		item.want = [][]byte{[]byte(item.text), coordinate[:], fixture.descriptor, fixture.canonical, []byte(fixture.hash), []byte(ordinary.String()), []byte(sourceHash.String()), encoded}
	}
	candidateAtlasProbeSize(t, "module_state_command_source", stateCommandSourceProbe, 512<<10, func(t *testing.T, image string) {
		for _, item := range trials {
			t.Run(item.name, func(t *testing.T) {
				exit, output := candidateAtlasProcess(t, image, []byte(item.text))
				if exit != 0 {
					t.Fatalf("native source preparation exit=%d output=%q", exit, output)
				}
				r := graphResponseReader{t, bufio.NewReader(strings.NewReader(output))}
				code, work := r.number(), r.number()
				if code != item.code || work == 0 || work > 16777216 {
					t.Fatalf("source preparation code/work=%d/%d want code=%d", code, work, item.code)
				}
				for i, want := range item.want {
					if got := r.blob(); !bytes.Equal(got, want) {
						t.Fatalf("Go source/interface/body/native identity field=%d len=%d/%d mismatch=%d", i, len(got), len(want), stateBodyFirstMismatch(got, want))
					}
				}
				remaining, err := io.ReadAll(r.r)
				if err != nil || len(remaining) != 0 {
					t.Fatalf("unexpected output suffix=%q err=%v", remaining, err)
				}
				t.Logf("code=%d charged work=%d; exact/one-short/overdraw/raw-input/owned-output assertions included for valid source", code, work)
			})
		}
	})
}

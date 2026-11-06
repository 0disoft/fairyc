package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// The packet contains only source, selected coordinate and detached compatibility
// facts. The expected framing and SHA256 are independently authored in Go.
type graphCacheKeyTrial struct {
	name, source, coordinate, standard string
	environment                        [8]string
	owner, mode                        uint64
	wantError                          string
}

func graphCacheKeyPacket(p graphCacheKeyTrial) []byte {
	var b bytes.Buffer
	n := func(v uint64) { _ = binary.Write(&b, binary.BigEndian, v) }
	s := func(v string) { n(uint64(len(v))); b.WriteString(v) }
	n(p.mode)
	n(p.owner)
	s(p.coordinate)
	for _, value := range p.environment {
		s(value)
	}
	s(p.standard)
	b.WriteString(p.source)
	return b.Bytes()
}

func graphCacheKeyOracle(t *testing.T, p graphCacheKeyTrial) []byte {
	t.Helper()
	if p.wantError != "" {
		return []byte(p.wantError)
	}
	f, _ := stateGraphAuthoredOracle(t, stateGraphAuthoredTrial{name: p.name, text: p.source})
	coordinate, err := hex.DecodeString(strings.TrimPrefix(p.coordinate, "fypc1-sha256-"))
	if err != nil || len(coordinate) != 32 {
		t.Fatalf("coordinate: %v", err)
	}
	var id packagegraph.CoordinateID
	copy(id[:], coordinate)
	// Recompute body identity under the selected coordinate, not the fixture's
	// default. A numeric package owner is deliberately absent from the framing.
	f = stateGraphBodyFixture(t, p.name, f.input, id)
	descriptor, err := moduleinterface.Extract(f.input, 7, map[hir.PackageID]packagegraph.CoordinateID{7: id})
	if err != nil {
		t.Fatal(err)
	}
	public, err := moduleinterface.Compute(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	original, err := moduleinterface.ComputeGeneric(descriptor, nil)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	b.WriteString("fairyc.state-graph-cache-key\x00\x01")
	for _, value := range p.environment {
		_ = binary.Write(&b, binary.BigEndian, uint64(len(value)))
		b.WriteString(value)
	}
	decoded := func(text, prefix string) {
		raw, err := hex.DecodeString(strings.TrimPrefix(text, prefix))
		if err != nil || len(raw) != 32 {
			t.Fatalf("oracle identity %q: %v", text, err)
		}
		b.Write(raw)
	}
	decoded(p.standard, "fypi1-sha256-")
	b.Write(coordinate)
	source := sha256.Sum256([]byte(p.source))
	b.Write(source[:])
	decoded(f.hash, "fycsbody1-sha256-")
	decoded(public.String(), "fyif5-sha256-")
	decoded(original.String(), "fygif0-sha256-")
	hash := sha256.Sum256(b.Bytes())
	return append(bytes.Clone(b.Bytes()), []byte("\nfysgraphcache1-sha256-"+hex.EncodeToString(hash[:]))...)
}

const graphCacheKeyProbe = `module main;
import project_state_graph_cache_key as key;import project_compiler_core as core;import project_compiler_state_graph as graph;import project_cache_key as compatibility;import project_cache_environment as environment;import module_state_graph_source_artifact as artifact;import module_state_graph_source as authored;import module_public_type_text as coordinates;import module_catalog as catalog;import scanner as scanner;import syntax as syntax;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn bytes::Buffer text(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){let n=wire::count_from(raw,c,1048576,1,work);assert(c.code==0);let start=c.offset;c.offset+=n;return switch(bytes::copy(raw[start..c.offset])){case .ok(p):{p}case .err(_):{assert(false);text(raw,c,work)}};}
fn core::Complete ordinary(ref coordinates::Coordinates table,usize owner){
 let b=switch(bytes::copy(b"module ledger;fn i32 main(){return 0;}")){case .ok(p):{p}case .err(_):{assert(false);return ordinary(table,owner);}};
 let tokens=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return ordinary(table,owner);}};
 let parsed=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return ordinary(table,owner);}};
 var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed).is_ok);var owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,owner).is_ok);let imports=v::empty<catalog::Binding>();var usize source_work=0;var usize link_work=0;
 return switch(core::build_complete_sources(ref sources,ref owners,ref imports,table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(_):{assert(false);ordinary(table,owner)}};
}
fn core::Complete built(ref u8[] actual,ref coordinates::Coordinates table,usize owner){var usize work=0;return switch(core::build_complete_state_graph(actual,table,owner,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);built(actual,table,owner)}};}
@effects(filesystem,environment)fn Result<key::Identity,u32> check(ref u8[] raw){
 var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);
 var names=text(raw,ref mut c,ref mut setup);var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=bytes::len(ref names)}).is_ok);var table=coordinates::assemble(names,rows);
 let a=text(raw,ref mut c,ref mut setup);let b=text(raw,ref mut c,ref mut setup);let d=text(raw,ref mut c,ref mut setup);let e=text(raw,ref mut c,ref mut setup);let f=text(raw,ref mut c,ref mut setup);let g=text(raw,ref mut c,ref mut setup);let h=text(raw,ref mut c,ref mut setup);let j=text(raw,ref mut c,ref mut setup);
 var env=compatibility::environment_of(a,b,d,e,f,g,h,j);var standard=text(raw,ref mut c,ref mut setup);var actual=switch(bytes::copy(raw[c.offset..])){case .ok(p):{p}case .err(_):{return .err(1003);}};
 let current=if(mode==1 || mode==4){ordinary(ref table,owner)}else{built(bytes::view(ref actual),ref table,owner)};
 if(mode==2){var huge=bytes::empty();assert(bytes::resize_zeroed(ref mut huge,16777216).is_ok);env=compatibility::environment_of(huge,bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty());}
 var usize work=if(mode==3 || mode==4){16777217}else{0};let first=switch(key::compute(ref current,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(code):{if(mode==3 || mode==4){assert(work==16777217);}return .err(code);}};let cost=work;assert(cost>0 && cost<=16777216);
 if(mode==5){work=0;let actual_env=switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{return .err(9013);}};work=0;
  let snapshot=switch(environment::key_environment(ref actual_env,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9014);}};work=0;
  let computed=switch(key::compute(ref current,ref snapshot,environment::standard_library(ref actual_env),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9015);}};work=0;
  let derived=switch(key::derive(ref current,ref actual_env,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9016);}};let derive_cost=work;
  if(!bytes::equal(key::canonical(ref computed),key::canonical(ref derived)) || !bytes::equal(key::digest(ref computed),key::digest(ref derived))){return .err(9017);}
  work=16777216-derive_cost;switch(key::derive(ref current,ref actual_env,ref mut work)){case .ok(_):{}case .err(_):{return .err(9018);}};if(work!=16777216){return .err(9019);}
  work=16777217-derive_cost;switch(key::derive(ref current,ref actual_env,ref mut work)){case .ok(_):{return .err(9020);}case .err(code):{if(code!=1002 || work>16777216){return .err(9021);}}};
  work=16777217;switch(key::derive(ref current,ref actual_env,ref mut work)){case .ok(_):{return .err(9022);}case .err(code):{if(code!=1002 || work!=16777217){return .err(9023);}}};
 }
 work=16777216-cost;let exact=switch(key::compute(ref current,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9001);}};
 if(work!=16777216 || !bytes::equal(key::canonical(ref first),key::canonical(ref exact)) || !bytes::equal(key::digest(ref first),key::digest(ref exact))){return .err(9002);}
 work=16777217-cost;switch(key::compute(ref current,ref env,bytes::view(ref standard),ref mut work)){case .ok(_):{return .err(9003);}case .err(code):{if(code!=1002 || work>16777216){return .err(9004);}}};
 work=16777217;switch(key::compute(ref current,ref env,bytes::view(ref standard),ref mut work)){case .ok(_):{return .err(9005);}case .err(code):{if(code!=1002 || work!=16777217){return .err(9006);}}};
 // Current Complete owns both source and coordinate after caller buffers change.
 {let raw=bytes::view_mut(ref mut actual);raw[0]=99;}actual=bytes::empty();table=coordinates::assemble(bytes::empty(),v::empty<coordinates::Coordinate>());
 work=0;let stable=switch(key::compute(ref current,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9007);}};if(!bytes::equal(key::canonical(ref first),key::canonical(ref stable))){return .err(9008);}
 // Restore under a new numeric owner with the identical selected coordinate.
 let owned=core::checked_state_graph(ref current);let retained=authored::source(graph::source(owned));
 let usize coordinate_start=16;var cursor=wire::Cursor{.offset=coordinate_start,.code=0};let coordinate=text(raw,ref mut cursor,ref mut setup);var rebound=v::empty<coordinates::Coordinate>();let usize next=if(owner==511){0}else{owner+1};assert(v::push<coordinates::Coordinate>(ref mut rebound,coordinates::Coordinate{.owner=next,.start=0,.end=bytes::len(ref coordinate)}).is_ok);let other_table=coordinates::assemble(coordinate,rebound);
 work=0;let saved=switch(artifact::encode(graph::source(owned),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9009);}};work=0;
 let restored=switch(core::restore_complete_state_graph(wire::content(ref saved),retained,ref other_table,next,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9010);}};work=0;
 let replay=switch(key::compute(ref restored,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9011);}};
 if(!bytes::equal(key::canonical(ref first),key::canonical(ref replay)) || !bytes::equal(key::digest(ref first),key::digest(ref replay))){return .err(9012);}
 // Return detached bytes after Complete and compatibility owners are destroyed.
 env=compatibility::environment_of(bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty());standard=bytes::empty();return .ok(first);
}
@effects(filesystem,environment,console)fn i32 main(){var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>1048576-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let result=switch(check(bytes::view(ref input))){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};input=bytes::empty();
 if(!io::write_bytes(key::canonical(ref result)).is_ok || !io::write("\n").is_ok || !io::write_bytes(key::digest(ref result)).is_ok){return 3;}return 0;
}`

func TestCandidateStateGraphCacheKey(t *testing.T) {
	base := graphCacheKeyTrial{
		name: "base", owner: 7,
		source:      stateGraphAuthoredSource("storage_write", `storage::write("count",1);storage::commit();`, ""),
		coordinate:  "fypc1-sha256-" + strings.Repeat("1", 64),
		standard:    "fypi1-sha256-" + strings.Repeat("2", 64),
		environment: [8]string{"build", "language", "mir", "target", "abi", "toolchain", "backend", "runtime"},
	}
	trials := []graphCacheKeyTrial{base}
	add := func(name string, mutate func(*graphCacheKeyTrial)) {
		p := base
		p.name = name
		mutate(&p)
		trials = append(trials, p)
	}
	add("owner-rebound", func(p *graphCacheKeyTrial) { p.owner = 511 })
	add("source-comment", func(p *graphCacheKeyTrial) { p.source += "// retained comment\n" })
	add("source-body", func(p *graphCacheKeyTrial) { p.source = strings.Replace(p.source, "count\",1", "count\",2", 1) })
	add("source-interface", func(p *graphCacheKeyTrial) {
		p.source = strings.Replace(p.source, "cell bool flag", "cell bool enabled", 1)
	})
	add("coordinate", func(p *graphCacheKeyTrial) { p.coordinate = "fypc1-sha256-" + strings.Repeat("3", 64) })
	add("stdlib", func(p *graphCacheKeyTrial) { p.standard = "fypi1-sha256-" + strings.Repeat("3", 64) })
	for i := range base.environment {
		add(fmt.Sprintf("environment-%d", i), func(p *graphCacheKeyTrial) { p.environment[i] += "-changed" })
		add(fmt.Sprintf("empty-environment-%d", i), func(p *graphCacheKeyTrial) { p.environment[i] = ""; p.wantError = "1001" })
		add(fmt.Sprintf("utf8-environment-%d", i), func(p *graphCacheKeyTrial) { p.environment[i] = "\xff"; p.wantError = "1001" })
	}
	for _, value := range []string{"", "fypi1-sha256-" + strings.Repeat("0", 64), "fypi1-sha256-" + strings.Repeat("A", 64), "fypc1-sha256-" + strings.Repeat("2", 64), "fypi1-sha256-" + strings.Repeat("2", 63)} {
		add(fmt.Sprintf("bad-stdlib-%d", len(trials)), func(p *graphCacheKeyTrial) { p.standard = value; p.wantError = "1001" })
	}
	add("ordinary-refused", func(p *graphCacheKeyTrial) { p.mode = 1; p.wantError = "1001" })
	add("oversized-compatibility", func(p *graphCacheKeyTrial) { p.mode = 2; p.wantError = "1002" })
	add("overdrawn-graph", func(p *graphCacheKeyTrial) { p.mode = 3; p.wantError = "1002" })
	add("overdrawn-ordinary", func(p *graphCacheKeyTrial) { p.mode = 4; p.wantError = "1002" })
	add("production-adapter", func(p *graphCacheKeyTrial) { p.mode = 5 })
	wants := make([][]byte, len(trials))
	for i, p := range trials {
		wants[i] = graphCacheKeyOracle(t, p)
		if i > 1 && p.mode == 0 && p.wantError == "" && bytes.Equal(wants[i], wants[0]) {
			t.Fatalf("oracle invalidation missing: %s", p.name)
		}
	}
	if !bytes.Equal(wants[0], wants[1]) {
		t.Fatal("numeric owner changed portable key")
	}
	candidateAtlasProbeSize(t, "project_state_graph_cache_key", graphCacheKeyProbe, 512<<10, func(t *testing.T, image string) {
		for i, p := range trials {
			t.Run(p.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, graphCacheKeyPacket(p))
				if code != 0 || !bytes.Equal([]byte(out), wants[i]) {
					t.Fatalf("exit=%d bytes=%d/%d output=%x", code, len(out), len(wants[i]), []byte(out[:min(len(out), 100)]))
				}
			})
		}
	}, "module_state_graph_source_artifact")
}

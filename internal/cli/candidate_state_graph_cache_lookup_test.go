package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

type graphCacheLookupTrial struct {
	name, source, coordinate, standard string
	environment                        [8]string
	owner, mode                        uint64
	wantError                          string
}

func graphCacheLookupPacket(p graphCacheLookupTrial) []byte {
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

// Independent framing/hash oracle: no semantic compiler or Complete is needed
// to select an untrusted disk candidate before compilation.
func graphCacheLookupOracle(t *testing.T, p graphCacheLookupTrial) []byte {
	t.Helper()
	if p.wantError != "" {
		return []byte(p.wantError)
	}
	var b bytes.Buffer
	b.WriteString("fairyc.state-graph-cache-lookup\x00\x01")
	for _, field := range p.environment {
		_ = binary.Write(&b, binary.BigEndian, uint64(len(field)))
		b.WriteString(field)
	}
	for _, id := range []struct{ text, prefix string }{{p.standard, "fypi1-sha256-"}, {p.coordinate, "fypc1-sha256-"}} {
		raw, err := hex.DecodeString(strings.TrimPrefix(id.text, id.prefix))
		if err != nil || len(raw) != 32 {
			t.Fatalf("oracle identity %q: %v", id.text, err)
		}
		b.Write(raw)
	}
	source := sha256.Sum256([]byte(p.source))
	b.Write(source[:])
	digest := sha256.Sum256(b.Bytes())
	return append(bytes.Clone(b.Bytes()), []byte("\nfylookup0-sha256-"+hex.EncodeToString(digest[:]))...)
}

const graphCacheLookupProbe = `module main;
import project_state_graph_cache_lookup as key;import project_compiler_core as core;import project_cache_key as compatibility;import project_cache_environment as environment;import project_cache_disk as disk;import module_public_type_text as coordinates;import module_catalog as catalog;import scanner as scanner;import syntax as syntax;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn bytes::Buffer text(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){let n=wire::count_from(raw,c,1048576,1,work);assert(c.code==0);let start=c.offset;c.offset+=n;return switch(bytes::copy(raw[start..c.offset])){case .ok(p):{p}case .err(_):{assert(false);text(raw,c,work)}};}
fn core::Complete ordinary(ref coordinates::Coordinates table,usize owner){
 let b=switch(bytes::copy(b"module ledger;fn i32 main(){return 0;}")){case .ok(p):{p}case .err(_):{assert(false);return ordinary(table,owner);}};
 let tokens=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return ordinary(table,owner);}};
 let parsed=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return ordinary(table,owner);}};
 var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed).is_ok);var owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,owner).is_ok);let imports=v::empty<catalog::Binding>();var usize source_work=0;var usize link_work=0;
 return switch(core::build_complete_sources(ref sources,ref owners,ref imports,table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(_):{assert(false);ordinary(table,owner)}};
}
@effects(filesystem,environment)fn Result<key::Identity,u32> check(ref u8[] raw){
 var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);
 var names=text(raw,ref mut c,ref mut setup);let original=switch(bytes::copy(bytes::view(ref names))){case .ok(p):{p}case .err(_):{return .err(9900);}};
 var rows=v::empty<coordinates::Coordinate>();let selected=coordinates::Coordinate{.owner=if(mode==8){owner+1}else{owner},.start=if(mode==9){1}else{0},.end=if(mode==9){0}else if(mode==10){bytes::len(ref names)+1}else{bytes::len(ref names)}};
 if(mode==16){assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=if(owner==0){1}else{0},.start=0,.end=bytes::len(ref names)}).is_ok);}
 assert(v::push<coordinates::Coordinate>(ref mut rows,selected).is_ok);
 if(mode==7){assert(v::push<coordinates::Coordinate>(ref mut rows,selected).is_ok);}
 if(mode==11){let usize other=if(owner==0){1}else{0};let malformed=coordinates::Coordinate{.owner=other,.start=9999999,.end=0};assert(v::push<coordinates::Coordinate>(ref mut rows,malformed).is_ok);assert(v::push<coordinates::Coordinate>(ref mut rows,malformed).is_ok);}
 if(mode==12){var usize i=1;while(i<513){assert(v::push<coordinates::Coordinate>(ref mut rows,selected).is_ok);i+=1;}}
 if(mode==13){assert(bytes::resize_zeroed(ref mut names,1048577).is_ok);}
 var table=coordinates::assemble(names,rows);
 let a=text(raw,ref mut c,ref mut setup);let b=text(raw,ref mut c,ref mut setup);let d=text(raw,ref mut c,ref mut setup);let e=text(raw,ref mut c,ref mut setup);let f=text(raw,ref mut c,ref mut setup);let g=text(raw,ref mut c,ref mut setup);let h=text(raw,ref mut c,ref mut setup);let j=text(raw,ref mut c,ref mut setup);
 var env=compatibility::environment_of(a,b,d,e,f,g,h,j);var standard=text(raw,ref mut c,ref mut setup);var actual=switch(bytes::copy(raw[c.offset..])){case .ok(p):{p}case .err(_):{return .err(1003);}};
 if(mode==1){var huge=bytes::empty();assert(bytes::resize_zeroed(ref mut huge,16777216).is_ok);env=compatibility::environment_of(huge,bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty());}
 if(mode==14){var huge=bytes::empty();assert(bytes::resize_zeroed(ref mut huge,16777100).is_ok);env=compatibility::environment_of(huge,bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty());}
 var usize work=if(mode==2 || mode==6){16777217}else{0};
 if(mode==5 || mode==6){let complete=ordinary(ref table,owner);return key::for_complete(ref complete,ref env,bytes::view(ref standard),ref mut work);}
 let first=switch(key::compute(bytes::view(ref actual),ref table,owner,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(code):{if(mode==2){assert(work==16777217);}return .err(code);}};let cost=work;assert(cost>0 && cost<=16777216);
 if(!disk::valid_route(key::value(ref first)) || !bytes::equal(bytes::view(ref actual),raw[c.offset..]) || !bytes::equal(coordinates::coordinate_text(ref table,if(mode==16){1}else{0}),bytes::view(ref original))){return .err(9000);}
 work=16777216-cost;let exact=switch(key::compute(bytes::view(ref actual),ref table,owner,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9001);}};
 if(work!=16777216 || !bytes::equal(key::canonical(ref first),key::canonical(ref exact)) || !bytes::equal(key::value(ref first),key::value(ref exact))){return .err(9002);}
 work=16777217-cost;switch(key::compute(bytes::view(ref actual),ref table,owner,ref env,bytes::view(ref standard),ref mut work)){case .ok(_):{return .err(9003);}case .err(code):{if(code!=1002 || work>16777216){return .err(9004);}}};
 work=16777217;switch(key::compute(bytes::view(ref actual),ref table,owner,ref env,bytes::view(ref standard),ref mut work)){case .ok(_):{return .err(9005);}case .err(code):{if(code!=1002 || work!=16777217){return .err(9006);}}};
 work=cost;let shared=switch(key::compute(bytes::view(ref actual),ref table,owner,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9007);}};if(work!=cost*2 || !bytes::equal(key::value(ref shared),key::value(ref first))){return .err(9008);}
 if(mode==3){work=0;let current=switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{return .err(9010);}};work=0;
  let snapshot=switch(environment::key_environment(ref current,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9011);}};work=0;
  let expected=switch(key::compute(bytes::view(ref actual),ref table,owner,ref snapshot,environment::standard_library(ref current),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9012);}};work=0;
  let derived=switch(key::derive(bytes::view(ref actual),ref table,owner,ref current,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9013);}};let derive_cost=work;
  if(!bytes::equal(key::canonical(ref expected),key::canonical(ref derived)) || !bytes::equal(key::value(ref expected),key::value(ref derived))){return .err(9014);}
  work=16777216-derive_cost;switch(key::derive(bytes::view(ref actual),ref table,owner,ref current,ref mut work)){case .ok(_):{}case .err(_):{return .err(9015);}};if(work!=16777216){return .err(9016);}
  work=16777217-derive_cost;switch(key::derive(bytes::view(ref actual),ref table,owner,ref current,ref mut work)){case .ok(_):{return .err(9017);}case .err(code):{if(code!=1002 || work>16777216){return .err(9018);}}};
  work=16777217;switch(key::derive(bytes::view(ref actual),ref table,owner,ref current,ref mut work)){case .ok(_):{return .err(9019);}case .err(code):{if(code!=1002 || work!=16777217){return .err(9020);}}};
 }
 // Identical coordinate rebinding and row order must not change the route.
 let rebound=switch(bytes::copy(bytes::view(ref original))){case .ok(p):{p}case .err(_):{return .err(9021);}};var other=v::empty<coordinates::Coordinate>();let usize next=if(owner==511){0}else{owner+1};assert(v::push<coordinates::Coordinate>(ref mut other,coordinates::Coordinate{.owner=next,.start=0,.end=bytes::len(ref rebound)}).is_ok);let other_table=coordinates::assemble(rebound,other);work=0;
 let same=switch(key::compute(bytes::view(ref actual),ref other_table,next,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9022);}};if(!bytes::equal(key::canonical(ref first),key::canonical(ref same)) || !bytes::equal(key::value(ref first),key::value(ref same))){return .err(9023);}
 if(mode==4){work=0;let complete=switch(core::build_complete_state_graph(bytes::view(ref actual),ref table,owner,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9030);}};
  // Complete keeps the admitted bytes when caller source/table are destroyed.
  if(bytes::len(ref actual)!=0){{let raw=bytes::view_mut(ref mut actual);raw[0]=99;}}actual=bytes::empty();table=coordinates::assemble(bytes::empty(),v::empty<coordinates::Coordinate>());work=0;
  let saved=switch(key::for_complete(ref complete,ref env,bytes::view(ref standard),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9031);}};let adapter_cost=work;
  if(!bytes::equal(key::canonical(ref first),key::canonical(ref saved)) || !bytes::equal(key::value(ref first),key::value(ref saved))){return .err(9032);}
  work=16777216-adapter_cost;switch(key::for_complete(ref complete,ref env,bytes::view(ref standard),ref mut work)){case .ok(_):{}case .err(_):{return .err(9033);}};if(work!=16777216){return .err(9034);}
  work=16777217-adapter_cost;switch(key::for_complete(ref complete,ref env,bytes::view(ref standard),ref mut work)){case .ok(_):{return .err(9035);}case .err(code):{if(code!=1002 || work>16777216){return .err(9036);}}};
  work=16777217;switch(key::for_complete(ref complete,ref env,bytes::view(ref standard),ref mut work)){case .ok(_):{return .err(9037);}case .err(code):{if(code!=1002 || work!=16777217){return .err(9038);}}};
 }
 // All published identity bytes survive the input owners and later mutations.
 if(bytes::len(ref actual)!=0){{let raw=bytes::view_mut(ref mut actual);raw[0]=99;}}actual=bytes::empty();table=coordinates::assemble(bytes::empty(),v::empty<coordinates::Coordinate>());
 env=compatibility::environment_of(bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty(),bytes::empty());standard=bytes::empty();return .ok(first);
}
@effects(filesystem,environment,console)fn i32 main(){var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>1048576-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let result=switch(check(bytes::view(ref input))){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};input=bytes::empty();
 if(!io::write_bytes(key::canonical(ref result)).is_ok || !io::write("\n").is_ok || !io::write_bytes(key::value(ref result)).is_ok){return 3;}return 0;
}`

func TestCandidateStateGraphCacheLookup(t *testing.T) {
	base := graphCacheLookupTrial{
		name: "base", owner: 7, source: "not semantically admitted source\x00\xff",
		coordinate:  "fypc1-sha256-" + strings.Repeat("1", 64),
		standard:    "fypi1-sha256-" + strings.Repeat("2", 64),
		environment: [8]string{"build", "language", "mir", "target", "abi", "toolchain", "backend", "runtime"},
	}
	trials := []graphCacheLookupTrial{base}
	add := func(name string, mutate func(*graphCacheLookupTrial)) {
		p := base
		p.name = name
		mutate(&p)
		trials = append(trials, p)
	}
	add("owner-rebound", func(p *graphCacheLookupTrial) { p.owner = 511 })
	add("source-comment", func(p *graphCacheLookupTrial) { p.source += "// comment\n" })
	add("source-byte", func(p *graphCacheLookupTrial) { p.source = "Not semantically admitted source\x00\xff" })
	add("empty-source-hint", func(p *graphCacheLookupTrial) { p.source = "" })
	add("source-at-limit", func(p *graphCacheLookupTrial) { p.source = strings.Repeat("x", 65536) })
	add("source-over-limit", func(p *graphCacheLookupTrial) { p.source = strings.Repeat("x", 65537); p.wantError = "1002" })
	add("coordinate", func(p *graphCacheLookupTrial) { p.coordinate = "fypc1-sha256-" + strings.Repeat("3", 64) })
	add("stdlib", func(p *graphCacheLookupTrial) { p.standard = "fypi1-sha256-" + strings.Repeat("3", 64) })
	for i := range base.environment {
		add(fmt.Sprintf("environment-%d", i), func(p *graphCacheLookupTrial) { p.environment[i] += "-changed" })
		add(fmt.Sprintf("empty-environment-%d", i), func(p *graphCacheLookupTrial) { p.environment[i] = ""; p.wantError = "1001" })
		add(fmt.Sprintf("utf8-environment-%d", i), func(p *graphCacheLookupTrial) { p.environment[i] = "\xff"; p.wantError = "1001" })
	}
	add("unicode-environment", func(p *graphCacheLookupTrial) { p.environment[0] = "빌드\x00환경" })
	add("framing-a", func(p *graphCacheLookupTrial) { p.environment[0] = "ab"; p.environment[1] = "c" })
	add("framing-b", func(p *graphCacheLookupTrial) { p.environment[0] = "a"; p.environment[1] = "bc" })
	for i, value := range []string{"", "fypi1-sha256-" + strings.Repeat("0", 64), "fypi1-sha256-" + strings.Repeat("A", 64), "fypc1-sha256-" + strings.Repeat("2", 64), "fypi1-sha256-" + strings.Repeat("2", 63), "fypi1-sha256-" + strings.Repeat("2", 65), "fypi1-sha256-" + strings.Repeat("g", 64)} {
		add(fmt.Sprintf("invalid-stdlib-%d", i), func(p *graphCacheLookupTrial) { p.standard = value; p.wantError = "1001" })
	}
	for i, value := range []string{"", "fypc1-sha256-" + strings.Repeat("0", 64), "fypc1-sha256-" + strings.Repeat("A", 64), "fypi1-sha256-" + strings.Repeat("2", 64), "fypc1-sha256-" + strings.Repeat("2", 63), "fypc1-sha256-" + strings.Repeat("2", 65), "fypc1-sha256-" + strings.Repeat("g", 64)} {
		add(fmt.Sprintf("invalid-coordinate-%d", i), func(p *graphCacheLookupTrial) { p.coordinate = value; p.wantError = "1001" })
	}
	for _, control := range []struct {
		name string
		mode uint64
		err  string
	}{{"oversized-environment", 1, "1002"}, {"overdrawn", 2, "1002"}, {"production-adapter", 3, ""}, {"ordinary-complete", 5, "1001"}, {"overdrawn-ordinary", 6, "1002"}, {"duplicate-selected", 7, "1001"}, {"missing-owner", 8, "1001"}, {"reversed-span", 9, "1001"}, {"out-of-range-span", 10, "1001"}, {"unrelated-malformed-duplicates", 11, ""}, {"too-many-coordinates", 12, "1001"}, {"oversized-coordinate-table", 13, "1001"}, {"canonical-length-bound", 14, "1002"}, {"selected-second-row", 16, ""}} {
		add(control.name, func(p *graphCacheLookupTrial) { p.mode = control.mode; p.wantError = control.err })
	}
	add("invalid-owner", func(p *graphCacheLookupTrial) { p.owner = 512; p.wantError = "1001" })
	for _, owner := range []uint64{7, 511} {
		add(fmt.Sprintf("sealed-complete-%d", owner), func(p *graphCacheLookupTrial) {
			p.mode = 4
			p.owner = owner
			p.source = stateGraphAuthoredSource("storage_write", `storage::write("count",1);storage::commit();`, "")
		})
	}
	wants := make([][]byte, len(trials))
	for i, p := range trials {
		wants[i] = graphCacheLookupOracle(t, p)
		if i > 1 && p.mode == 0 && p.wantError == "" && bytes.Equal(wants[i], wants[0]) {
			t.Fatalf("oracle invalidation missing: %s", p.name)
		}
	}
	if !bytes.Equal(wants[0], wants[1]) {
		t.Fatal("numeric owner changed route")
	}
	if bytes.Equal(wants[34], wants[35]) {
		t.Fatal("length framing failed to distinguish adjacent fields")
	}
	t.Logf("%d route/control trials per native profile", len(trials))
	candidateAtlasProbeSize(t, "project_state_graph_cache_lookup", graphCacheLookupProbe, 512<<10, func(t *testing.T, image string) {
		for i, p := range trials {
			t.Run(p.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, graphCacheLookupPacket(p))
				if code != 0 || !bytes.Equal([]byte(out), wants[i]) {
					t.Fatalf("exit=%d bytes=%d/%d output=%x", code, len(out), len(wants[i]), []byte(out[:min(len(out), 100)]))
				}
			})
		}
	})
}

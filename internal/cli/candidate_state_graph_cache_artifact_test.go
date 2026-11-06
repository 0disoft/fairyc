package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

const stateGraphCacheArtifactProbe = `module main;
import project_state_graph_cache_artifact as cache;import project_compiler_core as core;import project_compiler_state_graph as graph;import project_cache_key as compatibility;import project_cache_body_identity as body;import module_state_graph_source as source;import module_public_type_text as coordinates;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn bytes::Buffer copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{bytes::empty()}};}
fn compatibility::Environment snapshot(bool changed){return compatibility::environment_of(copy(if(changed){b"other-build"}else{b"build"}),copy(b"language"),copy(b"mir"),copy(b"target"),copy(b"abi"),copy(b"toolchain"),copy(b"backend"),copy(b"runtime"));}
@effects(console) fn i32 failure(u32 code){discard io::write_u32(code);return 0;}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref input)>9437184-n || !bytes::append(ref mut input,b[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let changed=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);let size=wire::count_from(raw,ref mut c,8388609,1,ref mut setup);let length=wire::count_from(raw,ref mut c,65536,1,ref mut setup);let names=wire::count_from(raw,ref mut c,128,1,ref mut setup);
 if(c.code!=0 || size>raw.len-c.offset || length>raw.len-c.offset-size || names!=raw.len-c.offset-size-length){return 3;}
 let saved=raw[c.offset..c.offset+size];let actual=raw[c.offset+size..c.offset+size+length];let namespace=raw[c.offset+size+length..];var rows=v::empty<coordinates::Coordinate>();if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=namespace.len}).is_ok){return 4;}let table=coordinates::assemble(copy(namespace),rows);let env=snapshot(changed==1);let standard=if(changed==2){b"fypi1-sha256-2222222222222222222222222222222222222222222222222222222222222222"}else{b"fypi1-sha256-1111111111111111111111111111111111111111111111111111111111111111"};
 var usize work=0;
 if(mode==1){let restored=switch(cache::restore_snapshot(saved,actual,ref table,owner,ref env,standard,ref mut work)){case .ok(p):{p}case .err(code):{return failure(code);}};let identity=switch(body::derive(ref restored,owner,ref mut setup)){case .ok(p):{p}case .err(_):{return 5;}};discard io::write_bytes(body::canonical(ref identity));return 0;}
 let current=switch(core::build_complete_state_graph(actual,ref table,owner,ref mut setup)){case .ok(p):{p}case .err(p):{return failure(p.code);}};
 let packet=switch(cache::encode_snapshot(ref current,ref env,standard,ref mut work)){case .ok(p):{p}case .err(code):{return failure(code);}};let encode_work=work;
 if(mode==0){discard io::write_bytes(wire::content(ref packet));return 0;}if(mode!=2){return 6;}
 work=0;let restored=switch(cache::restore_snapshot(wire::content(ref packet),actual,ref table,owner,ref env,standard,ref mut work)){case .ok(p):{p}case .err(code):{return failure(code);}};let restore_work=work;
 work=0;let again=switch(cache::encode_snapshot(ref restored,ref env,standard,ref mut work)){case .ok(p):{p}case .err(_):{return 7;}};if(!bytes::equal(wire::content(ref packet),wire::content(ref again)) || graph::owner(core::checked_state_graph(ref restored))!=owner){return 8;}
 work=16777216-encode_work;switch(cache::encode_snapshot(ref current,ref env,standard,ref mut work)){case .ok(_):{}case .err(_):{return 9;}};if(work!=16777216){return 10;}
 work=16777217-encode_work;switch(cache::encode_snapshot(ref current,ref env,standard,ref mut work)){case .ok(_):{return 11;}case .err(code):{if(code!=1002 || work>16777216){return 12;}}};
 work=16777216-restore_work;switch(cache::restore_snapshot(wire::content(ref packet),actual,ref table,owner,ref env,standard,ref mut work)){case .ok(_):{}case .err(_):{return 13;}};if(work!=16777216){return 14;}
 work=16777217-restore_work;switch(cache::restore_snapshot(wire::content(ref packet),actual,ref table,owner,ref env,standard,ref mut work)){case .ok(_):{return 15;}case .err(code):{if(code!=1002 || work>16777216){return 16;}}};
 work=16777217;switch(cache::restore_snapshot(wire::content(ref packet),actual,ref table,owner,ref env,standard,ref mut work)){case .ok(_):{return 17;}case .err(code):{if(code!=1002 || work!=16777217){return 18;}}};
 work=16777217;switch(cache::encode_snapshot(ref current,ref env,standard,ref mut work)){case .ok(_):{return 19;}case .err(code):{if(code!=1002 || work!=16777217){return 20;}}};
 // A returned Complete retains its source after the request buffer is destroyed.
 {let data=bytes::view_mut(ref mut input);data[data.len-1]=0;}input=bytes::empty();
 let owned=graph::source(core::checked_state_graph(ref restored));if(source::source(owned).len==0){return 21;}work=0;let identity=switch(body::derive(ref restored,owner,ref mut work)){case .ok(p):{p}case .err(_):{return 22;}};discard io::write_bytes(body::canonical(ref identity));return 0;
}`

func stateGraphCacheArtifactRequest(mode, changed, owner uint64, saved []byte, actual, coordinate string) []byte {
	var b bytes.Buffer
	for _, n := range []uint64{mode, changed, owner, uint64(len(saved)), uint64(len(actual)), uint64(len(coordinate))} {
		_ = binary.Write(&b, binary.BigEndian, n)
	}
	b.Write(saved)
	b.WriteString(actual)
	b.WriteString(coordinate)
	return b.Bytes()
}

func stateGraphCacheArtifactFrame(key, packet []byte) []byte {
	var b bytes.Buffer
	b.WriteString("FYSGCAC0")
	_ = binary.Write(&b, binary.BigEndian, uint64(len(key)))
	b.Write(key)
	_ = binary.Write(&b, binary.BigEndian, uint64(len(packet)))
	b.Write(packet)
	return b.Bytes()
}

func TestCandidateStateGraphCacheArtifact(t *testing.T) {
	actual := strings.Replace(stateGraphSourceArtifactSource, "module transaction;", "module ledger;", 1)
	fixture, code := stateGraphAuthoredOracle(t, stateGraphAuthoredTrial{name: "cache-envelope", text: actual})
	if code != "" {
		t.Fatal(code)
	}
	coordinate := fixture.coordinate.String()
	candidateAtlasProbeSize(t, "project_state_graph_cache_artifact", stateGraphCacheArtifactProbe, 512<<10, func(t *testing.T, image string) {
		run := func(mode, changed, owner uint64, saved []byte, source, namespace string) []byte {
			t.Helper()
			status, output := candidateAtlasProcess(t, image, stateGraphCacheArtifactRequest(mode, changed, owner, saved, source, namespace))
			if status != 0 {
				t.Fatalf("exit=%d output=%q", status, output)
			}
			return []byte(output)
		}
		base := run(0, 0, 7, nil, actual, coordinate)
		if len(base) < 24 || string(base[:8]) != "FYSGCAC0" {
			t.Fatalf("invalid cache envelope %q", base)
		}
		keyEnd := 16 + int(binary.BigEndian.Uint64(base[8:16]))
		if keyEnd > len(base)-8 || int(binary.BigEndian.Uint64(base[keyEnd:keyEnd+8])) != len(base)-keyEnd-8 {
			t.Fatal("invalid independent framing")
		}
		key, packet := base[16:keyEnd], base[keyEnd+8:]
		stored, graph := stateGraphSourceArtifactParts(t, packet)
		if stored != actual || !bytes.Equal(base, stateGraphCacheArtifactFrame(key, stateGraphSourceArtifactEnvelope(stored, graph))) {
			t.Fatal("cache envelope changed exact source framing")
		}
		for _, owner := range []uint64{0, 7, 511} {
			t.Run(fmt.Sprintf("rebind/%d", owner), func(t *testing.T) {
				if got := run(1, 0, owner, base, actual, coordinate); !bytes.Equal(got, fixture.canonical) {
					t.Fatalf("body identity mismatch %q", got)
				}
			})
		}
		if got := run(2, 0, 7, nil, actual, coordinate); !bytes.Equal(got, fixture.canonical) {
			t.Fatalf("boundary/lifetime mismatch %q", got)
		}
		changed := strings.Replace(actual, `"count",7`, `"count",8`, 1)
		other := run(0, 0, 7, nil, changed, coordinate)
		otherKeyEnd := 16 + int(binary.BigEndian.Uint64(other[8:16]))
		for _, trial := range []struct {
			name, source, namespace, want string
			changed                       uint64
			packet                        []byte
		}{
			{"source", changed, coordinate, "1001", 0, base},
			{"comment", actual + "\n// changed\n", coordinate, "1001", 0, base},
			{"coordinate", actual, "fypc1-sha256-" + strings.Repeat("2", 64), "1001", 0, base},
			{"environment", actual, coordinate, "1001", 1, base},
			{"standard-library", actual, coordinate, "1001", 2, base},
			{"valid-key-splice", actual, coordinate, "1001", 0, stateGraphCacheArtifactFrame(other[16:otherKeyEnd], packet)},
			{"valid-packet-splice", actual, coordinate, "1001", 0, stateGraphCacheArtifactFrame(key, other[otherKeyEnd+8:])},
			{"valid-source-wrong-key", changed, coordinate, "1001", 0, stateGraphCacheArtifactFrame(key, other[otherKeyEnd+8:])},
			{"marker", actual, coordinate, "1001", 0, append([]byte("FYSGSRC0"), base[8:]...)},
			{"trailing", actual, coordinate, "1001", 0, append(bytes.Clone(base), 0)},
			{"truncated", actual, coordinate, "1001", 0, base[:len(base)-1]},
			{"below-payload-limit", actual, coordinate, "1001", 0, stateGraphCacheArtifactFrame(nil, make([]byte, (8<<20)-25))},
			{"exact-payload-limit", actual, coordinate, "1001", 0, stateGraphCacheArtifactFrame(nil, make([]byte, (8<<20)-24))},
			{"oversized", actual, coordinate, "1002", 0, make([]byte, (8<<20)+1)},
		} {
			t.Run(trial.name, func(t *testing.T) {
				if got := run(1, trial.changed, 7, trial.packet, trial.source, trial.namespace); string(got) != trial.want {
					t.Fatalf("got=%q want=%s", got, trial.want)
				}
			})
		}
		for _, field := range []int{8, keyEnd} {
			malformed := bytes.Clone(base)
			binary.BigEndian.PutUint64(malformed[field:], (8<<20)+1)
			if got := run(1, 0, 7, malformed, actual, coordinate); string(got) != "1002" {
				t.Fatalf("oversized field at %d: %q", field, got)
			}
		}
	}, "project_cache_body_identity")
}

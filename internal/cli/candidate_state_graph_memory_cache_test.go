package cli

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const stateGraphMemoryCacheProbe = `module main;
import project_state_graph_memory_cache as cache;import project_compiler_core as core;import project_compiler_state_graph as graph;import module_state_graph_source as authored;import module_state_graph_source_identity as identity;import module_state_graph_body_hash as hash;import module_project_interfaces as records;import module_public_interface as public;import module_public_type_text as coordinates;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
@effects(console) fn void number(usize n){assert(io::write_usize(n).is_ok);assert(io::write("\n").is_ok);}
@effects(console) fn void blob(ref u8[] value){number(value.len);assert(io::write_bytes(value).is_ok);assert(io::write("\n").is_ok);}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>2097152-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let capacity=wire::number_from(raw,ref mut c,ref mut setup);let limit=wire::number_from(raw,ref mut c,ref mut setup);let count=wire::number_from(raw,ref mut c,ref mut setup);
 var session=switch(cache::create(capacity,limit)){case .ok(p):{p}case .err(code):{number(code as usize);return 0;}};
 var usize i=0;while(i<count){
  let owner=wire::number_from(raw,ref mut c,ref mut setup);var work=wire::number_from(raw,ref mut c,ref mut setup);let size=wire::number_from(raw,ref mut c,ref mut setup);let length=wire::number_from(raw,ref mut c,ref mut setup);if(c.code!=0 || size>raw.len-c.offset || length>raw.len-c.offset-size){return 3;}
  let names=switch(bytes::copy(raw[c.offset..c.offset+size])){case .ok(p):{p}case .err(_):{return 4;}};c.offset+=size;var actual=switch(bytes::copy(raw[c.offset..c.offset+length])){case .ok(p):{p}case .err(_):{return 5;}};c.offset+=length;
  var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=size}).is_ok);var table=coordinates::assemble(names,rows);var observation=cache::Observation{.hit=true};
  let result=cache::build(ref mut session,bytes::view(ref actual),ref table,owner,ref mut observation,ref mut work);
  // Destroy/modify every caller-owned input before inspecting the returned owner.
  {let changed=bytes::view_mut(ref mut actual);if(changed.len!=0){changed[0]=0;}}actual=bytes::empty();table=coordinates::assemble(bytes::empty(),v::empty<coordinates::Coordinate>());
  switch(result){case .ok(complete):{
  number(0);number(if(observation.hit){1}else{0});number(cache::count(ref session));number(cache::stored_bytes(ref session));number(work);
  assert(core::is_state_graph(ref complete));let owned=core::checked_state_graph(ref complete);assert(graph::owner(owned)==owner);assert(authored::source(graph::source(owned))[0]!=0);
  var usize verify=0;let body=identity::identity(graph::body_identity(owned));let decoded=switch(hash::derive(graph::program(owned),identity::context(graph::body_identity(owned)),ref mut verify)){case .ok(p):{p}case .err(_):{return 6;}};
  assert(bytes::equal(hash::canonical(body),hash::canonical(ref decoded)) && bytes::equal(hash::digest(body),hash::digest(ref decoded)));
  let all=core::public_interfaces(ref complete);assert(records::count(all)==1);let record=records::at(all,0);assert(records::owner(record)==owner);verify=0;
  let encoded=switch(public::encode_graph(authored::layouts(graph::source(owned)),authored::owner(graph::source(owned)).package,records::descriptor(record),ref mut verify)){case .ok(p):{p}case .err(_):{return 7;}};
  blob(hash::canonical(ref decoded));blob(hash::digest(ref decoded));blob(records::ordinary_hash(record));blob(records::source_hash(record));blob(bytes::view(ref encoded));
  }case .err(code):{number(code as usize);number(if(observation.hit){1}else{0});number(cache::count(ref session));number(cache::stored_bytes(ref session));number(work);}};i+=1;
 }return 0;
}`

type stateGraphMemoryRequest struct {
	owner, work        uint64
	coordinate, source string
}

type stateGraphMemoryResult struct {
	code, hit, entries, size, work uint64
	blobs                          [5][]byte
}

func stateGraphMemoryPacket(entries, limit uint64, requests ...stateGraphMemoryRequest) []byte {
	var out bytes.Buffer
	n := func(v uint64) { _ = binary.Write(&out, binary.BigEndian, v) }
	n(entries)
	n(limit)
	n(uint64(len(requests)))
	for _, r := range requests {
		n(r.owner)
		n(r.work)
		n(uint64(len(r.coordinate)))
		n(uint64(len(r.source)))
		out.WriteString(r.coordinate)
		out.WriteString(r.source)
	}
	return out.Bytes()
}

func stateGraphMemoryRead(t *testing.T, output string, count int) []stateGraphMemoryResult {
	t.Helper()
	r := bufio.NewReader(strings.NewReader(output))
	n := func() uint64 {
		t.Helper()
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("probe response: %v output=%q", err, output)
		}
		v, err := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	out := make([]stateGraphMemoryResult, count)
	for i := range out {
		p := &out[i]
		p.code, p.hit, p.entries, p.size, p.work = n(), n(), n(), n(), n()
		if p.code != 0 {
			continue
		}
		for j := range p.blobs {
			p.blobs[j] = make([]byte, n())
			if _, err := io.ReadFull(r, p.blobs[j]); err != nil {
				t.Fatal(err)
			}
			if end, err := r.ReadByte(); err != nil || end != '\n' {
				t.Fatalf("blob delimiter: %q %v", end, err)
			}
		}
	}
	if extra, _ := io.ReadAll(r); len(extra) != 0 {
		t.Fatalf("unexpected response suffix %q", extra)
	}
	return out
}

func stateGraphMemoryWant(t *testing.T, trial stateGraphAuthoredTrial, coordinate *packagegraph.CoordinateID) [5][]byte {
	t.Helper()
	f, code := stateGraphAuthoredOracle(t, trial)
	if code != "" {
		t.Fatalf("invalid cache oracle %s", code)
	}
	if coordinate != nil {
		f = stateGraphBodyFixture(t, trial.name, f.input, *coordinate)
	}
	descriptor, err := moduleinterface.Extract(f.input, 7, map[hir.PackageID]packagegraph.CoordinateID{7: f.coordinate})
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
	source, err := moduleinterface.ComputeGeneric(descriptor, nil)
	if err != nil {
		t.Fatal(err)
	}
	return [5][]byte{f.canonical, []byte(f.hash), []byte(ordinary.String()), []byte(source.String()), encoded}
}

func TestCandidateStateGraphMemoryCache(t *testing.T) {
	var trials []stateGraphAuthoredTrial
	for _, trial := range stateGraphAuthoredTrials() {
		switch trial.name {
		case "read-local", "branch-join", "helper-transitive", "borrow-alias-view-copy":
			trials = append(trials, trial)
		}
	}
	candidateAtlasProbeSize(t, "project_state_graph_memory_cache", stateGraphMemoryCacheProbe, 512<<10, func(t *testing.T, image string) {
		run := func(entries, limit uint64, req ...stateGraphMemoryRequest) []stateGraphMemoryResult {
			t.Helper()
			code, output := candidateAtlasProcess(t, image, stateGraphMemoryPacket(entries, limit, req...))
			if code != 0 {
				t.Fatalf("cache native exit=%d output=%q", code, output)
			}
			return stateGraphMemoryRead(t, output, len(req))
		}
		assert := func(p stateGraphMemoryResult, hit, entries uint64, want [5][]byte) {
			t.Helper()
			if p.code != 0 || p.hit != hit || p.entries != entries || p.work == 0 || p.work > 1<<24 || p.size > 8<<20 {
				t.Fatalf("cache result code/hit/count/bytes/work=%d/%d/%d/%d/%d", p.code, p.hit, p.entries, p.size, p.work)
			}
			for i := range want {
				if !bytes.Equal(p.blobs[i], want[i]) {
					t.Fatalf("Go parity blob %d: got %d bytes, want %d", i, len(p.blobs[i]), len(want[i]))
				}
			}
		}
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				f, _ := stateGraphAuthoredOracle(t, trial)
				a := stateGraphMemoryRequest{owner: 7, coordinate: f.coordinate.String(), source: trial.text}
				rebound := a
				rebound.owner = 511
				p := run(2, 8<<20, a, a, rebound)
				want := stateGraphMemoryWant(t, trial, nil)
				assert(p[0], 0, 1, want)
				assert(p[1], 1, 1, want)
				assert(p[2], 1, 1, want)
				if p[0].size <= 64 || p[0].size != p[1].size || p[1].size != p[2].size {
					t.Fatal("warm lookup changed retained bytes")
				}
			})
		}
		base := trials[0]
		f, _ := stateGraphAuthoredOracle(t, base)
		a := stateGraphMemoryRequest{owner: 7, coordinate: f.coordinate.String(), source: base.text}
		want := stateGraphMemoryWant(t, base, nil)
		changed := base
		changed.text += "\n// exact source changes the key\n"
		b := a
		b.source = changed.text
		other := f.coordinate
		other[0] ^= 1
		c := a
		c.coordinate = other.String()
		p := run(3, 8<<20, a, b, b, c, c, a)
		for i, expected := range []struct {
			hit, count uint64
			blobs      [5][]byte
		}{{0, 1, want}, {0, 2, stateGraphMemoryWant(t, changed, nil)}, {1, 2, stateGraphMemoryWant(t, changed, nil)}, {0, 3, stateGraphMemoryWant(t, base, &other)}, {1, 3, stateGraphMemoryWant(t, base, &other)}, {1, 3, want}} {
			assert(p[i], expected.hit, expected.count, expected.blobs)
		}
		for _, bounds := range [][2]uint64{{0, 8 << 20}, {2, 0}, {2, 64}, {2, p[0].size - 1}} {
			q := run(bounds[0], bounds[1], a, a)
			assert(q[0], 0, 0, want)
			assert(q[1], 0, 0, want)
		}
		for _, bounds := range [][2]uint64{{1, 8 << 20}, {2, p[0].size}} {
			q := run(bounds[0], bounds[1], a, b, b, a)
			assert(q[0], 0, 1, want)
			assert(q[1], 0, 1, stateGraphMemoryWant(t, changed, nil))
			assert(q[2], 0, 1, stateGraphMemoryWant(t, changed, nil))
			assert(q[3], 1, 1, want)
		}
		assert(run(512, 8<<20, a, a)[1], 1, 1, want)
		// Disabled and full sessions still cold-build under the same shared
		// ledger. Neither an exact success nor a one-short failure inserts.
		for _, scenario := range []struct {
			name           string
			entries, limit uint64
			prefix         []stateGraphMemoryRequest
			request        stateGraphMemoryRequest
			want           [5][]byte
		}{{"disabled", 0, 8 << 20, nil, a, want}, {"entry-full", 1, 8 << 20, []stateGraphMemoryRequest{a}, b, stateGraphMemoryWant(t, changed, nil)}, {"byte-full", 2, p[0].size, []stateGraphMemoryRequest{a}, b, stateGraphMemoryWant(t, changed, nil)}} {
			t.Run(scenario.name+"-ledger", func(t *testing.T) {
				requests := append(append([]stateGraphMemoryRequest{}, scenario.prefix...), scenario.request)
				baseline := run(scenario.entries, scenario.limit, requests...)[len(requests)-1]
				for _, short := range []uint64{0, 1} {
					r := scenario.request
					r.work = (1 << 24) - baseline.work + short
					requests[len(requests)-1] = r
					q := run(scenario.entries, scenario.limit, requests...)[len(requests)-1]
					if short == 0 {
						assert(q, 0, uint64(len(scenario.prefix)), scenario.want)
						if q.work != 1<<24 {
							t.Fatal("exact cold ledger not consumed")
						}
					} else if q.code != 1002 || q.hit != 0 || q.entries != baseline.entries || q.size != baseline.size || q.work > 1<<24 {
						t.Fatalf("cold fallback reset ledger or published: %+v", q)
					}
				}
			})
		}
		bad := a
		bad.source = "module ledger;fn void transaction(){}"
		q := run(2, 8<<20, bad, a, bad, a)
		if q[0].code != 1001 || q[0].entries != 0 || q[0].size != 0 || q[0].hit != 0 || q[2].code != 1001 || q[2].entries != 1 || q[2].hit != 0 {
			t.Fatal("invalid source published a cache entry")
		}
		assert(q[1], 0, 1, want)
		assert(q[3], 1, 1, want)
		// Exact and one-short work for cold publication and warm verification.
		costs := run(2, 8<<20, a, a)
		for mode, cost := range []uint64{costs[0].work, costs[1].work} {
			for _, short := range []uint64{0, 1} {
				r := a
				r.work = (1 << 24) - cost + short
				req := []stateGraphMemoryRequest{r}
				if mode == 1 {
					req = append([]stateGraphMemoryRequest{a}, req...)
				}
				q := run(2, 8<<20, req...)[mode]
				if short == 0 {
					assert(q, uint64(mode), 1, want)
					if q.work != 1<<24 {
						t.Fatal("exact ledger not consumed")
					}
				} else if q.code != 1002 || q.hit != 0 || q.entries != uint64(mode) || q.work > 1<<24 {
					t.Fatalf("short ledger changed cache state: %+v", q)
				}
			}
		}
		for _, edit := range []func(*stateGraphMemoryRequest){
			func(r *stateGraphMemoryRequest) { r.owner = 512 },
			func(r *stateGraphMemoryRequest) { r.coordinate = "fypc1-sha256-" + strings.Repeat("0", 64) },
			func(r *stateGraphMemoryRequest) { r.coordinate = strings.ToUpper(r.coordinate) },
			func(r *stateGraphMemoryRequest) { r.source = strings.Repeat("x", 65537) },
		} {
			r := a
			edit(&r)
			q := run(2, 8<<20, a, r, a)
			if q[1].code != 1001 && q[1].code != 1002 || q[1].hit != 0 || q[1].entries != 1 || q[1].size != q[0].size {
				t.Fatal("invalid request affected session")
			}
			assert(q[2], 1, 1, want)
		}
		for _, warm := range []bool{false, true} {
			r := a
			r.work = (1 << 24) + 1
			req := []stateGraphMemoryRequest{r}
			if warm {
				req = append([]stateGraphMemoryRequest{a}, req...)
			}
			q := run(2, 8<<20, req...)[len(req)-1]
			if q.code != 1002 || q.work != r.work || q.hit != 0 {
				t.Fatal("overdrawn meter reset")
			}
		}
		for _, bounds := range [][2]uint64{{513, 8 << 20}, {1, (8 << 20) + 1}} {
			code, output := candidateAtlasProcess(t, image, stateGraphMemoryPacket(bounds[0], bounds[1]))
			if code != 0 || output != "1002\n" {
				t.Fatalf("invalid constructor bounds: %d %q", code, output)
			}
		}
	})
}

func TestCandidateStateGraphMemoryCacheSealed(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"session-constructor", "E0103", `fn cache::Session forge(){return cache::Session{.entry_limit=0,.byte_limit=0,.header=true,.digest=0,.coordinate=0,.packets=0};}`},
		{"private-packets", "E0103", `fn ref v::Vec<wire::Payload> forge(ref cache::Session p){return ref p.packets;}`},
		{"immutable-session", "E1008", `fn void forge(ref cache::Session p){let ref mut cache::Session q=ref mut p;}`},
		{"ordinary-complete-cannot-seed", "E1006", `fn void forge(ref core::Complete p,ref coordinates::Coordinates table,ref mut v::Vec<cache::Session> session,ref mut cache::Observation observation,ref mut usize work){discard cache::build(session,p,table,0,observation,work);}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "project_state_graph_memory_cache")
			writeTestSource(t, root, "src/main.fy", "module main;import project_state_graph_memory_cache as cache;import project_compiler_core as core;import module_public_type_text as coordinates;import std::vec as v;import candidate_artifact_wire as wire;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed graph memory cache", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), "main.fy:") || !strings.Contains(diagnostic.String(), trial.code) {
				t.Fatalf("%s: code=%d diagnostics=%s", fmt.Sprint(trial.name), code, &diagnostic)
			}
		})
	}
}

package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const stateGraphDiskCacheProbe = `module main;
import project_state_graph_disk_cache as cache;import project_cache_disk as disk;import project_compiler_core as core;import project_compiler_state_graph as graph;import project_cache_key as compatibility;import project_cache_environment as environment;import module_state_graph_source as source;import module_state_graph_source_identity as identity;import module_state_graph_body_hash as hash;import module_project_interfaces as records;import module_public_type_text as coordinates;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::process as process;import std::io as io;
fn bytes::Buffer copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{bytes::empty()}};}
fn compatibility::Environment snapshot(usize change){return compatibility::environment_of(copy(if(change==1){b"other-build"}else{b"build"}),copy(b"language"),copy(b"mir"),copy(b"target"),copy(b"abi"),copy(b"toolchain"),copy(b"backend"),copy(b"runtime"));}
@effects(console)fn void number(usize n){assert(io::write_usize(n).is_ok);assert(io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok);assert(io::write("\n").is_ok);}
@effects(console)fn void status(u32 code,u32 phase,u32 stage,cache::Observation observed,cache::Work work,usize source_work){number(code as usize);number(phase as usize);number(stage as usize);number(observed.path as usize);number(if(observed.stored){1}else{0});number(observed.lookup_phase as usize);number(observed.lookup_code as usize);number(observed.store_phase as usize);number(observed.store_code as usize);number(work.lookup);number(work.scan);number(work.io);number(work.verify);number(work.encode);number(source_work);}
@effects(filesystem,environment,clock,console)fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref input)>1048576-n || !bytes::append(ref mut input,b[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let change=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);let entries=wire::number_from(raw,ref mut c,ref mut setup);let total=wire::number_from(raw,ref mut c,ref mut setup);let record_limit=wire::number_from(raw,ref mut c,ref mut setup);
 var work=cache::empty_work();work.lookup=wire::number_from(raw,ref mut c,ref mut setup);work.scan=wire::number_from(raw,ref mut c,ref mut setup);work.io=wire::number_from(raw,ref mut c,ref mut setup);work.verify=wire::number_from(raw,ref mut c,ref mut setup);work.encode=wire::number_from(raw,ref mut c,ref mut setup);var source_work=wire::number_from(raw,ref mut c,ref mut setup);let size=wire::count_from(raw,ref mut c,128,1,ref mut setup);let length=wire::count_from(raw,ref mut c,65537,1,ref mut setup);
 if(c.code!=0 || size>raw.len-c.offset || length!=raw.len-c.offset-size){return 3;}let namespace=raw[c.offset..c.offset+size];let actual=raw[c.offset+size..];var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=namespace.len}).is_ok);let table=coordinates::assemble(copy(namespace),rows);
 var u8[4096] path=[0;4096];let path_size=switch(process::arg(1,ref mut path)){case .ok(p):{p}case .err(_):{return 4;}};let ref u8[] root=ref path;let store=switch(disk::open(root[..path_size],disk::Limits{.entries,.bytes=total,.record=record_limit,.temporary_age=6000000000})){case .ok(p):{p}case .err(_):{return 5;}};
 let env=snapshot(change);let standard=if(change==2){b"fypi1-sha256-2222222222222222222222222222222222222222222222222222222222222222"}else{b"fypi1-sha256-1111111111111111111111111111111111111111111111111111111111111111"};var observed=cache::empty();
 let current=if(mode==3){var usize env_work=0;let actual_env=switch(environment::current(ref mut env_work)){case .ok(p):{p}case .err(_):{return 6;}};switch(cache::build(ref store,actual,ref table,owner,ref actual_env,ref mut work,ref mut source_work,ref mut observed)){case .ok(p):{p}case .err(p):{status(p.code,p.phase,p.stage,observed,work,source_work);return 0;}}}else if(mode==1){switch(cache::from_disk_snapshot(ref store,actual,ref table,owner,ref env,standard,ref mut work)){case .ok(p):{observed.path=2;p}case .err(p):{status(p.code,p.phase,0,observed,work,source_work);return 0;}}}else if(mode==2){observed.path=1;let fresh=switch(core::build_complete_state_graph(actual,ref table,owner,ref mut source_work)){case .ok(p):{p}case .err(p):{status(p.code,p.phase,p.stage,observed,work,source_work);return 0;}};switch(cache::to_disk_snapshot(ref store,ref fresh,ref env,standard,ref mut work)){case .ok(_):{observed.stored=true;}case .err(p):{status(p.code,p.phase,0,observed,work,source_work);return 0;}};fresh}else{switch(cache::build_snapshot(ref store,actual,ref table,owner,ref env,standard,ref mut work,ref mut source_work,ref mut observed)){case .ok(p):{p}case .err(p):{status(p.code,p.phase,p.stage,observed,work,source_work);return 0;}}};
 // The stored/read Complete owns source and coordinate beyond request lifetime.
 {let data=bytes::view_mut(ref mut input);data[0]=255;}input=bytes::empty();
 status(0,0,0,observed,work,source_work);let owned=core::checked_state_graph(ref current);let bound=graph::body_identity(owned);var usize checked=0;let decoded=switch(hash::derive(graph::program(owned),identity::context(bound),ref mut checked)){case .ok(p):{p}case .err(_):{return 7;}};if(!bytes::equal(hash::canonical(ref decoded),hash::canonical(identity::identity(bound)))){return 8;}
 let record=records::at(core::public_interfaces(ref current),0);blob(hash::canonical(ref decoded));blob(records::ordinary_hash(record));blob(records::source_hash(record));blob(source::source(graph::source(owned)));return 0;
}`

type graphDiskRequest struct {
	mode, changed, owner, entries, total, record uint64
	work                                         [6]uint64 // lookup, scan, io, verify, encode, cold source
	coordinate, source                           string
}

func (p graphDiskRequest) packet() []byte {
	var b bytes.Buffer
	for _, n := range append([]uint64{p.mode, p.changed, p.owner, p.entries, p.total, p.record}, p.work[:]...) {
		_ = binary.Write(&b, binary.BigEndian, n)
	}
	for _, n := range []uint64{uint64(len(p.coordinate)), uint64(len(p.source))} {
		_ = binary.Write(&b, binary.BigEndian, n)
	}
	b.WriteString(p.coordinate)
	b.WriteString(p.source)
	return b.Bytes()
}

type graphDiskResult struct {
	numbers [15]uint64
	blobs   [4][]byte
}

func graphDiskParse(t *testing.T, output string) graphDiskResult {
	t.Helper()
	r := bufio.NewReader(strings.NewReader(output))
	number := func() uint64 {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("result number: %v output=%q", err, output)
		}
		n, err := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	var result graphDiskResult
	for i := range result.numbers {
		result.numbers[i] = number()
	}
	if result.numbers[0] == 0 {
		for i := range result.blobs {
			size := number()
			if size > 1<<20 {
				t.Fatalf("unbounded output %d", size)
			}
			result.blobs[i] = make([]byte, size)
			if _, err := io.ReadFull(r, result.blobs[i]); err != nil {
				t.Fatal(err)
			}
			if end, err := r.ReadByte(); err != nil || end != '\n' {
				t.Fatalf("blob terminator: %d %v", end, err)
			}
		}
	}
	if extra, _ := io.ReadAll(r); len(extra) != 0 {
		t.Fatalf("unexpected result suffix %q", extra)
	}
	return result
}

func graphDiskInventory(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := candidateAggregateCacheFiles(t, root)
	for name, raw := range files {
		if len(name) != 144 || !strings.HasPrefix(name, "fyproject0-") || !strings.HasSuffix(name, ".fyc") || !bytes.HasPrefix(raw, []byte("FYSGCAC0")) {
			t.Fatalf("unexpected/staged cache leaf %s", name)
		}
		hash := sha256.Sum256(raw)
		if name[76:140] != fmt.Sprintf("%x", hash) {
			t.Fatalf("generation digest mismatch %s", name)
		}
	}
	return files
}

func TestCandidateStateGraphDiskCache(t *testing.T) {
	trial := stateGraphAuthoredTrial{name: "disk-source", text: stateGraphAuthoredSource("storage_write", `storage::write("count",1);storage::commit();`, "")}
	fixture, oracleCode := stateGraphAuthoredOracle(t, trial)
	if oracleCode != "" {
		t.Fatal(oracleCode)
	}
	want := stateGraphMemoryWant(t, trial, nil)
	base := graphDiskRequest{owner: 7, entries: 64, total: 64 << 20, record: 8 << 20, coordinate: fixture.coordinate.String(), source: trial.text}
	candidateAtlasProbeSize(t, "project_state_graph_disk_cache", stateGraphDiskCacheProbe, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		nativeCalls := 0
		run := func(root string, request graphDiskRequest) graphDiskResult {
			t.Helper()
			nativeCalls++
			status, out := candidateAtlasProcess(t, image, request.packet(), filepath.ToSlash(root))
			if status != 0 {
				t.Fatalf("native exit=%d output=%q", status, out)
			}
			return graphDiskParse(t, out)
		}
		assert := func(p graphDiskResult, path uint64, expected [5][]byte, source string) {
			t.Helper()
			if p.numbers[0] != 0 || p.numbers[3] != path || p.numbers[9] == 0 || (path == 2 && (p.numbers[12] == 0 || p.numbers[14] != 0)) || (path == 1 && p.numbers[14] == 0) {
				t.Fatalf("unexpected cache path/ledgers: %+v", p.numbers)
			}
			for i, expected := range [][]byte{expected[0], expected[2], expected[3], []byte(source)} {
				if !bytes.Equal(p.blobs[i], expected) {
					t.Fatalf("Go oracle blob %d: %d/%d bytes", i, len(p.blobs[i]), len(expected))
				}
			}
		}
		root := t.TempDir()
		cold := run(root, base)
		assert(cold, 1, want, base.source)
		if cold.numbers[4] != 1 {
			t.Fatalf("cold was not stored: %+v", cold.numbers)
		}
		original := graphDiskInventory(t, root)
		if len(original) != 1 {
			t.Fatalf("cold inventory %d", len(original))
		}
		warm := run(root, base)
		assert(warm, 2, want, base.source)
		for field, cap := range []uint64{1 << 24, 1 << 24, 1 << 30, 1 << 24} {
			for _, short := range []uint64{0, 1} {
				request := base
				request.mode = 1
				request.work[field] = cap - warm.numbers[9+field] + short
				result := run(root, request)
				if short == 0 {
					assert(result, 2, want, base.source)
					if result.numbers[9+field] != cap {
						t.Fatalf("exact load ledger %d not consumed: %+v", field, result.numbers)
					}
				} else {
					wantPhase, wantCode := uint64(2), uint64(9)
					if field == 0 {
						wantPhase, wantCode = 1, 1002
					}
					if field == 3 {
						wantPhase, wantCode = 3, 1002
					}
					if result.numbers[0] != wantCode || result.numbers[1] != wantPhase || result.numbers[9+field] > cap || result.numbers[14] != 0 {
						t.Fatalf("short load ledger %d changed/refilled: %+v", field, result.numbers)
					}
				}
			}
		}
		publication := base
		publication.mode = 2
		baselineSave := run(root, publication)
		assert(baselineSave, 1, want, base.source)
		for _, short := range []uint64{0, 1} {
			request := publication
			request.work[4] = (1 << 24) - baselineSave.numbers[13] + short
			p := run(root, request)
			if short == 0 {
				assert(p, 1, want, base.source)
				if p.numbers[13] != 1<<24 {
					t.Fatal("exact encode ledger not consumed")
				}
			} else if p.numbers[0] != 1002 || p.numbers[1] != 4 || p.numbers[4] != 0 || p.numbers[13] > 1<<24 {
				t.Fatalf("short encode published or reset: %+v", p.numbers)
			}
		}
		rebound := base
		rebound.owner = 511
		assert(run(root, rebound), 2, want, base.source)
		if current := graphDiskInventory(t, root); !maps.EqualFunc(current, original, bytes.Equal) {
			t.Fatal("warm/rebinding changed retained bytes")
		}
		for name, data := range original {
			corrupt := bytes.Clone(data)
			corrupt[len(corrupt)-1] ^= 1
			if err := os.WriteFile(filepath.Join(root, name), corrupt, 0600); err != nil {
				t.Fatal(err)
			}
			assert(run(root, base), 1, want, base.source)
			if current := graphDiskInventory(t, root); !maps.EqualFunc(current, original, bytes.Equal) {
				t.Fatal("corrupt collision was not repaired exactly")
			}
		}
		// A correct content address is insufficient: newer, individually framed
		// candidates must pass current-source/key admission before an older hit.
		for originalName, originalData := range original {
			info, err := os.Stat(filepath.Join(root, originalName))
			if err != nil {
				t.Fatal(err)
			}
			keyEnd := 16 + int(binary.BigEndian.Uint64(originalData[8:16]))
			badKey := bytes.Clone(originalData)
			badKey[keyEnd-1] ^= 1
			oldFamily := bytes.Clone(originalData)
			copy(oldFamily[:8], "FYCACHE0")
			for _, untrusted := range [][]byte{badKey, oldFamily} {
				hash := sha256.Sum256(untrusted)
				name := originalName[:76] + fmt.Sprintf("%x", hash) + ".fyc"
				path := filepath.Join(root, name)
				if err := os.WriteFile(path, untrusted, 0600); err != nil {
					t.Fatal(err)
				}
				newest := info.ModTime().Add(time.Minute)
				if err := os.Chtimes(path, newest, newest); err != nil {
					t.Fatal(err)
				}
				p := run(root, base)
				assert(p, 2, want, base.source)
				if p.numbers[12] <= warm.numbers[12] {
					t.Fatal("newest candidate bypassed semantic admission")
				}
				if after := candidateAggregateCacheFiles(t, root); len(after) != 2 || !bytes.Equal(after[name], untrusted) || !bytes.Equal(after[originalName], originalData) {
					t.Fatal("read admission changed either observed generation")
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			private := t.TempDir()
			badHash := sha256.Sum256(badKey)
			badName := originalName[:76] + fmt.Sprintf("%x", badHash) + ".fyc"
			if err := os.WriteFile(filepath.Join(private, badName), badKey, 0600); err != nil {
				t.Fatal(err)
			}
			repaired := run(private, base)
			assert(repaired, 1, want, base.source)
			if repaired.numbers[4] != 1 || repaired.numbers[5] != 3 || repaired.numbers[6] != 1001 || len(graphDiskInventory(t, private)) != 2 {
				t.Fatalf("semantic-only corruption did not cold-repair safely: %+v", repaired.numbers)
			}
			assert(run(private, base), 2, want, base.source)
		}
		for _, short := range []uint64{0, 1, 2} {
			request := base
			request.work[5] = (1 << 24) - cold.numbers[14] + short
			if short == 2 {
				request.work[5] = (1 << 24) + 1
			}
			private := t.TempDir()
			p := run(private, request)
			if short == 0 {
				assert(p, 1, want, base.source)
				if p.numbers[14] != 1<<24 {
					t.Fatal("exact cold source ledger not consumed")
				}
			} else if p.numbers[0] != 1002 || p.numbers[1] != 8 || p.numbers[2] != 1 || p.numbers[4] != 0 || len(graphDiskInventory(t, private)) != 0 || (short == 1 && p.numbers[14] > 1<<24) || (short == 2 && p.numbers[14] != request.work[5]) {
				t.Fatalf("cold source work reset or published: %+v", p.numbers)
			}
		}
		changed := base
		changed.source += "\n// exact source invalidates disk lookup\n"
		changedWant := stateGraphMemoryWant(t, stateGraphAuthoredTrial{name: "disk-comment", text: changed.source}, nil)
		assert(run(root, changed), 1, changedWant, changed.source)
		assert(run(root, changed), 2, changedWant, changed.source)
		for _, change := range []uint64{1, 2} {
			request := base
			request.changed = change
			assert(run(root, request), 1, want, base.source)
			assert(run(root, request), 2, want, base.source)
		}
		before := graphDiskInventory(t, root)
		bad := base
		bad.source = "module ledger;fn void transaction(){}"
		failure := run(root, bad)
		if failure.numbers[0] != 1001 || failure.numbers[1] != 8 || failure.numbers[2] != 1 || failure.numbers[4] != 0 {
			t.Fatalf("cold source error changed %+v", failure.numbers)
		}
		if after := graphDiskInventory(t, root); !maps.EqualFunc(before, after, bytes.Equal) {
			t.Fatal("failed Complete changed cache bytes")
		}
		for _, field := range []int{0, 1, 2, 3, 4} {
			t.Run(fmt.Sprintf("exhausted-%d", field), func(t *testing.T) {
				request := base
				request.work[field] = (1 << 24) + 1
				if field == 2 {
					request.work[field] = (1 << 30) + 1
				}
				cacheRoot := t.TempDir()
				p := run(cacheRoot, request)
				assert(p, 1, want, base.source)
				if p.numbers[4] != 0 || p.numbers[9+field] != request.work[field] || len(graphDiskInventory(t, cacheRoot)) != 0 {
					t.Fatalf("cache exhaustion reset work or published: %+v", p.numbers)
				}
			})
		}
		for _, authored := range stateGraphAuthoredTrials() {
			switch authored.name {
			case "read-local", "branch-join", "helper-transitive", "borrow-alias-view-copy":
				f, code := stateGraphAuthoredOracle(t, authored)
				if code != "" {
					t.Fatal(code)
				}
				request := base
				request.source, request.coordinate = authored.text, f.coordinate.String()
				expected := stateGraphMemoryWant(t, authored, nil)
				private := t.TempDir()
				assert(run(private, request), 1, expected, request.source)
				assert(run(private, request), 2, expected, request.source)
			}
		}
		production := base
		production.mode = 3
		productionRoot := t.TempDir()
		assert(run(productionRoot, production), 1, want, base.source)
		assert(run(productionRoot, production), 2, want, base.source)
		retention := base
		retention.entries = 1
		retentionRoot := t.TempDir()
		assert(run(retentionRoot, retention), 1, want, base.source)
		retention.changed = 1
		assert(run(retentionRoot, retention), 1, want, base.source)
		if len(graphDiskInventory(t, retentionRoot)) != 1 {
			t.Fatal("entry retention not enforced")
		}
		// Two independent processes race through immutable no-replace publication.
		parallelRoot := t.TempDir()
		type result struct {
			output string
			err    error
		}
		results := make(chan result, 2)
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		nativeCalls += 2
		for range 2 {
			go func() {
				cmd := exec.CommandContext(ctx, image, filepath.ToSlash(parallelRoot))
				cmd.Stdin = bytes.NewReader(base.packet())
				out, err := cmd.CombinedOutput()
				results <- result{string(out), err}
			}()
		}
		for range 2 {
			p := <-results
			if p.err != nil {
				t.Fatalf("concurrent process: %v %q", p.err, p.output)
			}
			parsed := graphDiskParse(t, p.output)
			assert(parsed, parsed.numbers[3], want, base.source)
			if parsed.numbers[3] != 1 && parsed.numbers[3] != 2 {
				t.Fatalf("concurrent result path %d", parsed.numbers[3])
			}
		}
		if len(graphDiskInventory(t, parallelRoot)) != 1 {
			t.Fatal("concurrent publication produced extra/staged generations")
		}
		assert(run(parallelRoot, base), 2, want, base.source)
		t.Logf("%d native graph-cache processes for this profile", nativeCalls)
	})
}

package cli

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func graphResponseProbe() string {
	prefix := strings.Split(graphTransportProbe, "@effects(console)fn i32 main(){")[0]
	prefix = strings.Replace(prefix, "module main;", `module main;import module_state_graph_response as response;import module_state_graph_response_common as common;import project_identity_bytes as identities;`, 1)
	prefix = strings.Replace(prefix, "fn bytes::Buffer copy", `@link_name("fairyc_test_graph_failure")unsafe extern C fn void allocation_failure(usize successes);
fn bytes::Buffer copy`, 1)
	return prefix + `
fn Result<request::Snapshot,u32> response_snapshot(ref transport::Prepared prepared,u32[8] hash,usize mode){
 if((mode&65536)!=0){
  let proof=graph::program(core::checked_state_graph(transport::checked(prepared)));let layouts=program::layouts(proof);let schema=layout::at(layouts,model::header(program::root(proof)).layout);var usize i=0;
  while(i<schema.field_count){let field=layout::field_at(layouts,schema.field_start+i);if(field.kind==1 && bytes::equal(layout::name(layouts,schema.field_start+i),b"blob")){var data=bytes::empty();if(!bytes::resize_zeroed(ref mut data,2097152).is_ok){return .err(1003);}var entries=v::empty<request::Entry>();if(!v::push<request::Entry>(ref mut entries,request::Entry{.key=field.prefix,.start=0,.end=2097152}).is_ok){return .err(1003);}return .ok(request::snapshot_of(hash,73,data,entries));}i+=1;}return .err(1001);
 }
 if((mode&128)==0){return seed_snapshot(prepared,hash,(mode&16)!=0);}
 let proof=graph::program(core::checked_state_graph(transport::checked(prepared)));let layouts=program::layouts(proof);let schema=layout::at(layouts,model::header(program::root(proof)).layout);var usize i=0;
 while(i<schema.field_count){let field=layout::field_at(layouts,schema.field_start+i);if(field.kind==1 && bytes::equal(layout::name(layouts,schema.field_start+i),b"flag")){var entries=v::empty<request::Entry>();if(!v::push<request::Entry>(ref mut entries,request::Entry{.key=field.prefix,.start=0,.end=1}).is_ok){return .err(1003);}return .ok(request::snapshot_of(hash,1,copy(b"\x00"),entries));}i+=1;}return .err(1001);
}
@effects(console)fn i32 main(){
 var incoming=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref incoming)>73400320-n || !bytes::append(ref mut incoming,b[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref incoming);var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);var work=common::empty_work();work.structure=wire::number_from(raw,ref mut c,ref mut setup);work.bytes=wire::number_from(raw,ref mut c,ref mut setup);
 let limits=request::Limits{.writes=wire::number_from(raw,ref mut c,ref mut setup),.bytes=wire::number_from(raw,ref mut c,ref mut setup),.value=wire::number_from(raw,ref mut c,ref mut setup),.key=wire::number_from(raw,ref mut c,ref mut setup)};
 let coordinate_size=wire::number_from(raw,ref mut c,ref mut setup);let source_size=wire::number_from(raw,ref mut c,ref mut setup);let runtime_size=wire::number_from(raw,ref mut c,ref mut setup);let frame_size=wire::number_from(raw,ref mut c,ref mut setup);let response_size=wire::number_from(raw,ref mut c,ref mut setup);
 if(c.code!=0 || coordinate_size>raw.len-c.offset || source_size>raw.len-c.offset-coordinate_size || runtime_size>raw.len-c.offset-coordinate_size-source_size || frame_size>raw.len-c.offset-coordinate_size-source_size-runtime_size || response_size!=raw.len-c.offset-coordinate_size-source_size-runtime_size-frame_size){return 3;}
 let coordinate=raw[c.offset..c.offset+coordinate_size];let actual=raw[c.offset+coordinate_size..c.offset+coordinate_size+source_size];let asset_at=c.offset+coordinate_size+source_size;let runtime_bytes=raw[asset_at..asset_at+runtime_size];let frame_bytes=raw[asset_at+runtime_size..asset_at+runtime_size+frame_size];let output=raw[asset_at+runtime_size+frame_size..];
 var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=coordinate.len}).is_ok);let table=coordinates::assemble(copy(coordinate),rows);
 var usize asset_work=0;let runtime=switch(runtime::load(copy(runtime_bytes),ref mut asset_work)){case .ok(p):{p}case .err(_):{return 4;}};asset_work=0;let framing=switch(frame::load(copy(frame_bytes),ref mut asset_work)){case .ok(p):{p}case .err(_):{return 5;}};
 let checked=switch(current(actual,ref table,owner,(mode&2)!=0)){case .ok(p):{p}case .err(_):{return 6;}};let hash=layout::at(program::layouts(graph::program(core::checked_state_graph(ref checked))),model::header(program::root(graph::program(core::checked_state_graph(ref checked)))).layout).hash;
 var prepare_work=transport::empty_work();let prepared=switch(transport::prepare(checked,ref runtime,ref framing,b"x86_64-windows-msvc",ref mut prepare_work)){case .ok(p):{p}case .err(_):{return 7;}};
 let snapshot=switch(response_snapshot(ref prepared,hash,mode)){case .ok(p):{p}case .err(_):{return 8;}};var input_work=request::empty_work();let packet=switch(input::encode(ref prepared,ref snapshot,limits,ref mut input_work)){case .ok(p):{p}case .err(_):{return 9;}};
 // Mode 64 publishes the actual Fairy C and immutable request for execution.
 if((mode&64)!=0){number(0);number(prepare_work.entry);number(prepare_work.framing);blob(transport::source(ref prepared));blob(transport::program_hash(ref prepared));blob(input::content(ref packet));blob(input::digest(ref packet));blob(native::program_hash(transport::entry(ref prepared)));return 0;}
 if((mode&256)!=0){unsafe{allocation_failure(mode/512)};}
 let decoded=response::decode(ref prepared,ref packet,output,ref mut work);if((mode&256)!=0){unsafe{allocation_failure(18446744073709551615)};}
 let admitted=switch(decoded){case .ok(p):{p}case .err(code):{number(code as usize);number(work.structure);number(work.bytes);return 0;}};
 // Result reads/writes/trap must survive destruction of the received bytes.
 {let changed=bytes::view_mut(ref mut incoming);var usize j=0;while(j<changed.len){changed[j]=255;j+=1;}}incoming=bytes::empty();
 number(0);number(work.structure);number(work.bytes);let h=response::header(ref admitted);number(h.status);number(h.exit);number(h.trap);number(h.cost);number(h.operations);number(h.write_bytes);number(response::version(ref admitted));blob(response::trap_category(ref admitted));
 number(response::read_count(ref admitted));var usize i=0;while(i<response::read_count(ref admitted)){let row=response::read_at(ref admitted,i);number(row.operation);number(if(row.found){1}else{0});number(row.codec as usize);blob(response::read_value(ref admitted,i));i+=1;}
 number(response::write_count(ref admitted));i=0;while(i<response::write_count(ref admitted)){let row=response::write_at(ref admitted,i);var usize meter=0;let key=switch(identities::words(row.key,ref mut meter)){case .ok(p):{p}case .err(_):{return 10;}};blob(bytes::view(ref key));number(if(row.deleted){1}else{0});blob(response::write_value(ref admitted,i));i+=1;}return 0;
}`
}

// The existing deterministic SDK heap adapter counts every real allocation.
// Inspect it after the Fairy entry returns, when all its owners must be gone.
func graphResponseNativeProbe(t *testing.T, verify func(*testing.T, string)) {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_state_graph_response")
	writeTestSource(t, root, "src/main.fy", graphResponseProbe())
	build, support := candidateParityBuildPartsWithLimit(t, root, map[string]bool{"fairyc_test_graph_failure": true}, 512<<10)
	allocator, err := os.ReadFile("../cstdlib/stage0_system_heap_probe_allocator.c")
	if err != nil {
		t.Fatal(err)
	}
	support = string(allocator) + "\nvoid fairyc_test_graph_failure(size_t successes) { if (successes == (size_t)-1) { fy_std_system_heap_probe_failure_armed_v0 = 0; } else { fy_std_system_heap_probe_fail_after_allocations_v0(successes); } }\n" + support
	renamed := 0
	for i, part := range build.Sources {
		renamed += strings.Count(part, "int main(void) {")
		build.Sources[i] = strings.ReplaceAll(part, "int main(void) {", "int fairyc_response_fixture_main(void) {")
	}
	if renamed != 1 {
		t.Fatal("missing unique native test entry", renamed)
	}
	driver := `#include "fy_project.h"
extern int fairyc_response_fixture_main(void);
extern size_t fy_std_system_heap_probe_live_allocations_v0(void);
int main(void) { int code = fairyc_response_fixture_main(); if (fy_std_system_heap_probe_live_allocations_v0() != 0u) { return 123; } return code; }
`
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			built, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, driver, build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "graph-response-"+profile.Name+".exe")
			if err = os.WriteFile(path, built.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			verify(t, path)
		})
	}
}

type graphResponseReader struct {
	t *testing.T
	r *bufio.Reader
}

func (r graphResponseReader) number() uint64 {
	r.t.Helper()
	line, err := r.r.ReadString('\n')
	if err != nil {
		r.t.Fatal("response result number", err, line)
	}
	n, err := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
	if err != nil {
		r.t.Fatal(err)
	}
	return n
}
func (r graphResponseReader) blob() []byte {
	r.t.Helper()
	n := r.number()
	if n > 64<<20 {
		r.t.Fatal("unbounded admitted bytes", n)
	}
	p := make([]byte, n)
	if _, err := io.ReadFull(r.r, p); err != nil {
		r.t.Fatal(err)
	}
	if b, err := r.r.ReadByte(); err != nil || b != '\n' {
		r.t.Fatal("blob delimiter", b, err)
	}
	return p
}

func graphResponseRead(t *testing.T, output string) (uint64, [2]uint64, contractstatenative.GraphResult) {
	t.Helper()
	r := graphResponseReader{t, bufio.NewReader(strings.NewReader(output))}
	code, work := r.number(), [2]uint64{r.number(), r.number()}
	var result contractstatenative.GraphResult
	if code == 0 {
		status, exit := r.number(), r.number()
		_ = r.number() // numeric trap is checked against the sealed path by Fairy
		result.Status = byte(status)
		result.Cost, result.Operations, result.WriteBytes, result.Version = r.number(), r.number(), r.number(), r.number()
		result.TrapCategory = string(r.blob())
		if status == 0 {
			result.Exit = []contractstatemir.ExitKind{contractstatemir.ExitCommit, contractstatemir.ExitReject, contractstatemir.ExitTrap}[exit]
		}
		for count := r.number(); count > 0; count-- {
			op, found, codec := r.number(), r.number(), r.number()
			ids := []string{"", "bool/v1", "u32/be/v1", "u64/be/v1", "bytes20/v1", "bytes32/v1", "bytes/v1", "utf8/v1"}
			if codec == 0 || codec >= uint64(len(ids)) {
				t.Fatal("invalid read codec", codec)
			}
			result.Reads = append(result.Reads, contractstatemir.Read{Operation: int(op), Found: found == 1, Value: contractstatemir.Value{TypeID: ids[codec], Encoded: r.blob()}})
		}
		for count := r.number(); count > 0; count-- {
			key := r.blob()
			if len(key) != 32 {
				t.Fatal("invalid write key", key)
			}
			result.Writes = append(result.Writes, contractstate.PersistentWrite{Key: contractstate.Key(key), Deleted: r.number() == 1, Value: r.blob()})
		}
	}
	if extra, _ := io.ReadAll(r.r); len(extra) != 0 {
		t.Fatalf("trailing response result %q", extra)
	}
	return code, work, result
}

func graphResponsePacket(f stateBodyFixture, source string, runtime, frame, output []byte, mode, owner uint64, work [2]uint64, limits contractstate.Limits) []byte {
	coordinate := "fypc1-sha256-" + hex.EncodeToString(f.coordinate[:])
	var data []byte
	for _, n := range []uint64{mode, owner, work[0], work[1], uint64(limits.MaxWrites), limits.MaxWriteBytes, uint64(limits.MaxValueBytes), uint64(limits.MaxMapKeyBytes), uint64(len(coordinate)), uint64(len(source)), uint64(len(runtime)), uint64(len(frame)), uint64(len(output))} {
		data = binary.BigEndian.AppendUint64(data, n)
	}
	data = append(data, coordinate...)
	data = append(data, source...)
	data = append(data, runtime...)
	data = append(data, frame...)
	return append(data, output...)
}

func TestCandidateStateGraphResponse(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native graph response; Linux execution is batched")
	}
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	graphResponseNativeProbe(t, func(t *testing.T, image string) {
		processes, images := 0, 0
		trials := append(graphTransportTrials(), stateGraphAuthoredTrial{"snapshot-above-literal-cap", stateGraphAuthoredSource("storage_read", `storage::read("blob");storage::commit();`, ""), ""})
		var growth strings.Builder
		for i := 0; i < 20; i++ {
			fmt.Fprintf(&growth, `storage::write("values",%d,%d);`, i, i+7)
		}
		for i := 0; i < 20; i++ {
			fmt.Fprintf(&growth, `storage::read("values",%d);`, i)
		}
		for i := 0; i < 10; i++ {
			fmt.Fprintf(&growth, `storage::delete("values",%d);`, i)
		}
		growth.WriteString(`storage::commit();`)
		trials = append(trials, stateGraphAuthoredTrial{"overlay-table-growth", stateGraphAuthoredSource("storage_read,storage_write", growth.String(), ""), ""})
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				f, code := stateGraphAuthoredOracle(t, trial)
				if code != "" {
					t.Fatal(code)
				}
				lowered, err := contractstatemir.LowerHIRTransactions(f.input)
				if err != nil || len(lowered) != 1 {
					t.Fatal("independent response program", err)
				}
				_, _, _, handoff := graphNativeOracle(t, f, true)
				prepared, err := contractstatenative.PrepareGraphTransport(handoff, target.X8664WindowsMSVC)
				if err != nil {
					t.Fatal(err)
				}
				mode := uint64(0)
				if trial.name == "seeded-branch" {
					mode = 16
				}
				if trial.name == "snapshot-above-literal-cap" {
					mode = 65536
				}
				limits := graphNativeLimits()
				invoke := func(output []byte, extraMode, owner uint64, work [2]uint64) string {
					t.Helper()
					processes++
					code, outputText := candidateAtlasProcess(t, image, graphResponsePacket(f, trial.text, runtimePack, framePack, output, mode|extraMode, owner, work, limits))
					if code != 0 {
						t.Fatalf("Fairy response process=%d %q", code, outputText)
					}
					return outputText
				}
				generated := graphTransportRead(t, invoke(nil, 64, 0, [2]uint64{}))
				if generated.code != 0 {
					t.Fatal("prepare", generated.code)
				}
				// Execute Fairy-produced C and request, rather than a manufactured
				// success frame. Go decoding remains only an independent test oracle.
				tc, err := cdriver.ResolveWindowsHostMSVC()
				if err != nil {
					t.Fatal(err)
				}
				profile := cdriver.LowProfile
				if strings.Contains(t.Name(), "/optimized/") {
					profile = cdriver.OptimizedProfile
				}
				support, err := cstdlib.ContractStateGraphRuntimeSource()
				if err != nil {
					t.Fatal(err)
				}
				built, err := cdriver.BuildWindowsMSVCTarget(tc, profile, string(generated.blobs[0]), support)
				if err != nil {
					t.Fatal(err)
				}
				images++
				outputLimit := 64 << 10
				if mode == 65536 {
					outputLimit = 64 << 20
				}
				out, err := cdriver.ExecuteWindowsMSVCTargetWithContextAndOutputLimit(t.Context(), built, bytes.NewReader(generated.blobs[2]), time.Minute, outputLimit)
				if err != nil || out.ExitCode != 0 || out.Stderr != "" {
					t.Fatal("native graph response", err, out)
				}
				output := []byte(out.Stdout)
				snapshot := contractstate.PersistentSnapshot{LayoutHash: graphNativeLayout(t, handoff)}
				if mode == 65536 {
					for _, field := range lowered[0].Program.Layout.Fields {
						if field.Kind == contractstate.FieldCell && field.Name == "blob" {
							snapshot.Version = 73
							snapshot.Entries = []contractstate.Entry{{Key: field.Prefix, Value: make([]byte, 2<<20)}}
						}
					}
				}
				if mode == 16 {
					lowered, err := contractstatemir.LowerHIRTransactions(f.input)
					if err != nil {
						t.Fatal(err)
					}
					for _, field := range lowered[0].Program.Layout.Fields {
						if field.Kind == contractstate.FieldCell && field.Name == "flag" {
							snapshot.Version = 1
							snapshot.Entries = []contractstate.Entry{{Key: field.Prefix, Value: []byte{1}}}
						}
					}
				}
				request, err := prepared.Input(snapshot, limits)
				if err != nil || !bytes.Equal(request.Bytes(), generated.blobs[2]) {
					t.Fatal("actual Fairy request mismatch", err)
				}
				want, err := prepared.Decode(request, output)
				if err != nil {
					t.Fatal("Go oracle admission", err)
				}
				admit, cost, got := graphResponseRead(t, invoke(output, 0, 0, [2]uint64{}))
				if admit != 0 || !reflect.DeepEqual(got, want) {
					t.Fatalf("Fairy response code=%d got=%+v want=%+v", admit, got, want)
				}
				if mode == 0 {
					graphNativeSemanticExpect(t, f.input, got)
				}
				if mode == 65536 && (got.Status != 0 || got.Exit != contractstatemir.ExitCommit || got.Version != 73 || got.Cost != 2 || got.Operations != 1 || len(got.Reads) != 1 || !got.Reads[0].Found || !bytes.Equal(got.Reads[0].Value.Encoded, make([]byte, 2<<20)) || len(got.Writes) != 0) {
					t.Fatal("large runtime read was narrowed by literal/write limits", got.Status, got.Version, got.Cost, len(got.Reads))
				}
				if mode == 16 {
					// Same layout/body/program/version, different snapshot bytes.
					code, _, _ := graphResponseRead(t, invoke(output, 128, 0, [2]uint64{}))
					if code != 1001 {
						t.Fatal("same-version snapshot splice admitted", code)
					}
				}
				if trial.name == "helper-transitive" {
					for name, prefix := range graphResponseFailurePrefixes(output) {
						t.Run(name, func(t *testing.T) {
							want, err := prepared.Decode(request, prefix)
							if err != nil {
								t.Fatal("independent failure-prefix oracle", err)
							}
							code, _, got := graphResponseRead(t, invoke(prefix, 0, 0, [2]uint64{}))
							if code != 0 || !reflect.DeepEqual(got, want) {
								t.Fatalf("failure-prefix code=%d got=%+v want=%+v", code, got, want)
							}
						})
					}
				}
				if trial.name == "helper-transitive" || trial.name == "seeded-branch" || trial.name == "present-borrow-copy" {
					for _, check := range []struct {
						name        string
						mode, owner uint64
						work        [2]uint64
						want        uint64
					}{
						{"restored-owner", 2, 511, [2]uint64{}, 0},
						{"exact-work", 0, 0, [2]uint64{16777216 - cost[0], 1073741824 - cost[1]}, 0},
						{"short-structure", 0, 0, [2]uint64{16777217 - cost[0], 0}, 1002},
						{"short-bytes", 0, 0, [2]uint64{0, 1073741825 - cost[1]}, 1002},
						{"overdrawn-structure", 0, 0, [2]uint64{16777217, 0}, 1002},
						{"overdrawn-bytes", 0, 0, [2]uint64{0, 1073741825}, 1002},
					} {
						t.Run(check.name, func(t *testing.T) {
							code, _, result := graphResponseRead(t, invoke(output, check.mode, check.owner, check.work))
							if code != check.want || code == 0 && !reflect.DeepEqual(result, want) {
								t.Fatalf("work/owner admission=%d want=%d", code, check.want)
							}
						})
					}
				}
				if trial.name == "present-borrow-copy" || trial.name == "overlay-table-growth" {
					failures := 0
					for n := uint64(0); n < 48; n++ {
						code, _, result := graphResponseRead(t, invoke(output, 256+n*512, 0, [2]uint64{}))
						if code == 1003 {
							failures++
							continue
						}
						if code != 0 || !reflect.DeepEqual(result, want) {
							t.Fatalf("allocation failure %d code=%d result=%+v", n, code, result)
						}
					}
					if failures < 4 {
						t.Fatal("allocation failures did not reach owned response storage", failures)
					}
					t.Logf("deterministic allocation failures=%d; all native calls assert zero live SDK heap blocks after owner cleanup", failures)
				}
				switch trial.name {
				case "empty-commit", "helper-transitive", "seeded-branch", "present-borrow-copy", "map-only", "branch-present", "six-storage-actions", "text-unicode", "trap":
					for name, corrupted := range graphResponseMutations(output, lowered[0].Program) {
						t.Run(name, func(t *testing.T) {
							_, oracleErr := prepared.Decode(request, corrupted)
							code, _, _ := graphResponseRead(t, invoke(corrupted, 0, 0, [2]uint64{}))
							if oracleErr == nil || code != 1001 {
								t.Fatalf("forged response admission=%d oracle=%v", code, oracleErr)
							}
						})
					}
				}
			})
		}
		t.Logf("Fairy response processes=%d actual Fairy C images=%d", processes, images)
	})
}

func graphResponseMutations(output []byte, program contractstatemir.Program) map[string][]byte {
	mutations := make(map[string][]byte)
	var operations []contractstatemir.Operation
	for _, block := range program.Blocks {
		operations = append(operations, block.Operations...)
	}
	for _, helper := range program.Helpers {
		for _, block := range helper.Blocks {
			operations = append(operations, block.Operations...)
		}
	}
	for _, site := range []struct {
		name string
		at   int
	}{
		{"magic", 0}, {"layout", 8}, {"body", 40}, {"program", 72}, {"version", 104}, {"request-digest", 112},
		{"status", 144}, {"exit", 145}, {"trap", 149}, {"cost", 157}, {"operations", 165}, {"events", 169}, {"writes", 173}, {"write-bytes", 181}, {"payload", 185},
	} {
		p := bytes.Clone(output)
		p[site.at] ^= 0x80
		mutations[site.name] = p
	}
	mutations["truncated"] = bytes.Clone(output[:len(output)-1])
	mutations["trailing"] = append(bytes.Clone(output), 0)
	if binary.BigEndian.Uint32(output[166:170]) > 0 {
		for _, at := range []int{193, 197, 201, 205, 206} {
			p := bytes.Clone(output)
			p[at] ^= 0x80
			mutations[fmt.Sprintf("event-byte-%d", at)] = p
		}
	}
	// Mutations preserve framing and counters to reach observation/overlay
	// admission. Original literals, received reads and final writes must agree.
	at := 186
	for i, n := 0, int(binary.BigEndian.Uint32(output[166:170])); i < n; i++ {
		keySize := int(binary.BigEndian.Uint32(output[at+12 : at+16]))
		valueSize := int(binary.BigEndian.Uint32(output[at+16 : at+20]))
		site := binary.BigEndian.Uint32(output[at+8 : at+12])
		literalOrRead := false
		if site > 0 && site <= uint32(len(operations)) {
			op := operations[site-1]
			literalOrRead = op.ValueInput == 0 || op.Kind == contractstatemir.OperationCellRead || op.Kind == contractstatemir.OperationMapRead || op.Kind == contractstatemir.OperationCellBorrow || op.Kind == contractstatemir.OperationMapBorrow
		}
		if valueSize > 0 && literalOrRead {
			p := bytes.Clone(output)
			p[at+21+keySize+valueSize-1] ^= 1
			mutations[fmt.Sprintf("observation-value-%d", i)] = p
		}
		if keySize > 0 && site > 0 && site <= uint32(len(operations)) && operations[site-1].KeyInput == 0 {
			p := bytes.Clone(output)
			p[at+21+keySize-1] ^= 1
			mutations[fmt.Sprintf("literal-map-key-%d", i)] = p
		}
		at += 21 + keySize + valueSize
	}
	for i, n := 0, int(binary.BigEndian.Uint32(output[170:174])); i < n; i++ {
		size := int(binary.BigEndian.Uint32(output[at+33 : at+37]))
		for _, part := range []struct {
			name string
			at   int
		}{{"key", at}, {"deleted", at + 32}} {
			p := bytes.Clone(output)
			p[part.at] ^= 1
			mutations[fmt.Sprintf("final-%s-%d", part.name, i)] = p
		}
		if size > 0 {
			p := bytes.Clone(output)
			p[at+37+size-1] ^= 1
			mutations[fmt.Sprintf("final-value-%d", i)] = p
		}
		at += 37 + size
	}
	return mutations
}

func graphResponseFailurePrefixes(output []byte) map[string][]byte {
	frames := map[string][]byte{}
	at := 186
	events := int(binary.BigEndian.Uint32(output[166:170]))
	keep := min(2, events-1)
	var operations uint64
	for i := 0; i < keep; i++ {
		operations = binary.BigEndian.Uint64(output[at : at+8])
		at += 21 + int(binary.BigEndian.Uint32(output[at+12:at+16])) + int(binary.BigEndian.Uint32(output[at+16:at+20]))
	}
	for status := byte(1); status <= 11; status++ {
		p := bytes.Clone(output[:at])
		p[144] = status
		p[145] = 0
		clear(p[146:150])
		binary.BigEndian.PutUint64(p[150:158], uint64(keep))
		binary.BigEndian.PutUint64(p[158:166], operations)
		binary.BigEndian.PutUint32(p[166:170], uint32(keep))
		clear(p[170:182])
		binary.BigEndian.PutUint32(p[182:186], uint32(at-186))
		frames[fmt.Sprintf("status-%d-prefix", status)] = p
		lost := bytes.Clone(p)
		binary.BigEndian.PutUint64(lost[150:158], uint64(keep+1))
		extra := uint64(0)
		if binary.BigEndian.Uint32(output[at+8:at+12]) < 1<<31 {
			extra = 1
		}
		binary.BigEndian.PutUint64(lost[158:166], operations+extra)
		frames[fmt.Sprintf("status-%d-lost-next-event", status)] = lost
	}
	return frames
}

func TestCandidateStateGraphResponseSealedOwner(t *testing.T) {
	for _, body := range []string{
		`fn response::Admitted forge(){return response::Admitted{.header=0,.version=0,.evidence=0,.category=bytes::empty()};}`,
		`fn void forge(ref mut response::Admitted r){r.version=99;}`,
	} {
		root := t.TempDir()
		copyLexerExample(t, root)
		copyCandidateParityModules(t, root, "module_state_graph_response")
		writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_response as response;import std::bytes as bytes;"+body+"fn i32 main(){return 0;}")
		workspace, err := workspaceloader.LoadVectorCandidate(root)
		if err != nil {
			t.Fatal(err)
		}
		var diagnostics bytes.Buffer
		_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed graph response", &diagnostics)
		if code == ExitSuccess || !strings.Contains(diagnostics.String(), "E0103") {
			t.Fatalf("owner code=%d: %s", code, &diagnostics)
		}
	}
}

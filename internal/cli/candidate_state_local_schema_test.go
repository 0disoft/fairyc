package cli

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
)

// No expected Plan is compiled into this probe: every layout, including a
// foreign UID and an appended schema, comes only from the received Go record.
const stateLocalSchemaProbe = `module main;
import module_state_local_schema as schema;import module_state_local_codec as local;import module_state_layout as layout;import project_identity_bytes as identities;import std::bytes as bytes;import std::io as io;
@effects(console)fn void number(usize n){assert(io::write_usize(n).is_ok && io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok && io::write("\n").is_ok);}
@effects(console)fn void words(u32[8] hash){var usize meter=0;let data=switch(identities::words(hash,ref mut meter)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};blob(bytes::view(ref data));}
@effects(console)fn void result(u32 code,usize stage,local::Work work){number(code as usize);number(stage);number(work.structure);number(work.bytes);}
@effects(console)fn void show(ref layout::Plan plan,usize size){assert(layout::count(plan)==1);let l=layout::at(plan,0);assert(l.owner==0 && l.module_index==0 && l.declaration==1 && l.start==0 && l.end==size && l.uid_start==0 && l.field_start==0);blob(layout::uid(plan,0));words(l.hash);number(l.field_count);var usize i=0;var usize end=l.uid_end;while(i<l.field_count){let f=layout::field_at(plan,i);assert(f.start==0 && f.end==0 && f.name_start==end);end=f.name_end;blob(layout::name(plan,i));number(f.kind as usize);blob(layout::codec(f.key));blob(layout::codec(f.value));words(f.prefix);i+=1;}assert(end==layout::name_bytes(plan) && layout::field_count(plan)==l.field_count);}
fn usize n64(ref u8[] raw,usize at){var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[at+i] as usize);i+=1;}return n;}
fn void destroy(ref mut bytes::Buffer incoming){let ref mut u8[] data=bytes::view_mut(incoming);var usize i=0;while(i<data.len){data[i]=0;i+=1;}}
@effects(console)fn i32 main(){var incoming=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref incoming)>16777264-n || !bytes::append(ref mut incoming,b[..n]).is_ok){return 2;}}
let raw=bytes::view(ref incoming);if(raw.len<48){return 3;}let mode=n64(raw,0);var work=local::Work{.structure=n64(raw,8),.bytes=n64(raw,16)};let limits=local::Limits{.bytes=n64(raw,24),.entries=n64(raw,32),.value=n64(raw,40)};let size=raw.len-48;
let plan=switch(schema::read(raw[48..],limits,ref mut work)){case .ok(p):{p}case .err(code):{result(code,0,work);return 0;}};
if(mode==1){destroy(ref mut incoming);incoming=bytes::empty();result(0,2,work);show(ref plan,size);return 0;}
// Full admission must immediately follow header reconstruction on the SAME Work.
let state=switch(local::decode(ref plan,0,raw[48..],limits,ref mut work)){case .ok(p):{p}case .err(code):{result(code,1,work);return 0;}};
// Destroy source storage before reading any copied Plan names or state bytes.
destroy(ref mut incoming);incoming=bytes::empty();var writing=local::empty_work();let encoded=switch(local::encode(ref state,limits,ref mut writing)){case .ok(p):{p}case .err(_):{return 4;}};
result(0,2,work);show(ref plan,size);number(local::version(ref state));number(local::count(ref state));blob(bytes::view(ref encoded));return 0;}
`

func stateLocalSchemaFixture(t *testing.T, uid string, count int) contractstate.Layout {
	t.Helper()
	builder, err := contractstate.NewLayoutBuilder(uid)
	if err != nil {
		t.Fatal(err)
	}
	codecs := []string{"bool/v1", "u32/be/v1", "u64/be/v1", "bytes20/v1", "bytes32/v1", "bytes/v1", "utf8/v1"}
	for i := 0; i < count; i++ {
		if i == 0 {
			_, err = contractstate.DeclareEncodedCell(builder, "flag", "bool/v1")
		} else {
			_, err = contractstate.DeclareEncodedMap(builder, fmt.Sprintf("data%d", i), codecs[(i-1)%7], codecs[(i+2)%7])
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	return layout
}

func TestCandidateStateLocalSchema(t *testing.T) {
	old := stateLocalSchemaFixture(t, "fixture.local-schema", 1)
	next := stateLocalSchemaFixture(t, old.ContractUID(), 8)
	if next.Extends(old) != nil || old.Hash() == next.Hash() {
		t.Fatal("append-only oracle")
	}
	foreign := stateLocalSchemaFixture(t, "another.contract-9", 8)
	empty := stateLocalSchemaFixture(t, "fixture.empty-schema", 0)
	maxUID := stateLocalSchemaFixture(t, strings.Repeat("a", 255), 1)
	maxFields := stateLocalSchemaFixture(t, "fixture.1024-fields", 1024)
	tooMany := stateLocalSchemaFixture(t, "fixture.1025-fields", 1025)
	nameBuilder, err := contractstate.NewLayoutBuilder("fixture.name-bound")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contractstate.DeclareEncodedCell(nameBuilder, strings.Repeat("n", 127), "utf8/v1"); err != nil {
		t.Fatal(err)
	}
	maxName, err := nameBuilder.Build()
	if err != nil {
		t.Fatal(err)
	}
	var rows []contractstate.Entry
	for i, value := range [][]byte{nil, {}, {0, 255, 13, 10, 26}, []byte("schema bytes")} {
		var key contractstate.Key
		key[31] = byte(i + 1)
		rows = append(rows, contractstate.Entry{Key: key, Value: value})
	}
	policy := [3]uint64{16 << 20, 65536, 65536}
	canonical := stateLocalCodecRecord(t, next, math.MaxUint64, rows)
	body := canonical[:len(canonical)-32]

	candidateAtlasProbeSize(t, "module_state_local_schema", stateLocalSchemaProbe, 128<<10, func(t *testing.T, image string) {
		// mode=1 inspects only the header; mode=0 admits and re-encodes the full
		// record. The result stage distinguishes header failure from decode failure.
		run := func(mode, structure, byteWork uint64, limits [3]uint64, record []byte, wantCode, stage uint64, schema contractstate.Layout, version uint64, entries int) [2]uint64 {
			t.Helper()
			var packet []byte
			for _, n := range []uint64{mode, structure, byteWork, limits[0], limits[1], limits[2]} {
				packet = binary.BigEndian.AppendUint64(packet, n)
			}
			exit, output := candidateAtlasProcess(t, image, append(packet, record...))
			if exit != 0 {
				t.Fatalf("schema exit=%d output=%q", exit, output[:min(len(output), 256)])
			}
			r := bufio.NewReader(strings.NewReader(output))
			number := func() uint64 {
				t.Helper()
				line, err := r.ReadString('\n')
				n, parse := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
				if err != nil || parse != nil {
					t.Fatalf("schema number %q: %v %v", line, err, parse)
				}
				return n
			}
			blob := func() []byte {
				t.Helper()
				n := number()
				if n > 16<<20 {
					t.Fatalf("unexpected blob size %d", n)
				}
				data := make([]byte, n)
				if _, err := io.ReadFull(r, data); err != nil {
					t.Fatal(err)
				}
				if b, err := r.ReadByte(); err != nil || b != '\n' {
					t.Fatal("blob terminator", err)
				}
				return data
			}
			code, gotStage := number(), number()
			cost := [2]uint64{number(), number()}
			if code != wantCode || gotStage != stage {
				t.Fatalf("schema code/stage=%d/%d want=%d/%d work=%v", code, gotStage, wantCode, stage, cost)
			}
			if cost[0] < structure || cost[1] < byteWork {
				t.Fatalf("caller work reset: %v < %d/%d", cost, structure, byteWork)
			}
			if code == 0 {
				if got := string(blob()); got != schema.ContractUID() {
					t.Fatalf("UID %q", got)
				}
				hash := schema.Hash()
				if !bytes.Equal(blob(), hash[:]) || number() != uint64(len(schema.Fields())) {
					t.Fatal("layout hash or field count")
				}
				for _, field := range schema.Fields() {
					if string(blob()) != field.Name || number() != uint64(field.Kind) || string(blob()) != field.KeyTypeID || string(blob()) != field.ValueTypeID || !bytes.Equal(blob(), field.Prefix[:]) {
						t.Fatalf("field %d identity", field.Index)
					}
				}
				if mode == 0 && (number() != version || number() != uint64(entries) || !bytes.Equal(blob(), record)) {
					t.Fatal("full decode / re-encode differs from Go oracle")
				}
			}
			if rest, _ := io.ReadAll(r); len(rest) != 0 {
				t.Fatal("trailing result", string(rest))
			}
			return cost
		}
		for _, trial := range []struct {
			name   string
			schema contractstate.Layout
		}{
			{"old", old}, {"append-all-seven", next}, {"foreign", foreign}, {"zero-fields", empty},
			{"uid-255", maxUID}, {"name-127", maxName}, {"fields-1024", maxFields},
		} {
			t.Run(trial.name, func(t *testing.T) {
				run(0, 0, 0, policy, stateLocalCodecRecord(t, trial.schema, math.MaxUint64, rows), 0, 2, trial.schema, math.MaxUint64, len(rows))
			})
		}
		t.Run("fields-1025", func(t *testing.T) {
			run(1, 0, 0, policy, stateLocalCodecRecord(t, tooMany, 0, nil), 1002, 0, empty, 0, 0)
		})
		t.Run("record-above-eight-MiB", func(t *testing.T) {
			large := []contractstate.Entry{{Value: bytes.Repeat([]byte{0xa5}, (6<<20)+1)}}
			record := stateLocalCodecRecord(t, foreign, 7, large)
			if len(record) <= 8<<20 {
				t.Fatal("large record fixture")
			}
			run(0, 0, 0, [3]uint64{16 << 20, 65536, 16 << 20}, record, 0, 2, foreign, 7, 1)
		})
		for _, mode := range []uint64{0, 1} {
			t.Run(fmt.Sprintf("work-mode-%d", mode), func(t *testing.T) {
				cost := run(mode, 0, 0, policy, canonical, 0, 2, next, math.MaxUint64, len(rows))
				if cost[0] < 1<<20 || cost[1] < uint64(len(canonical)) {
					t.Fatal("scan / bounded validation was not prepaid", cost)
				}
				got := run(mode, (16<<20)-cost[0], (1<<30)-cost[1], policy, canonical, 0, 2, next, math.MaxUint64, len(rows))
				if got != [2]uint64{16 << 20, 1 << 30} {
					t.Fatal("exact ledger", got)
				}
				failStage := uint64(0)
				if mode == 0 {
					failStage = 1
				}
				run(mode, (16<<20)+1-cost[0], 0, policy, canonical, 1002, failStage, empty, 0, 0)
				run(mode, 0, (1<<30)+1-cost[1], policy, canonical, 1002, failStage, empty, 0, 0)
			})
		}
		for _, work := range [][2]uint64{{(16 << 20) + 1, 0}, {0, (1 << 30) + 1}, {math.MaxUint64, 0}, {0, math.MaxUint64}, {0, (1 << 30) - uint64(len(canonical)) + 1}} {
			t.Run(fmt.Sprintf("work-overdrawn-%v", work), func(t *testing.T) { run(1, work[0], work[1], policy, canonical, 1002, 0, empty, 0, 0) })
		}
		for i, limits := range [][3]uint64{{0, 1, 1}, {(16 << 20) + 1, 1, 1}, {1, 0, 1}, {65536, 65537, 1}, {65536, 1, 0}, {65536, 1, 65537}} {
			t.Run(fmt.Sprintf("invalid-policy-%d", i), func(t *testing.T) { run(1, 17, 29, limits, canonical, 1001, 0, empty, 0, 0) })
		}
		t.Run("exact-file-bound", func(t *testing.T) {
			record := stateLocalCodecRecord(t, next, 0, nil)
			run(0, 0, 0, [3]uint64{uint64(len(record)), 65536, 1}, record, 0, 2, next, 0, 0)
		})
		t.Run("one-short-file-bound", func(t *testing.T) {
			run(1, 0, 0, [3]uint64{uint64(len(canonical) - 1), 65536, 1}, canonical, 1002, 0, empty, 0, 0)
		})
		mutate := func(from, to string) []byte {
			t.Helper()
			if !bytes.Contains(body, []byte(from)) {
				t.Fatalf("missing mutation %q", from)
			}
			return stateLocalCodecChecksum(bytes.Replace(body, []byte(from), []byte(to), 1))
		}
		for _, trial := range []struct{ name, from, to string }{
			{"schema", "fairyc.local-state/v1", "fairyc.local-state/v2"},
			{"unknown-header", `,"fields":`, `,"extra":0,"fields":`},
			{"unknown-field", `,"Name":`, `,"Other":0,"Name":`},
			{"unknown-before-version", `,"version":`, `,"other":0,"version":`},
			{"reordered-field", `{"Index":0,"Name":"flag"`, `{"Name":"flag","Index":0`},
			{"bad-index", `"Index":0`, `"Index":1`}, {"index-overflow", `"Index":0`, `"Index":18446744073709551616`},
			{"index-leading-zero", `"Index":0`, `"Index":00`}, {"index-negative", `"Index":0`, `"Index":-1`},
			{"index-range", `"Index":0`, `"Index":1024`}, {"kind-zero", `"Kind":1`, `"Kind":0`},
			{"kind-range", `"Kind":1`, `"Kind":3`}, {"kind-leading-zero", `"Kind":1`, `"Kind":01`},
			{"uid-empty", old.ContractUID(), ""}, {"uid-256", old.ContractUID(), strings.Repeat("u", 256)},
			{"uid-invalid", old.ContractUID(), "bad/uid"}, {"uid-escape", old.ContractUID(), `bad\u002euid`},
			{"uid-nonascii", old.ContractUID(), "한글"}, {"uid-control", old.ContractUID(), "bad\nuid"},
			{"uid-digest", old.ContractUID(), "fixture.other-schema"},
			{"name-empty", `"Name":"flag"`, `"Name":""`}, {"name-128", `"Name":"flag"`, `"Name":"` + strings.Repeat("n", 128) + `"`},
			{"name-invalid", `"Name":"flag"`, `"Name":"0flag"`}, {"name-escape", `"Name":"flag"`, `"Name":"fl\u0061g"`},
			{"name-digest", `"Name":"flag"`, `"Name":"flap"`}, {"name-duplicate", `"Name":"data2"`, `"Name":"data1"`},
			{"cell-key", `"KeyTypeID":""`, `"KeyTypeID":"bool/v1"`}, {"map-key-empty", `"KeyTypeID":"bool/v1"`, `"KeyTypeID":""`},
			{"unknown-key-codec", `"KeyTypeID":"bool/v1"`, `"KeyTypeID":"no/v1"`},
			{"unknown-value-codec", `"ValueTypeID":"bool/v1"`, `"ValueTypeID":"no/v1"`},
			{"empty-value-codec", `"ValueTypeID":"bool/v1"`, `"ValueTypeID":""`},
			{"codec-escape", `"ValueTypeID":"bool/v1"`, `"ValueTypeID":"bool\/v1"`},
			{"codec-too-long", `"ValueTypeID":"bool/v1"`, `"ValueTypeID":"xxxxxxxxxxx"`},
			{"hash-short", `"layoutHash":[`, `"layoutHash":[]`}, {"hash-byte-range", `"layoutHash":[`, `"layoutHash":[256,`},
			{"hash-overflow", `"layoutHash":[`, `"layoutHash":[18446744073709551616,`},
			{"hash-leading-zero", `"layoutHash":[`, `"layoutHash":[00,`}, {"hash-negative", `"layoutHash":[`, `"layoutHash":[-1,`},
			{"prefix-byte-range", `"Prefix":[`, `"Prefix":[256,`}, {"prefix-short", `"Prefix":[`, `"Prefix":[]`},
			{"space", `,"fields":`, `, "fields":`}, {"array-trailing-comma", `],"version":`, `,],"version":`},
			{"missing-version-colon", `,"version":`, `,"version"`},
		} {
			t.Run(trial.name, func(t *testing.T) { run(1, 17, 29, policy, mutate(trial.from, trial.to), 1001, 0, empty, 0, 0) })
		}
		// Valid decimal arrays of the right length still require digest admission.
		for _, label := range []string{"layoutHash", "Prefix"} {
			t.Run(label+"-digest", func(t *testing.T) {
				start := bytes.Index(body, []byte(`"`+label+`":[`)) + len(label) + 4
				end := start + bytes.IndexByte(body[start:], ',')
				first, err := strconv.Atoi(string(body[start:end]))
				if err != nil {
					t.Fatal(err)
				}
				changed := append(append(append([]byte{}, body[:start]...), []byte(strconv.Itoa(first^1))...), body[end:]...)
				run(1, 0, 0, policy, stateLocalCodecChecksum(changed), 1001, 0, empty, 0, 0)
			})
			for _, array := range []struct{ name, value string }{
				{"31", "[" + strings.Repeat("0,", 30) + "0]"},
				{"33", "[" + strings.Repeat("0,", 32) + "0]"},
				{"32-zero", "[" + strings.Repeat("0,", 31) + "0]"},
			} {
				t.Run(label+"-"+array.name, func(t *testing.T) {
					start := bytes.Index(body, []byte(`"`+label+`":[`)) + len(label) + 3
					end := start + bytes.IndexByte(body[start:], ']') + 1
					changed := append(append(append([]byte{}, body[:start]...), []byte(array.value)...), body[end:]...)
					run(1, 0, 0, policy, stateLocalCodecChecksum(changed), 1001, 0, empty, 0, 0)
				})
			}
		}
		emptyRecord := stateLocalCodecRecord(t, empty, 0, nil)
		t.Run("empty-array-not-null", func(t *testing.T) {
			run(1, 0, 0, policy, stateLocalCodecChecksum(bytes.Replace(emptyRecord[:len(emptyRecord)-32], []byte(`"fields":null`), []byte(`"fields":[]`), 1)), 1001, 0, empty, 0, 0)
		})
		for _, cut := range []int{0, 1, 31, 32, 33, 40, 77, 100, bytes.Index(body, []byte(`"Name"`)) + 10, bytes.Index(body, []byte(`,"version":`)) + 10} {
			t.Run(fmt.Sprintf("truncated-%d", cut), func(t *testing.T) {
				code := uint64(1001)
				if cut < 32 {
					code = 1002
				}
				run(1, 0, 0, policy, canonical[:cut], code, 0, empty, 0, 0)
			})
		}
		// A good structural header is NOT evidence of full record integrity.
		corruptChecksum := append([]byte{}, canonical...)
		corruptChecksum[len(corruptChecksum)-1] ^= 1
		versionAt := bytes.Index(body, []byte(`,"version":`)) + len(`,"version":`)
		for _, trial := range []struct {
			name string
			data []byte
		}{
			{"checksum", corruptChecksum}, {"missing-body", stateLocalCodecChecksum(body[:versionAt])},
			{"bad-body", stateLocalCodecChecksum(append(append([]byte{}, body[:versionAt]...), []byte(`00,"entries":[]}`)...))},
		} {
			t.Run("structural-only-"+trial.name, func(t *testing.T) {
				run(1, 0, 0, policy, trial.data, 0, 2, next, 0, 0)
				run(0, 0, 0, policy, trial.data, 1001, 1, empty, 0, 0)
			})
		}
	})
}

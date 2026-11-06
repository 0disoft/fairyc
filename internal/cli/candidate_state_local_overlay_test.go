package cli

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
)

const stateLocalOverlayProbe = `module main;
import module_state_local_codec as local;import module_state_layout as layout;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn bytes::Buffer owned(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn usize n64(ref u8[] raw,usize at){var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[at+i] as usize);i+=1;}return n;}
@effects(console)fn void number(usize n){assert(io::write_usize(n).is_ok && io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok && io::write("\n").is_ok);}
fn Result<local::State,u32> operation(ref local::State current,ref local::Mutations input,ref layout::Plan next,usize mode,u32[8] hash,usize version,local::Limits limits,ref mut local::Work work){
if(mode==1){return local::upgrade(current,hash,version,next,0,limits,work);}
let before=*work;let writes=local::canonical(input,limits,work)?;assert(local::mutation_count(ref writes)<=limits.entries);
// A second admission through the same borrowed owner has identical cost and bytes.
var repeat=before;let again=local::canonical(input,limits,ref mut repeat)?;assert(repeat.structure==work.structure && repeat.bytes==work.bytes && local::mutation_count(ref writes)==local::mutation_count(ref again));
return local::apply(current,hash,version,ref writes,limits,work);}
@effects(console)fn i32 main(){var incoming=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;assert(bytes::len(ref incoming)<=67108864-n && bytes::append(ref mut incoming,b[..n]).is_ok);}
let saved=owned(bytes::view(ref incoming));let raw=bytes::view(ref incoming);assert(raw.len>=112);let mode=n64(raw,0);var work=local::Work{.structure=n64(raw,8),.bytes=n64(raw,16)};let limits=local::Limits{.bytes=n64(raw,24),.entries=n64(raw,32),.value=n64(raw,40)};
let old=choose(n64(raw,48));let next=choose(n64(raw,56));let wrong=n64(raw,64);let expected=n64(raw,72);let size=n64(raw,80);let count=n64(raw,88);let data_size=n64(raw,96);let selected=n64(raw,104);let record=raw[112..112+size];var decoding=local::empty_work();let wide=local::Limits{.bytes=16777216,.entries=65536,.value=16777216};
let current=switch(local::decode(ref old,0,record,wide,ref mut decoding)){case .ok(p):{p}case .err(_):{return 2;}};var hash=local::layout_hash(ref current);if(wrong!=0){hash[0]=hash[0]^1;}
var rows=v::empty<local::Mutation>();var usize i=0;var usize at=112+size;while(i<count){var u32[8] key=[0;8];var usize j=0;while(j<32){key[j/4]=(key[j/4]<<8)|(raw[at+j] as u32);j+=1;}
assert(v::push<local::Mutation>(ref mut rows,local::Mutation{.key,.start=n64(raw,at+32),.end=n64(raw,at+40),.deleted=n64(raw,at+48)!=0,.was_null=n64(raw,at+56)!=0}).is_ok);at+=64;i+=1;}
let mutations=local::mutations_of(owned(raw[at..at+data_size]),rows);
var encoded=bytes::empty();var u32 status=0;
switch(if(selected==0){operation(ref current,ref mutations,ref next,mode,hash,expected,limits,ref mut work)}else{local::upgrade(ref current,hash,expected,ref next,selected,limits,ref mut work)}){case .err(code):{status=code;}case .ok(state):{var output=local::empty_work();encoded=switch(local::encode(ref state,wide,ref mut output)){case .ok(p):{p}case .err(_):{return 4;}};}};
var checking=local::empty_work();let preserved=switch(local::encode(ref current,wide,ref mut checking)){case .ok(p):{p}case .err(_):{return 3;}};assert(bytes::equal(bytes::view(ref preserved),record) && bytes::equal(bytes::view(ref saved),raw));
number(status as usize);number(work.structure);number(work.bytes);if(status==0){blob(bytes::view(ref encoded));}return 0;}
`

type stateLocalOverlayMutation struct {
	key           contractstate.Key
	start, end    uint64
	deleted, null bool
}

func TestCandidateStateLocalOverlay(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows local overlay native execution; Linux remains batched")
	}
	makeLayout := func(uid string, fields ...contractstate.Field) contractstate.Layout {
		t.Helper()
		b, err := contractstate.NewLayoutBuilder(uid)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fields {
			if f.Kind == contractstate.FieldCell {
				_, err = contractstate.DeclareEncodedCell(b, f.Name, f.ValueTypeID)
			} else {
				_, err = contractstate.DeclareEncodedMap(b, f.Name, f.KeyTypeID, f.ValueTypeID)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		p, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	cell := contractstate.Field{Name: "balance", Kind: contractstate.FieldCell, ValueTypeID: "u64/be/v1"}
	mapping := contractstate.Field{Name: "owners", Kind: contractstate.FieldMap, KeyTypeID: "bytes32/v1", ValueTypeID: "bytes/v1"}
	extra := contractstate.Field{Name: "memo", Kind: contractstate.FieldCell, ValueTypeID: "utf8/v1"}
	renamed, kind, value, key := cell, cell, cell, mapping
	renamed.Name = "renamed"
	kind.Kind = contractstate.FieldMap
	kind.KeyTypeID = "u32/be/v1"
	value.ValueTypeID = "bool/v1"
	key.KeyTypeID = "u32/be/v1"
	schemas := []contractstate.Layout{
		makeLayout("fixture.local-overlay", cell, mapping),
		makeLayout("fixture.local-overlay", cell, mapping, extra),
		makeLayout("fixture.local-overlay", renamed, mapping),
		makeLayout("fixture.local-overlay", mapping, cell),
		makeLayout("fixture.local-overlay", kind, mapping),
		makeLayout("fixture.local-overlay", value, mapping),
		makeLayout("fixture.local-overlay", cell, key),
		makeLayout("fixture.local-overlay", cell),
		makeLayout("fixture.other-overlay", cell, mapping),
		makeLayout("fixture.local-overlay"),
	}
	var plans strings.Builder
	for i, schema := range schemas {
		plans.WriteString(strings.Replace(stateLocalCodecPlan(schema), "fn layout::Plan schemas()", fmt.Sprintf("fn layout::Plan plan%d()", i), 1))
	}
	// Malformed next Plan retains valid field signatures but a false layout digest.
	bad := strings.Replace(stateLocalCodecPlan(schemas[1]), "fn layout::Plan schemas()", "fn layout::Plan invalid()", 1)
	at := strings.Index(bad, ".hash=[")
	end := strings.Index(bad[at:], "]") + at
	bad = bad[:at] + ".hash=[0;8]" + bad[end+1:]
	plans.WriteString(bad)
	plans.WriteString("fn layout::Plan choose(usize n){")
	for i := range schemas {
		fmt.Fprintf(&plans, "if(n==%d){return plan%d();}", i, i)
	}
	plans.WriteString("return invalid();}")
	keyOf := func(n uint32) contractstate.Key {
		var k contractstate.Key
		binary.BigEndian.PutUint32(k[28:], n)
		return k
	}
	base := []contractstate.Entry{{Key: keyOf(1), Value: nil}, {Key: keyOf(2), Value: []byte{}}, {Key: keyOf(4), Value: []byte("old")}, {Key: keyOf(6), Value: []byte{0, 255, 13, 10, 26}}}
	wide := [3]uint64{16 << 20, 65536, 65536}
	type trial struct {
		name                                                string
		mode, old, next, version, expected, wrong, selected uint64
		rows                                                []contractstate.Entry
		writes                                              []contractstate.PersistentWrite
		mutations                                           []stateLocalOverlayMutation
		data                                                []byte
		limits                                              [3]uint64
		code                                                uint64
		want                                                []byte
	}
	// Successful expectations come from the actual Go disk adapter and independent encoder.
	oracle := func(c trial) []byte {
		t.Helper()
		dir := t.TempDir()
		s, err := contractstate.CreateLocalPersistence(dir, schemas[c.old], contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		if err := os.WriteFile(filepath.Join(dir, "state-v1.bin"), stateLocalCodecRecord(t, schemas[c.old], c.version, c.rows), 0600); err != nil {
			t.Fatal(err)
		}
		if c.mode == 1 {
			_, err = s.PublishLayout(schemas[c.old].Hash(), c.expected, schemas[c.next])
		} else {
			_, err = s.CompareAndPublish(schemas[c.old].Hash(), c.expected, c.writes)
		}
		if err != nil {
			t.Fatal(c.name, err)
		}
		encoded, err := os.ReadFile(filepath.Join(dir, "state-v1.bin"))
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	var cases []trial
	add := func(c trial) {
		if c.limits == [3]uint64{} {
			c.limits = wide
		}
		if c.rows == nil {
			c.rows = base
		}
		if c.code == 0 && c.want == nil {
			c.want = oracle(c)
		}
		cases = append(cases, c)
	}
	write := func(n uint32, value []byte, deleted bool) contractstate.PersistentWrite {
		return contractstate.PersistentWrite{Key: keyOf(n), Value: value, Deleted: deleted}
	}
	add(trial{name: "unsorted-delete-replace-insert", version: 7, expected: 7, writes: []contractstate.PersistentWrite{write(5, []byte("new"), false), write(4, []byte{9, 0, 255}, false), write(3, nil, false), write(2, nil, true)}})
	add(trial{name: "preserve-null-empty", version: 7, expected: 7, writes: []contractstate.PersistentWrite{write(5, []byte{}, false)}})
	add(trial{name: "replace-null-with-empty", writes: []contractstate.PersistentWrite{write(1, []byte{}, false), write(2, nil, false)}})
	add(trial{name: "delete-absent-increments", writes: []contractstate.PersistentWrite{write(100, nil, true)}})
	add(trial{name: "same-value-increments", writes: []contractstate.PersistentWrite{write(4, []byte("old"), false)}})
	add(trial{name: "empty-noop"})
	add(trial{name: "empty-noop-max", version: math.MaxUint64, expected: math.MaxUint64})
	add(trial{name: "last-version", version: math.MaxUint64 - 1, expected: math.MaxUint64 - 1, writes: []contractstate.PersistentWrite{write(5, nil, false)}})
	add(trial{name: "version-exhausted", version: math.MaxUint64, expected: math.MaxUint64, writes: []contractstate.PersistentWrite{write(100, nil, true)}, code: 1006})
	add(trial{name: "wrong-layout", wrong: 1, expected: 99, code: 1004})
	add(trial{name: "stale-version", expected: 1, code: 1005})
	add(trial{name: "duplicate", writes: []contractstate.PersistentWrite{write(5, nil, false), write(1, nil, true), write(5, nil, true)}, code: 1001})
	add(trial{name: "deleted-value", writes: []contractstate.PersistentWrite{write(5, []byte("bad"), true)}, code: 1001})
	add(trial{name: "span-reversed", mutations: []stateLocalOverlayMutation{{key: keyOf(5), start: 1, end: 0}}, code: 1001})
	add(trial{name: "span-outside", mutations: []stateLocalOverlayMutation{{key: keyOf(5), end: 1}}, code: 1001})
	add(trial{name: "null-with-data", data: []byte{1}, mutations: []stateLocalOverlayMutation{{key: keyOf(5), end: 1, null: true}}, code: 1001})
	add(trial{name: "value-limit", writes: []contractstate.PersistentWrite{write(5, []byte("four"), false)}, limits: [3]uint64{16 << 20, 65536, 3}, code: 1002})
	add(trial{name: "mutation-count", writes: []contractstate.PersistentWrite{write(5, nil, false), write(7, nil, false)}, limits: [3]uint64{16 << 20, 1, 65536}, code: 1002})
	add(trial{name: "mutation-total", writes: []contractstate.PersistentWrite{write(5, bytes.Repeat([]byte{1}, 600), false), write(7, bytes.Repeat([]byte{2}, 600), false)}, limits: [3]uint64{1000, 65536, 600}, code: 1002})
	add(trial{name: "final-count", writes: []contractstate.PersistentWrite{write(5, nil, false)}, limits: [3]uint64{16 << 20, 4, 65536}, code: 1002})
	add(trial{name: "untouched-value-limit", limits: [3]uint64{16 << 20, 65536, 4}, code: 1002})
	encodedSize := uint64(len(stateLocalCodecRecord(t, schemas[0], 0, base)))
	add(trial{name: "encoded-exact", limits: [3]uint64{encodedSize, 65536, encodedSize}})
	add(trial{name: "encoded-one-short", limits: [3]uint64{encodedSize - 1, 65536, encodedSize - 1}, code: 1002})
	add(trial{name: "encoded-expansion", writes: []contractstate.PersistentWrite{write(5, bytes.Repeat([]byte{7}, 32), false)}, limits: [3]uint64{encodedSize + 32, 65536, 128}, code: 1002})
	add(trial{name: "append-only", mode: 1, next: 1, version: 7, expected: 7})
	add(trial{name: "same-layout", mode: 1, version: 7, expected: 7})
	add(trial{name: "same-layout-max", mode: 1, version: math.MaxUint64, expected: math.MaxUint64})
	add(trial{name: "append-empty-fields-null", mode: 1, old: 9, next: 0, version: 7, expected: 7})
	add(trial{name: "append-version-exhausted", mode: 1, next: 1, version: math.MaxUint64, expected: math.MaxUint64, code: 1006})
	for i, name := range []string{"rename", "reorder", "kind", "value-codec", "key-codec", "remove", "uid"} {
		add(trial{name: "incompatible-" + name, mode: 1, next: uint64(i + 2), code: 1007})
	}
	add(trial{name: "incompatible-null-fields", mode: 1, next: 9, code: 1007})
	add(trial{name: "invalid-next", mode: 1, next: 10, code: 1001})
	add(trial{name: "invalid-selected", mode: 1, selected: 1, code: 1001})
	add(trial{name: "upgrade-wrong-layout", mode: 1, next: 1, wrong: 1, code: 1004})
	add(trial{name: "upgrade-stale", mode: 1, next: 1, expected: 1, code: 1005})
	for i, limits := range [][3]uint64{{0, 1, 1}, {(16 << 20) + 1, 1, 1}, {1, 0, 1}, {65536, 65537, 1}, {65536, 1, 0}, {65536, 1, 65537}} {
		add(trial{name: fmt.Sprintf("invalid-policy-%d", i), limits: limits, code: 1001})
	}
	// Reverse order at the maximum admitted cardinality exercises bounded sorting.
	many := trial{name: "unsorted-65536", rows: []contractstate.Entry{}}
	for i := 65536; i > 0; i-- {
		many.writes = append(many.writes, write(uint32(i), nil, false))
	}
	add(many)
	tooMany := many
	tooMany.name = "count-65537"
	tooMany.writes = append(append([]contractstate.PersistentWrite{}, many.writes...), write(65537, nil, false))
	tooMany.code = 1002
	tooMany.want = nil
	add(tooMany)
	// An independent reference adapter also agrees with the disk merge result.
	reference, err := contractstate.NewReferencePersistence(schemas[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	seed := make([]contractstate.PersistentWrite, len(base))
	for i, e := range base {
		seed[i] = contractstate.PersistentWrite{Key: e.Key, Value: e.Value}
	}
	if _, err = reference.CompareAndPublish(schemas[0].Hash(), 0, seed); err != nil {
		t.Fatal(err)
	}
	if _, err = reference.CompareAndPublish(schemas[0].Hash(), 1, cases[0].writes); err != nil {
		t.Fatal(err)
	}
	snapshot, err := reference.Snapshot(schemas[0].Hash())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stateLocalCodecRecord(t, schemas[0], 8, snapshot.Entries), cases[0].want) {
		t.Fatal("reference/disk merge disagreement")
	}
	candidateAtlasProbeSize(t, "module_state_local_codec", stateLocalOverlayProbe+plans.String(), 128<<10, func(t *testing.T, image string) {
		run := func(c trial, structure, byteWork, wantCode uint64) (uint64, uint64) {
			t.Helper()
			record := stateLocalCodecRecord(t, schemas[c.old], c.version, c.rows)
			rows, data := c.mutations, append([]byte{}, c.data...)
			if rows == nil {
				for _, w := range c.writes {
					start := uint64(len(data))
					data = append(data, w.Value...)
					rows = append(rows, stateLocalOverlayMutation{key: w.Key, start: start, end: uint64(len(data)), deleted: w.Deleted, null: w.Value == nil})
				}
			}
			var packet []byte
			for _, n := range []uint64{c.mode, structure, byteWork, c.limits[0], c.limits[1], c.limits[2], c.old, c.next, c.wrong, c.expected, uint64(len(record)), uint64(len(rows)), uint64(len(data)), c.selected} {
				packet = binary.BigEndian.AppendUint64(packet, n)
			}
			packet = append(packet, record...)
			for _, r := range rows {
				packet = append(packet, r.key[:]...)
				deleted, null := uint64(0), uint64(0)
				if r.deleted {
					deleted = 1
				}
				if r.null {
					null = 1
				}
				for _, n := range []uint64{r.start, r.end, deleted, null} {
					packet = binary.BigEndian.AppendUint64(packet, n)
				}
			}
			packet = append(packet, data...)
			exit, out := candidateAtlasProcess(t, image, packet)
			if exit != 0 {
				t.Fatalf("%s exit=%d output=%q", c.name, exit, out[:min(len(out), 512)])
			}
			r := bufio.NewReader(strings.NewReader(out))
			number := func() uint64 {
				t.Helper()
				line, err := r.ReadString('\n')
				n, e := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
				if err != nil || e != nil {
					t.Fatalf("number %q %v %v", line, err, e)
				}
				return n
			}
			if code := number(); code != wantCode {
				t.Fatalf("%s code=%d want=%d", c.name, code, wantCode)
			}
			s, b := number(), number()
			if wantCode == 0 {
				n := number()
				got := make([]byte, n)
				if _, err := io.ReadFull(r, got); err != nil {
					t.Fatal(err)
				}
				if end, err := r.ReadByte(); err != nil || end != '\n' {
					t.Fatal("blob terminator", err)
				}
				if !bytes.Equal(got, c.want) {
					t.Fatalf("%s encoded mismatch lengths=%d/%d", c.name, len(got), len(c.want))
				}
			}
			if rest, _ := io.ReadAll(r); len(rest) != 0 {
				t.Fatal("trailing output")
			}
			return s, b
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) { run(c, 0, 0, c.code) })
		}
		for _, index := range []int{0, 5, 24, 27} {
			c := cases[index]
			t.Run("exact-work-"+c.name, func(t *testing.T) {
				s, b := run(c, 0, 0, 0)
				run(c, (16<<20)-s, (1<<30)-b, 0)
				run(c, (16<<20)+1-s, 0, 1002)
				run(c, 0, (1<<30)+1-b, 1002)
			})
		}
		c := cases[0]
		t.Run("structure-overdrawn", func(t *testing.T) { run(c, (16<<20)+1, 0, 1002) })
		t.Run("bytes-overdrawn", func(t *testing.T) { run(c, 0, (1<<30)+1, 1002) })
	})
}

package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
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
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// The Plan bytes come independently from the existing Go layout builder.
func stateLocalCodecPlan(schema contractstate.Layout) string {
	words := func(raw []byte) string {
		parts := make([]string, 8)
		for i := range parts {
			parts[i] = fmt.Sprint(binary.BigEndian.Uint32(raw[i*4:]))
		}
		return "[" + strings.Join(parts, ",") + "]"
	}
	codecs := map[string]int{"": 0, "bool/v1": 1, "u32/be/v1": 2, "u64/be/v1": 3, "bytes20/v1": 4, "bytes32/v1": 5, "bytes/v1": 6, "utf8/v1": 7}
	names := schema.ContractUID()
	var fields strings.Builder
	for _, field := range schema.Fields() {
		start := len(names)
		names += field.Name
		fmt.Fprintf(&fields, "assert(v::push<layout::Field>(ref mut fields,layout::Field{.kind=%d,.key=%d,.value=%d,.name_start=%d,.name_end=%d,.start=0,.end=1,.prefix=%s}).is_ok);", field.Kind, codecs[field.KeyTypeID], codecs[field.ValueTypeID], start, len(names), words(field.Prefix[:]))
	}
	hash := schema.Hash()
	return fmt.Sprintf(`fn layout::Plan schemas(){var fields=v::empty<layout::Field>();%s var rows=v::empty<layout::Layout>();assert(v::push<layout::Layout>(ref mut rows,layout::Layout{.owner=0,.module_index=0,.declaration=1,.start=0,.end=1,.uid_start=0,.uid_end=%d,.field_start=0,.field_count=%d,.hash=%s}).is_ok);return layout::assemble(owned(b%q),rows,fields);}`, &fields, len(schema.ContractUID()), len(schema.Fields()), words(hash[:]), names)
}

const stateLocalCodecProbe = `module main;
import module_state_local_codec as local;import module_state_layout as layout;import module_state_graph_request as request;import project_identity_bytes as identities;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn bytes::Buffer owned(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(console)fn void number(usize n){assert(io::write_usize(n).is_ok && io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok && io::write("\n").is_ok);}
fn usize n64(ref u8[] raw,usize at){var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[at+i] as usize);i+=1;}return n;}
@effects(console)fn i32 main(){var incoming=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref incoming)>16777264-n || !bytes::append(ref mut incoming,b[..n]).is_ok){return 2;}}
let raw=bytes::view(ref incoming);if(raw.len<48){return 3;}let mode=n64(raw,0);var work=local::Work{.structure=n64(raw,8),.bytes=n64(raw,16)};let limits=local::Limits{.bytes=n64(raw,24),.entries=n64(raw,32),.value=n64(raw,40)};let schemas=schemas();
var state=switch(if(mode==1){local::initial(ref schemas,0,limits,ref mut work)}else{local::decode(ref schemas,0,raw[48..],limits,ref mut work)}){case .ok(p):{p}case .err(code):{number(code as usize);number(work.structure);number(work.bytes);return 0;}};
let cost=work;if(mode==2){var exact=local::Work{.structure=16777216-cost.structure,.bytes=1073741824-cost.bytes};assert(local::decode(ref schemas,0,raw[48..],limits,ref mut exact).is_ok && exact.structure==16777216 && exact.bytes==1073741824);
var short=local::Work{.structure=16777217-cost.structure,.bytes=0};switch(local::decode(ref schemas,0,raw[48..],limits,ref mut short)){case .ok(_):{return 4;}case .err(code):{assert(code==1002);}};short=local::Work{.structure=0,.bytes=1073741825-cost.bytes};switch(local::decode(ref schemas,0,raw[48..],limits,ref mut short)){case .ok(_):{return 5;}case .err(code):{assert(code==1002);}};}
var writing=local::empty_work();let encoded=switch(local::encode(ref state,limits,ref mut writing)){case .ok(p):{p}case .err(code):{number(code as usize);number(writing.structure);number(writing.bytes);return 0;}};
var detached=local::empty_work();let snapshot=switch(local::snapshot(ref state,ref mut detached)){case .ok(p):{p}case .err(_):{return 6;}};
number(0);number(work.structure);number(work.bytes);blob(bytes::view(ref encoded));number(local::version(ref state));number(local::count(ref state));var usize i=0;while(i<local::count(ref state)){number(if(local::is_null(ref state,i)){1}else{0});i+=1;}
// Both the original wire and decoded State are destroyed before snapshot reads.
state=switch(local::initial(ref schemas,0,limits,ref mut detached)){case .ok(p):{p}case .err(_):{return 7;}};incoming=bytes::empty();number(request::snapshot_version(ref snapshot));number(request::snapshot_count(ref snapshot));var usize meter=0;let hash=switch(identities::words(request::snapshot_layout(ref snapshot),ref mut meter)){case .ok(p):{p}case .err(_):{return 8;}};blob(bytes::view(ref hash));i=0;while(i<request::snapshot_count(ref snapshot)){meter=0;let key=switch(identities::words(request::snapshot_key(ref snapshot,i),ref mut meter)){case .ok(p):{p}case .err(_):{return 9;}};blob(bytes::view(ref key));blob(request::snapshot_value(ref snapshot,i));i+=1;}return 0;}
`

func stateLocalCodecRecord(t *testing.T, schema contractstate.Layout, version uint64, entries []contractstate.Entry) []byte {
	t.Helper()
	if entries == nil {
		entries = []contractstate.Entry{}
	}
	body, err := json.Marshal(struct {
		Schema      string                   `json:"schema"`
		ContractUID string                   `json:"contractUid"`
		LayoutHash  contractstate.LayoutHash `json:"layoutHash"`
		Fields      []contractstate.Field    `json:"fields"`
		Version     uint64                   `json:"version"`
		Entries     []contractstate.Entry    `json:"entries"`
	}{"fairyc.local-state/v1", schema.ContractUID(), schema.Hash(), schema.Fields(), version, entries})
	if err != nil {
		t.Fatal(err)
	}
	return stateLocalCodecChecksum(body)
}

func stateLocalCodecChecksum(body []byte) []byte {
	digest := sha256.Sum256(body)
	return append(append([]byte{}, body...), digest[:]...)
}

func TestCandidateStateLocalCodec(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Fairy local codec native execution; Linux remains batched")
	}
	declaration := stateGraphIRBase(t).Layout
	builder, err := contractstate.NewLayoutBuilder(declaration.ContractUID)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range declaration.Fields {
		if field.Kind == contractstate.FieldCell {
			_, err = contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
		} else {
			_, err = contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	schema, err := builder.Build()
	if err != nil || schema.Hash() != declaration.Hash {
		t.Fatal("independent layout", err)
	}
	limits := [3]uint64{16 << 20, 65536, 65536}
	// Actual Go initialization is checked separately from the independent encoder.
	directory := t.TempDir()
	actualStore, err := contractstate.CreateLocalPersistence(directory, schema, contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = actualStore.Close() })
	actualEmpty, err := os.ReadFile(filepath.Join(directory, "state-v1.bin"))
	if err != nil || !bytes.Equal(actualEmpty, stateLocalCodecRecord(t, schema, 0, nil)) {
		t.Fatal("Go empty record mismatch", err)
	}
	var rows []contractstate.Entry
	for i, value := range [][]byte{nil, {}, {0, 255, 13, 10, 26}, bytes.Repeat([]byte{0xa5}, 65536)} {
		var key contractstate.Key
		key[31] = byte(i + 1)
		rows = append(rows, contractstate.Entry{Key: key, Value: value})
	}
	writes := make([]contractstate.PersistentWrite, len(rows))
	for i, e := range rows {
		writes[i] = contractstate.PersistentWrite{Key: e.Key, Value: e.Value}
	}
	if _, err = actualStore.CompareAndPublish(schema.Hash(), 0, writes); err != nil {
		t.Fatal(err)
	}
	actualPopulated, err := os.ReadFile(filepath.Join(directory, "state-v1.bin"))
	if err != nil || !bytes.Equal(actualPopulated, stateLocalCodecRecord(t, schema, 1, rows)) {
		t.Fatal("Go populated record mismatch", err)
	}
	candidateAtlasProbeSize(t, "module_state_local_codec", stateLocalCodecProbe+stateLocalCodecPlan(schema), 128<<10, func(t *testing.T, image string) {
		run := func(mode, structure, byteWork uint64, policy [3]uint64, record []byte, wantCode uint64, version uint64, rows []contractstate.Entry) {
			t.Helper()
			var packet []byte
			for _, n := range []uint64{mode, structure, byteWork, policy[0], policy[1], policy[2]} {
				packet = binary.BigEndian.AppendUint64(packet, n)
			}
			exit, output := candidateAtlasProcess(t, image, append(packet, record...))
			if exit != 0 {
				t.Fatalf("codec exit=%d output=%q", exit, output[:min(len(output), 256)])
			}
			r := bufio.NewReader(strings.NewReader(output))
			number := func() uint64 {
				t.Helper()
				line, err := r.ReadString('\n')
				value, parse := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
				if err != nil || parse != nil {
					t.Fatalf("codec number %q: %v %v", line, err, parse)
				}
				return value
			}
			blob := func() []byte {
				t.Helper()
				n := number()
				data := make([]byte, n)
				if _, err := io.ReadFull(r, data); err != nil {
					t.Fatal(err)
				}
				if b, err := r.ReadByte(); err != nil || b != '\n' {
					t.Fatal("blob terminator", err)
				}
				return data
			}
			if code := number(); code != wantCode {
				t.Fatalf("codec code=%d want=%d", code, wantCode)
			}
			_ = number()
			_ = number()
			if wantCode != 0 {
				return
			}
			if got := blob(); !bytes.Equal(got, record) {
				t.Fatal("encoded record differs from actual Go state bytes", len(got), len(record))
			}
			if number() != version || number() != uint64(len(rows)) {
				t.Fatal("decoded state header")
			}
			for _, row := range rows {
				if (number() != 0) != (row.Value == nil) {
					t.Fatal("null/empty value distinction lost")
				}
			}
			if number() != version || number() != uint64(len(rows)) {
				t.Fatal("detached snapshot header")
			}
			hash := schema.Hash()
			if !bytes.Equal(blob(), hash[:]) {
				t.Fatal("detached snapshot layout")
			}
			for _, row := range rows {
				if !bytes.Equal(blob(), row.Key[:]) || !bytes.Equal(blob(), row.Value) {
					t.Fatal("detached snapshot bytes")
				}
			}
			if rest, _ := io.ReadAll(r); len(rest) != 0 {
				t.Fatal("trailing codec result")
			}
		}
		t.Run("actual-empty-initial", func(t *testing.T) { run(1, 0, 0, limits, actualEmpty, 0, 0, nil) })
		t.Run("actual-populated", func(t *testing.T) { run(2, 0, 0, limits, actualPopulated, 0, 1, rows) })
		for _, version := range []uint64{0, math.MaxUint64 - 1, math.MaxUint64} {
			t.Run(fmt.Sprintf("version-%d", version), func(t *testing.T) {
				run(0, 0, 0, limits, stateLocalCodecRecord(t, schema, version, rows[:3]), 0, version, rows[:3])
			})
		}
		for _, size := range []uint64{uint64(len(actualPopulated)), uint64(len(actualPopulated) - 1)} {
			t.Run(fmt.Sprintf("file-bound-%d", size), func(t *testing.T) {
				code := uint64(0)
				if size < uint64(len(actualPopulated)) {
					code = 1002
				}
				run(0, 0, 0, [3]uint64{size, 65536, 65536}, actualPopulated, code, 1, rows)
			})
		}
		for _, trial := range []struct {
			name string
			data []byte
		}{
			{"checksum", append(append([]byte{}, actualEmpty[:len(actualEmpty)-1]...), actualEmpty[len(actualEmpty)-1]^1)},
			{"short", []byte("short")},
			{"whitespace", stateLocalCodecChecksum(append([]byte(" "), actualEmpty[:len(actualEmpty)-32]...))},
			{"unknown", stateLocalCodecChecksum(bytes.Replace(actualEmpty[:len(actualEmpty)-32], []byte(`"version":`), []byte(`"unknown":0,"version":`), 1))},
			{"leading-zero", stateLocalCodecChecksum(bytes.Replace(actualEmpty[:len(actualEmpty)-32], []byte(`"version":0`), []byte(`"version":00`), 1))},
			{"overflow", stateLocalCodecChecksum(bytes.Replace(actualEmpty[:len(actualEmpty)-32], []byte(`"version":0`), []byte(`"version":18446744073709551616`), 1))},
			{"layout", stateLocalCodecChecksum(bytes.Replace(actualEmpty[:len(actualEmpty)-32], []byte(`fixture.graph-ir`), []byte(`fixture.graph-xx`), 1))},
			{"duplicate-key", stateLocalCodecRecord(t, schema, 1, []contractstate.Entry{rows[0], rows[0]})},
			{"unordered", stateLocalCodecRecord(t, schema, 1, []contractstate.Entry{rows[1], rows[0]})},
		} {
			t.Run(trial.name, func(t *testing.T) {
				code := uint64(1001)
				if trial.name == "short" {
					code = 1002
				}
				run(0, 0, 0, limits, trial.data, code, 0, nil)
			})
		}
		t.Run("value-one-short", func(t *testing.T) { run(0, 0, 0, [3]uint64{16 << 20, 65536, 65535}, actualPopulated, 1002, 0, nil) })
		t.Run("entries-one-short", func(t *testing.T) { run(0, 0, 0, [3]uint64{16 << 20, 3, 65536}, actualPopulated, 1002, 0, nil) })
		for i, policy := range [][3]uint64{{0, 1, 1}, {(16 << 20) + 1, 1, 1}, {1, 0, 1}, {65536, 65537, 1}, {65536, 1, 0}, {65536, 1, 65537}} {
			t.Run(fmt.Sprintf("invalid-policy-%d", i), func(t *testing.T) { run(0, 0, 0, policy, actualEmpty, 1001, 0, nil) })
		}
		t.Run("structure-overdrawn", func(t *testing.T) { run(0, 16777217, 0, limits, actualEmpty, 1002, 0, nil) })
		t.Run("bytes-overdrawn", func(t *testing.T) { run(0, 0, 1073741825, limits, actualEmpty, 1002, 0, nil) })
	})
}

func TestCandidateStateLocalCodecEmptyLayout(t *testing.T) {
	builder, err := contractstate.NewLayoutBuilder("fixture.empty-local")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := stateLocalCodecRecord(t, schema, 0, nil)
	// Go encodes a zero-field layout as null, but empty entries as [].
	if !bytes.Contains(want, []byte(`"fields":null`)) || !bytes.Contains(want, []byte(`"entries":[]`)) {
		t.Fatal("Go empty canonical shape")
	}
	candidateAtlasProbeSize(t, "module_state_local_codec", stateLocalCodecProbe+stateLocalCodecPlan(schema), 128<<10, func(t *testing.T, image string) {
		var packet []byte
		for _, n := range []uint64{0, 0, 0, 16 << 20, 65536, 65536} {
			packet = binary.BigEndian.AppendUint64(packet, n)
		}
		exit, out := candidateAtlasProcess(t, image, append(packet, want...))
		if exit != 0 || !strings.HasPrefix(out, "0\n") || !bytes.Contains([]byte(out), want) {
			t.Fatalf("empty layout exit=%d output=%q", exit, out)
		}
	})
}

func TestCandidateStateLocalCodecSealedOwner(t *testing.T) {
	for _, body := range []string{
		`fn local::State forge(){return local::State{.prefix=bytes::empty(),.identity=0,.data=bytes::empty(),.entries=v::empty<usize>()};}`,
		`fn void forge(ref mut local::State state){state.data=bytes::empty();}`,
	} {
		root := t.TempDir()
		copyLexerExample(t, root)
		copyCandidateParityModules(t, root, "module_state_local_codec")
		writeTestSource(t, root, "src/main.fy", `module main;import module_state_local_codec as local;import std::bytes as bytes;import std::vec as v;`+body+`fn i32 main(){return 0;}`)
		workspace, err := workspaceloader.LoadVectorCandidate(root)
		if err != nil {
			t.Fatal(err)
		}
		var diagnostic bytes.Buffer
		_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed local state", &diagnostic)
		if code == ExitSuccess || !strings.Contains(diagnostic.String(), "E0103") {
			t.Fatalf("sealed state code=%d: %s", code, &diagnostic)
		}
	}
}

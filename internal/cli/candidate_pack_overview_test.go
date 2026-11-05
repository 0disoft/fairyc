package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/contextpack"
)

// Isolated current-map encoder probe; private rejection exit 7 is independent
// of the later actual source-backed CLI's 2/3/4/7 classification.
const candidatePackOverviewMain = `module main;import project_pack as pack;import project_atlas_current as current;import project_json_tree as json;import project_atlas_wire as wire;` + atlasReadInput + `
fn Result<bytes::Buffer,u32> line(ref u8[] raw,ref mut usize cursor){let start=*cursor;while(*cursor<raw.len && raw[*cursor]!=10){*cursor+=1;}if(*cursor==raw.len){return .err(9);}let result=switch(bytes::copy(raw[start..*cursor])){case .ok(p):{p}case .err(_):{return .err(2);}};*cursor+=1;return .ok(result);}
fn Result<bytes::Buffer,u32> query(ref u8[] raw){var usize cursor=0;var usize meter=0;let q=switch(json::parse(line(raw,ref mut cursor)?,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let data=line(raw,ref mut cursor)?;let expected=line(raw,ref mut cursor)?;if(cursor!=raw.len){return .err(9);}let version=wire::integer(ref q,0,b"read");
 let ready=if(version==2){current::read_v2(data,expected,ref mut meter)?}else if(version==3){current::read_generics(data,expected,ref mut meter)?}else{current::read(data,expected,ref mut meter)?};
 let mode=wire::string(ref q,0,b"mode");if(bytes::equal(mode,b"meter")){meter=wire::integer(ref q,0,b"meter");}let before=meter;
 let selected_version=wire::integer(ref q,0,b"version");let u32 selected=if(selected_version==1){1}else if(selected_version==2){2}else if(selected_version==3){3}else{0};
 let out=pack::encode_overview(ref ready,selected,wire::integer(ref q,0,b"records"),wire::integer(ref q,0,b"selection"),wire::integer(ref q,0,b"output"),ref mut meter)?;
 if(bytes::equal(mode,b"measure")){var result=bytes::empty();if(!bytes::append_usize(ref mut result,meter-before).is_ok){return .err(2);}return .ok(result);}return .ok(out);
}
@effects(console) fn i32 main(){let data=input();return switch(query(bytes::view(ref data))){case .ok(p):{if(io::write_bytes(bytes::view(ref p)).is_ok){0}else{4}}case .err(_):{7}};}`

func candidatePackOverviewOracle(t *testing.T, f candidateCatalogFixture, records int, selection, output uint64) ([]byte, error) {
	t.Helper()
	data, err := []func(*atlasmap.QueryIndex, atlasmap.QueryBudget, uint64) ([]byte, error){contextpack.EncodeOverview, contextpack.EncodeOverviewV2, contextpack.EncodeOverviewV3}[f.version-1](f.index, atlasmap.QueryBudget{MaxRecords: records, MaxSelectionBytes: selection}, output)
	if err != nil {
		return nil, err
	}
	candidatePackChecksum(t, data, f.version)
	return data, nil
}

func candidatePackChecksum(t *testing.T, data []byte, version int) {
	t.Helper()
	var e struct {
		Version  string          `json:"pack_version"`
		Encoding string          `json:"encoding"`
		Checksum string          `json:"checksum"`
		Pack     json.RawMessage `json:"pack"`
	}
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	h.Write([]byte("fairyc.context-pack-checksum"))
	h.Write([]byte{0, byte(version)})
	for _, value := range [][]byte{[]byte(e.Version), []byte(e.Encoding), e.Pack} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		h.Write(length[:])
		h.Write(value)
	}
	if want := fmt.Sprintf("fyctx%d-sha256-", version) + hex.EncodeToString(h.Sum(nil)); e.Checksum != want {
		t.Fatal("independent Context Pack checksum differs")
	}
}

func TestCandidatePackOverview(t *testing.T) {
	fixtures := candidateCatalogFixtures(t)
	candidateAtlasProbe(t, "project_pack", candidatePackOverviewMain, func(t *testing.T, image string) {
		processes := 0
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				type controls struct {
					Read      int    `json:"read"`
					Version   int    `json:"version"`
					Records   int    `json:"records"`
					Selection uint64 `json:"selection"`
					Output    uint64 `json:"output"`
					Mode      string `json:"mode"`
					Meter     int    `json:"meter"`
				}
				base := controls{Read: f.version, Version: f.version, Records: 65536, Selection: 1 << 20, Output: 1 << 20}
				run := func(q controls) (int, string) {
					t.Helper()
					processes++
					packet, err := json.Marshal(q)
					if err != nil {
						t.Fatal(err)
					}
					packet = append(packet, '\n')
					data, expected := f.data, f.expected
					if q.Mode == "corrupt" {
						data = slices.Clone(data)
						data[20] ^= 1
					}
					if q.Mode == "stale" {
						expected = bytes.Replace(expected, []byte(`"compiler_build":"`), []byte(`"compiler_build":"stale-`), 1)
					}
					packet = append(packet, data...)
					packet = append(packet, '\n')
					packet = append(packet, expected...)
					packet = append(packet, '\n')
					return candidateAtlasProcess(t, image, packet)
				}
				check := func(name string, q controls) {
					t.Helper()
					t.Run(name, func(t *testing.T) {
						want, err := candidatePackOverviewOracle(t, f, q.Records, q.Selection, q.Output)
						code, got := run(q)
						if err != nil && (code != 7 || got != "") || err == nil && (code != 0 || got != string(want)) {
							t.Fatalf("pack overview %+v: code=%d err=%v\n got=%s\nwant=%s", q, code, err, got, want)
						}
					})
				}
				full, err := f.index.Overview(atlasmap.QueryBudget{MaxRecords: 65536, MaxSelectionBytes: 1 << 20})
				if err != nil || len(full.Omissions) != 0 || len(full.Completeness) < 12 || len(full.Modules) == 0 {
					t.Fatalf("incomplete independent overview: %+v %v", full, err)
				}
				check("all", base)
				// Pin every cumulative record boundary and the priority of record vs
				// byte refusal, including the package's mandatory indivisible record.
				packageJSON, _ := json.Marshal(full.Package)
				prefix := uint64(len(packageJSON) + 1)
				boundaries := []uint64{1, prefix - 1, prefix, prefix + 1}
				for _, c := range full.Completeness {
					row, _ := json.Marshal(c)
					prefix += uint64(len(row) + 1)
					boundaries = append(boundaries, prefix-1, prefix, prefix+1)
				}
				for _, m := range full.Modules {
					row, _ := json.Marshal(m)
					prefix += uint64(len(row) + 1)
					boundaries = append(boundaries, prefix-1, prefix, prefix+1)
				}
				if prefix != full.Usage.SelectionBytes {
					t.Fatal("independent per-record selection cost differs")
				}
				slices.Sort(boundaries)
				for _, n := range slices.Compact(boundaries) {
					q := base
					q.Selection = n
					check(fmt.Sprintf("selection-%d", n), q)
				}
				for n := 0; n <= full.Usage.Records+1; n++ {
					q := base
					q.Records = n
					check(fmt.Sprintf("records-%d", n), q)
				}
				for _, n := range []int{1, full.Usage.Records - 1} {
					q := base
					q.Records, q.Selection = n, uint64(len(packageJSON)+1)
					check(fmt.Sprintf("record-byte-priority-%d", n), q)
				}
				want, err := candidatePackOverviewOracle(t, f, base.Records, base.Selection, base.Output)
				if err != nil {
					t.Fatal(err)
				}
				for _, n := range []int{0, len(want) - 1, len(want), len(want) + 1} {
					q := base
					q.Output = uint64(n)
					check(fmt.Sprintf("output-%d", n), q)
				}
				for _, mode := range []string{"corrupt", "stale", "version", "work"} {
					q := base
					q.Mode = mode
					if mode == "version" {
						q.Version = f.version%3 + 1
					} else if mode == "work" {
						q.Mode, q.Meter = "meter", 16<<20
					}
					if code, got := run(q); code != 7 || got != "" {
						t.Fatalf("atomic %s refusal: %d %q", mode, code, got)
					}
				}
				q := base
				q.Mode = "measure"
				code, measurement := run(q)
				cost, err := strconv.Atoi(measurement)
				if code != 0 || err != nil || cost <= 0 || cost >= 16<<20 {
					t.Fatalf("invalid measured work: %d %q %v", code, measurement, err)
				}
				for _, extra := range []int{-1, 0, 1} {
					q.Mode, q.Meter = "meter", (16<<20)-cost-extra
					code, got := run(q)
					if extra < 0 && (code != 7 || got != "") || extra >= 0 && (code != 0 || got != string(want)) {
						t.Fatalf("encoder work boundary %d: %d %q", extra, code, got)
					}
				}
			})
		}
		t.Logf("%d Context Pack overview native observations", processes)
	})
}

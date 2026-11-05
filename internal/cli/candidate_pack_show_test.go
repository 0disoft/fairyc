package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/contextpack"
)

const candidatePackShowMain = `module main;import project_pack as pack;import project_pack_query as query;import project_atlas_current as current;import project_json_tree as json;import project_atlas_wire as wire;` + atlasReadInput + `
fn Result<bytes::Buffer,u32> line(ref u8[] raw,ref mut usize cursor){let start=*cursor;while(*cursor<raw.len && raw[*cursor]!=10){*cursor+=1;}if(*cursor==raw.len){return .err(9);}let result=switch(bytes::copy(raw[start..*cursor])){case .ok(p):{p}case .err(_):{return .err(2);}};*cursor+=1;return .ok(result);}
fn Result<bytes::Buffer,u32> execute(ref u8[] raw){var usize cursor=0;var usize meter=0;let q=switch(json::parse(line(raw,ref mut cursor)?,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let data=line(raw,ref mut cursor)?;let expected=line(raw,ref mut cursor)?;if(cursor!=raw.len){return .err(9);}let read=wire::integer(ref q,0,b"read");
 let ready=if(read==2){current::read_v2(data,expected,ref mut meter)?}else if(read==3){current::read_generics(data,expected,ref mut meter)?}else{current::read(data,expected,ref mut meter)?};
 let mode=wire::string(ref q,0,b"mode");if(bytes::equal(mode,b"meter")){meter=wire::integer(ref q,0,b"meter");}let before=meter;
 let version=wire::integer(ref q,0,b"version");let u32 selected=if(version==1){1}else if(version==2){2}else if(version==3){3}else{0};
 let direction=wire::integer(ref q,0,b"direction");let u32 selected_direction=if(direction==0){0}else if(direction==1){1}else if(direction==2){2}else{3};
 let policy=query::Policy{.direction=selected_direction,.mask=wire::integer(ref q,0,b"mask"),.records=wire::integer(ref q,0,b"records"),.relations=wire::integer(ref q,0,b"relations"),.depth=wire::integer(ref q,0,b"depth"),.selection=wire::integer(ref q,0,b"selection")};
 let out=pack::encode_show(ref ready,selected,wire::string(ref q,0,b"symbol"),policy,wire::integer(ref q,0,b"output"),ref mut meter)?;
 if(bytes::equal(mode,b"measure")){var result=bytes::empty();if(!bytes::append_usize(ref mut result,meter-before).is_ok){return .err(2);}return .ok(result);}return .ok(out);
}
@effects(console) fn i32 main(){let data=input();return switch(execute(bytes::view(ref data))){case .ok(p):{if(io::write_bytes(bytes::view(ref p)).is_ok){0}else{4}}case .err(_):{7}};}`

type candidatePackQueryControls struct {
	Read      int    `json:"read"`
	Version   int    `json:"version"`
	Symbol    string `json:"symbol"`
	Direction int    `json:"direction"`
	Mask      int    `json:"mask"`
	Records   int    `json:"records"`
	Relations int    `json:"relations"`
	Depth     int    `json:"depth"`
	Selection uint64 `json:"selection"`
	Output    uint64 `json:"output"`
	Mode      string `json:"mode"`
	Meter     int    `json:"meter"`
}

func (q candidatePackQueryControls) request() atlasmap.RelationRequest {
	direction := atlasmap.QueryDirection("invalid")
	if q.Direction >= 0 && q.Direction < 3 {
		direction = []atlasmap.QueryDirection{atlasmap.DirectionOutgoing, atlasmap.DirectionIncoming, atlasmap.DirectionBoth}[q.Direction]
	}
	var kinds []atlasmap.RelationKind
	for i, k := range atlasmap.AllRelationKinds() {
		if q.Mask&(1<<i) != 0 {
			kinds = append(kinds, k)
		}
	}
	if q.Mask&^1023 != 0 {
		kinds = append(kinds, "invalid")
	}
	return atlasmap.RelationRequest{Direction: direction, Relations: kinds, Budget: atlasmap.QueryBudget{MaxRecords: q.Records, MaxRelations: q.Relations, MaxDepth: q.Depth, MaxSelectionBytes: q.Selection}}
}

func candidatePackQueryOracle(t *testing.T, f candidateCatalogFixture, q candidatePackQueryControls, around bool) ([]byte, error) {
	t.Helper()
	if q.Version < 1 || q.Version > 3 {
		return nil, fmt.Errorf("invalid pack version")
	}
	encoders := []func(*atlasmap.QueryIndex, string, atlasmap.RelationRequest, uint64) ([]byte, error){contextpack.EncodeShow, contextpack.EncodeShowV2, contextpack.EncodeShowV3}
	if around {
		encoders = []func(*atlasmap.QueryIndex, string, atlasmap.RelationRequest, uint64) ([]byte, error){contextpack.EncodeAround, contextpack.EncodeAroundV2, contextpack.EncodeAroundV3}
	}
	data, err := encoders[q.Version-1](f.index, q.Symbol, q.request(), q.Output)
	if err == nil {
		candidatePackChecksum(t, data, q.Version)
	}
	return data, err
}

func candidatePackQueryProcess(t *testing.T, image string, f candidateCatalogFixture, q candidatePackQueryControls) (int, string) {
	t.Helper()
	control, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	data, expected := f.data, f.expected
	if q.Mode == "corrupt" {
		data = slices.Clone(data)
		data[20] ^= 1
	}
	if q.Mode == "stale" {
		expected = bytes.Replace(expected, []byte(`"compiler_build":"`), []byte(`"compiler_build":"stale-`), 1)
	}
	packet := append(control, '\n')
	packet = append(packet, data...)
	packet = append(packet, '\n')
	packet = append(packet, expected...)
	packet = append(packet, '\n')
	return candidateAtlasProcess(t, image, packet)
}

func TestCandidatePackShow(t *testing.T) {
	fixtures := candidatePackRelationFixtures(t)
	candidateAtlasProbe(t, "project_pack", candidatePackShowMain, func(t *testing.T, image string) {
		processes := 0
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				run := func(q candidatePackQueryControls) (int, string) {
					t.Helper()
					processes++
					return candidatePackQueryProcess(t, image, f, q)
				}
				check := func(name string, q candidatePackQueryControls) {
					t.Helper()
					t.Run(name, func(t *testing.T) {
						want, err := candidatePackQueryOracle(t, f, q, false)
						code, got := run(q)
						if err != nil && (code != 7 || got != "") || err == nil && (code != 0 || got != string(want)) {
							t.Fatalf("show %+v: %d %v\n got=%s\nwant=%s", q, code, err, got, want)
						}
					})
				}
				base := candidatePackQueryControls{Read: f.version, Version: f.version, Mask: 1023, Records: 65536, Relations: 65536, Selection: 4 << 20, Output: 4 << 20}
				catalog, err := f.index.CatalogSymbols(atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 65536, MaxSelectionBytes: 4 << 20}})
				if err != nil || !catalog.Complete {
					t.Fatalf("source catalog: %v", err)
				}
				largest := -1
				for _, entry := range catalog.Symbols {
					q := base
					q.Symbol = entry.Symbol.Full
					for direction := 0; direction < 3; direction++ {
						q.Direction = direction
						check(entry.Name+"-"+strconv.Itoa(direction), q)
					}
					q.Direction = 2
					shown, err := f.index.Show(q.Symbol, q.request())
					if err != nil {
						t.Fatal(err)
					}
					if len(shown.Relations) > largest {
						largest = len(shown.Relations)
						base = q
					}
					q.Symbol = entry.Symbol.Visible
					check(entry.Name+"-visible", q)
				}
				for i := 0; i < 10; i++ {
					q := base
					q.Mask = 1 << i
					check("kind-"+strconv.Itoa(i), q)
				}
				for _, mask := range []int{6, 24, 341, 682} {
					q := base
					q.Mask = mask
					check("mask-"+strconv.Itoa(mask), q)
				}
				full, err := f.index.Show(base.Symbol, base.request())
				if err != nil {
					t.Fatal(err)
				}
				mandatory, err := json.Marshal(struct {
					Context atlasmap.QueryContext `json:"context"`
					Subject atlasmap.QueryNode    `json:"subject"`
				}{full.Context, full.Subject})
				if err != nil {
					t.Fatal(err)
				}
				prefix := uint64(len(mandatory) + 1)
				bounds := []uint64{0, 1, prefix - 1, prefix, prefix + 1}
				for _, r := range full.Relations {
					row, err := json.Marshal(r)
					if err != nil {
						t.Fatal(err)
					}
					prefix += uint64(len(row) + 1)
					bounds = append(bounds, prefix-1, prefix, prefix+1)
				}
				if prefix != full.Usage.SelectionBytes {
					t.Fatal("independent show record costs differ")
				}
				slices.Sort(bounds)
				for _, n := range slices.Compact(bounds) {
					q := base
					q.Selection = n
					check("selection-"+strconv.FormatUint(n, 10), q)
				}
				for n := 0; n <= full.Usage.Records+1; n++ {
					q := base
					q.Records = n
					check("records-"+strconv.Itoa(n), q)
				}
				for n := 0; n <= full.Usage.Relations+1; n++ {
					q := base
					q.Relations = n
					check("relations-"+strconv.Itoa(n), q)
				}
				for _, n := range []int{0, 1, full.Usage.Relations} {
					q := base
					q.Records = 1
					q.Relations = n
					q.Selection = uint64(len(mandatory) + 1)
					check("precedence-"+strconv.Itoa(n), q)
				}
				want, err := candidatePackQueryOracle(t, f, base, false)
				if err != nil {
					t.Fatal(err)
				}
				for _, n := range []int{0, len(want) - 1, len(want), len(want) + 1} {
					q := base
					q.Output = uint64(n)
					check("output-"+strconv.Itoa(n), q)
				}
				for _, selector := range []string{"main", strings.ToUpper(base.Symbol), "fysym1-sha256-128-" + strings.Repeat("0", 32), "fysym1-sha256-256-" + strings.Repeat("0", 64)} {
					q := base
					q.Symbol = selector
					check("bad-selector-"+selector, q)
				}
				for _, mask := range []int{0, 1024} {
					q := base
					q.Mask = mask
					check("invalid-mask-"+strconv.Itoa(mask), q)
				}
				q := base
				q.Direction = 3
				check("invalid-direction", q)
				q = base
				q.Version = f.version%3 + 1
				check("wrong-family", q)
				for _, mode := range []string{"corrupt", "stale"} {
					q = base
					q.Mode = mode
					if code, got := run(q); code != 7 || got != "" {
						t.Fatalf("%s published: %d %q", mode, code, got)
					}
				}
				q = base
				q.Mode = "measure"
				code, text := run(q)
				cost, err := strconv.Atoi(text)
				if code != 0 || err != nil || cost <= 0 || cost >= 16<<20 {
					t.Fatalf("invalid work: %d %q %v", code, text, err)
				}
				for _, extra := range []int{-1, 0, 1} {
					q.Mode = "meter"
					q.Meter = (16 << 20) - cost - extra
					code, got := run(q)
					if extra < 0 && (code != 7 || got != "") || extra >= 0 && (code != 0 || got != string(want)) {
						t.Fatalf("work boundary %d: %d %q", extra, code, got)
					}
				}
			})
		}
		t.Logf("%d native show observations", processes)
	})
}

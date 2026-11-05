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
	"github.com/0disoft/fairyc/internal/workspacecallcatalog"
)

// This isolated encoder probe uses 7 as its private rejection sentinel. Product
// CLI exit classification belongs to the separate source-backed command tests.
const candidateCallsCatalogMain = `module main;
import project_calls_catalog as catalog;import project_calls_options as options;
import project_atlas_current as current;import project_atlas_join as join;
import project_json_tree as json;import project_atlas_wire as wire;import std::vec as v;` + atlasReadInput + `
fn Result<bytes::Buffer,u32> line(ref u8[] raw,ref mut usize cursor){let start=*cursor;while(*cursor<raw.len && raw[*cursor]!=10){*cursor+=1;}if(*cursor==raw.len){return .err(9);}let result=switch(bytes::copy(raw[start..*cursor])){case .ok(p):{p}case .err(_):{return .err(2);}};*cursor+=1;return .ok(result);}
fn Result<bytes::Buffer,u32> query(ref u8[] raw,ref options::Options request){
 var usize cursor=0;var usize meter=0;let q=switch(json::parse(line(raw,ref mut cursor)?,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let version=wire::integer(ref q,0,b"read");var v::Vec<current::Ready> packages=v::empty<current::Ready>();
 while(cursor<raw.len){let data=line(raw,ref mut cursor)?;let expected=line(raw,ref mut cursor)?;
  let ready=if(version==2){current::read_v2(data,expected,ref mut meter)?}else if(version==3){current::read_generics(data,expected,ref mut meter)?}else{current::read(data,expected,ref mut meter)?};
  if(!v::push<current::Ready>(ref mut packages,ready).is_ok){return .err(2);}}
 let graph=join::join(packages,wire::integer(ref q,0,b"packages"),wire::integer(ref q,0,b"calls"),ref mut meter)?;
 let mode=wire::string(ref q,0,b"mode");if(bytes::equal(mode,b"meter")){meter=wire::integer(ref q,0,b"meter");}let before=meter;
 let out=catalog::encode(ref graph,request,ref mut meter)?;
 if(bytes::equal(mode,b"measure")){var result=bytes::empty();if(!bytes::append_usize(ref mut result,meter-before).is_ok){return .err(2);}return .ok(result);}return .ok(out);
}
@effects(console) fn i32 main(){let request=switch(options::arguments()){case .ok(p):{p}case .err(_):{return 2;}};let data=input();return switch(query(bytes::view(ref data),ref request)){case .ok(p):{if(io::write_bytes(bytes::view(ref p)).is_ok){0}else{4}}case .err(_):{7}};}`

type candidateCallsFixture struct {
	version  int
	prepared preparedAtlasWorkspace
	maps     [][]byte
	current  [][]byte
}

func candidateCallsFixtures(t *testing.T) []candidateCallsFixture {
	t.Helper()
	ordinary := `module main;import answer as a;
pub fn i32 isolated(){return 0;}fn i32 leaf(){return a::value();}
fn i32 self(i32 n){if(n==0){return leaf();}return self(n-1)+leaf();}
fn i32 main(){return self(2)+leaf()+leaf();}`
	generic := `module main;import answer as a;
pub fn T unused<T>(T value){return value;}
fn T wrap<T>(T value){return a::id<T>(value);}
fn i32 main(){discard wrap<u8>(3);return wrap<i32>(a::value());}`
	aggregate := `module main;import answer as a;
resource Token<T>{T value;cleanup{let T observed=a::id<T>(this.value);}}
pub fn T unused<T>(T value){return value;}
fn T wrap<T>(T value){return a::id<T>(value);}
fn i32 main(){let a=Token<i32>{.value=1};let b=Token<u8>{.value=2};discard wrap<u8>(3);return wrap<i32>(a::value());}`
	var result []candidateCallsFixture
	for i, source := range []string{ordinary, generic, aggregate} {
		root := writeWorkspaceSessionFixture(t, source)
		if i > 0 {
			writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;pub fn i32 value(){return 42;}pub fn T id<T>(T value){return value;}`)
		}
		var diagnostic bytes.Buffer
		p, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3}[i])
		if code != 0 {
			t.Fatalf("call source v%d: %d %s", i, code, &diagnostic)
		}
		f := candidateCallsFixture{version: i, prepared: p}
		for _, part := range p.partitions {
			data, err := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[i](part, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			f.maps = append(f.maps, data)
			f.current = append(f.current, candidateAtlasCurrent(t, p.expectations[part.Header().PackageInstance]))
		}
		result = append(result, f)
	}
	return result
}

func candidateCallsIndex(f candidateCallsFixture, budget atlasmap.WorkspaceCallBudget) (*atlasmap.WorkspaceCallIndex, error) {
	return []func([]atlasmap.Partition, map[string]atlasmap.LoadExpectation, atlasmap.WorkspaceCallBudget) (*atlasmap.WorkspaceCallIndex, error){
		atlasmap.NewWorkspaceCallIndex, atlasmap.NewWorkspaceCallIndexV2, atlasmap.NewWorkspaceCallIndexV3,
	}[f.version](f.prepared.partitions, f.prepared.expectations, budget)
}

func candidateCallsOracle(t *testing.T, f candidateCallsFixture, budget atlasmap.WorkspaceCallBudget, instance, symbol string, direction atlasmap.QueryDirection, output uint64) ([]byte, error) {
	t.Helper()
	index, err := candidateCallsIndex(f, budget)
	if err != nil {
		return nil, err
	}
	data, err := []func(*atlasmap.WorkspaceCallIndex, string, string, atlasmap.QueryDirection, uint64) ([]byte, error){workspacecallcatalog.Encode, workspacecallcatalog.EncodeV1, workspacecallcatalog.EncodeV2}[f.version](index, instance, symbol, direction, output)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Version  string          `json:"catalog_version"`
		Encoding string          `json:"encoding"`
		Checksum string          `json:"checksum"`
		Catalog  json.RawMessage `json:"catalog"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	h.Write([]byte("fairyc.workspace-call-catalog-checksum"))
	h.Write([]byte{0, byte(f.version)})
	for _, value := range [][]byte{[]byte(envelope.Version), []byte(envelope.Encoding), envelope.Catalog} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		h.Write(length[:])
		h.Write(value)
	}
	if want := fmt.Sprintf("fywcall%d-sha256-", f.version) + hex.EncodeToString(h.Sum(nil)); envelope.Checksum != want {
		t.Fatal("independent calls checksum differs")
	}
	return data, nil
}

func TestCandidateCallsCatalog(t *testing.T) {
	fixtures := candidateCallsFixtures(t)
	candidateAtlasProbeSize(t, "project_calls_catalog", candidateCallsCatalogMain, 128<<10, func(t *testing.T, image string) {
		processes := 0
		for _, f := range fixtures {
			t.Run(fmt.Sprintf("v%d", f.version), func(t *testing.T) {
				budget := atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}
				index, err := candidateCallsIndex(f, budget)
				if err != nil {
					t.Fatal(err)
				}
				calls := index.Calls()
				cleanups, specialized, self := 0, 0, 0
				for _, c := range calls {
					if c.SourceBody == "cleanup" {
						cleanups++
					}
					if c.SourceSpecialization != "" || c.TargetSpecialization != "" {
						specialized++
					}
					if c.Source == c.Target {
						self++
					}
				}
				if f.version == 0 && self != 1 || f.version > 0 && specialized < 4 || f.version == 2 && cleanups != 2 {
					t.Fatalf("source corpus lost required calls: self=%d specializations=%d cleanup=%d", self, specialized, cleanups)
				}
				type controls struct {
					Read     int    `json:"read"`
					Packages int    `json:"packages"`
					Calls    int    `json:"calls"`
					Mode     string `json:"mode"`
					Meter    int    `json:"meter"`
				}
				control := controls{Read: f.version + 1, Packages: budget.MaxPackages, Calls: budget.MaxCalls}
				order := make([]int, len(f.maps))
				for i := range order {
					order[i] = i
				}
				run := func(uid, symbol string, direction atlasmap.QueryDirection, limits atlasmap.WorkspaceCallBudget, output uint64, q controls, order []int) (int, string) {
					t.Helper()
					processes++
					packet, err := json.Marshal(q)
					if err != nil {
						t.Fatal(err)
					}
					packet = append(packet, '\n')
					for _, i := range order {
						data, expected := f.maps[i], f.current[i]
						if q.Mode == "corrupt" && i == order[0] {
							data = slices.Clone(data)
							data[20] ^= 1
						}
						if q.Mode == "stale" && i == order[0] {
							expected = bytes.Replace(expected, []byte(`"compiler_build":"`), []byte(`"compiler_build":"stale-`), 1)
						}
						packet = append(packet, data...)
						packet = append(packet, '\n')
						packet = append(packet, expected...)
						packet = append(packet, '\n')
					}
					version := f.version
					if q.Mode == "catalog-version" {
						version = (version + 1) % 3
					}
					return candidateAtlasProcess(t, image, packet, "experimental", "atlas-calls", "list", "--catalog-version", strconv.Itoa(version), "--package-uid", uid, "--symbol", symbol, "--direction", string(direction), "--max-packages", strconv.Itoa(limits.MaxPackages), "--max-calls", strconv.Itoa(limits.MaxCalls), "--max-output-bytes", strconv.FormatUint(output, 10))
				}
				for _, p := range f.prepared.partitions {
					for _, s := range p.Symbols() {
						for _, direction := range []atlasmap.QueryDirection{atlasmap.DirectionOutgoing, atlasmap.DirectionIncoming, atlasmap.DirectionBoth} {
							want, err := candidateCallsOracle(t, f, budget, p.Header().PackageInstance, s.Visible, direction, 1<<20)
							if err != nil {
								t.Fatal(err)
							}
							for _, reverse := range []bool{false, true} {
								indices, symbol := slices.Clone(order), s.Visible
								if reverse {
									slices.Reverse(indices)
									symbol = s.Full
								}
								if code, got := run(p.Header().PackageUID, symbol, direction, budget, 1<<20, control, indices); code != 0 || got != string(want) {
									t.Fatalf("%s/%s reverse=%v: code=%d\n got=%s\nwant=%s", s.Declaration.Name, direction, reverse, code, got, want)
								}
							}
						}
					}
				}
				p := f.prepared.partitions[0]
				s := p.Symbols()[0]
				uid, instance := p.Header().PackageUID, p.Header().PackageInstance
				direction := atlasmap.DirectionBoth
				want, err := candidateCallsOracle(t, f, budget, instance, s.Visible, direction, 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				for _, output := range []int{len(want) - 1, len(want), len(want) + 1} {
					code, got := run(uid, s.Visible, direction, budget, uint64(output), control, order)
					if output < len(want) {
						if code != 7 || got != "" {
							t.Fatalf("partial byte failure: %d %q", code, got)
						}
					} else if code != 0 || got != string(want) {
						t.Fatalf("exact byte bound %d: %d %q", output, code, got)
					}
				}
				for _, limits := range []atlasmap.WorkspaceCallBudget{{MaxPackages: len(order), MaxCalls: len(calls)}, {MaxPackages: len(order) - 1, MaxCalls: len(calls)}, {MaxPackages: len(order), MaxCalls: len(calls) - 1}} {
					q := control
					q.Packages, q.Calls = limits.MaxPackages, limits.MaxCalls
					expected, err := candidateCallsOracle(t, f, limits, instance, s.Visible, direction, 1<<20)
					code, got := run(uid, s.Visible, direction, limits, 1<<20, q, order)
					if err != nil && (code != 7 || got != "") || err == nil && (code != 0 || got != string(expected)) {
						t.Fatalf("whole graph bound %+v: %d %q, oracle=%v", limits, code, got, err)
					}
				}
				for _, mode := range []string{"map-version", "catalog-version", "package-budget", "call-budget", "duplicate", "missing-dependency", "unknown-uid", "unknown-symbol", "corrupt", "stale", "work"} {
					q, indices, u, symbol := control, slices.Clone(order), uid, s.Visible
					q.Mode = mode
					switch mode {
					case "map-version":
						q.Read = (q.Read % 3) + 1
					case "package-budget":
						q.Packages++
					case "call-budget":
						q.Calls++
					case "duplicate":
						indices = append(indices, indices[0])
					case "missing-dependency":
						for i, part := range f.prepared.partitions {
							if len(part.Header().Dependencies) > 0 {
								indices = []int{i}
								break
							}
						}
					case "unknown-uid":
						u = "example.missing"
					case "unknown-symbol":
						symbol = "fysym1-sha256-128-00000000000000000000000000000000"
					case "work":
						q.Mode, q.Meter = "meter", 16<<20
					}
					if code, got := run(u, symbol, direction, budget, 1<<20, q, indices); code != 7 || got != "" {
						t.Fatalf("atomic %s rejection: %d %q", mode, code, got)
					}
				}
				q := control
				q.Mode = "measure"
				code, measured := run(uid, s.Visible, direction, budget, 1<<20, q, order)
				cost, err := strconv.Atoi(measured)
				if code != 0 || err != nil || cost <= 0 || cost > 16<<20 {
					t.Fatalf("work measurement: %d %q %v", code, measured, err)
				}
				for _, extra := range []int{-1, 0, 1} {
					q.Mode, q.Meter = "meter", (16<<20)-cost-extra
					code, got := run(uid, s.Visible, direction, budget, 1<<20, q, order)
					if extra < 0 && (code != 7 || got != "") || extra >= 0 && (code != 0 || got != string(want)) {
						t.Fatalf("work boundary extra=%d cost=%d: %d %q", extra, cost, code, got)
					}
				}
			})
		}
		t.Logf("%d independent call-catalog native observations", processes)
	}, "project_calls_options")
}

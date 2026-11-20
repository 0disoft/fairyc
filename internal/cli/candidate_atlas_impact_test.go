package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const atlasImpactMain = `module main;
import project_atlas_current as current;import project_atlas_join as join;import project_atlas_impact as impact;import project_atlas_impact_json as output;import project_json_tree as json;import project_atlas_wire as wire;import std::vec as v;` + atlasReadInput + `
fn Result<bytes::Buffer,u32> line(ref u8[] raw,ref mut usize cursor){let start=*cursor;while(*cursor<raw.len && raw[*cursor]!=10){*cursor+=1;}if(*cursor==raw.len){return .err(9);}let result=switch(bytes::copy(raw[start..*cursor])){case .ok(p):{p}case .err(_):{return .err(2);}};*cursor+=1;return .ok(result);}
fn Result<bytes::Buffer,u32> query(ref u8[] raw){var usize cursor=0;var usize meter=0;let q=switch(json::parse(line(raw,ref mut cursor)?,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(9);}};var v::Vec<current::Ready> packages=v::empty<current::Ready>();while(cursor<raw.len){let data=line(raw,ref mut cursor)?;let expected=line(raw,ref mut cursor)?;let ready=current::read(data,expected,ref mut meter)?;if(!v::push<current::Ready>(ref mut packages,ready).is_ok){return .err(2);}}
let graph=join::join(packages,wire::integer(ref q,0,b"packages"),wire::integer(ref q,0,b"calls"),ref mut meter)?;let result=impact::explore(ref graph,wire::string(ref q,0,b"instance"),wire::string(ref q,0,b"symbol"),wire::integer(ref q,0,b"depth"),wire::integer(ref q,0,b"symbols"),ref mut meter)?;return output::render(ref graph,ref result,wire::integer(ref q,0,b"bytes"),ref mut meter);}
@effects(console) fn i32 main(){let data=input();return switch(query(bytes::view(ref data))){case .ok(p):{discard io::write_bytes(bytes::view(ref p));0}case .err(e):{discard io::write_u32(e);7}};}`

type atlasImpactFixture struct {
	name          string
	prepared      preparedAtlasWorkspace
	data, current [][]byte
	index         *atlasmap.WorkspaceCallIndex
}

func candidateAtlasImpactFixture(t *testing.T, schema, source string) atlasImpactFixture {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, source)
	var diagnostic bytes.Buffer
	p, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, schema)
	if code != 0 {
		t.Fatalf("prepare impact: %d %s", code, &diagnostic)
	}
	makeIndex := atlasmap.NewWorkspaceCallIndex
	encode := atlasmap.Encode
	if schema == atlasmap.LogicalSchemaV3 {
		makeIndex = atlasmap.NewWorkspaceCallIndexV3
		encode = atlasmap.EncodeV3
	}
	index, err := makeIndex(p.partitions, p.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	f := atlasImpactFixture{name: schema, prepared: p, index: index}
	for _, part := range p.partitions {
		data, err := encode(part, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		f.data = append(f.data, data)
		f.current = append(f.current, candidateAtlasCurrent(t, p.expectations[part.Header().PackageInstance]))
	}
	return f
}

// Controlled coverage metadata exercises bounded vs partial independently of cutoffs.
func candidateAtlasExactCoverage(t *testing.T, f atlasImpactFixture) atlasImpactFixture {
	t.Helper()
	f.data = append([][]byte{}, f.data...)
	parts := append([]atlasmap.Partition{}, f.prepared.partitions...)
	for i, data := range f.data {
		type completeness struct {
			Relation string   `json:"relation"`
			Status   string   `json:"status"`
			Resolved int      `json:"resolved"`
			External int      `json:"external"`
			Unknown  int      `json:"unknown"`
			Reasons  []string `json:"reasons"`
		}
		var doc struct {
			Map struct {
				Completeness []completeness `json:"completeness"`
			} `json:"map"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		old, _ := json.Marshal(doc.Map.Completeness)
		for j := range doc.Map.Completeness {
			doc.Map.Completeness[j].Status = "exact"
			doc.Map.Completeness[j].Reasons = []string{}
		}
		updated, _ := json.Marshal(doc.Map.Completeness)
		h := parts[i].Header()
		omissions, _ := json.Marshal(h.Omissions)
		data = bytes.Replace(data, []byte(`"completeness":`+string(old)), []byte(`"completeness":`+string(updated)), 1)
		data = bytes.Replace(data, []byte(`"omissions":`+string(omissions)), []byte(`"omissions":[]`), 1)
		data = bytes.Replace(data, []byte(`"generation_status":"partial"`), []byte(`"generation_status":"exact"`), 1)
		data = atlasRechecksum(t, data)
		decode := atlasmap.Decode
		if h.LogicalSchema == atlasmap.LogicalSchemaV3 {
			decode = atlasmap.DecodeV3
		}
		var err error
		parts[i], err = decode(data, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		f.data[i] = data
	}
	makeIndex := atlasmap.NewWorkspaceCallIndex
	if parts[0].Header().LogicalSchema == atlasmap.LogicalSchemaV3 {
		makeIndex = atlasmap.NewWorkspaceCallIndexV3
	}
	var err error
	f.index, err = makeIndex(parts, f.prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCandidateAtlasImpact(t *testing.T) {
	var fixtures []atlasImpactFixture
	for _, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV3} {
		for _, source := range []string{
			`module main;import answer as answer;fn i32 leaf(){return answer::value();}fn i32 left(){return right()+leaf();}fn i32 right(){return left()+leaf()+leaf();}fn i32 main(){return left()+right();}`,
			`module main;unsafe extern C fn i32 opaque(i32 n);fn i32 call(){return unsafe{opaque(1)};}fn i32 main(){return call();}`,
		} {
			fixtures = append(fixtures, candidateAtlasImpactFixture(t, schema, source))
		}
	}
	// Preserve the original four fixtures for schema-mixing rejection probes.
	baseFixtures := append([]atlasImpactFixture{}, fixtures...)
	for _, f := range baseFixtures {
		fixtures = append(fixtures, candidateAtlasExactCoverage(t, f))
	}
	main := strings.Replace(atlasImpactMain, "fn Result<bytes::Buffer,u32> query(", "@effects(console) fn Result<bytes::Buffer,u32> query(", 1)
	main = strings.Replace(main, "let result=impact::explore(", "let start=meter;let result=impact::explore(", 1)
	main = strings.Replace(main, ";return output::render(", `;if(process::arg_count()>1){discard io::write_usize(meter-start);discard io::write("\n");}return output::render(`, 1)
	candidateAtlasProbe(t, "project_atlas_impact_json", main, func(t *testing.T, image string) {
		t.Run("reverse-work", func(t *testing.T) { verifyCandidateAtlasReverseWork(t, image) })
		for fi, f := range fixtures {
			t.Run(fmt.Sprintf("fixture-%d", fi), func(t *testing.T) {
				run := func(instance, symbol string, depth, symbols, packages, calls, outputBytes int, order []int, expect bool) {
					t.Helper()
					q, _ := json.Marshal(struct {
						Instance string `json:"instance"`
						Symbol   string `json:"symbol"`
						Depth    int    `json:"depth"`
						Symbols  int    `json:"symbols"`
						Packages int    `json:"packages"`
						Calls    int    `json:"calls"`
						Bytes    int    `json:"bytes"`
					}{instance, symbol, depth, symbols, packages, calls, outputBytes})
					input := append(q, 10)
					for _, i := range order {
						input = append(input, f.data[i]...)
						input = append(input, 10)
						input = append(input, f.current[i]...)
						input = append(input, 10)
					}
					code, out := candidateAtlasProcess(t, image, input)
					if !expect {
						if code != 7 {
							t.Fatalf("expected rejection: code=%d %s", code, out)
						}
						return
					}
					if code != 0 {
						t.Fatalf("impact rejected: %d %q", code, out)
					}
					want, err := f.index.Impact(instance, symbol, depth, symbols)
					if err != nil {
						t.Fatal(err)
					}
					var got atlasmap.ImpactResult
					if err = json.Unmarshal([]byte(out), &got); err != nil {
						t.Fatalf("output: %v %q", err, out)
					}
					canonical, marshalErr := json.Marshal(want)
					if marshalErr != nil {
						t.Fatal(marshalErr)
					}
					if !reflect.DeepEqual(got, want) || out != string(canonical) {
						a, _ := json.Marshal(want)
						t.Fatalf("impact differs\ngot %s\nwant %s", out, a)
					}
				}
				order := []int{0, 1}
				reverse := []int{1, 0}
				for _, part := range f.prepared.partitions {
					for _, s := range part.Symbols() {
						for _, limit := range [][2]int{{8, 64}, {0, 64}, {1, 2}, {8, 1}, {64, 4096}} {
							run(part.Header().PackageInstance, s.Visible, limit[0], limit[1], 8, 128, 100000, order, true)
							run(part.Header().PackageInstance, s.Full, limit[0], limit[1], 8, 128, 100000, reverse, true)
						}
					}
				}
				instance := f.prepared.partitions[0].Header().PackageInstance
				symbol := f.prepared.partitions[0].Symbols()[0].Visible
				for _, bad := range []struct {
					depth, symbols, packages, calls, bytes int
					order                                  []int
				}{
					{65, 8, 8, 128, 100000, order}, {8, 0, 8, 128, 100000, order}, {8, 4097, 8, 128, 100000, order},
					{8, 8, 1, 128, 100000, order}, {8, 8, 8, 0, 100000, order}, {8, 8, 8, 128, 1, order},
					{8, 8, 8, 128, 100000, []int{0, 0, 1}}, {8, 8, 8, 128, 100000, []int{}},
				} {
					run(instance, symbol, bad.depth, bad.symbols, bad.packages, bad.calls, bad.bytes, bad.order, false)
				}
				// Omit the dependency partition of the root, rather than an unrelated caller.
				root := 0
				if len(f.prepared.partitions[root].Header().Dependencies) == 0 {
					root = 1
				}
				run(instance, symbol, 8, 8, 8, 128, 100000, []int{root}, false)
				run("unknown", symbol, 8, 8, 8, 128, 100000, order, false)
				run(instance, "unknown", 8, 8, 8, 128, 100000, order, false)
				run(instance, symbol, 8, 8, 2, len(f.index.Calls()), 100000, reverse, true)
				// Only resolved calls consume the call ceiling, even with foreign calls present.
				if len(f.index.Calls()) > 0 {
					run(instance, symbol, 8, 8, 2, len(f.index.Calls())-1, 100000, order, false)
				}
				original := f.data
				f.data = append([][]byte{}, original...)
				other := fi%4 + 2
				if fi%4 >= 2 {
					other = fi%4 - 2
				}
				f.data[0] = baseFixtures[other].data[0]
				savedCurrent := f.current
				f.current = append([][]byte{}, f.current...)
				f.current[0] = baseFixtures[other].current[0]
				run(instance, symbol, 8, 8, 8, 128, 100000, order, false)
				f.current = savedCurrent
				f.data = original
				if fi%2 == 0 {
					for _, change := range []struct {
						index    int
						old, new []byte
					}{
						{root, []byte(`"name":"value"`), []byte(`"name":"missing"`)},
						{1 - root, []byte(`"public":true`), []byte(`"public":false`)},
					} {
						mutated := bytes.Replace(original[change.index], change.old, change.new, 1)
						if bytes.Equal(mutated, original[change.index]) {
							t.Fatal("join mutation absent")
						}
						mutated = atlasRechecksum(t, mutated)
						decode := atlasmap.Decode
						makeIndex := atlasmap.NewWorkspaceCallIndex
						if fi%4 >= 2 {
							decode = atlasmap.DecodeV3
							makeIndex = atlasmap.NewWorkspaceCallIndexV3
						}
						part, err := decode(mutated, 4<<20)
						if err != nil {
							t.Fatal("mutation must reach graph join", err)
						}
						parts := append([]atlasmap.Partition{}, f.prepared.partitions...)
						parts[change.index] = part
						if _, err = makeIndex(parts, f.prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}); err == nil {
							t.Fatal("Go admitted invalid target")
						}
						f.data = append([][]byte{}, original...)
						f.data[change.index] = mutated
						run(instance, symbol, 8, 8, 8, 128, 100000, order, false)
						f.data = original
					}
				}
			})
		}
	})
}

func verifyCandidateAtlasReverseWork(t *testing.T, image string) {
	t.Helper()
	const chain, unrelated = 56, 70
	var source strings.Builder
	source.WriteString("module main;fn i32 link0(){return 0;}")
	for i := 1; i <= chain; i++ {
		fmt.Fprintf(&source, "fn i32 link%d(){return link%d();}", i, i-1)
	}
	source.WriteString("fn i32 unrelated(){return 0;}")
	for i := 0; i < unrelated; i++ {
		fmt.Fprintf(&source, "fn i32 other%d(){return unrelated();}", i)
	}
	fmt.Fprintf(&source, "fn i32 main(){return link%d();}", chain)
	fixture := candidateAtlasImpactFixture(t, atlasmap.LogicalSchema, source.String())
	var instance, symbol string
	resolveWork := 0
	for _, partition := range fixture.prepared.partitions {
		header := partition.Header()
		resolveWork += len(header.PackageInstance)*2 + 1
		if header.PackageUID != "example.session.app" {
			continue
		}
		instance = header.PackageInstance
		for i, candidate := range partition.Symbols() {
			if candidate.Declaration.Name == "link0" {
				symbol = candidate.Visible
				resolveWork += (i + 1) * (len(symbol) + 256)
				break
			}
		}
		break
	}
	if symbol == "" {
		t.Fatal("missing chain root")
	}

	query, err := json.Marshal(map[string]any{
		"instance": instance, "symbol": symbol, "depth": 64, "symbols": 64,
		"packages": 2, "calls": 128, "bytes": 100000,
	})
	if err != nil {
		t.Fatal(err)
	}
	packet := append(query, '\n')
	for i, data := range fixture.data {
		packet = append(packet, data...)
		packet = append(packet, '\n')
		packet = append(packet, fixture.current[i]...)
		packet = append(packet, '\n')
	}
	code, out := candidateAtlasProcess(t, image, packet, "meter")
	work, body, ok := strings.Cut(out, "\n")
	used, err := strconv.Atoi(work)
	if code != 0 || !ok || err != nil {
		t.Fatalf("work probe: %d %s", code, out)
	}
	want, err := fixture.index.Impact(instance, symbol, 64, 64)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(want)
	if err != nil || body != string(canonical) || len(want.Nodes) != chain+2 || !want.TraversalComplete {
		t.Fatalf("chain result differs: %v %s", err, body)
	}

	// A chain inserts one new caller per step, so the old visitation scan
	// charges 1+...+(N-1). Its ordered output charges one compare per node.
	nodes, edges := len(want.Nodes), len(fixture.index.Calls())
	commonWork := resolveWork + nodes*(nodes-1)/2 + (nodes-1)*256
	oldIncomingWork := nodes * edges
	newIncomingWork := used - commonWork
	t.Logf("nodes=%d calls=%d incoming work %d -> %d", nodes, edges, oldIncomingWork, newIncomingWork)
	if newIncomingWork <= 0 || newIncomingWork >= oldIncomingWork/2 {
		t.Fatalf("reverse index did not cut incoming work by at least half")
	}
}

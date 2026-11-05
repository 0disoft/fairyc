package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestCandidateAtlasGenericJoin(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, `module main;import answer as answer;
resource Token<T>{T value;cleanup{let T observed=answer::id<T>(this.value);}}
fn T wrap<T>(T value){return answer::id<T>(value);}
pub fn T unused<T>(T value){return value;}
fn i32 main(){let a=Token<i32>{.value=1};let b=Token<u8>{.value=2};discard wrap<u8>(3);return wrap<i32>(answer::other<i32>(4));}`)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", `module answer;
pub fn T id<T>(T value){let ref T view=ref value;return *view;}
pub fn T other<T>(T value){return value;}
pub fn i32 value(){return 42;}`)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("generic join source oracle: %d %s", code, &diagnostic)
	}
	index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
	if err != nil {
		t.Fatal(err)
	}
	var data, current [][]byte
	var wantEdges strings.Builder
	cleanups, crossSpecializations := 0, 0
	for _, c := range index.Calls() {
		fmt.Fprintf(&wantEdges, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\n", c.Source.PackageInstance, c.Source.Symbol, c.SourceSpecialization, c.SourceBody, c.Target.PackageInstance, c.Target.Symbol, c.TargetSpecialization, c.Span.Source, c.Span.StartByte, c.Span.EndByte)
		if c.SourceBody == "cleanup" {
			cleanups++
		}
		if c.Source.PackageInstance != c.Target.PackageInstance && c.SourceSpecialization != "" && c.TargetSpecialization != "" {
			crossSpecializations++
		}
	}
	if cleanups != 2 || crossSpecializations != 4 {
		t.Fatalf("missing generic/cleanup edges: %d %d", cleanups, crossSpecializations)
	}
	for _, p := range prepared.partitions {
		raw, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, raw)
		current = append(current, candidateAtlasCurrent(t, prepared.expectations[p.Header().PackageInstance]))
	}
	unused := candidateAtlasImpactFixture(t, atlasmap.LogicalSchemaV3, atlasGenericWireDeclarations+`fn i32 main(){return 0;}`)
	// A verified unused exported-template body edit is source drift even though
	// the ordinary public interface did not change and no implementation exists.
	before := prepared.partitions[0].Header()
	if before.PackageUID != "example.session.app" {
		before = prepared.partitions[1].Header()
	}
	writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(`module main;import answer as answer;
resource Token<T>{T value;cleanup{let T observed=answer::id<T>(this.value);}}
fn T wrap<T>(T value){return answer::id<T>(value);}
pub fn T unused<T>(T value){return value;}
fn i32 main(){let a=Token<i32>{.value=1};let b=Token<u8>{.value=2};discard wrap<u8>(3);return wrap<i32>(answer::other<i32>(4));}`, `pub fn T unused<T>(T value){return value;}`, `pub fn T unused<T>(T value){let T copy=value;return copy;}`, 1))
	diagnostic.Reset()
	drifted, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("source drift oracle: %d %s", code, &diagnostic)
	}
	var staleCurrent []byte
	for _, p := range drifted.partitions {
		h := p.Header()
		if h.PackageUID == before.PackageUID {
			if h.InterfaceHash != before.InterfaceHash || h.GenericInterfaceHash == before.GenericInterfaceHash {
				t.Fatal("unused generic edit did not preserve ordinary/change generic interface")
			}
			staleCurrent = candidateAtlasCurrent(t, drifted.expectations[h.PackageInstance])
		}
	}
	if len(staleCurrent) == 0 {
		t.Fatal("missing source drift expectation")
	}
	main := strings.ReplaceAll(atlasImpactMain, "current::read(", "current::read_generics(")
	main = strings.Replace(main, "import project_atlas_wire as wire;", "import project_atlas_wire as wire;import project_atlas_validate as validate;", 1)
	main = strings.Replace(main, "fn Result<bytes::Buffer,u32> query(", "@effects(console) fn Result<bytes::Buffer,u32> query(", 1)
	main = strings.Replace(main, ";let subject=join::resolve(", `;if(process::arg_count()>1){
 var usize i=0;while(i<join::edge_count(ref graph)){let e=join::edge(ref graph,i);let p=join::partition(ref graph,e.source.package);let t=validate::tree(p);let span=wire::field(t,e.call,b"span");
 discard io::write_bytes(join::instance(ref graph,e.source.package));discard io::write_bytes(b"\t");discard io::write_bytes(join::visible(ref graph,e.source));discard io::write_bytes(b"\t");discard io::write_bytes(join::source_specialization(ref graph,e));discard io::write_bytes(b"\t");discard io::write_bytes(if(e.resource_body){b"cleanup"}else{b""});discard io::write_bytes(b"\t");
 discard io::write_bytes(join::instance(ref graph,e.target.package));discard io::write_bytes(b"\t");discard io::write_bytes(join::visible(ref graph,e.target));discard io::write_bytes(b"\t");discard io::write_bytes(join::target_specialization(ref graph,e));discard io::write_bytes(b"\t");discard io::write_bytes(wire::string(t,span,b"source"));discard io::write_bytes(b"\t");discard io::write_usize(wire::integer(t,span,b"start_byte"));discard io::write_bytes(b"\t");discard io::write_usize(wire::integer(t,span,b"end_byte"));discard io::write_bytes(b"\n");i+=1;
 }return .ok(bytes::empty());}let subject=join::resolve(`, 1)
	candidateAtlasProbe(t, "project_atlas_impact_json", main, func(t *testing.T, image string) {
		run := func(maps, expected [][]byte, order []int, instance, symbol string, depth, symbols, packages, calls, output int, edges bool) (int, string) {
			t.Helper()
			q, _ := json.Marshal(struct {
				Instance string `json:"instance"`
				Symbol   string `json:"symbol"`
				Depth    int    `json:"depth"`
				Symbols  int    `json:"symbols"`
				Packages int    `json:"packages"`
				Calls    int    `json:"calls"`
				Bytes    int    `json:"bytes"`
			}{instance, symbol, depth, symbols, packages, calls, output})
			packet := append(q, '\n')
			for _, i := range order {
				packet = append(packet, maps[i]...)
				packet = append(packet, '\n')
				packet = append(packet, expected[i]...)
				packet = append(packet, '\n')
			}
			var args []string
			if edges {
				args = []string{"edges"}
			}
			return candidateAtlasProcess(t, image, packet, args...)
		}
		for _, order := range [][]int{{0, 1}, {1, 0}} {
			code, out := run(data, current, order, "", "", 8, 64, 2, len(index.Calls()), 100000, true)
			if code != 0 || out != wantEdges.String() {
				t.Fatalf("exact joined implementations differ: %d\ngot %s\nwant %s", code, out, wantEdges.String())
			}
			for _, p := range prepared.partitions {
				for _, s := range p.Symbols() {
					for _, limits := range [][2]int{{8, 64}, {0, 64}, {1, 1}} {
						want, err := index.Impact(p.Header().PackageInstance, s.Visible, limits[0], limits[1])
						if err != nil {
							t.Fatal(err)
						}
						canonical, _ := json.Marshal(want)
						code, out := run(data, current, order, p.Header().PackageInstance, s.Full, limits[0], limits[1], 2, len(index.Calls()), 100000, false)
						if code != 0 || out != string(canonical) {
							t.Fatalf("generic impact %s differs: %d\ngot %s\nwant %s", s.Declaration.Name, code, out, canonical)
						}
					}
				}
			}
		}
		for _, bad := range []struct {
			name            string
			order           []int
			packages, calls int
		}{
			{"call-limit", []int{0, 1}, 2, len(index.Calls()) - 1},
			{"package-limit", []int{0, 1}, 1, 128},
			{"duplicate", []int{0, 0, 1}, 3, 128},
		} {
			t.Run(bad.name, func(t *testing.T) {
				if code, out := run(data, current, bad.order, "", "", 8, 64, bad.packages, bad.calls, 100000, true); code != 7 || out != "9" {
					t.Fatalf("join boundary: %d %q", code, out)
				}
			})
		}
		rootPart, memberPart := 0, 1
		if len(prepared.partitions[rootPart].Header().Dependencies) == 0 {
			rootPart, memberPart = 1, 0
		}
		expected := append([][]byte{}, current...)
		expected[rootPart] = staleCurrent
		if code, out := run(data, expected, []int{0, 1}, "", "", 8, 64, 2, 128, 100000, true); code != 7 || out != "8" {
			t.Fatalf("real unused-template source drift: %d %q", code, out)
		}
		if code, out := run(unused.data, unused.current, []int{1, 0}, "", "", 8, 64, 2, 0, 100000, true); code != 0 || out != "" {
			t.Fatalf("unused templates invented calls: %d %q", code, out)
		}
		if code, out := run(data, current, []int{rootPart}, "", "", 8, 64, 2, 128, 100000, true); code != 7 || out != "9" {
			t.Fatalf("missing current dependency: %d %q", code, out)
		}
		for _, field := range []string{"generic_interface_hash", "compiler_build", "interface_hash", "package_source", "files"} {
			value := []byte(`"stale"`)
			if field == "files" {
				value = []byte(`[]`)
			}
			expected := append([][]byte{}, current...)
			expected[rootPart] = atlasGenericSet(t, current[rootPart], []string{field}, value)
			if code, out := run(data, expected, []int{0, 1}, "", "", 8, 64, 2, 128, 100000, true); code != 7 || out != "8" {
				t.Fatalf("stale %s: %d %q", field, code, out)
			}
		}
		// Remote implementation existence is deliberately not inferred by a
		// single-package decoder. Only the complete current workspace can check it.
		var remotePath string
		var functions []struct {
			Calls []struct {
				Specialization string
				Target         struct {
					PackageInstance string `json:"package_instance"`
				}
			}
		}
		if err := json.Unmarshal(atlasGenericGet(t, data[rootPart], "map", "functions"), &functions); err != nil {
			t.Fatal(err)
		}
		for i, f := range functions {
			for j, c := range f.Calls {
				if c.Target.PackageInstance == prepared.partitions[memberPart].Header().PackageInstance && c.Specialization != "" {
					remotePath = fmt.Sprintf("map/functions/%d/calls/%d", i, j)
				}
			}
		}
		if remotePath == "" {
			t.Fatal("missing ordinary-to-generic cross-package call")
		}
		for _, mode := range []string{"missing", "wrong-owner", "erased"} {
			changed := append([][]byte{}, data...)
			path := strings.Split(remotePath, "/")
			call := atlasGenericGet(t, data[rootPart], path...)
			id := atlasGenericGet(t, call, "specialization")
			switch mode {
			case "missing":
				call = atlasGenericSet(t, call, []string{"specialization"}, []byte(`"g0_`+strings.Repeat("0", 64)+`"`))
			case "wrong-owner":
				call = atlasGenericSet(t, call, []string{"target", "name"}, []byte(`"id"`)) // retained ID belongs to other<T>
			case "erased":
				call = bytes.Replace(call, append(append([]byte(`"specialization":`), id...), ','), nil, 1)
			}
			changed[rootPart] = atlasRechecksum(t, atlasGenericSet(t, data[rootPart], path, call))
			part, err := atlasmap.DecodeV3(changed[rootPart], 4<<20)
			if err != nil {
				t.Fatalf("single-package boundary %s: %v", mode, err)
			}
			joined := append([]atlasmap.Partition{}, prepared.partitions...)
			joined[rootPart] = part
			if _, err := atlasmap.NewWorkspaceCallIndexV3(joined, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}); err == nil {
				t.Fatal("Go joined remote contradiction", mode)
			}
			if code, out := run(changed, current, []int{1, 0}, "", "", 8, 64, 2, 128, 100000, true); code != 7 || out != "9" {
				t.Fatalf("remote %s: %d %q", mode, code, out)
			}
		}
	})
}

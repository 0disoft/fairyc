package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const atlasGenericWireDeclarations = `module main;
fn T finish<T>(T value){return value;}
struct Cell<T>{T value;}
struct Outer<T>{Cell<T> cell;T other;}
resource Empty<T>{T value;cleanup{}}
pub resource Token<T>{Cell<T> payload;cleanup{let T copy=this.payload.value;let ref T view=ref copy;let T observed=finish<T>(*view);}}
`

func TestCandidateAtlasGenericWire(t *testing.T) {
	type fixture struct {
		name    string
		data    []byte
		generic bool
	}
	var cases []fixture
	var seed, empty, functionRecord, aggregateRecord, functionBody, aggregateBody []byte
	var specializationCount int
	var sawFunction, sawAggregate, sawEmptyCleanup bool
	cleanupRecords := 0
	for _, used := range []bool{false, true} {
		main := `fn i32 main(){return 0;}`
		if used {
			main = `fn i32 main(){let outer=Outer<i32>{.cell=Cell<i32>{.value=2},.other=3};let empty=Empty<bool>{.value=true};let token=Token<i32>{.payload=Cell<i32>{.value=4}};let other=Token<u8>{.payload=Cell<u8>{.value=5}};return finish<i32>(outer.other);}`
		}
		root := writeWorkspaceSessionFixture(t, atlasGenericWireDeclarations+main)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("generic wire source oracle: %d %s", code, &diagnostic)
		}
		for _, partition := range prepared.partitions {
			data, err := atlasmap.EncodeV3(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := atlasmap.DecodeV3(data, 4<<20); err != nil {
				t.Fatal(err)
			}
			generic := bytes.Contains(data, []byte(`"type_parameters":`))
			cases = append(cases, fixture{fmt.Sprintf("used-%v-%s", used, partition.Header().PackageUID), data, generic})
			if !generic {
				continue
			}
			if !used {
				empty = data
				continue
			}
			seed, specializationCount = data, int(partition.Header().Counts.Specializations)
			var envelope struct {
				Map struct{ Specializations []json.RawMessage }
			}
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatal(err)
			}
			for _, record := range envelope.Map.Specializations {
				var row struct {
					Declaration                  struct{ Name string }
					Function, Aggregate, Cleanup json.RawMessage
				}
				if err := json.Unmarshal(record, &row); err != nil {
					t.Fatal(err)
				}
				if len(row.Function) != 0 {
					sawFunction, functionRecord, functionBody = true, record, row.Function
				}
				if len(row.Aggregate) != 0 {
					sawAggregate, aggregateRecord, aggregateBody = true, record, row.Aggregate
				}
				if len(row.Cleanup) != 0 {
					cleanupRecords++
				}
				sawEmptyCleanup = sawEmptyCleanup || row.Declaration.Name == "Empty" && len(row.Aggregate) != 0 && len(row.Cleanup) == 0
			}
		}
	}
	if len(seed) == 0 || len(empty) == 0 || !sawFunction || !sawAggregate || !sawEmptyCleanup || cleanupRecords != 2 || !bytes.Contains(seed, []byte(`"specialization":`)) {
		t.Fatal("oracle did not exercise every specialization record family")
	}
	type rejection struct {
		name string
		data []byte
	}
	var bad []rejection
	mutate := func(name string, base []byte, old, replacement string) {
		t.Helper()
		changed := bytes.Replace(base, []byte(old), []byte(replacement), 1)
		if bytes.Equal(changed, base) {
			t.Fatal("missing wire mutation", name)
		}
		changed = atlasRechecksum(t, changed)
		if _, err := atlasmap.DecodeV3(changed, 4<<20); err == nil {
			t.Fatal("Go admitted invalid generic wire", name)
		}
		bad = append(bad, rejection{name, changed})
	}
	mutate("unknown-specialization-field", seed, `"specializations":[{`, `"specializations":[{"untrusted":0,`)
	mutate("zero-omitted-count", seed, fmt.Sprintf(`"counts":{"specializations":%d,`, specializationCount), `"counts":{"specializations":0,`)
	mutate("empty-omitted-array", empty, `"map":{"header":`, `"map":{"specializations":[],"header":`)
	mutate("empty-omitted-binders", empty, `"type_parameters":["T"]`, `"type_parameters":[]`)
	mutate("unknown-call-field", seed, `"specialization":`, `"untrusted":0,"specialization":`)
	noBody := bytes.Replace(functionRecord, append([]byte(`,"function":`), functionBody...), nil, 1)
	mutate("missing-specialization-body", seed, string(functionRecord), string(noBody))
	twoBodies := bytes.Replace(aggregateRecord, []byte(`"aggregate":`), append(append([]byte(`"function":`), functionBody...), []byte(`,"aggregate":`)...), 1)
	mutate("two-specialization-bodies", seed, string(aggregateRecord), string(twoBodies))
	wrongBody := bytes.Replace(aggregateRecord, aggregateBody, []byte("null"), 1)
	mutate("null-aggregate-body", seed, string(aggregateRecord), string(wrongBody))

	legacy := bytes.ReplaceAll(seed, []byte("/v3"), []byte("/v1"))
	legacy = atlasRechecksum(t, legacy)
	if _, err := atlasmap.Decode(legacy, 4<<20); err == nil {
		t.Fatal("Go v1 admitted generic data")
	}
	bad = append(bad, rejection{"v1-isolation", legacy})
	// Well-shaped transport is not compiler or semantic authority.
	contradiction := atlasRechecksum(t, bytes.Replace(seed, []byte(`"id":"g0_`), []byte(`"id":"bad_`), 1))
	if _, err := atlasmap.DecodeV3(contradiction, 4<<20); err == nil {
		t.Fatal("Go admitted invalid specialization identity")
	}
	cases = append(cases, fixture{"semantic-contradiction-is-only-transport", contradiction, true})
	main := `module main;import project_atlas_wire as wire;import project_atlas_validate as validate;import project_json_tree as json;` + atlasReadInput + `
 @effects(console) fn i32 main(){let data=input();let size=bytes::len(ref data);var usize meter=0;var u8[16] flag=[0;16];
 let usize length=if(process::arg_count()>1){switch(process::arg(1,ref mut flag)){case .ok(n):{n}case .err(_):{0}}}else{0};let ref u8[] mode=ref flag;
 if(bytes::equal(mode[..length],b"budget")){meter=16777216;}
 let doc=switch(wire::read(data,if(bytes::equal(mode[..length],b"size")){size-1}else{4194304},ref mut meter)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);return 7;}};
 if(bytes::equal(mode[..length],b"admit")){return switch(validate::validate(doc,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_u32(code);7}};}
 return if(io::write_bytes(json::source(wire::tree(ref doc))).is_ok){0}else{4};}`
	candidateAtlasProbe(t, "project_atlas_validate", main, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, c.data)
				if code != 0 || out != string(c.data) {
					t.Fatalf("wire lost or rejected records: %d %q", code, out)
				}
				code, out = candidateAtlasProcess(t, image, c.data, "admit")
				if c.generic && (code != 7 || out != "7") || !c.generic && (code != 0 || out != "") {
					t.Fatalf("semantic boundary changed: %d %q", code, out)
				}
			})
		}
		for _, c := range bad {
			t.Run(c.name, func(t *testing.T) {
				if code, out := candidateAtlasProcess(t, image, c.data); code != 7 || out != "5" {
					t.Fatalf("invalid transport: %d %q", code, out)
				}
			})
		}
		for _, mode := range []string{"size", "budget"} {
			code, out := candidateAtlasProcess(t, image, seed, mode)
			if code != 7 || out != "3" {
				t.Fatalf("%s limit: %d %q", mode, code, out)
			}
		}
	})
}

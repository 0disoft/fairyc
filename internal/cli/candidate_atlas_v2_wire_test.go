package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const atlasV2WireSource = `module main;import answer as a;
pub fn T identity<T>(T value){return value;}
pub fn T unused<T>(T value){return value;}
fn i32 main(){let x=identity<i64>(42);return identity<i32>(a::value());}`

func TestCandidateAtlasV2Wire(t *testing.T) {
	type fixture struct {
		name  string
		data  []byte
		valid bool
	}
	var cases []fixture
	for _, source := range []string{workspaceSessionRootSource, atlasV2WireSource} {
		root := writeWorkspaceSessionFixture(t, source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV2)
		if code != ExitSuccess {
			t.Fatalf("v2 source oracle: %d %s", code, &diagnostic)
		}
		for _, partition := range prepared.partitions {
			data, err := atlasmap.EncodeV2(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := atlasmap.DecodeV2(data, 4<<20); err != nil {
				t.Fatal(err)
			}
			cases = append(cases, fixture{fmt.Sprintf("valid-%d", len(cases)), data, true})
		}
	}
	seed := cases[0].data
	for _, c := range cases {
		if bytes.Contains(c.data, []byte(`"generic_interface_hash":`)) {
			seed = c.data
			break
		}
	}
	var envelope struct {
		Map struct {
			Header struct {
				Generic string `json:"generic_interface_hash"`
			} `json:"header"`
		} `json:"map"`
	}
	if err := json.Unmarshal(seed, &envelope); err != nil || envelope.Map.Header.Generic == "" {
		t.Fatalf("missing generic interface seed: %v", err)
	}
	for name, data := range map[string][]byte{
		"whitespace":          append([]byte(" "), seed...),
		"trailing":            append(append([]byte{}, seed...), []byte("{}")...),
		"truncated":           seed[:len(seed)-1],
		"duplicate":           bytes.Replace(seed, []byte(`"schema":`), []byte(`"schema":"ignored","schema":`), 1),
		"unknown":             atlasRechecksum(t, bytes.Replace(seed, []byte(`"header":{`), []byte(`"header":{"untrusted":0,`), 1)),
		"checksum":            bytes.Replace(seed, []byte(`"package_uid":"example`), []byte(`"package_uid":"altered`), 1),
		"encoding":            atlasRechecksum(t, bytes.Replace(seed, []byte(`fairyc.fymap-json/v2`), []byte(`fairyc.fymap-json/v3`), 1)),
		"logical-version":     atlasRechecksum(t, bytes.Replace(seed, []byte(`fairyc.fymap-logical/v2`), []byte(`fairyc.fymap-logical/v3`), 1)),
		"noncanonical-string": bytes.Replace(seed, []byte(`"schema"`), []byte(`"\u0073chema"`), 1),
		"empty-generic-hash":  atlasRechecksum(t, bytes.Replace(seed, []byte(`"generic_interface_hash":"`+envelope.Map.Header.Generic+`"`), []byte(`"generic_interface_hash":""`), 1)),
	} {
		if _, err := atlasmap.DecodeV2(data, 4<<20); err == nil {
			t.Fatalf("invalid Go seed %s", name)
		}
		cases = append(cases, fixture{name, data, false})
	}
	for _, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV3} {
		prepared := candidateAtlasInputs(t, schema)
		for _, partition := range prepared.partitions {
			encode := atlasmap.Encode
			if schema == atlasmap.LogicalSchemaV3 {
				encode = atlasmap.EncodeV3
			}
			data, err := encode(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, fixture{fmt.Sprintf("wrong-version-%s-%d", schema, len(cases)), data, false})
		}
	}
	root := writeWorkspaceSessionFixture(t, atlasGenericWireDeclarations+`fn i32 main(){let cell=Cell<i32>{.value=42};let token=Token<i32>{.payload=cell};return 0;}`)
	var diagnostic bytes.Buffer
	aggregates, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("aggregate source oracle: %d %s", code, &diagnostic)
	}
	aggregateCases := 0
	for _, partition := range aggregates.partitions {
		data, err := atlasmap.EncodeV3(partition, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte(`"aggregate":`)) {
			continue
		}
		data = atlasRechecksum(t, bytes.ReplaceAll(data, []byte("/v3"), []byte("/v2")))
		if _, err := atlasmap.DecodeV2(data, 4<<20); err == nil {
			t.Fatal("Go admitted v3 aggregate under v2")
		}
		cases = append(cases, fixture{"aggregate-body", data, false})
		aggregateCases++
	}
	if aggregateCases == 0 {
		t.Fatal("aggregate rejection seed has no aggregate body")
	}
	main := `module main;import project_atlas_wire as wire;import project_json_tree as json;` + atlasReadInput + `
@effects(console) fn i32 main(){let data=input();let size=bytes::len(ref data);var usize limit=4194304;var usize meter=0;var bool legacy=false;
 if(process::arg_count()==2){var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 7;}};let ref u8[] mode=ref raw;
  if(bytes::equal(mode[..n],b"exact")){limit=size;}else if(bytes::equal(mode[..n],b"one-short")){limit=size-1;}else if(bytes::equal(mode[..n],b"work")){meter=16777216;}else if(bytes::equal(mode[..n],b"legacy")){legacy=true;}else{return 7;}
 }
 let document=switch(if(legacy){wire::read(data,limit,ref mut meter)}else{wire::read_v2(data,limit,ref mut meter)}){case .ok(p):{p}case .err(_):{return 7;}};
 return if(wire::version(ref document)==2 && io::write_bytes(json::source(wire::tree(ref document))).is_ok){0}else{7};
}`
	t.Logf("explicit v2 framing cases: %d per profile", len(cases))
	candidateAtlasProbe(t, "project_atlas_wire", main, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, c.data)
				if c.valid {
					if code != 0 || output != string(c.data) {
						t.Fatalf("valid v2 transport: %d %q", code, output)
					}
				} else if code != 7 || output != "" {
					t.Fatalf("invalid v2 transport: %d %q", code, output)
				}
			})
		}
		for _, mode := range []string{"exact", "one-short", "work", "legacy"} {
			code, output := candidateAtlasProcess(t, image, seed, mode)
			if mode == "exact" {
				if code != 0 || output != string(seed) {
					t.Fatalf("exact byte bound: %d %q", code, output)
				}
			} else if code != 7 || output != "" {
				t.Fatalf("v2 boundary %s: %d %q", mode, code, output)
			}
		}
	})
}

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

type candidateAtlasV2AdmissionFixture struct {
	name                     string
	data, expected           []byte
	symbols, implementations int
	valid                    bool
}

// Expectations belong to independently checked source, not the encoded map.
func candidateAtlasV2AdmissionFixtures(t *testing.T) []candidateAtlasV2AdmissionFixture {
	t.Helper()
	var fixtures []candidateAtlasV2AdmissionFixture
	for i, source := range []string{workspaceSessionRootSource, atlasV2WireSource, `module main;pub fn T unused<T>(T value){return value;}fn i32 main(){return 0;}`} {
		root := writeWorkspaceSessionFixture(t, source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV2)
		if code != 0 {
			t.Fatalf("v2 admission source oracle: %d %s", code, &diagnostic)
		}
		for _, partition := range prepared.partitions {
			data, err := atlasmap.EncodeV2(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := atlasmap.DecodeV2(data, 4<<20); err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, candidateAtlasV2AdmissionFixture{
				name: fmt.Sprintf("source-%d-%s", i, partition.Header().PackageUID), data: data,
				expected: candidateAtlasCurrent(t, prepared.expectations[partition.Header().PackageInstance]),
				symbols:  len(partition.Symbols()), implementations: partition.Header().Counts.Specializations, valid: true,
			})
		}
	}
	return fixtures
}

func TestCandidateAtlasV2Validation(t *testing.T) {
	cases := candidateAtlasV2AdmissionFixtures(t)
	var seed candidateAtlasV2AdmissionFixture
	for _, c := range cases {
		if c.implementations == 2 {
			seed = c
			break
		}
	}
	if seed.data == nil {
		t.Fatal("missing two-binding generic semantic seed")
	}
	mutate := func(name, path, value string, accepted bool) {
		t.Helper()
		data := atlasRechecksum(t, atlasGenericSet(t, seed.data, strings.Split(path, "/"), []byte(value)))
		_, err := atlasmap.DecodeV2(data, 4<<20)
		if (err == nil) != accepted {
			t.Fatalf("Go v2 semantic oracle %s accepted=%v: %v", name, accepted, err)
		}
		cases = append(cases, candidateAtlasV2AdmissionFixture{name: name, data: data, symbols: seed.symbols, implementations: seed.implementations, valid: accepted})
	}
	mutate("invalid-id", "map/specializations/0/id", `"g0_invalid"`, false)
	mutate("uppercase-id", "map/specializations/0/id", `"g0_`+strings.Repeat("A", 64)+`"`, false)
	mutate("missing-owner", "map/specializations/0/declaration/name", `"missing"`, false)
	mutate("ordinary-owner", "map/specializations/0/declaration/name", `"main"`, false)
	mutate("arity", "map/specializations/0/type_arguments", `[]`, false)
	mutate("unknown-scalar", "map/specializations/0/type_arguments/0/name", `"invented"`, false)
	mutate("void-argument", "map/specializations/0/type_arguments/0/name", `"void"`, false)
	mutate("mutable-scalar", "map/specializations/0/type_arguments/0/mutable", `true`, false)
	mutate("unbound-argument", "map/specializations/0/type_arguments/0/kind", `"parameter"`, false)
	mutate("wrong-summary", "map/specializations/0/function/function/name", `"main"`, false)
	mutate("wrong-provenance", "map/specializations/0/function/provenance", `"compiler-declared"`, false)
	mutate("wrong-count", "map/header/counts/specializations", `1`, false)
	mutate("zero-source-interface", "map/header/generic_interface_hash", `"fygif0-sha256-`+strings.Repeat("0", 64)+`"`, false)
	otherArgs := atlasGenericGet(t, seed.data, "map", "specializations", "1", "type_arguments")
	mutate("duplicate-binding", "map/specializations/0/type_arguments", string(otherArgs), false)
	var records []json.RawMessage
	if err := json.Unmarshal(atlasGenericGet(t, seed.data, "map", "specializations"), &records); err != nil {
		t.Fatal(err)
	}
	records[0], records[1] = records[1], records[0]
	reordered, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	mutate("record-order", "map/specializations", string(reordered), false)
	var functions []json.RawMessage
	if err := json.Unmarshal(atlasGenericGet(t, seed.data, "map", "functions"), &functions); err != nil {
		t.Fatal(err)
	}
	mainIndex := -1
	for i, f := range functions {
		var key struct{ Function struct{ Name string } }
		if err := json.Unmarshal(f, &key); err != nil {
			t.Fatal(err)
		}
		if key.Function.Name == "main" {
			mainIndex = i
		}
	}
	if mainIndex < 0 {
		t.Fatal("missing ordinary caller")
	}
	callPath := fmt.Sprintf("map/functions/%d/calls/0", mainIndex)
	call := atlasGenericGet(t, seed.data, strings.Split(callPath, "/")...)
	id := atlasGenericGet(t, call, "specialization")
	erased := bytes.Replace(call, append(append([]byte(`"specialization":`), id...), ','), nil, 1)
	if bytes.Equal(call, erased) {
		t.Fatal("missing local specialized call")
	}
	mutate("missing-local-call-id", callPath, string(erased), false)
	mutate("missing-local-implementation", callPath+"/specialization", `"g0_`+strings.Repeat("f", 64)+`"`, false)
	mutate("wrong-call-owner", callPath+"/target/name", `"main"`, false)
	mutate("ordinary-template-summary", fmt.Sprintf("map/functions/%d", mainIndex), string(atlasGenericGet(t, seed.data, "map", "specializations", "0", "function")), false)
	var complete []struct{ Relation string }
	if err := json.Unmarshal(atlasGenericGet(t, seed.data, "map", "completeness"), &complete); err != nil {
		t.Fatal(err)
	}
	for i, c := range complete {
		if c.Relation == "generic-template-bodies" {
			mutate("template-under-count", fmt.Sprintf("map/completeness/%d/unknown", i), "0", false)
		}
		if c.Relation == "generic-specializations" {
			mutate("implementation-under-count", fmt.Sprintf("map/completeness/%d/resolved", i), "0", false)
			mutate("snapshot-wide-count", fmt.Sprintf("map/completeness/%d/resolved", i), "10000", true)
		}
	}
	otherName := atlasGenericGet(t, otherArgs, "0", "name")
	for _, scalar := range []string{"bool", "i8", "i16", "i32", "i64", "isize", "u8", "u16", "u32", "u64", "usize", "f32", "f64"} {
		value, _ := json.Marshal(scalar)
		if bytes.Equal(value, otherName) {
			continue
		}
		mutate("scalar-"+scalar, "map/specializations/0/type_arguments/0/name", string(value), true)
	}
	// A legitimate v3 unused aggregate template passes framing after resealing,
	// but v2 semantic admission must still refuse its source declaration family.
	root := writeWorkspaceSessionFixture(t, atlasGenericWireDeclarations+`fn i32 main(){return 0;}`)
	var diagnostic bytes.Buffer
	aggregates, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("aggregate refusal source: %d %s", code, &diagnostic)
	}
	aggregateCases := 0
	for _, p := range aggregates.partitions {
		if p.Header().PackageUID != "example.session.app" {
			continue
		}
		data, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		data = atlasRechecksum(t, bytes.ReplaceAll(data, []byte("/v3"), []byte("/v2")))
		if _, err := atlasmap.DecodeV2(data, 4<<20); err == nil {
			t.Fatal("Go admitted v3 templates as v2")
		}
		cases = append(cases, candidateAtlasV2AdmissionFixture{name: "aggregate-owner", data: data})
		aggregateCases++
	}
	if aggregateCases != 1 {
		t.Fatal("missing aggregate refusal")
	}
	main := `module main;import project_atlas_wire as wire;import project_atlas_validate as validate;` + atlasReadInput + `
@effects(console) fn i32 main(){var usize meter=0;let doc=switch(wire::read_v2(input(),4194304,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 var u32 mode=0;if(process::arg_count()==2){var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 7;}};let ref u8[] s=ref raw;
  if(bytes::equal(s[..n],b"ordinary")){mode=1;}else if(bytes::equal(s[..n],b"v3")){mode=2;}else if(bytes::equal(s[..n],b"budget")){meter=16777216;}else{return 7;}}
 let part=switch(if(mode==1){validate::validate(doc,ref mut meter)}else if(mode==2){validate::validate_generics(doc,ref mut meter)}else{validate::validate_v2(doc,ref mut meter)}){case .ok(p):{p}case .err(_):{return 7;}};
 if(!io::write_usize(validate::symbol_count(ref part)).is_ok || !io::write("/").is_ok || !io::write_usize(validate::specialization_count(ref part)).is_ok){return 4;}return 0;
}`
	t.Logf("v2 semantic cases: %d per profile plus three entry/budget controls", len(cases))
	candidateAtlasProbe(t, "project_atlas_validate", main, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, c.data)
				if c.valid {
					if code != 0 || output != strconv.Itoa(c.symbols)+"/"+strconv.Itoa(c.implementations) {
						t.Fatalf("valid v2 facts: %d %q", code, output)
					}
				} else if code != 7 || output != "" {
					t.Fatalf("invalid v2 facts: %d %q", code, output)
				}
			})
		}
		for _, mode := range []string{"ordinary", "v3", "budget"} {
			code, output := candidateAtlasProcess(t, image, seed.data, mode)
			if code != 7 || output != "" {
				t.Fatalf("v2 entry %s: %d %q", mode, code, output)
			}
		}
	})
}

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

// Preserve member order so rechecksummed contradictions reach semantic checks.
func atlasGenericSet(t *testing.T, data []byte, path []string, replacement []byte) []byte {
	t.Helper()
	if len(path) == 0 {
		return replacement
	}
	if data[0] == '[' {
		var rows []json.RawMessage
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatal(err)
		}
		i, err := strconv.Atoi(path[0])
		if err != nil || i < 0 || i >= len(rows) {
			t.Fatal("invalid array path", path)
		}
		rows[i] = atlasGenericSet(t, rows[i], path[1:], replacement)
		result, err := json.Marshal(rows)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	old, ok := fields[path[0]]
	if !ok {
		t.Fatal("missing object path", path)
	}
	key, _ := json.Marshal(path[0])
	prefix := append(key, ':')
	return bytes.Replace(data, append(append([]byte{}, prefix...), old...), append(prefix, atlasGenericSet(t, old, path[1:], replacement)...), 1)
}

func atlasGenericGet(t *testing.T, data []byte, path ...string) json.RawMessage {
	t.Helper()
	for _, key := range path {
		if data[0] == '[' {
			var rows []json.RawMessage
			if err := json.Unmarshal(data, &rows); err != nil {
				t.Fatal(err)
			}
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(rows) {
				t.Fatal("invalid array path", path)
			}
			data = rows[i]
		} else {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			var ok bool
			data, ok = fields[key]
			if !ok {
				t.Fatal("missing object path", path)
			}
		}
	}
	return data
}

func TestCandidateAtlasGenericValidation(t *testing.T) {
	type fixture struct {
		name  string
		data  []byte
		count int
		ok    bool
	}
	var cases []fixture
	var seed []byte
	var records []json.RawMessage
	var function, aggregate, nested, cleanup, empty, mainFunction, otherFunction int
	function, aggregate, nested, cleanup, empty, mainFunction, otherFunction = -1, -1, -1, -1, -1, -1, -1
	for _, used := range []bool{false, true} {
		body := `fn i32 main(){return 0;}`
		if used {
			body = `fn i32 main(){let outer=Outer<i32>{.cell=Cell<i32>{.value=2},.other=3};let empty=Empty<bool>{.value=true};let token=Token<i32>{.payload=Cell<i32>{.value=4}};let other=Token<u8>{.payload=Cell<u8>{.value=5}};return finish<i32>(outer.other);}`
		}
		root := writeWorkspaceSessionFixture(t, atlasGenericWireDeclarations+body)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("generic semantic oracle: %d %s", code, &diagnostic)
		}
		for _, partition := range prepared.partitions {
			data, err := atlasmap.EncodeV3(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := atlasmap.DecodeV3(data, 4<<20); err != nil {
				t.Fatal(err)
			}
			cases = append(cases, fixture{fmt.Sprintf("used-%v-%s", used, partition.Header().PackageUID), data, partition.Header().Counts.Specializations, true})
			if !used || !bytes.Contains(data, []byte(`"specializations":[`)) {
				continue
			}
			seed = data
			var document struct {
				Map struct{ Specializations, Functions []json.RawMessage }
			}
			if err := json.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			records = document.Map.Specializations
			for i, r := range records {
				var row struct {
					Declaration                  struct{ Name string }
					Function, Aggregate, Cleanup json.RawMessage
				}
				if err := json.Unmarshal(r, &row); err != nil {
					t.Fatal(err)
				}
				if len(row.Function) > 0 {
					if function < 0 {
						function = i
					} else {
						otherFunction = i
					}
				}
				if row.Declaration.Name == "Cell" {
					aggregate = i
				}
				if row.Declaration.Name == "Outer" {
					nested = i
				}
				if len(row.Cleanup) > 0 {
					cleanup = i
				}
				if row.Declaration.Name == "Empty" {
					empty = i
				}
			}
			for i, r := range document.Map.Functions {
				if bytes.Contains(r, []byte(`"name":"main"`)) {
					mainFunction = i
				}
			}
		}
	}
	if function < 0 || otherFunction < 0 || aggregate < 0 || nested < 0 || cleanup < 0 || empty < 0 || mainFunction < 0 {
		t.Fatal("missing semantic fixture family")
	}
	mutate := func(name string, base []byte, path string, value string, accepted bool) {
		t.Helper()
		data := atlasRechecksum(t, atlasGenericSet(t, base, strings.Split(path, "/"), []byte(value)))
		_, err := atlasmap.DecodeV3(data, 4<<20)
		if (err == nil) != accepted {
			t.Fatalf("Go semantic oracle %s accepted=%v: %v", name, accepted, err)
		}
		cases = append(cases, fixture{name, data, len(records), accepted})
	}
	record := func(i int, tail string) string { return fmt.Sprintf("map/specializations/%d/%s", i, tail) }
	mutate("bad-id", seed, record(function, "id"), `"g0_invalid"`, false)
	mutate("uppercase-id", seed, record(function, "id"), `"g0_`+strings.Repeat("A", 64)+`"`, false)
	mutate("wrong-owner", seed, record(function, "declaration/name"), `"missing"`, false)
	mutate("ordinary-owner", seed, record(function, "declaration/name"), `"main"`, false)
	mutate("arity", seed, record(function, "type_arguments"), `[]`, false)
	mutate("unknown-scalar", seed, record(function, "type_arguments/0/name"), `"invented"`, false)
	mutate("void-argument", seed, record(function, "type_arguments/0/name"), `"void"`, false)
	mutate("unbound-argument", seed, record(function, "type_arguments/0/kind"), `"parameter"`, false)
	mutate("mutable-scalar", seed, record(function, "type_arguments/0/mutable"), `true`, false)
	mutate("wrong-function-summary", seed, record(function, "function/function/name"), `"main"`, false)
	mutate("wrong-function-provenance", seed, record(function, "function/provenance"), `"compiler-declared"`, false)
	mutate("wrong-aggregate-provenance", seed, record(aggregate, "aggregate/provenance"), `"compiler-declared"`, false)
	mutate("wrong-aggregate-owner", seed, record(aggregate, "aggregate/key/name"), `"Outer"`, false)
	mutate("aggregate-public", seed, record(aggregate, "aggregate/public"), `true`, false)
	mutate("aggregate-field-name", seed, record(aggregate, "aggregate/fields/0/name"), `"other"`, false)
	mutate("aggregate-field-count", seed, record(aggregate, "aggregate/fields"), `[]`, false)
	mutate("aggregate-field-type", seed, record(aggregate, "aggregate/fields/0/type/name"), `"bool"`, false)
	mutate("aggregate-self-type", seed, record(aggregate, "aggregate/type/arguments/0/name"), `"bool"`, false)
	mutate("nested-substitution", seed, record(nested, "aggregate/fields/0/type/arguments/0/name"), `"bool"`, false)
	mutate("field-span-start", seed, record(aggregate, "aggregate/fields/0/span/start_byte"), `9223372036854775807`, false)
	mutate("field-span-source", seed, record(aggregate, "aggregate/fields/0/span/source"), `"missing.fy"`, false)
	mutate("struct-drop", seed, record(aggregate, "aggregate/drop_kind"), `"empty"`, false)
	mutate("cleanup-owner", seed, record(cleanup, "cleanup/function/name"), `"Empty"`, false)
	mutate("cleanup-required", seed, record(cleanup, "aggregate/drop_kind"), `"empty"`, false)
	mutate("empty-cleanup", seed, record(empty, "aggregate/drop_kind"), `"non-empty"`, false)
	mutate("cleanup-call-span", seed, record(cleanup, "cleanup/calls/0/span/start_byte"), `0`, false)
	mutate("cleanup-borrow-span", seed, record(cleanup, "cleanup/borrows/0/span/start_byte"), `0`, false)
	mutate("cleanup-call-id", seed, record(cleanup, "cleanup/calls/0/specialization"), `"g0_`+strings.Repeat("0", 64)+`"`, false)
	mutate("ordinary-call-id", seed, fmt.Sprintf("map/functions/%d/calls/0/specialization", mainFunction), `"g0_`+strings.Repeat("0", 64)+`"`, false)
	mutate("specialization-count", seed, "map/header/counts/specializations", "1", false)
	mutate("relation-count", seed, "map/header/counts/relations", "0", false)
	mutate("generic-hash-zero", seed, "map/header/generic_interface_hash", `"fygif0-sha256-`+strings.Repeat("0", 64)+`"`, false)
	mutate("duplicate-binding", seed, record(function, "type_arguments"), string(atlasGenericGet(t, records[otherFunction], "type_arguments")), false)
	mutate("duplicate-id", seed, record(function, "id"), string(atlasGenericGet(t, records[otherFunction], "id")), false)
	callPath := fmt.Sprintf("map/functions/%d/calls/0", mainFunction)
	call := atlasGenericGet(t, seed, strings.Split(callPath, "/")...)
	id := atlasGenericGet(t, call, "specialization")
	erased := bytes.Replace(call, append(append([]byte(`"specialization":`), id...), ','), nil, 1)
	mutate("missing-local-call-id", seed, callPath, string(erased), false)
	mutate("id-on-wrong-declaration", seed, callPath+"/target/name", `"main"`, false)
	mutate("ordinary-template-body", seed, fmt.Sprintf("map/functions/%d", mainFunction), string(atlasGenericGet(t, records[function], "function")), false)
	var complete []struct{ Relation string }
	if err := json.Unmarshal(atlasGenericGet(t, seed, "map", "completeness"), &complete); err != nil {
		t.Fatal(err)
	}
	for i, c := range complete {
		if c.Relation == "generic-template-bodies" {
			mutate("template-under-count", seed, fmt.Sprintf("map/completeness/%d/unknown", i), "0", false)
		}
		if c.Relation == "generic-specializations" {
			mutate("implementation-under-count", seed, fmt.Sprintf("map/completeness/%d/resolved", i), "0", false)
			mutate("snapshot-wide-coverage", seed, fmt.Sprintf("map/completeness/%d/resolved", i), "10000", true)
		}
	}
	// IDs establish format/ownership here; compiler provenance/current source
	// remains independent. Function facts contain no concrete signature to alter.
	for _, scalar := range []string{"bool", "i8", "i16", "i32", "i64", "isize", "u8", "u16", "u32", "u64", "usize", "f32", "f64"} {
		value, _ := json.Marshal(scalar)
		other := atlasGenericGet(t, records[otherFunction], "type_arguments", "0", "name")
		if bytes.Equal(value, other) {
			continue
		} // would intentionally repeat a binding
		mutate("scalar-"+scalar, seed, record(function, "type_arguments/0/name"), string(value), true)
	}
	// Concrete span may widen to the containing declaration, but never narrow
	// away the original field. The complete validator must accept this boundary.
	mutate("field-span-containing", seed, record(aggregate, "aggregate/fields/0/span"), string(atlasGenericGet(t, records[aggregate], "aggregate", "span")), true)
	rows := append([]json.RawMessage{}, records...)
	rows[0], rows[1] = rows[1], rows[0]
	reordered, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	mutate("record-order", seed, "map/specializations", string(reordered), false)
	main := `module main;import project_atlas_wire as wire;import project_atlas_validate as validate;` + atlasReadInput + `
 @effects(console) fn i32 main(){var usize meter=0;let doc=switch(wire::read(input(),4194304,ref mut meter)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);return 7;}};
 if(process::arg_count()>1){meter=16777216;}
 return switch(validate::validate_generics(doc,ref mut meter)){case .ok(p):{discard io::write_usize(validate::specialization_count(ref p));0}case .err(e):{discard io::write_u32(e);7}};}`
	candidateAtlasProbe(t, "project_atlas_validate", main, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, c.data)
				if c.ok {
					if code != 0 || out != strconv.Itoa(c.count) {
						t.Fatalf("expected %d implementations: %d %q", c.count, code, out)
					}
				} else if code != 7 || out != "7" {
					t.Fatalf("invalid semantic document: %d %q", code, out)
				}
			})
		}
		if code, _ := candidateAtlasProcess(t, image, seed, "budget"); code != 7 {
			t.Fatal("exhausted shared semantic budget accepted", code)
		}
	})
}

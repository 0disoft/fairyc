package cli

import (
	"bytes"
	"fmt"
	"github.com/0disoft/fairyc/internal/atlasmap"
	"strings"
	"testing"
)

func TestCandidateAtlasValidation(t *testing.T) {
	type fixture struct {
		name string
		data []byte
		ok   bool
	}
	var cases []fixture
	for _, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV3} {
		p := candidateAtlasInputs(t, schema)
		for i, partition := range p.partitions {
			encode, decode := atlasmap.Encode, atlasmap.Decode
			if schema == atlasmap.LogicalSchemaV3 {
				encode, decode = atlasmap.EncodeV3, atlasmap.DecodeV3
			}
			data, err := encode(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, fixture{fmt.Sprintf("%s-%d", schema, i), data, true})
			if i != 0 {
				continue
			}
			h := partition.Header()
			f := partition.Files()[0]
			sym := partition.Symbols()[0]
			changes := []struct{ name, old, new string }{
				{"unsupported-generic", `"key":{`, `"type_parameters":["T"],"key":{`},
				{"compiler-algorithm", `"compiler_build_algorithm":"` + h.CompilerBuildAlgorithm + `"`, `"compiler_build_algorithm":"unknown"`},
				{"instance-zero", `"package_instance":"` + h.PackageInstance + `"`, `"package_instance":"fypi1-sha256-` + strings.Repeat("0", 64) + `"`},
				{"interface-zero", `"interface_hash":"` + h.InterfaceHash + `"`, `"interface_hash":"fyif5-sha256-` + strings.Repeat("0", 64) + `"`},
				{"count", fmt.Sprintf(`"files":%d`, h.Counts.Files), `"files":0`},
				{"file-digest", `"digest":"` + f.Digest + `"`, `"digest":"fyfile1-sha256-` + strings.Repeat("0", 64) + `"`},
				{"line-map", `"line_map":"` + f.LineMap + `"`, `"line_map":"fylm1-sha256-` + strings.Repeat("0", 64) + `"`},
				{"module-source", `"source":"` + f.Path + `"`, `"source":"missing.fy"`},
				{"symbol-digest", `"full":"` + sym.Full + `"`, `"full":"fysym1-sha256-256-` + strings.Repeat("a", 64) + `"`},
				{"symbol-visible", `"visible":"` + sym.Visible + `"`, `"visible":"fysym1-sha256-128-` + strings.Repeat("b", 32) + `"`},
				{"generation", `"generation_status":"` + h.GenerationStatus + `"`, `"generation_status":"invalid"`},
				{"provenance", `"provenance":"compiler-observed"`, `"provenance":"compiler-declared"`},
				{"span", fmt.Sprintf(`"start_byte":%d`, partition.Declarations()[0].Span.StartByte), `"start_byte":9223372036854775807`},
			}
			for _, change := range changes {
				if !bytes.Contains(data, []byte(change.old)) {
					if change.name == "provenance" {
						continue
					}
					t.Fatal("mutation absent", change.name)
				}
				mutated := atlasRechecksum(t, bytes.Replace(data, []byte(change.old), []byte(change.new), 1))
				if _, err = decode(mutated, 4<<20); err == nil {
					t.Fatal("Go admitted mutation", change.name)
				}
				cases = append(cases, fixture{change.name, mutated, false})
			}
		}
	}

	for _, source := range []string{
		`module main;struct Pair{i32 value;}enum State{ready;}fn i32 get(ref Pair p){return p.value;}fn i32 main(){let p=Pair{.value=1};return get(ref p);}`,
		`module main;unsafe extern C fn i32 opaque(i32 n);fn i32 call(i32 n){return unsafe{opaque(n)};}fn i32 main(){return call(1);}`,
		`module main;resource Box{i32 value;cleanup{}}fn i32 main(){let box=Box{.value=2};return 0;}`,
	} {
		root := writeWorkspaceSessionFixture(t, source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("prepare shape: %d %s", code, &diagnostic)
		}
		for _, partition := range prepared.partitions {
			data, err := atlasmap.EncodeV3(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, fixture{"record-enum-foreign-resource", data, true})
		}
	}
	main := `module main;import project_atlas_wire as wire;import project_atlas_validate as validate;` + atlasReadInput + `@effects(console) fn i32 main(){var usize meter=0;let doc=switch(wire::read(input(),4194304,ref mut meter)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);return 7;}};return switch(validate::validate(doc,ref mut meter)){case .ok(p):{discard io::write_usize(validate::symbol_count(ref p));0}case .err(e):{discard io::write_u32(e);7}};}`
	candidateAtlasProbe(t, "project_atlas_validate", main, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, c.data)
				if (code == 0) != c.ok {
					t.Fatalf("want accepted=%v got %d %q", c.ok, code, out)
				}
				if !c.ok && out != "7" {
					t.Fatalf("semantic mutation did not reach semantic checks: %q", out)
				}
			})
		}
	})
}

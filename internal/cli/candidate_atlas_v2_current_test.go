package cli

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func candidateAtlasV2StaleCurrent(t *testing.T) candidateAtlasV2AdmissionFixture {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, `module main;pub fn T unused<T>(T value){return value;}fn i32 main(){return 0;}`)
	var diagnostic bytes.Buffer
	before, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV2)
	if code != 0 {
		t.Fatalf("before source oracle: %d %s", code, &diagnostic)
	}
	writeWorkspaceFile(t, root, "src/main.fy", `module main;pub fn T unused<T>(T value){let T saved=value;return saved;}fn i32 main(){return 0;}`)
	diagnostic.Reset()
	after, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV2)
	if code != 0 {
		t.Fatalf("after source oracle: %d %s", code, &diagnostic)
	}
	for _, old := range before.partitions {
		if old.Header().PackageUID != "example.session.app" {
			continue
		}
		for _, fresh := range after.partitions {
			if fresh.Header().PackageUID != old.Header().PackageUID {
				continue
			}
			if old.Header().InterfaceHash != fresh.Header().InterfaceHash || old.Header().GenericInterfaceHash == fresh.Header().GenericInterfaceHash || old.Header().PackageSource == fresh.Header().PackageSource {
				t.Fatal("unused template edit did not isolate ordinary/generic source identities")
			}
			expectation := after.expectations[fresh.Header().PackageInstance]
			if _, err := atlasmap.NewQueryIndexV2(old, expectation); err == nil {
				t.Fatal("Go accepted stale current-source facts")
			}
			data, err := atlasmap.EncodeV2(old, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			return candidateAtlasV2AdmissionFixture{name: "unused-template-source-drift", data: data, expected: candidateAtlasCurrent(t, expectation)}
		}
	}
	t.Fatal("missing root current-source oracle")
	return candidateAtlasV2AdmissionFixture{}
}

func TestCandidateAtlasV2Current(t *testing.T) {
	fixtures := candidateAtlasV2AdmissionFixtures(t)
	stale := candidateAtlasV2StaleCurrent(t)
	var legacy []candidateAtlasV2AdmissionFixture
	for _, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV3} {
		prepared := candidateAtlasInputs(t, schema)
		for _, p := range prepared.partitions {
			encode := atlasmap.Encode
			if schema == atlasmap.LogicalSchemaV3 {
				encode = atlasmap.EncodeV3
			}
			data, err := encode(p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			legacy = append(legacy, candidateAtlasV2AdmissionFixture{name: fmt.Sprintf("legacy-%s-%s", schema, p.Header().PackageUID), data: data, expected: candidateAtlasCurrent(t, prepared.expectations[p.Header().PackageInstance])})
		}
	}
	main := `module main;import project_atlas_current as current;import project_atlas_validate as validate;` + atlasReadInput + `
@effects(console) fn i32 main(){let data=input();let raw=bytes::view(ref data);var usize at=0;while(at<raw.len && raw[at]!=10){at+=1;}if(at==raw.len){return 7;}
 let source=switch(bytes::copy(raw[..at])){case .ok(p):{p}case .err(_):{return 7;}};let expected=switch(bytes::copy(raw[at+1..])){case .ok(p):{p}case .err(_):{return 7;}};var usize meter=0;var u32 mode=0;
 if(process::arg_count()==2){var u8[64] raw=[0;64];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 7;}};let ref u8[] s=ref raw;if(bytes::equal(s[..n],b"ordinary")){mode=1;}else if(bytes::equal(s[..n],b"v3")){mode=2;}else if(bytes::equal(s[..n],b"budget")){meter=16777216;}else{return 7;}}
 let ready=switch(if(mode==1){current::read(source,expected,ref mut meter)}else if(mode==2){current::read_generics(source,expected,ref mut meter)}else{current::read_v2(source,expected,ref mut meter)}){case .ok(p):{p}case .err(_):{return 7;}};
 let p=current::partition(ref ready);if(validate::version(p)!=2){return 7;}return if(io::write_usize(validate::symbol_count(p)).is_ok){0}else{4};
}`
	candidateAtlasProbe(t, "project_atlas_current", main, func(t *testing.T, image string) {
		calls := 0
		run := func(name string, f candidateAtlasV2AdmissionFixture, expected []byte, ok bool, args ...string) {
			t.Helper()
			calls++
			t.Run(name, func(t *testing.T) {
				input := append(append(append([]byte{}, f.data...), 10), expected...)
				code, output := candidateAtlasProcess(t, image, input, args...)
				if ok {
					if code != 0 || output != strconv.Itoa(f.symbols) {
						t.Fatalf("current v2: %d %q", code, output)
					}
				} else if code != 7 || output != "" {
					t.Fatalf("stale/incompatible v2: %d %q", code, output)
				}
			})
		}
		for _, f := range fixtures {
			run(f.name+"/current", f, f.expected, true)
			for _, field := range []string{"compiler_build", "interface_hash", "generic_interface_hash", "language", "package_uid", "package_instance", "package_coordinate", "package_source", "package_source_kind", "runtime_abi"} {
				mutated := atlasGenericSet(t, f.expected, []string{field}, []byte(`"stale"`))
				run(f.name+"/"+field, f, mutated, false)
			}
			for _, field := range []string{"features", "dependencies", "package_references", "files"} {
				value := `[]`
				if field == "features" || field == "dependencies" {
					value = `["changed"]`
				}
				mutated := atlasGenericSet(t, f.expected, []string{field}, []byte(value))
				if bytes.Equal(mutated, f.expected) {
					t.Fatalf("no-op current mutation %s", field)
				}
				run(f.name+"/"+field, f, mutated, false)
			}
			run(f.name+"/unknown", f, bytes.Replace(f.expected, []byte(`"compiler_build":`), []byte(`"unknown":0,"compiler_build":`), 1), false)
			run(f.name+"/trailing", f, append(append([]byte{}, f.expected...), []byte("{}")...), false)
			for _, mode := range []string{"ordinary", "v3", "budget"} {
				run(f.name+"/"+mode, f, f.expected, false, mode)
			}
		}
		run(stale.name, stale, stale.expected, false)
		for _, f := range legacy {
			run(f.name, f, f.expected, false)
		}
		for _, expected := range [][]byte{nil, []byte("null"), []byte("{}"), []byte(strings.Repeat(" ", 32))} {
			run("invalid-expectation", fixtures[0], expected, false)
		}
		t.Logf("v2 current-source native calls: %d", calls)
	})
}

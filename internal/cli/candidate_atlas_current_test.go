package cli

import (
	"bytes"
	"encoding/json"
	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"testing"
)

// Derived from the verified compiler result, not decoded from the map under test.
func candidateAtlasCurrent(t *testing.T, e atlasmap.LoadExpectation) []byte {
	t.Helper()
	type reference struct {
		UID        string `json:"uid"`
		Instance   string `json:"instance"`
		Coordinate string `json:"coordinate"`
	}
	type file struct {
		Instance string `json:"package_instance"`
		Path     string `json:"path"`
		Digest   string `json:"digest"`
		LineMap  string `json:"line_map"`
		Kind     string `json:"kind"`
	}
	c := e.Current
	g := ""
	if e.GenericInterface != (moduleinterface.GenericHash{}) {
		g = e.GenericInterface.String()
	}
	expected := struct {
		Compiler     string      `json:"compiler_build"`
		Interface    string      `json:"interface_hash"`
		Generic      string      `json:"generic_interface_hash"`
		Language     string      `json:"language"`
		UID          string      `json:"package_uid"`
		Instance     string      `json:"package_instance"`
		Coordinate   string      `json:"package_coordinate"`
		Source       string      `json:"package_source"`
		SourceKind   string      `json:"package_source_kind"`
		Runtime      string      `json:"runtime_abi"`
		Features     []string    `json:"features"`
		Dependencies []string    `json:"dependencies"`
		References   []reference `json:"package_references"`
		Files        []file      `json:"files"`
	}{e.CompilerBuild.String(), e.PublicInterface.String(), g, c.Language, c.UID, c.Instance.String(), c.Coordinate.String(), c.Source.String(), c.SourceKind, c.RuntimeABI, append([]string{}, c.Features...), append([]string{}, c.Dependencies...), []reference{}, []file{}}
	for _, r := range c.PackageReferences {
		expected.References = append(expected.References, reference{r.UID, r.Instance, r.Coordinate})
	}
	for _, f := range c.Files {
		expected.Files = append(expected.Files, file{c.Instance.String(), f.Path, f.Digest.String(), f.LineMap.String(), string(f.Kind)})
	}
	out, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const atlasCurrentMain = `module main;import project_atlas_current as current;` + atlasReadInput + `
 @effects(console) fn i32 main(){let data=input();let raw=bytes::view(ref data);var usize at=0;while(at<raw.len && raw[at]!=10){at+=1;}if(at==raw.len){return 7;}let source=switch(bytes::copy(raw[..at])){case .ok(p):{p}case .err(_):{return 7;}};let expected=switch(bytes::copy(raw[at+1..])){case .ok(p):{p}case .err(_):{return 7;}};var usize meter=0;return switch(current::read(source,expected,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_u32(code);7}};}`

func TestCandidateAtlasCurrent(t *testing.T) {
	type fixture struct{ data, current []byte }
	var fixtures []fixture
	for _, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV3} {
		p := candidateAtlasInputs(t, schema)
		for _, partition := range p.partitions {
			encode := atlasmap.Encode
			if schema == atlasmap.LogicalSchemaV3 {
				encode = atlasmap.EncodeV3
			}
			data, err := encode(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, fixture{data, candidateAtlasCurrent(t, p.expectations[partition.Header().PackageInstance])})
		}
	}
	candidateAtlasProbe(t, "project_atlas_current", atlasCurrentMain, func(t *testing.T, image string) {
		for _, f := range fixtures {
			run := func(expected []byte, want int) {
				t.Helper()
				input := append(append(append([]byte{}, f.data...), 10), expected...)
				code, out := candidateAtlasProcess(t, image, input)
				if code != want || code != 0 && out != "8" {
					t.Fatalf("current source code=%d expected=%d output=%q", code, want, out)
				}
			}
			run(f.current, 0)
			for _, name := range []string{"compiler_build", "interface_hash", "generic_interface_hash", "language", "package_uid", "package_instance", "package_coordinate", "package_source", "package_source_kind", "runtime_abi"} {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(f.current, &fields); err != nil {
					t.Fatal(err)
				}
				old := []byte(`"` + name + `":` + string(fields[name]))
				mutated := bytes.Replace(f.current, old, []byte(`"`+name+`":"stale"`), 1)
				run(mutated, 7)
			}
			run(bytes.Replace(f.current, []byte(`"files":[`), []byte(`"files":[{"unknown":true},`), 1), 7)
			run(bytes.Replace(f.current, []byte(`"dependencies":[]`), []byte(`"dependencies":["untrusted"]`), 1), func() int {
				if bytes.Contains(f.current, []byte(`"dependencies":[]`)) {
					return 7
				}
				return 0
			}())
			run(bytes.Replace(f.current, []byte(`"compiler_build":`), []byte(`"unknown":0,"compiler_build":`), 1), 7)
		}
	})
}

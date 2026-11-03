package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func candidateAtlasInputs(t *testing.T, schema string) preparedAtlasWorkspace {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	var diagnostic bytes.Buffer
	p, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, schema)
	if code != 0 {
		t.Fatalf("prepare Atlas oracle: %d %s", code, &diagnostic)
	}
	return p
}
func candidateAtlasProbe(t *testing.T, module, main string, verify func(*testing.T, string)) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows Atlas native probe; Linux execution is batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, module)
	writeTestSource(t, root, "src/main.fy", main)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 128<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, module+"-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			verify(t, path)
		})
	}
}
func candidateAtlasProcess(t *testing.T, image string, input []byte, args ...string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Stdin = bytes.NewReader(input)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected probe stderr: %s", &stderr)
	}
	return code, out.String()
}
func atlasRechecksum(t *testing.T, data []byte) []byte {
	t.Helper()
	var e struct {
		Schema    string          `json:"schema"`
		Encoding  string          `json:"encoding"`
		Algorithm string          `json:"checksum_algorithm"`
		Checksum  string          `json:"checksum"`
		Map       json.RawMessage `json:"map"`
	}
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	h.Write([]byte("fairyc.fymap-checksum\x00\x01"))
	for _, s := range [][]byte{[]byte(e.Schema), []byte(e.Encoding), e.Map} {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(s)))
		h.Write(n[:])
		h.Write(s)
	}
	e.Checksum = "fymap1-sha256-" + hex.EncodeToString(h.Sum(nil))
	out, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const atlasReadInput = `import std::bytes as bytes;import std::io as io;import std::process as process;
 @effects(console) fn bytes::Buffer input(){var bytes::Buffer data=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return bytes::empty();}};if(n==0){break;}let ref u8[] raw=ref chunk;if(!bytes::append(ref mut data,raw[..n]).is_ok || bytes::len(ref data)>4194304){return bytes::empty();}}return data;}
`

func TestCandidateAtlasWire(t *testing.T) {
	type fixture struct {
		name string
		data []byte
		ok   bool
	}
	var cases []fixture
	for _, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV3} {
		p := candidateAtlasInputs(t, schema)
		for i, partition := range p.partitions {
			encode := atlasmap.Encode
			if schema == atlasmap.LogicalSchemaV3 {
				encode = atlasmap.EncodeV3
			}
			data, err := encode(partition, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			cases = append(cases, fixture{fmt.Sprintf("%s-%d", schema, i), data, true})
			if i != 0 {
				continue
			}
			for name, mutated := range map[string][]byte{
				"whitespace": append([]byte(" "), data...), "trailing": append(append([]byte{}, data...), []byte("{}")...), "truncated": data[:len(data)-1],
				"duplicate":           bytes.Replace(data, []byte(`"schema":`), []byte(`"schema":"ignored","schema":`), 1),
				"unknown":             atlasRechecksum(t, bytes.Replace(data, []byte(`"header":{`), []byte(`"header":{"untrusted":0,`), 1)),
				"checksum":            bytes.Replace(data, []byte(`"package_uid":"example`), []byte(`"package_uid":"altered`), 1),
				"encoding":            atlasRechecksum(t, bytes.Replace(data, []byte(`fairyc.fymap-json/`), []byte(`unknown.fymap-json/`), 1)),
				"noncanonical-string": bytes.Replace(data, []byte(`"schema"`), []byte(`"\u0073chema"`), 1),
			} {
				cases = append(cases, fixture{name, mutated, false})
			}
		}
	}
	main := `module main;import project_atlas_wire as wire;` + atlasReadInput + `@effects(console) fn i32 main(){var usize meter=0;let result=wire::read(input(),if(process::arg_count()==1){4194304}else{1},ref mut meter);return switch(result){case .ok(d):{discard io::write_usize(wire::header(ref d));0}case .err(p):{discard io::write_u32(p.code);7}};}`
	candidateAtlasProbe(t, "project_atlas_wire", main, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, c.data)
				if (code == 0) != c.ok || code != 0 && code != 7 {
					t.Fatalf("accepted=%v code=%d output=%q bytes=%d", c.ok, code, out, len(c.data))
				}
			})
		}
		if code, _ := candidateAtlasProcess(t, image, cases[0].data, "tiny"); code != 7 {
			t.Fatal("byte bound ignored")
		}
	})
}

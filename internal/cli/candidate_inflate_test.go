package cli

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCandidateInflate(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native probe; Linux remains batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_inflate")
	writeTestSource(t, root, "src/main.fy", `module main;import project_inflate as inflate;import std::bytes as bytes;import std::io as io;
 @effects(console) fn i32 main(){var bytes::Buffer input=bytes::empty();var u8[4096] scratch=[0;4096];while(true){let usize n=switch(io::read_stdin(ref mut scratch)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] view=ref scratch;if(!bytes::append(ref mut input,view[..n]).is_ok || bytes::len(ref input)>16777220){return 2;}}
 let ref u8[] data=bytes::view(ref input);if(data.len<4){return 7;}let usize expected=(data[0] as usize)+(data[1] as usize)*256+(data[2] as usize)*65536+(data[3] as usize)*16777216;
 let output=switch(inflate::decode(data[4..],expected)){case .ok(value):{value}case .err(_):{return 7;}};return if(io::write_bytes(bytes::view(ref output)).is_ok){0}else{4};}`)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 8<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	type fixture struct {
		name            string
		compressed, raw []byte
		size            uint32
		bad             bool
	}
	var fixtures []fixture
	for _, level := range []int{flate.NoCompression, flate.BestSpeed, flate.DefaultCompression, flate.BestCompression, flate.HuffmanOnly} {
		for i, raw := range [][]byte{nil, []byte("hello Fairy"), bytes.Repeat([]byte("abcabcabcz"), 8000), bytes.Repeat([]byte{0, 255, 1, 2, 3, 4, 5, 6, 7}, 200)} {
			var packed bytes.Buffer
			w, err := flate.NewWriter(&packed, level)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.Write(raw); err != nil {
				t.Fatal(err)
			}
			if err = w.Close(); err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, fixture{fmt.Sprintf("level%d-%d", level, i), bytes.Clone(packed.Bytes()), raw, uint32(len(raw)), false})
		}
	}
	fixtures = append(fixtures, fixture{name: "reserved", compressed: []byte{7}, bad: true}, fixture{name: "truncated", compressed: []byte{1}, bad: true}, fixture{name: "over-limit", size: 8388609, bad: true}, fixture{name: "bad-stored-complement", compressed: []byte{1, 0, 0, 0, 0}, bad: true}, fixture{name: "wrong-size", compressed: fixtures[1].compressed, size: 2, bad: true})
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "inflate-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, f := range fixtures {
				t.Run(f.name, func(t *testing.T) {
					data := make([]byte, 4)
					binary.LittleEndian.PutUint32(data, f.size)
					data = append(data, f.compressed...)
					cmd := exec.CommandContext(t.Context(), path)
					cmd.Stdin = bytes.NewReader(data)
					var out, stderr bytes.Buffer
					cmd.Stdout, cmd.Stderr = &out, &stderr
					err := cmd.Run()
					code := 0
					if err != nil {
						if exit, ok := err.(*exec.ExitError); ok {
							code = exit.ExitCode()
						} else {
							t.Fatal(err)
						}
					}
					want := 0
					if f.bad {
						want = 7
					}
					if code != want || stderr.Len() != 0 || (!f.bad && !bytes.Equal(out.Bytes(), f.raw)) {
						t.Fatalf("exit=%d want=%d stderr=%s output bytes=%d expected=%d", code, want, &stderr, out.Len(), len(f.raw))
					}
				})
			}
		})
	}
}

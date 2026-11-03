package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/0disoft/fairyc/internal/cdriver"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const remoteLibraryManifest = "schema='fairyc.manifest/v1'\n[package]\nuid='example.math'\nname='math'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n"

type remoteArchiveEntry struct {
	name string
	data []byte
	mode os.FileMode
}

func remoteArchiveBytes(t *testing.T, method uint16, entries ...remoteArchiveEntry) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, entry := range entries {
		h := &zip.FileHeader{Name: entry.name, Method: method}
		if entry.mode != 0 {
			h.SetMode(entry.mode)
		}
		file, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func remoteArchiveValid(t *testing.T, method uint16) []byte {
	return remoteArchiveBytes(t, method, remoteArchiveEntry{name: "fairy.toml", data: []byte(remoteLibraryManifest)}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;pub fn i32 answer(){return 42;}\n")}, remoteArchiveEntry{name: ".review", data: []byte{0, 255, 1, 2}})
}

func TestCandidateRemoteArchive(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native archive replay; Linux remains batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_remote_archive", "sha256_stream")
	writeTestSource(t, root, "src/main.fy", `module main;import project_remote_archive as archive;import sha256_stream as hash;import std::bytes as bytes;import std::io as io;
 @effects(console) fn i32 main(){var bytes::Buffer input=bytes::empty();var u8[4096] scratch=[0;4096];while(true){let usize n=switch(io::read_stdin(ref mut scratch)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] data=ref scratch;if(!bytes::append(ref mut input,data[..n]).is_ok || bytes::len(ref input)>16777216){return 7;}}
 let b=switch(archive::unpack(bytes::view(ref input),b"example.math",b"1.0.0",b"fairyc-language/20",67108864,4096)){case .ok(value):{value}case .err(_):{return 7;}};var bytes::Buffer ledger=bytes::empty();var usize i=0;while(i<archive::count(ref b)){if(!archive::directory(ref b,i)){var hash::State state=hash::empty();assert(hash::update(ref mut state,archive::contents(ref b,i)));assert(bytes::append(ref mut ledger,archive::name(ref b,i)).is_ok && bytes::append_byte(ref mut ledger,10).is_ok && hash::append_hex(ref state,ref mut ledger) && bytes::append_byte(ref mut ledger,10).is_ok);}i+=1;}return if(io::write_bytes(bytes::view(ref ledger)).is_ok){0}else{4};}`)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 64<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	type fixture struct {
		name string
		data []byte
		bad  bool
	}
	fixtures := []fixture{{"stored", remoteArchiveValid(t, zip.Store), false}, {"deflated", remoteArchiveValid(t, zip.Deflate), false}, {"truncated", []byte("PK"), true}}
	base := []remoteArchiveEntry{{name: "fairy.toml", data: []byte(remoteLibraryManifest)}, {name: "src/math.fy", data: []byte("module math;")}}
	for _, name := range []string{"../outside", "/outside", "src/../outside", "src\\evil", "src/con", ".git/config", ".HG/x", "src/A.fy", "src/math.fy"} {
		entries := append([]remoteArchiveEntry(nil), base...)
		entries = append(entries, remoteArchiveEntry{name: name, data: []byte("bad")})
		if name == "src/A.fy" {
			entries = append(entries, remoteArchiveEntry{name: "src/a.fy"})
		}
		fixtures = append(fixtures, fixture{name, remoteArchiveBytes(t, zip.Deflate, entries...), true})
	}
	for name, manifest := range map[string]string{"uid": strings.Replace(remoteLibraryManifest, "example.math", "example.other", 1), "version": strings.Replace(remoteLibraryManifest, "1.0.0", "2.0.0", 1), "language": strings.Replace(remoteLibraryManifest, "language/20", "language/19", 1), "workspace": remoteLibraryManifest + "[workspace]\nmembers=[]\n"} {
		fixtures = append(fixtures, fixture{name, remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(manifest)}, base[1]), true})
	}
	fixtures = append(fixtures, fixture{"symlink", remoteArchiveBytes(t, zip.Deflate, append(append([]remoteArchiveEntry(nil), base...), remoteArchiveEntry{name: "link", data: []byte("outside"), mode: os.ModeSymlink | 0777})...), true}, fixture{"file-directory", remoteArchiveBytes(t, zip.Store, append(append([]remoteArchiveEntry(nil), base...), remoteArchiveEntry{name: "src", data: []byte("x")})...), true}, fixture{"empty-dirs", remoteArchiveBytes(t, zip.Deflate, append(append([]remoteArchiveEntry(nil), base...), remoteArchiveEntry{name: "empty/", mode: os.ModeDir | 0755})...), false})
	crc := bytes.Clone(fixtures[0].data)
	first := bytes.Index(crc, []byte(remoteLibraryManifest))
	if first < 0 {
		t.Fatal("stored payload missing")
	}
	crc[first] ^= 1
	fixtures = append(fixtures, fixture{"crc", crc, true})
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "archive-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, f := range fixtures {
				t.Run(f.name, func(t *testing.T) {
					cmd := exec.CommandContext(t.Context(), path)
					cmd.Stdin = bytes.NewReader(f.data)
					out, err := cmd.CombinedOutput()
					code := 0
					if err != nil {
						if e, ok := err.(*exec.ExitError); ok {
							code = e.ExitCode()
						} else {
							t.Fatal(err)
						}
					}
					if f.bad {
						if code != 7 || len(out) != 0 {
							t.Fatalf("rejection: exit=%d output=%s", code, out)
						}
						return
					}
					if code != 0 {
						t.Fatalf("exit=%d output=%s", code, out)
					}
					reader, err := zip.NewReader(bytes.NewReader(f.data), int64(len(f.data)))
					if err != nil {
						t.Fatal(err)
					}
					var want strings.Builder
					for _, file := range reader.File {
						if file.FileInfo().IsDir() {
							continue
						}
						contents, err := file.Open()
						if err != nil {
							t.Fatal(err)
						}
						var data bytes.Buffer
						if _, err = data.ReadFrom(contents); err != nil {
							t.Fatal(err)
						}
						if err = contents.Close(); err != nil {
							t.Fatal(err)
						}
						sum := sha256.Sum256(data.Bytes())
						fmt.Fprintf(&want, "%s\n%s\n", file.Name, hex.EncodeToString(sum[:]))
					}
					if string(out) != want.String() {
						t.Fatalf("ledger=%q want=%q", out, want.String())
					}
				})
			}
		})
	}
}

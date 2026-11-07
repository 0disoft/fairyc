package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Keep transport policy separate from active bindings: detached artifacts still
// admit >8192 import rows. Only source-index storage follows source_files().
func TestCandidateSourceArtifactBounds(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows source artifact native probe; Linux execution is separate")
	}
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_project_artifact_current", "module_source_provenance", "module_source_artifact_encode", "module_source_artifact_decode", "module_source_artifact_rebind", "file_source")
	for _, name := range []string{"module_source_provenance", "module_source_artifact_model", "module_source_artifact_decode", "module_source_artifact_rebind", "module_project_artifact_current"} {
		data := read(filepath.Join(root, "src", name+".fy"))
		t.Logf("production source %s.fy sha256=%x", name, sha256.Sum256(data))
	}
	tests := filepath.Join("..", "..", "examples", "lexer_candidate", "tests")
	fixture := strings.ReplaceAll(string(read(filepath.Join(tests, "module_source_artifact_fixture.fy"))), "tests::module_source_artifact_fixture", "artifact_fixture")
	writeTestSource(t, root, "src/artifact_fixture.fy", fixture)
	regression := strings.ReplaceAll(string(read(filepath.Join(tests, "module_source_artifact.fy"))), "tests::module_source_artifact_fixture", "artifact_fixture")
	regression = strings.ReplaceAll(regression, "tests::module_source_artifact", "artifact_regression")
	regression = strings.ReplaceAll(regression, "@test ", "")
	regression = strings.ReplaceAll(regression, "fn void source_", "pub fn void source_")
	writeTestSource(t, root, "src/artifact_regression.fy", regression)
	// Append only in the temporary copy: exercise the actual private production
	// map without changing its visibility or duplicating its implementation.
	mapping := string(read(filepath.Join(root, "src", "module_project_artifact_current.fy"))) + `
pub fn void artifact_mapping_probe(ref stamps::Snapshot saved,ref bindings::Bindings current){var usize work=0;let result=switch(map(saved,current,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(result.count==1024 && result.owners.slots[0]==1 && result.owners.slots[511]==0);var usize i=0;while(i<result.count){assert(module_slot(ref result,i)==1024-i);i+=1;}}
`
	writeTestSource(t, root, "src/module_project_artifact_current.fy", mapping)
	dataRoot := filepath.Join("testdata", "source_artifact_bounds")
	main := read(filepath.Join(dataRoot, "main.fy"))
	t.Logf("boundary fixture sha256=%x", sha256.Sum256(main))
	writeTestSource(t, root, "src/main.fy", string(main))
	golden := read(filepath.Join(dataRoot, "legacy-golden.bin"))
	if got := fmt.Sprintf("%x", sha256.Sum256(golden)); got != "b2056ad173df9f45483038c4b78aeeef12c959ac227aa43d905b8e9434913f70" {
		t.Fatalf("independent FYPROV00 golden changed: %s", got)
	}
	goldenPath := filepath.Join(root, "legacy-golden.bin")
	if err := os.WriteFile(goldenPath, golden, 0600); err != nil {
		t.Fatal(err)
	}
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 256<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	owner512 := bytes.Clone(golden)
	binary.BigEndian.PutUint64(owner512[16:24], 512)
	cases := []struct {
		name  string
		input []byte
		want  string
		code  int
	}{
		{"imports-8192", sourceArtifactImportWire(8192), "64/8192", 0},
		{"imports-8193", sourceArtifactImportWire(8193), "64/8193", 0},
		{"sources-1025", sourceArtifactCountWire(1025, 0), "1002", 7},
		{"imports-32769", sourceArtifactCountWire(0, 32769), "1002", 7},
		{"owner-512", owner512, "1001", 7},
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			imagePath := filepath.Join(root, "artifact-"+profile.Name+".exe")
			if err := os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s PE sha256=%x; C partitions=%d", profile.Name, sha256.Sum256(image.Bytes), len(build.Sources))
			out, code := runSourceArtifactBounds(t, imagePath, nil, goldenPath)
			if code != 0 || out != "legacy golden/work/reorder/stale/forged ok\n512 sources ok\n513 sources ok\n1024 sources ok\n1025 refused\n" {
				t.Fatalf("source artifact regression: exit=%d output=%q", code, out)
			}
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					t.Logf("wire bytes=%d sha256=%x", len(c.input), sha256.Sum256(c.input))
					out, code := runSourceArtifactBounds(t, imagePath, c.input, goldenPath, "decode")
					if out != c.want || code != c.code {
						t.Fatalf("want exit=%d output=%q; got exit=%d output=%q", c.code, c.want, code, out)
					}
				})
			}
		})
	}
}

func runSourceArtifactBounds(t *testing.T, image string, input []byte, args ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, image, args...)
	cmd.Stdin = bytes.NewReader(input)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	started := time.Now()
	err := cmd.Run() // CommandContext kills only this child; Run also waits for it.
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if ctx.Err() != nil || stderr.Len() != 0 {
		t.Fatalf("native context=%v stderr=%q exit=%d", ctx.Err(), stderr.String(), code)
	}
	t.Logf("native duration=%s exit=%d output=%q", time.Since(started), code, out.String())
	return out.String(), code
}

func sourceArtifactNumbers(out []byte, values ...uint64) []byte {
	for _, n := range values {
		out = binary.BigEndian.AppendUint64(out, n)
	}
	return out
}

// Independent big-endian framing. Oversized counts have enough payload bytes
// to distinguish a policy refusal from a truncated-record refusal.
func sourceArtifactCountWire(sources, imports int) []byte {
	out := sourceArtifactNumbers([]byte("FYPROV00"), uint64(sources))
	out = append(out, make([]byte, sources*128)...)
	out = sourceArtifactNumbers(out, uint64(imports))
	out = append(out, make([]byte, imports*56)...)
	return sourceArtifactNumbers(out, 0)
}

func sourceArtifactImportWire(count int) []byte {
	var names []byte
	retain := func(data string) (uint64, uint64) {
		start := uint64(len(names))
		names = append(names, data...)
		return start, uint64(len(names))
	}
	ns := "package_coordinate_v1_" + strings.Repeat("1", 64)
	out := sourceArtifactNumbers([]byte("FYPROV00"), 64)
	for i := 0; i < 64; i++ {
		a, b := retain(ns)
		c, d := retain(fmt.Sprintf("m%d", i))
		first, rows := i*128, 128
		if i == 63 {
			rows = count - first
		}
		out = sourceArtifactNumbers(out, 0, a, b, c, d, uint64(first), uint64(rows), 0, 0, 0, 0, 0, 0, 0, 0, 0)
	}
	out = sourceArtifactNumbers(out, uint64(count))
	for i := 0; i < count; i++ {
		module := min(i/128, 63)
		local := i - module*128
		alias := fmt.Sprintf("a%04d", local)
		a, b := retain(alias)
		c, d := retain(alias + "=" + ns + "::m0")
		out = sourceArtifactNumbers(out, uint64(module), uint64(local+2), 0, a, b, c, d)
	}
	out = sourceArtifactNumbers(out, uint64(len(names)))
	return append(out, names...)
}

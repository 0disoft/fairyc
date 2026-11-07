package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Encode the fixture independently of the Fairy encoder, including over-limit
// data whose payload is otherwise structurally valid. Counts are literal test
// cases, not derived from candidate_limits or a textual source replacement.
func projectLayoutLimitGolden(count uint64) []byte {
	var out bytes.Buffer
	out.WriteString("FYMETA00")
	number := func(n uint64) { _ = binary.Write(&out, binary.BigEndian, n) }
	number(0) // legacy authority-free module context
	number(count + 1)
	number(count)
	for i := range count {
		for _, n := range []uint64{i + 2, 0, i + 1, i, 1, 7, 0, 0, 1, 0, 0, 0, 1} {
			number(n)
		}
	}
	number(count)
	for i := range count {
		for _, n := range []uint64{1, 1, 0, i + 1, 1, 0, 1, 1, 2, 0, 0} {
			number(n)
		}
	}
	number(0) // callables
	number(0) // parameters
	number(2)
	out.WriteString("Sv")
	return out.Bytes()
}

func TestCandidateProjectLayoutLimit(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows low/optimized native layout admission; Linux execution is batched")
	}
	var input bytes.Buffer
	input.WriteString("FYLIM001")
	number := func(n uint64) { _ = binary.Write(&input, binary.BigEndian, n) }
	number(5)
	for _, count := range []uint64{1024, 1116, 2047, 2048, 2049} {
		raw := projectLayoutLimitGolden(count)
		number(count)
		number(uint64(len(raw)))
		input.Write(raw)
		t.Logf("independent fixture layouts=%d bytes=%d sha256=%x", count, len(raw), sha256.Sum256(raw))
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_project_metadata", "module_metadata_artifact_encode", "module_metadata_artifact_decode")
	fixture, err := os.ReadFile(filepath.Join("testdata", "project_layout_limit", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/main.fy", string(fixture))
	for _, name := range []string{"candidate_limits", "module_project_metadata", "module_unit_body_ir_verify", "module_metadata_artifact_model", "module_metadata_artifact_decode"} {
		raw, err := os.ReadFile(filepath.Join(root, "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("production %s sha256=%x", name, sha256.Sum256(raw))
	}
	// Open constructors only in disposable fixture copies, preserving every
	// admission/verifier/encoder implementation and the production source files.
	for _, splice := range []struct{ module, fixture string }{
		{"module_metadata_artifact_model", "metadata_constructor.fy"},
		{"module_types", "types_constructor.fy"},
		{"module_project_types", "graph_constructor.fy"},
	} {
		path := filepath.Join(root, "src", splice.module+".fy")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		extra, err := os.ReadFile(filepath.Join("testdata", "project_layout_limit", splice.fixture))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, "src/"+splice.module+".fy", string(raw)+"\n"+string(extra))
	}
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 256<<10)
	t.Logf("C partitions=%d FY fixture sha256=%x", len(build.Sources), sha256.Sum256(fixture))
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const expected = "layouts=1024 admitted exact-roundtrip merge=exact\n" +
		"layouts=1116 admitted exact-roundtrip merge=exact\n" +
		"layouts=2047 admitted exact-roundtrip merge=exact\n" +
		"layouts=2048 admitted exact-roundtrip merge=exact\n" +
		"layouts=2049 refused=1002 merge=refused\n"
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, 3*time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "layout-limit-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, path)
			cmd.Stdin = bytes.NewReader(input.Bytes())
			out, err := cmd.CombinedOutput()
			if err != nil || strings.ReplaceAll(string(out), "\r\n", "\n") != expected {
				t.Fatalf("layout admission: %v output=%s", err, out)
			}
			t.Logf("PE sha256=%x\n%s", sha256.Sum256(image.Bytes), out)
		})
	}
}

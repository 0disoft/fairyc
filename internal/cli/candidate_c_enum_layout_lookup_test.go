package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestCandidateCEnumLayoutLookup(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows enum/switch lookup regression; Linux execution is batched")
	}
	read := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	inputs := map[string][32]byte{}
	for _, name := range []string{"module_unit_c_layout_index", "module_unit_c_sdk", "module_unit_c_enums", "module_unit_c_switches", "module_unit_c_function"} {
		path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy")
		inputs[path] = sha256.Sum256(read(path))
		t.Logf("%s sha256=%x", name, inputs[path])
	}
	fixtureDir := filepath.Join("testdata", "c_layout_lookup")
	for _, name := range []string{"main.fy", "cleanup_constructor.fy", "body_constructor.fy"} {
		path := filepath.Join(fixtureDir, name)
		inputs[path] = sha256.Sum256(read(path))
	}
	unchanged := func() {
		t.Helper()
		for path, want := range inputs {
			if sha256.Sum256(read(path)) != want {
				t.Fatalf("layout lookup input changed: %s", path)
			}
		}
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	// Include the production body caller so its new shared-plan call sites are
	// checked with the fixture, even though the raw malformed IR is never sent
	// through a public whole-program verifier or presented as checked input.
	copyCandidateParityModules(t, root, "module_unit_c_function")
	for _, splice := range []struct{ module, fixture string }{
		{"module_unit_cleanup_mir_model", "cleanup_constructor.fy"},
		{"module_unit_body_ir_model", "body_constructor.fy"},
	} {
		path := filepath.Join(root, "src", splice.module+".fy")
		writeTestSource(t, root, "src/"+splice.module+".fy", string(read(path))+"\n"+string(read(filepath.Join(fixtureDir, splice.fixture))))
	}
	writeTestSource(t, root, "src/main.fy", string(read(filepath.Join(fixtureDir, "main.fy"))))
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 256<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const expected = "layouts=2048 scan=4194304 index=2048 sdk=8198\n" +
		"enum=equal switch=equal work=2050->3 malformed=1001 budget=1002\n"
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			unchanged()
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "c-layout-lookup-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			var stderr bytes.Buffer
			cmd := exec.CommandContext(ctx, path)
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil || string(out) != expected || stderr.Len() != 0 {
				t.Fatalf("layout lookup: %v stdout=%s stderr=%s", err, out, &stderr)
			}
			t.Logf("PE sha256=%x\n%s", sha256.Sum256(image.Bytes), out)
			unchanged()
		})
	}
}

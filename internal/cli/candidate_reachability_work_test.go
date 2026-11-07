package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestCandidateReachabilityWork(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows reachability low/optimized native probe")
	}
	// Reuse the existing parity build/Windows execution harness. Test-only
	// constructors live in temporary model copies; the actual traversal, CFG
	// assembler, validation and indexes execute unchanged production functions.
	inputs := map[string][32]byte{}
	read := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for _, name := range []string{"module_unit_cfg_model", "module_unit_c_reachability", "module_unit_body_ir_model", "module_unit_cleanup_mir_model"} {
		path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy")
		inputs[path] = sha256.Sum256(read(path))
		t.Logf("production %s sha256=%x", name, inputs[path])
	}
	fixturePath := filepath.Join("testdata", "reachability_work", "main.fy")
	main := read(fixturePath)
	inputs[fixturePath] = sha256.Sum256(main)
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_unit_c_reachability")
	for _, fixture := range []struct{ module, file string }{
		{"module_unit_body_ir_model", "body_ir_fixture.fy"},
		{"module_unit_cleanup_mir_model", "cleanup_ir_fixture.fy"},
	} {
		path := filepath.Join("testdata", "reachability_work", fixture.file)
		data := read(path)
		inputs[path] = sha256.Sum256(data)
		target := filepath.Join(root, "src", fixture.module+".fy")
		writeTestSource(t, root, "src/"+fixture.module+".fy", string(read(target))+"\n"+string(data))
	}
	writeTestSource(t, root, "src/main.fy", string(main))
	unchanged := func() {
		t.Helper()
		for path, want := range inputs {
			if sha256.Sum256(read(path)) != want {
				t.Fatalf("reachability probe input changed: %s", path)
			}
		}
	}
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 256<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			unchanged()
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			imagePath := filepath.Join(root, "reachability-"+profile.Name+".exe")
			if err := os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			t.Logf("native PE sha256=%x", sha256.Sum256(image.Bytes))
			code, out := candidateAtlasProcess(t, imagePath, nil)
			lines := regexp.MustCompile(`\A(?:graph copies=1 work=[0-9]+\ngraph copies=16 work=[0-9]+\ngraph copies=128 work=[0-9]+\nforeign copies=128 work=[0-9]+\n)summary=ordered set=exact budget=boundary rejection=1001,1002 inputs=reused\n\z`)
			if code != 0 || !lines.MatchString(out) {
				t.Fatalf("reachability set/CFG summary/work: exit=%d output=%q", code, out)
			}
			for _, match := range regexp.MustCompile(`work=([0-9]+)`).FindAllStringSubmatch(out, -1) {
				work, err := strconv.Atoi(match[1])
				if err != nil || work <= 0 || work > 1048576 {
					t.Fatalf("invalid measured work %q: %v", match[1], err)
				}
			}
			unchanged()
			t.Log(out)
		})
	}
	unchanged()
}

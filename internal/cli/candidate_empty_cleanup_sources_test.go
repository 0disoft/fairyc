package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestCandidateEmptyCleanupSources(t *testing.T) {
	inputs := map[string][32]byte{}
	for _, name := range []string{"scanner", "module_unit_bodies"} {
		path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		inputs[path] = sha256.Sum256(data)
		t.Logf("production %s sha256=%x", name, inputs[path])
	}
	main, err := os.ReadFile(filepath.Join("testdata", "empty_cleanup_sources", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("fixture sha256=%x", sha256.Sum256(main))
	candidateAtlasProbeSize(t, "module_unit_bodies", string(main), 256<<10, func(t *testing.T, image string) {
		for path, want := range inputs {
			data, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(data) != want {
				t.Fatalf("production input changed during probe: %s", path)
			}
		}
		data, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("native PE sha256=%x", sha256.Sum256(data))
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 || !regexp.MustCompile(`^case=1 original=\d+ cloned=0 bytes=\d+ cooked=0\ncase=2 original=\d+ cloned=\d+ bytes=\d+ cooked=\d+\ncase=3 original=\d+ cloned=\d+ bytes=\d+ cooked=\d+\ncase=4 original=\d+ cloned=\d+ bytes=\d+ cooked=\d+\nok\n$`).MatchString(out) {
			t.Fatalf("empty cleanup source ownership/fallback: exit=%d output=%q", code, out)
		}
		t.Logf("real builders after original source removal: %s", out)
	}, "module_unit_names")
}

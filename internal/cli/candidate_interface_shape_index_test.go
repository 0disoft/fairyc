package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestCandidateInterfaceShapeIndex(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_unit_interfaces.fy")
	production, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantSource := sha256.Sum256(production)
	t.Logf("production Interfaces sha256=%x", wantSource)
	main, err := os.ReadFile(filepath.Join("testdata", "interface_shape_index", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("real builder fixture sha256=%x", sha256.Sum256(main))
	candidateAtlasProbeSize(t, "module_unit_interfaces", string(main), 256<<10, func(t *testing.T, image string) {
		current, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(current) != wantSource {
			t.Fatal("production Interfaces changed during the probe")
		}
		pe, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("native PE sha256=%x", sha256.Sum256(pe))
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 {
			t.Fatalf("real Interfaces builder/field_measured: exit=%d output=%q", code, out)
		}
		match := regexp.MustCompile(`^shapes=(\d+) types=(\d+) work=(\d+) indexed=(\d+) linear=(\d+) queries=(\d+)\nok\n$`).FindStringSubmatch(out)
		if match == nil {
			t.Fatalf("unexpected probe output %q", out)
		}
		var n [6]int
		for i := range n {
			n[i], err = strconv.Atoi(match[i+1])
			if err != nil {
				t.Fatal(err)
			}
		}
		if n[0] != 162 || n[1] > 4096 || n[2] > 1048576 || n[3] != 11 || n[4] < 170 || n[5] < 500 {
			t.Fatalf("shape/index policy or coverage differs: %q", out)
		}
		t.Logf("real builder and ordered linear oracle: %s", out)
	})
}

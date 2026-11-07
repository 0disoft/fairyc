package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestCandidateBorrowTypeIndex(t *testing.T) {
	inputs := map[string][32]byte{}
	for _, name := range []string{"module_unit_interfaces", "module_unit_value_place", "module_unit_value_scalar"} {
		path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		inputs[path] = sha256.Sum256(data)
		t.Logf("production %s sha256=%x", name, inputs[path])
	}
	main, err := os.ReadFile(filepath.Join("testdata", "borrow_type_index", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("real builder fixture sha256=%x", sha256.Sum256(main))
	candidateAtlasProbeSize(t, "module_unit_value_place", string(main), 256<<10, func(t *testing.T, image string) {
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
		if code != 0 || !regexp.MustCompile(`^small_types=\d+ large_types=\d+ small_setup=\d+ large_setup=\d+ queries=\d+ fast=1 graph_cap=4096\nok\n$`).MatchString(out) {
			t.Fatalf("borrow index oracle/budget/permission/cap: exit=%d output=%q", code, out)
		}
		t.Logf("real builder, canonical oracle and bounded graph: %s", out)
	}, "module_unit_value_scalar")
}

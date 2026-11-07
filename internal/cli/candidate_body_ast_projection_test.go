package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestCandidateBodyASTProjection(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_unit_body_ast.fy")
	production, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(production)
	t.Logf("production body AST sha256=%x", want)
	main, err := os.ReadFile(filepath.Join("testdata", "body_ast_projection", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual ast::build fixture sha256=%x", sha256.Sum256(main))
	candidateAtlasProbeSize(t, "module_unit_body_ast", string(main), 256<<10, func(t *testing.T, image string) {
		current, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(current) != want {
			t.Fatal("production body AST changed during native build")
		}
		pe, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("native PE sha256=%x", sha256.Sum256(pe))
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 {
			t.Fatalf("authored AST projection: exit=%d output=%q", code, out)
		}
		m := regexp.MustCompile(`^functions=80 arena=(\d+) selected=(\d+) work=(\d+) legacy_lower=(\d+)\ndense=ok cleanup=ok reach=ok\n$`).FindStringSubmatch(out)
		if m == nil {
			t.Fatalf("unexpected AST result %q", out)
		}
		var n [4]int
		for i := range n {
			n[i], err = strconv.Atoi(m[i+1])
			if err != nil {
				t.Fatal(err)
			}
		}
		if n[0] < 8000 || n[1] == 0 || n[2] != 80*n[0]+n[1] || n[2] >= 1048576 || n[3] != 160*n[0] || n[3] <= 1048576 {
			t.Fatalf("shared work/counterexample bounds differ: %q", out)
		}
		t.Logf("80 actual projections and dense/reach/cleanup guards: %s", out)
	})
}

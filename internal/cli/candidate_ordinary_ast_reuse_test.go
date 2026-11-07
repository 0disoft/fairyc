package cli

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestCandidateOrdinaryASTReuse(t *testing.T) {
	production := make(map[string][32]byte)
	for _, name := range []string{"module_unit_ordinary_names.fy", "module_unit_values.fy", "module_unit_body_ast.fy"} {
		path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		production[path] = sha256.Sum256(raw)
		t.Logf("production %s sha256=%x", name, production[path])
	}
	stable := func(t *testing.T) {
		t.Helper()
		for path, want := range production {
			raw, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(raw) != want {
				t.Fatalf("production source changed during regression: %s", path)
			}
		}
	}
	t.Run("work", func(t *testing.T) {
		main, err := os.ReadFile(filepath.Join("testdata", "ordinary_ast_reuse", "main.fy"))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("stress fixture sha256=%x", sha256.Sum256(main))
		var first string
		candidateAtlasProbeSize(t, "module_unit_values", string(main), 256<<10, func(t *testing.T, image string) {
			stable(t)
			pe, err := os.ReadFile(image)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("native PE sha256=%x", sha256.Sum256(pe))
			code, out := candidateAtlasProcess(t, image, nil)
			if code != 0 {
				t.Fatalf("ordinary AST reuse: exit=%d output=%q", code, out)
			}
			row := regexp.MustCompile(`(?m)^(small|large) functions=80 arena=(\d+) selected=(\d+) projection=(\d+) names=(\d+) values=(\d+) expressions=(\d+) locals=(\d+) returns=(\d+)$`)
			matches := row.FindAllStringSubmatch(out, -1)
			if len(matches) != 2 || matches[0][1] != "small" || matches[1][1] != "large" || !strings.HasSuffix(out, "reuse=ok origins=ok links=ok values=ok\n") {
				t.Fatalf("unexpected reuse report: %q", out)
			}
			var rows [2][8]int
			for i, match := range matches {
				for j := range rows[i] {
					rows[i][j], err = strconv.Atoi(match[j+2])
					if err != nil {
						t.Fatal(err)
					}
				}
				r := rows[i]
				if r[2] != 80*r[0]+r[1] || r[2] >= 1048576 || r[3] >= 1048576 || r[4] >= 1048576 || r[6] != 160 || r[7] != 80 {
					t.Fatalf("work/count bounds differ: %q", out)
				}
			}
			small, large := rows[0], rows[1]
			if large[0] <= 8000 || large[1] != small[1] || large[3]-small[3] != 80*(large[0]-small[0]) || large[4] != small[4] || large[5] != small[5] || large[2] <= 800000 || large[4] >= large[2] {
				t.Fatalf("unrelated arena was charged again in Values: %q", out)
			}
			if first != "" && out != first {
				t.Fatalf("profile reports differ: first=%q current=%q", first, out)
			}
			first = out
			stable(t)
			t.Logf("actual ordinary/closure/values work and origins:\n%s", out)
		})
	})
	t.Run("existing-boundaries", func(t *testing.T) {
		path := filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_unit_values.fy")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("existing value regression fixture sha256=%x", sha256.Sum256(raw))
		main := strings.Replace(string(raw), "module tests::module_unit_values;", "module main;", 1)
		main = strings.ReplaceAll(main, "@test", "") + `
@effects(console) fn i32 main(){
 call_results_reach_generic_cleanup_and_ordinary_locals();
 stale_sources_are_rejected_before_value_publication();
 value_admission_keeps_the_active_local_bound();
 return 0;
}
`
		candidateAtlasProbeSize(t, "module_unit_values", main, 256<<10, func(t *testing.T, image string) {
			stable(t)
			if code, out := candidateAtlasProcess(t, image, nil); code != 0 || out != "" {
				t.Fatalf("existing generic/cleanup/stale/128-local checks: exit=%d output=%q", code, out)
			}
			stable(t)
		}, "module_unit_plan")
	})
}

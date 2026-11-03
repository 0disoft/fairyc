package cli

import (
	"strings"
	"testing"
)

func TestCandidateAtlasOptions(t *testing.T) {
	main := `module main;import project_atlas_options as options;import std::io as io;@effects(console) fn i32 main(){return switch(options::arguments()){case .ok(p):{discard io::write_bytes(options::uid(ref p));0}case .err(_):{2}};}`
	symbol := "fysym1-sha256-128-" + strings.Repeat("a", 32)
	valid := impactArgs("example.session.answer", symbol)
	cases := [][]string{valid, append(append([]string{}, valid...), "--locked"), {}, {"--help"}, {"--unknown", "1"}, append(append([]string{}, valid...), "--locked", "--locked"), append(append([]string{}, valid...), "extra"), append(append([]string{}, valid...), "--symbol", symbol)}
	for i := range valid {
		if i%2 == 1 {
			for _, value := range []string{"", "-1", "+1", "01", "184467440737095516160", "--locked", "0", "65537", "4194305"} {
				a := append([]string{}, valid...)
				a[i] = value
				cases = append(cases, a)
			}
		}
	}
	candidateAtlasProbeSize(t, "project_atlas_options", main, 16<<10, func(t *testing.T, image string) {
		for _, args := range cases {
			_, err := parseImpact(args)
			want := 0
			if err != nil {
				want = 2
			}
			code, out := candidateAtlasProcess(t, image, nil, append([]string{"experimental", "atlas-impact"}, args...)...)
			if code != want || code == 0 && out != "example.session.answer" {
				t.Fatalf("%v: code=%d want=%d output=%q", args, code, want, out)
			}
		}
	})
}

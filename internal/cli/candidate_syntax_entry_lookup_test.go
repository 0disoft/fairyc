package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCandidateSyntaxEntryLookup(t *testing.T) {
	main, err := os.ReadFile(filepath.Join("testdata", "syntax_entry_lookup", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	// The shared helper builds and executes both low and optimized native profiles.
	candidateAtlasProbeSize(t, "module_unit_body_ast", string(main), 256<<10, func(t *testing.T, image string) {
		code, out := candidateAtlasProcess(t, image, nil)
		const want = "ordered=4 nested=4 origins=ok work=36,40 limits=ok\nfunctions=24 entries=408 work=37200\nduplicate=first copy=first\nassembled=ordered copy=ordered\n"
		if code != 0 || out != want {
			t.Fatalf("production parser/entry lookup/AST: exit=%d output=%q, want %q", code, out, want)
		}
	})
}

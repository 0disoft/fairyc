package cli

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestCandidateCleanupRecipeFields(t *testing.T) {
	productionPath := filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_unit_cleanup_mir_lower.fy")
	fixturePath := filepath.Join("testdata", "cleanup_recipe_fields", "main.fy")
	read := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	production, fixture := read(productionPath), read(fixturePath)
	productionHash, fixtureHash := sha256.Sum256(production), sha256.Sum256(fixture)
	t.Logf("production cleanup MIR lower sha256=%x fixture sha256=%x", productionHash, fixtureHash)
	unchanged := func(t *testing.T) {
		t.Helper()
		if sha256.Sum256(read(productionPath)) != productionHash || sha256.Sum256(read(fixturePath)) != fixtureHash {
			t.Fatal("production or fixture changed during the native replay")
		}
	}
	candidateAtlasProbeSize(t, "module_project_compile", string(fixture), 256<<10, func(t *testing.T, image string) {
		unchanged(t)
		pe := read(image)
		t.Logf("native PE sha256=%x", sha256.Sum256(pe))
		// This fixture's local execution deadline does not change compiler/runtime
		// policy or the shared helper. CommandContext waits for this child only.
		ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, image).CombinedOutput()
		if err != nil {
			t.Fatalf("real source/signatures/unit cleanup fields: %v output=%s", err, out)
		}
		lines := regexp.MustCompile(`(?m)^fields=(1023|1024|1025|2047|2048) accepted work=(\d+) input=unchanged$`).FindAllSubmatch(out, -1)
		want := []int{1023, 1024, 1025, 2047, 2048}
		if len(lines) != len(want) {
			t.Fatalf("missing field boundaries: %s", out)
		}
		for i, line := range lines {
			count, _ := strconv.Atoi(string(line[1]))
			work, err := strconv.Atoi(string(line[2]))
			if err != nil || count != want[i] || work <= 0 || work > 1048576 {
				t.Fatalf("field/work policy mismatch: %s", out)
			}
		}
		if !regexp.MustCompile(`(?m)^fields=2049 rejected stage=(3 code=10|6 code=1002) input=unchanged\n$`).Match(out) {
			t.Fatalf("2049 must fail at the interface or unit field gate: %s", out)
		}
		unchanged(t)
		t.Logf("actual compile::range, ordered owner/ordinal/type recipes, source hashes: %s", out)
	})
	unchanged(t)
}

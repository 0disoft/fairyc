package prevrevision

import (
	"bytes"
	"testing"
)

func TestMigrateRevisionOneSourceIsIdentity(t *testing.T) {
	t.Parallel()
	input := []byte(`module main;
fn i32 main() {
    var i32 value = 40;
    for (var i32 index = 0; index < 2; index = index + 1) {
        value = value + 1;
    }
    return value;
}
`)
	result, err := Migrate("main.fy", input)
	if err != nil || result.Changed || !bytes.Equal(result.Source, input) {
		t.Fatalf("Migrate() = %+v, %v", result, err)
	}
}

func TestMigrateDeclarationsPreservesRevisionOneTrivia(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\n// preserved\nfn i32 main() { return 42; }\n")
	result, err := MigrateDeclarations("main.fy", input)
	if err != nil || result.Changed || !bytes.Equal(result.Source, input) {
		t.Fatalf("MigrateDeclarations() = %+v, %v", result, err)
	}
}

func TestMigratePreservesRevisionTwoCompoundAssignment(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { var i32 value = 40; value += 2; return value; }\n")
	result, err := Migrate("main.fy", input)
	if err != nil || result.Changed || !bytes.Equal(result.Source, input) {
		t.Fatalf("Migrate() = %+v, %v", result, err)
	}
}

func TestMigrateRejectsRevisionThreeDefer(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn void record(i32 value) {}\nfn i32 main() { defer record(42); return 42; }\n")
	result, err := Migrate("main.fy", input)
	if err == nil {
		t.Fatalf("Migrate() result = %+v, want current-only syntax rejection", result)
	}
	if result.Source != nil || result.Changed {
		t.Fatalf("Migrate() returned writable result on error: %+v", result)
	}
}

func TestSpanMapperKeepsHalfOpenBoundariesAroundInsertions(t *testing.T) {
	t.Parallel()
	edits := []edit{
		{start: 0, end: 0, replacement: "return "},
		{start: 5, end: 5, replacement: ";"},
	}
	start, end, err := mapSpan([]byte("value"), []byte("return value;"), edits, 0, 5)
	if err != nil {
		t.Fatal(err)
	}
	if start != 7 || end != 12 {
		t.Fatalf("mapped span = %d..%d, want 7..12", start, end)
	}
}

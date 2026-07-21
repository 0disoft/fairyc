package prevrevision

import (
	"strings"
	"testing"
)

func TestMigrateDeclarationsPreservesSurroundingTrivia(t *testing.T) {
	t.Parallel()
	input := []byte(`module main;
// declaration comment
struct Pair {
    left: i32;
    data: &mut [u8];
}
fn add(left: i32, right: i32) -> i32 {
    let pair: Pair = Pair { left: left, data: bytes };
    left + right
}
`)
	result, err := MigrateDeclarations("main.fy", input)
	if err != nil {
		t.Fatalf("MigrateDeclarations() error = %v", err)
	}
	wantParts := []string{
		"// declaration comment", "i32 left;", "ref mut u8[] data;",
		"fn i32 add(i32 left, i32 right)", "let Pair pair =", "return left + right;",
	}
	for _, want := range wantParts {
		if !strings.Contains(string(result.Source), want) {
			t.Fatalf("migrated source missing %q:\n%s", want, result.Source)
		}
	}
}

func TestMigrateDeclarationsRejectsMutableParameters(t *testing.T) {
	t.Parallel()
	_, err := MigrateDeclarations("main.fy", []byte("module main;\nfn update(mut value: i32) -> i32 { value }\n"))
	if err == nil || !strings.Contains(err.Error(), "requires manual migration") {
		t.Fatalf("MigrateDeclarations() error = %v", err)
	}
}

func TestMigrateDeclarationsIsIdempotent(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { return 0; }\n")
	result, err := MigrateDeclarations("main.fy", input)
	if err != nil || result.Changed || string(result.Source) != string(input) {
		t.Fatalf("MigrateDeclarations() = %+v, %v", result, err)
	}
}

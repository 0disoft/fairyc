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

func TestMigrateDeclarationsLowersMutableParametersToLocals(t *testing.T) {
	t.Parallel()
	result, err := MigrateDeclarations("main.fy", []byte("module main;\nfn update(mut value: i32) -> i32 { value }\n"))
	if err != nil {
		t.Fatalf("MigrateDeclarations() error = %v", err)
	}
	if !strings.Contains(string(result.Source), "fn i32 update(i32 input) {\n    var i32 value = input; return value; }") {
		t.Fatalf("migrated source = %s", result.Source)
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

func TestMigrateControlRecordsAndVariants(t *testing.T) {
	t.Parallel()
	input := []byte(`module main;
enum Message { Empty; Pair { left: i32; right: i32; }; }
fn choose(value: i32) -> Result<Message, i32> {
    if value > 0 {
        Result::Ok(Message::Pair { left: value, right: 1 })
    } else {
        Result::Err(0)
    }
}
fn score(input: Result<Message, i32>) -> i32 {
    match input {
        Result::Ok(message) => match message {
            Message::Empty => 0,
            Message::Pair { left: first, right: second } => first + second,
        },
        Result::Err(error) => -error,
    }
}
`)
	result, err := Migrate("main.fy", input)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	wantParts := []string{
		"if (value > 0)", ".ok(.Pair { .left = value, .right = 1 })", ".err(0)",
		"switch (input)", "case .ok(message) : { switch (message)",
		"case .Empty : { 0 }", "case .Pair { .left = first, .right = second } : { first + second }",
	}
	for _, want := range wantParts {
		if !strings.Contains(string(result.Source), want) {
			t.Fatalf("migrated source missing %q:\n%s", want, result.Source)
		}
	}
}

func TestMigrateBorrowExpressions(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn borrow(value: i32) -> &i32 { &value }\n")
	result, err := Migrate("main.fy", input)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if !strings.Contains(string(result.Source), "fn ref i32 borrow(i32 value) { return ref value; }") {
		t.Fatalf("migrated source = %s", result.Source)
	}
}

func TestMigrateRejectsNewlyReservedForIdentifier(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn for() -> i32 { 0 }\n")
	result, err := Migrate("main.fy", input)
	if err == nil {
		t.Fatalf("Migrate() result = %+v, want fail-closed reserved-keyword error", result)
	}
	if result.Source != nil || result.Changed {
		t.Fatalf("Migrate() returned writable result on error: %+v", result)
	}
}

func TestMigrateWrapsEntirePartiallyParenthesizedControlHeader(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 { if (false) || true { 42 } else { 0 } }\n")
	result, err := Migrate("main.fy", input)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if !strings.Contains(string(result.Source), "if ((false) || true)") {
		t.Fatalf("migrated source = %s", result.Source)
	}
}

func TestMigrateCleanupReceiver(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nresource Token { value: i32; drop fn(self: &mut Self) { let seen: i32 = self.value; } }\n")
	result, err := Migrate("main.fy", input)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if !strings.Contains(string(result.Source), "cleanup { let i32 seen = this.value; }") {
		t.Fatalf("migrated source = %s", result.Source)
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

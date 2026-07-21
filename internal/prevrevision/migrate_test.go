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

package semantic_test

import "testing"

func TestAnalyzeFieldBorrowAssignmentDoesNotSilentlyDrop(t *testing.T) {
	for name, body := range map[string]string{
		"local":             "fn i32 main() { var State state = State { .count = 0 }; state.count = 1; return state.count; }",
		"mutable reference": "fn void edit(ref mut State state) { state.count = 1; }",
		"compound":          "fn void edit(ref mut State state) { state.count += 1; }",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; struct State { i32 count; } " + body})
			if len(result.Unsupported) == 0 {
				t.Fatal("field assignment accepted without executable storage")
			}
		})
	}
}

func TestAnalyzeSharedFieldBorrow(t *testing.T) {
	for name, body := range map[string]string{
		"direct":   "let ref i32 value = ref owner.inner.number; assert(*value == 42);",
		"borrowed": "let ref Owner view = ref owner; let ref i32 value = ref view.inner.number; assert(*value == 42);",
		"returned": "let ref i32 value = number(ref owner); assert(*value == 42);",
		"last use": "let ref i32 value = ref owner.inner.number; assert(*value == 42); let Owner next = owner;",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": fieldBorrowPrelude + `
fn i32 main() { let Owner owner = make(); ` + body + ` return 0; }`})
			assertCleanTypeResult(t, result)
		})
	}
}

func TestAnalyzeSharedFieldBorrowProtectsRoot(t *testing.T) {
	for name, test := range map[string]struct{ body, code string }{
		"move":      {"let Owner next = owner; assert(*value == 42);", "E2105"},
		"replace":   {"owner = make(); assert(*value == 42);", "E2101"},
		"exclusive": {"edit(ref mut owner); assert(*value == 42);", "E2101"},
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": fieldBorrowPrelude + `
fn void edit(ref mut Owner value) {}
fn i32 main() { var Owner owner = make(); let ref i32 value = number(ref owner); ` + test.body + ` return 0; }`})
			if len(result.Unsupported) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != test.code {
				t.Fatalf("root protection: %+v %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestAnalyzeSharedFieldBorrowRejectsLocalEscape(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": fieldBorrowPrelude + `
fn ref i32 escape() { let Owner owner = make(); return ref owner.inner.number; }`})
	if len(result.Unsupported) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2102" {
		t.Fatalf("local escape: %+v %+v", result.Diagnostics, result.Unsupported)
	}
}

func TestAnalyzeSharedFieldBorrowRejectsOwnedTemporary(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": fieldBorrowPrelude + `
fn ref i32 escape() { return ref make().inner.number; }`})
	if len(result.Unsupported) == 0 {
		t.Fatal("borrow of owned temporary accepted")
	}
}

func TestAnalyzeSharedFieldBorrowKeepsMutableReborrowClosed(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": fieldBorrowPrelude + `
fn ref i32 read_mut(ref mut Owner owner) { return ref owner.inner.number; }`})
	if len(result.Unsupported) == 0 {
		t.Fatal("mutable-owner reborrow admitted without exclusive lifetime support")
	}
}

func TestAnalyzeSharedFieldBorrowKeepsResourceFieldsPrivate(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"provider.fy": `module provider; pub resource Owner { i32 number; cleanup {} }`,
		"main.fy": `module main; import provider as p;
fn ref i32 read(ref p::Owner owner) { return ref owner.number; }`,
	})
	if len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != "E0103" {
		t.Fatalf("private field borrow: %+v %+v", result.Diagnostics, result.Unsupported)
	}
}

const fieldBorrowPrelude = `module main;
struct Inner { i32 number; }
resource Owner { Inner inner; cleanup {} }
fn Owner make() { return Owner { .inner = Inner { .number = 42 } }; }
fn ref i32 number(ref Owner value) { return ref value.inner.number; }
`

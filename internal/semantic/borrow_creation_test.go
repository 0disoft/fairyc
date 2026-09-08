package semantic_test

import "testing"

func TestAnalyzeTemporaryBorrowCreationConflicts(t *testing.T) {
	for name, body := range map[string]string{
		"retained shared":         "let ref u8[] view = ref data; edit(ref mut data); observe(view);",
		"retained mutable":        "let ref mut u8[] view = ref mut data; observe(ref data); edit(view);",
		"earlier argument":        "let ref u8[] view = ref data; both(view, ref mut data);",
		"later argument":          "let ref u8[] view = ref data; reverse(ref mut data, view);",
		"two temporary arguments": "both(ref data, ref mut data);",
		"two mutable arguments":   "exclusive(ref mut data, ref mut data);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": borrowCreationPrelude + "fn i32 main() { var u8[1] data = [42]; " + body + " return 0; }"})
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" || len(result.Unsupported) != 0 {
				t.Fatalf("borrow conflict diagnostics: %+v unsupported: %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestAnalyzeTemporaryBorrowCreationEndsAtLastUse(t *testing.T) {
	for name, body := range map[string]string{
		"after last use":       "let ref u8[] view = ref data; observe(view); edit(ref mut data);",
		"separate calls":       "edit(ref mut data); edit(ref mut data);",
		"compatible shared":    "let ref u8[] view = ref data; shared(view, ref data);",
		"two temporary shared": "shared(ref data, ref data);",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": borrowCreationPrelude + "fn i32 main() { var u8[1] data = [42]; " + body + " return 0; }"})
			assertCleanTypeResult(t, result)
		})
	}
}

const borrowCreationPrelude = `module main;
fn void observe(ref u8[] value) {}
fn void edit(ref mut u8[] value) {}
fn void both(ref u8[] left, ref mut u8[] right) {}
fn void reverse(ref mut u8[] left, ref u8[] right) {}
fn void exclusive(ref mut u8[] left, ref mut u8[] right) {}
fn void shared(ref u8[] left, ref u8[] right) {}
`

func TestAnalyzeDirectOwnerMoveBorrowLifetime(t *testing.T) {
	for name, move := range map[string]string{
		"binding":         "let Owner next = owner;",
		"grouped binding": "let Owner next = (owner);",
		"assignment":      "var Owner next = Owner {}; next = owner;",
	} {
		for _, live := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/after-last-use", true: "/live"}[live], func(t *testing.T) {
				body := "observe(view); " + move
				if live {
					body = move + " observe(view);"
				}
				result := analyzeText(t, map[string]string{"main.fy": `module main;
resource Owner { cleanup {} }
fn void observe(ref Owner value) {}
fn i32 main() { let Owner owner = Owner {}; let ref Owner view = ref owner; ` + body + ` return 0; }`})
				if !live {
					assertCleanTypeResult(t, result)
					return
				}
				if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2105" || len(result.Unsupported) != 0 {
					t.Fatalf("direct move with live borrow: %+v %+v", result.Diagnostics, result.Unsupported)
				}
			})
		}
	}
}

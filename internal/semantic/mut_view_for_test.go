package semantic_test

import (
	"strings"
	"testing"
)

func TestMutableViewForBorrowAndSourceRules(t *testing.T) {
	for _, element := range []string{"u8", "i64"} {
		for _, item := range []struct {
			name, body, code string
			unsupported      bool
		}{
			{"store", "for (let ELEMENT value in view) { view[0] = value; }", "", false},
			{"parameter-store", "for (let ELEMENT value in view) { modify(view); }", "", false},
			{"shared-reader", "for (let ELEMENT value in view) { read(view); }", "", false},
			{"owner-store", "for (let ELEMENT value in view) { data[0] = value; }", "E2101", false},
			{"owner-store-break", "for (let ELEMENT value in view) { data[0] = value; break; }", "E2101", false},
			{"owner-store-return", "for (let ELEMENT value in view) { data[0] = value; return; }", "E2101", false},
			{"owner-borrow-break", "for (let ELEMENT value in view) { let ref ELEMENT[] other = ref data; assert(other.len == 2); break; }", "E2101", false},
			{"after-break", "for (let ELEMENT value in view) { break; } data[0] = 9;", "", false},
			{"after-continue", "for (let ELEMENT value in view) { continue; } data[0] = 9;", "", false},
			{"child-store", "for (let ELEMENT value in view) { let ref mut ELEMENT[] child = view[..1]; child[0] = value; }", "", false},
			{"parent-read-break", "let ref mut ELEMENT[] child = view[..1]; for (let ELEMENT value in child) { assert(view.len == 2); break; }", "E2101", false},
			{"parent-write-return", "let ref mut ELEMENT[] child = view[..1]; for (let ELEMENT value in child) { view[0] = value; return; }", "E2101", false},
			{"sibling-break", "let ref mut ELEMENT[] child = view[..1]; for (let ELEMENT value in child) { let ref mut ELEMENT[] sibling = view[1..]; assert(sibling.len == 1); break; }", "E2101", false},
			{"parent-after", "let ref mut ELEMENT[] child = view[..1]; for (let ELEMENT value in child) { child[0] = value; } view[0] = 9;", "", false},
			{"retained-shared-reader", "for (let ELEMENT value in view) { let ref ELEMENT[] shared = as_shared(view); view[0] = value; read(shared); }", "E2101", false},
			{"item-write", "for (let ELEMENT value in view) { value = 9; }", "E1008", false},
			{"scope", "for (let ELEMENT value in view) {} let ELEMENT outside = value;", "E0101", false},
			{"mutable-descriptor", "var ref mut ELEMENT[] alias = view; for (let ELEMENT value in alias) {}", "", true},
		} {
			t.Run(element+"/"+item.name, func(t *testing.T) {
				text := `module main;
fn void modify(ref mut ELEMENT[] values) { values[0] = 7; }
fn void read(ref ELEMENT[] values) { assert(values.len > 0); }
fn ref ELEMENT[] as_shared(ref ELEMENT[] values) { return values; }
fn void run() { var ELEMENT[2] data = [1,2]; let ref mut ELEMENT[] view = ref mut data; ` + item.body + " }"
				result := analyzeText(t, map[string]string{"main.fy": strings.ReplaceAll(text, "ELEMENT", element)})
				if item.unsupported {
					if len(result.Unsupported) == 0 {
						t.Fatalf("mutable descriptor accepted: %+v", result.Diagnostics)
					}
				} else if item.code == "" {
					if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
						t.Fatalf("valid loop rejected: %+v %+v", result.Diagnostics, result.Unsupported)
					}
				} else {
					found := false
					for _, d := range result.Diagnostics {
						found = found || d.Code == item.code
					}
					if !found {
						t.Fatalf("expected %s: %+v %+v", item.code, result.Diagnostics, result.Unsupported)
					}
				}
			})
		}
		t.Run(element+"/parameter", func(t *testing.T) {
			text := "module main; fn void run(ref mut " + element + "[] view) { for (let " + element + " value in view) { view[0] = value; } }"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
				t.Fatalf("parameter: %+v %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestSharedByteViewForHoldsOwnerOnEarlyExit(t *testing.T) {
	for _, exit := range []string{"break;", "return;"} {
		text := "module main; fn void run() { var u8[2] data = [1,2]; let ref u8[] view = ref data; for (let u8 value in view) { data[0] = value; " + exit + " } }"
		result := analyzeText(t, map[string]string{"main.fy": text})
		found := false
		for _, d := range result.Diagnostics {
			found = found || d.Code == "E2101"
		}
		if !found {
			t.Fatalf("owner mutation accepted on %s: %+v %+v", exit, result.Diagnostics, result.Unsupported)
		}
	}
}

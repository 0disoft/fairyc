package cli

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestCandidateAtlasBorrowCompleteness(t *testing.T) {
	const main = `module main;import project_atlas_header as header;import std::bytes as bytes;import std::io as io;
@effects(console) fn i32 main(){var usize work=0;
 let legacy=switch(header::completeness_version(3,0,0,0,3,ref mut work)){case .ok(p):{p}case .err(_):{return 7;}};
 let observed=switch(header::completeness_observed_version(3,0,0,0,3,true,ref mut work)){case .ok(p):{p}case .err(_):{return 7;}};
 discard io::write("[");discard io::write_bytes(bytes::view(ref legacy));discard io::write(",");discard io::write_bytes(bytes::view(ref observed));discard io::write("]");return 0;}
`
	candidateAtlasProbeSize(t, "project_atlas_header", main, 32<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 {
			t.Fatalf("borrow completeness: %d %s", code, out)
		}
		var views [][]semanticsnapshot.Completeness
		if err := json.Unmarshal([]byte(out), &views); err != nil || len(views) != 2 || len(views[0]) != 14 || len(views[1]) != 14 {
			t.Fatalf("completeness views: %v %s", err, out)
		}
		found := false
		for i, before := range views[0] {
			after := views[1][i]
			if before.Relation != "borrows" {
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("borrow metadata changed unrelated completeness: %+v %+v", before, after)
				}
				continue
			}
			found = true
			if !reflect.DeepEqual(before, semanticsnapshot.Completeness{Relation: "borrows", Status: "unavailable", Reasons: []string{"candidate-atlas-not-collected"}}) || after.Relation != "borrows" || after.Status != "exact" || after.Resolved != 0 || after.External != 0 || after.Unknown != 0 || len(after.Reasons) != 0 {
				t.Fatalf("unobserved/observed completeness: %+v %+v", before, after)
			}
		}
		if !found {
			t.Fatal("missing borrow completeness")
		}
	})
}

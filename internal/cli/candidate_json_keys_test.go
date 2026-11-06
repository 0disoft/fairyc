package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

const candidateJSONKeysMain = `module main;import project_json_tree as json;` + atlasReadInput + `
@effects(console) fn i32 reject(u32 code,usize start,usize end){discard io::write_usize(code as usize);discard io::write("/");discard io::write_usize(start);discard io::write("/");discard io::write_usize(end);return 7;}
@effects(console,environment) fn i32 main(){
 let source=input();let copy=switch(bytes::copy(bytes::view(ref source))){case .ok(p):{p}case .err(_):{return 2;}};
 var usize measured=0;let tree=switch(json::parse(copy,ref mut measured)){case .ok(p):{p}case .err(p):{return reject(p.code,p.start,p.end);}};
 if(!bytes::equal(json::source(ref tree),bytes::view(ref source)) || json::next(ref tree,0)!=json::count(ref tree)){return 9;}
 if(process::arg_count()==2){var u8[32] raw=[0;32];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{return 2;}};let ref u8[] mode=ref raw;
  var usize meter=if(bytes::equal(mode[..n],b"budget")){16777216}else if(bytes::equal(mode[..n],b"short")){16777216-measured+1}else if(bytes::equal(mode[..n],b"exact")){16777216-measured}else{16777216-measured-1};
  let result=switch(json::parse(source,ref mut meter)){case .ok(p):{p}case .err(p):{return reject(p.code,p.start,p.end);}};return if(json::next(ref result,0)==json::count(ref result)){0}else{9};
 }
 discard io::write_usize(measured);return 0;
}`

func TestCandidateJSONKeys(t *testing.T) {
	object := func(reverse bool) string {
		fields := make([]string, 128)
		for i := range fields {
			index := i
			if reverse {
				index = len(fields) - 1 - i
			}
			fields[i] = fmt.Sprintf(`"field_%04d":0`, index)
		}
		return "{" + strings.Join(fields, ",") + "}"
	}
	cases := []struct{ name, source, duplicate string }{
		{"empty", `{}`, ""},
		{"nested-reuse", `{"a":{"a":1},"b":[{"a":2},{"a":3}]}`, ""},
		{"same-length-order", `{"c":1,"a":2,"b":3}`, ""},
		{"ascending", object(false), ""},
		{"descending", object(true), ""},
		{"duplicate-after-child", `{"a":{"b":0},"c":1,"a":2}`, `"a"`},
		{"duplicate-nested", `{"parent":{"z":0,"a":1,"z":2}}`, `"z"`},
		{"duplicate-escaped", `{"a":0,"\u0061":1}`, `"\u0061"`},
		{"duplicate-empty", `{"":0,"other":1,"":2}`, `""`},
		{"duplicate-unicode", `{"가":0,"\uac00":1}`, `"\uac00"`},
	}
	candidateAtlasProbe(t, "project_json_tree", candidateJSONKeysMain, func(t *testing.T, image string) {
		for _, fixture := range cases {
			t.Run(fixture.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, []byte(fixture.source))
				if fixture.duplicate != "" {
					start := strings.LastIndex(fixture.source, fixture.duplicate)
					want := fmt.Sprintf("4/%d/%d", start, start+len(fixture.duplicate))
					if code != 7 || out != want {
						t.Fatalf("decoded duplicate: %d %q want %q", code, out, want)
					}
					return
				}
				work, err := strconv.Atoi(out)
				if code != 0 || err != nil || work <= 0 {
					t.Fatalf("JSON key index: %d %q %v", code, out, err)
				}
				if strings.HasSuffix(fixture.name, "ending") {
					// The old key uniqueness loop alone charged this much, before
					// token scanning or structural validation. Keep a strict gain.
					legacyComparisons := 128 * 127 / 2 * (2*len("field_0000") + 1)
					if work >= legacyComparisons {
						t.Fatalf("duplicate-key work %d did not improve %d", work, legacyComparisons)
					}
					t.Logf("total parse work=%d; legacy uniqueness alone=%d", work, legacyComparisons)
				}
			})
		}
		for _, mode := range []string{"budget", "short", "exact", "more"} {
			code, out := candidateAtlasProcess(t, image, []byte(object(true)), mode)
			if mode == "budget" || mode == "short" {
				if code != 7 || !strings.HasPrefix(out, "3/") {
					t.Fatalf("JSON work boundary %s: %d %q", mode, code, out)
				}
			} else if code != 0 || out != "" {
				t.Fatalf("JSON exact work %s: %d %q", mode, code, out)
			}
		}
	})
}

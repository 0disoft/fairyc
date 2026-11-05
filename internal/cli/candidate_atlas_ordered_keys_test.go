package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestCandidateAtlasOrderedKeys(t *testing.T) {
	main := `module main;import project_atlas_model as model;import project_atlas_wire as wire;
import project_json_tree as json;` + atlasReadInput + `
@effects(console)fn i32 main(){var usize meter=0;let t=switch(json::parse(input(),ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
let rows=switch(model::rows(ref t,wire::field(ref t,0,b"rows"),4096,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
meter=if(process::arg_count()==2){16777216}else{0};let initial=meter;
let position=model::key_position(ref t,ref rows,b"key",wire::field(ref t,0,b"key"),ref mut meter);
discard io::write_usize(position);discard io::write("/");discard io::write_usize(meter-initial);return 0;}`
	candidateAtlasProbe(t, "project_atlas_model", main, func(t *testing.T, image string) {
		key := func(name string) map[string]string {
			return map[string]string{"package_instance": "p", "module": "m", "kind": "function", "name": name}
		}
		for _, count := range []int{0, 1, 2, 257} {
			rows := make([]map[string]any, count)
			for i := range rows {
				rows[i] = map[string]any{"key": key(fmt.Sprintf("n%04d", i*2))}
			}
			for _, requested := range []int{-1, 0, count / 2 * 2, (count - 1) * 2, count*2 + 1, 1} {
				name := fmt.Sprintf("n%04d", requested)
				data, err := json.Marshal(map[string]any{"rows": rows, "key": key(name)})
				if err != nil {
					t.Fatal(err)
				}
				want := count
				if requested >= 0 && requested%2 == 0 && requested/2 < count {
					want = requested / 2
				}
				for _, exhausted := range []bool{false, true} {
					args := []string{}
					if exhausted {
						args = []string{"exhausted"}
					}
					code, out := candidateAtlasProcess(t, image, data, args...)
					fields := strings.Split(out, "/")
					if code != 0 || len(fields) != 2 {
						t.Fatal("ordered key lookup", code, out)
					}
					position, e1 := strconv.Atoi(fields[0])
					work, e2 := strconv.Atoi(fields[1])
					if e1 != nil || e2 != nil || position != want && !exhausted || exhausted && (position != count || work != 0) {
						t.Fatalf("lookup n=%d key=%s exhausted=%v: %s want %d", count, name, exhausted, out, want)
					}
					// At most nine comparisons for 257 rows, each charging less
					// than 350 units for these complete four-part keys.
					if work > 9*350 {
						t.Fatalf("lookup exceeds logarithmic comparison budget: %d", work)
					}
				}
			}
		}
	})
}

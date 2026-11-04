package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestCandidateEditOptions(t *testing.T) {
	main := `module main;import project_edit_options as options;import std::io as io;
@effects(console) fn i32 main(){
 let p=switch(options::arguments()){case .ok(p):{p}case .err(_):{return 2;}};
 discard io::write_bytes(options::uid(ref p));discard io::write("\n");discard io::write_bytes(options::symbol(ref p));discard io::write("\n");
 discard io::write_bytes(options::body_file(ref p));discard io::write("\n");discard io::write_bytes(options::expected(ref p));discard io::write("\n");
 return if(io::write(if(options::apply(ref p)){"1"}else{"0"}).is_ok){0}else{4};
}`
	root, valid := editFixture(t)
	before := remoteTree(t, root)
	type invocation struct {
		args  []string
		apply bool
	}
	accepted := []invocation{{valid, false}}
	reverse := append([]string{}, valid[:3]...)
	for i := len(valid) - 2; i >= 3; i -= 2 {
		reverse = append(reverse, valid[i:i+2]...)
	}
	accepted = append(accepted, invocation{reverse, false}, invocation{append(append([]string{}, valid...), "--apply"), true})
	var rejected [][]string
	for i := 3; i < len(valid); i += 2 {
		missing := append([]string{}, valid[:i]...)
		rejected = append(rejected, append(missing, valid[i+2:]...))
		rejected = append(rejected, append(append([]string{}, valid...), valid[i:i+2]...))
	}
	rejected = append(rejected, valid[:len(valid)-1], append(append([]string{}, valid...), "extra"), append(append([]string{}, valid...), "--apply", "--apply"))
	unknown := append([]string{}, valid...)
	unknown[3] = "--unknown"
	rejected = append(rejected, unknown)
	command := append([]string{}, valid...)
	command[2] = "rename"
	rejected = append(rejected, command)

	candidateAtlasProbeSize(t, "project_edit_options", main, 16<<10, func(t *testing.T, image string) {
		for _, tc := range accepted {
			// Preview syntax acceptance is also checked against the real Go CLI.
			// Apply is parsed only here; the fixture must remain unchanged.
			if !tc.apply {
				var out, diagnostic bytes.Buffer
				if code := runAtlasEdit(tc.args[2:], root, &out, &diagnostic); code != 0 || !strings.Contains(out.String(), `"verified":true`) {
					t.Fatalf("Go preview oracle: %d %s %s", code, &out, &diagnostic)
				}
			}
			code, out := candidateAtlasProcess(t, image, nil, tc.args...)
			want := strings.Join([]string{valid[4], valid[6], valid[8], valid[10], "0"}, "\n")
			if tc.apply {
				want = strings.TrimSuffix(want, "0") + "1"
			}
			if code != 0 || out != want {
				t.Fatalf("edit options %v: %d %q; want %q", tc.args, code, out, want)
			}
		}
		for _, args := range rejected {
			var out, diagnostic bytes.Buffer
			if code := runAtlasEdit(args[2:], root, &out, &diagnostic); code != 2 {
				t.Fatalf("Go flag oracle: %v = %d %s", args, code, &diagnostic)
			}
			if code, out := candidateAtlasProcess(t, image, nil, args...); code != 2 || out != "" {
				t.Fatalf("invalid options %v: %d %q", args, code, out)
			}
		}
		if !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatal("option parsing or Go preview changed the workspace")
		}
	})
}

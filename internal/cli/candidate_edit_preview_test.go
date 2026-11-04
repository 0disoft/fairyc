package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

const candidateEditPreviewMain = `module main;import project_edit_preview as preview;
` + atlasReadInput + `
@effects(filesystem,environment,console) fn i32 main(){
 if(process::arg_count()!=4){return 2;}
 var u8[4096] a=[0;4096];var u8[4096] b=[0;4096];var u8[65] c=[0;65];
 let x=switch(process::arg(1,ref mut a)){case .ok(p):{p}case .err(_):{return 2;}};
 let y=switch(process::arg(2,ref mut b)){case .ok(p):{p}case .err(_):{return 2;}};
 let z=switch(process::arg(3,ref mut c)){case .ok(p):{p}case .err(_):{return 2;}};
 let ref u8[] uid=ref a;let ref u8[] symbol=ref b;let ref u8[] expected=ref c;let body=input();
 let result=switch(preview::prepare(b".",uid[..x],symbol[..y],expected[..z],bytes::view(ref body))){
  case .ok(p):{p}case .err(p):{discard io::write_usize(p.stage as usize);discard io::write("/");discard io::write_usize(p.code as usize);return 2;}
 };
 return if(io::write_bytes(preview::report(ref result)).is_ok){0}else{4};
}`

func TestCandidateEditPreview(t *testing.T) {
	const source = "module main;\r\nimport answer as answer;\r\n// 보존할 주석\r\nfn i32 leaf(){return answer::value();}\r\nfn i32 main(){return leaf();}\r\n"
	candidateAtlasProbeSize(t, "project_edit_preview", candidateEditPreviewMain, 2<<20, func(t *testing.T, image string) {
		root := writeWorkspaceSessionFixture(t, source)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatal(code, &diagnostic)
		}
		pkg, ok := prepared.workspace.Package(".")
		if !ok {
			t.Fatal("missing root package")
		}
		uid := pkg.Manifest.Package.UID
		var visible, full string
		for _, p := range prepared.partitions {
			if p.Header().PackageUID != uid {
				continue
			}
			for _, s := range p.Symbols() {
				if s.Declaration.Module == "main" && s.Declaration.Name == "leaf" {
					visible, full = s.Visible, s.Full
				}
			}
		}
		if visible == "" || full == "" {
			t.Fatal("missing exact Go symbol identities")
		}
		expected := fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
		before := remoteTree(t, root)
		t.Setenv("PATH", "") // Both complete checks must be executed by Fairy.
		for _, body := range []string{"{return 43;}", "{return answer::value()+1;}", "{if(true){return answer::value();}else{return 43;}}", "{return false;}", "{return absent();}", "{return answer::value(true);}", "{}", "{return 42;}fn void injected(){}"} {
			plan, err := sourceedit.PlanBody(prepared.workspace, "main", "leaf", expected, []byte(body))
			for _, symbol := range []string{visible, full} {
				code, out, stderr := candidateMCPProcess(t, image, root, []string{uid, symbol, expected}, body)
				if stderr != "" {
					t.Fatal(stderr)
				}
				if err != nil {
					stage := "5/"
					if strings.Contains(body, "injected") {
						stage = "4/"
					}
					if code != 2 || !strings.HasPrefix(out, stage) {
						t.Fatalf("bad body %q: %d %q; Go error %v", body, code, out, err)
					}
				} else {
					var got sourceedit.Report
					if code != 0 || len(out) > 8192 || json.Unmarshal([]byte(out), &got) != nil || got != plan.Report {
						t.Fatalf("preview %q: %d %s; want %+v", body, code, out, plan.Report)
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, root)) {
					t.Fatal("preview changed source, lock or created a file")
				}
			}
		}
		for _, bad := range []struct{ uid, symbol, hash, stage string }{
			{"example.session.answer", visible, expected, "1/2"},
			{uid, "fysym1-sha256-128-" + strings.Repeat("0", 32), expected, "1/3"},
			{uid, visible, strings.Repeat("0", 64), "4/4"},
		} {
			code, out, stderr := candidateMCPProcess(t, image, root, []string{bad.uid, bad.symbol, bad.hash}, "{return 43;}")
			if code != 2 || out != bad.stage || stderr != "" {
				t.Fatalf("stale/foreign selection: %d %q %q", code, out, stderr)
			}
		}
		path := filepath.Join(root, "src", "main.fy")
		if err := os.WriteFile(path, []byte(strings.Replace(source, "return leaf();", "return 44;", 1)), 0600); err != nil {
			t.Fatal(err)
		}
		changed := remoteTree(t, root)
		code, out, stderr := candidateMCPProcess(t, image, root, []string{uid, visible, expected}, "{return 43;}")
		if code != 2 || !strings.HasPrefix(out, "2/") || stderr != "" || !reflect.DeepEqual(changed, remoteTree(t, root)) {
			t.Fatalf("stale lock: %d %q %q", code, out, stderr)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		code, out, stderr = candidateMCPProcess(t, image, root, []string{uid, visible, expected}, "{return 43;}")
		if code != 0 || stderr != "" || !strings.Contains(out, `"verified":true`) || !reflect.DeepEqual(before, remoteTree(t, root)) {
			t.Fatalf("restored source: %d %q %q", code, out, stderr)
		}
	})
}

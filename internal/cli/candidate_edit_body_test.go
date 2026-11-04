package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

const candidateEditBodyMain = `module main;
import project_edit_body as edit;import syntax as syntax;
` + atlasReadInput + `
@effects(console) fn i32 main(){
 let data=input();let raw=bytes::view(ref data);if(raw.len<4){return 2;}
 let n=(raw[0] as usize)+(raw[1] as usize)*256+(raw[2] as usize)*65536+(raw[3] as usize)*16777216;
 if(n>raw.len-4){return 2;}
 var u8[65] expected=[0;65];var u8[256] name=[0;256];
 let e=switch(process::arg(1,ref mut expected)){case .ok(p):{p}case .err(_):{return 2;}};
 let k=switch(process::arg(2,ref mut name)){case .ok(p):{p}case .err(_):{return 2;}};
 let source=switch(bytes::copy(raw[4..4+n])){case .ok(p):{p}case .err(_):{return 2;}};
 let parsed=switch(edit::parse(source)){case .ok(p):{p}case .err(_):{return 2;}};
 var usize selected=syntax::declaration_count(ref parsed);var usize matches=0;var usize i=0;let ref u8[] spelling=ref name;
 while(i<syntax::declaration_count(ref parsed)){let d=syntax::declaration_at(ref parsed,i);if(d.kind==3 && bytes::equal(syntax::declaration_name(ref parsed,i+1),spelling[..k])){selected=i;matches+=1;}i+=1;}
 if(matches!=1){return 2;}
 let ref u8[] hash=ref expected;var usize meter=if(process::arg_count()>3){16777216}else{0};
 let result=switch(edit::replace(ref parsed,selected,hash[..e],raw[4+n..],ref mut meter)){case .ok(p):{p}case .err(_):{return 2;}};
 discard io::write_usize(edit::start(ref result));discard io::write("/");discard io::write_usize(edit::end(ref result));discard io::write("\n");
 discard io::write_bytes(edit::before_hash(ref result));discard io::write("\n");discard io::write_bytes(edit::after_hash(ref result));discard io::write("\n");
 return if(io::write_bytes(edit::after(ref result)).is_ok){0}else{4};
}`

func TestCandidateEditBody(t *testing.T) {
	const prefix = "module main;\r\n// 보존할 주석\r\nfn i32 leaf() "
	const suffix = "\r\n// suffix\r\nfn i32 main(){return leaf();}\r\n"
	const source = prefix + "{return 42;}" + suffix
	type fixture struct {
		name, source, target, body string
		stale, exhausted           bool
	}
	cases := []fixture{
		{name: "replace", body: "{return 43;}"},
		{name: "empty", body: "{}"}, // syntax-only; semantic validation is a host obligation.
		{name: "nested", body: "{if(true){return 42;}else{return 43;}}"},
		{name: "quoted-braces", body: "{/* } { */ return 42;}"},
		{name: "injected-sibling", body: "{return 42;}fn void injected(){}"},
		{name: "escaped-brace", body: "{return 42;}} fn void injected(){"},
		{name: "malformed", body: "{return ; return +;}"},
		{name: "missing-braces", body: "return 43;"},
		{name: "all-space", body: " \t\r\n"},
		{name: "nul", body: "{return 42;}\x00"},
		{name: "invalid-utf8", body: "{/*\xff*/return 42;}"},
		{name: "bom", body: "\ufeff{return 42;}"},
		{name: "zero-width-space", body: "\u200b{return 42;}"},
		{name: "outer-comment", body: "/*comment*/{return 42;}"},
		{name: "stale", body: "{return 43;}", stale: true},
		{name: "budget", body: "{return 43;}", exhausted: true},
		{name: "entry", target: "main", body: "{return 43;}"},
		{name: "missing", target: "absent", body: "{return 43;}"},
		{name: "public", source: "module main;pub fn i32 leaf(){return 42;}fn i32 main(){return leaf();}", body: "{return 43;}"},
		{name: "unsafe", source: "module main;unsafe fn i32 leaf(){return 42;}fn i32 main(){return 42;}", body: "{return 43;}"},
		{name: "generic", source: "module main;fn T leaf<T>(T value){return value;}fn i32 main(){return 42;}", body: "{return 43;}"},
		{name: "attributed", source: "module main;@effects(console) fn i32 leaf(){return 42;}fn i32 main(){return 42;}", body: "{return 43;}"},
	}
	for _, r := range "\t\n\v\f\r \u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000" {
		cases = append(cases, fixture{name: fmt.Sprintf("whitespace-U%04X", r), body: string(r) + "{return 43;}" + string(r)})
	}
	for _, n := range []int{sourceedit.MaxBody - 1, sourceedit.MaxBody, sourceedit.MaxBody + 1} {
		cases = append(cases, fixture{name: fmt.Sprint("body-bytes-", n), body: "{return 43;}" + strings.Repeat(" ", n-len("{return 43;}"))})
	}

	candidateAtlasProbeSize(t, "project_edit_body", candidateEditBodyMain, 512<<10, func(t *testing.T, image string) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				original := tc.source
				if original == "" {
					original = source
				}
				target := tc.target
				if target == "" {
					target = "leaf"
				}
				expected := fmt.Sprintf("%x", sha256.Sum256([]byte(original)))
				if tc.stale {
					expected = strings.Repeat("0", 64)
				}
				var packet bytes.Buffer
				if err := binary.Write(&packet, binary.LittleEndian, uint32(len(original))); err != nil {
					t.Fatal(err)
				}
				packet.WriteString(original)
				packet.WriteString(tc.body)
				args := []string{expected, target}
				if tc.exhausted {
					args = append(args, "exhausted")
				}
				code, out := candidateAtlasProcess(t, image, packet.Bytes(), args...)
				// The Go owner supplies independent AST spans, raw byte hashes and
				// complete semantic verification for the positive bodies below.
				root := writeWorkspaceSessionFixture(t, original)
				var diagnostic bytes.Buffer
				prepared, status := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
				if status != 0 {
					t.Fatalf("prepare Go edit oracle: %d %s", status, &diagnostic)
				}
				plan, err := sourceedit.PlanBody(prepared.workspace, "main", target, expected, []byte(tc.body))
				if tc.name == "empty" {
					if code != 0 || err == nil { // private core must not claim semantic proof.
						t.Fatalf("syntax/semantic separation: %d %q %v", code, out, err)
					}
					return
				}
				if err != nil || tc.exhausted {
					if code != 2 || out != "" {
						t.Fatalf("rejected input: %d %q, Go error %v", code, out, err)
					}
					return
				}
				r := plan.Report
				after := original[:r.BodyStartByte] + strings.TrimSpace(tc.body) + original[r.BodyEndByte:]
				want := fmt.Sprintf("%d/%d\n%s\n%s\n%s", r.BodyStartByte, r.BodyEndByte, r.BeforeSHA256, r.AfterSHA256, after)
				if code != 0 || out != want {
					t.Fatalf("body replacement: code=%d\ngot %q\nwant %q", code, out, want)
				}
			})
		}
	})
}

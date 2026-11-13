package cli

import "testing"

func TestCandidateCopiedClosureU32Policy(t *testing.T) {
	candidateInlineProbe(t, []string{"copied_closure_admission"}, "import copied_closure_admission as admission;", `
let p=switch(make(b"module root;fn i32 main(){let u32 captured=40;let direct=fn [u32 captured] u32(u32 value){let u32 result=captured+value;return result;};return 0;}")){case .ok(p):{p}case .err(_):{assert(false);return 1;}};
var usize i=1;var usize found=0;while(i<=syntax::expression_count(ref p)){if(syntax::expression_at(ref p,i).kind==34){found=i;}i+=1;}assert(found!=0 && admission::validate(ref p,found)==0 && !admission::value_signature(ref p,found));
let q=switch(make(b"module root;fn i32 main(){let direct=fn [] i32(i32 value){let u32 unused=0;return value;};return 0;}")){case .ok(p):{p}case .err(_):{assert(false);return 1;}};
i=1;found=0;while(i<=syntax::expression_count(ref q)){if(syntax::expression_at(ref q,i).kind==34){found=i;}i+=1;}assert(found!=0 && admission::validate(ref q,found)==0 && !admission::value_signature(ref q,found));
`)
}

"""Observe unchanged MIR/C behavior after lowering source responsibility splits."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "mir-source-split"
observer.SDK = observer.REPO / "examples/lexer_candidate/src/candidate_limits.fy"
observer.DEPENDENCIES = ()
observer.RUNTIME_NAMES = ()
observer.PREAMBLE = "module main;"
observer.PREFIX = """#include <stdint.h>
static int32_t fixture_total;
void fixture_drop(int32_t n){fixture_total+=n;}
int32_t fixture_drops(void){return fixture_total;}
"""
observer.SUFFIX = ""
observer.CASE_OBSERVER = Path(__file__)
observer.CASES = {
    "array-view-and-short-circuit": 'fn i32 fill(ref mut u8[] b){b[0]=9;return b[0] as i32;}fn i32 main(){var u8[2] a=[1,2];let i32 n=fill(ref mut a);assert(n==9 && a[1]==2);let bool b=false && {assert(false);true};if(b){return 0;}return 7;}',
    "enum-result-and-nested-switch": 'enum E{A;B;}fn Result<i32,E> choose(bool b){return if(b){.ok(7)}else{.err(.B)};}fn i32 main(){let i32 n=switch(choose(true)){case .ok(v):{v}case .err(_):{0}};assert(n==7);assert(switch(choose(false)){case .ok(_):{false}case .err(e):{switch(e){case .A:{false}case .B:{true}}}});return 7;}',
    "factory-move-and-cleanup": 'unsafe extern C fn void fixture_drop(i32 n);unsafe extern C fn i32 fixture_drops();resource R{i32 n;cleanup{unsafe{fixture_drop(this.n);};}}fn R make(){return R{.n=1};}fn i32 main(){{let R x=make();assert(x.n==1);};assert(unsafe{fixture_drops()}==1);return 7;}',
}
observer.REJECT = {
    "later-argument-borrow-conflict": ('fn void fill(ref mut u8[] b,i32 n){discard n;b[0]=8;}fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;fill(b,{b[0]=7;0});return 7;}', "code 2101"),
}
if __name__ == "__main__":
    observer.main()

"""Execute fixed-array source-set signatures through the actual Fairy pipeline."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "module-array-signatures"
observer.SDK = observer.REPO / "examples/lexer_candidate/src/candidate_limits.fy"
observer.DEPENDENCIES = (observer.REPO / "examples/lexer_candidate/probes/module_array_provider.fy",)
observer.RUNTIME_NAMES = ()
observer.PREAMBLE = "module main;import array_provider as p;import array_provider as q;"
observer.PREFIX = "#include <stddef.h>\n"
observer.SUFFIX = ""
observer.CASE_OBSERVER = Path(__file__)
observer.CASES = {
    "array-parameter-return": 'fn i32 main(){var i32[20] a=[0;20];a[19]=7;let i32[20] b=p::copy(a);a[19]=9;assert(b.len==20 && a[19]==9);return b[19];}',
    "large-length": 'fn i32 main(){var usize[1153] a=[0;1153];a[1152]=7;let b=p::large(a);assert(b.len==1153 && b[1152]==7);return 7;}',
    "qualified-nominal-alias": 'struct Token{i32 n;}fn i32 main(){let p::Token[2] a=[p::Token{.n=7};2];let q::Token[2] b=q::tokens(a);let Token[2] local=[Token{.n=9};2];assert(local[1].n==9 && b.len==2);return b[1].n;}',
    "rank-two": 'fn i32 main(){let i32[3][4] a=[[7;3];4];let b=p::matrix(a);assert(b.len==4 && b[3].len==3);return b[3][2];}',
    "enum-and-option": 'fn i32 main(){let p::Kind[2] a=[.On;2];let b=q::kinds(a);assert(switch(b[1]){case .Off:{false}case .On:{true}});let Option<i32>[2] n=[.some(7);2];let result=p::options(n);return switch(result[1]){case .some(v):{v}case .none:{0}};}',
}
observer.REJECT = {
    "wrong-length": ('fn i32 main(){let i32[19] a=[7;19];discard p::copy(a);return 7;}', "phase 5 code 1"),
    "wrong-element": ('fn i32 main(){let i64[20] a=[7;20];discard p::copy(a);return 7;}', "phase 5 code 1"),
    "wrong-nominal-owner": ('struct Token{i32 n;}fn i32 main(){let Token[2] a=[Token{.n=7};2];discard p::tokens(a);return 7;}', "phase 5 code 1"),
    "resource-element": ('resource R{i32 n;cleanup{}}fn R[2] bad(){return [R{.n=7};2];}fn i32 main(){return 7;}', "phase 1 code 9"),
    "borrow-element": ('fn Option<ref i32>[2] bad(){assert(false);}fn i32 main(){return 7;}', "phase 1 code 9"),
    "zero-length": ('fn i32[0] bad(){assert(false);}fn i32 main(){return 7;}', "phase 1 code 9"),
    "too-long": ('fn i32[4097] bad(){assert(false);}fn i32 main(){return 7;}', "phase 1 code 9"),
    "rank-two-too-large": ('fn i32[65][2] bad(){assert(false);}fn i32 main(){return 7;}', "phase 1 code 9"),
    "rank-three": ('fn i32[2][2][2] bad(){assert(false);}fn i32 main(){return 7;}', "phase 1 code 9"),
    "foreign-array": ('unsafe extern C fn i32 host(i32[20] n);fn i32 main(){return 7;}', "phase 1 code 5102"),
}

if __name__ == "__main__":
    observer.main()

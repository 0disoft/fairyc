module tests::resource_loops;
import tests::result_types as fixture;
import tests::hir_nominal as hir;
import type_check as typing;
import hir_model as h;
import std::bytes as bytes;

fn void mismatch(ref u8[] source,ref u8[] loop,ref u8[] header){
    let t=typing::check(fixture::resolved(source));let p=typing::problem(ref t);
    assert(p.code==47 && bytes::equal(source[p.start..p.end],loop));
    assert(bytes::equal(source[p.loop_start..p.loop_end],header));
    assert(bytes::equal(source[p.move_start..p.move_end],b"r"));
}
@test
fn void every_while_edge_has_loop_and_move_provenance(){
    mismatch(b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R r){while(b){take(r);}}",b"while(b){take(r);}",b"while(b");
    mismatch(b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R r){while(b){take(r);break;}}",b"while(b){take(r);break;}",b"while(b");
    mismatch(b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R r){while(b){take(r);continue;}}",b"while(b){take(r);continue;}",b"while(b");
}
@test
fn void classic_for_preserves_each_exit_and_post_clause_edge(){
    mismatch(b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(R r){for(var i32 i=0;i<3;i+=1){take(r);}}",b"for(var i32 i=0;i<3;i+=1){take(r);}",b"for(var i32 i=0;i<3;i+=1)");
    mismatch(b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(R r){for(;;){take(r);break;}}",b"for(;;){take(r);break;}",b"for(;;)");
}
@test
fn void restored_roots_and_body_local_generations_are_independent(){
    let p=hir::retained(b"module main;resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R input){var R r=input;while(b){let R local=R{};take(local);take(r);r=R{};continue;}return r;}");
    assert(h::function_count(ref p)==2);
    let dead=hir::retained(b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R input){var R r=input;take(r);while(b){r=R{};take(r);}}");
    assert(h::function_count(ref dead)==2);
}
@test
fn void nested_control_targets_and_terminal_returns_keep_owners(){
    let p=hir::retained(b"module main;resource R{cleanup{}}fn void take(R r){}fn R f(bool a,bool b,R r){while(a){let R local=R{};while(b){break;}take(local);continue;}return r;}");
    assert(h::function_count(ref p)==2);
    let terminal=hir::retained(b"module main;resource R{cleanup{}}fn R f(bool b,R r){while(b){return r;}return r;}");
    assert(h::function_count(ref terminal)==1);
    mismatch(b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(bool a,bool b,R r){while(a){while(b){take(r);break;}}}",b"while(b){take(r);break;}",b"while(b");
}
@test
fn void condition_effects_and_repeated_reads_use_the_post_condition_baseline(){
    let p=hir::retained(b"module main;resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R input){var R r=input;take(r);while({r=R{};b}){let R next=r;r=next;break;}return r;}");
    assert(h::function_count(ref p)==2);
    let source=b"module main;resource R{cleanup{}}fn bool inspect(R r){return true;}fn void f(R r){while(inspect(r)){}}";
    let t=typing::check(fixture::resolved(source));let e=typing::problem(ref t);assert(e.code==45);
    assert(bytes::equal(source[e.start..e.end],b"r") && e.start==e.move_start);
}

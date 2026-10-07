module tests::mir_resource_exit;
import tests::hir_nominal as fixture;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;
import mir_output as output;
import c_emit as c;
import std::bytes as bytes;

fn m::Program compiled(ref u8[] source){return switch(lower::lower(fixture::retained(source))){case .ok(p):{p}case .err(_):{assert(false);return compiled(source);}};}
@test fn void moved_roots_drop_once_in_reverse_scope_order(){
    let p=compiled(b"module main;resource R{i32 n;cleanup{this.n+=1;visit(this.n);}}fn void visit(i32 n){}fn i32 main(){{let R a=R{.n=1};let R b=a;let R c=R{.n=2};};return 0;}");
    assert(m::drop_count(ref p)==1 && m::drop_at(ref p,0).nonempty && verify::check(ref p).code==0);
    var usize calls=0;var usize moves=0;var usize updates=0;var usize i=1;
    while(i<=m::block_count(ref p)){let b=m::block_at(ref p,i);if(b.kind==5 && b.type!=0){assert(b.type==m::drop_at(ref p,0).type && b.callee==m::drop_at(ref p,0).target);calls+=1;}i+=1;}
    i=1;while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);if(o.kind==24){moves+=1;}if(o.kind==8){assert(o.result==o.left);updates+=1;}i+=1;}
    assert(calls==2 && moves==4 && updates==1);
    var bytes::Buffer mir=bytes::empty();var bytes::Buffer native=bytes::empty();assert(output::render(ref p,ref mut mir)==0 && c::render(ref p,ref mut native)==0);
    assert(bytes::equal(bytes::view(ref mir)[..23],b"fairyc.resource-mir/v1\n"));
}
@test fn void empty_cleanup_and_consuming_parameter_do_not_duplicate_drops(){
    let p=compiled(b"module main;resource R{cleanup{}}fn void take(R r){}fn i32 main(){{let R r=R{};take(r);};return 0;}");
    assert(m::drop_count(ref p)==1 && !m::drop_at(ref p,0).nonempty && verify::check(ref p).code==0);
    var usize drops=0;var usize i=1;while(i<=m::operation_count(ref p)){if(m::operation_at(ref p,i).kind==25){drops+=1;}i+=1;}assert(drops==1);
}
@test fn void loop_exits_discharge_only_body_and_nested_roots(){
    let p=compiled(b"module main;resource R{cleanup{}}fn i32 main(){let R outer=R{};var i32 n=0;while(n<2){let R r=R{};n+=1;{let R inner=R{};if(n==1){continue;}else{break;};};}return 0;}");assert(verify::check(ref p).code==0);
    var usize drops=0;var usize i=1;while(i<=m::operation_count(ref p)){if(m::operation_at(ref p,i).kind==25){drops+=1;}i+=1;}assert(drops==5);
}
@test fn void return_and_propagation_discharge_live_roots_and_preserve_donors(){
    let p=compiled(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=65};}fn R id(R r){return r;}fn i32 main(){let R r=id(make());return r.n;}");assert(verify::check(ref p).code==0);
    let result=compiled(b"module main;resource R{cleanup{}}fn Result<i32,bool> f(bool ok){let R r=R{};let Result<i32,bool> input=if(ok){.ok(7)}else{.err(true)};let i32 n=input?;return .ok(n);}fn i32 main(){return switch(f(false)){case .ok(n):{n}case .err(_):{1}};}");assert(verify::check(ref result).code==0);
    let selected=compiled(b"module main;resource R{cleanup{}}fn R f(bool yes){return if(yes){R{}}else{R{}};}fn i32 main(){let R r=f(true);return 0;}");assert(verify::check(ref selected).code==0);
}

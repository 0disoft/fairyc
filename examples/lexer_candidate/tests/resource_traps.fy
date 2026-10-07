module tests::resource_traps;
import tests::hir_nominal as fixture;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;

@test fn void assertion_traps_have_no_unwind_or_cleanup_successor(){
    let p=switch(lower::lower(fixture::retained(b"module main;fn void visit(i32 n){}resource R{cleanup{visit(65);}}fn i32 main(){let R r=R{};defer visit(66);assert(false);}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0);var usize traps=0;var usize i=1;
    while(i<=m::block_count(ref p)){let b=m::block_at(ref p,i);if(b.kind==6){assert(b.first==0 && b.last==0 && b.target==0 && b.alternative==0 && b.callee==0);traps+=1;}i+=1;}assert(traps==1);
}

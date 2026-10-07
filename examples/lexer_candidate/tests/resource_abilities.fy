module tests::resource_abilities;
import tests::hir_nominal as fixture;
import mir_lower as lower;
import mir_model as m;
import mir_verify as verify;

@test fn void burn_is_distinct_and_never_schedules_drop(){
    let p=switch(lower::lower(fixture::retained(b"module main;@abilities(store,key)resource R{i32 n;cleanup{}}fn i32 main(){let R r=R{.n=1};burn r;return 0;}"))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(verify::check(ref p).code==0 && m::drop_at(ref p,0).abilities==6);
    var usize burns=0;var usize i=1;while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);assert(o.kind!=25 && o.kind!=26);if(o.kind==27){burns+=1;assert(m::burnable(ref p,o.type));}i+=1;}assert(burns==1);
}

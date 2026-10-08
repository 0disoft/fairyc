module tests::hir_borrow_cache;
import tests::hir_nominal as fixture;
import hir_borrow_cache as cache;
import hir_borrow_paths as paths;
import hir_model as h;

fn void equivalent(ref u8[] source){
    let p=fixture::retained(source);var usize work=0;
    let slots=switch(cache::create(ref p,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    var usize id=1;var usize dead=0;
    while(id<=h::expression_count(ref p)){
        let e=h::expression_at(ref p,id);var usize reference_work=0;
        let bool expected=paths::reachable(ref p,e.start,e.end,ref mut reference_work);
        assert(cache::reachable(ref slots,id,false,ref mut work)==expected);if(!expected){dead+=1;}id+=1;
    }
    id=1;while(id<=h::statement_count(ref p)){
        let s=h::statement_at(ref p,id);var usize reference_work=0;
        assert(cache::reachable(ref slots,id,true,ref mut work)==paths::reachable(ref p,s.start,s.end,ref mut reference_work));id+=1;
    }
    assert(dead>0);
    work=1048576;assert(!cache::reachable(ref slots,1,false,ref mut work));
    work=1048576;assert(switch(cache::create(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==2}});
}
@test fn void indexed_reachability_preserves_definite_exits_and_loop_backedges(){
    equivalent(b"module main;fn i32 f(bool flag){if(flag){return 1;discard 2;}else{discard 3;}while(flag){if(flag){continue;discard 4;}break;discard 5;}return 7;discard 6;}fn i32 main(){return f(false);}");
    equivalent(b"module main;enum Kind{A;B;}fn i32 f(Kind k){switch(k){case .A:{return 1;}case .B:{return 2;}};discard 3;return 7;}fn i32 main(){return f(.A);}");
}

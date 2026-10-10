module module_unit_cleanup_liveness;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_cleanup_mir_model as ir;
import std::vec as v;

fn ir::Problem fail(u32 code,usize module_index){return ir::Problem{.code,.module_index,.start=0,.end=0};}
// Match the old carries relation: tagged payloads, arrays, safe references,
// typed pointers and iterator carriers. Nominal fields and Vec elements are
// resolved by their own cleanup recipes rather than this reachability test.
fn ir::Problem mark(ref values::Plan p,usize type,usize module_index,usize depth,ref mut v::Vec<u8> seen,ref mut usize work){
    if(depth>=64 || !m::charge(work,1)){return fail(1002,module_index);}if(type==0){return fail(0,0);}if(type>values::type_count(p)){return fail(1001,module_index);}if(*v::at<u8>(seen,type)!=0){return fail(0,0);}
    let slot=v::at_mut<u8>(seen,type);*slot=1;let t=values::type_at(p,type);
    if(t.kind>=5 && t.kind<=8 || t.kind==12 && t.first!=0 || t.kind==16){let first=mark(p,t.first,module_index,depth+1,seen,work);if(first.code!=0){return first;}if(t.kind==6){return mark(p,t.second,module_index,depth+1,seen,work);}}
    return fail(0,0);
}
pub fn Result<v::Vec<u8>,ir::Problem> build(ref values::Plan p,ref mut usize work){
    let count=values::type_count(p);if(count>4096){return .err(fail(1002,0));}var v::Vec<u8> seen=v::empty<u8>();var usize i=0;
    while(i<=count){if(!m::charge(work,1)){return .err(fail(1002,0));}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(fail(1003,0));}i+=1;}
    i=0;while(i<values::count(p)){let body=values::at(p,i);let f=m::frame(body);let result=mark(p,f.result,f.module_index,0,ref mut seen,work);if(result.code!=0){return .err(result);}var usize j=1;
        while(j<=m::local_count(body)){let e=mark(p,m::local_at(body,j).type,f.module_index,0,ref mut seen,work);if(e.code!=0){return .err(e);}j+=1;}
        j=1;while(j<=m::expression_count(body)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index));}let n=m::expression_at(body,j);if(n.state==1){let e=mark(p,n.type,f.module_index,0,ref mut seen,work);if(e.code!=0){return .err(e);}}j+=1;}i+=1;
    }return .ok(seen);
}

module module_unit_origin_check;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_origin_returns as origins;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_origin_shape as shape;
import module_unit_moves as moves;
import module_unit_cleanup_policy as cleanup_policy;
import module_unit_value_model as m;
import module_unit_value_builtin as borrows;
import module_unit_interfaces as interfaces;
import syntax as syntax;
import std::vec as v;

pub struct Problem {u32 phase;u32 code;usize module_index;usize start;usize end;}
fn Result<bool,m::Problem> contains_borrow(ref values::Plan typed,ref interfaces::Interfaces imported,usize id,m::Frame frame,ref mut usize work,usize depth){
    if(depth>=64 || !m::charge(work,1)){return .err(m::fail(frame,10,0,0,0,0));}if(id==0){return .ok(false);}let t=values::type_at(typed,id);
    if(borrows::borrowed(t) || t.kind==17 || t.kind==18){return .ok(true);}
    if(t.kind==5 || t.kind==6 || t.kind==13 || t.kind==16){let first=switch(contains_borrow(typed,imported,t.first,frame,work,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(first){return .ok(true);}return if(t.kind==6){contains_borrow(typed,imported,t.second,frame,work,depth+1)}else{.ok(false)};}
    if(t.kind==2 || t.kind==3 || t.kind==4){var usize i=0;while(i<interfaces::shape_count(imported)){if(!m::charge(work,1)){return .err(m::fail(frame,10,0,0,0,0));}let s=interfaces::shape_at(imported,i);if(s.type==id){var usize j=0;while(j<s.field_count){if(!m::charge(work,1)){return .err(m::fail(frame,10,0,0,0,0));}let f=interfaces::field_at(imported,s.field_start+j);let first=switch(contains_borrow(typed,imported,f.type,frame,work,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(first){return .ok(true);}var usize k=0;while(k<f.payload_count){let payload=interfaces::field_at(imported,f.payload_start+k);let found=switch(contains_borrow(typed,imported,payload.type,frame,work,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};if(found){return .ok(true);}k+=1;}j+=1;}return .ok(false);}i+=1;}}
    return .ok(false);
}
// One bounded primary unit, in the exact manifest/source order. This does not
// claim loan/lifetime validation or split-unit HIR/MIR execution.
pub fn Result<diagnostics::Report,Problem> check(ref v::Vec<syntax::Parsed> sources){
    let metadata=switch(signatures::build(sources)){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=1,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    let bound=switch(binding::bind(sources,ref metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=2,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=3,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    let typed=switch(values::build(sources,ref metadata,ref ready)){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=4,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    var usize work=values::work(ref typed);var usize i=0;while(i<values::count(ref typed)){let body=values::at(ref typed,i);let f=m::frame(body);let leaves=switch(shape::leaves(ref typed,f.result,f,ref mut work,0)){case .ok(n):{n}case .err(p):{return .err(Problem{.phase=5,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};if(leaves>1){let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,f.module_index),f.declaration-1);return .err(Problem{.phase=5,.code=2,.module_index=f.module_index,.start=d.start,.end=d.end});}if(f.result!=0 && leaves==0){let borrowed=switch(contains_borrow(ref typed,closure::imported(ref ready),f.result,f,ref mut work,0)){case .ok(b):{b}case .err(p):{return .err(Problem{.phase=5,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};if(borrowed){let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,f.module_index),f.declaration-1);return .err(Problem{.phase=5,.code=2,.module_index=f.module_index,.start=d.start,.end=d.end});}}i+=1;}
    let inferred=switch(origins::build_budgeted(sources,ref metadata,ref typed,work)){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=5,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    let moved=switch(moves::build(sources,ref typed,origins::work(ref inferred))){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=7,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    let checked_cleanup=switch(cleanup_policy::build(sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=9,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    return switch(diagnostics::collect_with_cleanup(ref inferred,ref moved,ref checked_cleanup)){case .ok(p):{.ok(p)}case .err(p):{.err(Problem{.phase=8,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end})}};
}

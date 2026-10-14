module module_project_artifact_coverage;
import module_project_artifact_metadata as matching;
import module_project_artifact_frames as frames;
import module_project_artifact_types as types;
import module_project_compile as compile;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_identity_artifact_model as identities;
import module_units as grouping;
import generic_source_plan as source;
import module_signatures as signatures;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::vec as v;

pub struct Problem {u32 stage;u32 code;compile::Problem source;}
pub struct Stats {usize files;usize units;usize retries;usize frames;usize types;usize layouts;usize fields;usize cleanup_fields;usize callables;usize parameters;usize names;}
// Published only after every source range and every retained table is covered.
// State/SDK matching and a same-owner Complete/cache factory still follow.
pub resource Covered {Stats stats;cleanup{}}
pub fn Stats stats(ref Covered p){return p.stats;}
struct Domains {usize frames;usize types;usize layouts;usize fields;usize cleanup_fields;usize callables;usize parameters;usize names;usize end;}
struct Progress {usize files;usize units;usize retries;}
fn Problem fail(u32 stage,u32 code,usize first,usize stop){return Problem{.stage,.code,.source=compile::Problem{.stage=0,.code=0,.module_index=first,.start=0,.end=0,.first,.stop,.expected=0,.actual=0,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=0,.field_start=0,.field_end=0}};}
fn Domains domains(ref ir::Program p){let c=ir::cleanup_ir(p);let types=ir::count(p);let layouts=types+clean::type_count(c);let fields=layouts+ir::layout_count(p);let cleanup_fields=fields+ir::field_count(p);let callables=cleanup_fields+clean::field_count(c);let parameters=callables+ir::callable_count(p);let names=parameters+ir::parameter_count(p);return Domains{.frames=0,.types,.layouts,.fields,.cleanup_fields,.callables,.parameters,.names,.end=names+ir::name_bytes(p)};}
fn u32 mark(ref mut v::Vec<u8> seen,usize base,usize id,usize count,ref mut usize work){if(!budget::charge(work,1)){return 1002;}if(id==0 || id>count || base>v::len<u8>(seen) || count>v::len<u8>(seen)-base){return 1001;}let cell=v::at_mut<u8>(seen,base+id-1);*cell=1;return 0;}
fn u32 range(ref mut v::Vec<u8> seen,usize base,usize first,usize stop,usize count,ref mut usize work){if(first>stop || stop>count){return 1001;}var usize i=first;while(i<stop){let code=mark(seen,base,i+1,count,work);if(code!=0){return code;}i+=1;}return 0;}
fn u32 unit(ref ir::Program p,ref matching::Matched matched,Domains d,ref mut v::Vec<u8> seen,ref mut usize work){let map=matching::frame_mapping(matched);let typed=frames::type_mapping(map);var usize i=1;
    while(i<=frames::target_count(map)){if(!budget::charge(work,1)){return 1002;}if(frames::used(map,i)){let code=mark(seen,d.frames,i,ir::count(p),work);if(code!=0){return code;}}i+=1;}
    i=1;while(i<=types::target_count(typed)){if(!budget::charge(work,1)){return 1002;}if(types::used(typed,i)){let code=mark(seen,d.types,i,clean::type_count(ir::cleanup_ir(p)),work);if(code!=0){return code;}}i+=1;}
    i=0;while(i<matching::layout_count(matched)){let code=mark(seen,d.layouts,matching::layout_at(matched,i),ir::layout_count(p),work);if(code!=0){return code;}i+=1;}
    i=0;while(i<matching::callable_count(matched)){let code=mark(seen,d.callables,matching::callable_at(matched,i),ir::callable_count(p),work);if(code!=0){return code;}i+=1;}return 0;
}
fn Problem append(ref ir::Program p,ref identities::Snapshot ids,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop,usize depth,Domains d,ref mut v::Vec<u8> seen,ref mut Progress progress,ref mut usize work){
    if(!budget::charge(work,1)){return fail(2,1002,first,stop);}if(first!=progress.files || first>=stop || stop>v::len<syntax::Parsed>(sources) || progress.units>=256){return fail(2,if(progress.units>=256){1002}else{1001},first,stop);}
    switch(compile::check_range(sources,metadata,first,stop)){
        case .ok(checked):{
            let matched=switch(matching::match(p,ids,compile::checked_state(ref checked),plan,sources,metadata,work)){case .ok(p):{p}case .err(code):{return fail(3,code,first,stop);}};
            let code=unit(p,ref matched,d,seen,work);if(code!=0){return fail(3,code,first,stop);}progress.files=stop;progress.units+=1;return fail(0,0,first,stop);
        }
        case .err(e):{
            // Reuse the cold compiler's source-check capacity policy. Comparison
            // failures never split or reset the shared caller-owned history.
            if(!compile::capacity_failure(e) || stop-first<=1 || depth>=8){return Problem{.stage=2,.code=e.code,.source=e};};progress.retries+=1;let middle=first+(stop-first)/2;
            let left=append(p,ids,plan,sources,metadata,first,middle,depth+1,d,seen,progress,work);if(left.code!=0){return left;}return append(p,ids,plan,sources,metadata,middle,stop,depth+1,d,seen,progress,work);
        }
    };return fail(2,1001,first,stop);
}
fn u32 tables(ref ir::Program p,Domains d,ref mut v::Vec<u8> seen,ref mut usize work){var usize i=0;let cleaned=ir::cleanup_ir(p);
    while(i<ir::layout_count(p)){if(!budget::charge(work,1)){return 1002;}if(*v::at<u8>(seen,d.layouts+i)==0){return 1001;}let s=ir::layout_at(p,i);let fields=range(seen,d.fields,s.value.field_start,s.value.field_start+s.value.field_count,ir::field_count(p),work);if(fields!=0){return fields;}let text=range(seen,d.names,s.name_start,s.name_end,ir::name_bytes(p),work);if(text!=0){return text;}
        let g=clean::glue_at(cleaned,s.value.type);if(g.kind==1){let code=range(seen,d.cleanup_fields,g.field_start,g.field_start+g.field_count,clean::field_count(cleaned),work);if(code!=0){return code;}}i+=1;}
    i=0;while(i<ir::field_count(p)){if(!budget::charge(work,1)){return 1002;}if(*v::at<u8>(seen,d.fields+i)==0){return 1001;}let f=ir::field_at(p,i);let code=range(seen,d.names,f.name_start,f.name_end,ir::name_bytes(p),work);if(code!=0){return code;}i+=1;}
    i=0;while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return 1002;}if(*v::at<u8>(seen,d.callables+i)==0){return 1001;}let c=ir::callable_at(p,i);let parameters=range(seen,d.parameters,c.parameter_start,c.parameter_start+c.parameter_count,ir::parameter_count(p),work);if(parameters!=0){return parameters;}let text=range(seen,d.names,c.link_start,c.link_end,ir::name_bytes(p),work);if(text!=0){return text;}i+=1;}
    // A signature-only unused resource has no selected layout/recipe. Preserve
    // its canonical empty placeholder and reject injected cleanup authority.
    var v::Vec<u8> shaped=v::empty<u8>();i=0;while(i<clean::type_count(cleaned)){if(!budget::charge(work,1)){return 1002;}if(!v::push<u8>(ref mut shaped,0).is_ok){return 1003;}i+=1;}i=0;while(i<ir::layout_count(p)){if(!budget::charge(work,1)){return 1002;}let cell=v::at_mut<u8>(ref mut shaped,ir::layout_at(p,i).value.type-1);*cell=1;i+=1;}
    i=1;while(i<=clean::type_count(cleaned)){if(!budget::charge(work,1)){return 1002;}let g=clean::glue_at(cleaned,i);if(g.kind==1 && *v::at<u8>(ref shaped,i-1)==0 && (g.frame!=0 || g.nonempty || g.effects!=0 || g.abilities!=0 || g.field_start!=0 || g.field_count!=0)){return 1001;}i+=1;}
    i=0;while(i<v::len<u8>(seen)){if(!budget::charge(work,1)){return 1002;}if(*v::at<u8>(seen,i)!=1){return 1001;}i+=1;}return 0;
}
// One package-owned range must cover the entire saved unit, not merely a
// matching subset. Fresh checking owns the source/Flow diagnostics; no lowering
// is needed to reject injected frames, tables, operands or cleanup authority.
pub fn Result<Covered,Problem> checked_range(ref ir::Program p,ref identities::Snapshot ids,ref source::Plan plan,ref compile::CheckedBodies checked,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    let r=compile::checked_range(checked);if(r.first>=r.stop || r.stop>v::len<syntax::Parsed>(sources)){return .err(fail(1,1001,r.first,r.stop));}
    if(!budget::charge(work,0)){return .err(fail(1,1002,r.first,r.stop));}var usize measured=0;let valid=proof::check_metered(p,ref mut measured);if(!budget::charge(work,measured)){return .err(fail(1,1002,r.first,r.stop));}if(valid.code!=0){return .err(fail(1,valid.code,r.first,r.stop));}
    measured=0;let original=identities::check(ids,ir::cleanup_ir(p),ref mut measured);if(!budget::charge(work,measured)){return .err(fail(1,1002,r.first,r.stop));}if(original!=0){return .err(fail(1,original,r.first,r.stop));}
    let d=domains(p);var v::Vec<u8> seen=v::empty<u8>();var usize i=0;while(i<d.end){if(!budget::charge(work,1)){return .err(fail(1,1002,r.first,r.stop));}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(fail(1,1003,r.first,r.stop));}i+=1;}
    let matched=switch(matching::match_unit(p,ids,compile::checked_state(checked),plan,sources,metadata,work)){case .ok(p):{p}case .err(code):{return .err(fail(3,code,r.first,r.stop));}};
    let covered=unit(p,ref matched,d,ref mut seen,work);if(covered!=0){return .err(fail(3,covered,r.first,r.stop));}let code=tables(p,d,ref mut seen,work);if(code!=0){return .err(fail(4,code,r.first,r.stop));}
    return .ok(Covered{.stats=Stats{.files=r.stop-r.first,.units=1,.retries=0,.frames=ir::count(p),.types=clean::type_count(ir::cleanup_ir(p)),.layouts=ir::layout_count(p),.fields=ir::field_count(p),.cleanup_fields=clean::field_count(ir::cleanup_ir(p)),.callables=ir::callable_count(p),.parameters=ir::parameter_count(p),.names=ir::name_bytes(p)}});
}
pub fn Result<Covered,Problem> build(ref ir::Program p,ref identities::Snapshot ids,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    if(!budget::charge(work,0)){return .err(fail(1,1002,0,0));}var usize measured=0;let valid=proof::check_metered(p,ref mut measured);if(!budget::charge(work,measured)){return .err(fail(1,1002,0,0));}if(valid.code!=0){return .err(fail(1,valid.code,0,0));}
    measured=0;let original=identities::check(ids,ir::cleanup_ir(p),ref mut measured);if(!budget::charge(work,measured)){return .err(fail(1,1002,0,0));}if(original!=0){return .err(fail(1,original,0,0));}
    let grouped=switch(grouping::build(sources)){case .ok(p):{p}case .err(e):{var problem=fail(1,e.code,0,v::len<syntax::Parsed>(sources));problem.source.module_index=e.module_index;problem.source.start=e.start;problem.source.end=e.end;return .err(problem);}};
    let d=domains(p);var v::Vec<u8> seen=v::empty<u8>();var usize i=0;while(i<d.end){if(!budget::charge(work,1)){return .err(fail(1,1002,0,0));}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(fail(1,1003,0,0));}i+=1;}var progress=Progress{.files=0,.units=0,.retries=0};i=0;
    while(i<grouping::count(ref grouped)){let u=grouping::at(ref grouped,i);let problem=append(p,ids,plan,sources,metadata,u.first,u.stop,0,d,ref mut seen,ref mut progress,work);if(problem.code!=0){return .err(problem);}i+=1;}if(progress.files!=v::len<syntax::Parsed>(sources)){return .err(fail(4,1001,0,progress.files));}
    let code=tables(p,d,ref mut seen,work);if(code!=0){return .err(fail(4,code,0,progress.files));}return .ok(Covered{.stats=Stats{.files=progress.files,.units=progress.units,.retries=progress.retries,.frames=ir::count(p),.types=clean::type_count(ir::cleanup_ir(p)),.layouts=ir::layout_count(p),.fields=ir::field_count(p),.cleanup_fields=clean::field_count(ir::cleanup_ir(p)),.callables=ir::callable_count(p),.parameters=ir::parameter_count(p),.names=ir::name_bytes(p)}});
}

module module_project_identified_compile;
import module_project_compile as compile;
import module_units as grouping;
import module_identified_ir as identified;
import module_signatures as signatures;
import generic_source_plan as source;
import syntax as syntax;
import std::vec as v;
import generic_source_plan_input as original_input;
import generic_source_catalog as templates;
import generic_package_bindings as bindings;
import project_compiler_input as project_input;
import project_cache_unit_key as cache_key;
import project_cache_unit_store as cache;
import project_cache_disk as disk;
import project_cache_environment as environment;
import module_catalog as catalog;

// Every retained unit owns its checked IR and its original-source identities.
// No resource field is moved out and no borrowed source/planner owner is stored.
pub resource Plan{v::Vec<identified::Program> units;v::Vec<compile::Range> ranges;usize files;usize retries;bindings::Bindings provenance;cleanup{}}
pub fn Plan empty(){return Plan{.units=v::empty<identified::Program>(),.ranges=v::empty<compile::Range>(),.files=0,.retries=0,.provenance=bindings::empty()};}
pub fn ref bindings::Bindings provenance(ref Plan p){return ref p.provenance;}
pub fn usize count(ref Plan p){return v::len<identified::Program>(ref p.units);}
pub fn usize file_count(ref Plan p){return p.files;}
pub fn usize retries(ref Plan p){return p.retries;}
pub fn ref identified::Program at(ref Plan p,usize index){assert(index<count(p));return v::at<identified::Program>(ref p.units,index);}
pub fn compile::Range range_at(ref Plan p,usize index){assert(index<count(p));return *v::at<compile::Range>(ref p.ranges,index);}
fn compile::Problem failed(u32 code,usize first,usize stop){return compile::Problem{.stage=1,.code,.module_index=first,.start=0,.end=0,.first,.stop,.expected=0,.actual=0,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=0,.field_start=0,.field_end=0};}
fn compile::Problem append(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref source::Plan original,usize first,usize stop,usize depth,ref mut v::Vec<identified::Program> units,ref mut v::Vec<compile::Range> ranges,ref mut usize retries,ref mut usize work){
    switch(compile::range_identified(sources,metadata,original,first,stop,work)){
        case .ok(p):{
            if(!v::push<identified::Program>(units,p).is_ok || !v::push<compile::Range>(ranges,compile::Range{.first,.stop}).is_ok){return failed(11,first,stop);}
            var compile::Problem result=failed(0,first,stop);result.stage=0;return result;
        }
        case .err(e):{
            // Checking capacity can split. Attachment failures (stage 8) never
            // retry with a reset meter or expose previously collected units.
            if(!compile::capacity_failure(e) || stop-first<=1 || depth>=8){return e;};
            *retries=*retries+1;let middle=first+(stop-first)/2;
            let left=append(sources,metadata,original,first,middle,depth+1,units,ranges,retries,work);if(left.code!=0){return left;}
            return append(sources,metadata,original,middle,stop,depth+1,units,ranges,retries,work);
        }
    };return failed(12,first,stop);
}
pub fn Result<Plan,compile::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref source::Plan original,ref mut usize work){
    let grouped=switch(grouping::build(sources)){case .ok(p):{p}case .err(e):{var compile::Problem result=failed(e.code,0,v::len<syntax::Parsed>(sources));result.module_index=e.module_index;result.start=e.start;result.end=e.end;return .err(result);}};
    var v::Vec<identified::Program> units=v::empty<identified::Program>();var v::Vec<compile::Range> ranges=v::empty<compile::Range>();var usize retries=0;var usize next=0;var usize i=0;
    while(i<grouping::count(ref grouped)){let unit=grouping::at(ref grouped,i);assert(unit.first==next && unit.first<unit.stop);let e=append(sources,metadata,original,unit.first,unit.stop,0,ref mut units,ref mut ranges,ref mut retries,work);if(e.code!=0){return .err(e);}next=unit.stop;i+=1;}
    assert(next==grouping::file_count(ref grouped) && v::len<identified::Program>(ref units)==v::len<compile::Range>(ref ranges));let stamps=switch(bindings::copy(templates::packages(original_input::templates(source::input(original))),work)){case .ok(p):{p}case .err(code):{var compile::Problem e=failed(code,0,next);e.stage=8;return .err(e);}};return .ok(Plan{.units,.ranges,.files=next,.retries,.provenance=stamps});
}

// Cacheable units never cross package ownership. Existing capacity splitting
// and whole-project relinking remain in force; no cached local ID escapes into
// the linked result without the ordinary type/frame remapping and verifier.
@effects(filesystem)
fn compile::Problem append_cached(ref disk::Store store,ref environment::Environment env,ref project_input::Input input,ref catalog::Catalog scope,ref signatures::Program metadata,ref source::Plan original,usize first,usize stop,usize depth,ref mut v::Vec<identified::Program> units,ref mut v::Vec<compile::Range> ranges,ref mut usize retries,ref mut usize work,ref mut cache::Work cache_work){
    if(v::len<identified::Program>(units)>=512){return failed(10,first,stop);}let sources=project_input::sources(input);var usize key_work=cache_work.key;let selected=cache_key::collect(input,original,env,first,stop,ref mut key_work);cache_work.key=key_work;
    switch(selected){case .ok(key):{switch(cache::load(store,ref key,sources,scope,metadata,original,first,stop,cache_work)){case .ok(p):{if(!v::push<identified::Program>(units,p).is_ok || !v::push<compile::Range>(ranges,compile::Range{.first,.stop}).is_ok){return failed(11,first,stop);}cache_work.hits+=1;var result=failed(0,first,stop);result.stage=0;return result;}case .err(_):{}};}case .err(_):{}};
    switch(compile::range_identified(sources,metadata,original,first,stop,work)){
        case .ok(p):{cache::mark_cold(cache_work,v::len<identified::Program>(units));if(!v::push<identified::Program>(units,p).is_ok || !v::push<compile::Range>(ranges,compile::Range{.first,.stop}).is_ok){return failed(11,first,stop);}cache_work.misses+=1;var result=failed(0,first,stop);result.stage=0;return result;}
        case .err(e):{if(!compile::capacity_failure(e) || stop-first<=1 || depth>=8){return e;};*retries=*retries+1;let middle=first+(stop-first)/2;let left=append_cached(store,env,input,scope,metadata,original,first,middle,depth+1,units,ranges,retries,work,cache_work);if(left.code!=0){return left;}return append_cached(store,env,input,scope,metadata,original,middle,stop,depth+1,units,ranges,retries,work,cache_work);}
    };return failed(12,first,stop);
}
@effects(filesystem)
pub fn Result<Plan,compile::Problem> build_cached(ref disk::Store store,ref environment::Environment env,ref project_input::Input input,ref catalog::Catalog scope,ref signatures::Program metadata,ref source::Plan original,ref mut usize work,ref mut cache::Work cache_work){
    let sources=project_input::sources(input);let grouped=switch(grouping::build(sources)){case .ok(p):{p}case .err(e):{var result=failed(e.code,0,project_input::count(input));result.module_index=e.module_index;result.start=e.start;result.end=e.end;return .err(result);}};
    var v::Vec<identified::Program> units=v::empty<identified::Program>();var v::Vec<compile::Range> ranges=v::empty<compile::Range>();var usize retries=0;var usize next=0;var usize i=0;
    while(i<grouping::count(ref grouped)){let grouped=grouping::at(ref grouped,i);var usize first=grouped.first;while(first<grouped.stop){let owner=project_input::origin(input,first).owner;var usize stop=first+1;while(stop<grouped.stop && project_input::origin(input,stop).owner==owner){stop+=1;}let e=append_cached(store,env,input,scope,metadata,original,first,stop,0,ref mut units,ref mut ranges,ref mut retries,work,cache_work);if(e.code!=0){return .err(e);}first=stop;}next=grouped.stop;i+=1;}
    let stamps=switch(bindings::copy(templates::packages(original_input::templates(source::input(original))),work)){case .ok(p):{p}case .err(code):{var e=failed(code,0,next);e.stage=8;return .err(e);}};return .ok(Plan{.units,.ranges,.files=next,.retries,.provenance=stamps});
}
// Publication follows successful whole-project linking and interface creation.
// A later source error therefore cannot publish a successful cache prefix.
@effects(filesystem,clock)
pub fn void publish_cached(ref disk::Store store,ref environment::Environment env,ref project_input::Input input,ref source::Plan original,ref Plan units,ref mut cache::Work work){
    var usize i=0;while(i<count(units) && i<512){if(cache::is_cold(work,i)){let r=range_at(units,i);var usize key_work=work.key;let result=cache_key::collect(input,original,env,r.first,r.stop,ref mut key_work);work.key=key_work;switch(result){case .ok(key):{if(cache::save(store,ref key,at(units,i),work)){work.stored+=1;}}case .err(_):{}};}i+=1;}
}

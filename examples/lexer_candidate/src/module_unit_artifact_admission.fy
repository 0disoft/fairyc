module module_unit_artifact_admission;
import module_program_artifact_decode as decoder;
import module_program_artifact_encode as encoder;
import candidate_artifact_wire as component;
import module_program_artifact_wire as budget;
import std::bytes as bytes;
import module_project_artifact_coverage as coverage;
import module_project_artifact_state as states;
import module_project_artifact_sdk as sdk;
import module_unit_artifact_identities as ids;
import module_identified_ir as identified;
import module_project_compile as compile;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_signatures as signatures;
import module_catalog as catalog;
import generic_source_plan as source;
import syntax as syntax;
import std::vec as v;
import module_borrow_observations as observations;
import module_artifact_borrow_observations as observation_work;

// A saved unit never skips current body/Flow checking. It can skip lowering
// only after complete source-recipe, metadata, state and SDK comparison.
// Identity attachment rejects incompatible canonical type/planner numbering;
// the cache caller treats any such rejection as a disposable miss.
pub fn Result<identified::Program,u32> restore(ref u8[] raw,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref source::Plan plan,usize first,usize stop,ref mut usize source_work,ref mut usize artifact_work){
    return restore_mode(raw,sources,scope,metadata,plan,first,stop,false,source_work,artifact_work);
}
pub fn Result<identified::Program,u32> restore_aggregates(ref u8[] raw,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref source::Plan plan,usize first,usize stop,ref mut usize source_work,ref mut usize artifact_work){
    return restore_mode(raw,sources,scope,metadata,plan,first,stop,true,source_work,artifact_work);
}
fn Result<identified::Program,u32> restore_mode(ref u8[] raw,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref source::Plan plan,usize first,usize stop,bool aggregates,ref mut usize source_work,ref mut usize artifact_work){
    let saved=switch(if(aggregates){decoder::decode_aggregates(raw,artifact_work)}else{decoder::decode(raw,artifact_work)}){case .ok(p):{p}case .err(code):{return .err(code);}};
    if(clean::project(ir::cleanup_ir(ref saved)) || clean::aggregates(ir::cleanup_ir(ref saved))!=signatures::aggregate_types(metadata)){return .err(1001);}
    let checked=switch(compile::check_range(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let current=switch(identified::prepare(saved,plan,sources,metadata,compile::checked_interfaces(ref checked),source_work)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let snapshot=switch(ids::extract_core(ref current,artifact_work)){case .ok(p):{p}case .err(code):{return .err(code);}};let program=identified::core_program(ref current);
    let covered=switch(coverage::checked_range(program,ref snapshot,plan,ref checked,sources,metadata,artifact_work)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    switch(states::build(ir::state_layouts(program),sources,scope,artifact_work)){case .ok(_):{}case .err(code):{return .err(code);}};
    switch(sdk::build(ir::sdk_units(program),sources,scope,artifact_work)){case .ok(_):{}case .err(code):{return .err(code);}};
    let rows=switch(observation_work::copy(coverage::borrows(ref covered),artifact_work)){case .ok(p):{p}case .err(code):{return .err(if(code==3){1002}else if(code==2){1003}else{1001});}};
    return switch(identified::observed(current,rows)){case .ok(p):{.ok(p)}case .err(_):{.err(1001)}};
}

// Explicit source-bound closure UNIT admission. Until the separate coverage/
// call-recipe family admits closure invocations, rebuild the current checked
// unit and compare its complete canonical recipe. Return that fresh source
// owner, never the detached artifact as source authority. No Complete/cache
// publication or project envelope is introduced by this entry point.
pub fn Result<identified::Program,u32> restore_closures(ref u8[] raw,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref source::Plan plan,usize first,usize stop,ref mut usize source_work,ref mut usize artifact_work){
    let saved=switch(decoder::decode_closures(raw,artifact_work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let cleaned=ir::cleanup_ir(ref saved);if(clean::project(cleaned) || !clean::closures(cleaned) || clean::aggregates(cleaned)!=signatures::aggregate_types(metadata)){return .err(1001);}
    let current=switch(compile::range_identified(sources,metadata,plan,first,stop,source_work)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    switch(ids::extract_closures(ref current,artifact_work)){case .ok(_):{}case .err(code):{return .err(code);}};let program=identified::ir_program(ref current);
    if(clean::project(ir::cleanup_ir(program)) || !clean::closures(ir::cleanup_ir(program))){return .err(1001);}
    // Identity validation above uses the fresh owner and current source plan;
    // no generic planner handle is borrowed from the detached artifact.
    let recipe=switch(encoder::encode_closures(program,artifact_work)){case .ok(p):{p}case .err(code):{return .err(code);}};let expected=component::content(ref recipe);
    if(!budget::charge(artifact_work,raw.len+1)){return .err(1002);}if(!bytes::equal(raw,expected)){return .err(1001);}
    switch(states::build(ir::state_layouts(program),sources,scope,artifact_work)){case .ok(_):{}case .err(code):{return .err(code);}};
    switch(sdk::build(ir::sdk_units(program),sources,scope,artifact_work)){case .ok(_):{}case .err(code):{return .err(code);}};
    return .ok(current);
}

module module_unit_artifact_admission;
import module_program_artifact_decode as decoder;
import module_project_artifact_coverage as coverage;
import module_project_artifact_state as states;
import module_project_artifact_sdk as sdk;
import module_unit_artifact_identities as ids;
import module_identified_ir as identified;
import module_project_compile as compile;
import module_unit_body_ir_model as ir;
import module_signatures as signatures;
import module_catalog as catalog;
import generic_source_plan as source;
import syntax as syntax;
import std::vec as v;

// A saved unit never skips current body/Flow checking. It can skip lowering
// only after complete source-recipe, metadata, state and SDK comparison.
// Identity attachment rejects incompatible canonical type/planner numbering;
// the cache caller treats any such rejection as a disposable miss.
pub fn Result<identified::Program,u32> restore(ref u8[] raw,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref source::Plan plan,usize first,usize stop,ref mut usize source_work,ref mut usize artifact_work){
    let saved=switch(decoder::decode(raw,artifact_work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let checked=switch(compile::check_range(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let current=switch(identified::build(saved,plan,sources,metadata,compile::checked_interfaces(ref checked),source_work)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let snapshot=switch(ids::extract(ref current,artifact_work)){case .ok(p):{p}case .err(code):{return .err(code);}};let program=identified::ir_program(ref current);
    switch(coverage::checked_range(program,ref snapshot,plan,ref checked,sources,metadata,artifact_work)){case .ok(_):{}case .err(e):{return .err(e.code);}};
    switch(states::build(ir::state_layouts(program),sources,scope,artifact_work)){case .ok(_):{}case .err(code):{return .err(code);}};
    switch(sdk::build(ir::sdk_units(program),sources,scope,artifact_work)){case .ok(_):{}case .err(code):{return .err(code);}};
    return .ok(current);
}

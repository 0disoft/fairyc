module module_project_artifact_admission;
import module_project_artifact_current as current;
import module_project_artifact_coverage as coverage;
import module_project_artifact_state as states;
import module_project_artifact_sdk as sdk;
import module_identity_artifact_model as identities;
import module_source_specializations as selection;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_project_compile as compile;
import module_source_provenance as provenance;
import generic_package_bindings as bindings;
import module_catalog as catalog;
import module_signatures as signatures;
import module_public_type_text as coordinates;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Problem {u32 stage;u32 code;compile::Problem source;}
pub struct Stats {coverage::Stats coverage;usize states;usize sdk_units;usize sdk_bytes;}
// One retained IR owner passes every source/body/state/SDK gate before escaping.
// It is still separate from Complete's public interfaces and cache publication.
pub resource Admitted {ir::Program program;identities::Snapshot identities;bindings::Bindings provenance;Stats stats;cleanup{}}
pub fn ref ir::Program program(ref Admitted p){return ref p.program;}
pub fn ref identities::Snapshot source_identities(ref Admitted p){return ref p.identities;}
pub fn ref bindings::Bindings source_provenance(ref Admitted p){return ref p.provenance;}
pub fn Stats stats(ref Admitted p){return p.stats;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code,.source=compile::Problem{.stage=0,.code=0,.module_index=0,.start=0,.end=0,.first=0,.stop=0,.expected=0,.actual=0,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=0,.field_start=0,.field_end=0}};}
fn u32 translated(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn Result<identities::Snapshot,u32> copy_ids(ref identities::Snapshot input,ref clean::Program context,ref mut usize work){
    let raw=identities::names(input);if(!budget::charge(work,raw.len+1)){return .err(1002);}let names=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();var usize i=1;
    while(i<=identities::type_count(input)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<selection::Row>(ref mut types,identities::type_at(input,i)).is_ok){return .err(1003);}i+=1;}i=1;
    while(i<=identities::frame_count(input)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<selection::Row>(ref mut frames,identities::frame_at(input,i)).is_ok){return .err(1003);}i+=1;}
    var usize measured=0;let result=switch(identities::assemble(names,types,frames,context,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(1002);}return .err(code);}};if(!budget::charge(work,measured)){return .err(1002);}return .ok(result);
}
// The catalog and signatures come from current sources; selected supplies freshly
// rederived original-template handles, never cached ordinals or SDK booleans.
// Independent phase histories fold once into the caller's aggregate ledger.
pub fn Result<Admitted,Problem> build(ir::Program program,ref current::Rebound selected,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){
    if(!budget::charge(work,0)){return .err(fail(1,1002));}let count=v::len<syntax::Parsed>(sources);if(catalog::module_count(scope)!=count || signatures::module_count(metadata)!=count){return .err(fail(1,1001));}var usize i=0;
    while(i<count){let parsed=v::at<syntax::Parsed>(sources,i);if(!budget::charge(work,syntax::source_view(parsed).len+1)){return .err(fail(1,1002));}if(catalog::module_owner(scope,i)!=signatures::module_owner(metadata,i) || !catalog::source_matches(scope,i,parsed)){return .err(fail(1,1001));}i+=1;}
    var usize measured=0;let fresh=provenance::current(current::provenance(selected),sources,metadata,table,ref mut measured);if(!budget::charge(work,measured)){return .err(fail(1,1002));}if(fresh!=0){return .err(fail(1,translated(fresh)));}
    let covered=switch(coverage::build(ref program,current::source_identities(selected),current::source_plan(selected),sources,metadata,work)){case .ok(p):{p}case .err(e):{return .err(Problem{.stage=2,.code=e.code,.source=e.source});}};
    let state=switch(states::build(ir::state_layouts(ref program),sources,scope,work)){case .ok(p):{p}case .err(code):{return .err(fail(3,code));}};
    let sdk=switch(sdk::build(ir::sdk_units(ref program),sources,scope,work)){case .ok(p):{p}case .err(code):{return .err(fail(4,code));}};
    let ids=switch(copy_ids(current::source_identities(selected),ir::cleanup_ir(ref program),work)){case .ok(p):{p}case .err(code):{return .err(fail(5,code));}};
    measured=0;let stamps=switch(bindings::copy(current::provenance(selected),ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(fail(5,1002));}return .err(fail(5,translated(code)));}};if(!budget::charge(work,measured)){return .err(fail(5,1002));}
    let sdk_stats=sdk::stats(ref sdk);let stats=Stats{.coverage=coverage::stats(ref covered),.states=states::count(ref state),.sdk_units=sdk_stats.units,.sdk_bytes=sdk_stats.bytes};
    if(!budget::charge(work,1)){return .err(fail(6,1002));}return .ok(Admitted{.program,.identities=ids,.provenance=stamps,.stats});
}

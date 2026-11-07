module project_compiler_core;
import project_compiler_input as input;
import project_compiler_state_graph as state_graph;
import module_project_artifact_model as artifacts;
import module_project_artifact_current as rebinding;
import module_project_artifact_rebase as rebasing;
import module_project_artifact_admission as admission;
import module_signatures as signatures;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import syntax as syntax;
import module_public_type_text as coordinates;
import project_source_plan as source_plan;
import module_project_identified_compile as identified_units;
import module_project_identified_ir as identified;
import module_project_interfaces as interfaces;
import generic_source_plan as original_plan;
import generic_aggregate_types as aggregate_types;
import generic_signature_diagnostics as diagnostics;
import std::vec as v;
import module_catalog as modules;
import project_cache_unit_store as unit_cache;
import project_cache_disk as disk;
import project_cache_environment as environment;

// 1 declaration catalog, 2 signatures, 3 checked units, 4 project link,
// Stage/expected/actual retain the original semantic failure.
pub struct Problem{u32 phase;u32 stage;u32 code;usize module_index;usize start;usize end;usize expected;usize actual;usize declaration_start;usize declaration_end;usize move_start;usize move_end;usize header_start;usize header_end;usize field_module;usize field_start;usize field_end;}
pub fn Problem empty(){return fail(0,0,0,0,0,0,0,0);}
fn Problem fail(u32 phase,u32 stage,u32 code,usize module_index,usize start,usize end,usize expected,usize actual){return Problem{.phase,.stage,.code,.module_index,.start,.end,.expected,.actual,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=0,.field_start=0,.field_end=0};}
pub fn bool has_location(ref input::Input sources,Problem p){return p.phase<4 && p.module_index<input::count(sources) && p.start<=p.end && p.end<=syntax::source_view(input::source(sources,p.module_index)).len;}
// The caller retains the exact source map across errors and later C emission.
// The successful IR is a separate whole owner, with no borrowed AST arenas.
pub fn Result<ir::Program,Problem> build(ref input::Input sources){
    let catalog=switch(input::declarations(sources)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_catalog(input::sources(sources),catalog)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let units=switch(compile::build(input::sources(sources),ref metadata)){case .ok(p):{p}case .err(e):{var Problem p=fail(3,e.stage,e.code,e.module_index,e.start,e.end,e.expected,e.actual);p.declaration_start=e.declaration_start;p.declaration_end=e.declaration_end;p.move_start=e.move_start;p.move_end=e.move_end;p.header_start=e.header_start;p.header_end=e.header_end;p.field_module=e.field_module;p.field_start=e.field_start;p.field_end=e.field_end;return .err(p);}};
    return switch(link::build(ref metadata,compile::programs(ref units))){case .ok(p):{.ok(p)}case .err(e):{.err(fail(4,0,e.code,e.module_index,e.start,e.end,0,0))}};
}

// Phase 5 owns source-plan failures. Source planning/attachment keeps its 16MiB
// history; project linking keeps the existing independent 16MiB history. A
// successful owner is published only after all body/Flow checks and rebinding.
pub fn Result<identified::Program,Problem> build_identified(ref input::Input sources,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize link_work){return build_identified_sources(input::sources(sources),input::owners(sources),input::bindings(sources),table,source_work,link_work);}
pub fn Result<identified::Program,Problem> build_identified_sources(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize link_work){
    let catalog=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_catalog(sources,catalog)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let original=switch(source_plan::build_sources(sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.stage,e.code,e.module_index,e.start,e.end,0,0));}};
    return link_identified(sources,ref metadata,ref original,source_work,link_work);
}
fn Result<identified::Program,Problem> link_identified(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref original_plan::Plan original,ref mut usize source_work,ref mut usize link_work){
    let units=switch(identified_units::build(sources,metadata,original,source_work)){case .ok(p):{p}case .err(e):{var Problem p=fail(3,e.stage,e.code,e.module_index,e.start,e.end,e.expected,e.actual);p.declaration_start=e.declaration_start;p.declaration_end=e.declaration_end;p.move_start=e.move_start;p.move_end=e.move_end;p.header_start=e.header_start;p.header_end=e.header_end;p.field_module=e.field_module;p.field_start=e.field_start;p.field_end=e.field_end;return .err(p);}};
    return switch(identified::build(metadata,ref units,link_work)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(4,0,e.code,e.module_index,e.start,e.end,0,0))}};
}

pub resource Complete{v::Vec<identified::Program> program;v::Vec<interfaces::Records> interfaces;v::Vec<state_graph::Complete> state_graphs;cleanup{}}
pub fn bool is_state_graph(ref Complete p){return v::len<state_graph::Complete>(ref p.state_graphs)==1;}
// The explicit graph profile has its own verified IR, never an empty ordinary body.
pub fn ref identified::Program checked_program(ref Complete p){assert(!is_state_graph(p));return v::at<identified::Program>(ref p.program,0);}
pub fn ref state_graph::Complete checked_state_graph(ref Complete p){assert(is_state_graph(p));return v::at<state_graph::Complete>(ref p.state_graphs,0);}
pub fn ref interfaces::Records public_interfaces(ref Complete p){return if(is_state_graph(p)){state_graph::public_interfaces(checked_state_graph(p))}else{v::at<interfaces::Records>(ref p.interfaces,0)};}
fn Result<Complete,Problem> ordinary_complete(identified::Program program,interfaces::Records interfaces){var programs=v::empty<identified::Program>();var records=v::empty<interfaces::Records>();if(!v::push<identified::Program>(ref mut programs,program).is_ok){return .err(fail(6,0,11,0,0,0,0,0));}if(!v::push<interfaces::Records>(ref mut records,interfaces).is_ok){return .err(fail(6,0,11,0,0,0,0,0));}return .ok(Complete{.program=programs,.interfaces=records,.state_graphs=v::empty<state_graph::Complete>()});}
fn Result<Complete,Problem> graph_complete(state_graph::Complete graph){var graphs=v::empty<state_graph::Complete>();if(!v::push<state_graph::Complete>(ref mut graphs,graph).is_ok){return .err(fail(8,3,1003,0,0,0,0,0));}return .ok(Complete{.program=v::empty<identified::Program>(),.interfaces=v::empty<interfaces::Records>(),.state_graphs=graphs});}
pub fn Result<Complete,Problem> build_complete_state_graph(ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref mut usize work){return switch(state_graph::build(actual,table,owner,work)){case .ok(p):{graph_complete(p)}case .err(code):{.err(fail(8,1,code,0,0,0,0,0))}};}
pub fn Result<Complete,Problem> restore_complete_state_graph(ref u8[] saved,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref mut usize work){return switch(state_graph::restore(saved,actual,table,owner,work)){case .ok(p):{graph_complete(p)}case .err(code):{.err(fail(8,2,code,0,0,0,0,0))}};}
// Phase 6 rejects interface publication. Planning, body/Flow verification,
// rebinding and all package interfaces must finish before either member escapes.
pub fn Result<Complete,Problem> build_complete(ref input::Input sources,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize link_work){return build_complete_sources(input::sources(sources),input::owners(sources),input::bindings(sources),table,source_work,link_work);}
pub fn Result<Complete,Problem> build_complete_sources(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize link_work){
    var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();
    return build_complete_reporting_sources(sources,owners,imports,table,ref mut reports,source_work,link_work);
}
// Explicit private aggregate route: the same Complete owner is published only
// after original-source planning, concrete checking, linking and all interfaces.
// Ordinary/cache restoration entry points keep their existing admission policy.
pub fn Result<Complete,Problem> build_complete_aggregate_sources(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize link_work){
    let plan=switch(source_plan::build_aggregate_sources(sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.stage,e.code,e.module_index,e.start,e.end,0,0));}};
    let prepared=switch(aggregate_types::build(plan,sources,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_aggregates(sources,prepared,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let original=aggregate_types::source_plan(signatures::aggregate_source(ref metadata));
    let program=switch(link_identified(sources,ref metadata,original,source_work,link_work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let interfaces=switch(interfaces::build(ref program,original,sources,ref metadata,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(6,0,e,0,0,0,0,0));}};
    return ordinary_complete(program,interfaces);
}
// An initially empty error-only sink receives one detached report owner only
// for an unsupported signature. Normal and other failed builds leave it empty.
pub fn Result<Complete,Problem> build_complete_reporting(ref input::Input sources,ref coordinates::Coordinates table,ref mut v::Vec<diagnostics::Reports> reports,ref mut usize source_work,ref mut usize link_work){return build_complete_reporting_sources(input::sources(sources),input::owners(sources),input::bindings(sources),table,reports,source_work,link_work);}
pub fn Result<Complete,Problem> build_complete_reporting_sources(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut v::Vec<diagnostics::Reports> reports,ref mut usize source_work,ref mut usize link_work){
    if(v::len<diagnostics::Reports>(reports)!=0){return .err(fail(5,6,9,0,0,0,0,0));}
    let catalog=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_catalog(sources,catalog)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let original=switch(source_plan::build_sources(sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.stage,e.code,e.module_index,e.start,e.end,0,0));}};
    let unsupported=switch(diagnostics::collect(ref original,sources,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(5,6,code,0,0,0,0,0));}};
    if(diagnostics::count(ref unsupported)!=0){let first=diagnostics::at(ref unsupported,0);if(!v::push<diagnostics::Reports>(reports,unsupported).is_ok){return .err(fail(5,6,11,0,0,0,0,0));}return .err(fail(5,6,4505,first.module_index,first.start,first.end,0,0));}
    let program=switch(link_identified(sources,ref metadata,ref original,source_work,link_work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let interfaces=switch(interfaces::build(ref program,ref original,sources,ref metadata,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(6,0,e,0,0,0,0,0));}};
    return ordinary_complete(program,interfaces);
}
// Package units are admitted or cold-lowered independently, then passed through
// the same whole-project linker and interface factory. No unit is published
// before every source and the complete linked owner have succeeded.
@effects(filesystem,clock)
pub fn Result<Complete,Problem> build_complete_mixed(ref disk::Store store,ref environment::Environment env,ref input::Input input,ref coordinates::Coordinates table,ref mut v::Vec<diagnostics::Reports> reports,ref mut unit_cache::Work cache_work,ref mut usize source_work,ref mut usize link_work){
    if(v::len<diagnostics::Reports>(reports)!=0){return .err(fail(5,6,9,0,0,0,0,0));}let sources=input::sources(input);
    let catalog=switch(modules::build_scoped(sources,input::owners(input),input::bindings(input))){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let scope=switch(modules::build_scoped(sources,input::owners(input),input::bindings(input))){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_catalog(sources,catalog)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let original=switch(source_plan::build(input,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.stage,e.code,e.module_index,e.start,e.end,0,0));}};
    let unsupported=switch(diagnostics::collect(ref original,sources,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(5,6,code,0,0,0,0,0));}};
    if(diagnostics::count(ref unsupported)!=0){let first=diagnostics::at(ref unsupported,0);if(!v::push<diagnostics::Reports>(reports,unsupported).is_ok){return .err(fail(5,6,11,0,0,0,0,0));}return .err(fail(5,6,4505,first.module_index,first.start,first.end,0,0));}
    let units=switch(identified_units::build_cached(store,env,input,ref scope,ref metadata,ref original,source_work,cache_work)){case .ok(p):{p}case .err(e):{var p=fail(3,e.stage,e.code,e.module_index,e.start,e.end,e.expected,e.actual);p.declaration_start=e.declaration_start;p.declaration_end=e.declaration_end;p.move_start=e.move_start;p.move_end=e.move_end;p.header_start=e.header_start;p.header_end=e.header_end;p.field_module=e.field_module;p.field_start=e.field_start;p.field_end=e.field_end;return .err(p);}};
    let program=switch(identified::build(ref metadata,ref units,link_work)){case .ok(p):{p}case .err(e):{return .err(fail(4,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let interfaces=switch(interfaces::build(ref program,ref original,sources,ref metadata,table,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(6,0,code,0,0,0,0,0));}};
    let complete=switch(ordinary_complete(program,interfaces)){case .ok(p):{p}case .err(e):{return .err(e);}};identified_units::publish_cached(store,env,input,ref original,ref units,cache_work);return .ok(complete);
}
// Explicit aggregate unit-cache composition. Reuse still runs fresh checking;
// no successful unit prefix is published before linked Complete/interfaces.
@effects(filesystem,clock)
pub fn Result<Complete,Problem> build_complete_aggregate_mixed(ref disk::Store store,ref environment::Environment env,ref input::Input input,ref coordinates::Coordinates table,ref mut unit_cache::Work cache_work,ref mut usize source_work,ref mut usize link_work){
    let sources=input::sources(input);let scope=switch(modules::build_scoped(sources,input::owners(input),input::bindings(input))){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let plan=switch(source_plan::build_aggregate_sources(sources,input::owners(input),input::bindings(input),table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.stage,e.code,e.module_index,e.start,e.end,0,0));}};
    let prepared=switch(aggregate_types::build(plan,sources,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_aggregates(sources,prepared,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let original=aggregate_types::source_plan(signatures::aggregate_source(ref metadata));
    let units=switch(identified_units::build_cached(store,env,input,ref scope,ref metadata,original,source_work,cache_work)){case .ok(p):{p}case .err(e):{var p=fail(3,e.stage,e.code,e.module_index,e.start,e.end,e.expected,e.actual);p.declaration_start=e.declaration_start;p.declaration_end=e.declaration_end;p.move_start=e.move_start;p.move_end=e.move_end;p.header_start=e.header_start;p.header_end=e.header_end;p.field_module=e.field_module;p.field_start=e.field_start;p.field_end=e.field_end;return .err(p);}};
    let program=switch(identified::build(ref metadata,ref units,link_work)){case .ok(p):{p}case .err(e):{return .err(fail(4,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let interfaces=switch(interfaces::build(ref program,original,sources,ref metadata,table,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(6,0,code,0,0,0,0,0));}};
    let complete=switch(ordinary_complete(program,interfaces)){case .ok(p):{p}case .err(e):{return .err(e);}};identified_units::publish_cached_aggregates(store,env,input,original,ref units,cache_work);return .ok(complete);
}
// Restore keeps independent 16MiB source, aggregate artifact and 16MiB link
// histories. Neither IR nor interfaces escape until their same-owner result is complete.
pub fn Result<Complete,Problem> restore_complete_sources(artifacts::Restored saved,ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize artifact_work,ref mut usize link_work){
    let current=switch(rebinding::build_sources(saved,sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(7,1,code,0,0,0,0,0));}};
    let scope=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata_scope=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};let metadata=switch(signatures::build_catalog(sources,metadata_scope)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    return restore_current(current,sources,ref scope,ref metadata,table,source_work,artifact_work,link_work);
}
// Aggregate transport is admitted only through this explicit current-source
// factory. The metadata owner independently retains its freshly prepared plan;
// both planning histories accrue to the same bounded source ledger.
pub fn Result<Complete,Problem> restore_complete_aggregate_sources(artifacts::Restored saved,ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize artifact_work,ref mut usize link_work){
    let current=switch(rebinding::build_aggregate_sources(saved,sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(7,1,code,0,0,0,0,0));}};
    let scope=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let plan=switch(source_plan::build_aggregate_sources(sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.stage,e.code,e.module_index,e.start,e.end,0,0));}};
    let prepared=switch(aggregate_types::build(plan,sources,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_aggregates(sources,prepared,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    return restore_current(current,sources,ref scope,ref metadata,table,source_work,artifact_work,link_work);
}
// Closure restore uses current source rebind, full recipes and fresh proof of
// the saved graph. It does not replace a cache hit with cold lowering.
pub fn Result<Complete,Problem> restore_complete_closure_sources(artifacts::Restored saved,ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,bool aggregates,ref mut usize source_work,ref mut usize artifact_work,ref mut usize link_work){
    let current=switch(rebinding::build_closure_sources(saved,sources,owners,imports,table,aggregates,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(7,1,code,0,0,0,0,0));}};
    let scope=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(if(aggregates){closure_aggregate_metadata(sources,owners,imports,table,source_work)}else{closure_scalar_metadata(sources,owners,imports)}){case .ok(p):{p}case .err(e):{return .err(e);}};
    return restore_current(current,sources,ref scope,ref metadata,table,source_work,artifact_work,link_work);
}
fn Result<signatures::Program,Problem> closure_scalar_metadata(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports){
    let scope=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    return switch(signatures::build_catalog(sources,scope)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0))}};
}
fn Result<signatures::Program,Problem> closure_aggregate_metadata(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work){
    let plan=switch(source_plan::build_aggregate_sources(sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.stage,e.code,e.module_index,e.start,e.end,0,0));}};
    let prepared=switch(aggregate_types::build(plan,sources,source_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    return switch(signatures::build_aggregates(sources,prepared,source_work)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0))}};
}
fn Result<Complete,Problem> restore_current(rebinding::Rebound current,ref v::Vec<syntax::Parsed> sources,ref modules::Catalog scope,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize artifact_work,ref mut usize link_work){
    let raw=switch(rebasing::build(ref current,artifact_work)){case .ok(p):{p}case .err(code):{return .err(fail(7,2,code,0,0,0,0,0));}};
    let admitted=switch(admission::build(raw,ref current,sources,scope,metadata,table,artifact_work)){case .ok(p):{p}case .err(e):{if(e.source.stage!=0){var Problem p=fail(3,e.source.stage,e.source.code,e.source.module_index,e.source.start,e.source.end,e.source.expected,e.source.actual);p.declaration_start=e.source.declaration_start;p.declaration_end=e.source.declaration_end;p.move_start=e.source.move_start;p.move_end=e.source.move_end;p.header_start=e.source.header_start;p.header_end=e.source.header_end;p.field_module=e.source.field_module;p.field_start=e.source.field_start;p.field_end=e.source.field_end;return .err(p);}return .err(fail(7,2+e.stage,e.code,0,0,0,0,0));}};
    let interfaces=switch(interfaces::build_admitted(ref admitted,rebinding::source_plan(ref current),sources,metadata,table,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(6,0,code,0,0,0,0,0));}};
    let program=switch(identified::from_admitted(admitted,link_work)){case .ok(p):{p}case .err(e):{return .err(fail(4,0,e.code,e.module_index,e.start,e.end,0,0));}};return ordinary_complete(program,interfaces);
}

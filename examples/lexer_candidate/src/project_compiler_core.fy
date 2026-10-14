module project_compiler_core;
import project_compiler_input as input;
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
import generic_signature_diagnostics as diagnostics;
import std::vec as v;
import module_catalog as modules;

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
// history; project linking keeps the existing independent 8MiB history. A
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

pub resource Complete{identified::Program program;interfaces::Records interfaces;cleanup{}}
pub fn ref identified::Program checked_program(ref Complete p){return ref p.program;}
pub fn ref interfaces::Records public_interfaces(ref Complete p){return ref p.interfaces;}
// Phase 6 rejects interface publication. Planning, body/Flow verification,
// rebinding and all package interfaces must finish before either member escapes.
pub fn Result<Complete,Problem> build_complete(ref input::Input sources,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize link_work){return build_complete_sources(input::sources(sources),input::owners(sources),input::bindings(sources),table,source_work,link_work);}
pub fn Result<Complete,Problem> build_complete_sources(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize link_work){
    var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();
    return build_complete_reporting_sources(sources,owners,imports,table,ref mut reports,source_work,link_work);
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
    return .ok(Complete{.program,.interfaces});
}
// Restore keeps independent 16MiB source, aggregate artifact and 8MiB link
// histories. Neither IR nor interfaces escape until their same-owner result is complete.
pub fn Result<Complete,Problem> restore_complete_sources(artifacts::Restored saved,ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<modules::Binding> imports,ref coordinates::Coordinates table,ref mut usize source_work,ref mut usize artifact_work,ref mut usize link_work){
    let current=switch(rebinding::build_sources(saved,sources,owners,imports,table,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(7,1,code,0,0,0,0,0));}};
    let scope=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata_scope=switch(modules::build_scoped(sources,owners,imports)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};let metadata=switch(signatures::build_catalog(sources,metadata_scope)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let raw=switch(rebasing::build(ref current,artifact_work)){case .ok(p):{p}case .err(code):{return .err(fail(7,2,code,0,0,0,0,0));}};
    let admitted=switch(admission::build(raw,ref current,sources,ref scope,ref metadata,table,artifact_work)){case .ok(p):{p}case .err(e):{if(e.source.stage!=0){var Problem p=fail(3,e.source.stage,e.source.code,e.source.module_index,e.source.start,e.source.end,e.source.expected,e.source.actual);p.declaration_start=e.source.declaration_start;p.declaration_end=e.source.declaration_end;p.move_start=e.source.move_start;p.move_end=e.source.move_end;p.header_start=e.source.header_start;p.header_end=e.source.header_end;p.field_module=e.source.field_module;p.field_start=e.source.field_start;p.field_end=e.source.field_end;return .err(p);}return .err(fail(7,2+e.stage,e.code,0,0,0,0,0));}};
    let interfaces=switch(interfaces::build_admitted(ref admitted,rebinding::source_plan(ref current),sources,ref metadata,table,source_work)){case .ok(p):{p}case .err(code):{return .err(fail(6,0,code,0,0,0,0,0));}};
    let program=switch(identified::from_admitted(admitted,link_work)){case .ok(p):{p}case .err(e):{return .err(fail(4,0,e.code,e.module_index,e.start,e.end,0,0));}};return .ok(Complete{.program,.interfaces});
}

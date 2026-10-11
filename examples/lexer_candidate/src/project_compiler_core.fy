module project_compiler_core;
import project_compiler_input as input;
import module_signatures as signatures;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import syntax as syntax;

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

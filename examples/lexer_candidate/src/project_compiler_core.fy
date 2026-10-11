module project_compiler_core;
import project_compiler_input as input;
import module_signatures as signatures;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import syntax as syntax;

// 1 declaration catalog, 2 signatures, 3 checked units, 4 project link,
// Stage/expected/actual retain the original semantic failure.
pub struct Problem{u32 phase;u32 stage;u32 code;usize module_index;usize start;usize end;usize expected;usize actual;}
fn Problem fail(u32 phase,u32 stage,u32 code,usize module_index,usize start,usize end,usize expected,usize actual){return Problem{.phase,.stage,.code,.module_index,.start,.end,.expected,.actual};}
pub fn bool has_location(ref input::Input sources,Problem p){return p.phase<4 && p.module_index<input::count(sources) && p.start<=p.end && p.end<=syntax::source_view(input::source(sources,p.module_index)).len;}
// The caller retains the exact source map across errors and later C emission.
// The successful IR is a separate whole owner, with no borrowed AST arenas.
pub fn Result<ir::Program,Problem> build(ref input::Input sources){
    let catalog=switch(input::declarations(sources)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let metadata=switch(signatures::build_catalog(input::sources(sources),catalog)){case .ok(p):{p}case .err(e):{return .err(fail(2,0,e.code,e.module_index,e.start,e.end,0,0));}};
    let units=switch(compile::build(input::sources(sources),ref metadata)){case .ok(p):{p}case .err(e):{return .err(fail(3,e.stage,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    return switch(link::build(ref metadata,compile::programs(ref units))){case .ok(p):{.ok(p)}case .err(e):{.err(fail(4,0,e.code,e.module_index,e.start,e.end,0,0))}};
}

module module_program;
import module_ast_join as join;
import module_bind as binding;
import module_signatures as signatures;
import name_resolution as names;
import type_check as typing;
import hir_lower as hir;
import mir_model as mir;
import mir_lower as lower;
import syntax as syntax;
import std::vec as v;
import sdk_ast_compact as compact;
pub struct Problem {u32 phase;u32 code;usize module_index;usize start;usize end;}
fn Problem location(ref v::Vec<syntax::Parsed> sources,u32 phase,u32 code,usize start,usize end){var usize base=0;var usize i=0;while(i<v::len<syntax::Parsed>(sources)){let usize stop=base+syntax::source_view(v::at<syntax::Parsed>(sources,i)).len;if(start<=stop){return Problem{.phase,.code,.module_index=i,.start=start-base,.end=if(end>stop){stop-base}else{end-base}};}base=stop+1;i+=1;}return Problem{.phase,.code,.module_index=0,.start=0,.end=0};}
pub fn Result<mir::Program,Problem> compile(ref v::Vec<syntax::Parsed> sources){
    let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(Problem{.phase=1,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    let combined=switch(join::join(sources)){case .ok(p):{p}case .err(p):{return .err(Problem{.phase=2,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    let parsed=switch(binding::bind(sources,ref metadata,combined)){case .ok(p):{p}case .err(p):{return .err(location(sources,3,p.code,p.start,p.end));}};
    let resolved=names::resolve_program(parsed);let name_problem=names::problem(ref resolved);if(name_problem.code!=0){return .err(location(sources,4,name_problem.code,name_problem.start,name_problem.end));}
    let typed=typing::check(resolved);let type_problem=typing::problem(ref typed);if(type_problem.code!=0){return .err(location(sources,5,type_problem.code,type_problem.start,type_problem.end));}
    let owned=switch(hir::lower(typed,0)){case .ok(p):{p}case .err(p):{return .err(location(sources,6,p.code,p.start,p.end));}};
    return switch(lower::lower(owned)){case .ok(p):{.ok(p)}case .err(p):{.err(location(sources,7,p.code,p.start,p.end))}};
}

pub fn Result<mir::Program,Problem> compile_sdk(ref mut v::Vec<syntax::Parsed> sources){
    let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(Problem{.phase=1,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    switch(compact::compact_in_place(sources,ref metadata)){case .ok(_):{}case .err(p):{return .err(Problem{.phase=8,.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end});}};
    return compile(sources);
}

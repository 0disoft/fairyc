module project_source_plan;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import generic_package_bindings as bindings;
import generic_source_catalog as templates;
import generic_source_requests as requests;
import generic_source_plan_input as prepared;
import generic_source_plan as source;
import generic_package_context as context;
import project_path as budget;
import syntax as syntax;

// One current project input owns the source order, package owners and resolved
// imports. No caller-created template/plan can replace any of those facts.
pub struct Problem{u32 stage;u32 code;usize module_index;usize start;usize end;}
fn Problem fail(ref input::Input sources,u32 stage,u32 code,usize module_index,usize declaration){var usize start=0;var usize end=0;if(module_index<input::count(sources) && declaration<syntax::declaration_count(input::source(sources,module_index))){let d=syntax::declaration_at(input::source(sources,module_index),declaration);start=d.start;end=d.end;}return Problem{.stage,.code,.module_index,.start,.end};}
pub fn Result<source::Plan,Problem> build(ref input::Input sources,ref coordinates::Coordinates table,ref mut usize work){
    if(!budget::charge(work,0)){return .err(fail(sources,0,10,0,0));}
    let bound=switch(bindings::build(input::sources(sources),input::owners(sources),input::bindings(sources),table,work)){case .ok(p):{p}case .err(e):{return .err(fail(sources,1,e.code,e.module_index,e.declaration));}};
    let catalog=switch(templates::build(bound,input::sources(sources),work)){case .ok(p):{p}case .err(e):{return .err(fail(sources,2,e.code,e.module_index,e.declaration));}};
    let roots=switch(requests::build(catalog,input::sources(sources),work)){case .ok(p):{p}case .err(e):{return .err(Problem{.stage=3,.code=e.code,.module_index=e.module_index,.start=e.start,.end=e.end});}};
    var context::Budget scoped=context::empty();scoped.work=*work;
    let original=switch(prepared::build(roots,input::sources(sources),ref mut scoped)){case .ok(p):{p}case .err(e):{*work=scoped.work;return .err(Problem{.stage=4,.code=e.source.code,.module_index=e.source.module_index,.start=e.source.start,.end=e.source.end});}};*work=scoped.work;
    return switch(source::build(original,source::defaults(),work)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(sources,5,e.code,0,0))}};
}

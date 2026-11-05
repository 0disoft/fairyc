module generic_aggregate_types;
import generic_source_plan as plan;
import generic_source_plan_input as input;
import generic_source_catalog as templates;
import generic_concrete_types as concrete;
import generic_package_bindings as bindings;
import module_catalog as catalog;
import module_types as types;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Concrete type preparation owns the exact original plan. It is not Complete:
// ordinary type assembly still refuses the new aggregate/tuple node kinds.
pub struct Instance{usize node;usize type;usize module_index;usize declaration;usize first;usize count;}
pub struct Field{usize type;usize source_index;usize start;usize end;}
// kind 1: result, 2: written parameter/local binding, 3: cast, 4: constructor.
// node is zero for ordinary source, otherwise the exact plan specialization.
pub struct Use{u32 kind;usize node;usize module_index;usize declaration;usize source_index;usize type;}
pub resource Prepared{plan::Plan plan;catalog::Catalog catalog;v::Vec<types::Type> graph;v::Vec<Instance> instances;v::Vec<Field> fields;v::Vec<Use> uses;cleanup{}}
pub struct Problem{u32 code;usize module_index;usize start;usize end;}
pub fn ref plan::Plan source_plan(ref Prepared p){return ref p.plan;}
pub fn ref catalog::Catalog declarations(ref Prepared p){return ref p.catalog;}
pub fn ref v::Vec<types::Type> type_records(ref Prepared p){return ref p.graph;}
pub fn usize count(ref Prepared p){return v::len<Instance>(ref p.instances);}
pub fn Instance at(ref Prepared p,usize i){return *v::at<Instance>(ref p.instances,i);}
pub fn Field field_at(ref Prepared p,usize i){return *v::at<Field>(ref p.fields,i);}
pub fn usize use_count(ref Prepared p){return v::len<Use>(ref p.uses);}
pub fn Use use_at(ref Prepared p,usize i){return *v::at<Use>(ref p.uses,i);}
fn Problem fail(u32 code,usize module_index,usize start,usize end){return Problem{.code,.module_index,.start,.end};}
fn Problem problem(types::Problem p){return fail(p.code,p.module_index,p.start,p.end);}
fn Result<v::Vec<usize>,u32> arguments(ref plan::Plan p,plan::Node n,ref mut v::Vec<types::Type> graph){
    var out=v::empty<usize>();var usize i=0;let c=input::types(plan::input(p));
    while(i<n.count){let t=concrete::at(c,n.arguments[i]-1);if(t.kind!=1){return .err(4502);}let id=types::canonical(graph,types::Type{.kind=1,.scalar=t.scalar,.module_index=0,.declaration=0,.first=0,.second=0});if(id==0){return .err(if(types::count(graph)>=4096){10}else{11});}if(!v::push<usize>(ref mut out,id).is_ok){return .err(11);}i+=1;}return .ok(out);
}
pub fn Result<catalog::Catalog,Problem> build_catalog(ref plan::Plan p,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    let c=input::templates(plan::input(p));let b=templates::packages(c);if(bindings::count(b)!=v::len<syntax::Parsed>(sources)){return .err(fail(12,0,0,0));}var owners=v::empty<usize>();var imports=v::empty<catalog::Binding>();var usize i=0;
    while(i<bindings::count(b)){let code=templates::current(b,i,v::at<syntax::Parsed>(sources,i),work);if(code!=0){return .err(fail(code,i,0,0));}if(!v::push<usize>(ref mut owners,bindings::module_at(b,i).owner).is_ok){return .err(fail(11,i,0,0));}i+=1;}
    i=0;while(i<bindings::import_count(b)){let row=bindings::import_at(b,i);if(!v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=row.module_index,.declaration=row.declaration,.target=row.target}).is_ok){return .err(fail(11,row.module_index,0,0));}i+=1;}
    return switch(catalog::build_scoped(sources,ref owners,ref imports)){case .ok(c):{.ok(c)}case .err(p):{.err(fail(p.code,p.module_index,p.start,p.end))}};
}
fn Problem retain_use(ref catalog::Catalog c,ref mut v::Vec<types::Type> graph,ref mut v::Vec<Use> records,ref syntax::Parsed source,usize module_index,usize declaration,usize node,ref v::Vec<usize> args,u32 kind,usize source_index,usize root,ref mut usize work){
    if(root==0){return fail(0,0,0,0);}if(v::len<Use>(records)>=16384 || !budget::charge(work,syntax::type_count(source)+1)){return fail(10,module_index,0,0);}
    let r=types::aggregate_resolve(c,source,module_index,if(node==0){0}else{declaration},root,args,graph);if(r.problem.code!=0){return problem(r.problem);}
    if(!v::push<Use>(records,Use{.kind,.node,.module_index,.declaration,.source_index,.type=r.type}).is_ok){return fail(11,module_index,0,0);}return fail(0,0,0,0);
}
fn Problem uses(ref catalog::Catalog c,ref mut v::Vec<types::Type> graph,ref mut v::Vec<Use> records,ref v::Vec<syntax::Parsed> sources,usize module_index,usize declaration,usize node,ref v::Vec<usize> args,ref mut usize work){
    let source=v::at<syntax::Parsed>(sources,module_index);let d=syntax::declaration_at(source,declaration-1);let usize scope=if(node==0){0}else{declaration};
    let result=retain_use(c,graph,records,source,module_index,declaration,node,args,1,declaration,d.result_type,work);if(result.code!=0){return result;}var usize i=0;
    while(i<syntax::entry_count(source)){if(!budget::charge(work,1)){return fail(10,module_index,d.start,d.end);}let e=syntax::entry_at(source,i);if(e.start>=d.start && e.end<=d.end){let r=retain_use(c,graph,records,source,module_index,declaration,node,args,2,i,e.type_root,work);if(r.code!=0){return r;}}i+=1;}
    i=1;while(i<=syntax::expression_count(source)){if(!budget::charge(work,1)){return fail(10,module_index,d.start,d.end);}let e=syntax::expression_at(source,i);if(e.start<d.start || e.end>d.end){i+=1;continue;}
        if(e.kind==23){let r=retain_use(c,graph,records,source,module_index,declaration,node,args,3,i,e.right,work);if(r.code!=0){return r;}}
        if(e.kind==10 && syntax::expression_at(source,e.left).kind==20){let applied=syntax::expression_at(source,e.left);let path=syntax::expression_at(source,applied.left);let target=catalog::lookup(c,module_index,syntax::source_view(source)[path.start..path.end]);if(target.code!=0){return fail(target.code,module_index,path.start,path.end);}let target_declaration=catalog::declaration_at(c,target.item);var values=v::empty<usize>();var usize j=0;let total=syntax::specialization_argument_count(source,e.left);
            while(j<total){let r=types::aggregate_resolve(c,source,module_index,scope,syntax::specialization_argument(source,e.left,j),args,graph);if(r.problem.code!=0){return problem(r.problem);}if(!v::push<usize>(ref mut values,r.type).is_ok){return fail(11,module_index,e.start,e.end);}j+=1;}
            let r=types::aggregate(c,target_declaration.module_index,target_declaration.declaration,ref values,graph);if(r.problem.code!=0){return problem(r.problem);}if(v::len<Use>(records)>=16384 || !budget::charge(work,total+1)){return fail(10,module_index,e.start,e.end);}if(!v::push<Use>(records,Use{.kind=4,.node,.module_index,.declaration,.source_index=i,.type=r.type}).is_ok){return fail(11,module_index,e.start,e.end);}
        }i+=1;
    }return fail(0,0,0,0);
}
pub fn Result<Prepared,Problem> build(plan::Plan original,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    let declarations=switch(build_catalog(ref original,sources,work)){case .ok(c):{c}case .err(p):{return .err(p);}};
    var graph=v::empty<types::Type>();var instances=v::empty<Instance>();var fields=v::empty<Field>();var records=v::empty<Use>();var usize i=0;
    while(i<plan::count(ref original)){if(!budget::charge(work,1)){return .err(fail(10,0,0,0));}let id=plan::ordered_at(ref original,i);let n=plan::at(ref original,id-1);let c=input::templates(plan::input(ref original));let t=templates::template_at(c,n.template_index);let source=v::at<syntax::Parsed>(sources,t.module_index);let d=syntax::declaration_at(source,t.declaration);let args=switch(arguments(ref original,n,ref mut graph)){case .ok(a):{a}case .err(code):{return .err(fail(code,t.module_index,d.start,d.end));}};
        if(t.kind==4 || t.kind==6){let r=types::aggregate(ref declarations,t.module_index,t.declaration+1,ref args,ref mut graph);if(r.problem.code!=0){return .err(problem(r.problem));}let first=v::len<Field>(ref fields);var usize j=0;
            while(j<d.field_count){if(v::len<Field>(ref fields)>=32768 || !budget::charge(work,syntax::type_count(source)+1)){return .err(fail(10,t.module_index,d.start,d.end));}let f=syntax::field_at(source,d.field_start+j);let field=types::aggregate_resolve(ref declarations,source,t.module_index,t.declaration+1,f.type_root,ref args,ref mut graph);if(field.problem.code!=0){return .err(problem(field.problem));}if(!v::push<Field>(ref mut fields,Field{.type=field.type,.source_index=d.field_start+j,.start=f.start,.end=f.end}).is_ok){return .err(fail(11,t.module_index,f.start,f.end));}j+=1;}
            if(!v::push<Instance>(ref mut instances,Instance{.node=id,.type=r.type,.module_index=t.module_index,.declaration=t.declaration+1,.first,.count=d.field_count}).is_ok){return .err(fail(11,t.module_index,d.start,d.end));}
        }
        let prepared=uses(ref declarations,ref mut graph,ref mut records,sources,t.module_index,t.declaration+1,id,ref args,work);if(prepared.code!=0){return .err(prepared);}i+=1;
    }
    var usize m=0;let empty=v::empty<usize>();while(m<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,m);i=1;while(i<syntax::declaration_count(source)){if(!budget::charge(work,1)){return .err(fail(10,m,0,0));}let d=syntax::declaration_at(source,i);if(d.kind==3 && d.generic_parameters==0){let prepared=uses(ref declarations,ref mut graph,ref mut records,sources,m,i+1,0,ref empty,work);if(prepared.code!=0){return .err(prepared);}}i+=1;}m+=1;}
    // Every concrete value type must join a materialized source instance. A
    // type request outside the sealed plan cannot publish an orphan layout.
    i=1;while(i<=types::count(ref graph)){let t=types::at(ref graph,i);if(t.kind==20 || t.kind==21){var usize matches=0;var usize j=0;while(j<v::len<Instance>(ref instances)){if(!budget::charge(work,1)){return .err(fail(10,t.module_index,0,0));}if(v::at<Instance>(ref instances,j).type==i){matches+=1;}j+=1;}if(matches!=1){return .err(fail(12,t.module_index,0,0));}}i+=1;}
    return .ok(Prepared{.plan=original,.catalog=declarations,.graph,.instances,.fields,.uses=records});
}

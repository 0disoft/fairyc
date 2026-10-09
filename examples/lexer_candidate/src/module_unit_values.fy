module module_unit_values;
import module_unit_value_model as m;
import module_unit_value_collect as collect;
import module_unit_call_closure as closure;
import module_unit_interfaces as interfaces;
import module_unit_bodies as bodies;
import module_unit_body_ast as ast;
import module_unit_names as names;
import module_unit_ordinary_names as ordinary;
import module_unit_calls as calls;
import module_unit_generics as generics;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;
import std::vec as v;

pub resource Plan {v::Vec<m::Values> contexts;v::Vec<types::Type> graph;usize expressions;usize locals;usize returns;usize unresolved;usize inferred;usize checked_returns;usize work;cleanup{}}
fn m::Problem failed(u32 code,usize module_index,usize start,usize end){return m::Problem{.code,.module_index,.start,.end,.expected=0,.actual=0};}
fn Result<m::Values,m::Problem> original_body(ref syntax::Parsed source,ref signatures::Program metadata,ref closure::Ready ready,m::Frame frame,usize offset,ref v::Vec<types::Type> graph,ref mut usize cursor,ref mut usize work){
    let a=switch(ast::build(source,frame.declaration,work)){case .ok(a):{a}case .err(code):{return .err(m::fail(frame,code,0,0,0,0));}};
    var table=switch(collect::seeds(ref a,frame)){case .ok(t):{t}case .err(p):{return .err(p);}};let seeded=collect::ordinary_seeds(ref a,closure::imported(ready),frame,offset,ref mut table,work);if(seeded.code!=0){return .err(seeded);}
    return collect::project(source,ref a,ordinary::resolved(ordinary::at(closure::original_names(ready),frame.context)),frame,closure::call_plan(ready),cursor,ref table,graph);
}
fn Result<m::Values,m::Problem> concrete_body(ref syntax::Parsed source,ref closure::Ready ready,m::Frame frame,ref v::Vec<types::Type> graph,ref mut usize cursor,ref mut usize work){
    let owned=closure::owned(ready);let a=if(frame.kind==1){bodies::body_ast(bodies::at(owned,frame.context))}else{bodies::cleanup_ast(bodies::cleanup_at(owned,frame.context))};let resolved=if(frame.kind==1){names::function_at(closure::resolved_names(ready),frame.context)}else{names::cleanup_at(closure::resolved_names(ready),frame.context)};
    var table=switch(collect::seeds(a,frame)){case .ok(t):{t}case .err(p):{return .err(p);}};let seeded=collect::concrete_seeds(owned,closure::imported(ready),frame,ref mut table,work);if(seeded.code!=0){return .err(seeded);}
    return collect::project(source,a,resolved,frame,closure::call_plan(ready),cursor,ref table,graph);
}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref closure::Ready ready){
    let imported=closure::imported(ready);let owned=closure::owned(ready);var v::Vec<types::Type> graph=v::empty<types::Type>();var usize work=0;var usize i=1;
    if(bodies::type_count(owned)!=interfaces::type_count(imported)){return .err(failed(12,0,0,0));}
    while(i<=interfaces::type_count(imported)){if(!m::charge(ref mut work,1)){return .err(failed(10,0,0,0));}let a=interfaces::type_at(imported,i);let b=bodies::type_at(owned,i);if(a.kind!=b.kind || a.scalar!=b.scalar || a.module_index!=b.module_index || a.declaration!=b.declaration || a.first!=b.first || a.second!=b.second){return .err(failed(12,0,0,0));}if(!v::push<types::Type>(ref mut graph,a).is_ok){return .err(failed(11,0,0,0));}i+=1;}
    let valid=generics::validate(interfaces::specializations(imported),sources,metadata,ref graph);if(valid.code!=0){return .err(failed(valid.code,valid.module_index,valid.start,valid.end));}
    // Append only builtin value types; existing canonical IDs and owners stay exact.
    var u32 scalar=1;while(scalar<=7){if(!m::charge(ref mut work,types::count(ref graph)+1)){return .err(failed(10,0,0,0));}if(types::canonical(ref mut graph,types::Type{.kind=1,.scalar,.module_index=0,.declaration=0,.first=0,.second=0})==0){return .err(failed(if(types::count(ref graph)>=4096){10}else{11},0,0,0));}scalar+=1;}
    var u32 literal=9;while(literal<=14){if(literal!=11 && literal!=12 && literal!=13){if(!m::charge(ref mut work,types::count(ref graph)+1)){return .err(failed(10,0,0,0));}if(types::canonical(ref mut graph,types::Type{.kind=literal,.scalar=0,.module_index=0,.declaration=0,.first=0,.second=0})==0){return .err(failed(if(types::count(ref graph)>=4096){10}else{11},0,0,0));}}literal+=1;}
    var v::Vec<m::Values> contexts=v::empty<m::Values>();var usize cursor=0;var usize expressions=0;var usize locals=0;var usize returns=0;var usize unresolved=0;var usize inferred=0;var usize checked_returns=0;var u32 kind=1;
    while(kind<=3){i=1;let original=closure::original_names(ready);let usize count=if(kind==1){bodies::count(owned)}else if(kind==2){bodies::cleanup_count(owned)}else{ordinary::count(original)};var usize offset=0;var usize module_at=interfaces::first(imported);
        while(i<=count){let usize module_index=if(kind==1){bodies::signature(bodies::at(owned,i)).module_index}else if(kind==2){bodies::cleanup_info(bodies::cleanup_at(owned,i)).module_index}else{ordinary::module_index(ordinary::at(original,i))};let usize declaration=if(kind==1){bodies::signature(bodies::at(owned,i)).declaration}else if(kind==2){bodies::cleanup_info(bodies::cleanup_at(owned,i)).declaration}else{ordinary::declaration(ordinary::at(original,i))};let source=v::at<syntax::Parsed>(sources,module_index);
            let usize result=if(kind==1){bodies::signature(bodies::at(owned,i)).result}else if(kind==2){m::find(ref graph,1,7)}else{signatures::at(metadata,signatures::identity(metadata,module_index,declaration)).result};let frame=m::Frame{.kind,.context=i,.module_index,.declaration,.result};
            if(kind==3){while(module_at<module_index){if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,0,0,0,0));}offset+=syntax::expression_count(v::at<syntax::Parsed>(sources,module_at));module_at+=1;}}
            let p=if(kind==3){switch(original_body(source,metadata,ready,frame,offset,ref graph,ref mut cursor,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}}}else{switch(concrete_body(source,ready,frame,ref graph,ref mut cursor,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}}};
            if(!m::charge(ref mut work,m::work(ref p)) || expressions>16384-m::expression_count(ref p) || locals>16384-m::local_count(ref p) || returns>8192-m::return_count(ref p) || v::len<m::Values>(ref contexts)>=1024){return .err(m::fail(frame,10,0,0,0,0));}
            expressions+=m::expression_count(ref p);locals+=m::local_count(ref p);returns+=m::return_count(ref p);unresolved+=m::unresolved(ref p);inferred+=m::inferred(ref p);checked_returns+=m::checked_returns(ref p);
            if(!v::push<m::Values>(ref mut contexts,p).is_ok){return .err(m::fail(frame,11,0,0,0,0));}i+=1;
        }kind+=1;
    }if(cursor!=calls::count(closure::call_plan(ready))){return .err(failed(12,0,0,0));}return .ok(Plan{.contexts,.graph,.expressions,.locals,.returns,.unresolved,.inferred,.checked_returns,.work});
}
pub fn usize count(ref Plan p){return v::len<m::Values>(ref p.contexts);}
pub fn ref m::Values at(ref Plan p,usize index){return v::at<m::Values>(ref p.contexts,index);}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize type_count(ref Plan p){return types::count(ref p.graph);}
pub fn types::Type type_at(ref Plan p,usize id){return types::at(ref p.graph,id);}
pub fn usize expressions(ref Plan p){return p.expressions;}
pub fn usize locals(ref Plan p){return p.locals;}
pub fn usize returns(ref Plan p){return p.returns;}
pub fn usize unresolved(ref Plan p){return p.unresolved;}
pub fn usize inferred(ref Plan p){return p.inferred;}
pub fn usize checked_returns(ref Plan p){return p.checked_returns;}

module module_unit_calls;
import module_unit_names as names;
import module_unit_name_model as model;
import module_unit_ordinary_names as ordinary;
import module_unit_bodies as bodies;
import module_unit_body_ast as ast;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_unit_generics as generics;
import module_signatures as signatures;
import module_types as types;
import module_ast_join as join;
import syntax as syntax;
import std::vec as v;
import std::bytes as bytes;

pub struct Call {u32 context_kind;usize context;usize module_index;usize declaration;usize expression;usize source_index;usize target_module;usize target_declaration;usize signature;usize result;usize parameter_start;usize parameter_count;u32 effects;bool must_use;bool unsafe_call;bool foreign;bool unsafe_context;usize start;usize end;}
pub resource Plan {v::Vec<Call> calls;v::Vec<binding::Coordinate> targets;usize work;usize specialized;usize variants;usize builtins;cleanup{}}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut usize work,usize amount){return ast::charge(work,amount);}
fn Result<usize,join::Problem> signature(ref signatures::Program metadata,ref interfaces::Interfaces imported,model::Reference r,usize module_index,usize start,usize end){
    let usize id=signatures::identity(metadata,r.module_index,r.declaration);if(id==0){return .err(fail(9,module_index,start,end));}let s=signatures::at(metadata,id);
    if(s.result==0 || s.result>interfaces::type_count(imported)){return .err(fail(12,module_index,start,end));}var usize i=0;while(i<s.parameter_count){let usize t=signatures::parameter_at(metadata,s.parameter_start+i);if(t==0 || t>interfaces::type_count(imported)){return .err(fail(12,module_index,start,end));}i+=1;}return .ok(id);
}
fn join::Problem record(ref syntax::Parsed source,ref signatures::Program metadata,ref interfaces::Interfaces imported,ref model::Resolved resolved,u32 context_kind,usize context,usize module_index,usize declaration,usize expression,usize source_index,usize callee,ref mut v::Vec<Call> calls,ref mut v::Vec<binding::Coordinate> targets,ref mut usize work,ref mut usize builtins){
    let e=syntax::expression_at(source,source_index);let target=syntax::expression_at(source,e.left);let usize use=names::use_at(resolved,callee);if(use==0){return fail(12,module_index,e.start,e.end);}let r=names::reference_at(resolved,use);if(r.expression!=callee || r.source_index!=e.left || target.kind!=2 || r.start!=target.start || r.end!=target.end){return fail(12,module_index,e.start,e.end);}
    if(r.kind==3){*builtins=*builtins+1;return fail(0,0,0,0);}if(r.kind!=2 || (r.declaration_kind!=3 && r.declaration_kind!=9)){return fail(9,module_index,target.start,target.end);}
    var usize probes=0;let found=signatures::declaration_lookup_measured(metadata,module_index,syntax::expression_text(source,e.left),ref mut probes);if(!charge(work,probes)){return fail(10,module_index,e.start,e.end);}if(found.code!=0){return fail(found.code,module_index,target.start,target.end);}let d=signatures::declaration_at(metadata,found.item);if(d.module_index!=r.module_index || d.declaration!=r.declaration || d.kind!=r.declaration_kind){return fail(12,module_index,target.start,target.end);}
    let id=switch(signature(metadata,imported,r,module_index,e.start,e.end)){case .ok(id):{id}case .err(p):{return p;}};let s=signatures::at(metadata,id);
    if(!charge(work,s.parameter_count+1) || v::len<Call>(calls)>=16384){return fail(10,module_index,e.start,e.end);}if(!v::push<Call>(calls,Call{.context_kind,.context,.module_index,.declaration,.expression,.source_index,.target_module=s.module_index,.target_declaration=s.declaration,.signature=id,.result=s.result,.parameter_start=s.parameter_start,.parameter_count=s.parameter_count,.effects=s.effects,.must_use=s.must_use,.unsafe_call=s.unsafe_call,.foreign=s.foreign,.unsafe_context=r.unsafe_context,.start=e.start,.end=e.end}).is_ok){return fail(11,module_index,e.start,e.end);}
    var usize i=0;while(i<interfaces::callable_count(imported)){if(!charge(work,1)){return fail(10,module_index,e.start,e.end);}let c=interfaces::callable_at(imported,i);if(c.module_index==s.module_index && c.declaration==s.declaration){if(c.template || c.result!=s.result || c.parameter_count!=s.parameter_count || c.effects!=s.effects || c.must_use!=s.must_use || c.unsafe_call!=s.unsafe_call || c.foreign!=s.foreign){return fail(12,module_index,e.start,e.end);}var usize j=0;while(j<s.parameter_count){if(!charge(work,1)){return fail(10,module_index,e.start,e.end);}if(interfaces::parameter_at(imported,c.parameter_start+j)!=signatures::parameter_at(metadata,s.parameter_start+j)){return fail(12,module_index,e.start,e.end);}j+=1;}return fail(0,0,0,0);}i+=1;}
    i=0;while(i<v::len<binding::Coordinate>(targets)){if(!charge(work,1)){return fail(10,module_index,e.start,e.end);}let c=*v::at<binding::Coordinate>(targets,i);if(c.module_index==s.module_index && c.declaration==s.declaration){return fail(0,0,0,0);}i+=1;}
    if(v::len<binding::Coordinate>(targets)>=512){return fail(10,module_index,e.start,e.end);}if(!v::push<binding::Coordinate>(targets,binding::Coordinate{.kind=d.kind,.module_index=d.module_index,.declaration=d.declaration,.start=d.source_start,.end=d.source_end}).is_ok){return fail(11,module_index,e.start,e.end);}return fail(0,0,0,0);
}
fn usize local_expression(ref ordinary::Function f,usize source_index){var usize low=1;var usize high=names::use_count(ordinary::resolved(f))+1;while(low<high){let usize mid=low+(high-low)/2;if(ordinary::expression_origin(f,mid)<source_index){low=mid+1;}else{high=mid;}}return if(low<=names::use_count(ordinary::resolved(f)) && ordinary::expression_origin(f,low)==source_index){low}else{0};}
pub fn Result<Plan,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref interfaces::Interfaces imported,ref bodies::Plan owned,ref names::Plan concrete,ref ordinary::Plan original){
    if(names::function_count(concrete)!=bodies::count(owned) || names::cleanup_count(concrete)!=bodies::cleanup_count(owned) || bodies::type_count(owned)!=interfaces::type_count(imported)){return .err(fail(12,0,0,0));}var usize work=0;var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;
    while(i<=interfaces::type_count(imported)){if(!charge(ref mut work,1)){return .err(fail(10,0,0,0));}let a=interfaces::type_at(imported,i);let b=bodies::type_at(owned,i);if(a.kind!=b.kind || a.scalar!=b.scalar || a.module_index!=b.module_index || a.declaration!=b.declaration || a.first!=b.first || a.second!=b.second){return .err(fail(12,0,0,0));}if(!v::push<types::Type>(ref mut graph,a).is_ok){return .err(fail(11,0,0,0));}i+=1;}let current=generics::validate(interfaces::specializations(imported),sources,metadata,ref graph);if(current.code!=0){return .err(current);}
    var v::Vec<Call> calls=v::empty<Call>();var v::Vec<binding::Coordinate> targets=v::empty<binding::Coordinate>();var usize specialized=0;var usize variants=0;var usize builtins=0;var u32 kind=1;
    while(kind<=3){i=1;let usize count=if(kind==1){bodies::count(owned)}else if(kind==2){bodies::cleanup_count(owned)}else{ordinary::count(original)};while(i<=count){let usize module_index=if(kind==1){bodies::signature(bodies::at(owned,i)).module_index}else if(kind==2){bodies::cleanup_info(bodies::cleanup_at(owned,i)).module_index}else{ordinary::module_index(ordinary::at(original,i))};let usize declaration=if(kind==1){bodies::signature(bodies::at(owned,i)).declaration}else if(kind==2){bodies::cleanup_info(bodies::cleanup_at(owned,i)).declaration}else{ordinary::declaration(ordinary::at(original,i))};let source=v::at<syntax::Parsed>(sources,module_index);let resolved=if(kind==1){names::function_at(concrete,i)}else if(kind==2){names::cleanup_at(concrete,i)}else{ordinary::resolved(ordinary::at(original,i))};var usize expression=1;
        while(expression<=names::use_count(resolved)){if(!charge(ref mut work,1)){return .err(fail(10,module_index,0,0));}let usize source_index=if(kind==1){ast::expression_origin(bodies::body_ast(bodies::at(owned,i)),expression)}else if(kind==2){ast::expression_origin(bodies::cleanup_ast(bodies::cleanup_at(owned,i)),expression)}else{ordinary::expression_origin(ordinary::at(original,i),expression)};let e=syntax::expression_at(source,source_index);
            if(e.kind==6){let target=syntax::expression_at(source,e.left);if(target.kind==8 && bytes::equal(syntax::expression_text(source,e.left),b"place")){builtins+=1;}else if(target.kind==20){specialized+=1;}else if(target.kind==28){variants+=1;}else if(target.kind==2){let usize callee=if(kind==1){ast::expression_at(bodies::body_ast(bodies::at(owned,i)),expression).left}else if(kind==2){ast::expression_at(bodies::cleanup_ast(bodies::cleanup_at(owned,i)),expression).left}else{local_expression(ordinary::at(original,i),e.left)};if(callee==0){return .err(fail(12,module_index,e.start,e.end));}let p=record(source,metadata,imported,resolved,kind,i,module_index,declaration,expression,source_index,callee,ref mut calls,ref mut targets,ref mut work,ref mut builtins);if(p.code!=0){return .err(p);}}else{return .err(fail(9,module_index,target.start,target.end));}}expression+=1;
        }i+=1;
    }kind+=1;}return .ok(Plan{.calls,.targets,.work,.specialized,.variants,.builtins});
}
pub fn usize count(ref Plan p){return v::len<Call>(ref p.calls);}
pub fn Call at(ref Plan p,usize index){return *v::at<Call>(ref p.calls,index);}
pub fn usize target_count(ref Plan p){return v::len<binding::Coordinate>(ref p.targets);}
pub fn binding::Coordinate target_at(ref Plan p,usize index){return *v::at<binding::Coordinate>(ref p.targets,index);}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize specialized(ref Plan p){return p.specialized;}
pub fn usize builtins(ref Plan p){return p.builtins;}

pub fn usize variants(ref Plan p){return p.variants;}

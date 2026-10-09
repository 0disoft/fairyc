module module_unit_generics;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_types as types;
import module_ast_join as join;
import syntax as syntax;
import effect_declarations as effects;
import storage_intrinsics as storage;
import sdk_authority as sdk;
import std::bytes as bytes;
import std::vec as v;

pub struct Instance {usize module_index;usize declaration;usize argument_start;usize argument_count;usize result;usize parameter_start;usize parameter_count;u32 effects;bool must_use;bool unsafe_call;}
pub struct Call {usize instance;u32 intrinsic;bool deferred;}
pub resource Plan {v::Vec<Instance> instances;v::Vec<usize> arguments;v::Vec<usize> parameters;v::Vec<Call> calls;usize work;cleanup{}}
struct Budget {usize work;}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut Budget work,usize amount){if(work.work>1048576 || amount>1048576-work.work){return false;}work.work+=amount;return true;}
fn usize owner(ref syntax::Parsed p,usize start,usize end,ref mut Budget work){var usize i=0;while(i<syntax::declaration_count(p)){if(!charge(work,1)){return 0;}let d=syntax::declaration_at(p,i);if(d.kind!=1 && start>=d.start && end<=d.end){return i+1;}i+=1;}return 0;}
fn bool shadow(ref syntax::Parsed p,usize caller,ref u8[] name,ref mut Budget work){let d=syntax::declaration_at(p,caller-1);var usize i=0;while(i<syntax::entry_count(p)){if(!charge(work,1)){return true;}let e=syntax::entry_at(p,i);if(e.start>=d.start && e.end<=d.end && bytes::equal(syntax::source_view(p)[e.name_start..e.name_end],name)){return true;}i+=1;}return false;}
fn Result<usize,join::Problem> instantiate(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize module_index,usize declaration,ref v::Vec<usize> args,ref mut v::Vec<types::Type> graph,ref mut v::Vec<Instance> instances,ref mut v::Vec<usize> arguments,ref mut v::Vec<usize> parameters,ref mut Budget work,usize caller,usize start,usize end){
    let source=v::at<syntax::Parsed>(sources,module_index);let d=syntax::declaration_at(source,declaration-1);let usize count=v::len<usize>(args);
    if(d.kind!=3 || d.generic_parameters==0 || d.owner_end!=0 || count==0 || count>8 || count!=syntax::generic_parameter_count(source,declaration-1)){return .err(fail(4501,caller,start,end));}
    var usize i=0;while(i<count){let a=syntax::generic_parameter_at(source,declaration-1,i);var usize j=0;while(j<i){let b=syntax::generic_parameter_at(source,declaration-1,j);if(bytes::equal(syntax::source_view(source)[a.start..a.end],syntax::source_view(source)[b.start..b.end])){return .err(fail(4501,module_index,a.start,a.end));}j+=1;}i+=1;}
    let bool vector=sdk::authentic(b"std::vec",syntax::source_view(source));i=0;while(i<count){let t=types::at(graph,*v::at<usize>(args,i));if(!(t.kind==1 && t.scalar>=1 && t.scalar<=6) && !(vector && (t.kind==2 || t.kind==3))){return .err(fail(4502,caller,start,end));}i+=1;}
    i=0;var usize per=0;while(i<v::len<Instance>(instances)){if(!charge(work,count+1)){return .err(fail(4503,caller,start,end));}let old=*v::at<Instance>(instances,i);if(old.module_index==module_index && old.declaration==declaration){per+=1;var bool same=old.argument_count==count;var usize j=0;while(j<count && same){same=*v::at<usize>(arguments,old.argument_start+j)==*v::at<usize>(args,j);j+=1;}if(same){return .ok(i+1);}}i+=1;}
    if(v::len<Instance>(instances)>=64 || per>=16){return .err(fail(4503,caller,start,end));}
    let result=signatures::specialize_type(metadata,source,module_index,declaration,d.result_type,args,graph);if(result.problem.code!=0){return .err(fail(result.problem.code,module_index,result.problem.start,result.problem.end));}
    let contract=effects::read(source,declaration-1);if(contract.code!=0){return .err(fail(contract.code,module_index,contract.start,contract.end));}
    let t=types::at(graph,result.type);if(contract.must_use && (t.kind==1 && t.scalar==7 || t.kind!=1 && t.kind!=2 && t.kind!=4 && t.kind!=12)){return .err(fail(4201,module_index,contract.must_start,contract.must_end));}
    let usize parameter_start=v::len<usize>(parameters);i=d.entry_start;while(i<d.entry_start+d.entry_count){if(!charge(work,1)){return .err(fail(4503,caller,start,end));}let e=syntax::entry_at(source,i);if(e.kind==1){let r=signatures::specialize_type(metadata,source,module_index,declaration,e.type_root,args,graph);if(r.problem.code!=0){return .err(fail(r.problem.code,module_index,r.problem.start,r.problem.end));}let t=types::at(graph,r.type);if(t.kind==1 && t.scalar==7){return .err(fail(9,module_index,e.type_start,e.type_end));}if(v::len<usize>(parameters)>=32768 || !v::push<usize>(parameters,r.type).is_ok){return .err(fail(if(v::len<usize>(parameters)>=32768){4503}else{11},caller,start,end));}}i+=1;}
    let usize argument_start=v::len<usize>(arguments);i=0;while(i<count){if(!v::push<usize>(arguments,*v::at<usize>(args,i)).is_ok){return .err(fail(11,caller,start,end));}i+=1;}
    if(!v::push<Instance>(instances,Instance{.module_index,.declaration,.argument_start,.argument_count=count,.result=result.type,.parameter_start,.parameter_count=v::len<usize>(parameters)-parameter_start,.effects=contract.bits,.must_use=contract.must_use,.unsafe_call=d.unsafe_end!=0}).is_ok){return .err(fail(11,caller,start,end));}return .ok(v::len<Instance>(instances));
}
fn Result<Plan,join::Problem> collect(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref mut v::Vec<types::Type> graph){
    let valid=binding::validate(bound,sources,metadata);if(valid.code!=0){return .err(valid);}var v::Vec<Instance> instances=v::empty<Instance>();var v::Vec<usize> arguments=v::empty<usize>();var v::Vec<usize> parameters=v::empty<usize>();var v::Vec<Call> calls=v::empty<Call>();var Budget budget=Budget{.work=0};var usize module_index=binding::first(bound);
    while(module_index<binding::stop(bound)){let source=v::at<syntax::Parsed>(sources,module_index);var usize i=1;while(i<=syntax::expression_count(source)){let e=syntax::expression_at(source,i);var Call call=Call{.instance=0,.intrinsic=0,.deferred=false};if(!charge(ref mut budget,1)){return .err(fail(4503,module_index,e.start,e.end));}
        if(e.kind==20){let usize caller=owner(source,e.start,e.end,ref mut budget);if(caller==0){return .err(fail(if(budget.work>=1048576){4503}else{4501},module_index,e.start,e.end));}let d=syntax::declaration_at(source,caller-1);
            if(d.generic_parameters!=0 || d.kind==12){call.deferred=true;}
            else{let base=syntax::expression_at(source,e.left);if(base.kind!=2){return .err(fail(4501,module_index,e.start,e.end));}let name=syntax::expression_text(source,e.left);let u32 intrinsic=storage::named(name);
                if(intrinsic!=0 && sdk::authentic(b"std::vec",syntax::source_view(source))){call.intrinsic=intrinsic;}
                else{if(shadow(source,caller,name,ref mut budget)){return .err(fail(if(budget.work>=1048576){4503}else{4501},module_index,base.start,base.end));}let found=signatures::declaration_lookup(metadata,module_index,name);if(found.code!=0){return .err(fail(found.code,module_index,base.start,base.end));}let target=signatures::declaration_at(metadata,found.item);var v::Vec<usize> args=v::empty<usize>();var usize j=0;
                    while(j<syntax::specialization_argument_count(source,i)){let r=signatures::resolve_type(metadata,source,module_index,syntax::specialization_argument(source,i,j),graph);if(r.problem.code!=0){return .err(fail(r.problem.code,module_index,r.problem.start,r.problem.end));}if(!v::push<usize>(ref mut args,r.type).is_ok){return .err(fail(11,module_index,e.start,e.end));}j+=1;}
                    call.instance=switch(instantiate(sources,metadata,target.module_index,target.declaration,ref args,graph,ref mut instances,ref mut arguments,ref mut parameters,ref mut budget,module_index,e.start,e.end)){case .ok(id):{id}case .err(p):{return .err(p);}};
                }
            }
        }if(!v::push<Call>(ref mut calls,call).is_ok){return .err(fail(11,module_index,e.start,e.end));}i+=1;
    }module_index+=1;}
    return .ok(Plan{.instances,.arguments,.parameters,.calls,.work=budget.work});
}
pub fn usize count(ref Plan p){return v::len<Instance>(ref p.instances);}
pub fn Instance at(ref Plan p,usize id){assert(id>0 && id<=count(p));return *v::at<Instance>(ref p.instances,id-1);}
pub fn usize argument_at(ref Plan p,usize index){return *v::at<usize>(ref p.arguments,index);}
pub fn usize parameter_at(ref Plan p,usize index){return *v::at<usize>(ref p.parameters,index);}
pub fn usize call_count(ref Plan p){return v::len<Call>(ref p.calls);}
pub fn Call call_at(ref Plan p,usize id){assert(id>0 && id<=call_count(p));return *v::at<Call>(ref p.calls,id-1);}
pub fn usize work(ref Plan p){return p.work;}
pub fn Result<Plan,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref mut v::Vec<types::Type> graph){
    var usize i=0;while(i<v::len<syntax::Parsed>(sources)){if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return .err(fail(12,i,0,0));}i+=1;}
    let usize before=types::count(graph);return switch(collect(sources,metadata,bound,graph)){case .ok(p):{.ok(p)}case .err(p):{while(types::count(graph)>before){discard v::pop<types::Type>(graph);}.err(p)}};
}

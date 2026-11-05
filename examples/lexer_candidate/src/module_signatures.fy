module module_signatures;
import candidate_limits as limits;
import generic_aggregate_types as aggregates;
import module_catalog as catalog;
import module_state_declarations as state_declarations;
import module_state_layout as state;
import module_state_interface as state_interface;
import module_public_interface as public_interface;
import module_public_interface_model as public_model;
import module_types as types;
import sdk_vector_plan as vectors;
import syntax as syntax;
import static_operation_declarations as operations;
import module_const_eval_declarations as const_eval;
import module_inline_declarations as inline;
import effect_declarations as effects;
import foreign_declarations as foreign;
import repr_c_declarations as repr;
import ffi_symbols as ffi;
import std::bytes as bytes;
import std::vec as v;

pub struct Signature {usize module_index;usize declaration;bool public;usize result;usize parameter_start;usize parameter_count;u32 effects;bool must_use;bool unsafe_call;bool foreign;usize link_start;usize link_end;}
pub resource Program {state::Plan states;vectors::Plan types;v::Vec<Signature> signatures;v::Vec<usize> parameters;v::Vec<bool> vector_sources;bytes::Buffer links;v::Vec<aggregates::Prepared> aggregates;cleanup{}}
pub fn ref state::Plan state_layouts(ref Program p){return ref p.states;}
pub fn Result<bytes::Buffer,u32> public_state_prefix(ref Program p,usize owner,ref mut usize work){return state_interface::prefix(ref p.states,owner,work);}
pub fn Result<bytes::Buffer,u32> public_interface_hash(ref Program p,usize owner,ref public_model::Descriptor descriptor,ref mut usize work){return public_interface::hash(ref p.states,owner,descriptor,work);}
pub fn bool sdk_vector_source(ref Program p,usize module_index){return module_index<v::len<bool>(ref p.vector_sources) && *v::at<bool>(ref p.vector_sources,module_index);}
pub fn usize module_count(ref Program p){return vectors::module_count(ref p.types);}
pub fn usize module_owner(ref Program p,usize module_index){return vectors::module_owner(ref p.types,module_index);}
pub fn usize declaration_count(ref Program p){return vectors::declaration_count(ref p.types);}
pub fn bool source_matches(ref Program p,usize module_index,ref syntax::Parsed source){return vectors::source_matches(ref p.types,module_index,source);}
pub fn u32[8] source_digest(ref Program p,usize module_index){return vectors::source_digest(ref p.types,module_index);}
pub fn catalog::Lookup declaration_lookup(ref Program p,usize module_index,ref u8[] spelling){return vectors::declaration_lookup(ref p.types,module_index,spelling);}
pub fn catalog::Lookup declaration_lookup_measured(ref Program p,usize module_index,ref u8[] spelling,ref mut usize probes){return vectors::declaration_lookup_measured(ref p.types,module_index,spelling,probes);}
pub fn catalog::Lookup import_target_measured(ref Program p,usize module_index,ref u8[] alias,ref mut usize probes){return vectors::import_target_measured(ref p.types,module_index,alias,probes);}
pub fn catalog::Declaration declaration_at(ref Program p,usize id){return vectors::declaration_at(ref p.types,id);}
pub fn usize vector_count(ref Program p){return vectors::count(ref p.types);}
pub fn vectors::Instance vector_at(ref Program p,usize index){return vectors::at(ref p.types,index);}
pub fn usize count(ref Program p){return v::len<Signature>(ref p.signatures);}
pub fn Signature at(ref Program p,usize id){assert(id>0 && id<=count(p));return *v::at<Signature>(ref p.signatures,id-1);}
pub fn usize parameter_at(ref Program p,usize index){return *v::at<usize>(ref p.parameters,index);}
pub fn types::Type type_at(ref Program p,usize id){return vectors::type_at(ref p.types,id);}
pub fn usize type_count(ref Program p){return vectors::type_count(ref p.types);}
// Owner identities are retained even when an associated function has no receiver.
// Consumers still authenticate this signature program against the current sources.
pub fn usize owner_type(ref Program p,ref syntax::Parsed source,usize module_index,usize declaration){
    if(module_index>=module_count(p) || declaration==0 || declaration>syntax::declaration_count(source)){return 0;}let d=syntax::declaration_at(source,declaration-1);if(d.owner_end==0){return 0;}
    let found=declaration_lookup(p,module_index,syntax::source_view(source)[d.owner_start..d.owner_end]);if(found.code!=0){return 0;}let nominal=declaration_at(p,found.item);var usize i=1;
    while(i<=type_count(p)){let t=type_at(p,i);if((t.kind==2 || t.kind==3) && t.module_index==nominal.module_index && t.declaration==nominal.declaration){return i;}i+=1;}return 0;
}
// 0 associated/nonmethod; 1 shared; 2 exclusive; 3 consuming value.
pub fn u32 receiver_kind(ref Program p,ref syntax::Parsed source,usize module_index,usize declaration){
    if(module_index>=module_count(p) || declaration==0 || declaration>syntax::declaration_count(source)){return 0;}let d=syntax::declaration_at(source,declaration-1);let id=identity(p,module_index,declaration);if(d.owner_end==0 || id==0 || at(p,id).parameter_count==0){return 0;}
    let first=syntax::entry_at(source,d.entry_start);if(!bytes::equal(syntax::source_view(source)[first.name_start..first.name_end],b"this")){return 0;}let t=type_at(p,parameter_at(p,at(p,id).parameter_start));return if(t.kind==7){1}else if(t.kind==8){2}else{3};
}
pub fn types::Resolution resolve_type(ref Program p,ref syntax::Parsed source,usize module_index,usize root,ref mut v::Vec<types::Type> graph){return vectors::resolve_type(ref p.types,source,module_index,root,graph);}
pub fn types::Resolution resolve_argument(ref Program p,ref syntax::Parsed source,usize module_index,usize declaration,usize root,ref v::Vec<usize> args,ref mut v::Vec<types::Type> graph){return vectors::resolve_argument(ref p.types,source,module_index,declaration,root,args,graph);}
pub fn Result<v::Vec<usize>,types::BatchProblem> resolve_types(ref Program p,ref syntax::Parsed source,usize module_index,ref v::Vec<usize> roots,ref mut v::Vec<types::Type> graph){return vectors::resolve_types(ref p.types,source,module_index,roots,graph);}
pub fn Result<v::Vec<usize>,types::BatchProblem> specialize_types(ref Program p,ref syntax::Parsed source,usize module_index,usize declaration,ref v::Vec<usize> roots,ref v::Vec<usize> arguments,ref mut v::Vec<types::Type> graph){return vectors::specialize_types(ref p.types,source,module_index,declaration,roots,arguments,graph);}
pub fn types::Resolution declaration_type(ref Program p,usize module_index,usize declaration,ref mut v::Vec<types::Type> graph){return vectors::declaration_type(ref p.types,module_index,declaration,graph);}
pub fn Result<types::Graph,u32> own_types(ref Program p,v::Vec<types::Type> graph){return vectors::own_types(ref p.types,graph);}
pub fn usize identity(ref Program p,usize module_index,usize declaration){var usize low=0;var usize high=count(p);while(low<high){let usize mid=low+(high-low)/2;let s=at(p,mid+1);if(s.module_index<module_index || s.module_index==module_index && s.declaration<declaration){low=mid+1;}else{high=mid;}}if(low<count(p)){let s=at(p,low+1);if(s.module_index==module_index && s.declaration==declaration){return low+1;}}return 0;}
pub fn ref u8[] linker_name(ref Program p,usize id){let s=at(p,id);return bytes::view(ref p.links)[s.link_start..s.link_end];}
pub fn catalog::Lookup lookup(ref Program p,usize module_index,ref u8[] spelling){let found=declaration_lookup(p,module_index,spelling);if(found.code!=0){return found;}let d=declaration_at(p,found.item);var usize i=1;while(i<=count(p)){let s=at(p,i);if(s.module_index==d.module_index && s.declaration==d.declaration){return catalog::Lookup{.code=0,.item=i};}i+=1;}return catalog::Lookup{.code=9,.item=0};}
fn catalog::Problem fail(u32 code,usize module_index,usize start,usize end){return catalog::Problem{.code,.module_index,.start,.end};}
fn bool ffi_type(ref v::Vec<syntax::Parsed> sources,ref v::Vec<types::Type> graph,usize id,bool result){
    let t=types::at(graph,id);if(t.kind==1){return ffi::scalar(t.scalar,result);}if(t.kind!=12){return false;}
    if(t.first==0){return ffi::scalar(t.scalar,result);}let child=types::at(graph,t.first);return child.kind==2 && repr::read(v::at<syntax::Parsed>(sources,child.module_index),child.declaration-1).repr_c;
}
pub fn Result<Program,catalog::Problem> build(ref v::Vec<syntax::Parsed> sources){
    let c=switch(catalog::build(sources)){case .ok(c):{c}case .err(p):{return .err(p);}};
    return build_catalog(sources,c);
}
// Own the package-resolved catalog, accepting only its exact retained sources.
pub fn Result<Program,catalog::Problem> build_catalog(ref v::Vec<syntax::Parsed> sources,catalog::Catalog c){
    let graph=v::empty<types::Type>();let prepared=v::empty<aggregates::Prepared>();return build_mode(sources,c,graph,prepared,false);
}
// Private signature admission retains the exact current source plan and fields.
// It does not claim body/ownership, executable MIR, cache or Atlas acceptance.
pub fn Result<Program,catalog::Problem> build_aggregates(ref v::Vec<syntax::Parsed> sources,aggregates::Prepared prepared,ref mut usize work){
    let c=switch(aggregates::build_catalog(aggregates::source_plan(ref prepared),sources,work)){case .ok(c):{c}case .err(p):{return .err(fail(p.code,p.module_index,p.start,p.end));}};
    var graph=v::empty<types::Type>();var usize i=1;while(i<=types::count(aggregates::type_records(ref prepared))){if(!limits::project_charge(work,1)){return .err(fail(10,0,0,0));}if(!v::push<types::Type>(ref mut graph,types::at(aggregates::type_records(ref prepared),i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    var records=v::empty<aggregates::Prepared>();if(!v::push<aggregates::Prepared>(ref mut records,prepared).is_ok){return .err(fail(11,0,0,0));}return build_mode(sources,c,graph,records,true);
}
pub fn bool aggregate_types(ref Program p){return vectors::aggregate_types(ref p.types);}
pub fn ref aggregates::Prepared aggregate_source(ref Program p){assert(v::len<aggregates::Prepared>(ref p.aggregates)==1);return v::at<aggregates::Prepared>(ref p.aggregates,0);}
fn types::Resolution resolve_signature(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref mut v::Vec<types::Type> graph,bool aggregates){if(aggregates){let empty=v::empty<usize>();return types::aggregate_resolve(c,p,module_index,0,root,ref empty,graph);}return types::resolve(c,p,module_index,root,graph);}
fn catalog::Problem check_owner(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,syntax::Declaration d,ref v::Vec<usize> parameters,usize start,ref mut v::Vec<types::Type> graph){
    if(d.owner_end==0){return fail(0,0,0,0);}let found=catalog::lookup(c,module_index,syntax::source_view(p)[d.owner_start..d.owner_end]);if(found.code!=0){return fail(4501,module_index,d.start,d.end);}let nominal=catalog::declaration_at(c,found.item);
    if(nominal.parameters!=0 || nominal.module_index!=module_index || nominal.kind!=4 && nominal.kind!=6){return fail(4501,module_index,d.start,d.end);}
    let owner=types::declaration(c,module_index,nominal.declaration,graph);if(owner.problem.code!=0){return fail(owner.problem.code,module_index,d.start,d.end);}var usize i=0;var usize position=0;
    while(i<d.entry_count){let e=syntax::entry_at(p,d.entry_start+i);i+=1;if(e.kind!=1){continue;}if(bytes::equal(syntax::source_view(p)[e.name_start..e.name_end],b"this")){
        if(position!=0){return fail(4501,module_index,d.start,d.end);}let receiver=types::at(graph,*v::at<usize>(parameters,start));let target=if(receiver.kind==7 || receiver.kind==8){types::at(graph,receiver.first)}else{receiver};
        if(target.kind!=if(nominal.kind==4){2}else{3} || target.module_index!=nominal.module_index || target.declaration!=nominal.declaration){return fail(4501,module_index,d.start,d.end);}
    }position+=1;}return fail(0,0,0,0);
}
fn Result<Program,catalog::Problem> build_mode(ref v::Vec<syntax::Parsed> sources,catalog::Catalog c,v::Vec<types::Type> initial,v::Vec<aggregates::Prepared> prepared,bool aggregates){
    if(catalog::module_count(ref c)!=v::len<syntax::Parsed>(sources)){return .err(fail(9,0,0,0));}
    var usize checked=0;while(checked<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,checked);if(!catalog::source_matches(ref c,checked,source)){return .err(fail(9,checked,0,0));}var usize declaration=0;while(declaration<syntax::declaration_count(source)){let inlined=inline::read(source,declaration);if(inlined.code!=0){return .err(fail(inlined.code,checked,inlined.start,inlined.end));}let admission=const_eval::read(source,declaration);if(admission.code!=0){return .err(fail(admission.code,checked,admission.start,admission.end));}declaration+=1;}checked+=1;}
    let states=switch(state_declarations::build(sources,ref c)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var v::Vec<types::Type> graph=initial;var v::Vec<Signature> signatures=v::empty<Signature>();var v::Vec<usize> parameters=v::empty<usize>();var v::Vec<bool> vector_sources=v::empty<bool>();var bytes::Buffer links=bytes::empty();
    var usize module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,module_index);var usize index=1;var usize operation_work=0;let operation_valid=operations::validate(p,ref mut operation_work);if(operation_valid!=0){return .err(fail(operation_valid,module_index,0,syntax::source_view(p).len));}
        let bool vector_sdk=catalog::sdk_vector_source(ref c,module_index);if(!v::push<bool>(ref mut vector_sources,vector_sdk).is_ok){return .err(fail(11,module_index,0,0));}
        while(index<syntax::declaration_count(p)){let d=syntax::declaration_at(p,index);if((d.kind==3 || d.kind==9) && d.generic_parameters==0){
            if(d.generic_parameters!=0){return .err(fail(4501,module_index,d.start,d.end));}
            let contract=effects::read(p,index);if(contract.code!=0){return .err(fail(contract.code,module_index,contract.start,contract.end));}
            var usize link_start=0;var usize link_end=0;
            let result=resolve_signature(ref c,p,module_index,d.result_type,ref mut graph,aggregates);if(result.problem.code!=0){return .err(fail(result.problem.code,module_index,result.problem.start,result.problem.end));}
            let result_type=types::at(ref graph,result.type);if(result_type.kind==19){let location=syntax::type_at(p,d.result_type);return .err(fail(3103,module_index,location.start,location.end));}if(contract.must_use && (result_type.kind==1 && result_type.scalar==7 || result_type.kind!=1 && result_type.kind!=2 && result_type.kind!=4 && result_type.kind!=12 && !(aggregates && result_type.kind==20))){return .err(fail(4201,module_index,contract.must_start,contract.must_end));}
            let usize parameter_start=v::len<usize>(ref parameters);var usize e=d.entry_start;
            while(e<d.entry_start+d.entry_count){let entry=syntax::entry_at(p,e);if(entry.kind==1){let type=resolve_signature(ref c,p,module_index,entry.type_root,ref mut graph,aggregates);if(type.problem.code!=0){return .err(fail(type.problem.code,module_index,type.problem.start,type.problem.end));}let t=types::at(ref graph,type.type);if(t.kind==1 && t.scalar==7){return .err(fail(9,module_index,entry.type_start,entry.type_end));}if(d.kind==9 && !ffi_type(sources,ref graph,type.type,false)){return .err(fail(5102,module_index,entry.type_start,entry.type_end));}if(v::len<usize>(ref parameters)>=32768 || !v::push<usize>(ref mut parameters,type.type).is_ok){return .err(fail(10,module_index,entry.start,entry.end));}}e+=1;}
            if(d.kind==9){let t=types::at(ref graph,result.type);
                let bool repr_result=t.kind==2 && repr::read(v::at<syntax::Parsed>(sources,t.module_index),t.declaration-1).repr_c;
                let ffi=foreign::read_signature(p,index,repr_result || ffi_type(sources,ref graph,result.type,true),true);
                if(ffi.code!=0){return .err(fail(ffi.code,module_index,ffi.start,ffi.end));}link_start=bytes::len(ref links);let name=syntax::source_view(p)[ffi.link_start..ffi.link_end];if(name.len>1048576-bytes::len(ref links) || !bytes::append(ref mut links,name).is_ok){return .err(fail(11,module_index,d.start,d.end));}link_end=bytes::len(ref links);}
            if(v::len<usize>(ref parameters)-parameter_start>128 || v::len<Signature>(ref signatures)>=limits::project_callables()){return .err(fail(10,module_index,d.start,d.end));}
            let usize parameter_count=v::len<usize>(ref parameters)-parameter_start;
            let owner=check_owner(ref c,p,module_index,d,ref parameters,parameter_start,ref mut graph);if(owner.code!=0){return .err(owner);}
            if(d.kind==9){var usize previous=0;while(previous<v::len<Signature>(ref signatures)){let old=*v::at<Signature>(ref signatures,previous);if(old.foreign && bytes::equal(bytes::view(ref links)[old.link_start..old.link_end],bytes::view(ref links)[link_start..link_end])){if(old.result!=result.type || old.parameter_count!=parameter_count){return .err(fail(5106,module_index,d.start,d.end));}var usize n=0;while(n<parameter_count){if(*v::at<usize>(ref parameters,old.parameter_start+n)!=*v::at<usize>(ref parameters,parameter_start+n)){return .err(fail(5106,module_index,d.start,d.end));}n+=1;}}previous+=1;}}
            if(!v::push<Signature>(ref mut signatures,Signature{.module_index,.declaration=index+1,.public=d.public,.result=result.type,.parameter_start,.parameter_count,.effects=contract.bits,.must_use=contract.must_use,.unsafe_call=d.unsafe_end!=0,.foreign=d.kind==9,.link_start,.link_end}).is_ok){return .err(fail(11,module_index,d.start,d.end));}
        }index+=1;}module_index+=1;
    }
    let owned=switch(if(aggregates){types::assemble_aggregates(ref c,graph)}else{types::assemble(ref c,graph)}){case .ok(g):{g}case .err(code):{return .err(fail(code,0,0,0));}};
    let planned=switch(vectors::build(c,owned)){case .ok(p):{p}case .err(code):{return .err(fail(code,0,0,0));}};
    return .ok(Program{.states,.types=planned,.signatures,.parameters,.vector_sources,.links,.aggregates=prepared});
}

pub fn types::Resolution specialize_type(ref Program p,ref syntax::Parsed source,usize module_index,usize declaration,usize root,ref v::Vec<usize> args,ref mut v::Vec<types::Type> graph){return vectors::specialize_type(ref p.types,source,module_index,declaration,root,args,graph);}

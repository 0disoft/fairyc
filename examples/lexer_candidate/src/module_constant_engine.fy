module module_constant_engine;
import module_unit_value_model as m;
import module_constant_model as model;
import module_constant_graph as graph;
import module_constant_types as types;
import module_constant_runtime as runtime;
import module_const_eval_declarations as const_eval;
import module_signatures as signatures;
import syntax as syntax;
import type_state as scalars;
import std::bytes as bytes;
import std::vec as v;

pub struct Value{u32 scalar;usize bits;}
pub resource Plan{v::Vec<model::Entry> entries;usize work;cleanup{}}
// Owners are checked package-instance coordinates, never module-name spellings.
pub fn Result<Plan,m::Problem> build_owned(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners){
    if(v::len<usize>(owners)!=0 && v::len<usize>(owners)!=v::len<syntax::Parsed>(sources)){return .err(model::fail(0,12,0,0));}
    var v::Vec<model::Entry> entries=v::empty<model::Entry>();var v::Vec<model::Parameter> parameters=v::empty<model::Parameter>();var v::Vec<u32> states=v::empty<u32>();var v::Vec<model::Counter> counters=v::empty<model::Counter>();var usize work=0;var usize module_index=0;var usize constants=0;var usize functions=0;
    while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);var usize i=0;
        while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(!model::charge(ref mut work)){return .err(model::fail(module_index,4304,d.start,d.end));}let admission=const_eval::read(source,i);if(admission.code!=0){return .err(model::fail(module_index,admission.code,admission.start,admission.end));}
            if(d.kind==7 || admission.annotated){let callable=admission.annotated;let u32 scalar=if(callable){admission.result}else{scalars::primitive(source,d.result_type)};
                if(!callable && (d.public || d.attribute_count!=0 || d.result_type==0 || syntax::type_at(source,d.result_type).kind!=1 || !model::admitted(scalar))){return .err(model::fail(module_index,4301,d.start,d.end));}
                if(callable && functions>=4096 || !callable && constants>=1024){return .err(model::fail(module_index,4304,d.start,d.end));}
                let parameter_start=v::len<model::Parameter>(ref parameters);if(callable){var usize j=d.entry_start;while(j<d.entry_start+d.entry_count){if(!model::charge(ref mut work)){return .err(model::fail(module_index,4304,d.start,d.end));}let p=syntax::entry_at(source,j);if(p.kind==1){var usize prior=parameter_start;while(prior<v::len<model::Parameter>(ref parameters)){if(!model::charge(ref mut work)){return .err(model::fail(module_index,4304,p.start,p.end));}let old=*v::at<model::Parameter>(ref parameters,prior);if(bytes::equal(syntax::source_view(source)[old.start..old.end],syntax::source_view(source)[p.name_start..p.name_end])){return .err(model::fail(module_index,102,p.name_start,p.name_end));}prior+=1;}
                    if(v::len<model::Parameter>(ref parameters)>=32768 || v::len<model::Parameter>(ref parameters)-parameter_start>=128){return .err(model::fail(module_index,4304,p.start,p.end));}if(!v::push<model::Parameter>(ref mut parameters,model::Parameter{.start=p.name_start,.end=p.name_end,.scalar=scalars::primitive(source,p.type_root)}).is_ok){return .err(model::fail(module_index,11,p.start,p.end));}}j+=1;}functions+=1;}else{constants+=1;}
                if(!v::push<model::Entry>(ref mut entries,model::Entry{.module_index,.declaration=i+1,.start=d.start,.end=d.end,.expression=if(callable){admission.expression}else{d.expression},.callable,.parameter_start,.parameter_count=v::len<model::Parameter>(ref parameters)-parameter_start,.value=model::Value{.scalar,.bits=0}}).is_ok || !v::push<u32>(ref mut states,0).is_ok){return .err(model::fail(module_index,11,d.start,d.end));}
            }
            if(d.kind==8 && (d.public || d.attribute_count!=0)){return .err(model::fail(module_index,4301,d.start,d.end));}i+=1;
        }module_index+=1;
    }
    let dependencies=graph::check(sources,ref entries,ref parameters,ref mut work);if(dependencies.code!=0){return .err(dependencies);}
    var usize i=0;while(i<v::len<model::Entry>(ref entries)){let row=*v::at<model::Entry>(ref entries,i);let scalar=switch(types::expression(sources,ref entries,ref parameters,row.module_index,row.expression,row.value.scalar,if(row.callable){i+1}else{0},ref mut work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};
        if(scalar!=row.value.scalar){let e=syntax::expression_at(v::at<syntax::Parsed>(sources,row.module_index),row.expression);return .err(model::fail(row.module_index,1006,e.start,e.end));}i+=1;
    }
    module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);i=1;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(d.kind==8){let scalar=switch(types::expression(sources,ref entries,ref parameters,module_index,d.expression,6,0,ref mut work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};if(scalar!=6){let e=syntax::expression_at(source,d.expression);return .err(model::fail(module_index,1007,e.start,e.end));}}i+=1;}module_index+=1;}
    i=0;while(i<v::len<model::Entry>(ref entries)){if(!v::at<model::Entry>(ref entries,i).callable){let value=switch(runtime::constant(sources,ref mut entries,ref parameters,i+1,ref mut states,owners,ref mut counters,ref mut work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};discard value;}i+=1;}
    let environment=v::empty<model::Value>();module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);i=1;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(d.kind==8){let value=switch(runtime::expression(sources,ref mut entries,ref parameters,module_index,d.expression,6,ref mut states,owners,ref mut counters,ref mut work,0,ref environment,0,0)){case .ok(v):{v}case .err(p):{return .err(p);}};if(value.bits==0){return .err(model::fail(module_index,4305,d.start,d.end));}}i+=1;}module_index+=1;}
    return .ok(Plan{.entries,.work});
}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources){let owners=v::empty<usize>();return build_owned(sources,ref owners);}
pub fn Result<Plan,m::Problem> build_project(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){var v::Vec<usize> owners=v::empty<usize>();var usize i=0;if(signatures::module_count(metadata)!=v::len<syntax::Parsed>(sources)){return .err(model::fail(0,12,0,0));}while(i<v::len<syntax::Parsed>(sources)){if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return .err(model::fail(i,12,0,0));}if(!v::push<usize>(ref mut owners,signatures::module_owner(metadata,i)).is_ok){return .err(model::fail(i,11,0,0));}i+=1;}return build_owned(sources,ref owners);}
pub fn Result<Value,m::Problem> get(ref Plan p,usize module_index,usize declaration,ref mut usize work){var usize i=0;while(i<v::len<model::Entry>(ref p.entries)){if(!model::charge(work)){return .err(model::fail(module_index,4304,0,0));}let e=*v::at<model::Entry>(ref p.entries,i);if(!e.callable && e.module_index==module_index && e.declaration==declaration){return .ok(Value{.scalar=e.value.scalar,.bits=e.value.bits});}i+=1;}return .err(model::fail(module_index,4301,0,0));}
pub fn usize work(ref Plan p){return p.work;}

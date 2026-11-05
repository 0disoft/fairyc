module module_unit_interface_declarations;
import module_unit_bind as binding;
import generic_aggregate_types as aggregates;
import generic_transport_scalars as transport;
import c_abi_types as abi;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;
import module_ast_join as join;
import resource_abilities as abilities;
import repr_c_declarations as repr;
import effect_declarations as effects;
import std::bytes as bytes;
import std::vec as v;
import module_unit_shape_contracts as contracts;
import candidate_limits as limits;

import module_unit_interfaces as owner;
import module_unit_interface_work as meter;

pub fn join::Problem callable(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,binding::Coordinate c,ref mut v::Vec<types::Type> graph,ref mut v::Vec<owner::Callable> callables,ref mut v::Vec<usize> parameters,ref mut bytes::Buffer names,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut meter::Budget work){
    if(!meter::charge(work,1)){return meter::fail(10,c.module_index,c.start,c.end);}var usize i=0;while(i<v::len<owner::Callable>(callables)){if(!meter::charge(work,1)){return meter::fail(10,c.module_index,c.start,c.end);}let old=*v::at<owner::Callable>(callables,i);if(old.module_index==c.module_index && old.declaration==c.declaration){return meter::fail(0,0,0,0);}i+=1;}
    i=0;
    let source=v::at<syntax::Parsed>(sources,c.module_index);let d=syntax::declaration_at(source,c.declaration-1);let usize id=signatures::identity(metadata,c.module_index,c.declaration);
    let usize start=v::len<usize>(parameters);var usize result=0;var u32 required=0;var bool must_use=false;var bool unsafe_call=false;var bool foreign=false;var usize link_start=0;var usize link_end=0;
    if(d.generic_parameters!=0){let contract=effects::read(source,c.declaration-1);if(contract.code!=0){return meter::fail(contract.code,c.module_index,contract.start,contract.end);}required=contract.bits;must_use=contract.must_use;unsafe_call=d.unsafe_end!=0;}
    else{if(id==0){return meter::fail(9,c.module_index,c.start,c.end);}let s=signatures::at(metadata,id);result=s.result;required=s.effects;must_use=s.must_use;unsafe_call=s.unsafe_call;foreign=s.foreign;
        let u32 code=meter::mark(graph,pending,seen,work,result);if(code!=0){return meter::fail(code,c.module_index,c.start,c.end);}i=0;while(i<s.parameter_count){if(!meter::charge(work,1)){return meter::fail(10,c.module_index,c.start,c.end);}let usize type=signatures::parameter_at(metadata,s.parameter_start+i);let u32 code=meter::mark(graph,pending,seen,work,type);if(code!=0){return meter::fail(code,c.module_index,c.start,c.end);}if(v::len<usize>(parameters)>=32768 || !v::push<usize>(parameters,type).is_ok){return meter::fail(if(v::len<usize>(parameters)>=32768){10}else{11},c.module_index,c.start,c.end);}i+=1;}
        if(foreign){link_start=switch(meter::append(names,work,signatures::linker_name(metadata,id))){case .ok(n):{n}case .err(code):{return meter::fail(code,c.module_index,c.start,c.end);}};link_end=bytes::len(names);}
    }
    if(v::len<owner::Callable>(callables)>=512){return meter::fail(10,c.module_index,c.start,c.end);}if(!v::push<owner::Callable>(callables,owner::Callable{.module_index=c.module_index,.declaration=c.declaration,.template=d.generic_parameters!=0,.result,.parameter_start=start,.parameter_count=v::len<usize>(parameters)-start,.effects=required,.must_use,.unsafe_call,.foreign,.link_start,.link_end}).is_ok){return meter::fail(11,c.module_index,c.start,c.end);}return meter::fail(0,0,0,0);
}
pub fn join::Problem shape(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize id,ref mut v::Vec<types::Type> graph,ref mut v::Vec<owner::Shape> shapes,ref mut v::Vec<owner::Field> fields,ref mut bytes::Buffer names,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,usize first,usize stop,ref mut meter::Budget work){
    if(id==0 || id>types::count(graph)){return meter::fail(12,0,0,0);}let t=types::at(graph,id);if(t.module_index>=v::len<syntax::Parsed>(sources)){return meter::fail(12,t.module_index,0,0);}let source=v::at<syntax::Parsed>(sources,t.module_index);if(!signatures::source_matches(metadata,t.module_index,source) || t.declaration==0 || t.declaration>syntax::declaration_count(source)){return meter::fail(12,t.module_index,0,0);}let d=syntax::declaration_at(source,t.declaration-1);let usize field_start=v::len<owner::Field>(fields);
    let bool aggregate=t.kind==20 || t.kind==21;var usize selected=0;
    if(aggregate){if(!signatures::aggregate_types(metadata) || !signatures::source_matches(metadata,t.module_index,source) || id>signatures::type_count(metadata)){return meter::fail(12,t.module_index,d.start,d.end);}let expected=signatures::type_at(metadata,id);if(expected.kind!=t.kind || expected.scalar!=t.scalar || expected.module_index!=t.module_index || expected.declaration!=t.declaration || expected.first!=t.first || expected.second!=t.second){return meter::fail(12,t.module_index,d.start,d.end);}let prepared=signatures::aggregate_source(metadata);var usize n=0;while(n<aggregates::count(prepared)){if(!meter::charge(work,1)){return meter::fail(10,t.module_index,d.start,d.end);}let candidate=aggregates::at(prepared,n);if(candidate.type==id){if(selected!=0 || candidate.module_index!=t.module_index || candidate.declaration!=t.declaration || candidate.count!=d.field_count){return meter::fail(12,t.module_index,d.start,d.end);}selected=n+1;}n+=1;}if(selected==0){return meter::fail(12,t.module_index,d.start,d.end);}let checked=contracts::aggregate_declaration(source,t.module_index,t.declaration-1,ref mut work.work);if(checked.code!=0){return checked;}}
    else if(t.module_index<first || t.module_index>=stop){let checked=contracts::declaration(source,t.module_index,t.declaration-1,signatures::sdk_vector_source(metadata,t.module_index),ref mut work.work);if(checked.code!=0){return checked;}}
    if(v::len<owner::Shape>(shapes)>=512 || field_start>limits::metadata_fields() || d.field_count>limits::metadata_fields()-field_start){return meter::fail(10,t.module_index,d.start,d.end);}var usize i=0;
    while(i<d.field_count){if(!meter::charge(work,1)){return meter::fail(10,t.module_index,d.start,d.end);}let f=syntax::field_at(source,d.field_start+i);var usize type=0;
        if(f.type_root!=0){if(aggregate){let prepared=signatures::aggregate_source(metadata);let instance=aggregates::at(prepared,selected-1);let concrete=aggregates::field_at(prepared,instance.first+i);if(concrete.source_index!=d.field_start+i || concrete.start!=f.start || concrete.end!=f.end || concrete.type==0 || concrete.type>types::count(graph)){return meter::fail(12,t.module_index,f.start,f.end);}type=concrete.type;}
            else{let resolved=signatures::resolve_type(metadata,source,t.module_index,f.type_root,graph);if(resolved.problem.code!=0){return meter::fail(resolved.problem.code,t.module_index,resolved.problem.start,resolved.problem.end);}type=resolved.type;}
            let u32 code=meter::mark(graph,pending,seen,work,type);if(code!=0){return meter::fail(code,t.module_index,f.start,f.end);}}
        let usize name_start=switch(meter::append(names,work,syntax::source_view(source)[f.name_start..f.name_end])){case .ok(n):{n}case .err(code):{return meter::fail(code,t.module_index,f.start,f.end);}};
        if(!v::push<owner::Field>(fields,owner::Field{.kind=f.kind,.type,.module_index=t.module_index,.declaration=t.declaration,.source_index=d.field_start+i,.start=f.start,.end=f.end,.name_start,.name_end=bytes::len(names),.payload_start=if(f.payload_count==0){0}else{field_start+f.payload_start-d.field_start},.payload_count=f.payload_count}).is_ok){return meter::fail(11,t.module_index,f.start,f.end);}i+=1;
    }
    let representation=if(aggregate){repr::read_template(source,t.declaration-1)}else{repr::read(source,t.declaration-1)};if(representation.code!=0){return meter::fail(representation.code,t.module_index,representation.start,representation.end);}if(aggregate && representation.repr_c){i=0;while(i<d.field_count){if(!meter::charge(work,1)){return meter::fail(10,t.module_index,d.start,d.end);}let field=*v::at<owner::Field>(fields,field_start+i);let value=types::at(graph,field.type);if(value.kind!=12 && !(value.kind==1 && (transport::valid(value.scalar) || abi::valid(value.scalar)))){let original=syntax::field_at(source,d.field_start+i);let root=syntax::type_at(source,original.type_root);return meter::fail(5102,t.module_index,root.start,root.end);}i+=1;}}
    var u32 ability_bits=0;var u32 cleanup_effects=0;if(d.kind==6){let ability=abilities::read(source,t.declaration-1);if(!meter::charge(work,ability.work)){return meter::fail(10,t.module_index,d.start,d.end);}if(ability.code!=0){return meter::fail(ability.code,t.module_index,ability.start,ability.end);}ability_bits=ability.bits;let contract=effects::read(source,t.declaration-1);if(contract.code!=0){return meter::fail(contract.code,t.module_index,contract.start,contract.end);}cleanup_effects=contract.bits;}
    let bool nonempty=d.kind==6 && (syntax::block_statement_count(source,d.cleanup_body)!=0 || syntax::statement_at(source,d.cleanup_body).expression!=0);
    if(!v::push<owner::Shape>(shapes,owner::Shape{.type=id,.module_index=t.module_index,.declaration=t.declaration,.field_start,.field_count=d.field_count,.abilities=ability_bits,.repr_c=representation.repr_c,.start=d.start,.end=d.end,.cleanup_nonempty=nonempty,.cleanup_effects}).is_ok){return meter::fail(11,t.module_index,d.start,d.end);}return meter::fail(0,0,0,0);
}

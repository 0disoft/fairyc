module module_project_metadata;
import candidate_limits as limits;
import module_project_types as types;
import module_unit_body_ir_model as ir;
import module_unit_interfaces as interfaces;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import std::bytes as bytes;
import std::vec as v;

pub struct Range {usize start;usize count;}
pub struct Choice {usize unit;usize type;usize layout;}
pub resource Plan {v::Vec<ir::Layout> layouts;v::Vec<interfaces::Field> fields;v::Vec<interfaces::Callable> callables;v::Vec<usize> parameters;bytes::Buffer names;v::Vec<Range> ranges;v::Vec<usize> maps;v::Vec<Choice> choices;usize work;cleanup{}}
fn clean::Problem fail(u32 code,usize module_index){return clean::Problem{.code,.module_index,.start=0,.end=0};}
fn bool shape(interfaces::Shape a,interfaces::Shape b){return a.type==b.type && a.module_index==b.module_index && a.declaration==b.declaration && a.field_count==b.field_count && a.abilities==b.abilities && a.repr_c==b.repr_c && a.start==b.start && a.end==b.end && a.cleanup_nonempty==b.cleanup_nonempty && a.cleanup_effects==b.cleanup_effects;}
fn bool field(interfaces::Field a,interfaces::Field b){return a.kind==b.kind && a.type==b.type && a.module_index==b.module_index && a.declaration==b.declaration && a.source_index==b.source_index && a.start==b.start && a.end==b.end && a.payload_start==b.payload_start && a.payload_count==b.payload_count;}
fn Result<usize,clean::Problem> name(ref mut bytes::Buffer names,ref mut usize work,ref u8[] value){if(!budget::charge(work,value.len+1) || bytes::len(names)>1048576 || value.len>1048576-bytes::len(names)){return .err(fail(1002,0));}let start=bytes::len(names);return if(bytes::append(names,value).is_ok){.ok(start)}else{.err(fail(1003,0))};}
fn clean::Problem layout(ref ir::Program input,ref types::Plan graph,usize unit,usize index,usize map_start,ref mut v::Vec<ir::Layout> layouts,ref mut v::Vec<interfaces::Field> fields,ref mut bytes::Buffer names,ref mut v::Vec<usize> maps,ref mut v::Vec<Choice> choices,ref mut usize work){
    let original=ir::layout_at(input,index);var s=original.value;s.type=types::mapped(graph,unit,s.type);if(s.type==0){return fail(1001,s.module_index);}var usize selected=0;var usize i=0;
    while(i<v::len<ir::Layout>(layouts)){if(!budget::charge(work,1)){return fail(1002,s.module_index);}if(v::at<ir::Layout>(layouts,i).value.type==s.type){selected=i+1;break;}i+=1;}
    if(selected!=0){let old=*v::at<ir::Layout>(layouts,selected-1);if(!shape(old.value,s) || !bytes::equal(bytes::view(names)[old.name_start..old.name_end],ir::layout_name(input,index))){return fail(1001,s.module_index);}s.field_start=old.value.field_start;}
    else{if(v::len<ir::Layout>(layouts)>=1024 || s.field_count>limits::project_fields()-v::len<interfaces::Field>(fields)){return fail(1002,s.module_index);}s.field_start=v::len<interfaces::Field>(fields);let start=switch(name(names,work,ir::layout_name(input,index))){case .ok(n):{n}case .err(e):{return e;}};if(!v::push<ir::Layout>(layouts,ir::Layout{.value=s,.name_start=start,.name_end=bytes::len(names)}).is_ok){return fail(1003,s.module_index);}let pick=v::at_mut<Choice>(choices,s.type-1);*pick=Choice{.unit,.type=original.value.type,.layout=index+1};}
    i=0;while(i<s.field_count){if(!budget::charge(work,1)){return fail(1002,s.module_index);}let source=original.value.field_start+i;var f=ir::field_at(input,source);f.type=types::mapped(graph,unit,f.type);if(f.payload_count!=0){f.payload_start=s.field_start+f.payload_start-original.value.field_start;}
        if(selected!=0){let old=*v::at<interfaces::Field>(fields,s.field_start+i);if(!field(old,f) || !bytes::equal(bytes::view(names)[old.name_start..old.name_end],ir::field_name(input,source))){return fail(1001,s.module_index);}}
        else{f.name_start=switch(name(names,work,ir::field_name(input,source))){case .ok(n):{n}case .err(e):{return e;}};f.name_end=bytes::len(names);if(!v::push<interfaces::Field>(fields,f).is_ok){return fail(1003,s.module_index);}}
        let slot=v::at_mut<usize>(maps,map_start+source);if(*slot!=0 && *slot!=s.field_start+i+1){return fail(1001,s.module_index);};*slot=s.field_start+i+1;i+=1;
    }return fail(0,0);
}
fn clean::Problem callable(ref ir::Program input,ref types::Plan graph,usize unit,usize index,ref mut v::Vec<interfaces::Callable> callables,ref mut v::Vec<usize> indexes,ref mut v::Vec<usize> parameters,ref mut bytes::Buffer names,ref mut usize work){var c=ir::callable_at(input,index);c.result=types::mapped(graph,unit,c.result);var usize i=0;var usize bucket=(c.module_index*104729 ^ c.declaration*3266489917)&8191;
    while(*v::at<usize>(indexes,bucket)!=0){if(!budget::charge(work,1)){return fail(1002,c.module_index);}let old=*v::at<interfaces::Callable>(callables,*v::at<usize>(indexes,bucket)-1);if(old.module_index==c.module_index && old.declaration==c.declaration){if(old.template!=c.template || old.result!=c.result || old.parameter_count!=c.parameter_count || old.effects!=c.effects || old.must_use!=c.must_use || old.unsafe_call!=c.unsafe_call || old.foreign!=c.foreign || !bytes::equal(bytes::view(names)[old.link_start..old.link_end],ir::linker_name(input,index))){return fail(1001,c.module_index);}var usize j=0;while(j<c.parameter_count){if(!budget::charge(work,1)){return fail(1002,c.module_index);}if(*v::at<usize>(parameters,old.parameter_start+j)!=types::mapped(graph,unit,ir::parameter_at(input,c.parameter_start+j))){return fail(1001,c.module_index);}j+=1;}return fail(0,0);}bucket=(bucket+1)&8191;}
    if(v::len<interfaces::Callable>(callables)>=limits::project_callables() || c.parameter_count>32768-v::len<usize>(parameters)){return fail(1002,c.module_index);}let source=c.parameter_start;c.parameter_start=v::len<usize>(parameters);i=0;while(i<c.parameter_count){if(!budget::charge(work,1)){return fail(1002,c.module_index);}if(!v::push<usize>(parameters,types::mapped(graph,unit,ir::parameter_at(input,source+i))).is_ok){return fail(1003,c.module_index);}i+=1;}
    c.link_start=switch(name(names,work,ir::linker_name(input,index))){case .ok(n):{n}case .err(e):{return e;}};c.link_end=bytes::len(names);if(!v::push<interfaces::Callable>(callables,c).is_ok){return fail(1003,c.module_index);}let slot=v::at_mut<usize>(indexes,bucket);*slot=v::len<interfaces::Callable>(callables);return fail(0,0);
}
pub fn Result<Plan,clean::Problem> build(ref v::Vec<ir::Program> inputs,ref types::Plan graph){var v::Vec<ir::Layout> layouts=v::empty<ir::Layout>();var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();var v::Vec<usize> parameters=v::empty<usize>();var bytes::Buffer names=bytes::empty();var v::Vec<Range> ranges=v::empty<Range>();var v::Vec<usize> maps=v::empty<usize>();var v::Vec<Choice> choices=v::empty<Choice>();var v::Vec<usize> indexes=v::empty<usize>();var usize work=0;if(v::len<ir::Program>(inputs)!=types::unit_count(graph)){return .err(fail(1001,0));}var usize i=0;
    while(i<8192){if(!budget::charge(ref mut work,1)){return .err(fail(1002,0));}if(!v::push<usize>(ref mut indexes,0).is_ok){return .err(fail(1003,0));}i+=1;}i=0;
    while(i<types::type_count(graph)){if(!v::push<Choice>(ref mut choices,Choice{.unit=0,.type=0,.layout=0}).is_ok){return .err(fail(1003,0));}i+=1;}
    var usize unit=0;while(unit<v::len<ir::Program>(inputs)){let input=v::at<ir::Program>(inputs,unit);let start=v::len<usize>(ref maps);i=0;while(i<ir::field_count(input)){if(!budget::charge(ref mut work,1)){return .err(fail(1002,0));}if(!v::push<usize>(ref mut maps,0).is_ok){return .err(fail(1003,0));}i+=1;}if(!v::push<Range>(ref mut ranges,Range{.start,.count=ir::field_count(input)}).is_ok){return .err(fail(1003,0));}
        i=0;while(i<ir::layout_count(input)){let e=layout(input,graph,unit,i,start,ref mut layouts,ref mut fields,ref mut names,ref mut maps,ref mut choices,ref mut work);if(e.code!=0){return .err(e);}i+=1;}i=0;while(i<ir::callable_count(input)){let e=callable(input,graph,unit,i,ref mut callables,ref mut indexes,ref mut parameters,ref mut names,ref mut work);if(e.code!=0){return .err(e);}i+=1;}unit+=1;
    }return .ok(Plan{.layouts,.fields,.callables,.parameters,.names,.ranges,.maps,.choices,.work});
}
pub fn usize mapped_field(ref Plan p,usize unit,usize id){if(id==0 || unit>=v::len<Range>(ref p.ranges)){return 0;}let r=*v::at<Range>(ref p.ranges,unit);return if(id>r.count){0}else{*v::at<usize>(ref p.maps,r.start+id-1)};}
pub fn Choice choice(ref Plan p,usize type){return *v::at<Choice>(ref p.choices,type-1);}
pub fn usize layout_count(ref Plan p){return v::len<ir::Layout>(ref p.layouts);}
pub fn ir::Layout layout_at(ref Plan p,usize i){return *v::at<ir::Layout>(ref p.layouts,i);}
pub fn usize field_count(ref Plan p){return v::len<interfaces::Field>(ref p.fields);}
pub fn interfaces::Field field_at(ref Plan p,usize i){return *v::at<interfaces::Field>(ref p.fields,i);}
pub fn usize callable_count(ref Plan p){return v::len<interfaces::Callable>(ref p.callables);}
pub fn interfaces::Callable callable_at(ref Plan p,usize i){return *v::at<interfaces::Callable>(ref p.callables,i);}
pub fn usize parameter_count(ref Plan p){return v::len<usize>(ref p.parameters);}
pub fn usize parameter_at(ref Plan p,usize i){return *v::at<usize>(ref p.parameters,i);}
pub fn ref u8[] names(ref Plan p){return bytes::view(ref p.names);}
pub fn usize work(ref Plan p){return p.work;}

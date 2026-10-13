module module_project_public_view;
import module_project_identified_ir as checked;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as records;
import module_types as types;
import module_public_type_ids as ids;
import module_public_interface_model as model;
import module_public_type_text as coordinates;
import module_signatures as signatures;
import generic_package_bindings as bindings;
import module_source_provenance as provenance;
import candidate_limits as limits;
import module_state_layout as phase;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// A detached read-only projection of an already checked linked owner. This is
// not a typed Interfaces object: it contains no body requests or planner state.
// Type, field, callable and parameter IDs all retain the linked graph numbering.
pub resource View{v::Vec<types::Type> types;v::Vec<records::Shape> shapes;v::Vec<records::Field> fields;v::Vec<records::Callable> callables;v::Vec<usize> parameters;bytes::Buffer names;ids::Table identities;bindings::Bindings provenance;cleanup{}}
pub fn usize type_count(ref View p){return v::len<types::Type>(ref p.types);}
pub fn types::Type type_at(ref View p,usize id){assert(id>0 && id<=type_count(p));return *v::at<types::Type>(ref p.types,id-1);}
pub fn ref v::Vec<types::Type> type_records(ref View p){return ref p.types;}
pub fn ref ids::Table type_names(ref View p){return ref p.identities;}
pub fn ref v::Vec<records::Shape> shape_records(ref View p){return ref p.shapes;}
pub fn ref v::Vec<records::Field> field_records(ref View p){return ref p.fields;}
pub fn ref bytes::Buffer name_records(ref View p){return ref p.names;}
pub fn usize shape_count(ref View p){return v::len<records::Shape>(ref p.shapes);}
pub fn records::Shape shape_at(ref View p,usize index){return *v::at<records::Shape>(ref p.shapes,index);}
pub fn usize field_count(ref View p){return v::len<records::Field>(ref p.fields);}
pub fn records::Field field_at(ref View p,usize index){return *v::at<records::Field>(ref p.fields,index);}
pub fn ref u8[] field_name(ref View p,usize index){let f=field_at(p,index);return bytes::view(ref p.names)[f.name_start..f.name_end];}
pub fn usize callable_count(ref View p){return v::len<records::Callable>(ref p.callables);}
pub fn records::Callable callable_at(ref View p,usize index){return *v::at<records::Callable>(ref p.callables,index);}
pub fn usize parameter_count(ref View p){return v::len<usize>(ref p.parameters);}
pub fn usize parameter_at(ref View p,usize index){return *v::at<usize>(ref p.parameters,index);}
pub fn ref u8[] linker_name(ref View p,usize index){let c=callable_at(p,index);return bytes::view(ref p.names)[c.link_start..c.link_end];}
pub fn u32 current(ref View p,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){return provenance::current(ref p.provenance,sources,metadata,table,work);}

fn Result<ids::Table,u32> names(ref checked::Program input,ref ir::Program program,ref signatures::Program metadata,ref mut usize work){
    var bytes::Buffer text=bytes::empty();var v::Vec<ids::Row> rows=v::empty<ids::Row>();var v::Vec<ids::Stamp> stamps=v::empty<ids::Stamp>();var usize i=0;
    while(i<signatures::module_count(metadata)){if(!phase::charge(work,1)){return .err(10);}if(!v::push<ids::Stamp>(ref mut stamps,ids::Stamp{.owner=signatures::module_owner(metadata,i),.digest=signatures::source_digest(metadata,i)}).is_ok){return .err(11);}i+=1;}
    i=1;while(i<=clean::type_count(ir::cleanup_ir(program))){if(!phase::charge(work,1)){return .err(10);}let type=clean::type_at(ir::cleanup_ir(program),i);var usize owner=0;var model::Text name=model::Text{.start=0,.end=0};
        if(type.kind==13){let row=checked::type_at(input,i);let id=checked::identity(input,row);if(row.node==0 || id.len!=67 || type.module_index>=signatures::module_count(metadata)){return .err(9);}let prefix=b"std::vec::_fy_generic_";if(!phase::charge(work,prefix.len+id.len+1) || bytes::len(ref text)>1048576-prefix.len-id.len){return .err(10);}let start=bytes::len(ref text);if(!bytes::append(ref mut text,prefix).is_ok || !bytes::append(ref mut text,id).is_ok){return .err(11);}name=model::Text{.start,.end=bytes::len(ref text)};if(!ids::valid(bytes::view(ref text)[name.start..name.end])){return .err(9);}owner=signatures::module_owner(metadata,type.module_index);}
        if(!v::push<ids::Row>(ref mut rows,ids::Row{.type,.owner,.name}).is_ok){return .err(11);}i+=1;
    }return .ok(ids::assemble(text,rows,stamps));
}
fn u32 complete_signatures(ref signatures::Program metadata,ref mut v::Vec<records::Callable> callables,ref mut v::Vec<usize> parameters,ref mut bytes::Buffer names,ref mut usize work){
    var v::Vec<usize> index=v::empty<usize>();var usize i=0;while(i<8192){if(!phase::charge(work,1)){return 10;}if(!v::push<usize>(ref mut index,0).is_ok){return 11;}i+=1;}i=0;
    while(i<v::len<records::Callable>(callables)){let c=*v::at<records::Callable>(callables,i);var usize slot=(c.module_index*104729 ^ c.declaration*3266489917)&8191;while(*v::at<usize>(ref index,slot)!=0){if(!phase::charge(work,1)){return 10;}slot=(slot+1)&8191;}if(!phase::charge(work,1)){return 10;}let target=v::at_mut<usize>(ref mut index,slot);*target=i+1;i+=1;}
    i=1;while(i<=signatures::count(metadata)){if(!phase::charge(work,1)){return 10;}let s=signatures::at(metadata,i);var usize slot=(s.module_index*104729 ^ s.declaration*3266489917)&8191;
        while(*v::at<usize>(ref index,slot)!=0){if(!phase::charge(work,1)){return 10;}let c=*v::at<records::Callable>(callables,*v::at<usize>(ref index,slot)-1);if(c.module_index==s.module_index && c.declaration==s.declaration){break;}slot=(slot+1)&8191;}
        let found=*v::at<usize>(ref index,slot);if(found!=0){let c=*v::at<records::Callable>(callables,found-1);if(c.template || c.result!=s.result || c.parameter_count!=s.parameter_count || c.effects!=s.effects || c.must_use!=s.must_use || c.unsafe_call!=s.unsafe_call || c.foreign!=s.foreign){return 9;}let link=signatures::linker_name(metadata,i);if(!phase::charge(work,link.len+1)){return 10;}if(!bytes::equal(bytes::view(names)[c.link_start..c.link_end],link)){return 9;}var usize j=0;while(j<c.parameter_count){if(!phase::charge(work,1)){return 10;}if(*v::at<usize>(parameters,c.parameter_start+j)!=signatures::parameter_at(metadata,s.parameter_start+j)){return 9;}j+=1;}}
        else{if(v::len<records::Callable>(callables)>=limits::project_callables() || v::len<usize>(parameters)>32768 || s.parameter_count>32768-v::len<usize>(parameters)){return 10;}let first=v::len<usize>(parameters);var usize j=0;while(j<s.parameter_count){if(!phase::charge(work,1)){return 10;}if(!v::push<usize>(parameters,signatures::parameter_at(metadata,s.parameter_start+j)).is_ok){return 11;}j+=1;}let link=signatures::linker_name(metadata,i);if(!phase::charge(work,link.len+1) || bytes::len(names)>1048576 || link.len>1048576-bytes::len(names)){return 10;}let start=bytes::len(names);if(!bytes::append(names,link).is_ok){return 11;}if(!v::push<records::Callable>(callables,records::Callable{.module_index=s.module_index,.declaration=s.declaration,.template=false,.result=s.result,.parameter_start=first,.parameter_count=s.parameter_count,.effects=s.effects,.must_use=s.must_use,.unsafe_call=s.unsafe_call,.foreign=s.foreign,.link_start=start,.link_end=bytes::len(names)}).is_ok){return 11;}let target=v::at_mut<usize>(ref mut index,slot);*target=v::len<records::Callable>(callables);}i+=1;
    }return 0;
}
fn Result<View,u32> copy(ref checked::Program input,ref signatures::Program metadata,bindings::Bindings stamps,ref mut usize work){
    let program=checked::ir_program(input);let count=clean::type_count(ir::cleanup_ir(program));if(count==0 || count>4096 || ir::layout_count(program)>1024 || ir::field_count(program)>limits::project_fields() || ir::callable_count(program)>limits::project_callables() || ir::parameter_count(program)>32768 || ir::name_bytes(program)>1048576){return .err(9);}
    var v::Vec<types::Type> types=v::empty<types::Type>();var v::Vec<records::Shape> shapes=v::empty<records::Shape>();var v::Vec<records::Field> fields=v::empty<records::Field>();var v::Vec<records::Callable> callables=v::empty<records::Callable>();var v::Vec<usize> parameters=v::empty<usize>();var usize i=1;
    if(signatures::type_count(metadata)>count){return .err(9);}while(i<=count){if(!phase::charge(work,1)){return .err(10);}let type=clean::type_at(ir::cleanup_ir(program),i);if(i<=signatures::type_count(metadata) && !ids::same(type,signatures::type_at(metadata,i))){return .err(9);}if(!v::push<types::Type>(ref mut types,type).is_ok){return .err(11);}i+=1;}i=0;
    while(i<ir::layout_count(program)){if(!phase::charge(work,1)){return .err(10);}if(!v::push<records::Shape>(ref mut shapes,ir::layout_at(program,i).value).is_ok){return .err(11);}i+=1;}i=0;
    while(i<ir::field_count(program)){if(!phase::charge(work,1)){return .err(10);}if(!v::push<records::Field>(ref mut fields,ir::field_at(program,i)).is_ok){return .err(11);}i+=1;}i=0;
    while(i<ir::callable_count(program)){if(!phase::charge(work,1)){return .err(10);}if(!v::push<records::Callable>(ref mut callables,ir::callable_at(program,i)).is_ok){return .err(11);}i+=1;}i=0;
    while(i<ir::parameter_count(program)){if(!phase::charge(work,1)){return .err(10);}if(!v::push<usize>(ref mut parameters,ir::parameter_at(program,i)).is_ok){return .err(11);}i+=1;}
    if(!phase::charge(work,ir::name_bytes(program)+1)){return .err(10);}var bytes::Buffer text=switch(bytes::copy(ir::names_view(program))){case .ok(p):{p}case .err(_):{return .err(11);}};let completed=complete_signatures(metadata,ref mut callables,ref mut parameters,ref mut text,work);if(completed!=0){return .err(completed);}let identities=switch(names(input,program,metadata,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return .ok(View{.types,.shapes,.fields,.callables,.parameters,.names=text,.identities,.provenance=stamps});
}
// Freshness is checked before any public facts are copied. Copying has its own
// 1MiB phase history, folded exactly once into the unchanged 16MiB caller ledger.
// No parser, body typing, Flow analysis or executable IR is copied or rebuilt.
pub fn Result<View,u32> build(ref checked::Program input,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){
    let code=checked::current(input,sources,metadata,table,work);if(code!=0){return .err(code);}var usize provenance_work=0;let stamps=switch(bindings::copy(checked::source_provenance(input),ref mut provenance_work)){case .ok(p):{p}case .err(e):{discard budget::charge(work,provenance_work);return .err(e);}};if(!budget::charge(work,provenance_work)){return .err(10);}
    var usize measured=0;switch(copy(input,metadata,stamps,ref mut measured)){case .ok(p):{if(!budget::charge(work,measured)){return .err(10);}return .ok(p);}case .err(e):{discard budget::charge(work,measured);return .err(e);}};return .err(9);
}

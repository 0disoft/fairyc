module module_metadata_artifact_decode;
import module_metadata_artifact_model as model;
import module_unit_body_ir_model as ir;
import module_unit_interfaces as interfaces;
import module_unit_cleanup_mir_model as clean;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import candidate_limits as limits;
import std::bytes as bytes;
import std::vec as v;
// A decoded bounded component still requires full Program verification and
// exact source/package rebinding before executable or cache-hit publication.
pub fn Result<model::Metadata,u32> decode(ref u8[] raw,ref clean::Program cleaned,usize modules,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}if(modules>limits::source_files()){return .err(1002);}
    var cursor=switch(wire::cursor(raw,b"FYMETA00",work)){case .ok(c):{c}case .err(code):{return .err(code);}};
    let stored_modules=wire::number_from(raw,ref mut cursor,work);let types=wire::number_from(raw,ref mut cursor,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    if(stored_modules!=modules || types!=clean::type_count(cleaned)){return .err(1001);}var usize i=0;
    let layout_count=wire::count_from(raw,ref mut cursor,1024,104,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}var v::Vec<ir::Layout> layouts=v::empty<ir::Layout>();i=0;while(i<layout_count){let row=ir::Layout{.value=interfaces::Shape{.type=wire::number_from(raw,ref mut cursor,work),.module_index=wire::number_from(raw,ref mut cursor,work),.declaration=wire::number_from(raw,ref mut cursor,work),.field_start=wire::number_from(raw,ref mut cursor,work),.field_count=wire::number_from(raw,ref mut cursor,work),.abilities=wire::word_from(raw,ref mut cursor,work),.repr_c=wire::flag_from(raw,ref mut cursor,work),.start=wire::number_from(raw,ref mut cursor,work),.end=wire::number_from(raw,ref mut cursor,work),.cleanup_nonempty=wire::flag_from(raw,ref mut cursor,work),.cleanup_effects=wire::word_from(raw,ref mut cursor,work)},.name_start=wire::number_from(raw,ref mut cursor,work),.name_end=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<ir::Layout>(ref mut layouts,row).is_ok){return .err(1003);}i+=1;}
    let field_count=wire::count_from(raw,ref mut cursor,limits::project_fields(),88,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();i=0;while(i<field_count){let row=interfaces::Field{.kind=wire::word_from(raw,ref mut cursor,work),.type=wire::number_from(raw,ref mut cursor,work),.module_index=wire::number_from(raw,ref mut cursor,work),.declaration=wire::number_from(raw,ref mut cursor,work),.source_index=wire::number_from(raw,ref mut cursor,work),.start=wire::number_from(raw,ref mut cursor,work),.end=wire::number_from(raw,ref mut cursor,work),.name_start=wire::number_from(raw,ref mut cursor,work),.name_end=wire::number_from(raw,ref mut cursor,work),.payload_start=wire::number_from(raw,ref mut cursor,work),.payload_count=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<interfaces::Field>(ref mut fields,row).is_ok){return .err(1003);}i+=1;}
    let callable_count=wire::count_from(raw,ref mut cursor,limits::project_callables(),96,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();i=0;while(i<callable_count){let row=interfaces::Callable{.module_index=wire::number_from(raw,ref mut cursor,work),.declaration=wire::number_from(raw,ref mut cursor,work),.template=wire::flag_from(raw,ref mut cursor,work),.result=wire::number_from(raw,ref mut cursor,work),.parameter_start=wire::number_from(raw,ref mut cursor,work),.parameter_count=wire::number_from(raw,ref mut cursor,work),.effects=wire::word_from(raw,ref mut cursor,work),.must_use=wire::flag_from(raw,ref mut cursor,work),.unsafe_call=wire::flag_from(raw,ref mut cursor,work),.foreign=wire::flag_from(raw,ref mut cursor,work),.link_start=wire::number_from(raw,ref mut cursor,work),.link_end=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<interfaces::Callable>(ref mut callables,row).is_ok){return .err(1003);}i+=1;}
    let parameter_count=wire::count_from(raw,ref mut cursor,32768,8,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}var v::Vec<usize> parameters=v::empty<usize>();i=0;while(i<parameter_count){let type=wire::number_from(raw,ref mut cursor,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<usize>(ref mut parameters,type).is_ok){return .err(1003);}i+=1;}
    let length=wire::count_from(raw,ref mut cursor,1048576,1,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!budget::charge(work,length+1)){return .err(1002);}
    let names=switch(bytes::copy(raw[cursor.offset..cursor.offset+length])){case .ok(p):{p}case .err(_):{return .err(1003);}};cursor.offset+=length;let code=wire::end(raw,ref mut cursor,work);if(code!=0){return .err(code);}
    return model::assemble(layouts,fields,callables,parameters,names,cleaned,modules,work);
}

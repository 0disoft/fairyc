module module_cleanup_artifact_decode;
import module_unit_cleanup_mir_model as ir;
import module_types as types;
import candidate_limits as limits;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import std::vec as v;
// Preflight counts against remaining bytes before allocation. Reconstruction
// re-enters cleanup verification; no partially decoded owner escapes.
pub fn Result<ir::Program,u32> decode(ref u8[] raw,ref mut usize work){
    var cursor=switch(wire::cursor(raw,b"FYCLEAN0",work)){case .ok(c):{c}case .err(code):{return .err(code);}};let project=wire::flag_from(raw,ref mut cursor,work);var usize i=0;
    let types_count=wire::count_from(raw,ref mut cursor,4096,104,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<types::Type> types=v::empty<types::Type>();i=0;while(i<types_count){let row=types::Type{.kind=wire::word_from(raw,ref mut cursor,work),.scalar=wire::word_from(raw,ref mut cursor,work),.module_index=wire::number_from(raw,ref mut cursor,work),.declaration=wire::number_from(raw,ref mut cursor,work),.first=wire::number_from(raw,ref mut cursor,work),.second=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(row.module_index>=limits::source_files() || row.declaration>1048576 || row.first>4096 || row.second>4096){return .err(1001);}if(!v::push<types::Type>(ref mut types,row).is_ok){return .err(1003);}i+=1;}
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<ir::Glue> glues=v::empty<ir::Glue>();i=0;while(i<types_count){let row=ir::Glue{.kind=wire::word_from(raw,ref mut cursor,work),.frame=wire::number_from(raw,ref mut cursor,work),.nonempty=wire::flag_from(raw,ref mut cursor,work),.effects=wire::word_from(raw,ref mut cursor,work),.abilities=wire::word_from(raw,ref mut cursor,work),.field_start=wire::number_from(raw,ref mut cursor,work),.field_count=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<ir::Glue>(ref mut glues,row).is_ok){return .err(1003);}i+=1;}
    let frames_count=wire::count_from(raw,ref mut cursor,limits::project_frames(),64,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<ir::Frame> frames=v::empty<ir::Frame>();i=0;while(i<frames_count){let row=ir::Frame{.kind=wire::word_from(raw,ref mut cursor,work),.context=wire::number_from(raw,ref mut cursor,work),.module_index=wire::number_from(raw,ref mut cursor,work),.declaration=wire::number_from(raw,ref mut cursor,work),.result=wire::number_from(raw,ref mut cursor,work),.effects=wire::word_from(raw,ref mut cursor,work),.receiver=wire::number_from(raw,ref mut cursor,work),.receiver_type=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<ir::Frame>(ref mut frames,row).is_ok){return .err(1003);}i+=1;}
    let fields_count=wire::count_from(raw,ref mut cursor,limits::project_fields(),24,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<ir::Field> fields=v::empty<ir::Field>();i=0;while(i<fields_count){let row=ir::Field{.owner=wire::number_from(raw,ref mut cursor,work),.ordinal=wire::number_from(raw,ref mut cursor,work),.type=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<ir::Field>(ref mut fields,row).is_ok){return .err(1003);}i+=1;}
    let operands_count=wire::count_from(raw,ref mut cursor,16384,48,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<ir::Operand> operands=v::empty<ir::Operand>();i=0;while(i<operands_count){let row=ir::Operand{.frame=wire::number_from(raw,ref mut cursor,work),.kind=wire::word_from(raw,ref mut cursor,work),.id=wire::number_from(raw,ref mut cursor,work),.type=wire::number_from(raw,ref mut cursor,work),.start=wire::number_from(raw,ref mut cursor,work),.end=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<ir::Operand>(ref mut operands,row).is_ok){return .err(1003);}i+=1;}
    let drops_count=wire::count_from(raw,ref mut cursor,16384,32,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<ir::Drop> drops=v::empty<ir::Drop>();i=0;while(i<drops_count){let row=ir::Drop{.operand=wire::number_from(raw,ref mut cursor,work),.exit=wire::word_from(raw,ref mut cursor,work),.start=wire::number_from(raw,ref mut cursor,work),.end=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<ir::Drop>(ref mut drops,row).is_ok){return .err(1003);}i+=1;}
    let code=wire::end(raw,ref mut cursor,work);if(code!=0){return .err(code);}var usize measured=0;let restored=switch(ir::assemble_artifact(types,frames,glues,fields,operands,drops,project,ref mut measured)){case .ok(p):{p}case .err(e):{if(!budget::charge(work,measured)){return .err(1002);}return .err(e.code);}};if(!budget::charge(work,measured)){return .err(1002);}return .ok(restored);
}

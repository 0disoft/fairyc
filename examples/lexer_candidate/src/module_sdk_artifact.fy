module module_sdk_artifact;
import module_unit_sdk_sources as sources;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import candidate_limits as limits;
import std::bytes as bytes;
import std::vec as v;

// FYSDK000: module count, ordered Unit rows, exact bundled source bytes.
// This component alone grants no checked-package or executable authority.
pub fn Result<wire::Payload,u32> encode(ref sources::Sources p,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}var usize measured=0;
    let checked=sources::check(p,ref mut measured);
    if(!budget::charge(work,measured)){return .err(1002);}if(checked.code!=0){return .err(checked.code);}
    var wire::Status status=wire::Status{.code=0};var out=wire::writer(b"FYSDK000",ref mut status,work);
    wire::number(ref mut out,sources::module_count(p),ref mut status,work);
    wire::number(ref mut out,sources::count(p),ref mut status,work);var usize i=0;
    while(i<sources::count(p)){let row=sources::at(p,i);
        wire::number(ref mut out,row.module_index,ref mut status,work);
        wire::number(ref mut out,row.sdk_id,ref mut status,work);
        wire::number(ref mut out,row.start,ref mut status,work);
        wire::number(ref mut out,row.end,ref mut status,work);i+=1;
    }
    wire::number(ref mut out,sources::data(p).len,ref mut status,work);
    if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}
    let code=budget::raw(ref mut out,sources::data(p),work);
    if(code!=0){return .err(if(code==4){1003}else{1002});}
    return wire::finish(out,status);
}
pub fn Result<sources::Sources,u32> decode(ref u8[] raw,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}
    var cursor=switch(wire::cursor(raw,b"FYSDK000",work)){case .ok(c):{c}case .err(code):{return .err(code);}};
    let modules=wire::number_from(raw,ref mut cursor,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(modules>limits::source_files()){return .err(1002);}
    let count=wire::count_from(raw,ref mut cursor,8,32,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<sources::Unit> units=v::empty<sources::Unit>();var usize i=0;
    while(i<count){let row=sources::Unit{.module_index=wire::number_from(raw,ref mut cursor,work),.sdk_id=wire::number_from(raw,ref mut cursor,work),.start=wire::number_from(raw,ref mut cursor,work),.end=wire::number_from(raw,ref mut cursor,work)};
        if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
        if(!v::push<sources::Unit>(ref mut units,row).is_ok){return .err(1003);}i+=1;
    }
    let length=wire::count_from(raw,ref mut cursor,8*limits::source_bytes(),1,work);
    if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    if(!budget::charge(work,length+1)){return .err(1002);}
    let data=switch(bytes::copy(raw[cursor.offset..cursor.offset+length])){case .ok(p):{p}case .err(_):{return .err(1003);}};
    cursor.offset+=length;let code=wire::end(raw,ref mut cursor,work);if(code!=0){return .err(code);}
    var usize measured=0;let restored=switch(sources::assemble_artifact(modules,units,data,ref mut measured)){case .ok(p):{p}case .err(e):{if(!budget::charge(work,measured)){return .err(1002);}return .err(e.code);}};
    if(!budget::charge(work,measured)){return .err(1002);}
    return .ok(restored);
}

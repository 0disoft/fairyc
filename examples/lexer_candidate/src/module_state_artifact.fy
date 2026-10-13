module module_state_artifact;
import module_state_layout as state;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// FYSTATE0 stores the original identities, not a trusted-hash flag. Both
// directions recompute field prefixes and layout hashes with the same checker.
fn u32 checked(ref state::Plan p,ref mut usize work){var usize measured=0;let code=state::validate(p,ref mut measured);if(!budget::charge(work,measured)){return 1002;}return if(code==0){0}else if(code==10){1002}else if(code==11){1003}else{1001};}
pub fn Result<wire::Payload,u32> encode(ref state::Plan p,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}let code=checked(p,work);if(code!=0){return .err(code);}
    var wire::Status status=wire::Status{.code=0};var out=wire::writer(b"FYSTATE0",ref mut status,work);
    wire::number(ref mut out,state::count(p),ref mut status,work);var usize i=0;
    while(i<state::count(p)){let row=state::at(p,i);
        wire::number(ref mut out,row.owner,ref mut status,work);wire::number(ref mut out,row.module_index,ref mut status,work);wire::number(ref mut out,row.declaration,ref mut status,work);
        wire::number(ref mut out,row.start,ref mut status,work);wire::number(ref mut out,row.end,ref mut status,work);wire::number(ref mut out,row.uid_start,ref mut status,work);wire::number(ref mut out,row.uid_end,ref mut status,work);
        wire::number(ref mut out,row.field_start,ref mut status,work);wire::number(ref mut out,row.field_count,ref mut status,work);var usize j=0;while(j<8){wire::number(ref mut out,row.hash[j] as usize,ref mut status,work);j+=1;}
        if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;
    }
    wire::number(ref mut out,state::field_count(p),ref mut status,work);i=0;
    while(i<state::field_count(p)){let row=state::field_at(p,i);
        wire::number(ref mut out,row.kind as usize,ref mut status,work);wire::number(ref mut out,row.key as usize,ref mut status,work);wire::number(ref mut out,row.value as usize,ref mut status,work);
        wire::number(ref mut out,row.name_start,ref mut status,work);wire::number(ref mut out,row.name_end,ref mut status,work);wire::number(ref mut out,row.start,ref mut status,work);wire::number(ref mut out,row.end,ref mut status,work);
        var usize j=0;while(j<8){wire::number(ref mut out,row.prefix[j] as usize,ref mut status,work);j+=1;}if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}i+=1;
    }
    wire::number(ref mut out,state::name_bytes(p),ref mut status,work);if(wire::output_status(ref status)!=0){return .err(wire::output_status(ref status));}
    let written=budget::raw(ref mut out,state::names(p),work);if(written!=0){return .err(if(written==4){1003}else{1002});}return wire::finish(out,status);
}
pub fn Result<state::Plan,u32> decode(ref u8[] raw,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}var cursor=switch(wire::cursor(raw,b"FYSTATE0",work)){case .ok(c):{c}case .err(code):{return .err(code);}};
    let count=wire::count_from(raw,ref mut cursor,512,136,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
    var v::Vec<state::Layout> layouts=v::empty<state::Layout>();var usize i=0;
    while(i<count){var row=state::Layout{.owner=wire::number_from(raw,ref mut cursor,work),.module_index=wire::number_from(raw,ref mut cursor,work),.declaration=wire::number_from(raw,ref mut cursor,work),.start=wire::number_from(raw,ref mut cursor,work),.end=wire::number_from(raw,ref mut cursor,work),.uid_start=wire::number_from(raw,ref mut cursor,work),.uid_end=wire::number_from(raw,ref mut cursor,work),.field_start=wire::number_from(raw,ref mut cursor,work),.field_count=wire::number_from(raw,ref mut cursor,work),.hash=[0;8]};
        var usize j=0;while(j<8){row.hash[j]=wire::word_from(raw,ref mut cursor,work);j+=1;}if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}
        if(!v::push<state::Layout>(ref mut layouts,row).is_ok){return .err(1003);}i+=1;
    }
    let fields_count=wire::count_from(raw,ref mut cursor,524288,120,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}var v::Vec<state::Field> fields=v::empty<state::Field>();i=0;
    while(i<fields_count){var row=state::Field{.kind=wire::word_from(raw,ref mut cursor,work),.key=wire::word_from(raw,ref mut cursor,work),.value=wire::word_from(raw,ref mut cursor,work),.name_start=wire::number_from(raw,ref mut cursor,work),.name_end=wire::number_from(raw,ref mut cursor,work),.start=wire::number_from(raw,ref mut cursor,work),.end=wire::number_from(raw,ref mut cursor,work),.prefix=[0;8]};
        var usize j=0;while(j<8){row.prefix[j]=wire::word_from(raw,ref mut cursor,work);j+=1;}if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!v::push<state::Field>(ref mut fields,row).is_ok){return .err(1003);}i+=1;
    }
    let size=wire::count_from(raw,ref mut cursor,1048576,1,work);if(wire::status(ref cursor)!=0){return .err(wire::status(ref cursor));}if(!budget::charge(work,size+1)){return .err(1002);}
    let names=switch(bytes::copy(raw[cursor.offset..cursor.offset+size])){case .ok(p):{p}case .err(_):{return .err(1003);}};cursor.offset+=size;let code=wire::end(raw,ref mut cursor,work);if(code!=0){return .err(code);}
    let p=state::assemble(names,layouts,fields);let verified=checked(ref p,work);if(verified!=0){return .err(verified);}return .ok(p);
}

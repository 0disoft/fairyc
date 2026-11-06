module module_state_graph_values;
import module_state_graph_model as m;
import project_cache_encoding as budget;
import std::text as text;
import std::vec as v;

// All scans are charged before touching their next row. 0 means unavailable.
pub fn Result<u32,u32> lookup(ref v::Vec<m::Value> values,usize id,ref mut usize work){
    var usize i=0;while(i<v::len<m::Value>(values)){if(!budget::charge(work,1)){return .err(1002);}let row=*v::at<m::Value>(values,i);if(row.id==id){return .ok(row.codec);}i+=1;}return .ok(0);
}
pub fn u32 define(ref mut v::Vec<m::Value> values,m::Value row,ref mut usize work){
    if(row.id==0 || row.id>4294967295 || row.codec<1 || row.codec>7){return 1001;}
    let before=switch(lookup(values,row.id,work)){case .ok(t):{t}case .err(code):{return code;}};if(before!=0){return 1001;}
    if(!budget::charge(work,1)){return 1002;}return if(v::push<m::Value>(values,row).is_ok){0}else{1003};
}
pub fn bool absent(m::Operand value){return value.input==0 && value.codec==0 && value.start==0 && value.end==0;}
// Dynamic bytes/text are raw canonical bytes, including a present empty value.
// Storage limits belong to the execution attempt, not this type/codec verifier.
pub fn u32 canonical(u32 codec,ref u8[] raw,ref mut usize work){
    if(raw.len>m::byte_limit() || !budget::charge(work,raw.len+1)){return 1002;}
    if(codec==1){return if(raw.len==1 && raw[0]<=1){0}else{1001};}
    if(codec==2){return if(raw.len==4){0}else{1001};}
    if(codec==3){return if(raw.len==8){0}else{1001};}
    if(codec==4){return if(raw.len==20){0}else{1001};}
    if(codec==5){return if(raw.len==32){0}else{1001};}
    if(codec==6){return 0;}
    if(codec==7){return if(text::validate_utf8(raw).is_ok){0}else{1001};}return 1001;
}
pub fn u32 operand(m::Operand value,u32 expected,ref u8[] data,ref v::Vec<m::Value> live,ref mut usize work){
    if(value.input!=0){if(value.codec!=0 || value.start!=0 || value.end!=0){return 1001;}let found=switch(lookup(live,value.input,work)){case .ok(t):{t}case .err(code):{return code;}};return if(found==expected){0}else{1001};}
    if(value.codec!=expected || value.start>value.end || value.end>data.len){return 1001;}return canonical(value.codec,data[value.start..value.end],work);
}
pub fn u32 category(ref u8[] raw,ref mut usize work){
    if(!budget::charge(work,raw.len+1)){return 1002;}if(raw.len==0 || raw.len>127 || raw[0]==45 || raw[raw.len-1]==45){return 1001;}
    var usize i=0;while(i<raw.len){let c=raw[i];if(!(c>=97 && c<=122 || c>=48 && c<=57 || c==45)){return 1001;}i+=1;}return 0;
}

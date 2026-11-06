module module_state_graph_native_text;
import module_state_graph_native_plan as plan;
import project_cache_encoding as budget;
import std::bytes as bytes;

// One sticky failure and shared phase ledger; a partial C buffer is never an artifact.
pub struct State{u32 code;usize work;usize site;usize term;bool audit;}
pub fn usize byte_limit(){return 8388608;}
pub fn bool text(ref mut bytes::Buffer out,ref mut State s,ref u8[] raw){
    if(s.code!=0){return false;}
    if(raw.len>byte_limit() || bytes::len(out)>byte_limit()-raw.len || !budget::charge(ref mut s.work,raw.len+1)){s.code=1002;return false;}
    if(!bytes::append(out,raw).is_ok){s.code=1003;return false;}return true;
}
pub fn bool number(ref mut bytes::Buffer out,ref mut State s,usize value){
    if(s.code!=0){return false;}var usize digits=1;var usize rest=value;while(rest>=10){rest/=10;digits+=1;}
    if(bytes::len(out)>byte_limit()-digits || !budget::charge(ref mut s.work,digits+1)){s.code=1002;return false;}
    if(!bytes::append_usize(out,value).is_ok){s.code=1003;return false;}return true;
}
pub fn bool checked(ref mut bytes::Buffer out,ref mut State s){return text(out,s,b"    status = ");}
pub fn bool end(ref mut bytes::Buffer out,ref mut State s){return text(out,s,b"; if (status != 0) { goto finished; }\n");}
pub fn bool value(ref mut bytes::Buffer out,ref mut State s,ref plan::Plan p,usize body,usize id){
    if(s.code!=0){return false;}let slot=switch(plan::value_slot(p,body,id,ref mut s.work)){case .ok(n):{n}case .err(code):{s.code=code;return false;}};
    return text(out,s,b"values[") && number(out,s,slot) && text(out,s,b"]");
}
pub fn bool borrow(ref mut bytes::Buffer out,ref mut State s,ref plan::Plan p,usize body,usize id){
    if(s.code!=0){return false;}let slot=switch(plan::borrow_slot(p,body,id,ref mut s.work)){case .ok(n):{n}case .err(code):{s.code=code;return false;}};
    return text(out,s,b"borrows[") && number(out,s,slot) && text(out,s,b"]");
}
pub fn bool octets(ref mut bytes::Buffer out,ref mut State s,ref u8[] raw){
    var usize i=0;while(i<raw.len){if(i!=0 && !text(out,s,b", ")){return false;}if(!hex(out,s,raw[i] as usize)){return false;}i+=1;}return true;
}
fn bool hex(ref mut bytes::Buffer out,ref mut State s,usize value){let digits=b"0123456789abcdef";return text(out,s,b"0x") && text(out,s,digits[value/16..value/16+1]) && text(out,s,digits[value%16..value%16+1]);}
pub fn bool digest(ref mut bytes::Buffer out,ref mut State s,u32[8] hash){
    var usize i=0;var u32 shift=24;while(i<32){if(i!=0 && !text(out,s,b", ")){return false;}if(!hex(out,s,((hash[i/4]>>shift)&255) as usize)){return false;}i+=1;if(i%4==0){shift=24;}else{shift-=8;}}return true;
}

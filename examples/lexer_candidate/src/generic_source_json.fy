module generic_source_json;
import module_public_interface_write as write;
import module_state_layout as budget;
import project_json_string as strings;
import std::bytes as bytes;

// Shared bounded JSON writer for the source-AST definition identity.
// 9 malformed facts, 10 exhausted work/output, 11 allocation failure.
pub struct Ledger { usize used; u32 code; }
pub fn Ledger empty() { return Ledger { .used=0,.code=0 }; }
pub fn bool reject(ref mut Ledger work,u32 code) { if(work.code==0){work.code=code;}return false; }
pub fn bool charge(ref mut Ledger work,usize cost) {
    if(work.code!=0){return false;}return if(budget::charge(ref mut work.used,cost)){true}else{reject(work,10)};
}
pub fn bool raw(ref mut bytes::Buffer out,ref u8[] value,ref mut Ledger work) {
    if(work.code!=0){return false;}let code=write::raw(out,value,ref mut work.used);
    return if(code==0){true}else{reject(work,code)};
}
pub fn bool quoted(ref mut bytes::Buffer out,ref u8[] value,ref mut Ledger work) {
    if(work.code!=0){return false;}if(value.len>1048576){return reject(work,9);}
    // The established Go-compatible escaper has its own bounded history. Fold
    // that measured history into this identity ledger before publishing bytes.
    var usize meter=0;
    let encoded=switch(strings::quote(value,ref mut meter)) {
        case .ok(value): { value }
        case .err(code): { discard reject(work,if(code==1){10}else if(code==2){9}else{11});return false; }
    };
    return charge(work,meter) && raw(out,bytes::view(ref encoded),work);
}
pub fn bool number(ref mut bytes::Buffer out,usize value,ref mut Ledger work) {
    if(!charge(work,21)){return false;}var bytes::Buffer text=bytes::empty();
    if(!bytes::append_usize(ref mut text,value).is_ok){return reject(work,11);}
    return raw(out,bytes::view(ref text),work);
}
pub fn bool flag(ref mut bytes::Buffer out,bool value,ref mut Ledger work) { return raw(out,if(value){b"true"}else{b"false"},work); }
pub fn bool span(ref mut bytes::Buffer out,usize start,usize end,ref mut Ledger work) {
    if(start>end){return reject(work,9);}
    return raw(out,b"{\"StartByte\":",work) && number(out,start,work) && raw(out,b",\"EndByte\":",work) && number(out,end,work) && raw(out,b"}",work);
}

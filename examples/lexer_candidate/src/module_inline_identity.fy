module module_inline_identity;
import generic_source_json as json;
import sha256 as sha;
import std::bytes as bytes;

// Shared by source admission and detached typed-IR verification. JSON field
// spelling/order and null leaves match hir.InlineFunctionID exactly.
pub fn bool zero(u32[8] hash){var usize i=0;while(i<8){if(hash[i]!=0){return false;}i+=1;}return true;}
pub fn bool equal(u32[8] a,u32[8] b){var usize i=0;while(i<8){if(a[i]!=b[i]){return false;}i+=1;}return true;}
pub fn bool header(ref mut bytes::Buffer out,usize arity,ref mut json::Ledger work){return arity<=4 && json::raw(out,b"{\"Version\":\"fairyc.inline-function/v1\",\"Arity\":",work) && json::number(out,arity,work) && json::raw(out,b",\"Root\":",work);}
// Locals use a new canonical tuple while keeping the existing digest transport.
pub fn bool locals_header(ref mut bytes::Buffer out,usize arity,ref mut json::Ledger work){return arity<=4 && json::raw(out,b"{\"Version\":\"fairyc.inline-function/v2\",\"Arity\":",work) && json::number(out,arity,work) && json::raw(out,b",\"Root\":",work) && branch(out,b"bindings",work);}
pub fn bool calls_header(ref mut bytes::Buffer out,usize arity,bool locals,ref mut json::Ledger work){return arity<=4 && json::raw(out,b"{\"Version\":\"fairyc.inline-function/v3\",\"Arity\":",work) && json::number(out,arity,work) && json::raw(out,b",\"Root\":",work) && (!locals || branch(out,b"bindings",work));}
// Straight-line mutable bindings use SSA-versioned initializer positions.
pub fn bool steps_header(ref mut bytes::Buffer out,usize arity,ref mut json::Ledger work){return arity<=4 && json::raw(out,b"{\"Version\":\"fairyc.inline-function/v4\",\"Arity\":",work) && json::number(out,arity,work) && json::raw(out,b",\"Root\":",work) && branch(out,b"bindings",work);}
pub fn bool conditional_header(ref mut bytes::Buffer out,usize arity,bool locals,bool mutating,ref mut json::Ledger work){
    return arity<=4 && json::raw(out,b"{\"Version\":\"fairyc.inline-function/v5\",\"Arity\":",work) &&
        json::number(out,arity,work) && json::raw(out,b",\"Root\":",work) &&
        (!locals || branch(out,if(mutating){b"mutable-bindings"}else{b"bindings"},work));
}
pub fn bool comparison(ref u8[] op){
    return bytes::equal(op,b"==") || bytes::equal(op,b"!=") || bytes::equal(op,b"<") ||
        bytes::equal(op,b"<=") || bytes::equal(op,b">") || bytes::equal(op,b">=");
}
pub fn bool logical(ref u8[] op){return bytes::equal(op,b"&&") || bytes::equal(op,b"||");}
pub fn usize arm_position(usize depth){return 64+depth;}
pub fn bool statement_header(ref mut bytes::Buffer out,usize arity,bool loop,ref mut json::Ledger work){
    return arity<=4 && json::raw(out,if(loop){b"{\"Version\":\"fairyc.inline-function/v7\",\"Arity\":"}else{b"{\"Version\":\"fairyc.inline-function/v6\",\"Arity\":"},work) &&
        json::number(out,arity,work) && json::raw(out,b",\"Root\":",work);
}
pub fn bool empty_block(ref mut bytes::Buffer out,ref mut json::Ledger work){
    return json::raw(out,b"{\"Op\":\"block\",\"Value\":\"\",\"Children\":null}",work);
}
pub fn bool assignment(ref mut bytes::Buffer out,usize slot,ref mut json::Ledger work){
    return json::raw(out,b"{\"Op\":\"assign\",\"Value\":\"",work) && json::number(out,slot,work) &&
        json::raw(out,b"\",\"Children\":[",work);
}
pub fn bool boolean(ref mut bytes::Buffer out,ref u8[] value,ref mut json::Ledger work){
    return (bytes::equal(value,b"true") || bytes::equal(value,b"false")) &&
        json::raw(out,b"{\"Op\":\"literal\",\"Value\":",work) && json::quoted(out,value,work) && json::raw(out,b",\"Children\":null}",work);
}
pub fn bool call(ref mut bytes::Buffer out,ref u8[] name,bool arguments,ref mut json::Ledger work){return json::raw(out,b"{\"Op\":\"call\",\"Value\":",work) && json::quoted(out,name,work) && json::raw(out,if(arguments){b",\"Children\":["}else{b",\"Children\":null}"},work);}
pub fn bool leaf(ref mut bytes::Buffer out,ref u8[] op,usize value,ref mut json::Ledger work){return json::raw(out,b"{\"Op\":",work) && json::quoted(out,op,work) && json::raw(out,b",\"Value\":\"",work) && json::number(out,value,work) && json::raw(out,b"\",\"Children\":null}",work);}
pub fn bool operator(ref u8[] op,bool binary){return bytes::equal(op,b"+") || bytes::equal(op,b"-") || binary && (bytes::equal(op,b"*") || bytes::equal(op,b"/") || bytes::equal(op,b"%"));}
pub fn bool branch(ref mut bytes::Buffer out,ref u8[] op,ref mut json::Ledger work){return json::raw(out,b"{\"Op\":",work) && json::quoted(out,op,work) && json::raw(out,b",\"Value\":\"\",\"Children\":[",work);}
pub fn bool minimum(ref mut bytes::Buffer out,ref mut json::Ledger work){return json::raw(out,b"{\"Op\":\"literal\",\"Value\":\"-2147483648\",\"Children\":null}",work);}
pub fn Result<usize,u32> integer(ref u8[] raw){var usize at=0;var usize base=10;if(raw.len>=2 && raw[0]==48){if(raw[1]==120 || raw[1]==88){base=16;at=2;}else if(raw[1]==98 || raw[1]==66){base=2;at=2;}}var usize n=0;var usize digits=0;while(at<raw.len){let c=raw[at];at+=1;if(c==95){continue;}let d=if(c>=48 && c<=57){(c-48) as usize}else if(c>=97 && c<=102){(c-97+10) as usize}else if(c>=65 && c<=70){(c-65+10) as usize}else{base};if(d>=base || n>(2147483648-d)/base){return .err(9);}n=n*base+d;digits+=1;}return if(digits==0){.err(9)}else{.ok(n)};}
pub fn Result<u32[8],u32> finish(ref mut bytes::Buffer out,ref mut json::Ledger work){if(!json::raw(out,b"}",work)){return .err(work.code);}return .ok(sha::digest(bytes::view(out)));}

pub fn bool omitted(ref mut bytes::Buffer out,ref mut json::Ledger work){
    return json::raw(out,b"{\"Op\":\"omitted\",\"Value\":\"\",\"Children\":null}",work);
}
pub fn bool loop_control(ref mut bytes::Buffer out,bool breaking,ref mut json::Ledger work){
    return json::raw(out,if(breaking){b"{\"Op\":\"break\",\"Value\":\"\",\"Children\":null}"}else{b"{\"Op\":\"continue\",\"Value\":\"\",\"Children\":null}"},work);
}

module module_unit_c_literals;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_types as types;
import c_text as w;
import sha256 as sha;
import std::bytes as bytes;
import std::vec as v;

struct Literal {u32[8] hash;usize frame;usize node;}
pub fn bool descriptor(types::Type t){return t.kind==9 || t.kind==10 || t.kind==14;}
fn u32 compare(u32[8] a,u32[8] b){var usize i=0;while(i<8){if(a[i]!=b[i]){return if(a[i]<b[i]){1}else{2};}i+=1;}return 0;}
fn bool symbol(u32[8] hash,ref mut bytes::Buffer out,ref mut w::State s){
    if(!w::text(out,s,b"fy_unit_literal_")){return false;}let digits=b"0123456789abcdef";var usize i=0;
    while(i<8){var u32 shift=28;var usize j=0;while(j<8){let n=((hash[i]>>shift)&15) as usize;if(!w::text(out,s,digits[n..n+1])){return false;}j+=1;if(j<8){shift-=4;}}i+=1;}return true;
}
// Numeric static storage has no dependence on C string spelling or host encoding.
// Full SHA-256 symbol order and exact-byte collision checks match ADRs 0065-0067.
pub fn bool definitions(ref ir::Program p,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    var v::Vec<Literal> pool=v::empty<Literal>();var usize frame=1;
    while(frame<=ir::count(p)){let b=ir::at(p,frame);var usize node=0;while(node<ir::node_count(b)){
        if(!budget::charge(work,1)){s.code=1002;return false;}let n=ir::node_at(b,node);
        if(n.value.kind==1 && n.value.state==1 && descriptor(clean::type_at(ir::cleanup_ir(p),n.value.type))){
            let data=ir::node_cooked(b,node);if(!budget::charge(work,data.len+1)){s.code=1002;return false;}let entry=Literal{.hash=sha::digest(data),.frame,.node};var usize position=0;var bool duplicate=false;
            while(position<v::len<Literal>(ref pool)){if(!budget::charge(work,8)){s.code=1002;return false;}let prior=*v::at<Literal>(ref pool,position);let order=compare(entry.hash,prior.hash);if(order==0){let old=ir::node_cooked(ir::at(p,prior.frame),prior.node);if(!budget::charge(work,data.len+1)){s.code=1002;return false;}if(!bytes::equal(data,old)){s.code=1001;return false;}duplicate=true;break;}if(order==1){break;}position+=1;}
            if(!duplicate){if(!v::push<Literal>(ref mut pool,entry).is_ok){s.code=1003;return false;}var usize index=v::len<Literal>(ref pool)-1;while(index>position){if(!budget::charge(work,1)){s.code=1002;return false;}let prior=*v::at<Literal>(ref pool,index-1);{let slot=v::at_mut<Literal>(ref mut pool,index);*slot=prior;}index-=1;}{let slot=v::at_mut<Literal>(ref mut pool,position);*slot=entry;}}
        }node+=1;
    }frame+=1;}
    var usize i=0;while(i<v::len<Literal>(ref pool)){let entry=*v::at<Literal>(ref pool,i);let data=ir::node_cooked(ir::at(p,entry.frame),entry.node);
        if(!w::text(out,s,b"static const uint8_t ") || !symbol(entry.hash,out,s) || !w::text(out,s,b"[]={")){return false;}
        var usize j=0;while(j<data.len){if(!budget::charge(work,1)){s.code=1002;return false;}if(j!=0 && !w::text(out,s,b",")){return false;}if(!w::number(out,s,data[j] as usize)){return false;}j+=1;}
        if(data.len==0 && !w::text(out,s,b"0")){return false;}if(!w::text(out,s,b"};\n")){return false;}i+=1;
    }return true;
}
pub fn bool expression(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let n=ir::node_at(b,id-1);let t=clean::type_at(ir::cleanup_ir(p),n.value.type);let data=ir::node_cooked(b,id-1);
    if(n.value.kind!=1 || !descriptor(t) || n.literal_kind!=if(t.kind==9){7}else if(t.kind==10){8}else{9}){s.code=1001;return false;}if(!budget::charge(work,data.len+1)){s.code=1002;return false;}
    return w::text(out,s,b"fy_v_") && w::number(out,s,id) && w::text(out,s,b"=(struct fy_unit_t_") && w::number(out,s,n.value.type) && w::text(out,s,b"){.fy_data=") && symbol(sha::digest(data),out,s) && w::text(out,s,if(t.kind==14){b",.fy_storage_length="}else{b",.fy_length="}) && w::number(out,s,data.len) && w::text(out,s,b"};\n");
}

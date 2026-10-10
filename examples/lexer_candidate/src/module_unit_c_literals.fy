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
// 0 equal, 1 less, 2 greater, 3 exhausted. Charge examined words.
fn u32 compare(u32[8] a,u32[8] b,ref mut usize work){var usize i=0;while(i<8){if(!budget::charge(work,1)){return 3;}if(a[i]!=b[i]){return if(a[i]<b[i]){1}else{2};}i+=1;}return 0;}
fn bool grow(ref v::Vec<Literal> pool,ref mut v::Vec<usize> slots,ref mut w::State s,ref mut usize work){
    let usize size=if(v::len<usize>(slots)==0){64}else{v::len<usize>(slots)*2};if(!budget::charge(work,size)){s.code=1002;return false;}if(!v::reserve<usize>(slots,size).is_ok){s.code=1003;return false;}var usize i=0;let old=v::len<usize>(slots);while(i<old){let cell=v::at_mut<usize>(slots,i);*cell=0;i+=1;}while(i<size){if(!v::push<usize>(slots,0).is_ok){s.code=1003;return false;}i+=1;}
    i=0;while(i<v::len<Literal>(pool)){let entry=*v::at<Literal>(pool,i);var usize bucket=(entry.hash[0] as usize)&(size-1);while(*v::at<usize>(slots,bucket)!=0){if(!budget::charge(work,1)){s.code=1002;return false;}bucket=(bucket+1)&(size-1);}if(!budget::charge(work,1)){s.code=1002;return false;}{let cell=v::at_mut<usize>(slots,bucket);*cell=i+1;}i+=1;}return true;
}
fn bool insert(ref ir::Program p,Literal entry,ref mut v::Vec<Literal> pool,ref mut v::Vec<usize> slots,ref mut w::State s,ref mut usize work){
    if(v::len<usize>(slots)==0){if(!grow(pool,slots,s,work)){return false;}}
    var usize mask=v::len<usize>(slots)-1;var usize bucket=(entry.hash[0] as usize)&mask;
    while(true){if(!budget::charge(work,1)){s.code=1002;return false;}let id=*v::at<usize>(slots,bucket);if(id==0){if(v::len<Literal>(pool)>=v::len<usize>(slots)/2){if(!grow(pool,slots,s,work)){return false;}mask=v::len<usize>(slots)-1;bucket=(entry.hash[0] as usize)&mask;continue;}if(!v::push<Literal>(pool,entry).is_ok){s.code=1003;return false;}let cell=v::at_mut<usize>(slots,bucket);*cell=v::len<Literal>(pool);return true;}
        let prior=*v::at<Literal>(pool,id-1);let order=compare(entry.hash,prior.hash,work);if(order==3){s.code=1002;return false;}if(order==0){let data=ir::node_cooked(ir::at(p,entry.frame),entry.node);let old=ir::node_cooked(ir::at(p,prior.frame),prior.node);if(!budget::charge(work,data.len+1)){s.code=1002;return false;}if(!bytes::equal(data,old)){s.code=1001;return false;}return true;}bucket=(bucket+1)&mask;
    }return false;
}
fn bool swap(ref mut v::Vec<usize> order,usize first,usize second,ref mut usize work){if(!budget::charge(work,1)){return false;}let a=*v::at<usize>(order,first);let b=*v::at<usize>(order,second);{let cell=v::at_mut<usize>(order,first);*cell=b;}{let cell=v::at_mut<usize>(order,second);*cell=a;}return true;}
fn u32 ordered(ref v::Vec<Literal> pool,ref v::Vec<usize> order,usize a,usize b,ref mut usize work){return compare(v::at<Literal>(pool,*v::at<usize>(order,a)).hash,v::at<Literal>(pool,*v::at<usize>(order,b)).hash,work);}
fn bool sift(ref v::Vec<Literal> pool,ref mut v::Vec<usize> order,usize start,usize count,ref mut usize work){var usize root=start;while(root<count/2){var usize child=root*2+1;if(child+1<count){let c=ordered(pool,order,child,child+1,work);if(c==3){return false;}if(c==1){child+=1;}}let c=ordered(pool,order,root,child,work);if(c==3){return false;}if(c!=1){return true;}if(!swap(order,root,child,work)){return false;}root=child;}return true;}
fn bool sorted(ref v::Vec<Literal> pool,ref mut v::Vec<usize> order,ref mut w::State s,ref mut usize work){
    var usize i=0;while(i<v::len<Literal>(pool)){if(!budget::charge(work,1)){s.code=1002;return false;}if(!v::push<usize>(order,i).is_ok){s.code=1003;return false;}i+=1;}let count=v::len<usize>(order);i=count/2;while(i>0){i-=1;if(!sift(pool,order,i,count,work)){s.code=1002;return false;}}i=count;while(i>1){i-=1;if(!swap(order,0,i,work) || !sift(pool,order,0,i,work)){s.code=1002;return false;}}return true;
}
fn bool symbol(u32[8] hash,ref mut bytes::Buffer out,ref mut w::State s){
    if(!w::text(out,s,b"fy_unit_literal_")){return false;}let digits=b"0123456789abcdef";var usize i=0;
    while(i<8){var u32 shift=28;var usize j=0;while(j<8){let n=((hash[i]>>shift)&15) as usize;if(!w::text(out,s,digits[n..n+1])){return false;}j+=1;if(j<8){shift-=4;}}i+=1;}return true;
}
// Numeric static storage has no dependence on C string spelling or host encoding.
// Full SHA-256 symbol order and exact-byte collision checks match ADRs 0065-0067.
pub fn bool definitions(ref ir::Program p,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    var v::Vec<Literal> pool=v::empty<Literal>();var v::Vec<usize> slots=v::empty<usize>();var usize frame=1;
    while(frame<=ir::count(p)){let b=ir::at(p,frame);var usize node=0;while(node<ir::node_count(b)){
        if(!budget::charge(work,1)){s.code=1002;return false;}let n=ir::node_at(b,node);
        if(n.value.kind==1 && n.value.state==1 && descriptor(clean::type_at(ir::cleanup_ir(p),n.value.type))){
            let data=ir::node_cooked(b,node);if(!budget::charge(work,data.len+1)){s.code=1002;return false;}let entry=Literal{.hash=sha::digest(data),.frame,.node};if(!insert(p,entry,ref mut pool,ref mut slots,s,work)){return false;}
        }node+=1;
    }frame+=1;}
    var v::Vec<usize> order=v::empty<usize>();if(!sorted(ref pool,ref mut order,s,work)){return false;}var usize i=0;while(i<v::len<Literal>(ref pool)){let entry=*v::at<Literal>(ref pool,*v::at<usize>(ref order,i));let data=ir::node_cooked(ir::at(p,entry.frame),entry.node);
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

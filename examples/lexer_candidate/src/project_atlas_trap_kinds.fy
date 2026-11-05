module project_atlas_trap_kinds;
import std::bytes as bytes;

// Pure names/masks only. This module grants no CFG or source authority.
pub fn ref u8[] category(usize index){
    assert(index<6);let names=b"assertion-failed|bounds|division-by-zero|integer-overflow|invalid-shift|signed-division-overflow";
    var usize start=0;var usize i=0;while(i<index){while(names[start]!=124){start+=1;}start+=1;i+=1;}
    var usize end=start;while(end<names.len && names[end]!=124){end+=1;}return names[start..end];
}
pub fn bool contains(u32 mask,usize index){assert(index<6);let u32 bit=if(index==0){1}else if(index==1){2}else if(index==2){4}else if(index==3){8}else if(index==4){16}else{32};return mask & bit!=0;}
pub fn u32 arithmetic(ref u8[] op,u32 scalar,bool unary){
    if(scalar<1 || scalar>5){return 0;}
    if(unary){return if(bytes::equal(op,b"-")){8}else{0};}
    if(bytes::equal(op,b"/") || bytes::equal(op,b"%")){return if(scalar==1 || scalar==2){36}else{4};}
    if(bytes::equal(op,b"+") || bytes::equal(op,b"-") || bytes::equal(op,b"*")){return 8;}
    return if(bytes::equal(op,b"<<")){24}else if(bytes::equal(op,b">>")){16}else{0};
}

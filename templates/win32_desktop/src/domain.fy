module domain;
import std::bytes as bytes;
// Replace this pure function with your app's behavior. Both GUI and CLI call it.
pub fn bool transform(ref u8[] input,ref mut bytes::Buffer output){
    return bytes::clear(output).is_ok && bytes::append(output,if(input.len==0){b"내용을 입력하세요."}else{b"입력: "}).is_ok &&
        bytes::append(output,input).is_ok;
}
pub fn i32 key_action(i32 key,i32 flags){
    if(flags!=0 && flags!=1){return 0;}
    if(key==9){return if(flags==1){3}else{2};}
    if(flags!=0){return 0;}
    return if(key==13){1}else{if(key==27){4}else{0}};
}

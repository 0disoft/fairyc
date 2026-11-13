module comparison;
import std::bytes as bytes;
fn bool whitespace(u8 value){return value==32 || (value>=9 && value<=13);}
pub fn bool normalize(ref u8[] input,ref mut u8[] output){
    var usize start=0;var usize end=input.len;
    while(start<end && whitespace(input[start])){start+=1;}
    while(end>start && whitespace(input[end-1])){end-=1;}
    if(end-start!=64 || output.len!=64){return false;}
    for(var usize i=0;i<64;i+=1){
        var u8 value=input[start+i];if(value>=65 && value<=70){value+=32;}
        if(!((value>=48 && value<=57) || (value>=97 && value<=102))){return false;}
        output[i]=value;
    }
    return true;
}
// 0 empty, 1 invalid, 2 no usable calculated result, 3 match, 4 mismatch.
pub fn u32 classify(ref u8[] input,ref u8[] actual,bool ready){
    var bool empty=true;let length=input.len;
    for(var usize i=0;i<length;i+=1){if(!whitespace(input[i])){empty=false;}}
    if(empty){return 0;}
    var u8[64] expected=[0;64];
    if(!normalize(input,ref mut expected)){return 1;}
    if(!ready || actual.len!=64){return 2;}
    return if(bytes::equal(actual,ref expected)){3}else{4};
}
pub fn bool append_report(ref mut bytes::Buffer out,ref u8[] expected,bool matches){
    let length=bytes::view(out).len;
    if(expected.len!=64 || length<2 || !bytes::equal(bytes::view(out)[length-2..],b"}\n")){return false;}
    return bytes::resize_zeroed(out,length-2).is_ok && bytes::append(out,b",\"expectedSha256\":\"").is_ok &&
        bytes::append(out,expected).is_ok && bytes::append(out,b"\",\"matches\":").is_ok &&
        bytes::append(out,if(matches){b"true}\n"}else{b"false}\n"}).is_ok;
}

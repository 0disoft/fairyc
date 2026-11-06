module module_state_local_base64;
import std::bytes as bytes;

// Standard, padded, canonical base64. No text/JSON parser or external I/O.
// Errors: 1001 malformed, 1002 byte limit/work, 1003 allocation.
fn bool charge(ref mut usize work,usize cost){
    if(*work>1073741824 || cost>1073741824-*work){return false;};
    *work+=cost;return true;
}
fn u8 octet(usize value){
    let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";
    return (digits[(value/16)%16]<<4)|digits[value%16];
}
fn usize sextet(u8 c){
    if(c>=65 && c<=90){return (c as usize)-65;}
    if(c>=97 && c<=122){return (c as usize)-97+26;}
    if(c>=48 && c<=57){return (c as usize)-48+52;}
    if(c==43){return 62;}if(c==47){return 63;}return 64;
}

pub fn Result<bytes::Buffer,u32> encode(ref u8[] raw,ref mut usize work){
    if(!charge(work,0) || raw.len>16777216){return .err(1002);}
    let size=((raw.len+2)/3)*4;
    // Read source once, initialize output once, write output once.
    if(!charge(work,raw.len+2*size)){return .err(1002);}
    var output=bytes::empty();
    if(!bytes::resize_zeroed(ref mut output,size).is_ok){return .err(1003);}
    let alphabet=b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let out=bytes::view_mut(ref mut output);
    var usize i=0;var usize at=0;
    while(i<raw.len){
        let a=raw[i] as usize;
        let usize b=if(raw.len-i>1){raw[i+1] as usize}else{0};
        let usize c=if(raw.len-i>2){raw[i+2] as usize}else{0};
        out[at]=alphabet[a/4];
        out[at+1]=alphabet[(a%4)*16+b/16];
        out[at+2]=if(raw.len-i>1){alphabet[(b%16)*4+c/64]}else{61};
        out[at+3]=if(raw.len-i>2){alphabet[c%64]}else{61};
        i+=3;at+=4;
    }
    return .ok(output);
}

pub fn Result<bytes::Buffer,u32> decode(ref u8[] encoded,usize maximum,ref mut usize work){
    if(!charge(work,0) || maximum==0 || maximum>16777216 || encoded.len>22369624){return .err(1002);}
    if(encoded.len==0){return .ok(bytes::empty());}
    // Prepay validation plus at most four tail/pad-bit lookups. Errors retain it.
    if(!charge(work,encoded.len+4)){return .err(1002);}
    if(encoded.len%4!=0){return .err(1001);}
    var usize padding=0;
    if(encoded[encoded.len-1]==61){padding=1;if(encoded[encoded.len-2]==61){padding=2;}}
    let size=(encoded.len/4)*3-padding;
    if(size>maximum){return .err(1002);}
    var usize i=0;
    while(i<encoded.len-padding){
        if(sextet(encoded[i])==64){return .err(1001);}i+=1;
    }
    if(padding==2 && sextet(encoded[encoded.len-3])%16!=0){return .err(1001);}
    if(padding==1 && sextet(encoded[encoded.len-2])%4!=0){return .err(1001);}
    // Second source scan, zero initialization and decoded-byte writes.
    if(!charge(work,encoded.len+2*size)){return .err(1002);}
    var output=bytes::empty();
    if(!bytes::resize_zeroed(ref mut output,size).is_ok){return .err(1003);}
    let out=bytes::view_mut(ref mut output);i=0;var usize at=0;
    while(i<encoded.len){
        let a=sextet(encoded[i]);let b=sextet(encoded[i+1]);
        let c_byte=encoded[i+2];let d_byte=encoded[i+3];
        let usize c=if(c_byte!=61){sextet(c_byte)}else{0};
        let usize d=if(d_byte!=61){sextet(d_byte)}else{0};
        out[at]=octet(a*4+b/16);
        if(size-at>1){out[at+1]=octet((b%16)*16+c/4);}
        if(size-at>2){out[at+2]=octet((c%4)*64+d);}
        i+=4;at+=3;
    }
    return .ok(output);
}

module project_inflate;
import std::bytes as bytes;

// Raw RFC 1951 blocks only. Private errors: 1 malformed/truncated input,
// 2 allocation, 3 output bound. Every loop consumes input or produces bytes.
struct Bits{usize at;}
struct Huff{usize[16] counts;usize[288] symbols;}
fn Result<usize,u32> take(ref u8[] input,ref mut Bits bits,usize n){
    if(n>16 || bits.at>input.len*8 || n>input.len*8-bits.at){return .err(1);}
    var usize value=0;var usize shift=0;while(shift<n){value|=((input[bits.at/8] as usize)>>(bits.at%8)&1)<<shift;bits.at+=1;shift+=1;}return .ok(value);
}
fn Result<Huff,u32> tree(ref usize[] lengths,bool code_tree){
    var Huff h=Huff{.counts=[0;16],.symbols=[0;288]};if(lengths.len>288){return .err(1);}
    var usize i=0;var usize used=0;var usize longest=0;while(i<lengths.len){let usize n=lengths[i];if(n>15){return .err(1);}if(n!=0){h.counts[n]+=1;used+=1;if(n>longest){longest=n;}}i+=1;}
    var usize left=1;i=1;while(i<16){left*=2;if(h.counts[i]>left){return .err(1);}left-=h.counts[i];i+=1;}
    if(used==0 || (left!=0 && (code_tree || longest!=1))){return .err(1);}
    var usize[16] next=[0;16];i=1;while(i<15){next[i+1]=next[i]+h.counts[i];i+=1;}
    i=0;while(i<lengths.len){let usize n=lengths[i];if(n!=0){h.symbols[next[n]]=i;next[n]+=1;}i+=1;}return .ok(h);
}
fn Result<usize,u32> symbol(ref u8[] input,ref mut Bits bits,ref Huff h){
    var usize code=0;var usize first=0;var usize offset=0;var usize n=1;
    while(n<16){let usize bit=switch(take(input,bits,1)){case .ok(v):{v}case .err(e):{return .err(e);}};code=code*2+bit;
        if(code>=first && code-first<h.counts[n]){return .ok(h.symbols[offset+code-first]);}offset+=h.counts[n];first=(first+h.counts[n])*2;n+=1;
    }return .err(1);
}
fn Result<bool,u32> compressed(ref u8[] input,ref mut Bits bits,ref Huff literals,ref Huff distances,ref mut bytes::Buffer output,usize maximum){
    let usize[29] bases=[3,4,5,6,7,8,9,10,11,13,15,17,19,23,27,31,35,43,51,59,67,83,99,115,131,163,195,227,258];
    let usize[29] extra=[0,0,0,0,0,0,0,0,1,1,1,1,2,2,2,2,3,3,3,3,4,4,4,4,5,5,5,5,0];
    let usize[30] db=[1,2,3,4,5,7,9,13,17,25,33,49,65,97,129,193,257,385,513,769,1025,1537,2049,3073,4097,6145,8193,12289,16385,24577];
    let usize[30] de=[0,0,0,0,1,1,2,2,3,3,4,4,5,5,6,6,7,7,8,8,9,9,10,10,11,11,12,12,13,13];
    let ref u8[] octets=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";
    while(true){let usize code=switch(symbol(input,bits,literals)){case .ok(v):{v}case .err(e):{return .err(e);}};
        if(code==256){return .ok(true);}if(code<256){if(bytes::len(output)>=maximum){return .err(3);}let u8 value=(octets[code/16]<<4)|octets[code%16];if(!bytes::append_byte(output,value).is_ok){return .err(2);}continue;}
        if(code>285){return .err(1);}let usize n=switch(take(input,bits,extra[code-257])){case .ok(v):{bases[code-257]+v}case .err(e):{return .err(e);}};
        let usize d=switch(symbol(input,bits,distances)){case .ok(v):{v}case .err(e):{return .err(e);}};if(d>=30){return .err(1);}
        let usize distance=switch(take(input,bits,de[d])){case .ok(v):{db[d]+v}case .err(e):{return .err(e);}};
        if(distance>bytes::len(output)){return .err(1);}if(n>maximum-bytes::len(output)){return .err(3);}
        var usize i=0;while(i<n){let u8 value=bytes::view(output)[bytes::len(output)-distance];if(!bytes::append_byte(output,value).is_ok){return .err(2);}i+=1;}
    }return .err(1);
}
pub fn Result<bytes::Buffer,u32> decode(ref u8[] input,usize expected){
    if(input.len>16777216 || expected>8388608){return .err(3);}var bytes::Buffer output=bytes::empty();var Bits bits=Bits{.at=0};var bool final=false;
    while(!final){let usize header=switch(take(input,ref mut bits,3)){case .ok(v):{v}case .err(e):{return .err(e);}};final=(header&1)!=0;let usize kind=header>>1;
        if(kind==0){bits.at=(bits.at+7)/8*8;let usize n=switch(take(input,ref mut bits,16)){case .ok(v):{v}case .err(e):{return .err(e);}};let usize inverse=switch(take(input,ref mut bits,16)){case .ok(v):{v}case .err(e):{return .err(e);}};
            if((n^inverse)!=65535 || bits.at/8>input.len || n>input.len-bits.at/8){return .err(1);}if(n>expected-bytes::len(ref output)){return .err(3);}if(!bytes::append(ref mut output,input[bits.at/8..bits.at/8+n]).is_ok){return .err(2);}bits.at+=n*8;continue;
        }
        if(kind==3){return .err(1);}var usize[320] lengths=[0;320];var usize ln=288;var usize dn=32;
        if(kind==1){var usize i=0;while(i<288){lengths[i]=if(i<144){8}else if(i<256){9}else if(i<280){7}else{8};i+=1;}while(i<320){lengths[i]=5;i+=1;}}
        else{
            ln=switch(take(input,ref mut bits,5)){case .ok(v):{v+257}case .err(e):{return .err(e);}};dn=switch(take(input,ref mut bits,5)){case .ok(v):{v+1}case .err(e):{return .err(e);}};
            let usize cn=switch(take(input,ref mut bits,4)){case .ok(v):{v+4}case .err(e):{return .err(e);}};if(ln>286){return .err(1);}
            let usize[19] positions=[16,17,18,0,8,7,9,6,10,5,11,4,12,3,13,2,14,1,15];
            var usize[19] cl=[0;19];var usize i=0;while(i<cn){cl[positions[i]]=switch(take(input,ref mut bits,3)){case .ok(v):{v}case .err(e):{return .err(e);}};i+=1;}
            let ref usize[] cview=ref cl;let codes=switch(tree(cview,true)){case .ok(v):{v}case .err(e):{return .err(e);}};i=0;
            while(i<ln+dn){let usize code=switch(symbol(input,ref mut bits,ref codes)){case .ok(v):{v}case .err(e):{return .err(e);}};
                if(code<16){lengths[i]=code;i+=1;}else{var usize value=0;if(code==16){if(i==0){return .err(1);}value=lengths[i-1];}let usize n=switch(take(input,ref mut bits,if(code==16){2}else if(code==17){3}else{7})){case .ok(v):{v+if(code==18){11}else{3}}case .err(e):{return .err(e);}};
                    if(n>ln+dn-i){return .err(1);}var usize j=0;while(j<n){lengths[i]=value;i+=1;j+=1;}
                }
            }
        }
        if(lengths[256]==0){return .err(1);}let ref usize[] view=ref lengths;let literals=switch(tree(view[..ln],false)){case .ok(v):{v}case .err(e):{return .err(e);}};
        // An unused distance alphabet may be empty; represent it by an empty
        // tree so an actual length/distance pair still fails during decoding.
        var Huff distances=Huff{.counts=[0;16],.symbols=[0;288]};var usize nonzero=0;var usize i=ln;while(i<ln+dn){nonzero+=lengths[i];i+=1;}
        if(nonzero!=0){distances=switch(tree(view[ln..ln+dn],false)){case .ok(v):{v}case .err(e):{return .err(e);}};}
        switch(compressed(input,ref mut bits,ref literals,ref distances,ref mut output,expected)){case .ok(_):{}case .err(e):{return .err(e);}};
    }
    if(bytes::len(ref output)!=expected){return .err(1);}return .ok(output);
}

module sha256;

// Bounded SHA-256 over source bytes. Arithmetic stays checked: modular addition
// uses 16-bit limbs and rotation masks discarded bits before the left shift.
fn u32 octet(ref u8[] input,usize index,usize padded){
    if(index<input.len){return input[index] as u32;}
    if(index==input.len){return 128;}
    if(index<padded-8){return 0;}
    var usize bits=input.len*8;var usize shifts=padded-1-index;
    while(shifts>0){bits/=256;shifts-=1;}
    var usize left=bits%256;var u32 byte=0;while(left>0){byte+=1;left-=1;}return byte;
}
pub fn u32[8] digest(ref u8[] input){
    assert(input.len<=1048576);var u32[8] h=[1779033703,3144134277,1013904242,2773480762,1359893119,2600822924,528734635,1541459225];let usize padded=((input.len+72)/64)*64;var usize offset=0;
    while(offset<padded){var u32[16] words=[0;16];var usize i=0;while(i<16){let usize p=offset+i*4;words[i]=octet(input,p,padded)<<24 | octet(input,p+1,padded)<<16 | octet(input,p+2,padded)<<8 | octet(input,p+3,padded);i+=1;}h=compress(h,words);offset+=64;}return h;
}
pub fn bool matches(ref u8[] input,ref u8[] hex){
    if(input.len>1048576 || hex.len!=64){return false;}let u32[8] h=digest(input);let ref u8[] digits=b"0123456789abcdef";var usize i=0;
    while(i<8){var usize j=0;var u32 shift=28;while(j<8){if(hex[i*8+j]!=digits[((h[i]>>shift)&15) as usize]){return false;}j+=1;if(j<8){shift-=4;}}i+=1;}return true;
}


fn u32 add(u32 a,u32 b){let u32 lo=(a&65535)+(b&65535);return (((a>>16)+(b>>16)+(lo>>16))&65535)<<16 | (lo&65535);}
fn u32 rotate(u32 n,u32 bits){return n>>bits | (n&((1<<bits)-1))<<(32-bits);}
pub fn u32[8] compress(u32[8] previous,u32[16] input){
    let u32[64] k=[1116352408,1899447441,3049323471,3921009573,961987163,1508970993,2453635748,2870763221,3624381080,310598401,607225278,1426881987,1925078388,2162078206,2614888103,3248222580,3835390401,4022224774,264347078,604807628,770255983,1249150122,1555081692,1996064986,2554220882,2821834349,2952996808,3210313671,3336571891,3584528711,113926993,338241895,666307205,773529912,1294757372,1396182291,1695183700,1986661051,2177026350,2456956037,2730485921,2820302411,3259730800,3345764771,3516065817,3600352804,4094571909,275423344,430227734,506948616,659060556,883997877,958139571,1322822218,1537002063,1747873779,1955562222,2024104815,2227730452,2361852424,2428436474,2756734187,3204031479,3329325298];
    var u32[8] h=previous;var u32[64] w=[0;64];var usize i=0;while(i<16){w[i]=input[i];i+=1;}
        while(i<64){let u32 x=w[i-15];let u32 y=w[i-2];let u32 a=rotate(x,7)^rotate(x,18)^(x>>3);let u32 b=rotate(y,17)^rotate(y,19)^(y>>10);w[i]=add(add(w[i-16],a),add(w[i-7],b));i+=1;}
        var u32 a=h[0];var u32 b=h[1];var u32 c=h[2];var u32 d=h[3];var u32 e=h[4];var u32 f=h[5];var u32 g=h[6];var u32 z=h[7];i=0;
        while(i<64){let u32 s=rotate(e,6)^rotate(e,11)^rotate(e,25);let u32 ch=(e&f)^((~e)&g);let u32 t=add(add(add(z,s),ch),add(k[i],w[i]));let u32 u=add(rotate(a,2)^rotate(a,13)^rotate(a,22),(a&b)^(a&c)^(b&c));z=g;g=f;f=e;e=add(d,t);d=c;c=b;b=a;a=add(t,u);i+=1;}
        h[0]=add(h[0],a);h[1]=add(h[1],b);h[2]=add(h[2],c);h[3]=add(h[3],d);h[4]=add(h[4],e);h[5]=add(h[5],f);h[6]=add(h[6],g);h[7]=add(h[7],z);return h;
}

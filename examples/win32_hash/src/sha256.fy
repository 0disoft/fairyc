module sha256;

// SHA words are bounded 32-bit values carried in 64-bit usize intermediates.
// At most five words are added (below 2^35); rotations shift at most 31 bits.
// Every result is masked to 32 bits. Ordinary Fairy arithmetic remains checked.
fn usize octet(ref u8[] input,usize index,usize padded){
    if(index<input.len){return input[index] as usize;}
    if(index==input.len){return 128;}
    if(index<padded-8){return 0;}
    var usize bits=input.len*8;var usize shifts=padded-1-index;
    while(shifts>0){bits/=256;shifts-=1;}
    return bits%256;
}
pub fn usize[8] digest(ref u8[] input){
    assert(input.len<=1048576);var usize[8] h=[1779033703,3144134277,1013904242,2773480762,1359893119,2600822924,528734635,1541459225];let usize padded=((input.len+72)/64)*64;var usize offset=0;
    while(offset<padded){var usize[16] words=[0;16];var usize i=0;while(i<16){let usize p=offset+i*4;words[i]=octet(input,p,padded)<<24 | octet(input,p+1,padded)<<16 | octet(input,p+2,padded)<<8 | octet(input,p+3,padded);i+=1;}h=compress(h,words);offset+=64;}return h;
}
pub fn bool matches(ref u8[] input,ref u8[] hex){
    if(input.len>1048576 || hex.len!=64){return false;}let usize[8] h=digest(input);let ref u8[] digits=b"0123456789abcdef";var usize i=0;
    while(i<8){var usize j=0;var usize shift=28;while(j<8){if(hex[i*8+j]!=digits[(h[i]>>shift)&15]){return false;}j+=1;if(j<8){shift-=4;}}i+=1;}return true;
}


fn usize add(usize a,usize b){return (a+b)&4294967295;}
fn usize rotate(usize n,usize bits){return (n>>bits | n<<(32-bits))&4294967295;}
pub fn usize[8] compress(usize[8] previous,usize[16] input){
    let usize[64] k=[1116352408,1899447441,3049323471,3921009573,961987163,1508970993,2453635748,2870763221,3624381080,310598401,607225278,1426881987,1925078388,2162078206,2614888103,3248222580,3835390401,4022224774,264347078,604807628,770255983,1249150122,1555081692,1996064986,2554220882,2821834349,2952996808,3210313671,3336571891,3584528711,113926993,338241895,666307205,773529912,1294757372,1396182291,1695183700,1986661051,2177026350,2456956037,2730485921,2820302411,3259730800,3345764771,3516065817,3600352804,4094571909,275423344,430227734,506948616,659060556,883997877,958139571,1322822218,1537002063,1747873779,1955562222,2024104815,2227730452,2361852424,2428436474,2756734187,3204031479,3329325298];
    // Borrow the hot tables once: value-array reads in the authored MSVC
    // backend use opaque owner copies to preserve aliasing correctness.
    let ref usize[] constants=ref k;
    var usize[8] h=previous;var usize[64] schedule=[0;64];
    let ref mut usize[] w=ref mut schedule;
    var usize i=0;while(i<16){w[i]=input[i];i+=1;}
        while(i<64){let usize x=w[i-15];let usize y=w[i-2];let usize a=rotate(x,7)^rotate(x,18)^(x>>3);let usize b=rotate(y,17)^rotate(y,19)^(y>>10);w[i]=(w[i-16]+a+w[i-7]+b)&4294967295;i+=1;}
        var usize a=h[0];var usize b=h[1];var usize c=h[2];var usize d=h[3];var usize e=h[4];var usize f=h[5];var usize g=h[6];var usize z=h[7];i=0;
        while(i<64){let usize s=rotate(e,6)^rotate(e,11)^rotate(e,25);let usize ch=(e&f)^((~e)&g);let usize t=(z+s+ch+constants[i]+w[i])&4294967295;let usize u=add(rotate(a,2)^rotate(a,13)^rotate(a,22),(a&b)^(a&c)^(b&c));z=g;g=f;f=e;e=add(d,t);d=c;c=b;b=a;a=add(t,u);i+=1;}
        h[0]=add(h[0],a);h[1]=add(h[1],b);h[2]=add(h[2],c);h[3]=add(h[3],d);h[4]=add(h[4],e);h[5]=add(h[5],f);h[6]=add(h[6],g);h[7]=add(h[7],z);return h;
}

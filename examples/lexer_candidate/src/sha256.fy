module sha256;
import sha256_block as blocks;

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
    while(offset<padded){var u32[16] words=[0;16];var usize i=0;while(i<16){let usize p=offset+i*4;words[i]=octet(input,p,padded)<<24 | octet(input,p+1,padded)<<16 | octet(input,p+2,padded)<<8 | octet(input,p+3,padded);i+=1;}h=blocks::compress(h,words);offset+=64;}return h;
}
pub fn bool matches(ref u8[] input,ref u8[] hex){
    if(input.len>1048576 || hex.len!=64){return false;}let u32[8] h=digest(input);let ref u8[] digits=b"0123456789abcdef";var usize i=0;
    while(i<8){var usize j=0;var u32 shift=28;while(j<8){if(hex[i*8+j]!=digits[((h[i]>>shift)&15) as usize]){return false;}j+=1;if(j<8){shift-=4;}}i+=1;}return true;
}

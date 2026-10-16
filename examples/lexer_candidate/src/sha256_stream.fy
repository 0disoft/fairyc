module sha256_stream;
import sha256 as blocks;
import std::bytes as bytes;

// This private state keeps one partial block, independent of total input size.
// Canonical source encoding stays below the existing 96 MiB transport ceiling.
pub struct State{u32[8] words;u8[64] block;usize used;usize length;}
pub fn State empty(){return State{.words=[1779033703,3144134277,1013904242,2773480762,1359893119,2600822924,528734635,1541459225],.block=[0;64],.used=0,.length=0};}
pub fn bool valid(ref State state){return state.length<=100663296 && state.used<64 && state.used==state.length%64;}
fn u32[16] words(ref u8[] block){assert(block.len==64);var u32[16] output=[0;16];var usize index=0;while(index<16){let usize at=index*4;output[index]=(block[at] as u32)<<24 | (block[at+1] as u32)<<16 | (block[at+2] as u32)<<8 | (block[at+3] as u32);index+=1;}return output;}
fn void flush(ref mut State state){let ref u8[] block=ref state.block;let u32[16] input=words(block);state.words=blocks::compress(state.words,input);state.used=0;}
fn void put(ref mut State state,u8 value){let usize at=state.used;state.block[at]=value;state.used+=1;}
pub fn bool update(ref mut State state,ref u8[] input){
    if(!valid(state) || input.len>100663296-state.length){return false;}var usize at=0;
    if(state.used>0){while(at<input.len && state.used<64){put(state,input[at]);at+=1;}if(state.used==64){flush(state);}}
    while(input.len-at>=64){state.words=blocks::compress(state.words,words(input[at..at+64]));at+=64;}
    while(at<input.len){put(state,input[at]);at+=1;}state.length+=input.len;return true;
}
fn u8 octet(usize value){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
// Finalization copies the small state; callers can keep updating the original.
pub fn u32[8] finish(ref State input){
    assert(valid(input));var State state=*input;put(ref mut state,128);
    if(state.used>56){while(state.used<64){put(ref mut state,0);}flush(ref mut state);}
    while(state.used<56){put(ref mut state,0);}var usize bits=input.length*8;var usize index=0;while(index<8){state.block[63-index]=octet(bits%256);bits/=256;index+=1;}let ref u8[] block=ref state.block;return blocks::compress(state.words,words(block));
}
pub fn bool matches(ref State state,ref u8[] hex){if(!valid(state) || hex.len!=64){return false;}let u32[8] hash=finish(state);let ref u8[] digits=b"0123456789abcdef";var usize index=0;var u32 shift=28;while(index<64){let u32 word=hash[index/8];if(digits[((word>>shift)&15) as usize]!=hex[index]){return false;}index+=1;if(index%8==0){shift=28;}else{shift-=4;}}return true;}
pub fn bool append_hex(ref State state,ref mut bytes::Buffer output){if(!valid(state)){return false;}let u32[8] hash=finish(state);let ref u8[] digits=b"0123456789abcdef";var usize index=0;var u32 shift=28;while(index<64){let u32 word=hash[index/8];if(!bytes::append_byte(output,digits[((word>>shift)&15) as usize]).is_ok){return false;}index+=1;if(index%8==0){shift=28;}else{shift-=4;}}return true;}

module project_json_decode;
import project_path as work;
import utf8 as utf8;
import std::bytes as bytes;

pub struct Problem{u32 code;usize start;usize end;}
fn Problem fail(u32 code,usize start,usize end){return Problem{.code,.start,.end};}
fn u8 octet(u32 value){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[((value/16)%16) as usize]<<4)|digits[(value%16) as usize];}
fn u32 append(ref mut bytes::Buffer output,ref u8[] input,ref mut usize meter){if(bytes::len(output)>8388608 || input.len>8388608-bytes::len(output) || !work::charge(meter,input.len+1)){return 3;}return if(bytes::append(output,input).is_ok){0}else{2};}
fn u32 point(ref mut bytes::Buffer output,u32 value,ref mut usize meter){var u8[4] data=[0;4];var usize count=1;if(value<128){data[0]=octet(value);}else if(value<2048){count=2;data[0]=octet(192+value/64);data[1]=octet(128+value%64);}else if(value<65536){count=3;data[0]=octet(224+value/4096);data[1]=octet(128+(value/64)%64);data[2]=octet(128+value%64);}else{count=4;data[0]=octet(240+value/262144);data[1]=octet(128+(value/4096)%64);data[2]=octet(128+(value/64)%64);data[3]=octet(128+value%64);}let ref u8[] input=ref data;return append(output,input[..count],meter);}
fn u32 hex(u8 value){if(value>=48 && value<=57){return (value as u32)-48;}if(value>=65 && value<=70){return (value as u32)-55;}if(value>=97 && value<=102){return (value as u32)-87;}return 16;}
fn Result<u32,Problem> four(ref u8[] input,usize at){if(at>input.len || input.len-at<4){return .err(fail(1,at,input.len));}var u32 value=0;var usize index=0;while(index<4){let u32 part=hex(input[at+index]);if(part==16){return .err(fail(1,at+index,at+index+1));}value=value*16+part;index+=1;}return .ok(value);}
fn Result<usize,Problem> decode(ref u8[] input,usize start,ref mut bytes::Buffer output,ref mut usize meter){if(start>=input.len || input[start]!=34){return .err(fail(1,start,start));}var usize at=start+1;var usize run=at;
    while(at<input.len){if(!work::charge(meter,1)){return .err(fail(3,at,at));}let u8 byte=input[at];if(byte==34){let u32 code=append(output,input[run..at],meter);if(code!=0){return .err(fail(code,start,at+1));}return .ok(at+1);}if(byte<32){return .err(fail(1,at,at+1));}
        if(byte==92 || byte>=128 && utf8::width(input,at)==0){let u32 copied=append(output,input[run..at],meter);if(copied!=0){return .err(fail(copied,start,at));}if(byte!=92){let u32 code=point(output,65533,meter);if(code!=0){return .err(fail(code,at,at+1));}at+=1;run=at;continue;}let usize escape=at;at+=1;if(at==input.len){return .err(fail(1,escape,at));}let u8 kind=input[at];at+=1;var u32 scalar=0;
            if(kind==117){scalar=switch(four(input,at)){case .ok(value):{value}case .err(p):{return .err(p);}};at+=4;if(scalar>=55296 && scalar<=56319){var bool paired=false;if(input.len-at>=6 && input[at]==92 && input[at+1]==117){let u32 low=switch(four(input,at+2)){case .ok(value):{value}case .err(_):{0}};if(low>=56320 && low<=57343){scalar=65536+(scalar-55296)*1024+low-56320;at+=6;paired=true;}}if(!paired){scalar=65533;}}else if(scalar>=56320 && scalar<=57343){scalar=65533;}}
            else if(kind==34 || kind==92 || kind==47){scalar=kind as u32;}else if(kind==98){scalar=8;}else if(kind==102){scalar=12;}else if(kind==110){scalar=10;}else if(kind==114){scalar=13;}else if(kind==116){scalar=9;}else{return .err(fail(1,escape,at));}let u32 code=point(output,scalar,meter);if(code!=0){return .err(fail(code,escape,at));}run=at;continue;
        }let usize width=utf8::width(input,at);at+=if(width==0){1}else{width};
    }return .err(fail(1,start,input.len));
}
// Go JSON decoding replaces invalid raw UTF-8 and unpaired UTF-16 surrogates
// with U+FFFD. Syntax errors and bounds restore the caller's prior decoded pool.
// Codes: 1 syntax, 2 allocation, 3 input/output/work bound.
pub fn Result<usize,Problem> string(ref u8[] input,usize start,ref mut bytes::Buffer output,ref mut usize meter){if(input.len>8388608 || start>input.len || bytes::len(output)>8388608){return .err(fail(3,if(start<=input.len){start}else{input.len},input.len));}let usize before=bytes::len(output);return switch(decode(input,start,output,meter)){case .ok(end):{.ok(end)}case .err(p):{discard bytes::truncate(output,before);.err(p)}};}

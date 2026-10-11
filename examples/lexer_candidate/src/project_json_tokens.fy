module project_json_tokens;
import project_json_decode as decode;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// 1 {, 2 }, 3 [, 4 ], 5 :, 6 comma, 7 string, 8 number,
// 9 true, 10 false, 11 null. Container grammar belongs to the reader.
pub struct Token{u32 kind;usize start;usize end;usize value_start;usize value_end;}
pub resource Tokens{bytes::Buffer source;bytes::Buffer decoded;v::Vec<Token> tokens;cleanup{}}
pub fn usize count(ref Tokens tokens){return v::len<Token>(ref tokens.tokens);}
pub fn Token at(ref Tokens tokens,usize index){assert(index<count(tokens));return *v::at<Token>(ref tokens.tokens,index);}
pub fn ref u8[] source(ref Tokens tokens){return bytes::view(ref tokens.source);}
pub fn ref u8[] value(ref Tokens tokens,usize index){let t=at(tokens,index);if(t.kind==7){return bytes::view(ref tokens.decoded)[t.value_start..t.value_end];}if(t.kind==8){return source(tokens)[t.start..t.end];}return bytes::view(ref tokens.decoded)[..0];}
fn decode::Problem fail(u32 code,usize start,usize end){return decode::Problem{.code,.start,.end};}
fn bool space(u8 byte){return byte==32 || byte==9 || byte==10 || byte==13;}
fn bool digit(u8 byte){return byte>=48 && byte<=57;}
fn bool end(ref u8[] input,usize at){if(at==input.len){return true;}let u8 byte=input[at];return space(byte) || byte==123 || byte==125 || byte==91 || byte==93 || byte==58 || byte==44;}
fn Result<usize,decode::Problem> number(ref u8[] input,usize start,ref mut usize meter){var usize at=start;if(input[at]==45){at+=1;}if(at==input.len || !digit(input[at])){return .err(fail(1,start,at));}if(input[at]==48){at+=1;}else{while(at<input.len && digit(input[at])){if(!work::charge(meter,1)){return .err(fail(3,at,at));}at+=1;}}
    if(at<input.len && input[at]==46){at+=1;let usize begin=at;while(at<input.len && digit(input[at])){if(!work::charge(meter,1)){return .err(fail(3,at,at));}at+=1;}if(at==begin){return .err(fail(1,start,at));}}
    if(at<input.len && (input[at]==101 || input[at]==69)){at+=1;if(at<input.len && (input[at]==43 || input[at]==45)){at+=1;}let usize begin=at;while(at<input.len && digit(input[at])){if(!work::charge(meter,1)){return .err(fail(3,at,at));}at+=1;}if(at==begin){return .err(fail(1,start,at));}}if(!end(input,at)){return .err(fail(1,at,at+1));}return .ok(at);
}
pub fn Result<Tokens,decode::Problem> scan(bytes::Buffer source,ref mut usize meter){let ref u8[] input=bytes::view(ref source);if(input.len>8388608 || !work::charge(meter,input.len+1)){return .err(fail(3,0,0));}var bytes::Buffer decoded=bytes::empty();var v::Vec<Token> tokens=v::empty<Token>();var usize at=0;
    while(at<input.len){if(!work::charge(meter,1)){return .err(fail(3,at,at));}if(space(input[at])){at+=1;continue;}let usize start=at;let usize value_start=bytes::len(ref decoded);let u8 byte=input[at];var u32 kind=if(byte==123){1}else if(byte==125){2}else if(byte==91){3}else if(byte==93){4}else if(byte==58){5}else if(byte==44){6}else{0};if(kind!=0){at+=1;}else if(byte==34){kind=7;at=switch(decode::string(input,at,ref mut decoded,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};}else if(byte==45 || digit(byte)){kind=8;at=switch(number(input,at,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};}else{let usize size=if(byte==116 || byte==110){4}else if(byte==102){5}else{0};if(size==0 || input.len-at<size){return .err(fail(1,at,input.len));}let ref u8[] literal=input[at..at+size];if(bytes::equal(literal,b"true")){kind=9;}else if(bytes::equal(literal,b"false")){kind=10;}else if(bytes::equal(literal,b"null")){kind=11;}else{return .err(fail(1,at,at+size));}at+=size;if(!end(input,at)){return .err(fail(1,at,at+1));}}
        if(v::len<Token>(ref tokens)>=1048576){return .err(fail(3,start,at));}if(!v::push<Token>(ref mut tokens,Token{.kind,.start,.end=at,.value_start,.value_end=bytes::len(ref decoded)}).is_ok){return .err(fail(2,start,at));}
    }return .ok(Tokens{.source,.decoded,.tokens});
}

module project_json_string;
import project_path as work;
import std::bytes as bytes;
import std::text as text;

// Canonical Go encoding/json string spelling, including HTML and JS separators.
// 1 output/work bound, 2 invalid UTF-8, 3 allocation. No partial result escapes.
fn u32 append(ref mut bytes::Buffer output,ref u8[] input,ref mut usize meter){if(bytes::len(output)>8388608 || input.len>8388608-bytes::len(output) || !work::charge(meter,input.len+1)){return 1;}return if(bytes::append(output,input).is_ok){0}else{3};}
fn u32 escaped(ref mut bytes::Buffer output,u8 value,ref mut usize meter){if(value==34){return append(output,b"\\\"",meter);}if(value==92){return append(output,b"\\\\",meter);}if(value==8){return append(output,b"\\b",meter);}if(value==12){return append(output,b"\\f",meter);}if(value==10){return append(output,b"\\n",meter);}if(value==13){return append(output,b"\\r",meter);}if(value==9){return append(output,b"\\t",meter);}let ref u8[] hex=b"0123456789abcdef";var u8[6] encoded=[92,117,48,48,hex[(value as usize)/16],hex[(value as usize)%16]];let ref u8[] input=ref encoded;return append(output,input,meter);}
pub fn Result<bytes::Buffer,u32> quote(ref u8[] input,ref mut usize meter){if(input.len>8388608 || !work::charge(meter,input.len+1)){return .err(1);}if(!text::validate_utf8(input).is_ok){return .err(2);}var bytes::Buffer output=bytes::empty();var u32 code=append(ref mut output,b"\"",meter);if(code!=0){return .err(code);}var usize at=0;var usize start=0;
    while(at<input.len){if(!work::charge(meter,1)){return .err(1);}let u8 value=input[at];let bool separator=value==226 && input.len-at>=3 && input[at+1]==128 && (input[at+2]==168 || input[at+2]==169);if(value<32 || value==34 || value==92 || value==60 || value==62 || value==38 || separator){code=append(ref mut output,input[start..at],meter);if(code!=0){return .err(code);}if(separator){code=append(ref mut output,if(input[at+2]==168){b"\\u2028"}else{b"\\u2029"},meter);at+=3;}else{code=escaped(ref mut output,value,meter);at+=1;}if(code!=0){return .err(code);}start=at;}else{at+=1;}
    }code=append(ref mut output,input[start..],meter);if(code!=0){return .err(code);}code=append(ref mut output,b"\"",meter);if(code!=0){return .err(code);}return .ok(output);
}

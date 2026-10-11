module tests::project_json_tokens;
import project_json_decode as decode;
import project_json_tokens as tokens;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;return switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] input,ref mut usize at){assert(*at<=input.len && input.len-*at>=8);var usize value=0;var usize index=0;while(index<8){value=value*256+(input[*at+index] as usize);index+=1;}*at+=8;return value;}
fn ref u8[] field(ref u8[] input,ref mut usize at){let usize size=number(input,at);assert(*at<=input.len && size<=input.len-*at);let usize start=*at;*at+=size;return input[start..*at];}
fn bytes::Buffer copy(ref u8[] input){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,input).is_ok);return result;}
@test
@effects(filesystem)
fn void string_decoding_matches_go_and_rolls_back_failures(){let owned=fixture();let input=bytes::view(ref owned);assert(input.len>=16 && bytes::equal(input[..8],b"FYJDEC1\n"));var usize at=8;let usize count=number(input,ref mut at);assert(count==24);var usize index=0;while(index<count){let source=field(input,ref mut at);let bool valid=number(input,ref mut at)==1;let expected=field(input,ref mut at);var bytes::Buffer output=copy(b"prefix:");var usize meter=0;switch(decode::string(source,0,ref mut output,ref mut meter)){case .ok(end):{assert(valid && end==source.len && bytes::equal(bytes::view(ref output)[7..],expected));}case .err(p):{assert(!valid && p.code==1 && p.start<=p.end && p.end<=source.len && bytes::equal(bytes::view(ref output),b"prefix:"));}};index+=1;}assert(at==input.len);}
fn tokens::Tokens retained(){var usize meter=0;return switch(tokens::scan(copy(b" \r\n{\"a\\u0062\":[-0,12.50e+2,true,false,null]}\t"),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return retained();}};}
@test
fn void tokens_keep_owned_values_raw_numbers_and_source_spans(){let value=retained();assert(tokens::count(ref value)==15);let source=tokens::source(ref value);assert(source.len==45);assert(tokens::at(ref value,0).kind==1 && tokens::at(ref value,0).start==3);assert(tokens::at(ref value,1).kind==7 && bytes::equal(tokens::value(ref value,1),b"ab"));assert(tokens::at(ref value,4).kind==8 && bytes::equal(tokens::value(ref value,4),b"-0"));assert(tokens::at(ref value,6).kind==8 && bytes::equal(tokens::value(ref value,6),b"12.50e+2"));assert(tokens::at(ref value,8).kind==9 && tokens::at(ref value,10).kind==10 && tokens::at(ref value,12).kind==11);var usize index=0;var usize end=0;while(index<tokens::count(ref value)){let t=tokens::at(ref value,index);assert(t.start>=end && t.start<t.end && t.end<=source.len);end=t.end;index+=1;}assert(tokens::at(ref value,13).kind==4 && tokens::at(ref value,14).kind==2);}
fn void bad(ref u8[] input){var usize meter=0;assert(switch(tokens::scan(copy(input),ref mut meter)){case .ok(_):{false}case .err(p):{p.code==1 && p.start<=p.end && p.end<=input.len}});}
@test
fn void malformed_lexemes_and_work_limits_fail_transactionally(){bad(b"01");bad(b"-");bad(b"1.");bad(b"1e");bad(b"1e+");bad(b"+1");bad(b".1");bad(b"NaN");bad(b"trueX");bad(b"falsee");bad(b"nul");bad(b"\xff");bad(b"\"a\n\"");bad(b"\"\\x\"");var bytes::Buffer output=copy(b"prefix:");var usize meter=16777214;assert(switch(decode::string(b"\"\\nmore\"",0,ref mut output,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==3}});assert(bytes::equal(bytes::view(ref output),b"prefix:"));meter=16777216;assert(switch(tokens::scan(copy(b"[]"),ref mut meter)){case .ok(_):{false}case .err(p):{p.code==3}});var bytes::Buffer huge=bytes::empty();assert(bytes::resize_zeroed(ref mut huge,8388609).is_ok);meter=0;assert(switch(tokens::scan(huge,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==3}});assert(meter==0);}

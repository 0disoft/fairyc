module tests::project_json_tree;
import project_json_tree as json;
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
fn void json_grammar_and_decoded_duplicate_keys_match_go(){let owned=fixture();let input=bytes::view(ref owned);assert(input.len>=16 && bytes::equal(input[..8],b"FYJTREE1"));var usize at=8;let usize count=number(input,ref mut at);assert(count==45);var usize index=0;while(index<count){let source=field(input,ref mut at);let usize code=number(input,ref mut at);var usize meter=0;switch(json::parse(copy(source),ref mut meter)){case .ok(tree):{assert(code==0 && json::next(ref tree,0)==json::count(ref tree) && bytes::equal(json::source(ref tree),source));}case .err(p):{assert((p.code as usize)==code && p.start<=p.end && p.end<=source.len);}};index+=1;}assert(at==input.len);}
fn json::Tree retained(){var usize meter=0;return switch(json::parse(copy(b"{\"a\":[{\"x\":\"\\u0061\"},null],\"b\":true}"),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return retained();}};}
@test
fn void container_skip_indices_and_decoded_values_survive_source_close(){let tree=retained();assert(json::count(ref tree)==17 && json::next(ref tree,0)==17);assert(json::next(ref tree,3)==12 && json::next(ref tree,4)==9);assert(json::token(ref tree,7).kind==7 && bytes::equal(json::value(ref tree,7),b"a") && json::next(ref tree,7)==8);assert(json::token(ref tree,13).kind==7 && bytes::equal(json::value(ref tree,13),b"b"));}
fn bytes::Buffer nested(usize depth){var bytes::Buffer result=bytes::empty();var usize at=0;while(at<depth){assert(bytes::append_byte(ref mut result,91).is_ok);at+=1;}assert(bytes::append_byte(ref mut result,48).is_ok);at=0;while(at<depth){assert(bytes::append_byte(ref mut result,93).is_ok);at+=1;}return result;}
@test
fn void explicit_stack_depth_and_work_bounds_reject_without_recursion(){var usize meter=0;let tree=switch(json::parse(nested(512),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(json::count(ref tree)==1025 && json::next(ref tree,0)==1025);meter=0;assert(switch(json::parse(nested(513),ref mut meter)){case .ok(_):{false}case .err(p):{p.code==3 && p.start==512 && p.end==513}});meter=16777216;assert(switch(json::parse(copy(b"{}"),ref mut meter)){case .ok(_):{false}case .err(p):{p.code==3}});}

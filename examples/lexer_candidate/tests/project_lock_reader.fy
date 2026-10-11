module tests::project_lock_reader;
import project_lock_reader as reader;
import project_lock_document as document;
import project_lock_json as json;
import project_lock_fields as fields;
import project_lock_validate as validate;
import project_source_file as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;var usize at=0;var usize separator=0;while(at<length){if(path[at]==47 || path[at]==92){separator=at;}at+=1;}assert(separator>0 && separator+1<length);var usize meter=0;return switch(files::read(path[..separator],path[separator+1..length],8388608,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] input,ref mut usize at){assert(*at<=input.len && input.len-*at>=8);var usize value=0;var usize index=0;while(index<8){value=value*256+(input[*at+index] as usize);index+=1;}*at+=8;return value;}
fn ref u8[] field(ref u8[] input,ref mut usize at){let usize size=number(input,at);assert(*at<=input.len && size<=input.len-*at);let usize start=*at;*at+=size;return input[start..*at];}
fn bytes::Buffer copy(ref u8[] input){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,input).is_ok);return result;}
@test
@effects(filesystem,console)
fn void strict_lock_semantics_and_canonical_roundtrip_match_go(){let owned=fixture();let input=bytes::view(ref owned);assert(input.len>=16 && bytes::equal(input[..8],b"FYLVALID"));var usize at=8;let usize count=number(input,ref mut at);assert(count==95);var usize index=0;var usize accepted=0;while(index<count){let name=field(input,ref mut at);let source=field(input,ref mut at);let bool valid=number(input,ref mut at)==1;let canonical=field(input,ref mut at);var usize meter=0;
        switch(reader::parse(copy(source),ref mut meter)){case .ok(value):{if(!valid){discard io::write_stderr_bytes(name);discard io::write_stderr(" unexpected acceptance\n");assert(false);}let actual=switch(json::marshal(ref value,ref mut meter)){case .ok(value):{value}case .err(_):{discard io::write_stderr_bytes(name);assert(false);return;}};if(!bytes::equal(bytes::view(ref actual),canonical)){discard io::write_stderr_bytes(name);discard io::write_stderr(" canonical mismatch\n");assert(false);}accepted+=1;}case .err(p):{if(valid){discard io::write_stderr_bytes(name);discard io::write_stderr(" unexpected rejection\n");discard io::write_stderr_u32(p.stage);discard io::write_stderr_u32(p.code);discard io::write_stderr_usize(p.node);discard io::write_stderr_usize(p.edge);assert(false);}}};index+=1;
    }assert(at==input.len && accepted==29);}
@test
@effects(filesystem)
fn void semantic_validation_and_retained_projection_enforce_work_bounds(){let owned=fixture();let input=bytes::view(ref owned);var usize at=8;discard number(input,ref mut at);discard field(input,ref mut at);let source=field(input,ref mut at);assert(number(input,ref mut at)==1);let canonical=field(input,ref mut at);var usize meter=0;
    let draft=switch(fields::parse(copy(source),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};meter=16777216;assert(switch(validate::validate(draft,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==1}});
    meter=0;let next=switch(fields::parse(copy(source),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let checked=switch(validate::validate(next,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};meter=16777216;assert(switch(document::from_validated(ref checked,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==1}});
    meter=0;let retained=switch(reader::parse(copy(source),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let actual=switch(json::marshal(ref retained,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref actual),canonical));meter=16777216;assert(switch(reader::parse(copy(source),ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==1 && p.code==3}});
}

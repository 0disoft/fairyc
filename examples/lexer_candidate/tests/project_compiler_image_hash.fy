module tests::project_compiler_image_hash;
import project_compiler_image_hash as image;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,leaf).is_ok);return out;}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref data;return switch(files::manifest(path[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] raw,ref mut usize at){assert(*at<=raw.len && raw.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] raw,ref mut usize at){let n=number(raw,at);assert(*at<=raw.len && n<=raw.len-*at);let start=*at;*at+=n;return raw[start..*at];}
@effects(filesystem)
fn image::Identity hash(ref u8[] path,ref mut usize work){return switch(image::read_path(path,work)){case .ok(p):{p}case .err(_):{assert(false);return hash(path,work);}};}
@test @effects(filesystem)
fn void matches_go_file_identities_across_padding_chunks_and_unicode_paths(){let root=root();let path=join(bytes::view(ref root),b"oracle.bin");let data=switch(files::manifest(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};let raw=bytes::view(ref data);assert(bytes::equal(raw[..8],b"FYCIMAG1"));var usize at=8;let cases=number(raw,ref mut at);assert(cases==11);var usize i=0;
    while(i<cases){let leaf=field(raw,ref mut at);let size=number(raw,ref mut at);let expected=field(raw,ref mut at);let path=join(bytes::view(ref root),leaf);var usize work=0;let actual=hash(bytes::view(ref path),ref mut work);assert(image::size(ref actual)==size && bytes::equal(image::value(ref actual),expected) && work>0 && work<=16777216);i+=1;}assert(at==raw.len);
}
@test @effects(filesystem)
fn void retains_closed_file_identity_and_rejects_exhausted_or_invalid_work(){let root=root();let path=join(bytes::view(ref root),b"size-65539.bin");var usize cost=0;let measured=hash(bytes::view(ref path),ref mut cost);assert(image::size(ref measured)==65539 && cost>0 && cost<16777216);var usize work=16777216-cost;let exact=hash(bytes::view(ref path),ref mut work);assert(work==16777216 && bytes::equal(image::value(ref measured),image::value(ref exact)));work=16777216-cost+1;
    assert(switch(image::read_path(bytes::view(ref path),ref mut work)){case .ok(_):{false}case .err(e):{e.stage==5}});assert(work<=16777216);work=16777217;assert(switch(image::read_path(bytes::view(ref path),ref mut work)){case .ok(_):{false}case .err(e):{e.stage==5 && e.completed==0}});assert(work==16777217);
}
@test @effects(filesystem)
fn void rejects_missing_empty_oversized_and_malformed_paths_without_a_digest(){let root=root();var usize i=0;while(i<3){let path=join(bytes::view(ref root),if(i==0){b"missing.bin"}else if(i==1){b"empty.bin"}else{b"oversized.bin"});var usize work=0;assert(switch(image::read_path(bytes::view(ref path),ref mut work)){case .ok(_):{false}case .err(e):{e.stage==if(i==0){1}else{4} && e.completed==0}});i+=1;}var usize work=0;assert(switch(image::read_path(b"\xff",ref mut work)){case .ok(_):{false}case .err(e):{e.stage==1}});work=0;assert(switch(image::read_path(b"",ref mut work)){case .ok(_):{false}case .err(e):{e.stage==1}});work=0;assert(switch(image::read_path(b"a\x00b",ref mut work)){case .ok(_):{false}case .err(e):{e.stage==1}});
}

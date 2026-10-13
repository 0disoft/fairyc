module tests::project_cache_profile;
import project_cache_profile as profile;
import file_source as files;
import std::process as process;
import std::bytes as bytes;

@effects(filesystem)
pub fn bytes::Buffer fixture(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref data;return switch(files::manifest(path[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
pub fn usize number(ref u8[] raw,ref mut usize at){assert(*at<=raw.len && raw.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[*at+i] as usize);i+=1;}*at+=8;return n;}
pub fn ref u8[] field(ref u8[] raw,ref mut usize at){let n=number(raw,at);assert(*at<=raw.len && n<=raw.len-*at);let start=*at;*at+=n;return raw[start..*at];}
fn bool matches(usize i,ref u8[] expected){return if(i==0){bytes::equal(profile::language(),expected)}else if(i==1){bytes::equal(profile::mir_version(),expected)}else if(i==2){bytes::equal(profile::target_schema(),expected)}else if(i==3){bytes::equal(profile::abi_profile(),expected)}else if(i==4){bytes::equal(profile::toolchain(),expected)}else if(i==5){bytes::equal(profile::backend(),expected)}else if(i==6){bytes::equal(profile::runtime_abi(),expected)}else{bytes::equal(profile::target(),expected)};}
@test @effects(filesystem)
fn void matches_actual_go_sdk_and_target_with_separate_owned_ir_identity(){let fixture=fixture();let raw=bytes::view(ref fixture);assert(bytes::equal(raw[..8],b"FYCPROF1"));var usize at=8;assert(number(raw,ref mut at)==8);var usize i=0;while(i<8){assert(matches(i,field(raw,ref mut at)));i+=1;}var usize work=0;let p=switch(profile::current(ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(profile::standard_library(ref p),field(raw,ref mut at)));assert(at==raw.len && work>0 && work<=16777216);assert(!bytes::equal(profile::mir_version(),b"fairyc-mir/2") && !bytes::equal(profile::backend(),b"verified-mir"));}
@test
fn void owns_authenticated_default_sdk_after_bundle_close_and_honors_work(){var usize cost=0;let p=switch(profile::current(ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(cost>0 && cost<16777216);var usize work=16777216-cost;let exact=switch(profile::current(ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(work==16777216 && bytes::equal(profile::standard_library(ref p),profile::standard_library(ref exact)));work=16777216-cost+1;assert(switch(profile::current(ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=16777216);work=16777217;assert(switch(profile::current(ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work==16777217);}

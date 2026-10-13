module tests::project_cache_environment;
import project_cache_environment as environment;
import project_cache_key as key;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::io as io;

@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref data;return switch(files::manifest(path[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] raw,ref mut usize at){assert(*at<=raw.len && raw.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] raw,ref mut usize at){let n=number(raw,at);assert(*at<=raw.len && n<=raw.len-*at);let start=*at;*at+=n;return raw[start..*at];}
@effects(filesystem,environment,console)
fn environment::Environment current(ref mut usize work){return switch(environment::current(work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.phase);discard io::write_stderr_u32(e.code);assert(false);return current(work);}};}
fn key::Environment copied(ref environment::Environment p,ref mut usize work){return switch(environment::key_environment(p,work)){case .ok(p):{p}case .err(_):{assert(false);return copied(p,work);}};}
@test @effects(filesystem,environment,console)
fn void derives_current_image_and_go_profile_then_detaches_key_environment(){var usize cost=0;let p=current(ref mut cost);assert(cost>0 && cost<16777216);let fixture=fixture();let raw=bytes::view(ref fixture);assert(bytes::equal(raw[..8],b"FYCPROF1"));var usize at=8;assert(number(raw,ref mut at)==8);var usize copy_work=0;let values=copied(ref p,ref mut copy_work);var usize i=1;while(i<8){assert(bytes::equal(key::environment_field(ref values,i),field(raw,ref mut at)));i+=1;}discard field(raw,ref mut at);assert(bytes::equal(environment::standard_library(ref p),field(raw,ref mut at)) && at==raw.len && bytes::equal(environment::compiler_build(ref p),key::environment_field(ref values,0)));
    var usize work=16777216-copy_work;let exact=copied(ref p,ref mut work);assert(work==16777216);i=0;while(i<8){assert(bytes::equal(key::environment_field(ref values,i),key::environment_field(ref exact,i)));i+=1;}work=16777216-copy_work+1;assert(switch(environment::key_environment(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});work=16777217;assert(switch(environment::current(ref mut work)){case .ok(_):{false}case .err(e):{e.phase==1 && e.code==10}});assert(work==16777217);
    let detached={var usize image_work=0;let p=current(ref mut image_work);var usize clone_work=0;copied(ref p,ref mut clone_work)};
    assert(bytes::equal(key::environment_field(ref detached,0),key::environment_field(ref values,0)));assert(io::write_bytes(key::environment_field(ref detached,0)).is_ok && io::write("\n").is_ok);
}

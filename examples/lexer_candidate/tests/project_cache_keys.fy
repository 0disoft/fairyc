module tests::project_cache_keys;
import project_cache_keys as keys;
import project_cache_environment as environment;
import project_compiler_core as core;
import project_compiler_input as input;
import project_coordinate_graph as coordinates;
import project_module_visibility as visibility;
import project_source_set as sources;
import project_cache_key as ordinary;
import project_generic_cache_key as generic;
import project_cache_encoding as wire;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,leaf).is_ok);return out;}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref data;return switch(files::manifest(path[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem,console)
fn visibility::Resolved load(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(visibility::load_phase(ref e));assert(false);return load(root,leaf);}};}
@effects(console)
fn core::Complete complete(ref visibility::Resolved loaded){var usize work=0;let table=switch(coordinates::collect(visibility::workspace_graph(loaded),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return complete(loaded);}};let projected=switch(input::project(loaded,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return complete(loaded);}};var usize source_work=0;var usize link_work=0;
    return switch(core::build_complete(ref projected,ref table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.phase);discard io::write_stderr_u32(e.stage);discard io::write_stderr_u32(e.code);assert(false);return complete(loaded);}};
}
@effects(filesystem,environment,console)
fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.phase);discard io::write_stderr_u32(e.code);assert(false);return current();}};}
@effects(console)
fn keys::Keys collect(ref visibility::Resolved loaded,ref core::Complete checked,ref environment::Environment env,ref mut usize work){return switch(keys::collect(loaded,checked,env,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr("keys code ");discard io::write_stderr_u32(code);assert(false);return collect(loaded,checked,env,work);}};}
fn usize number(ref u8[] raw,ref mut usize at){assert(*at<=raw.len && raw.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] raw,ref mut usize at){let n=number(raw,at);assert(*at<=raw.len && n<=raw.len-*at);let start=*at;*at+=n;return raw[start..*at];}
fn usize find(ref keys::Keys p,ref u8[] id){var usize i=0;while(i<keys::count(p)){if(bytes::equal(keys::instance(keys::at(p,i)),id)){return i;}i+=1;}return keys::count(p);}
fn void template(ref u8[] actual,ref u8[] expected,ref u8[] compiler){assert(actual.len==expected.len);let prefix=b"fairyc.compiler-executable-sha256/v1:";var usize start=expected.len;var usize i=0;while(i+prefix.len<=expected.len){if(bytes::equal(expected[i..i+prefix.len],prefix)){assert(start==expected.len);start=i;}i+=1;}assert(start<expected.len && compiler.len==prefix.len+64 && compiler.len<=expected.len-start);i=0;
    while(i<expected.len){let value=if(i>=start && i-start<compiler.len){compiler[i-start]}else{expected[i]};assert(actual[i]==value);i+=1;}
}
fn void append(ref mut bytes::Buffer out,ref u8[] value,ref mut usize work){assert(wire::string(out,value,work)==0);}
@effects(filesystem,console)
fn void oracle(ref u8[] root,ref u8[] leaf,ref keys::Keys actual,ref u8[] compiler,ref mut bytes::Buffer output){let path=join(root,leaf);let file=join(bytes::view(ref path),b"keys.bin");let data=switch(files::manifest(bytes::view(ref file))){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=bytes::view(ref data);assert(bytes::equal(raw[..8],b"FYCKEYS1"));var usize at=8;let count=number(raw,ref mut at);assert(count==keys::count(actual));var usize work=0;append(output,leaf,ref mut work);assert(wire::length(output,count,ref mut work)==0);var bool[512] seen=[false;512];var usize i=0;
    while(i<count){let id=field(raw,ref mut at);let coordinate=field(raw,ref mut at);let index=find(actual,id);assert(index<count && !seen[index]);seen[index]=true;let p=keys::at(actual,index);assert(keys::owner(p)==index && bytes::equal(keys::coordinate(p),coordinate));let base=keys::ordinary_key(p);let source=keys::generic_key(p);
        template(ordinary::canonical(base),field(raw,ref mut at),compiler);template(generic::canonical(source),field(raw,ref mut at),compiler);assert(ordinary::valid_text(ordinary::digest(base)) && generic::valid_text(generic::digest(source)));
        append(output,id,ref mut work);append(output,coordinate,ref mut work);append(output,ordinary::canonical(base),ref mut work);append(output,ordinary::digest(base),ref mut work);append(output,generic::canonical(source),ref mut work);append(output,generic::digest(source),ref mut work);i+=1;
    }assert(at==raw.len);
}
@effects(filesystem,console)
fn keys::Keys detached(ref u8[] root,ref u8[] leaf,ref environment::Environment env){let loaded=load(root,leaf);let checked=complete(ref loaded);var usize work=0;return collect(ref loaded,ref checked,env,ref mut work);}
@test @effects(filesystem,environment,console)
fn void matches_complete_go_keys_with_actual_image_and_source_specializations(){let root=root();let env=current();var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,b"FYCKOUT1").is_ok);var usize work=0;append(ref mut output,environment::compiler_build(ref env),ref mut work);assert(wire::length(ref mut output,6,ref mut work)==0);var usize i=0;
    while(i<6){let leaf=if(i==0){b"Aliases"}else if(i==1){b"PrivateEdit"}else if(i==2){b"Reordered"}else if(i==3){b"Vec"}else if(i==4){b"State"}else{b"Plain"};let actual=detached(bytes::view(ref root),leaf,ref env);oracle(bytes::view(ref root),leaf,ref actual,environment::compiler_build(ref env),ref mut output);i+=1;}
    assert(switch(io::write_bytes(bytes::view(ref output))){case .ok(n):{n==bytes::len(ref output)}case .err(_):{false}});
}
@test @effects(filesystem,environment,console)
fn void preserves_source_order_and_limits_generic_dependency_invalidation(){let root=root();let env=current();let base=detached(bytes::view(ref root),b"Aliases",ref env);let reordered=detached(bytes::view(ref root),b"Reordered",ref env);assert(keys::count(ref base)==3 && keys::count(ref reordered)==3);var usize i=0;
    while(i<3){let a=keys::at(ref base,i);let j=find(ref reordered,keys::instance(a));assert(j<3);let b=keys::at(ref reordered,j);assert(bytes::equal(ordinary::canonical(keys::ordinary_key(a)),ordinary::canonical(keys::ordinary_key(b))) && bytes::equal(generic::canonical(keys::generic_key(a)),generic::canonical(keys::generic_key(b))));i+=1;}
    let edited=detached(bytes::view(ref root),b"PrivateEdit",ref env);let a=keys::at(ref base,0);let j=find(ref edited,keys::instance(a));assert(j<3);let b=keys::at(ref edited,j);assert(bytes::equal(ordinary::digest(keys::ordinary_key(a)),ordinary::digest(keys::ordinary_key(b))) && !bytes::equal(generic::digest(keys::generic_key(a)),generic::digest(keys::generic_key(b))));
}
@test @effects(filesystem,environment,console)
fn void rejects_stale_graph_and_work_exhaustion_without_partial_package_keys(){let root=root();let env=current();let loaded=load(bytes::view(ref root),b"Aliases");let checked=complete(ref loaded);var usize cost=0;let measured=collect(ref loaded,ref checked,ref env,ref mut cost);assert(cost>0 && cost<16777216);var usize work=16777216-cost;let exact=collect(ref loaded,ref checked,ref env,ref mut work);assert(work==16777216 && keys::count(ref measured)==keys::count(ref exact));var usize i=0;while(i<keys::count(ref exact)){assert(bytes::equal(generic::canonical(keys::generic_key(keys::at(ref measured,i))),generic::canonical(keys::generic_key(keys::at(ref exact,i)))));i+=1;}
    work=16777216-cost+1;assert(switch(keys::collect(ref loaded,ref checked,ref env,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=16777216);work=16777217;assert(switch(keys::collect(ref loaded,ref checked,ref env,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work==16777217);let stale=load(bytes::view(ref root),b"PrivateEdit");work=0;assert(switch(keys::collect(ref stale,ref checked,ref env,ref mut work)){case .ok(_):{false}case .err(code):{code==12}});
}

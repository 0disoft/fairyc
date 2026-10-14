module tests::project_cache_lookup;
import project_cache_lookup as lookup;
import project_cache_environment as environment;
import project_module_visibility as visibility;
import project_source_set as sources;
import project_cache_encoding as encoding;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] path=ref data;return switch(files::manifest(path[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem,console)
fn visibility::Resolved load(ref u8[] root,ref u8[] leaf){var bytes::Buffer path=bytes::empty();assert(bytes::append(ref mut path,root).is_ok && bytes::append_byte(ref mut path,47).is_ok && bytes::append(ref mut path,leaf).is_ok);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(visibility::load_phase(ref e));assert(false);load(root,leaf)}};}
@effects(filesystem,environment,console)
fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.phase);assert(false);current()}};}
fn lookup::Identity identity(ref visibility::Resolved loaded,ref environment::Environment env,ref mut usize work){return switch(lookup::collect(loaded,env,work)){case .ok(p):{p}case .err(_):{assert(false);identity(loaded,env,work)}};}
@effects(filesystem,console)
fn lookup::Identity detached(ref u8[] root,ref u8[] leaf,ref environment::Environment env){let loaded=load(root,leaf);var usize work=0;return identity(ref loaded,env,ref mut work);}
@effects(console)
fn void emit(ref mut bytes::Buffer out,ref u8[] name,ref lookup::Identity p){var usize work=0;assert(encoding::string(out,name,ref mut work)==0 && encoding::string(out,lookup::canonical(p),ref mut work)==0 && encoding::string(out,lookup::digest(p),ref mut work)==0);}
@test @effects(filesystem,environment,console)
fn void canonical_routes_survive_owner_order_and_invalidate_changed_inputs(){
    let root=root();let env=current();let aliases=detached(bytes::view(ref root),b"Aliases",ref env);let reordered=detached(bytes::view(ref root),b"Reordered",ref env);assert(bytes::equal(lookup::canonical(ref aliases),lookup::canonical(ref reordered)) && bytes::equal(lookup::digest(ref aliases),lookup::digest(ref reordered)));
    let plain=detached(bytes::view(ref root),b"Plain",ref env);let edited=detached(bytes::view(ref root),b"PrivateEdit",ref env);let state=detached(bytes::view(ref root),b"State",ref env);let vec=detached(bytes::view(ref root),b"Vec",ref env);
    assert(!bytes::equal(lookup::digest(ref plain),lookup::digest(ref edited)) && !bytes::equal(lookup::digest(ref plain),lookup::digest(ref state)) && !bytes::equal(lookup::digest(ref plain),lookup::digest(ref vec)));
    let binding=detached(bytes::view(ref root),b"BindingEdit",ref env);let feature=detached(bytes::view(ref root),b"FeatureEdit",ref env);let invalid=detached(bytes::view(ref root),b"InvalidBody",ref env);
    assert(!bytes::equal(lookup::digest(ref aliases),lookup::digest(ref binding)) && !bytes::equal(lookup::digest(ref aliases),lookup::digest(ref feature)) && !bytes::equal(lookup::digest(ref plain),lookup::digest(ref invalid)));
    var bytes::Buffer out=bytes::empty();var usize work=0;assert(encoding::raw(ref mut out,b"FYLOOK00",ref mut work)==0 && encoding::length(ref mut out,9,ref mut work)==0);emit(ref mut out,b"Aliases",ref aliases);emit(ref mut out,b"Reordered",ref reordered);emit(ref mut out,b"Plain",ref plain);emit(ref mut out,b"PrivateEdit",ref edited);emit(ref mut out,b"State",ref state);emit(ref mut out,b"Vec",ref vec);emit(ref mut out,b"BindingEdit",ref binding);emit(ref mut out,b"FeatureEdit",ref feature);emit(ref mut out,b"InvalidBody",ref invalid);assert(io::write_bytes(bytes::view(ref out)).is_ok);
}
@test @effects(filesystem,environment,console)
fn void independent_exact_budget_rejects_without_publishing_a_route(){
    let root=root();let env=current();let loaded=load(bytes::view(ref root),b"Aliases");var usize cost=0;let first=identity(ref loaded,ref env,ref mut cost);assert(cost>0 && cost<16777216);var usize work=16777216-cost;let exact=identity(ref loaded,ref env,ref mut work);assert(work==16777216 && bytes::equal(lookup::canonical(ref first),lookup::canonical(ref exact)) && bytes::equal(lookup::digest(ref first),lookup::digest(ref exact)));
    work=16777216-cost+1;assert(switch(lookup::collect(ref loaded,ref env,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});work=16777217;assert(switch(lookup::collect(ref loaded,ref env,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work==16777217);
}
